package admin

import (
	"strconv"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// SchedulingPolicyHandler 渠道调度策略管理后台 handler。
type SchedulingPolicyHandler struct {
	policyService *service.SchedulingPolicyService
}

// NewSchedulingPolicyHandler 创建 handler。
func NewSchedulingPolicyHandler(policyService *service.SchedulingPolicyService) *SchedulingPolicyHandler {
	return &SchedulingPolicyHandler{policyService: policyService}
}

// --- Request / Response ---

type schedulingPolicyRequest struct {
	Name                        string  `json:"name" binding:"required,max=100"`
	Enabled                     *bool   `json:"enabled"`
	MonitorID                   int64   `json:"monitor_id" binding:"required,min=1"`
	AccountIDs                  []int64 `json:"account_ids" binding:"required,min=1"`
	TriggerConsecutiveFailures  int     `json:"trigger_consecutive_failures" binding:"required,min=1,max=100"`
	TriggerLatencyMs            int     `json:"trigger_latency_ms" binding:"omitempty,min=0,max=600000"`
	ActionType                  string  `json:"action_type" binding:"required,oneof=pause deprioritize"`
	PauseMinutes                int     `json:"pause_minutes" binding:"omitempty,min=0,max=10080"`
	PriorityDelta               int     `json:"priority_delta" binding:"omitempty,min=1,max=1000"`
	RecoverConsecutiveSuccesses int     `json:"recover_consecutive_successes" binding:"omitempty,min=0,max=100"`
	CooldownMinutes             int     `json:"cooldown_minutes" binding:"omitempty,min=0,max=1440"`
}

type schedulingPolicyResponse struct {
	ID                          int64   `json:"id"`
	Name                        string  `json:"name"`
	Enabled                     bool    `json:"enabled"`
	MonitorID                   int64   `json:"monitor_id"`
	AccountIDs                  []int64 `json:"account_ids"`
	TriggerConsecutiveFailures  int     `json:"trigger_consecutive_failures"`
	TriggerLatencyMs            int     `json:"trigger_latency_ms"`
	ActionType                  string  `json:"action_type"`
	PauseMinutes                int     `json:"pause_minutes"`
	PriorityDelta               int     `json:"priority_delta"`
	RecoverConsecutiveSuccesses int     `json:"recover_consecutive_successes"`
	CooldownMinutes             int     `json:"cooldown_minutes"`
	CreatedAt                   string  `json:"created_at"`
	UpdatedAt                   string  `json:"updated_at"`
}

type schedulingActionResponse struct {
	ID               int64  `json:"id"`
	PolicyID         int64  `json:"policy_id"`
	AccountID        int64  `json:"account_id"`
	MonitorID        int64  `json:"monitor_id"`
	Action           string `json:"action"`
	Reason           string `json:"reason"`
	OriginalPriority int    `json:"original_priority"`
	Restored         bool   `json:"restored"`
	CreatedAt        string `json:"created_at"`
}

func schedulingPolicyToResponse(p *service.SchedulingPolicy) *schedulingPolicyResponse {
	if p == nil {
		return nil
	}
	accountIDs := p.AccountIDs
	if accountIDs == nil {
		accountIDs = []int64{}
	}
	return &schedulingPolicyResponse{
		ID:                          p.ID,
		Name:                        p.Name,
		Enabled:                     p.Enabled,
		MonitorID:                   p.MonitorID,
		AccountIDs:                  accountIDs,
		TriggerConsecutiveFailures:  p.TriggerConsecutiveFailures,
		TriggerLatencyMs:            p.TriggerLatencyMs,
		ActionType:                  p.ActionType,
		PauseMinutes:                p.PauseMinutes,
		PriorityDelta:               p.PriorityDelta,
		RecoverConsecutiveSuccesses: p.RecoverConsecutiveSuccesses,
		CooldownMinutes:             p.CooldownMinutes,
		CreatedAt:                   p.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:                   p.UpdatedAt.UTC().Format(time.RFC3339),
	}
}

func schedulingActionToResponse(a *service.SchedulingActionRecord) *schedulingActionResponse {
	return &schedulingActionResponse{
		ID:               a.ID,
		PolicyID:         a.PolicyID,
		AccountID:        a.AccountID,
		MonitorID:        a.MonitorID,
		Action:           a.Action,
		Reason:           a.Reason,
		OriginalPriority: a.OriginalPriority,
		Restored:         a.Restored,
		CreatedAt:        a.CreatedAt.UTC().Format(time.RFC3339),
	}
}

func requestToSchedulingPolicy(req *schedulingPolicyRequest) *service.SchedulingPolicy {
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	priorityDelta := req.PriorityDelta
	if priorityDelta == 0 {
		priorityDelta = 10
	}
	return &service.SchedulingPolicy{
		Name:                        req.Name,
		Enabled:                     enabled,
		MonitorID:                   req.MonitorID,
		AccountIDs:                  req.AccountIDs,
		TriggerConsecutiveFailures:  req.TriggerConsecutiveFailures,
		TriggerLatencyMs:            req.TriggerLatencyMs,
		ActionType:                  req.ActionType,
		PauseMinutes:                req.PauseMinutes,
		PriorityDelta:               priorityDelta,
		RecoverConsecutiveSuccesses: req.RecoverConsecutiveSuccesses,
		CooldownMinutes:             req.CooldownMinutes,
	}
}

func parseSchedulingPolicyID(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.ErrorFrom(c, infraerrors.BadRequest("INVALID_POLICY_ID", "invalid policy id"))
		return 0, false
	}
	return id, true
}

