package service

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

func TestParseOpencodeUsagePayload(t *testing.T) {
	raw := []byte(`{
		"usage": {
			"rolling": {"status":"ok","percent":18.4,"resetsAt":"2026-08-14T12:00:00Z"},
			"weekly": {"status":"rate-limited","percent":100,"resetsAt":"2026-08-17T00:00:00Z"},
			"monthly": {"status":"ok","percent":50.2,"resetsAt":"2026-09-01T00:00:00Z"}
		}
	}`)
	quota, err := parseOpencodeUsagePayload(raw)
	if err != nil {
		t.Fatal(err)
	}
	if quota.Source != OpencodeQuotaSourceUsageAPI {
		t.Fatalf("source = %q", quota.Source)
	}
	if quota.Rolling.Percent != 18.4 || quota.Rolling.Status != "ok" {
		t.Fatalf("rolling = %+v", quota.Rolling)
	}
	if quota.Weekly.Percent != 100 || quota.Weekly.Status != "rate-limited" {
		t.Fatalf("weekly = %+v", quota.Weekly)
	}
	if quota.Weekly.ResetAt != "2026-08-17T00:00:00Z" {
		t.Fatalf("weekly reset_at = %q", quota.Weekly.ResetAt)
	}
	if quota.Monthly.Percent != 50.2 {
		t.Fatalf("monthly = %+v", quota.Monthly)
	}
}

func TestFetchOpencodeQuotaFromUsageAPIEntitlement(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Errorf("missing bearer, got %q", r.Header.Get("Authorization"))
		}
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"error":{"type":"EntitlementError","message":"Go subscription required"}}`))
	}))
	defer srv.Close()

	quota, err := fetchOpencodeQuotaFromUsageAPI(context.Background(), srv.URL, "test-key", time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if quota.Plan != "go_not_subscribed" {
		t.Fatalf("plan = %q", quota.Plan)
	}
}

func TestFetchOpencodeQuotaFromUsageAPIOK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"usage": map[string]any{
				"rolling": map[string]any{"status": "ok", "percent": 0, "resetsAt": "2026-08-14T20:00:00Z"},
				"weekly":  map[string]any{"status": "ok", "percent": 42, "resetsAt": "2026-08-17T00:00:00Z"},
				"monthly": map[string]any{"status": "ok", "percent": 21, "resetsAt": "2026-09-01T00:00:00Z"},
			},
		})
	}))
	defer srv.Close()

	quota, err := fetchOpencodeQuotaFromUsageAPI(context.Background(), srv.URL, "k", time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if quota.Weekly.Percent != 42 {
		t.Fatalf("weekly percent = %v", quota.Weekly.Percent)
	}
}

func TestQueryOpencodeAccountQuotaValidation(t *testing.T) {
	_, err := QueryOpencodeAccountQuota(context.Background(), &Account{Platform: PlatformOpenAI})
	if err == nil {
		t.Fatal("expected platform error")
	}
	if status := infraerrors.FromError(err); status == nil || status.Reason != "OPENCODE_QUOTA_INVALID_PLATFORM" {
		t.Fatalf("got %v", err)
	}

	_, err = QueryOpencodeAccountQuota(context.Background(), &Account{
		Platform:    PlatformOpencode,
		Type:        AccountTypeAPIKey,
		Credentials: map[string]any{},
	})
	if err == nil {
		t.Fatal("expected missing key error")
	}
	if status := infraerrors.FromError(err); status == nil || status.Reason != "OPENCODE_QUOTA_MISSING_API_KEY" {
		t.Fatalf("got %v", err)
	}
}

func TestQueryOpencodeAccountQuotaUsesUsageAPI(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"usage": map[string]any{
				"weekly": map[string]any{"status": "ok", "percent": 12, "resetsAt": "2026-08-17T00:00:00Z"},
			},
		})
	}))
	defer srv.Close()

	prev := opencodeUsageURLOverride
	opencodeUsageURLOverride = srv.URL
	defer func() { opencodeUsageURLOverride = prev }()

	quota, err := QueryOpencodeAccountQuota(context.Background(), &Account{
		Platform: PlatformOpencode,
		Type:     AccountTypeAPIKey,
		Credentials: map[string]any{
			"api_key": "secret",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if quota.Weekly.Percent != 12 {
		t.Fatalf("weekly = %+v", quota.Weekly)
	}
	updates := OpencodeQuotaExtraUpdates(quota)
	if updates["opencode_quota_weekly_pct"] != float64(12) {
		t.Fatalf("extra weekly = %#v", updates["opencode_quota_weekly_pct"])
	}
	if updates["opencode_quota_source"] != OpencodeQuotaSourceUsageAPI {
		t.Fatalf("source extra = %#v", updates["opencode_quota_source"])
	}
}
