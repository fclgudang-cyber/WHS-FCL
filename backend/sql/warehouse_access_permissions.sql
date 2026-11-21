-- Sinkronkan fitur Warehouse ke modul Kelola Hak Akses & Menu Management.
-- Aman diulang. Juga memigrasikan permission/menu lama berprefix fcl: -> warehouse:.

do $$ begin
  perform setval(pg_get_serial_sequence('sys_menus', 'id'), greatest(coalesce((select max(id) from sys_menus), 1), 1));
exception when others then null; end $$;

-- 1. Grup permission
do $$ begin
  update sys_permission_groups
     set name = 'Warehouse - Fitur Gudang',
         description = 'Semua fitur Warehouse',
         module = 'warehouse'
   where name = 'FCL - Fitur Gudang';
exception when others then null; end $$;

do $$
declare
  gid bigint;
begin
  select id into gid
    from sys_permission_groups
   where name = 'Warehouse - Fitur Gudang' and deleted_at is null
   limit 1;

  if gid is null then
    insert into sys_permission_groups (name, description, module, sort_order, status)
    values ('Warehouse - Fitur Gudang', 'Semua fitur Warehouse', 'warehouse', 900, 'ON')
    returning id into gid;
  end if;

  insert into sys_permissions (name, code, group_id, status, description)
  select v.name, v.code, gid, 'ON', 'Warehouse: ' || v.name
  from (values
    ('Dashboard', 'warehouse:dashboard'),
    ('Modul Inventaris & Logistik', 'warehouse:warehouse'),
    ('Modul Keuangan', 'warehouse:keuangan'),
    ('Modul Operasional', 'warehouse:operasional'),
    ('Modul SDM', 'warehouse:sdm'),
    ('Modul Dokumentasi', 'warehouse:dokumentasi'),
    ('Stock Barang', 'warehouse:stock'),
    ('Stock Opname', 'warehouse:stockOpname'),
    ('Inbound', 'warehouse:inbound'),
    ('Retur', 'warehouse:retur'),
    ('Order', 'warehouse:order'),
    ('Booking Mobil', 'warehouse:bookingMobil'),
    ('Asset Warehouse', 'warehouse:assetWarehouse'),
    ('Analisis Stock', 'warehouse:analisis'),
    ('Kas Gudang', 'warehouse:kasGudang'),
    ('Team Building', 'warehouse:teamBuilding'),
    ('Pengeluaran', 'warehouse:expense'),
    ('Petty Cash', 'warehouse:pettyCash'),
    ('Kategori Petty Cash', 'warehouse:pettyCashKategori'),
    ('Klaim Paket', 'warehouse:klaim'),
    ('Laporan Kerja', 'warehouse:laporanKerja'),
    ('Grafik Laporan', 'warehouse:grafikLaporan'),
    ('Instruksi Lembur', 'warehouse:instruksiLembur'),
    ('Ijin & Cuti', 'warehouse:ijin'),
    ('Lembur', 'warehouse:lembur'),
    ('Karyawan', 'warehouse:karyawan'),
    ('Absensi Karyawan', 'warehouse:absensiKaryawan'),
    ('Jadwal Shift', 'warehouse:jadwalShift'),
    ('SOP', 'warehouse:sop'),
    ('Webhook Fingerprint', 'warehouse:fingerprintWebhook')
  ) as v(name, code)
  where not exists (select 1 from sys_permissions p where p.code = v.code);
exception when others then null;
end $$;

-- 2. Pindahkan hak role lama ke kode baru, lalu hapus kode lama.
do $$ begin
  insert into sys_role_permissions (role_id, permission_id, status, effect)
  select rp.role_id, np.id, rp.status, rp.effect
    from sys_role_permissions rp
    join sys_permissions op on op.id = rp.permission_id and op.code like 'fcl:%'
    join sys_permissions np on np.code = 'warehouse:' || substring(op.code from 5)
   where not exists (
     select 1 from sys_role_permissions rp2
      where rp2.role_id = rp.role_id and rp2.permission_id = np.id
   );

  delete from sys_role_permissions
   where permission_id in (select id from sys_permissions where code like 'fcl:%');
  delete from sys_permissions where code like 'fcl:%';