// --- Handlers ---

// List GET /admin/scheduling-policies
func (h *SchedulingPolicyHandler) List(c *gin.Context) {
	page, pageSize := response.ParsePagination(c)
	items, total, err := h.policyService.List(c.Request.Context(), service.SchedulingPolicyListParams{
		Page:     page,
		PageSize: pageSize,
		Enabled:  parseListEnabled(c.Query("enabled")),
		Search:   strings.TrimSpace(c.Query("search")),
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	out := make([]*schedulingPolicyResponse, 0, len(items))
	for _, p := range items {
		out = append(out, schedulingPolicyToResponse(p))
	}
	response.Paginated(c, out, total, page, pageSize)
}

// Get GET /admin/scheduling-policies/:id
func (h *SchedulingPolicyHandler) Get(c *gin.Context) {
	id, ok := parseSchedulingPolicyID(c)
	if !ok {
		return
	}
	p, err := h.policyService.Get(c.Request.Context(), id)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, schedulingPolicyToResponse(p))
}

// Create POST /admin/scheduling-policies
func (h *SchedulingPolicyHandler) Create(c *gin.Context) {
	var req schedulingPolicyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorFrom(c, infraerrors.BadRequest("VALIDATION_ERROR", err.Error()))
		return
	}
	p := requestToSchedulingPolicy(&req)
	if err := h.policyService.Create(c.Request.Context(), p); err != nil {
		response.ErrorFrom(c, infraerrors.BadRequest("VALIDATION_ERROR", err.Error()))
		return
	}
	response.Success(c, schedulingPolicyToResponse(p))
}

// Update PUT /admin/scheduling-policies/:id
func (h *SchedulingPolicyHandler) Update(c *gin.Context) {
	id, ok := parseSchedulingPolicyID(c)
	if !ok {
		return
	}
	var req schedulingPolicyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorFrom(c, infraerrors.BadRequest("VALIDATION_ERROR", err.Error()))
		return
	}
	p := requestToSchedulingPolicy(&req)
	p.ID = id
	if err := h.policyService.Update(c.Request.Context(), p); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, schedulingPolicyToResponse(p))
}

// Delete DELETE /admin/scheduling-policies/:id
func (h *SchedulingPolicyHandler) Delete(c *gin.Context) {
	id, ok := parseSchedulingPolicyID(c)
	if !ok {
		return
	}
	if err := h.policyService.Delete(c.Request.Context(), id); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"deleted": true})
}

// Actions GET /admin/scheduling-policies/actions
func (h *SchedulingPolicyHandler) Actions(c *gin.Context) {
	page, pageSize := response.ParsePagination(c)
	policyID, _ := strconv.ParseInt(c.Query("policy_id"), 10, 64)
	accountID, _ := strconv.ParseInt(c.Query("account_id"), 10, 64)
	items, total, err := h.policyService.ListActions(c.Request.Context(), service.SchedulingActionListParams{
		Page:      page,
		PageSize:  pageSize,
		PolicyID:  policyID,
		AccountID: accountID,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	out := make([]*schedulingActionResponse, 0, len(items))
	for _, a := range items {
		out = append(out, schedulingActionToResponse(a))
	}
	response.Paginated(c, out, total, page, pageSize)
}
