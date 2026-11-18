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

// Order 订单（Marketplace / 手动补录）：接收外部订单，经确认生成出库拣货单。
// 状态机：PENDING → CONFIRMED → FULFILLING → COMPLETED；PENDING/CONFIRMED 可取消。
type Order struct {
	ent.Schema
}

func (Order) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{
			Table:     "ord_orders",
			Charset:   "utf8mb4",
			Collation: "utf8mb4_bin",
		},
		entsql.WithComments(true),
		schema.Comment("订单表"),
	}
}

// Fields of the Order.
func (Order) Fields() []ent.Field {
	return []ent.Field{
		field.String("order_number").
			Comment("单号（服务端生成，租户范围内唯一）").
			Optional().
			Nillable(),

		field.Enum("source").
			Comment("订单来源：MANUAL=手动补录，SHOPEE/TOKOPEDIA/LAZADA/TIKTOK=Marketplace").
			NamedValues(
				"Manual", "MANUAL",
				"Shopee", "SHOPEE",
				"Tokopedia", "TOKOPEDIA",
				"Lazada", "LAZADA",
				"TikTok", "TIKTOK",
				"Other", "OTHER",
			).
			Default("MANUAL").
			Optional().
			Nillable(),

		field.String("source_ref").
			Comment("外部平台订单号（Marketplace），手动订单为空").
			Optional().
			Nillable(),

		field.String("source_shop").
			Comment("店铺名称/店铺编码（Marketplace 店铺）").
			Optional().
			Nillable(),

		field.String("customer_name").
			Comment("收货人姓名").
			Optional().
			Nillable(),

		field.String("customer_phone").
			Comment("收货人电话").
			Optional().
			Nillable(),

		field.String("shipping_address").
			Comment("收货地址").
			Optional().
			Nillable(),

		field.String("warehouse_code").
			Comment("发货仓库编码（缺省 MAIN）").
			Optional().
			Nillable(),

		field.Enum("status").
			Comment("状态：PENDING=待确认，CONFIRMED=已确认，FULFILLING=履约中，COMPLETED=已完结，CANCELLED=已取消").
			NamedValues(
				"Pending", "PENDING",
				"Confirmed", "CONFIRMED",
				"Fulfilling", "FULFILLING",
				"Sorted", "SORTED",
				"Packed", "PACKED",
				"Completed", "COMPLETED",
				"Cancelled", "CANCELLED",
			).
			Default("PENDING").
			Optional().
			Nillable(),

		field.Uint32("picking_id").
			Comment("履约生成的出库拣货单ID").
			Optional().
			Nillable(),

		field.Int64("total_amount_cents").
			Comment("订单总额（分）").
			Optional().
			Nillable(),

		field.String("currency").
			Comment("币种，默认 IDR").
			Optional().
			Nillable(),

		field.Time("ordered_at").
			Comment("外部平台下单时间").
			Optional().
			Nillable(),
	}
}

// Edges of the Order.
func (Order) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("lines", OrderLine.Type),
	}
}

// Mixin of the Order.
func (Order) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.AutoIncrementId{},
		mixin.TimeAt{}, appmixin.SoftDeleteQuery{},
		mixin.OperatorID{},
		mixin.Remark{},
		mixin.TenantID[uint32]{},
	}
}

// Indexes of the Order.
func (Order) Indexes() []ent.Index {
	return []ent.Index{
		// 租户范围内单号唯一
		index.Fields("tenant_id", "order_number").
			Unique().
			StorageKey("idx_ord_order_tenant_number"),

		// 外部平台订单幂等键：同一租户 + 渠道 + 平台单号唯一
		index.Fields("tenant_id", "source", "source_ref").
			Unique().
			StorageKey("idx_ord_order_tenant_source_ref"),

		// 按仓库 + 状态查待履约订单
		index.Fields("tenant_id", "warehouse_code", "status").
			StorageKey("idx_ord_order_tenant_wh_status"),
	}
}
