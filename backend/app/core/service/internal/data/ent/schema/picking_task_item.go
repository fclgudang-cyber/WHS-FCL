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

// PickingTaskItem 拣货任务明细：由波次 SKU 行生成，记录扫码进度。
type PickingTaskItem struct {
	ent.Schema
}

func (PickingTaskItem) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{
			Table:     "wms_picking_task_items",
			Charset:   "utf8mb4",
			Collation: "utf8mb4_bin",
		},
		entsql.WithComments(true),
		schema.Comment("拣货任务明细表"),
	}
}

// Fields of the PickingTaskItem.
func (PickingTaskItem) Fields() []ent.Field {
	return []ent.Field{
		field.Uint32("task_id").
			Comment("所属任务ID").
			Optional().
			Nillable(),

		field.String("sku_code").
			Comment("SKU 编码").
			Optional().
			Nillable(),

		field.String("product_name").
			Comment("商品名称").
			Optional().
			Nillable(),

		field.Int64("qty_required").
			Comment("需要数量").
			Optional().
			Nillable(),

		field.Int64("qty_picked").
			Comment("已拣数量").
			Default(0).
			Optional().
			Nillable(),

		field.String("allocated_location_id").
			Comment("分配 lokasi（人读摘要）").
			Optional().
			Nillable(),

		field.String("allocated_location_code").
			Comment("分配 lokasi 编码（扫码匹配用）").
			Optional().
			Nillable(),

		field.Enum("status").
			Comment("状态：PENDING/PICKING/DONE/SHORT").
			NamedValues(
				"Pending", "PENDING",
				"Picking", "PICKING",
				"Done", "DONE",
				"Short", "SHORT",
			).
			Default("PENDING").
			Optional().
			Nillable(),

		field.String("short_reason").
			Comment("short pick 原因").
			Optional().
			Nillable(),
	}
}

// Mixin of the PickingTaskItem.
func (PickingTaskItem) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.AutoIncrementId{},
		mixin.TimeAt{}, appmixin.SoftDeleteQuery{},
		mixin.OperatorID{},
		mixin.TenantID[uint32]{},
	}
}

// Indexes of the PickingTaskItem.
func (PickingTaskItem) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("task_id").StorageKey("idx_wms_picking_task_item_task"),
		index.Fields("tenant_id", "sku_code").StorageKey("idx_wms_picking_task_item_sku"),
	}
}
