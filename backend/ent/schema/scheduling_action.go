package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// SchedulingAction holds the schema definition for the SchedulingAction entity.
// 调度策略动作审计：记录每次策略触发/恢复对账号做了什么，以及恢复所需的原始状态。
type SchedulingAction struct {
	ent.Schema
}

func (SchedulingAction) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "scheduling_actions"},
	}
}

func (SchedulingAction) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("policy_id"),
		field.Int64("account_id"),
		field.Int64("monitor_id"),
		field.String("action").
			MaxLen(32).
			Comment("pause / deprioritize / recover"),
		field.String("reason").
			Default("").
			MaxLen(500),
		field.Int("original_priority").
			Default(0).
			Comment("deprioritize 动作执行前的账号优先级，恢复时还原"),
		field.Bool("restored").
			Default(false).
			Comment("该动作是否已被恢复（recover 动作本身恒为 true）"),
		field.Time("created_at").
			Immutable().
			Annotations(entsql.Default("CURRENT_TIMESTAMP")),
	}
}

func (SchedulingAction) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("policy_id", "created_at"),
		index.Fields("account_id", "restored"),
	}
}
