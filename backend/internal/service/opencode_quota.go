package service

// OpenCode Go 额度抓取：OpenCode Go 没有公开的 quota API，社区做法是抓取
// 已认证的 workspace dashboard 页面（Next.js RSC payload）并解析嵌入的
// rollingUsage / weeklyUsage / monthlyUsage 用量窗口。逻辑移植自
// account-pool 的 opencode_client.py（PoC 验证过线上环境）。
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
)

// OpencodeQuotaWindow 是 OpenCode Go 的单个用量窗口（5 小时 / 周 / 月）。
type OpencodeQuotaWindow struct {
	Percent    float64 `json:"percent"`
	ResetInSec int64   `json:"reset_in_sec"`
}

// OpencodeQuota 是 OpenCode Go 账号的额度快照。
type OpencodeQuota struct {
	Rolling  OpencodeQuotaWindow `json:"rolling"`
	Weekly   OpencodeQuotaWindow `json:"weekly"`
	Monthly  OpencodeQuotaWindow `json:"monthly"`
	Plan     string              `json:"plan"`
	FetchedAt time.Time          `json:"fetched_at"`
}

// opencodeGoDashboardUA 与网关测活请求保持一致。
const opencodeGoDashboardUA = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36"

func opencodeWorkspaceDashboardURL(workspaceID string) string {
	return fmt.Sprintf("https://opencode.ai/workspace/%s/go", workspaceID)
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
	if win.ResetInSec == 0 {
		for _, k := range []string{"resetAt", "resetsAt", "reset_at"} {
			if s, ok := obj[k].(string); ok {
				if t, err := time.Parse("2006-01-02T15:04:05", strings.TrimSuffix(s, "Z")); err == nil {
					win.ResetInSec = int64(time.Until(t).Seconds())
					break
				}
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

// opencodeQuotaExtraUpdates 把额度快照转为 account.extra 的更新字段。
func opencodeQuotaExtraUpdates(quota *OpencodeQuota) map[string]any {
	if quota == nil {
		return nil
	}
	return map[string]any{
		"opencode_plan":               quota.Plan,
		"opencode_quota_5h_pct":       quota.Rolling.Percent,
		"opencode_quota_5h_reset_in":  quota.Rolling.ResetInSec,
		"opencode_quota_weekly_pct":   quota.Weekly.Percent,
		"opencode_quota_weekly_reset_in": quota.Weekly.ResetInSec,
		"opencode_quota_monthly_pct":  quota.Monthly.Percent,
		"opencode_quota_monthly_reset_in": quota.Monthly.ResetInSec,
		"opencode_quota_refreshed_at": quota.FetchedAt.UTC().Format(time.RFC3339),
	}
}
