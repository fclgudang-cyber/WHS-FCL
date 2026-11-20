-- =====================================================================
-- FCL bootstrap schema: semua tabel yang dipakai frontend FCL dibuat
-- otomatis bila belum ada di database produksi.
-- Aman diulang (CREATE TABLE IF NOT EXISTS / ADD COLUMN IF NOT EXISTS).
-- Tidak menyentuh tabel ent core-service (sys_*, inv_*, wms_*).
-- =====================================================================

-- helper: kolom tambahan ditambahkan bila belum ada
create or replace function fcl_add_column(p_table text, p_col text, p_type text default 'text')
returns void language plpgsql as $$
begin
  if not exists (
    select 1 from information_schema.columns
    where table_schema = 'public' and table_name = p_table and column_name = p_col
  ) then
    execute format('alter table %I add column %I %s', p_table, p_col, p_type);
  end if;
end $$;

-- ---------------------------------------------------------------- users (auth FCL)
create table if not exists users (
  id text primary key default gen_random_uuid()::text,
  username text,
  password text,
  nama text,
  role text,
  divisi text,
  permissions text,
  "createdAt" text,
  email text,
  status text,
  "updatedAt" text
);
do $$ begin
  create unique index users_username_key on users (lower(username));
exception when others then null; end $$;

-- ---------------------------------------------------------------- master karyawan
create table if not exists karyawan (
  id text primary key default gen_random_uuid()::text,
  nama text,
  divisi text,
  jabatan text,
  cabang text,
  telepon text,
  email text,
  tanggalMasuk text,
  tanggalSelesai text,
  status text default 'Kontrak',
  sisaCuti numeric(10,2) default 0,
  fingerprintId text,
  fotoUrl text,
  alamat text,
  "createdAt" text,
  "createdBy" text,
  "updatedAt" text
);

-- ---------------------------------------------------------------- stock / inventory FCL
create table if not exists stock (
  id text primary key default gen_random_uuid()::text,
  sku text,
  nama text,
  barcode text,
  satuan text,
  kategori text,
  stok numeric(14,2) default 0,
  stokMin numeric(14,2) default 0,
  lokasi text,
  batch text,
  expDate text,
  gudang text,
  rak text,
  bin text,
  "createdAt" text,
  "createdBy" text,
  "updatedAt" text
);
do $$ begin
  create index if not exists stock_sku_idx on stock (sku);;
exception when others then null; end $$;
do $$ begin
  create index if not exists stock_lokasi_idx on stock (lokasi);;
exception when others then null; end $$;
do $$ begin
  create index if not exists stock_batch_idx on stock (batch);;
exception when others then null; end $$;

-- ---------------------------------------------------------------- order
create table if not exists "order" (
  id text primary key default gen_random_uuid()::text,
  noresi text,
  keterangan text,
  kategori text,
  status text default 'Menunggu',
  totalItem numeric(14,2) default 0,
  total numeric(14,2) default 0,
  diskon numeric(14,2) default 0,
  ongkir numeric(14,2) default 0,
  namaCustomer text,
  alamat text,
  noTelp text,
  kurir text,
  tanggal text,
  "createdAt" text,
  "createdBy" text,
  "updatedAt" text
);
create table if not exists orderdetail (
  id text primary key default gen_random_uuid()::text,
  "orderId" text,
  sku text,
  nama text,
  barcode text,
  satuan text,
  qty numeric(14,2) default 0,
  harga numeric(14,2) default 0,
  subtotal numeric(14,2) default 0,
  keterangan text,
  "createdAt" text
);
do $$ begin
  create index if not exists orderdetail_order_idx on orderdetail ("orderId");;
exception when others then null; end $$;

-- ---------------------------------------------------------------- petty cash
create table if not exists pettycashperiod (
  id text primary key default gen_random_uuid()::text,
  nama text,
  tanggalMulai text,
  tanggalSelesai text,
  saldoAwal numeric(14,2) default 0,
  status text default 'Aktif',
  keterangan text,
  createdBy text,
  "createdAt" text,
  "updatedAt" text
);
create table if not exists pettycash (
  id text primary key default gen_random_uuid()::text,
  periodId text,
  tanggal text,
  kategori text,
  keterangan text,
  nominal numeric(14,2) default 0,
  tipe text default 'Keluar',
  buktiUrl text,
  statusBayar text default 'Belum Bayar',
  createdBy text,
  "createdAt" text,
  "updatedAt" text
);
do $$ begin
  create index if not exists pettycash_period_idx on pettycash (periodId);;
exception when others then null; end $$;

