package service

import (
	"context"
	"time"
)

// 调度策略动作类型。
const (
	SchedulingActionPause        = "pause"
	SchedulingActionDeprioritize = "deprioritize"
	SchedulingActionRecover      = "recover"
)

// SchedulingPolicy 渠道调度策略领域模型。
// 条件源为一个渠道监控（channel_monitor），动作目标为若干账号。
type SchedulingPolicy struct {
	ID                          int64
	Name                        string
	Enabled                     bool
	MonitorID                   int64
	AccountIDs                  []int64
	TriggerConsecutiveFailures  int
	TriggerLatencyMs            int
	ActionType                  string // pause / deprioritize
	PauseMinutes                int    // 0 = 停用至恢复条件满足
	PriorityDelta               int
	RecoverConsecutiveSuccesses int // 0 = 不自动恢复
	CooldownMinutes             int
	CreatedAt                   time.Time
	UpdatedAt                   time.Time
}

// SchedulingActionRecord 策略动作审计记录。
type SchedulingActionRecord struct {
	ID               int64
	PolicyID         int64
	AccountID        int64
	MonitorID        int64
	Action           string
	Reason           string
	OriginalPriority int
	Restored         bool
	CreatedAt        time.Time
}

// SchedulingPolicyListParams 策略列表过滤参数。
type SchedulingPolicyListParams struct {
	Page     int
	PageSize int
	Enabled  *bool
	Search   string
}

// SchedulingActionListParams 动作历史查询参数。
type SchedulingActionListParams struct {
	Page      int
	PageSize  int
	PolicyID  int64 // 0 = 全部
	AccountID int64 // 0 = 全部
}

// SchedulingPolicyRepository 调度策略数据访问接口。
type SchedulingPolicyRepository interface {
	Create(ctx context.Context, p *SchedulingPolicy) error
	GetByID(ctx context.Context, id int64) (*SchedulingPolicy, error)
	Update(ctx context.Context, p *SchedulingPolicy) error
	Delete(ctx context.Context, id int64) error
	List(ctx context.Context, params SchedulingPolicyListParams) ([]*SchedulingPolicy, int64, error)
	ListEnabledByMonitorID(ctx context.Context, monitorID int64) ([]*SchedulingPolicy, error)

	InsertAction(ctx context.Context, a *SchedulingActionRecord) error
	ListActions(ctx context.Context, params SchedulingActionListParams) ([]*SchedulingActionRecord, int64, error)
	// ListUnrestoredActions 返回策略下未恢复的触发动作（pause/deprioritize）。
	ListUnrestoredActions(ctx context.Context, policyID int64) ([]*SchedulingActionRecord, error)
	MarkActionsRestored(ctx context.Context, ids []int64) error
	// LatestTriggerAt 返回策略最近一次触发动作（非 recover）的时间；无记录返回 nil。
	LatestTriggerAt(ctx context.Context, policyID int64) (*time.Time, error)
}
