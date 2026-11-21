-- Sinkronkan fitur Warehouse ke modul Kelola Hak Akses & Menu Management.
-- Aman diulang. Juga memigrasikan permission/menu lama berprefix fcl: -> warehouse:.

do $$ begin
  perform setval(pg_get_serial_sequence('sys_menus', 'id'), greatest(coalesce((select max(id) from sys_menus), 1), 1));
exception when others then null; end $$;

-- 0. Pastikan permission akses backend ada dan melekat pada semua role aktif.
-- Tanpa ini login ditolak dengan "insufficient authority" meski user punya role.
do $$
declare
  gid bigint;
  pid bigint;
begin
  select id into gid
    from sys_permission_groups
   where deleted_at is null
   order by id
   limit 1;

  if gid is null then
    insert into sys_permission_groups (name, description, module, sort_order, status)
    values ('系统管理', '系统权限', 'system', 1, 'ON')
    returning id into gid;
  end if;

  insert into sys_permissions (name, code, group_id, status, description)
  select '访问后台', 'sys:access_backend', gid, 'ON', '允许用户访问系统后台管理界面'
   where not exists (select 1 from sys_permissions where code = 'sys:access_backend');

  select id into pid from sys_permissions where code = 'sys:access_backend' limit 1;

  if pid is not null then
    insert into sys_role_permissions (role_id, permission_id, status, effect)
    select r.id, pid, 'ON', 'ALLOW'
      from sys_roles r
     where r.deleted_at is null and r.status = 'ON'
       and not exists (
         select 1 from sys_role_permissions rp
          where rp.role_id = r.id and rp.permission_id = pid
       );
  end if;
exception when others then null;
end $$;

-- 0b. Permission role sistem yang dipakai route authority ([sys:platform_admin], [sys:tenant_manager]).
do $$
declare
  gid bigint;
begin
  select id into gid from sys_permission_groups where deleted_at is null order by id limit 1;
  if gid is null then
    insert into sys_permission_groups (name, description, module, sort_order, status)
    values ('系统管理', '系统权限', 'system', 1, 'ON') returning id into gid;
  end if;

  insert into sys_permissions (name, code, group_id, status, description)
  select v.name, v.code, gid, 'ON', v.name
  from (values
    ('平台管理员', 'sys:platform_admin'),
    ('租户管理员', 'sys:tenant_manager')
  ) as v(name, code)
  where not exists (select 1 from sys_permissions p where p.code = v.code);

  insert into sys_role_permissions (role_id, permission_id, status, effect)
  select r.id, p.id, 'ON', 'ALLOW'
    from sys_roles r
    cross join sys_permissions p
   where p.code in ('sys:platform_admin', 'sys:tenant_manager')
     and r.deleted_at is null and r.status = 'ON'
     and not exists (
       select 1 from sys_role_permissions rp
        where rp.role_id = r.id and rp.permission_id = p.id
     );
exception when others then null;
end $$;

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

