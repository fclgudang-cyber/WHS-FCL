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

// PickingCart 拣货 Keranjang（cart/keranjang）：移动端扫码带货作业单元。
// 同一只 keranjang 同时只能分配给一个运行中的任务。
type PickingCart struct {
	ent.Schema
}

func (PickingCart) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{
			Table:     "wms_picking_carts",
			Charset:   "utf8mb4",
			Collation: "utf8mb4_bin",
		},
		entsql.WithComments(true),
		schema.Comment("拣货Kerancang表"),
	}
}

// Fields of the PickingCart.
func (PickingCart) Fields() []ent.Field {
	return []ent.Field{
		field.String("cart_number").
			Comment("Keranjang 编号").
			Optional().
			Nillable(),

		field.Enum("status").
			Comment("状态：FREE=空闲，IN_USE=使用中，DISABLED=停用").
			NamedValues(
				"Free", "FREE",
				"InUse", "IN_USE",
				"Disabled", "DISABLED",
			).
			Default("FREE").
			Optional().
			Nillable(),
	}
}

// Mixin of the PickingCart.
func (PickingCart) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.AutoIncrementId{},
		mixin.TimeAt{}, appmixin.SoftDeleteQuery{},
		mixin.OperatorID{},
		mixin.TenantID[uint32]{},
	}
}

// Indexes of the PickingCart.
func (PickingCart) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("tenant_id", "cart_number").
			Unique().
			StorageKey("idx_wms_picking_cart_tenant_number"),
	}
}
