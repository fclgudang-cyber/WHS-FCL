-- =====================================================================
-- WAREHOUSE 3D — Digital Twin & Custom Layout Designer
-- Schema ini HANYA menyimpan konfigurasi visual + referensi ke data WMS
-- yang sudah ada. Tidak ada duplikat inventory / karyawan / transaksi.
-- Aman dijalankan berulang (idempoten).
-- =====================================================================

-- ---------------------------------------------------------------------
-- 1. Informasi gudang
-- ---------------------------------------------------------------------
create table if not exists wh3d_warehouse (
  id             bigserial primary key,
  code           text not null unique,
  name           text not null,
  description    text,
  address        text,
  length_m       numeric(12, 2) not null default 0,
  width_m        numeric(12, 2) not null default 0,
  height_m       numeric(12, 2) not null default 0,
  area_m2        numeric(14, 2) generated always as (length_m * width_m) stored,
  total_floor    integer not null default 1,
  capacity       numeric(14, 2) not null default 0,
  status         text not null default 'ACTIVE',
  building_shape text not null default 'RECTANGLE',
  wall_thickness numeric(8, 2) not null default 0.2,
  created_at     timestamptz not null default now(),
  updated_at     timestamptz not null default now(),
  created_by     text
);

-- ---------------------------------------------------------------------
-- 2. Layout (draft / published) + version history
-- ---------------------------------------------------------------------
create table if not exists wh3d_layout (
  id            bigserial primary key,
  warehouse_id  bigint not null references wh3d_warehouse(id) on delete cascade,
  code          text not null,
  name          text not null,
  status        text not null default 'DRAFT',
  version       integer not null default 1,
  published_at  timestamptz,
  description   text,
  created_at    timestamptz not null default now(),
  updated_at    timestamptz not null default now(),
  created_by    text,
  building_seeded boolean not null default false,
  unique (warehouse_id, code)
);

-- Kompatibilitas database yang sudah ada.
alter table wh3d_layout add column if not exists building_seeded boolean not null default false;

create table if not exists wh3d_layout_version (
  id           bigserial primary key,
  layout_id    bigint not null references wh3d_layout(id) on delete cascade,
  version      integer not null,
  payload      jsonb not null default '{}'::jsonb,
  note         text,
  created_at   timestamptz not null default now(),
  created_by   text,
  unique (layout_id, version)
);

-- ---------------------------------------------------------------------
-- 3. Objek layout generik (semua komponen bisa diedit bebas)
--    config = JSON bebas per tipe objek (bentuk polygon, level rak, dll.)
-- ---------------------------------------------------------------------
create table if not exists wh3d_object (
  id            bigserial primary key,
  layout_id     bigint not null references wh3d_layout(id) on delete cascade,
  object_uid    text not null,
  object_type   text not null,
  category      text not null default 'BUILDING',
  name          text not null,
  label         text,
  position_x    numeric(12, 3) not null default 0,
  position_y    numeric(12, 3) not null default 0,
  position_z    numeric(12, 3) not null default 0,
  length_m      numeric(12, 3) not null default 1,
  width_m       numeric(12, 3) not null default 1,
  height_m      numeric(12, 3) not null default 1,
  rotation      numeric(7, 2) not null default 0,
  color         text,
  parent_uid    text,
  group_uid     text,
  locked        boolean not null default false,
  visible       boolean not null default true,
  z_index       integer not null default 0,
  area_uid      text,
  rack_uid      text,
  bin_uid       text,
  location_code text,
  sku           text,
  employee_id   text,
  equipment_uid text,
  config        jsonb not null default '{}'::jsonb,
  created_at    timestamptz not null default now(),
  updated_at    timestamptz not null default now(),
  unique (layout_id, object_uid)
);

create index if not exists wh3d_object_layout_idx on wh3d_object (layout_id);
create index if not exists wh3d_object_type_idx on wh3d_object (layout_id, object_type);
create index if not exists wh3d_object_location_idx on wh3d_object (layout_id, location_code);
create index if not exists wh3d_object_config_gin on wh3d_object using gin (config jsonb_path_ops);