exception when others then null; end $$;

-- 3. Berikan semua permission Warehouse kepada role aktif.
do $$ begin
  insert into sys_role_permissions (role_id, permission_id, status, effect)
  select r.id, p.id, 'ON', 'ALLOW'
    from sys_roles r
    cross join sys_permissions p
   where p.code like 'warehouse:%'
     and r.deleted_at is null
     and not exists (
       select 1 from sys_role_permissions rp
        where rp.role_id = r.id and rp.permission_id = p.id
     );
exception when others then null; end $$;

-- 4. Akun tanpa role tidak bisa login ("insufficient authority").
--    Beri role default yang memiliki permission sys:access_backend.
do $$
declare
  default_role bigint;
begin
  select r.id into default_role
    from sys_roles r
    join sys_role_permissions rp on rp.role_id = r.id and rp.status = 'ON' and rp.effect = 'ALLOW'
    join sys_permissions p on p.id = rp.permission_id and p.code = 'sys:access_backend'
   where r.deleted_at is null and r.status = 'ON'
   order by case
       when r.code = 'platform:admin' then 0
       when r.code like 'tenant:%' then 1
       else 2
     end
   limit 1;

  if default_role is not null then
    insert into sys_user_roles (tenant_id, user_id, role_id, is_primary, status, assigned_at, created_at)
    select u.tenant_id, u.id, default_role, true, 'ACTIVE', now(), now()
      from sys_users u
     where u.deleted_at is null
       and not exists (
         select 1 from sys_user_roles ur
          where ur.user_id = u.id and ur.deleted_at is null
       );
  end if;
exception when others then null;
end $$;

