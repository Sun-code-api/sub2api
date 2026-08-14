package admin

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func setupOpencodeQuotaRouter(adminSvc service.AdminService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	handler := NewAccountHandler(adminSvc, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	router.GET("/api/v1/admin/accounts/:id/opencode-quota", handler.QueryOpencodeQuota)
	return router
}

func TestAccountHandlerQueryOpencodeQuotaInvalidID(t *testing.T) {
	router := setupOpencodeQuotaRouter(newStubAdminService())
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/accounts/abc/opencode-quota", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestAccountHandlerQueryOpencodeQuotaRejectsNonOpencode(t *testing.T) {
	svc := newStubAdminService()
	svc.getAccountResult = &service.Account{
		ID:       7,
		Platform: service.PlatformOpenAI,
		Type:     service.AccountTypeAPIKey,
	}
	router := setupOpencodeQuotaRouter(svc)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/accounts/7/opencode-quota", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusBadRequest, rec.Code)

	var body map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Equal(t, "OPENCODE_QUOTA_INVALID_PLATFORM", body["reason"])
}

func TestAccountHandlerQueryOpencodeQuotaRequiresAPIKey(t *testing.T) {
	svc := newStubAdminService()
	svc.getAccountResult = &service.Account{
		ID:          8,
		Platform:    service.PlatformOpencode,
		Type:        service.AccountTypeAPIKey,
		Credentials: map[string]any{},
	}
	router := setupOpencodeQuotaRouter(svc)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/accounts/8/opencode-quota", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusBadRequest, rec.Code)

	var body map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Equal(t, "OPENCODE_QUOTA_MISSING_API_KEY", body["reason"])
}
