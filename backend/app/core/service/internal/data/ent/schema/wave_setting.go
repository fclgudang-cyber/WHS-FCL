package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	"github.com/tx7do/go-crud/entgo/mixin"
)

// WaveSetting 波次设置（per tenant, per type）：单包/多件两种配置。
type WaveSetting struct {
	ent.Schema
}

func (WaveSetting) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{
			Table:     "wms_wave_settings",
			Charset:   "utf8mb4",
			Collation: "utf8mb4_bin",
		},
		entsql.WithComments(true),
		schema.Comment("波次设置表"),
	}
}

func (WaveSetting) Fields() []ent.Field {
	return []ent.Field{
		field.Enum("wave_type").
			Comment("SINGLE/MULTI").
			NamedValues(
				"Single", "SINGLE",
				"Multi", "MULTI",
			).
			Default("SINGLE"),

		field.Int("max_orders_per_cart").
			Comment("Maks order per keranjang/ picking cart").
			Default(20),

		field.Int("max_sku_per_wave").
			Comment("Maks SKU per wave; 0 = unlimited").
			Optional().
			Nillable(),

		field.Enum("grouping_rule").
			Comment("PER_WAREHOUSE/PER_ZONE/PER_COURIER/NONE").
			NamedValues(
				"PerWarehouse", "PER_WAREHOUSE",
				"PerZone", "PER_ZONE",
				"PerCourier", "PER_COURIER",
				"None", "NONE",
			).
			Default("PER_WAREHOUSE").
			Optional().
			Nillable(),

		field.Bool("auto_split").
			Comment("Auto-pecah wave jika melebihi kapasitas").
			Default(true),

		field.Int("default_cart_capacity").
			Comment("Default kapasitas keranjang picking").
			Default(20),
	}
}

func (WaveSetting) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.AutoIncrementId{},
		mixin.TimeAt{},
		mixin.TenantID[uint32]{},
	}
}

func (WaveSetting) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("tenant_id", "wave_type").
			Unique().
			StorageKey("idx_wms_wave_setting_tenant_type"),
	}
}
