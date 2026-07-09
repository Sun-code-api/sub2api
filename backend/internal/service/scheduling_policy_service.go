package service

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

// ErrSchedulingPolicyNotFound 策略不存在。
var ErrSchedulingPolicyNotFound = infraerrors.NotFound(
	"SCHEDULING_POLICY_NOT_FOUND", "scheduling policy not found",
)

// schedulingPauseMaxDuration pause_minutes=0（停用至恢复）时使用的兜底时长，
// 防止恢复链路异常导致账号被永久停用。
const schedulingPauseMaxDuration = 30 * 24 * time.Hour

// schedulingAccountRepo 策略引擎用到的账号仓储子集（AccountRepository 自然满足），
// 缩小接口面便于单测注入轻量 stub。
type schedulingAccountRepo interface {
	GetByID(ctx context.Context, id int64) (*Account, error)
	SetTempUnschedulable(ctx context.Context, id int64, until time.Time, reason string) error
	ClearTempUnschedulable(ctx context.Context, id int64) error
	BulkUpdate(ctx context.Context, ids []int64, updates AccountBulkUpdate) (int64, error)
}

// schedulingPolicyState 单个策略的运行时计数器（进程内）。
type schedulingPolicyState struct {
	consecutiveFailures  int
	consecutiveSuccesses int
}

// SchedulingPolicyService 渠道调度策略服务：
// CRUD + 监控结果驱动的账号调度联动（自动临停/降级/恢复）+ 动作审计。
type SchedulingPolicyService struct {
	repo        SchedulingPolicyRepository
	accountRepo schedulingAccountRepo

	mu     sync.Mutex
	states map[int64]*schedulingPolicyState
}

// NewSchedulingPolicyService 创建调度策略服务实例。
func NewSchedulingPolicyService(repo SchedulingPolicyRepository, accountRepo schedulingAccountRepo) *SchedulingPolicyService {
	return &SchedulingPolicyService{
		repo:        repo,
		accountRepo: accountRepo,
		states:      make(map[int64]*schedulingPolicyState),
	}
}

// ---------- CRUD ----------

func (s *SchedulingPolicyService) List(ctx context.Context, params SchedulingPolicyListParams) ([]*SchedulingPolicy, int64, error) {
	if params.Page < 1 {
		params.Page = 1
	}
	if params.PageSize < 1 || params.PageSize > 100 {
		params.PageSize = 20
	}
	return s.repo.List(ctx, params)
}

func (s *SchedulingPolicyService) Get(ctx context.Context, id int64) (*SchedulingPolicy, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *SchedulingPolicyService) Create(ctx context.Context, p *SchedulingPolicy) error {
	if err := validateSchedulingPolicy(p); err != nil {
		return err
	}
	return s.repo.Create(ctx, p)
}

func (s *SchedulingPolicyService) Update(ctx context.Context, p *SchedulingPolicy) error {
	if err := validateSchedulingPolicy(p); err != nil {
		return err
	}
	if err := s.repo.Update(ctx, p); err != nil {
		return err
	}
	s.resetState(p.ID)
	return nil
}

func (s *SchedulingPolicyService) Delete(ctx context.Context, id int64) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		return err
	}
	s.resetState(id)
	return nil
}

func (s *SchedulingPolicyService) ListActions(ctx context.Context, params SchedulingActionListParams) ([]*SchedulingActionRecord, int64, error) {
	if params.Page < 1 {
		params.Page = 1
	}
	if params.PageSize < 1 || params.PageSize > 100 {
		params.PageSize = 20
	}
	return s.repo.ListActions(ctx, params)
}

func validateSchedulingPolicy(p *SchedulingPolicy) error {
	p.Name = strings.TrimSpace(p.Name)
	if p.Name == "" {
		return schedulingPolicyInvalid("name is required")
	}
	if p.MonitorID <= 0 {
		return schedulingPolicyInvalid("monitor_id is required")
	}
	if len(p.AccountIDs) == 0 {
		return schedulingPolicyInvalid("account_ids is required")
	}
	if p.ActionType != SchedulingActionPause && p.ActionType != SchedulingActionDeprioritize {
		return schedulingPolicyInvalid(fmt.Sprintf("invalid action_type: %s", p.ActionType))
	}
	if p.TriggerConsecutiveFailures < 1 || p.TriggerConsecutiveFailures > 100 {
		return schedulingPolicyInvalid("trigger_consecutive_failures must be within [1, 100]")
	}
	if p.TriggerLatencyMs < 0 || p.TriggerLatencyMs > 600000 {
		return schedulingPolicyInvalid("trigger_latency_ms must be within [0, 600000]")
	}
	if p.PauseMinutes < 0 || p.PauseMinutes > 10080 {
		return schedulingPolicyInvalid("pause_minutes must be within [0, 10080]")
	}
	if p.PriorityDelta < 1 || p.PriorityDelta > 1000 {
		return schedulingPolicyInvalid("priority_delta must be within [1, 1000]")
	}
	if p.RecoverConsecutiveSuccesses < 0 || p.RecoverConsecutiveSuccesses > 100 {
		return schedulingPolicyInvalid("recover_consecutive_successes must be within [0, 100]")
	}
	if p.CooldownMinutes < 0 || p.CooldownMinutes > 1440 {
		return schedulingPolicyInvalid("cooldown_minutes must be within [0, 1440]")
	}
	return nil
}

