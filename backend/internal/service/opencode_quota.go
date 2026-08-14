package service

// OpenCode Go 额度：优先走官方 GET /zen/go/v1/usage（Bearer API key），
// 返回 {usage: {rolling|weekly|monthly: {status, percent, resetsAt}}}。
// Dashboard HTML 抓取仅作 fallback（补 plan，或 usage API 不可用时）。
//
// 抓取失败不阻塞测活主流程：调用方把错误当作"额度未知"处理。

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

const (
	OpencodeQuotaSourceUsageAPI  = "usage_api"
	OpencodeQuotaSourceDashboard = "dashboard"
)

// OpencodeQuotaWindow 是 OpenCode Go 的单个用量窗口（5 小时 / 周 / 月）。
type OpencodeQuotaWindow struct {
	Percent    float64 `json:"percent"`
	ResetInSec int64   `json:"reset_in_sec"`
	Status     string  `json:"status,omitempty"`
	ResetAt    string  `json:"reset_at,omitempty"`
}

// OpencodeQuota 是 OpenCode Go 账号的额度快照。
type OpencodeQuota struct {
	Rolling   OpencodeQuotaWindow `json:"rolling"`
	Weekly    OpencodeQuotaWindow `json:"weekly"`
	Monthly   OpencodeQuotaWindow `json:"monthly"`
	Plan      string              `json:"plan"`
	Source    string              `json:"source"`
	FetchedAt time.Time           `json:"fetched_at"`
}

// opencodeGoDashboardUA 与网关测活请求保持一致。
const opencodeGoDashboardUA = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36"

func opencodeWorkspaceDashboardURL(workspaceID string) string {
	return fmt.Sprintf("https://opencode.ai/workspace/%s/go", workspaceID)
}

// opencodeUsageURLOverride is used by unit tests to point GET /usage at httptest.
var opencodeUsageURLOverride string

func opencodeUsageURL(baseURL string) string {
	base := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if base == "" {
		base = DefaultOpencodeBaseURL
	}
	return base + "/usage"
}

func parseOpencodeResetAt(raw string) (resetAt string, resetInSec int64) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", 0
	}
	var (
		t   time.Time
		err error
	)
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339} {
		t, err = time.Parse(layout, raw)
		if err == nil {
			break
		}
	}
	if err != nil {
		t, err = time.Parse("2006-01-02T15:04:05", strings.TrimSuffix(raw, "Z"))
	}
	if err != nil {
		return raw, 0
	}
	sec := int64(time.Until(t).Seconds())
	if sec < 0 {
		sec = 0
	}
	return t.UTC().Format(time.RFC3339), sec
}

func usageWindowFromAPI(obj map[string]any) OpencodeQuotaWindow {
	win := OpencodeQuotaWindow{}
	if obj == nil {
		return win
	}
	if v, ok := numberFromRSC(obj["percent"]); ok {
		win.Percent = v
	}
	if s, ok := obj["status"].(string); ok {
		win.Status = strings.TrimSpace(s)
	}
	if s, ok := obj["resetsAt"].(string); ok {
		win.ResetAt, win.ResetInSec = parseOpencodeResetAt(s)
	} else if s, ok := obj["resetAt"].(string); ok {
		win.ResetAt, win.ResetInSec = parseOpencodeResetAt(s)
	}
	return win
}