-- 4. Menu: hapus semua menu /fcl/* lama, pasang menu /warehouse/* baru.
do $$
declare
  g_inventory bigint;
  g_finance bigint;
  g_opm bigint;
  g_system bigint;
begin
  delete from sys_menus where parent_id in (
    select id from sys_menus where path like '/fcl/%'
  );
  delete from sys_menus
   where path like '/fcl/%' or path = '/opm/org-units';

  insert into sys_menus (name, path, type, status, parent_id, meta)
  select v.name, v.path, 'CATALOG', 'ON', null::bigint, v.meta::jsonb
  from (values
    ('Inventaris & Logistik', '/warehouse/inventory', '{"title":"Inventaris & Logistik","icon":"lucide:warehouse","order":1000,"authority":["sys:platform_admin","sys:tenant_manager"]}'),
    ('Keuangan', '/warehouse/finance', '{"title":"Keuangan","icon":"lucide:wallet","order":1100,"authority":["sys:platform_admin","sys:tenant_manager"]}'),
    ('Organisasi & SDM', '/opm', '{"title":"Organisasi & SDM","icon":"lucide:users","order":2001,"authority":["sys:platform_admin","sys:tenant_manager"]}'),
    ('Sistem', '/system', '{"title":"Sistem","icon":"lucide:settings","order":2100,"authority":["sys:platform_admin","sys:tenant_manager"]}'),
    ('Dashboard', '/warehouse/dashboard', '{"title":"Dashboard","icon":"lucide:layout-dashboard","order":-3,"authority":["sys:platform_admin","sys:tenant_manager"]}')
  ) as v(name, path, meta)
  where not exists (select 1 from sys_menus m where m.parent_id is null and m.path = v.path);

  select id into g_inventory from sys_menus where parent_id is null and path = '/warehouse/inventory' limit 1;
  select id into g_finance from sys_menus where parent_id is null and path = '/warehouse/finance' limit 1;
  select id into g_opm from sys_menus where parent_id is null and path = '/opm' limit 1;
  select id into g_system from sys_menus where parent_id is null and path = '/system' limit 1;

  insert into sys_menus (name, path, type, status, parent_id, meta)
  select v.name, v.path, 'MENU', 'ON', v.parent, v.meta::jsonb
  from (values
    -- Inventaris & Logistik
    ('Stock Barang', '/warehouse/inventory/stock', g_inventory, '{"title":"Stock Barang","icon":"lucide:package","order":1}'),
    ('Stock Opname', '/warehouse/inventory/stockOpname', g_inventory, '{"title":"Stock Opname","icon":"lucide:clipboard-check","order":2}'),
    ('Warehouse 3D', '/warehouse/inventory/warehouse3d', g_inventory, '{"title":"Warehouse 3D","icon":"lucide:box","order":3}'),
    ('Inbound', '/warehouse/inventory/inbound', g_inventory, '{"title":"Inbound","icon":"lucide:package-plus","order":4}'),
    ('Retur', '/warehouse/inventory/retur', g_inventory, '{"title":"Retur","icon":"lucide:undo-2","order":6}'),
    ('Order', '/warehouse/inventory/order', g_inventory, '{"title":"Order","icon":"lucide:shopping-cart","order":7}'),
    ('Booking Mobil', '/warehouse/inventory/bookingMobil', g_inventory, '{"title":"Booking Mobil","icon":"lucide:car","order":9}'),
    ('Asset Warehouse', '/warehouse/inventory/assetWarehouse', g_inventory, '{"title":"Asset Warehouse","icon":"lucide:building-2","order":10}'),
    ('Analisis Stock', '/warehouse/inventory/analisis', g_inventory, '{"title":"Analisis Stock","icon":"lucide:chart-bar","order":11}'),

    -- Keuangan
    ('Kas Gudang', '/warehouse/finance/kasGudang', g_finance, '{"title":"Kas Gudang","icon":"lucide:vault","order":1}'),
    ('Team Building', '/warehouse/finance/teamBuilding', g_finance, '{"title":"Team Building","icon":"lucide:users","order":2}'),
    ('Pengeluaran', '/warehouse/finance/expense', g_finance, '{"title":"Pengeluaran","icon":"lucide:receipt","order":3}'),
    ('Petty Cash', '/warehouse/finance/pettyCash', g_finance, '{"title":"Petty Cash","icon":"lucide:coins","order":4}'),
    ('Kategori Petty Cash', '/warehouse/finance/pettyCashKategori', g_finance, '{"title":"Kategori Petty Cash","icon":"lucide:list","order":5}'),
    ('Klaim Paket', '/warehouse/finance/klaim', g_finance, '{"title":"Klaim Paket","icon":"lucide:badge-alert","order":6}'),

    -- Organisasi & SDM (modul utama)
    ('Karyawan', '/opm/karyawan', g_opm, '{"title":"Karyawan","icon":"lucide:id-card","order":5}'),
    ('Absensi Karyawan', '/opm/absensiKaryawan', g_opm, '{"title":"Absensi Karyawan","icon":"lucide:fingerprint","order":6}'),
    ('Jadwal Shift', '/opm/jadwalShift', g_opm, '{"title":"Jadwal Shift","icon":"lucide:calendar-clock","order":7}'),
    ('Laporan Kerja', '/opm/laporanKerja', g_opm, '{"title":"Laporan Kerja","icon":"lucide:clipboard-pen","order":8}'),
    ('Grafik Laporan', '/opm/grafikLaporan', g_opm, '{"title":"Grafik Laporan","icon":"lucide:chart-line","order":9}'),
    ('Instruksi Lembur', '/opm/instruksiLembur', g_opm, '{"title":"Instruksi Lembur","icon":"lucide:alarm-clock","order":10}'),
    ('Ijin & Cuti', '/opm/ijin', g_opm, '{"title":"Ijin & Cuti","icon":"lucide:calendar-x","order":11}'),
    ('Lembur', '/opm/lembur', g_opm, '{"title":"Lembur","icon":"lucide:clock","order":12}'),

    -- Sistem (modul utama)
    ('SOP', '/system/sop', g_system, '{"title":"SOP","icon":"lucide:book-open","order":90}')
  ) as v(name, path, parent, meta)
  where v.parent is not null
    and not exists (select 1 from sys_menus m where m.parent_id = v.parent and m.path = v.path);
exception when others then null;
end $$;