-- ---------------------------------------------------------------------
-- 4. Area gudang custom (default + buatan pengguna)
-- ---------------------------------------------------------------------
create table if not exists wh3d_area (
  id           bigserial primary key,
  layout_id    bigint not null references wh3d_layout(id) on delete cascade,
  area_uid     text not null,
  code         text not null,
  name         text not null,
  area_type    text not null default 'CUSTOM',
  shape        text not null default 'RECTANGLE',
  length_m     numeric(12, 3) not null default 1,
  width_m      numeric(12, 3) not null default 1,
  height_m     numeric(12, 3) not null default 0,
  area_m2      numeric(14, 2) generated always as (length_m * width_m) stored,
  position_x   numeric(12, 3) not null default 0,
  position_y   numeric(12, 3) not null default 0,
  position_z   numeric(12, 3) not null default 0,
  rotation     numeric(7, 2) not null default 0,
  color        text not null default '#3B82F6',
  capacity     numeric(14, 2) not null default 0,
  description  text,
  config       jsonb not null default '{}'::jsonb,
  created_at   timestamptz not null default now(),
  updated_at   timestamptz not null default now(),
  unique (layout_id, area_uid)
);

-- ---------------------------------------------------------------------
-- 5. Rak & Bin — bin terhubung ke lokasi WMS (sys_stock_location / stock.lokasi)
-- ---------------------------------------------------------------------
create table if not exists wh3d_rack (
  id            bigserial primary key,
  layout_id     bigint not null references wh3d_layout(id) on delete cascade,
  rack_uid      text not null,
  code          text not null,
  name          text not null,
  rack_type     text not null default 'PALLET_RACK',
  pattern       text not null default 'PARALLEL',
  length_m      numeric(12, 3) not null default 2.7,
  width_m       numeric(12, 3) not null default 1.0,
  height_m      numeric(12, 3) not null default 4.5,
  levels        integer not null default 3,
  bays          integer not null default 1,
  bins          integer not null default 3,
  capacity      numeric(14, 2) not null default 0,
  position_x    numeric(12, 3) not null default 0,
  position_y    numeric(12, 3) not null default 0,
  position_z    numeric(12, 3) not null default 0,
  rotation      numeric(7, 2) not null default 0,
  color         text not null default '#F59E0B',
  config        jsonb not null default '{}'::jsonb,
  created_at    timestamptz not null default now(),
  updated_at    timestamptz not null default now(),
  unique (layout_id, rack_uid)
);

create table if not exists wh3d_bin (
  id            bigserial primary key,
  layout_id     bigint not null references wh3d_layout(id) on delete cascade,
  rack_uid      text not null,
  bin_uid       text not null,
  code          text not null,
  level         integer not null default 1,
  bay           integer not null default 1,
  position_x    numeric(12, 3) not null default 0,
  position_y    numeric(12, 3) not null default 0,
  position_z    numeric(12, 3) not null default 0,
  location_code text,
  capacity      numeric(14, 2) not null default 0,
  occupancy     numeric(14, 2) not null default 0,
  status        text not null default 'EMPTY',
  config        jsonb not null default '{}'::jsonb,
  created_at    timestamptz not null default now(),
  unique (layout_id, bin_uid)
);

create index if not exists wh3d_bin_rack_idx on wh3d_bin (layout_id, rack_uid);
create index if not exists wh3d_bin_location_idx on wh3d_bin (layout_id, location_code);

-- ---------------------------------------------------------------------
-- 6. Equipment / forklift (operator = karyawan, bukan data baru)
-- ---------------------------------------------------------------------
create table if not exists wh3d_equipment (
  id             bigserial primary key,
  layout_id      bigint not null references wh3d_layout(id) on delete cascade,
  equipment_uid  text not null,
  code           text not null,
  name           text not null,
  equipment_type text not null default 'FORKLIFT',
  status         text not null default 'IDLE',
  operator_id    text,
  task_uid       text,
  position_x     numeric(12, 3) not null default 0,
  position_y     numeric(12, 3) not null default 0,
  position_z     numeric(12, 3) not null default 0,
  rotation       numeric(7, 2) not null default 0,
  speed          numeric(8, 3) not null default 1.5,
  path           jsonb not null default '[]'::jsonb,
  color          text not null default '#FBBF24',
  config         jsonb not null default '{}'::jsonb,
  created_at     timestamptz not null default now(),
  updated_at     timestamptz not null default now(),
  unique (layout_id, equipment_uid)
);

-- ---------------------------------------------------------------------
-- 7. Profil tambahan pekerja 3D — relasi ke karyawan (TANPA duplikat data)
-- ---------------------------------------------------------------------
create table if not exists wh3d_employee (
  id             bigserial primary key,
  -- employee_id = karyawan.id (text). Tanpa FK fisik agar aman jika karyawan
  -- berada di schema/DB lain; relasi dijamin oleh backend.
  employee_id    text not null unique,
  photo_url      text,
  warehouse_id   bigint references wh3d_warehouse(id) on delete set null,
  current_area   text,
  activity       text not null default 'IDLE',
  activity_label text,
  task_code      text,
  current_task   text,
  position_x     numeric(12, 3) not null default 0,
  position_y     numeric(12, 3) not null default 0,
  position_z     numeric(12, 3) not null default 0,
  rotation       numeric(7, 2) not null default 0,
  animation      text not null default 'IDLE',
  updated_at     timestamptz not null default now()
);

