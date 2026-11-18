package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	"github.com/tx7do/go-crud/entgo/mixin"
)

// SortSlot 1 kotak di Sortir Keranjang — 1 order Multi (per sesi sesi).
type SortSlot struct {
	ent.Schema
}

func (SortSlot) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{
			Table:     "wms_sort_slots",
			Charset:   "utf8mb4",
			Collation: "utf8mb4_bin",
		},
		entsql.WithComments(true),
		schema.Comment("Sortir slot 表"),
	}
}

func (SortSlot) Fields() []ent.Field {
	return []ent.Field{
		field.Uint32("session_id").
			Comment("FK wms_sort_sessions.id"),

		field.Uint32("sort_basket_id").
			Comment("FK sort_baskets.id"),

		field.Int("slot_number").
			Comment("No. kotak (1..N)"),

		field.Uint32("order_id").
			Comment("").
			Optional().
			Nillable(),

		field.Enum("status").
			Comment("EMPTY/FILLED/COMPLETE/PACKED").
			NamedValues(
				"Empty", "EMPTY",
				"Filled", "FILLED",
				"Complete", "COMPLETE",
				"Packed", "PACKED",
			).
			Default("EMPTY").
			Optional().
			Nillable(),

		field.Int("qty_scanned").
			Comment("Jumlah item masuk").
			Default(0).
			Optional().
			Nillable(),
	}
}

func (SortSlot) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.AutoIncrementId{},
		mixin.TimeAt{},
	}
}

func (SortSlot) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("session_id", "slot_number").
			Unique().
			StorageKey("idx_wms_sort_slot_session_number"),
	}
}
