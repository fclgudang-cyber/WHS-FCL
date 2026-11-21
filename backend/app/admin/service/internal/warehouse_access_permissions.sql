-- Sinkronkan semua fitur FCL ke modul Kelola Hak Akses.
-- Dijalankan sekali; aman diulang (idempoten).

-- Pastikan sequence menu tidak tertinggal dari data yang sudah ada.
do $$ begin
  perform setval(pg_get_serial_sequence('sys_menus', 'id'), greatest(coalesce((select max(id) from sys_menus), 1), 1));
exception when others then null; end $$;

do $$
declare
  gid bigint;
begin
  select id into gid
  from sys_permission_groups
  where name = 'FCL - Fitur Gudang' and deleted_at is null
  limit 1;

  if gid is null then
    insert into sys_permission_groups (name, description, module, sort_order, status)
    values ('FCL - Fitur Gudang', 'Semua fitur FCL Warehouse', 'fcl', 900, 'ON')
    returning id into gid;
  end if;

  insert into sys_permissions (name, code, group_id, status, description)
  select v.name, v.code, gid, 'ON', 'FCL: ' || v.name
  from (values
    ('Dashboard', 'fcl:dashboard'),
    ('Modul Inventaris & Logistik', 'fcl:warehouse'),
    ('Modul Keuangan', 'fcl:keuangan'),
    ('Modul Operasional', 'fcl:operasional'),
    ('Modul SDM', 'fcl:sdm'),
    ('Modul Dokumentasi', 'fcl:dokumentasi'),
    ('Stock Barang', 'fcl:stock'),
    ('Stock Opname', 'fcl:stockOpname'),
    ('Inbound', 'fcl:inbound'),
    ('Retur', 'fcl:retur'),
    ('Order', 'fcl:order'),
    ('Booking Mobil', 'fcl:bookingMobil'),
    ('Asset Warehouse', 'fcl:assetWarehouse'),
    ('Analisis Stock', 'fcl:analisis'),
    ('Kas Gudang', 'fcl:kasGudang'),
    ('Team Building', 'fcl:teamBuilding'),
    ('Pengeluaran', 'fcl:expense'),
    ('Petty Cash', 'fcl:pettyCash'),
    ('Kategori Petty Cash', 'fcl:pettyCashKategori'),
    ('Klaim Paket', 'fcl:klaim'),
    ('Laporan Kerja', 'fcl:laporanKerja'),
    ('Grafik Laporan', 'fcl:grafikLaporan'),
    ('Instruksi Lembur', 'fcl:instruksiLembur'),
    ('Ijin & Cuti', 'fcl:ijin'),
    ('Lembur', 'fcl:lembur'),
    ('Karyawan', 'fcl:karyawan'),
    ('Absensi Karyawan', 'fcl:absensiKaryawan'),
    ('Jadwal Shift', 'fcl:jadwalShift'),
    ('SOP', 'fcl:sop'),
    ('Webhook Fingerprint', 'fcl:fingerprintWebhook')
  ) as v(name, code)
  where not exists (select 1 from sys_permissions p where p.code = v.code);
exception when others then null;
end $$;

-- Berikan semua permission FCL kepada setiap role yang aktif.
do $$ begin
  insert into sys_role_permissions (role_id, permission_id, status, effect)
  select r.id, p.id, 'ON', 'ALLOW'
  from sys_roles r
  cross join sys_permissions p
  where p.code like 'fcl:%'
    and r.deleted_at is null
    and not exists (
      select 1 from sys_role_permissions rp
      where rp.role_id = r.id and rp.permission_id = p.id
    );
exception when others then null; end $$;

-- Menu FCL terintegrasi ke modul utama.
-- Tidak ada grup "SDM / Operasional / Dokumentasi" FCL yang berdiri sendiri.
do $$
declare
  g_warehouse bigint;
  g_keuangan bigint;
  g_opm bigint;
  g_system bigint;
