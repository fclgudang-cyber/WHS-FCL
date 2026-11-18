package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	"github.com/tx7do/go-crud/entgo/mixin"
)

// WaveOrder 订单↔波次 关系表（每个 order 入波次一行），用于分拣时按订单分配 kotak。
type WaveOrder struct {
	ent.Schema
}

func (WaveOrder) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{
			Table:     "wms_wave_orders",
			Charset:   "utf8mb4",
			Collation: "utf8mb4_bin",
		},
		entsql.WithComments(true),
		schema.Comment("波次-订单关联表"),
	}
}

func (WaveOrder) Fields() []ent.Field {
	return []ent.Field{
		field.Uint32("wave_id").
			Comment("FK waves.id"),
		field.Uint32("order_id").
			Comment("FK ord_orders.id"),
	}
}

func (WaveOrder) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.AutoIncrementId{},
		mixin.TenantID[uint32]{},
	}
}

func (WaveOrder) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("wave_id", "order_id").
			Unique().
			StorageKey("idx_wms_wave_order_wave_order"),
	}
}
