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

// PickingTaskLog 拣货审计日志：每一步扫码/改数量都记录（谁、何时、什么 qty）。
type PickingTaskLog struct {
	ent.Schema
}

func (PickingTaskLog) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{
			Table:     "wms_picking_task_logs",
			Charset:   "utf8mb4",
			Collation: "utf8mb4_bin",
		},
		entsql.WithComments(true),
		schema.Comment("拣货任务审计日志表"),
	}
}

// Fields of the PickingTaskLog.
func (PickingTaskLog) Fields() []ent.Field {
	return []ent.Field{
		field.Uint32("task_id").
			Comment("任务ID").
			Optional().
			Nillable(),

		field.Uint32("item_id").
			Comment("明细ID（0 表示任务级动作）").
			Optional().
			Nillable(),

		field.String("action").
			Comment("动作：SCAN_CART / SCAN_LOCATION / SCAN_SKU / SCAN_QTY / SHORT_PICK / COMPLETE_ITEM / CANCEL / RELEASE").
			Optional().
			Nillable(),

		field.String("detail").
			Comment("明细（扫码值/reason/error）").
			Optional().
			Nillable(),

		field.String("sku_code").
			Comment("SKU 编码").
			Optional().
			Nillable(),

		field.Int64("qty").
			Comment("数量（正数）").
			Optional().
			Nillable(),
	}
}

// Mixin of the PickingTaskLog.
func (PickingTaskLog) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.AutoIncrementId{},
		mixin.TimeAt{}, appmixin.SoftDeleteQuery{},
		mixin.OperatorID{},
		mixin.TenantID[uint32]{},
	}
}

// Indexes of the PickingTaskLog.
func (PickingTaskLog) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("task_id").StorageKey("idx_wms_picking_task_log_task"),
		index.Fields("tenant_id", "created_at").StorageKey("idx_wms_picking_task_log_tenant_created"),
	}
}