func schedulingPolicyInvalid(msg string) error {
	return infraerrors.BadRequest("SCHEDULING_POLICY_INVALID", msg)
}

// ---------- 引擎：监控结果驱动 ----------

// OnCheckResults 渠道监控检查完成后的回调入口。
// 由 ChannelMonitorService.RunCheck 在写入历史后调用；内部自行吞掉错误（只记日志），
// 保证监控主链路不受策略引擎故障影响。
func (s *SchedulingPolicyService) OnCheckResults(ctx context.Context, monitor *ChannelMonitor, results []*CheckResult) {
	if s == nil || monitor == nil || len(results) == 0 {
		return
	}
	policies, err := s.repo.ListEnabledByMonitorID(ctx, monitor.ID)
	if err != nil {
		slog.Error("scheduling policy: list by monitor failed", "monitor_id", monitor.ID, "error", err)
		return
	}
	for _, p := range policies {
		s.evaluate(ctx, p, monitor, results)
	}
}

// evaluate 用一次检查结果推进单个策略的状态机。
func (s *SchedulingPolicyService) evaluate(ctx context.Context, p *SchedulingPolicy, monitor *ChannelMonitor, results []*CheckResult) {
	failed, reason := checkFailedForPolicy(p, monitor, results)

	s.mu.Lock()
	st, ok := s.states[p.ID]
	if !ok {
		st = &schedulingPolicyState{}
		s.states[p.ID] = st
	}
	if failed {
		st.consecutiveFailures++
		st.consecutiveSuccesses = 0
	} else {
		st.consecutiveSuccesses++
		st.consecutiveFailures = 0
	}
	failures, successes := st.consecutiveFailures, st.consecutiveSuccesses
	s.mu.Unlock()

	if failed && failures >= p.TriggerConsecutiveFailures {
		s.trigger(ctx, p, reason)
		return
	}
	if !failed && p.RecoverConsecutiveSuccesses > 0 && successes >= p.RecoverConsecutiveSuccesses {
		s.recover(ctx, p)
	}
}

// checkFailedForPolicy 判定一次检查是否计为失败。
// 取主模型结果（缺失时取第一条）；status 为 failed/error 或延迟超过阈值均计为失败。
func checkFailedForPolicy(p *SchedulingPolicy, monitor *ChannelMonitor, results []*CheckResult) (bool, string) {
	r := results[0]
	for _, cand := range results {
		if cand.Model == monitor.PrimaryModel {
			r = cand
			break
		}
	}
	if r.Status == "failed" || r.Status == "error" {
		msg := r.Message
		if len(msg) > 200 {
			msg = msg[:200]
		}
		return true, fmt.Sprintf("monitor %q model %s status=%s %s", monitor.Name, r.Model, r.Status, msg)
	}
	if p.TriggerLatencyMs > 0 && r.LatencyMs != nil && *r.LatencyMs > p.TriggerLatencyMs {
		return true, fmt.Sprintf("monitor %q model %s latency %dms > %dms", monitor.Name, r.Model, *r.LatencyMs, p.TriggerLatencyMs)
	}
	return false, ""
}

