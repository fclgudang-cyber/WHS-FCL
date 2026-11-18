package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	"github.com/tx7do/go-crud/entgo/mixin"
)

// SortSlotItem item yang harus masuk ke kotak.
type SortSlotItem struct {
	ent.Schema
}

func (SortSlotItem) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{
			Table:     "wms_sort_slot_items",
			Charset:   "utf8mb4",
			Collation: "utf8mb4_bin",
		},
		entsql.WithComments(true),
		schema.Comment("Sortir slot item 表"),
	}
}

func (SortSlotItem) Fields() []ent.Field {
	return []ent.Field{
		field.Uint32("slot_id").
			Comment("FK sort_slots.id"),

		field.String("sku_code").
			Comment(""),

		field.String("product_name").
			Comment("").
			Optional().
			Nillable(),

		field.Int64("qty_required").
			Comment(""),

		field.Int64("qty_scanned").
			Comment("").
			Default(0).
			Optional().
			Nillable(),

		field.Enum("status").
			Comment("PENDING/DONE").
			NamedValues(
				"Pending", "PENDING",
				"Done", "DONE",
			).
			Default("PENDING").
			Optional().
			Nillable(),
	}
}

func (SortSlotItem) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.AutoIncrementId{},
		mixin.TimeAt{},
	}
}

func (SortSlotItem) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("slot_id", "sku_code").
			Unique().
			StorageKey("idx_wms_sort_slot_item_slot_sku"),
	}
}
