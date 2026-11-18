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

// PickingBasket 拣货 Keranjang（picking cart）。
type PickingBasket struct {
	ent.Schema
}

func (PickingBasket) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{
			Table:     "wms_picking_baskets",
			Charset:   "utf8mb4",
			Collation: "utf8mb4_bin",
		},
		entsql.WithComments(true),
		schema.Comment("拣货 Keranjang 表"),
	}
}

func (PickingBasket) Fields() []ent.Field {
	return []ent.Field{
		field.String("code").
			Comment("Keranjang code (PS01, PM01…)").
			NotEmpty().
			Optional().
			Nillable(),

		field.Enum("basket_type").
			Comment("SINGLE/MULTI").
			NamedValues(
				"Single", "SINGLE",
				"Multi", "MULTI",
			).
			Default("SINGLE").
			Optional().
			Nillable(),

		field.Int("capacity").
			Comment("Kapasitas order").
			Default(20),

		field.Enum("status").
			Comment("FREE/IN_USE/WAITING_SORT/WAITING_PACK").
			NamedValues(
				"Free", "FREE",
				"InUse", "IN_USE",
				"WaitingSort", "WAITING_SORT",
				"WaitingPack", "WAITING_PACK",
			).
			Default("FREE").
			Optional().
			Nillable(),

		field.Uint32("current_wave_id").
			Comment("").
			Optional().
			Nillable(),
		field.Uint32("current_task_id").
			Comment("").
			Optional().
			Nillable(),
	}
}

func (PickingBasket) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.AutoIncrementId{},
		mixin.TimeAt{}, appmixin.SoftDeleteQuery{},
		mixin.OperatorID{},
		mixin.TenantID[uint32]{},
	}
}

func (PickingBasket) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("tenant_id", "code").
			Unique().
			StorageKey("idx_wms_picking_basket_tenant_code"),
	}
}
