-- Sinkronkan semua fitur FCL ke modul Kelola Hak Akses.
-- Dijalankan sekali; aman diulang (idempoten).

-- Pastikan sequence menu tidak tertinggal dari data yang sudah ada.
select setval(pg_get_serial_sequence('sys_menus', 'id'), greatest(coalesce((select max(id) from sys_menus), 1), 1));

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
end $$;

-- Berikan semua permission FCL kepada setiap role yang aktif.
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

-- Daftar menu FCL di Menu Management.
do $$
declare
  g_warehouse bigint;
  g_keuangan bigint;
  g_operasional bigint;
  g_sdm bigint;
  g_dokumentasi bigint;
begin
  -- Modul FCL berada di level teratas (tanpa katalog induk "/fcl").
  insert into sys_menus (name, path, type, status, parent_id, meta)
  select v.name, v.path, 'CATALOG', 'ON', null::bigint, v.meta::jsonb
  from (values
    ('Inventaris & Logistik', '/fcl/warehouse', '{"title":"Inventaris & Logistik","icon":"lucide:warehouse","order":1000,"authority":["fcl:warehouse","sys:platform_admin"]}'),
    ('Keuangan', '/fcl/keuangan', '{"title":"Keuangan","icon":"lucide:wallet","order":1100,"authority":["fcl:keuangan","sys:platform_admin"]}'),
    ('Operasional', '/fcl/operasional', '{"title":"Operasional","icon":"lucide:settings-2","order":1200,"authority":["fcl:operasional","sys:platform_admin"]}'),
    ('SDM', '/fcl/sdm', '{"title":"SDM","icon":"lucide:users-round","order":1300,"authority":["fcl:sdm","sys:platform_admin"]}'),
    ('Dokumentasi', '/fcl/dokumentasi', '{"title":"Dokumentasi","icon":"lucide:file-text","order":1400,"authority":["fcl:dokumentasi","sys:platform_admin"]}'),
    ('Dashboard', '/fcl/dashboard', '{"title":"Dashboard","icon":"lucide:layout-dashboard","order":-3,"authority":["fcl:dashboard","sys:platform_admin"]}')
  ) as v(name, path, meta)
  where not exists (select 1 from sys_menus m where m.parent_id is null and m.path = v.path);

  select id into g_warehouse from sys_menus where parent_id is null and path = '/fcl/warehouse' limit 1;
  select id into g_keuangan from sys_menus where parent_id is null and path = '/fcl/keuangan' limit 1;
  select id into g_operasional from sys_menus where parent_id is null and path = '/fcl/operasional' limit 1;
  select id into g_sdm from sys_menus where parent_id is null and path = '/fcl/sdm' limit 1;
  select id into g_dokumentasi from sys_menus where parent_id is null and path = '/fcl/dokumentasi' limit 1;

  insert into sys_menus (name, path, type, status, parent_id, meta)
  select v.name, v.path, 'MENU', 'ON', v.parent, v.meta::jsonb
  from (values
    ('Stock Barang', '/fcl/warehouse/stock', g_warehouse, '{"title":"Stock Barang","icon":"lucide:package","order":1}'),
    ('Stock Opname', '/fcl/warehouse/stockOpname', g_warehouse, '{"title":"Stock Opname","icon":"lucide:clipboard-check","order":2}'),
    ('Inbound', '/fcl/warehouse/inbound', g_warehouse, '{"title":"Inbound","icon":"lucide:package-plus","order":4}'),
    ('Retur', '/fcl/warehouse/retur', g_warehouse, '{"title":"Retur","icon":"lucide:undo-2","order":6}'),
    ('Order', '/fcl/warehouse/order', g_warehouse, '{"title":"Order","icon":"lucide:shopping-cart","order":7}'),
    ('Booking Mobil', '/fcl/warehouse/bookingMobil', g_warehouse, '{"title":"Booking Mobil","icon":"lucide:car","order":9}'),
    ('Asset Warehouse', '/fcl/warehouse/assetWarehouse', g_warehouse, '{"title":"Asset Warehouse","icon":"lucide:building-2","order":10}'),
    ('Analisis Stock', '/fcl/warehouse/analisis', g_warehouse, '{"title":"Analisis Stock","icon":"lucide:chart-bar","order":11}'),

    ('Kas Gudang', '/fcl/keuangan/kasGudang', g_keuangan, '{"title":"Kas Gudang","icon":"lucide:vault","order":1}'),
    ('Team Building', '/fcl/keuangan/teamBuilding', g_keuangan, '{"title":"Team Building","icon":"lucide:users","order":2}'),
    ('Pengeluaran', '/fcl/keuangan/expense', g_keuangan, '{"title":"Pengeluaran","icon":"lucide:receipt","order":3}'),
    ('Petty Cash', '/fcl/keuangan/pettyCash', g_keuangan, '{"title":"Petty Cash","icon":"lucide:coins","order":4}'),
    ('Kategori Petty Cash', '/fcl/keuangan/pettyCashKategori', g_keuangan, '{"title":"Kategori Petty Cash","icon":"lucide:list","order":5}'),
    ('Klaim Paket', '/fcl/keuangan/klaim', g_keuangan, '{"title":"Klaim Paket","icon":"lucide:badge-alert","order":6}'),

    ('Laporan Kerja', '/fcl/operasional/laporanKerja', g_operasional, '{"title":"Laporan Kerja","icon":"lucide:clipboard-pen","order":1}'),
    ('Grafik Laporan', '/fcl/operasional/grafikLaporan', g_operasional, '{"title":"Grafik Laporan","icon":"lucide:chart-line","order":2}'),
    ('Instruksi Lembur', '/fcl/operasional/instruksiLembur', g_operasional, '{"title":"Instruksi Lembur","icon":"lucide:alarm-clock","order":3}'),
    ('Ijin & Cuti', '/fcl/operasional/ijin', g_operasional, '{"title":"Ijin & Cuti","icon":"lucide:calendar-x","order":4}'),
    ('Lembur', '/fcl/operasional/lembur', g_operasional, '{"title":"Lembur","icon":"lucide:clock","order":5}'),

    ('Karyawan', '/fcl/sdm/karyawan', g_sdm, '{"title":"Karyawan","icon":"lucide:id-card","order":1}'),
    ('Absensi Karyawan', '/fcl/sdm/absensiKaryawan', g_sdm, '{"title":"Absensi Karyawan","icon":"lucide:fingerprint","order":3}'),
    ('Jadwal Shift', '/fcl/sdm/jadwalShift', g_sdm, '{"title":"Jadwal Shift","icon":"lucide:calendar-clock","order":4}'),

    ('SOP', '/fcl/dokumentasi/sop', g_dokumentasi, '{"title":"SOP","icon":"lucide:book-open","order":1}')
  ) as v(name, path, parent, meta)
  where v.parent is not null
    and not exists (select 1 from sys_menus m where m.parent_id = v.parent and m.path = v.path);
end $$;