// trigger 对策略目标账号执行动作（跳过已有未恢复动作的账号），并写审计。
func (s *SchedulingPolicyService) trigger(ctx context.Context, p *SchedulingPolicy, reason string) {
	if s.accountRepo == nil {
		return
	}
	last, err := s.repo.LatestTriggerAt(ctx, p.ID)
	if err != nil {
		slog.Error("scheduling policy: query latest trigger failed", "policy_id", p.ID, "error", err)
		return
	}
	if last != nil && p.CooldownMinutes > 0 && time.Since(*last) < time.Duration(p.CooldownMinutes)*time.Minute {
		return
	}

	active, err := s.repo.ListUnrestoredActions(ctx, p.ID)
	if err != nil {
		slog.Error("scheduling policy: list unrestored actions failed", "policy_id", p.ID, "error", err)
		return
	}
	activeAccounts := make(map[int64]struct{}, len(active))
	for _, a := range active {
		activeAccounts[a.AccountID] = struct{}{}
	}

	for _, accountID := range p.AccountIDs {
		if _, exists := activeAccounts[accountID]; exists {
			continue
		}
		account, err := s.accountRepo.GetByID(ctx, accountID)
		if err != nil {
			slog.Warn("scheduling policy: account not found, skip", "policy_id", p.ID, "account_id", accountID)
			continue
		}
		record := &SchedulingActionRecord{
			PolicyID:  p.ID,
			AccountID: accountID,
			MonitorID: p.MonitorID,
			Action:    p.ActionType,
			Reason:    reason,
		}
		switch p.ActionType {
		case SchedulingActionPause:
			until := time.Now().Add(schedulingPauseMaxDuration)
			if p.PauseMinutes > 0 {
				until = time.Now().Add(time.Duration(p.PauseMinutes) * time.Minute)
			}
			if err := s.accountRepo.SetTempUnschedulable(ctx, accountID, until, "scheduling policy: "+p.Name); err != nil {
				slog.Error("scheduling policy: pause account failed", "policy_id", p.ID, "account_id", accountID, "error", err)
				continue
			}
		case SchedulingActionDeprioritize:
			record.OriginalPriority = account.Priority
			newPriority := account.Priority + p.PriorityDelta
			if _, err := s.accountRepo.BulkUpdate(ctx, []int64{accountID}, AccountBulkUpdate{Priority: &newPriority}); err != nil {
				slog.Error("scheduling policy: deprioritize account failed", "policy_id", p.ID, "account_id", accountID, "error", err)
				continue
			}
		default:
			continue
		}
		if err := s.repo.InsertAction(ctx, record); err != nil {
			slog.Error("scheduling policy: insert action failed", "policy_id", p.ID, "account_id", accountID, "error", err)
		}
		slog.Info("scheduling policy triggered",
			"policy_id", p.ID, "policy", p.Name, "account_id", accountID, "action", p.ActionType, "reason", reason)
	}
}

// recover 恢复策略下所有未恢复的动作并写审计。
func (s *SchedulingPolicyService) recover(ctx context.Context, p *SchedulingPolicy) {
	if s.accountRepo == nil {
		return
	}
	active, err := s.repo.ListUnrestoredActions(ctx, p.ID)
	if err != nil {
		slog.Error("scheduling policy: list unrestored actions failed", "policy_id", p.ID, "error", err)
		return
	}
	if len(active) == 0 {
		return
	}
	restoredIDs := make([]int64, 0, len(active))
	for _, a := range active {
		switch a.Action {
		case SchedulingActionPause:
			if err := s.accountRepo.ClearTempUnschedulable(ctx, a.AccountID); err != nil {
				slog.Error("scheduling policy: clear pause failed", "policy_id", p.ID, "account_id", a.AccountID, "error", err)
				continue
			}
		case SchedulingActionDeprioritize:
			original := a.OriginalPriority
			if _, err := s.accountRepo.BulkUpdate(ctx, []int64{a.AccountID}, AccountBulkUpdate{Priority: &original}); err != nil {
				slog.Error("scheduling policy: restore priority failed", "policy_id", p.ID, "account_id", a.AccountID, "error", err)
				continue
			}
		default:
			continue
		}
		restoredIDs = append(restoredIDs, a.ID)
		recoverRecord := &SchedulingActionRecord{
			PolicyID:  p.ID,
			AccountID: a.AccountID,
			MonitorID: p.MonitorID,
			Action:    SchedulingActionRecover,
			Reason:    fmt.Sprintf("recovered after %d consecutive successes", p.RecoverConsecutiveSuccesses),
			Restored:  true,
		}
		if err := s.repo.InsertAction(ctx, recoverRecord); err != nil {
			slog.Error("scheduling policy: insert recover action failed", "policy_id", p.ID, "account_id", a.AccountID, "error", err)
		}
		slog.Info("scheduling policy recovered", "policy_id", p.ID, "policy", p.Name, "account_id", a.AccountID)
	}
	if len(restoredIDs) > 0 {
		if err := s.repo.MarkActionsRestored(ctx, restoredIDs); err != nil {
			slog.Error("scheduling policy: mark restored failed", "policy_id", p.ID, "error", err)
		}
	}
}

// resetState 清空策略的进程内计数器（配置变更/删除后重新累计）。
func (s *SchedulingPolicyService) resetState(policyID int64) {
	s.mu.Lock()
	delete(s.states, policyID)
	s.mu.Unlock()
}
