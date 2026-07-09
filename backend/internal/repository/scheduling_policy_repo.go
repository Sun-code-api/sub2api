package repository

import (
	"context"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/schedulingaction"
	"github.com/Wei-Shaw/sub2api/ent/schedulingpolicy"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

// schedulingPolicyRepository 实现 service.SchedulingPolicyRepository。
type schedulingPolicyRepository struct {
	client *dbent.Client
}

// NewSchedulingPolicyRepository 创建仓储实例。
func NewSchedulingPolicyRepository(client *dbent.Client) service.SchedulingPolicyRepository {
	return &schedulingPolicyRepository{client: client}
}

// ---------- 策略 CRUD ----------

func (r *schedulingPolicyRepository) Create(ctx context.Context, p *service.SchedulingPolicy) error {
	client := clientFromContext(ctx, r.client)
	created, err := client.SchedulingPolicy.Create().
		SetName(p.Name).
		SetEnabled(p.Enabled).
		SetMonitorID(p.MonitorID).
		SetAccountIds(emptyInt64SliceIfNil(p.AccountIDs)).
		SetTriggerConsecutiveFailures(p.TriggerConsecutiveFailures).
		SetTriggerLatencyMs(p.TriggerLatencyMs).
		SetActionType(p.ActionType).
		SetPauseMinutes(p.PauseMinutes).
		SetPriorityDelta(p.PriorityDelta).
		SetRecoverConsecutiveSuccesses(p.RecoverConsecutiveSuccesses).
		SetCooldownMinutes(p.CooldownMinutes).
		Save(ctx)
	if err != nil {
		return err
	}
	*p = *schedulingPolicyFromEnt(created)
	return nil
}

func (r *schedulingPolicyRepository) GetByID(ctx context.Context, id int64) (*service.SchedulingPolicy, error) {
	client := clientFromContext(ctx, r.client)
	found, err := client.SchedulingPolicy.Get(ctx, id)
	if err != nil {
		if dbent.IsNotFound(err) {
			return nil, service.ErrSchedulingPolicyNotFound
		}
		return nil, err
	}
	return schedulingPolicyFromEnt(found), nil
}

func (r *schedulingPolicyRepository) Update(ctx context.Context, p *service.SchedulingPolicy) error {
	client := clientFromContext(ctx, r.client)
	updated, err := client.SchedulingPolicy.UpdateOneID(p.ID).
		SetName(p.Name).
		SetEnabled(p.Enabled).
		SetMonitorID(p.MonitorID).
		SetAccountIds(emptyInt64SliceIfNil(p.AccountIDs)).
		SetTriggerConsecutiveFailures(p.TriggerConsecutiveFailures).
		SetTriggerLatencyMs(p.TriggerLatencyMs).
		SetActionType(p.ActionType).
		SetPauseMinutes(p.PauseMinutes).
		SetPriorityDelta(p.PriorityDelta).
		SetRecoverConsecutiveSuccesses(p.RecoverConsecutiveSuccesses).
		SetCooldownMinutes(p.CooldownMinutes).
		Save(ctx)
	if err != nil {
		if dbent.IsNotFound(err) {
			return service.ErrSchedulingPolicyNotFound
		}
		return err
	}
	*p = *schedulingPolicyFromEnt(updated)
	return nil
}

func (r *schedulingPolicyRepository) Delete(ctx context.Context, id int64) error {
	client := clientFromContext(ctx, r.client)
	if err := client.SchedulingPolicy.DeleteOneID(id).Exec(ctx); err != nil {
		if dbent.IsNotFound(err) {
			return service.ErrSchedulingPolicyNotFound
		}
		return err
	}
	return nil
}

func (r *schedulingPolicyRepository) List(ctx context.Context, params service.SchedulingPolicyListParams) ([]*service.SchedulingPolicy, int64, error) {
	client := clientFromContext(ctx, r.client)
	q := client.SchedulingPolicy.Query()
	if params.Enabled != nil {
		q = q.Where(schedulingpolicy.EnabledEQ(*params.Enabled))
	}
	if params.Search != "" {
		q = q.Where(schedulingpolicy.NameContainsFold(params.Search))
	}
	total, err := q.Clone().Count(ctx)
	if err != nil {
		return nil, 0, err
	}
	rows, err := q.
		Order(dbent.Desc(schedulingpolicy.FieldID)).
		Offset((params.Page - 1) * params.PageSize).
		Limit(params.PageSize).
		All(ctx)
	if err != nil {
		return nil, 0, err
	}
	items := make([]*service.SchedulingPolicy, 0, len(rows))
	for _, row := range rows {
		items = append(items, schedulingPolicyFromEnt(row))
	}
	return items, int64(total), nil
}

func (r *schedulingPolicyRepository) ListEnabledByMonitorID(ctx context.Context, monitorID int64) ([]*service.SchedulingPolicy, error) {
	client := clientFromContext(ctx, r.client)
	rows, err := client.SchedulingPolicy.Query().
		Where(schedulingpolicy.MonitorIDEQ(monitorID), schedulingpolicy.EnabledEQ(true)).
		All(ctx)
	if err != nil {
		return nil, err
	}
	items := make([]*service.SchedulingPolicy, 0, len(rows))
	for _, row := range rows {
		items = append(items, schedulingPolicyFromEnt(row))
	}
	return items, nil
}

// ---------- 动作审计 ----------

func (r *schedulingPolicyRepository) InsertAction(ctx context.Context, a *service.SchedulingActionRecord) error {
	client := clientFromContext(ctx, r.client)
	created, err := client.SchedulingAction.Create().
		SetPolicyID(a.PolicyID).
		SetAccountID(a.AccountID).
		SetMonitorID(a.MonitorID).
		SetAction(a.Action).
		SetReason(a.Reason).
		SetOriginalPriority(a.OriginalPriority).
		SetRestored(a.Restored).
		Save(ctx)
	if err != nil {
		return err
	}
	*a = *schedulingActionFromEnt(created)
	return nil
}

func (r *schedulingPolicyRepository) ListActions(ctx context.Context, params service.SchedulingActionListParams) ([]*service.SchedulingActionRecord, int64, error) {
	client := clientFromContext(ctx, r.client)
	q := client.SchedulingAction.Query()
	if params.PolicyID > 0 {
		q = q.Where(schedulingaction.PolicyIDEQ(params.PolicyID))
	}
	if params.AccountID > 0 {
		q = q.Where(schedulingaction.AccountIDEQ(params.AccountID))
	}
	total, err := q.Clone().Count(ctx)
	if err != nil {
		return nil, 0, err
	}
	rows, err := q.
		Order(dbent.Desc(schedulingaction.FieldID)).
		Offset((params.Page - 1) * params.PageSize).
		Limit(params.PageSize).
		All(ctx)
	if err != nil {
		return nil, 0, err
	}
	items := make([]*service.SchedulingActionRecord, 0, len(rows))
	for _, row := range rows {
		items = append(items, schedulingActionFromEnt(row))
	}
	return items, int64(total), nil
}

func (r *schedulingPolicyRepository) ListUnrestoredActions(ctx context.Context, policyID int64) ([]*service.SchedulingActionRecord, error) {
	client := clientFromContext(ctx, r.client)
	rows, err := client.SchedulingAction.Query().
		Where(
			schedulingaction.PolicyIDEQ(policyID),
			schedulingaction.RestoredEQ(false),
			schedulingaction.ActionNEQ(service.SchedulingActionRecover),
		).
		All(ctx)
	if err != nil {
		return nil, err
	}
	items := make([]*service.SchedulingActionRecord, 0, len(rows))
	for _, row := range rows {
		items = append(items, schedulingActionFromEnt(row))
	}
	return items, nil
}

func (r *schedulingPolicyRepository) MarkActionsRestored(ctx context.Context, ids []int64) error {
	if len(ids) == 0 {
		return nil
	}
	client := clientFromContext(ctx, r.client)
	_, err := client.SchedulingAction.Update().
		Where(schedulingaction.IDIn(ids...)).
		SetRestored(true).
		Save(ctx)
	return err
}

func (r *schedulingPolicyRepository) LatestTriggerAt(ctx context.Context, policyID int64) (*time.Time, error) {
	client := clientFromContext(ctx, r.client)
	row, err := client.SchedulingAction.Query().
		Where(
			schedulingaction.PolicyIDEQ(policyID),
			schedulingaction.ActionNEQ(service.SchedulingActionRecover),
		).
		Order(dbent.Desc(schedulingaction.FieldID)).
		First(ctx)
	if err != nil {
		if dbent.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	t := row.CreatedAt
	return &t, nil
}

// ---------- 转换 ----------

func emptyInt64SliceIfNil(s []int64) []int64 {
	if s == nil {
		return []int64{}
	}
	return s
}

func schedulingPolicyFromEnt(e *dbent.SchedulingPolicy) *service.SchedulingPolicy {
	return &service.SchedulingPolicy{
		ID:                          e.ID,
		Name:                        e.Name,
		Enabled:                     e.Enabled,
		MonitorID:                   e.MonitorID,
		AccountIDs:                  e.AccountIds,
		TriggerConsecutiveFailures:  e.TriggerConsecutiveFailures,
		TriggerLatencyMs:            e.TriggerLatencyMs,
		ActionType:                  e.ActionType,
		PauseMinutes:                e.PauseMinutes,
		PriorityDelta:               e.PriorityDelta,
		RecoverConsecutiveSuccesses: e.RecoverConsecutiveSuccesses,
		CooldownMinutes:             e.CooldownMinutes,
		CreatedAt:                   e.CreatedAt,
		UpdatedAt:                   e.UpdatedAt,
	}
}

func schedulingActionFromEnt(e *dbent.SchedulingAction) *service.SchedulingActionRecord {
	return &service.SchedulingActionRecord{
		ID:               e.ID,
		PolicyID:         e.PolicyID,
		AccountID:        e.AccountID,
		MonitorID:        e.MonitorID,
		Action:           e.Action,
		Reason:           e.Reason,
		OriginalPriority: e.OriginalPriority,
		Restored:         e.Restored,
		CreatedAt:        e.CreatedAt,
	}
}