begin
  -- Bersihkan katalog FCL lama beserta anaknya (database yang sudah terpasang).
  delete from sys_menus where parent_id in (
    select id from sys_menus
     where path in ('/fcl/operasional', '/fcl/sdm', '/fcl/dokumentasi')
  );
  delete from sys_menus
   where path in ('/fcl/operasional', '/fcl/sdm', '/fcl/dokumentasi', '/opm/org-units');

  -- Katalog modul utama / domain.
  insert into sys_menus (name, path, type, status, parent_id, meta)
  select v.name, v.path, 'CATALOG', 'ON', null::bigint, v.meta::jsonb
  from (values
    ('Inventaris & Logistik', '/fcl/warehouse', '{"title":"Inventaris & Logistik","icon":"lucide:warehouse","order":1000,"authority":["sys:platform_admin","sys:tenant_manager"]}'),
    ('Keuangan', '/fcl/keuangan', '{"title":"Keuangan","icon":"lucide:wallet","order":1100,"authority":["sys:platform_admin","sys:tenant_manager"]}'),
    ('Organisasi & SDM', '/opm', '{"title":"Organisasi & SDM","icon":"lucide:users","order":2001,"authority":["sys:platform_admin","sys:tenant_manager"]}'),
    ('Sistem', '/system', '{"title":"Sistem","icon":"lucide:settings","order":2100,"authority":["sys:platform_admin","sys:tenant_manager"]}'),
    ('Dashboard', '/fcl/dashboard', '{"title":"Dashboard","icon":"lucide:layout-dashboard","order":-3,"authority":["sys:platform_admin","sys:tenant_manager"]}')
  ) as v(name, path, meta)
  where not exists (select 1 from sys_menus m where m.parent_id is null and m.path = v.path);

  select id into g_warehouse from sys_menus where parent_id is null and path = '/fcl/warehouse' limit 1;
  select id into g_keuangan from sys_menus where parent_id is null and path = '/fcl/keuangan' limit 1;
  select id into g_opm from sys_menus where parent_id is null and path = '/opm' limit 1;
  select id into g_system from sys_menus where parent_id is null and path = '/system' limit 1;

  insert into sys_menus (name, path, type, status, parent_id, meta)
  select v.name, v.path, 'MENU', 'ON', v.parent, v.meta::jsonb
  from (values
    -- Inventaris & Logistik
    ('Stock Barang', '/fcl/warehouse/stock', g_warehouse, '{"title":"Stock Barang","icon":"lucide:package","order":1}'),
    ('Stock Opname', '/fcl/warehouse/stockOpname', g_warehouse, '{"title":"Stock Opname","icon":"lucide:clipboard-check","order":2}'),
    ('Warehouse 3D', '/fcl/warehouse/warehouse3d', g_warehouse, '{"title":"Warehouse 3D","icon":"lucide:box","order":3}'),
    ('Inbound', '/fcl/warehouse/inbound', g_warehouse, '{"title":"Inbound","icon":"lucide:package-plus","order":4}'),
    ('Retur', '/fcl/warehouse/retur', g_warehouse, '{"title":"Retur","icon":"lucide:undo-2","order":6}'),
    ('Order', '/fcl/warehouse/order', g_warehouse, '{"title":"Order","icon":"lucide:shopping-cart","order":7}'),
    ('Booking Mobil', '/fcl/warehouse/bookingMobil', g_warehouse, '{"title":"Booking Mobil","icon":"lucide:car","order":9}'),
    ('Asset Warehouse', '/fcl/warehouse/assetWarehouse', g_warehouse, '{"title":"Asset Warehouse","icon":"lucide:building-2","order":10}'),
    ('Analisis Stock', '/fcl/warehouse/analisis', g_warehouse, '{"title":"Analisis Stock","icon":"lucide:chart-bar","order":11}'),

    -- Keuangan
    ('Kas Gudang', '/fcl/keuangan/kasGudang', g_keuangan, '{"title":"Kas Gudang","icon":"lucide:vault","order":1}'),
    ('Team Building', '/fcl/keuangan/teamBuilding', g_keuangan, '{"title":"Team Building","icon":"lucide:users","order":2}'),
    ('Pengeluaran', '/fcl/keuangan/expense', g_keuangan, '{"title":"Pengeluaran","icon":"lucide:receipt","order":3}'),
    ('Petty Cash', '/fcl/keuangan/pettyCash', g_keuangan, '{"title":"Petty Cash","icon":"lucide:coins","order":4}'),
    ('Kategori Petty Cash', '/fcl/keuangan/pettyCashKategori', g_keuangan, '{"title":"Kategori Petty Cash","icon":"lucide:list","order":5}'),
    ('Klaim Paket', '/fcl/keuangan/klaim', g_keuangan, '{"title":"Klaim Paket","icon":"lucide:badge-alert","order":6}'),

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
