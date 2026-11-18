package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	"github.com/tx7do/go-crud/entgo/mixin"
	appmixin "github.com/Mikdad-Force/Ghost/pkg/entgo/mixin"
)

// PickingTask 拣货任务：波次 lepaskan 后生成，与所选 Keranjang 绑定。
// 状态机：WAITING → RUNNING → DONE；WAITING 可 CANCELLED。
type PickingTask struct {
	ent.Schema
}

func (PickingTask) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{
			Table:     "wms_picking_tasks",
			Charset:   "utf8mb4",
			Collation: "utf8mb4_bin",
		},
		entsql.WithComments(true),
		schema.Comment("拣货任务表"),
	}
}

// Fields of the PickingTask.
func (PickingTask) Fields() []ent.Field {
	return []ent.Field{
		field.String("task_number").
			Comment("任务号（服务端生成，租户范围内唯一）").
			Optional().
			Nillable(),

		field.Uint32("wave_id").
			Comment("来源波次ID").
			Optional().
			Nillable(),

		field.Uint32("cart_id").
			Comment("绑定的 Keranjang ID").
			Optional().
			Nillable(),

		field.Enum("status").
			Comment("状态：WAITING=等待，RUNNING=进行中，DONE=完成，CANCELLED=取消").
			NamedValues(
				"Waiting", "WAITING",
				"Running", "RUNNING",
				"Done", "DONE",
				"Cancelled", "CANCELLED",
			).
			Default("WAITING").
			Optional().
			Nillable(),
	}
}

// Mixin of the PickingTask.
func (PickingTask) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.AutoIncrementId{},
		mixin.TimeAt{}, appmixin.SoftDeleteQuery{},
		mixin.OperatorID{},
		mixin.TenantID[uint32]{},
	}
}

// Indexes of the PickingTask.
func (PickingTask) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("tenant_id", "task_number").
			Unique().
			StorageKey("idx_wms_picking_task_tenant_number"),
		index.Fields("wave_id").StorageKey("idx_wms_picking_task_wave"),
	}
}