-- ---------------------------------------------------------------- asset warehouse
create table if not exists assetwarehouse (
  id text primary key default gen_random_uuid()::text,
  kode text,
  nama text,
  kategori text,
  kondisi text,
  jumlah numeric(14,2) default 0,
  satuan text,
  lokasi text,
  keterangan text,
  foto text,
  "createdAt" text,
  "createdBy" text,
  "updatedAt" text
);
create table if not exists assetopnamesession (
  id text primary key default gen_random_uuid()::text,
  kode text,
  nama text,
  tanggal text,
  status text default 'DRAFT',
  keterangan text,
  createdBy text,
  "createdAt" text,
  "finishedAt" text
);
create table if not exists assetopnamelog (
  id text primary key default gen_random_uuid()::text,
  "sessionId" text,
  "assetId" text,
  kode text,
  nama text,
  systemQty numeric(14,2) default 0,
  actualQty numeric(14,2) default 0,
  selisih numeric(14,2) default 0,
  keterangan text,
  createdBy text,
  "createdAt" text
);
do $$ begin
  create index if not exists assetopnamelog_session_idx on assetopnamelog ("sessionId");;
exception when others then null; end $$;

-- ---------------------------------------------------------------- absensi
create table if not exists absensikaryawan (
  id text primary key default gen_random_uuid()::text,
  karyawanId text,
  nama text,
  divisi text,
  jabatan text,
  tanggal text,
  jamMasuk text,
  jamPulang text,
  status text,
  keterangan text,
  fingerprintId text,
  "createdAt" text,
  "updatedAt" text
);
do $$ begin
  create index if not exists absensi_karyawan_idx on absensikaryawan (karyawanId);;
exception when others then null; end $$;

-- ---------------------------------------------------------------- SDM
create table if not exists suratperingatan (
  id text primary key default gen_random_uuid()::text,
  karyawanId text,
  karyawanNama text,
  jenisSP text,
  alasan text,
  tanggalSP text,
  masaBerlaku text,
  tanggalKadaluarsa text,
  status text,
  "createdAt" text,
  "createdBy" text
);
create table if not exists laporankerja (
  id text primary key default gen_random_uuid()::text,
  karyawanId text,
  karyawanNama text,
  divisi text,
  jabatan text,
  tanggal text,
  shift text,
  jamMasuk text,
  jamPulang text,
  pekerjaan text,
  target text,
  realisasi text,
  keterangan text,
  status text,
  "createdAt" text,
  "createdBy" text,
  "updatedAt" text
);
create table if not exists instruksilembur (
  id text primary key default gen_random_uuid()::text,
  nama text,
  karyawanId text,
  divisi text,
  tanggal text,
  jamMulai text,
  jamSelesai text,
  durasi numeric(10,2) default 0,
  pekerjaan text,
  keterangan text,
  status text default 'Menunggu',
  "createdAt" text,
  "createdBy" text
);
create table if not exists instruksilemburdetail (
  id text primary key default gen_random_uuid()::text,
  "instruksiId" text,
  karyawanId text,
  karyawanNama text,
  divisi text,
  status text,
  keterangan text,
  "createdAt" text
);
do $$ begin
  create index if not exists instruksilemburdetail_parent_idx on instruksilemburdetail ("instruksiId");;
exception when others then null; end $$;
create table if not exists jadwalroster (
  id text primary key default gen_random_uuid()::text,
  karyawanId text,
  karyawanNama text,
  divisi text,
  jabatan text,
  tanggal text,
  shift text,
  jamMasuk text,
  jamPulang text,
  lokasi text,
  keterangan text,
  status text,
  "createdAt" text,
  "createdBy" text
);

-- ---------------------------------------------------------------- API devices / ijin / lembur
create table if not exists fcl_api_devices (
  id text primary key default gen_random_uuid()::text,
  nama text,
  kode text,
  token text,
  ip text,
  endpoint text,
  status text default 'AKTIF',
  keterangan text,
  "createdAt" text,
  "updatedAt" text
);
create table if not exists ijin (
  id text primary key default gen_random_uuid()::text,
  karyawanId text,
  karyawanNama text,
  jenis text,
  tanggalMulai text,
  tanggalSelesai text,
  alasan text,
  status text default 'Menunggu',
  "createdAt" text,
  "createdBy" text
);
create table if not exists lembur (
  id text primary key default gen_random_uuid()::text,
  karyawanId text,
  karyawanNama text,
  tanggal text,
  jamMulai text,
  jamSelesai text,
  durasi numeric(10,2) default 0,
  alasan text,
  status text default 'Menunggu',
  "createdAt" text,
  "createdBy" text
);

-- ---------------------------------------------------------------- kolom umum yang sering dipakai lintas modul
select fcl_add_column('stock', 'stokAwal', 'numeric(14,2)');
select fcl_add_column('stock', 'harga', 'numeric(14,2)');
select fcl_add_column('stock', 'expired', 'text');
select fcl_add_column('karyawan', 'foto', 'text');
select fcl_add_column('karyawan', 'tanggalLahir', 'text');
select fcl_add_column('karyawan', 'jenisKelamin', 'text');
