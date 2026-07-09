package schema

import (
	"github.com/Wei-Shaw/sub2api/ent/schema/mixins"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// SchedulingPolicy holds the schema definition for the SchedulingPolicy entity.
// 渠道调度策略：将渠道监控（channel_monitors）的检测结果与账号调度动作联动。
// 触发条件满足时对目标账号执行临时停用/降低优先级，恢复条件满足后自动还原。
type SchedulingPolicy struct {
	ent.Schema
}

func (SchedulingPolicy) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "scheduling_policies"},
	}
}

func (SchedulingPolicy) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixins.TimeMixin{},
	}
}

func (SchedulingPolicy) Fields() []ent.Field {
	return []ent.Field{
		field.String("name").
			NotEmpty().
			MaxLen(100),
		field.Bool("enabled").
			Default(true),
		field.Int64("monitor_id").
			Comment("条件源：channel_monitors.id，监控被删除时策略停摆但保留配置"),
		field.JSON("account_ids", []int64{}).
			Default([]int64{}).
			Comment("动作目标账号 ID 列表"),
		field.Int("trigger_consecutive_failures").
			Default(3).
			Range(1, 100).
			Comment("连续失败 N 次后触发（failed/error 或延迟超阈值均计为失败）"),
		field.Int("trigger_latency_ms").
			Default(0).
			Range(0, 600000).
			Comment("延迟阈值（毫秒），检测延迟超过该值计为一次失败；0 表示不启用延迟条件"),
		field.String("action_type").
			Default("pause").
			MaxLen(32).
			Comment("触发动作：pause（临时停用）或 deprioritize（降低优先级）"),
		field.Int("pause_minutes").
			Default(0).
			Range(0, 10080).
			Comment("pause 动作的停用时长（分钟）；0 表示停用至恢复条件满足"),
		field.Int("priority_delta").
			Default(10).
			Range(1, 1000).
			Comment("deprioritize 动作在原优先级上增加的数值（数值越大优先级越低）"),
		field.Int("recover_consecutive_successes").
			Default(2).
			Range(0, 100).
			Comment("连续成功 N 次后自动恢复；0 表示不自动恢复"),
		field.Int("cooldown_minutes").
			Default(10).
			Range(0, 1440).
			Comment("两次触发之间的最小间隔（分钟），防止抖动反复触发"),
	}
}

func (SchedulingPolicy) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("monitor_id"),
		index.Fields("enabled"),
	}
}