-- ---------------------------------------------------------------------
-- 8. Library objek custom (GLB / GLTF)
-- ---------------------------------------------------------------------
create table if not exists wh3d_object_library (
  id          bigserial primary key,
  uid         text not null unique,
  name        text not null,
  category    text not null default 'CUSTOM',
  model_url   text not null,
  scale       numeric(10, 4) not null default 1,
  thumbnail   text,
  description text,
  created_at  timestamptz not null default now(),
  created_by  text
);

-- ---------------------------------------------------------------------
-- 9. Aktivitas & alert real-time (feed untuk panel kanan)
-- ---------------------------------------------------------------------
create table if not exists wh3d_activity (
  id           bigserial primary key,
  warehouse_id bigint,
  event_type   text not null,
  title        text not null,
  employee_id  text,
  employee     text,
  location     text,
  sku          text,
  quantity     numeric(14, 2),
  status       text not null default 'INFO',
  event_time   timestamptz not null default now(),
  meta         jsonb not null default '{}'::jsonb
);
create index if not exists wh3d_activity_time_idx on wh3d_activity (event_time desc);

create table if not exists wh3d_alert (
  id           bigserial primary key,
  warehouse_id bigint,
  alert_type   text not null,
  severity     text not null default 'WARNING',
  title        text not null,
  message      text,
  location     text,
  object_uid   text,
  position_x   numeric(12, 3),
  position_y   numeric(12, 3),
  position_z   numeric(12, 3),
  status       text not null default 'OPEN',
  created_at   timestamptz not null default now(),
  resolved_at  timestamptz
);
create index if not exists wh3d_alert_open_idx on wh3d_alert (warehouse_id, status, created_at desc);

-- ---------------------------------------------------------------------
-- 10. Task operasional (assignment nyata, bukan simulasi palsu)
-- ---------------------------------------------------------------------
create table if not exists wh3d_task (
  id           bigserial primary key,
  warehouse_id bigint,
  task_uid     text not null unique,
  task_type    text not null,
  reference_no text,
  employee_id  text,
  equipment_uid text,
  location     text,
  status       text not null default 'PENDING',
  priority     text not null default 'NORMAL',
  path         jsonb not null default '[]'::jsonb,
  started_at   timestamptz,
  finished_at  timestamptz,
  created_at   timestamptz not null default now()
);

-- ---------------------------------------------------------------------
-- 11. Permission Warehouse agar menu Warehouse 3D mengikuti RBAC yang ada
-- ---------------------------------------------------------------------
do $$
declare gid bigint; g_inventory bigint;
begin
  select id into gid from sys_permission_groups
   where name = 'Warehouse - Fitur Gudang' and deleted_at is null limit 1;
  if gid is not null then
    insert into sys_permissions (name, code, group_id, status, description)
    select 'Warehouse 3D', 'warehouse:warehouse3d', gid, 'ON', 'Warehouse: Digital Twin Warehouse 3D'
    where not exists (select 1 from sys_permissions where code = 'warehouse:warehouse3d');

    insert into sys_role_permissions (role_id, permission_id, status, effect)
    select r.id, p.id, 'ON', 'ALLOW'
      from sys_roles r
      join sys_permissions p on p.code = 'warehouse:warehouse3d'
     where r.deleted_at is null
       and not exists (
         select 1 from sys_role_permissions rp
         where rp.role_id = r.id and rp.permission_id = p.id
       );
  end if;

  -- Menu navigasi di Menu Management (induk: Inventaris & Logistik)
  select id into g_inventory from sys_menus
   where parent_id is null and path = '/warehouse/inventory' limit 1;
  if g_inventory is not null then
    insert into sys_menus (name, path, type, status, parent_id, component, meta)
    select 'Warehouse 3D', '/warehouse/inventory/warehouse3d', 'MENU', 'ON', g_inventory,
           'app/warehouse-3d/index.vue',
           '{"title":"Warehouse 3D","icon":"lucide:box","order":3}'::jsonb
    where not exists (
      select 1 from sys_menus where parent_id = g_inventory and path = '/warehouse/inventory/warehouse3d'
    );
  end if;
end $$;
