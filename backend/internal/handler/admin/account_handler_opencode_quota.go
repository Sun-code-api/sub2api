package admin

import (
	"strconv"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

// QueryOpencodeQuota queries live OpenCode Go usage windows and persists them
// to account.extra. GET /api/v1/admin/accounts/:id/opencode-quota
func (h *AccountHandler) QueryOpencodeQuota(c *gin.Context) {
	accountID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid account ID")
		return
	}

	account, err := h.adminService.GetAccount(c.Request.Context(), accountID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	quota, err := service.QueryOpencodeAccountQuota(c.Request.Context(), account)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	updates := service.OpencodeQuotaExtraUpdates(quota)
	if len(updates) > 0 {
		if err := h.adminService.UpdateAccountExtra(c.Request.Context(), account.ID, updates); err != nil {
			response.ErrorFrom(c, err)
			return
		}
	}

	response.Success(c, gin.H{
		"rolling":    quota.Rolling,
		"weekly":     quota.Weekly,
		"monthly":    quota.Monthly,
		"plan":       quota.Plan,
		"source":     quota.Source,
		"fetched_at": quota.FetchedAt.UTC().Format(time.RFC3339),
		"extra":      updates,
	})
}
