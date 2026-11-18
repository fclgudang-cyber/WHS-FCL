package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	"github.com/tx7do/go-crud/entgo/mixin"
	appmixin "github.com/Mikdad-Force/Ghost/pkg/entgo/mixin"
)

// OrderLine 订单明细：SKU + 数量 + 单价。
type OrderLine struct {
	ent.Schema
}

func (OrderLine) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{
			Table:     "ord_order_lines",
			Charset:   "utf8mb4",
			Collation: "utf8mb4_bin",
		},
		entsql.WithComments(true),
		schema.Comment("订单明细表"),
	}
}

// Fields of the OrderLine.
func (OrderLine) Fields() []ent.Field {
	return []ent.Field{
		field.Uint32("order_id").
			Comment("所属订单ID").
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

		field.Int64("quantity").
			Comment("数量").
			Optional().
			Nillable(),

		field.Int64("unit_price_cents").
			Comment("单价（分）").
			Optional().
			Nillable(),

		field.String("remark").
			Comment("备注").
			Optional().
			Nillable(),
	}
}

// Mixin of the OrderLine.
func (OrderLine) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.AutoIncrementId{},
		mixin.TimeAt{}, appmixin.SoftDeleteQuery{},
		mixin.OperatorID{},
		mixin.TenantID[uint32]{},
	}
}

// Edges of the OrderLine.
func (OrderLine) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("order", Order.Type).
			Ref("lines").
			Field("order_id").
			Unique().
			Annotations(entsql.Annotation{OnDelete: entsql.Cascade}),
	}
}

// Indexes of the OrderLine.
func (OrderLine) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("order_id").StorageKey("idx_ord_order_line_order"),
		index.Fields("tenant_id", "sku_code").StorageKey("idx_ord_order_line_tenant_sku"),
	}
}
