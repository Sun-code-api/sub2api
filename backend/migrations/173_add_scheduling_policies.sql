-- 173_add_scheduling_policies.sql
-- 渠道调度策略：监控结果 → 账号调度动作联动（自动临停/降级/恢复）+ 动作审计。

CREATE TABLE IF NOT EXISTS scheduling_policies (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    monitor_id BIGINT NOT NULL,
    account_ids JSONB NOT NULL DEFAULT '[]',
    trigger_consecutive_failures INTEGER NOT NULL DEFAULT 3,
    trigger_latency_ms INTEGER NOT NULL DEFAULT 0,
    action_type VARCHAR(32) NOT NULL DEFAULT 'pause',
    pause_minutes INTEGER NOT NULL DEFAULT 0,
    priority_delta INTEGER NOT NULL DEFAULT 10,
    recover_consecutive_successes INTEGER NOT NULL DEFAULT 2,
    cooldown_minutes INTEGER NOT NULL DEFAULT 10,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS schedulingpolicy_monitor_id ON scheduling_policies (monitor_id);
CREATE INDEX IF NOT EXISTS schedulingpolicy_enabled ON scheduling_policies (enabled);

CREATE TABLE IF NOT EXISTS scheduling_actions (
    id BIGSERIAL PRIMARY KEY,
    policy_id BIGINT NOT NULL,
    account_id BIGINT NOT NULL,
    monitor_id BIGINT NOT NULL,
    action VARCHAR(32) NOT NULL,
    reason VARCHAR(500) NOT NULL DEFAULT '',
    original_priority INTEGER NOT NULL DEFAULT 0,
    restored BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS schedulingaction_policy_id_created_at ON scheduling_actions (policy_id, created_at);
CREATE INDEX IF NOT EXISTS schedulingaction_account_id_restored ON scheduling_actions (account_id, restored);