func parseOpencodeUsagePayload(raw []byte) (*OpencodeQuota, error) {
	var payload struct {
		Usage map[string]map[string]any `json:"usage"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, fmt.Errorf("decode usage payload: %w", err)
	}
	quota := &OpencodeQuota{
		Source:    OpencodeQuotaSourceUsageAPI,
		FetchedAt: time.Now(),
	}
	if payload.Usage != nil {
		quota.Rolling = usageWindowFromAPI(payload.Usage["rolling"])
		quota.Weekly = usageWindowFromAPI(payload.Usage["weekly"])
		quota.Monthly = usageWindowFromAPI(payload.Usage["monthly"])
	}
	return quota, nil
}

func usageAPIErrorKind(statusCode int, body []byte) (plan string, message string) {
	lower := strings.ToLower(string(body))
	switch {
	case strings.Contains(lower, "entitlementerror") || strings.Contains(lower, "go subscription required") || strings.Contains(lower, "no payment method"):
		return "go_not_subscribed", fmt.Sprintf("OpenCode Go not subscribed (HTTP %d)", statusCode)
	case strings.Contains(lower, "insufficient balance") || strings.Contains(lower, "creditserror"):
		return "go_subscribed", fmt.Sprintf("OpenCode Go balance exhausted (HTTP %d)", statusCode)
	case statusCode == http.StatusUnauthorized:
		return "", fmt.Sprintf("API key rejected (HTTP %d)", statusCode)
	default:
		return "", fmt.Sprintf("usage API failed (HTTP %d)", statusCode)
	}
}

// fetchOpencodeQuotaFromUsageAPI 用 API key 查询官方额度窗口。
func fetchOpencodeQuotaFromUsageAPI(ctx context.Context, usageURL, apiKey string, timeout time.Duration) (*OpencodeQuota, error) {
	if strings.TrimSpace(apiKey) == "" {
		return nil, fmt.Errorf("missing api_key")
	}
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, usageURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", opencodeGoDashboardUA)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	client := &http.Client{Timeout: timeout}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode == http.StatusOK {
		quota, err := parseOpencodeUsagePayload(raw)
		if err != nil {
			return nil, err
		}
		if quota.Plan == "" {
			quota.Plan = "unknown"
		}
		return quota, nil
	}

	plan, msg := usageAPIErrorKind(resp.StatusCode, raw)
	if plan != "" && resp.StatusCode != http.StatusUnauthorized {
		quota := &OpencodeQuota{
			Plan:      plan,
			Source:    OpencodeQuotaSourceUsageAPI,
			FetchedAt: time.Now(),
		}
		if plan == "go_subscribed" {
			quota.Weekly = OpencodeQuotaWindow{Percent: 100, Status: "rate-limited"}
		}
		return quota, nil
	}
	return nil, fmt.Errorf("%s", msg)
}

// QueryOpencodeAccountQuota 查询账号当前 Go 额度并返回快照（不落库）。
// 优先 usage API；cookie 可用时再抓 dashboard 补 plan。
func QueryOpencodeAccountQuota(ctx context.Context, account *Account) (*OpencodeQuota, error) {
	if account == nil || !account.IsOpencode() {
		return nil, infraerrors.New(http.StatusBadRequest, "OPENCODE_QUOTA_INVALID_PLATFORM", "account is not an OpenCode account")
	}
	apiKey := account.GetOpencodeAPIKey()
	if apiKey == "" {
		return nil, infraerrors.New(http.StatusBadRequest, "OPENCODE_QUOTA_MISSING_API_KEY", "OpenCode API key is missing")
	}

	timeout := 30 * time.Second
	usageURL := opencodeUsageURL(account.GetOpencodeBaseURL())
	if opencodeUsageURLOverride != "" {
		usageURL = opencodeUsageURLOverride
	}
	quota, usageErr := fetchOpencodeQuotaFromUsageAPI(ctx, usageURL, apiKey, timeout)

	workspaceID := strings.TrimSpace(account.GetCredential("workspace_id"))
	authCookie := strings.TrimSpace(account.GetCredential("auth_cookie"))
	if usageErr != nil {
		if workspaceID == "" || authCookie == "" {
			return nil, infraerrors.New(http.StatusBadGateway, "OPENCODE_QUOTA_UPSTREAM_FAILED", usageErr.Error()).WithCause(usageErr)
		}
		dash, dashErr := fetchOpencodeQuota(ctx, workspaceID, authCookie, timeout)
		if dashErr != nil {
			return nil, infraerrors.New(http.StatusBadGateway, "OPENCODE_QUOTA_UPSTREAM_FAILED", usageErr.Error()).WithCause(usageErr)
		}
		return dash, nil
	}

	if workspaceID != "" && authCookie != "" && (quota.Plan == "" || quota.Plan == "unknown") {
		if dash, dashErr := fetchOpencodeQuota(ctx, workspaceID, authCookie, timeout); dashErr == nil && dash.Plan != "" && dash.Plan != "unknown" {
			quota.Plan = dash.Plan
		}
	}
	return quota, nil
}

// fetchOpencodeQuota 抓取并解析 OpenCode Go workspace dashboard 的额度窗口。
// authCookie 为 FusionAuth session（credentials.auth_cookie）。任何一步失败
// 都返回 error，由调用方决定是否降级（仅连通性结果）。
func fetchOpencodeQuota(ctx context.Context, workspaceID, authCookie string, timeout time.Duration) (*OpencodeQuota, error) {
	if strings.TrimSpace(workspaceID) == "" || strings.TrimSpace(authCookie) == "" {
		return nil, fmt.Errorf("missing workspace_id or auth_cookie")
	}
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, opencodeWorkspaceDashboardURL(workspaceID), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", opencodeGoDashboardUA)
	req.Header.Set("Accept", "text/html,application/xhtml+xml")
	req.Header.Set("Cookie", "auth="+authCookie)
	req.Header.Set("Referer", "https://opencode.ai/")

	client := &http.Client{Timeout: timeout}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return nil, fmt.Errorf("auth cookie rejected (HTTP %d)", resp.StatusCode)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("dashboard fetch failed (HTTP %d)", resp.StatusCode)
	}
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return nil, err
	}
	html := string(raw)
	lower := strings.ToLower(html)
	if strings.Contains(lower, "/auth") || strings.Contains(lower, "continue with github") {
		return nil, fmt.Errorf("session expired, redirected to login")
	}

	quota := &OpencodeQuota{
		Source:    OpencodeQuotaSourceDashboard,
		FetchedAt: time.Now(),
	}
	if obj := extractRSCJSONObject(html, "rollingUsage"); obj != nil {
		quota.Rolling = rscUsageWindow(obj)
	}
	if obj := extractRSCJSONObject(html, "weeklyUsage"); obj != nil {
		quota.Weekly = rscUsageWindow(obj)
	}
	if obj := extractRSCJSONObject(html, "monthlyUsage"); obj != nil {
		quota.Monthly = rscUsageWindow(obj)
	}

	// 订阅状态以页面文案为准（免费账号也可能带全零的用量窗口）。
	switch {
	case strings.Contains(lower, "you are subscribed to opencode go") || strings.Contains(lower, "manage subscription"):
		quota.Plan = "go_subscribed"
	case strings.Contains(lower, "subscribe to go") || (strings.Contains(lower, "subscribe") && strings.Contains(lower, "go plan")):
		quota.Plan = "go_not_subscribed"
	case strings.Contains(lower, "use balance"):
		quota.Plan = "zen_balance"
	default:
		quota.Plan = "unknown"
	}
	return quota, nil
}

// rscUsageWindow 从解析出的 RSC 对象中提取 percent 与剩余秒数。
func rscUsageWindow(obj map[string]any) OpencodeQuotaWindow {
	// 部分构建会嵌套 {"usage": {...}} / {"quota": {...}} / {"window": {...}}
	for _, inner := range []string{"usage", "quota", "window"} {
		if nested, ok := obj[inner].(map[string]any); ok {
			obj = nested
			break
		}
	}
	win := OpencodeQuotaWindow{}
	if s, ok := obj["status"].(string); ok {
		win.Status = strings.TrimSpace(s)
	}
	for _, k := range []string{"percent", "usagePercent", "usedPercent", "usage_pct", "pct"} {
		if v, ok := numberFromRSC(obj[k]); ok {
			win.Percent = v
			break
		}
	}
	for _, k := range []string{"resetInSec", "resetInSeconds", "reset_in_sec", "resetsInSec"} {
		if v, ok := numberFromRSC(obj[k]); ok && v >= 0 {
			win.ResetInSec = int64(v)
			break
		}
	}
	if win.ResetAt == "" {
		for _, k := range []string{"resetAt", "resetsAt", "reset_at"} {
			if s, ok := obj[k].(string); ok {
				resetAt, resetIn := parseOpencodeResetAt(s)
				win.ResetAt = resetAt
				if win.ResetInSec == 0 {
					win.ResetInSec = resetIn
				}
				break
			}
		}
	}
	return win
}

func numberFromRSC(v any) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case float32:
		return float64(n), true
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	case json.Number:
		f, err := n.Float64()
		return f, err == nil
	case string:
		f, err := strconv.ParseFloat(strings.TrimSpace(n), 64)
		return f, err == nil
	default:
		return 0, false
	}
}

// rscJSONKeyFix 修复 RSC payload 中未加引号的 JS 风格键（SolidJS）。
var rscJSONKeyFix = regexp.MustCompile(`([{,]\s*)([A-Za-z_$][A-Za-z0-9_$]*)\s*:`)

// extractRSCJSONObject 从 RSC payload 文本中找到 key 附近起始的平衡 JSON 对象
// 并解析。容忍 \u0022 等转义与未加引号的键。
func extractRSCJSONObject(text, key string) map[string]any {
	idx := strings.Index(text, key)
	if idx < 0 {
		return nil
	}
	start := strings.Index(text[idx:], "{")
	if start < 0 {
		return nil
	}
	start += idx
	depth := 0
	inStr := false
	esc := false
	for i := start; i < len(text); i++ {
		c := text[i]
		if esc {
			esc = false
		} else if c == '\\' {
			esc = true
		} else if c == '"' {
			inStr = !inStr
		} else if !inStr {
			if c == '{' {
				depth++
			} else if c == '}' {
				depth--
				if depth == 0 {
					raw := unescapeRSC(text[start : i+1])
					obj := map[string]any{}
					if err := json.Unmarshal([]byte(raw), &obj); err == nil {
						return obj
					}
					fixed := rscJSONKeyFix.ReplaceAllString(raw, `${1}"${2}":`)
					if err := json.Unmarshal([]byte(fixed), &obj); err == nil {
						return obj
					}
					return nil
				}
			}
		}
	}
	return nil
}

// unescapeRSC 还原 Next.js/SolidJS RSC payload 的转义。
func unescapeRSC(s string) string {
	replacer := strings.NewReplacer(
		`\"`, `"`,
		`\u0022`, `"`,
		`\u002F`, `/`,
		`\u003C`, `<`,
		`\u003E`, `>`,
		`\u0026`, `&`,
	)
	return replacer.Replace(s)
}

// OpencodeQuotaExtraUpdates 把额度快照转为 account.extra 的更新字段。
func OpencodeQuotaExtraUpdates(quota *OpencodeQuota) map[string]any {
	if quota == nil {
		return nil
	}
	return map[string]any{
		"opencode_plan":                   quota.Plan,
		"opencode_quota_source":           quota.Source,
		"opencode_quota_5h_pct":           quota.Rolling.Percent,
		"opencode_quota_5h_reset_in":      quota.Rolling.ResetInSec,
		"opencode_quota_5h_reset_at":      quota.Rolling.ResetAt,
		"opencode_quota_5h_status":        quota.Rolling.Status,
		"opencode_quota_weekly_pct":       quota.Weekly.Percent,
		"opencode_quota_weekly_reset_in":  quota.Weekly.ResetInSec,
		"opencode_quota_weekly_reset_at":  quota.Weekly.ResetAt,
		"opencode_quota_weekly_status":    quota.Weekly.Status,
		"opencode_quota_monthly_pct":      quota.Monthly.Percent,
		"opencode_quota_monthly_reset_in": quota.Monthly.ResetInSec,
		"opencode_quota_monthly_reset_at": quota.Monthly.ResetAt,
		"opencode_quota_monthly_status":   quota.Monthly.Status,
		"opencode_quota_refreshed_at":     quota.FetchedAt.UTC().Format(time.RFC3339),
	}
}

func opencodeQuotaExtraUpdates(quota *OpencodeQuota) map[string]any {
	return OpencodeQuotaExtraUpdates(quota)
}
