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

// SortBasket 拣货 Sortir Keranjang：Multi 时使用，编码 AB01/AB02…。
type SortBasket struct {
	ent.Schema
}

func (SortBasket) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{
			Table:     "wms_sort_baskets",
			Charset:   "utf8mb4",
			Collation: "utf8mb4_bin",
		},
		entsql.WithComments(true),
		schema.Comment("拣货 Sortir Keranjang 表"),
	}
}

func (SortBasket) Fields() []ent.Field {
	return []ent.Field{
		field.String("prefix").
			Comment("Prefix (AB)").
			NotEmpty(),

		field.Int("number").
			Comment("Nomor urut (01-99, 100+ → 3 digit)"),

		field.String("code").
			Comment("Kode unik (prefix + number)").
			NotEmpty(),

		field.Int("box_count").
			Comment("Jumlah kotak/slot").
			Default(20),

		field.Enum("status").
			Comment("FREE/IN_USE/FULL").
			NamedValues(
				"Free", "FREE",
				"InUse", "IN_USE",
				"Full", "FULL",
			).
			Default("FREE").
			Optional().
			Nillable(),
	}
}

func (SortBasket) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.AutoIncrementId{},
		mixin.TimeAt{}, appmixin.SoftDeleteQuery{},
		mixin.OperatorID{},
		mixin.TenantID[uint32]{},
	}
}

func (SortBasket) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("tenant_id", "code").
			Unique().
			StorageKey("idx_wms_sort_basket_tenant_code"),
	}
}