-- 5. Menu: hapus semua menu /fcl/* lama, pasang menu /warehouse/* baru.
--    component wajib diisi agar Vben bisa memetakan path ke file view.
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

  -- Perbarui component untuk menu yang sudah ada (dari seed tanpa component).
  update sys_menus m set component = v.component
  from (values
    ('/warehouse/inventory', 'BasicLayout'),
    ('/warehouse/finance', 'BasicLayout'),
    ('/opm', 'BasicLayout'),
    ('/system', 'BasicLayout'),
    ('/warehouse/dashboard', 'app/dashboard/index.vue'),
    ('/warehouse/inventory/stock', 'app/stock/index.vue'),
    ('/warehouse/inventory/stockOpname', 'app/stock-opname/index.vue'),
    ('/warehouse/inventory/warehouse3d', 'app/warehouse-3d/index.vue'),
    ('/warehouse/inventory/inbound', 'app/inbound/index.vue'),
    ('/warehouse/inventory/retur', 'app/retur/index.vue'),
    ('/warehouse/inventory/order', 'app/order/index.vue'),
    ('/warehouse/inventory/bookingMobil', 'app/booking-mobil/index.vue'),
    ('/warehouse/inventory/assetWarehouse', 'app/asset-warehouse/index.vue'),
    ('/warehouse/inventory/analisis', 'app/analisis/index.vue'),
    ('/warehouse/finance/kasGudang', 'app/kas-gudang/index.vue'),
    ('/warehouse/finance/teamBuilding', 'app/team-building/index.vue'),
    ('/warehouse/finance/expense', 'app/expense/index.vue'),
    ('/warehouse/finance/pettyCash', 'app/petty-cash/index.vue'),
    ('/warehouse/finance/pettyCashKategori', 'app/petty-cash-kategori/index.vue'),
    ('/warehouse/finance/klaim', 'app/klaim/index.vue'),
    ('/opm/karyawan', 'app/karyawan/index.vue'),
    ('/opm/absensiKaryawan', 'app/absensi-karyawan/index.vue'),
    ('/opm/jadwalShift', 'app/jadwal-shift/index.vue'),
    ('/opm/laporanKerja', 'app/laporan-kerja/index.vue'),
    ('/opm/grafikLaporan', 'app/grafik-laporan/index.vue'),
    ('/opm/instruksiLembur', 'app/instruksi-lembur/index.vue'),
    ('/opm/ijin', 'app/ijin/index.vue'),
    ('/opm/lembur', 'app/lembur/index.vue'),
    ('/system/sop', 'app/sop/index.vue')
  ) as v(path, component)
  where m.path = v.path;

  insert into sys_menus (name, path, type, status, parent_id, component, meta)
  select v.name, v.path, 'CATALOG', 'ON', null::bigint, v.component, v.meta::jsonb
  from (values
    ('Inventaris & Logistik', '/warehouse/inventory', 'BasicLayout', '{"title":"Inventaris & Logistik","icon":"lucide:warehouse","order":1000,"authority":["sys:platform_admin","sys:tenant_manager"]}'),
    ('Keuangan', '/warehouse/finance', 'BasicLayout', '{"title":"Keuangan","icon":"lucide:wallet","order":1100,"authority":["sys:platform_admin","sys:tenant_manager"]}'),
    ('Organisasi & SDM', '/opm', 'BasicLayout', '{"title":"Organisasi & SDM","icon":"lucide:users","order":2001,"authority":["sys:platform_admin","sys:tenant_manager"]}'),
    ('Sistem', '/system', 'BasicLayout', '{"title":"Sistem","icon":"lucide:settings","order":2100,"authority":["sys:platform_admin","sys:tenant_manager"]}'),
    ('Dashboard', '/warehouse/dashboard', 'app/dashboard/index.vue', '{"title":"Dashboard","icon":"lucide:layout-dashboard","order":-3,"authority":["sys:platform_admin","sys:tenant_manager"]}')
  ) as v(name, path, component, meta)
  where not exists (select 1 from sys_menus m where m.parent_id is null and m.path = v.path);

  select id into g_inventory from sys_menus where parent_id is null and path = '/warehouse/inventory' limit 1;
  select id into g_finance from sys_menus where parent_id is null and path = '/warehouse/finance' limit 1;
  select id into g_opm from sys_menus where parent_id is null and path = '/opm' limit 1;
  select id into g_system from sys_menus where parent_id is null and path = '/system' limit 1;

  insert into sys_menus (name, path, type, status, parent_id, component, meta)
  select v.name, v.path, 'MENU', 'ON', v.parent, v.component, v.meta::jsonb
  from (values
    ('Stock Barang', '/warehouse/inventory/stock', g_inventory, 'app/stock/index.vue', '{"title":"Stock Barang","icon":"lucide:package","order":1}'),
    ('Stock Opname', '/warehouse/inventory/stockOpname', g_inventory, 'app/stock-opname/index.vue', '{"title":"Stock Opname","icon":"lucide:clipboard-check","order":2}'),
    ('Warehouse 3D', '/warehouse/inventory/warehouse3d', g_inventory, 'app/warehouse-3d/index.vue', '{"title":"Warehouse 3D","icon":"lucide:box","order":3}'),
    ('Inbound', '/warehouse/inventory/inbound', g_inventory, 'app/inbound/index.vue', '{"title":"Inbound","icon":"lucide:package-plus","order":4}'),
    ('Retur', '/warehouse/inventory/retur', g_inventory, 'app/retur/index.vue', '{"title":"Retur","icon":"lucide:undo-2","order":6}'),
    ('Order', '/warehouse/inventory/order', g_inventory, 'app/order/index.vue', '{"title":"Order","icon":"lucide:shopping-cart","order":7}'),
    ('Booking Mobil', '/warehouse/inventory/bookingMobil', g_inventory, 'app/booking-mobil/index.vue', '{"title":"Booking Mobil","icon":"lucide:car","order":9}'),
    ('Asset Warehouse', '/warehouse/inventory/assetWarehouse', g_inventory, 'app/asset-warehouse/index.vue', '{"title":"Asset Warehouse","icon":"lucide:building-2","order":10}'),
    ('Analisis Stock', '/warehouse/inventory/analisis', g_inventory, 'app/analisis/index.vue', '{"title":"Analisis Stock","icon":"lucide:chart-bar","order":11}'),

    ('Kas Gudang', '/warehouse/finance/kasGudang', g_finance, 'app/kas-gudang/index.vue', '{"title":"Kas Gudang","icon":"lucide:vault","order":1}'),
    ('Team Building', '/warehouse/finance/teamBuilding', g_finance, 'app/team-building/index.vue', '{"title":"Team Building","icon":"lucide:users","order":2}'),
    ('Pengeluaran', '/warehouse/finance/expense', g_finance, 'app/expense/index.vue', '{"title":"Pengeluaran","icon":"lucide:receipt","order":3}'),
    ('Petty Cash', '/warehouse/finance/pettyCash', g_finance, 'app/petty-cash/index.vue', '{"title":"Petty Cash","icon":"lucide:coins","order":4}'),
    ('Kategori Petty Cash', '/warehouse/finance/pettyCashKategori', g_finance, 'app/petty-cash-kategori/index.vue', '{"title":"Kategori Petty Cash","icon":"lucide:list","order":5}'),
    ('Klaim Paket', '/warehouse/finance/klaim', g_finance, 'app/klaim/index.vue', '{"title":"Klaim Paket","icon":"lucide:badge-alert","order":6}'),

    ('Karyawan', '/opm/karyawan', g_opm, 'app/karyawan/index.vue', '{"title":"Karyawan","icon":"lucide:id-card","order":5}'),
    ('Absensi Karyawan', '/opm/absensiKaryawan', g_opm, 'app/absensi-karyawan/index.vue', '{"title":"Absensi Karyawan","icon":"lucide:fingerprint","order":6}'),
    ('Jadwal Shift', '/opm/jadwalShift', g_opm, 'app/jadwal-shift/index.vue', '{"title":"Jadwal Shift","icon":"lucide:calendar-clock","order":7}'),
    ('Laporan Kerja', '/opm/laporanKerja', g_opm, 'app/laporan-kerja/index.vue', '{"title":"Laporan Kerja","icon":"lucide:clipboard-pen","order":8}'),
    ('Grafik Laporan', '/opm/grafikLaporan', g_opm, 'app/grafik-laporan/index.vue', '{"title":"Grafik Laporan","icon":"lucide:chart-line","order":9}'),
    ('Instruksi Lembur', '/opm/instruksiLembur', g_opm, 'app/instruksi-lembur/index.vue', '{"title":"Instruksi Lembur","icon":"lucide:alarm-clock","order":10}'),
    ('Ijin & Cuti', '/opm/ijin', g_opm, 'app/ijin/index.vue', '{"title":"Ijin & Cuti","icon":"lucide:calendar-x","order":11}'),
    ('Lembur', '/opm/lembur', g_opm, 'app/lembur/index.vue', '{"title":"Lembur","icon":"lucide:clock","order":12}'),

    ('SOP', '/system/sop', g_system, 'app/sop/index.vue', '{"title":"SOP","icon":"lucide:book-open","order":90}')
  ) as v(name, path, parent, component, meta)
  where v.parent is not null
    and not exists (select 1 from sys_menus m where m.parent_id = v.parent and m.path = v.path);
exception when others then null;
end $$;
