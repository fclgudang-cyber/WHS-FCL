-- =====================================================================
-- FCL bootstrap schema (auto-generated dari audit seluruh view FCL).
-- Dijalankan otomatis oleh fcl-service saat start. Aman diulang.
-- Menyesuaikan database yang sudah punya tabel tapi kurang kolom.
-- =====================================================================

create or replace function fcl_add_column(p_table text, p_col text, p_type text default 'text')
returns void language plpgsql as $$
begin
  execute format('alter table %I add column if not exists %I %s', p_table, p_col, p_type);
end $$;

-- users
create table if not exists "users" (
  id text primary key default gen_random_uuid()::text,
  "username" text,
  "password" text,
  "nama" text,
  "role" text,
  "divisi" text,
  "permissions" text,
  "createdAt" text,
  "email" text,
  "status" text,
  "updatedAt" text
);

-- karyawan
create table if not exists "karyawan" (
  id text primary key default gen_random_uuid()::text,
  "nama" text,
  "divisi" text,
  "jabatan" text,
  "cabang" text,
  "telepon" text,
  "email" text,
  "tanggalMasuk" text,
  "tanggalSelesai" text,
  "status" text,
  "sisaCuti" numeric(10,2),
  "fingerprintId" text,
  "fingerprint_id" text,
  "fotoUrl" text,
  "foto" text,
  "alamat" text,
  "tanggalLahir" text,
  "jenisKelamin" text,
  "createdAt" text,
  "createdBy" text,
  "updatedAt" text
);

-- stock
create table if not exists "stock" (
  id text primary key default gen_random_uuid()::text,
  "sku" text,
  "nama" text,
  "barcode" text,
  "kategori" text,
  "satuan" text,
  "batch" text,
  "lokasi" text,
  "expDate" text,
  "stok" text,
  "stokMin" text,
  "stokAwal" numeric(14,2),
  "harga" numeric(14,2),
  "gudang" text,
  "rak" text,
  "bin" text,
  "createdAt" text,
  "createdBy" text,
  "updatedAt" text
);

-- stockopname
create table if not exists "stockopname" (
  id text primary key default gen_random_uuid()::text,
  "tanggal" text,
  "sku" text,
  "nama" text,
  "lokasi" text,
  "batch" text,
  "expDate" text,
  "stokSistem" numeric(14,2),
  "stokFisik" numeric(14,2),
  "selisih" numeric(14,2),
  "status" text,
  "catatan" text,
  "createdAt" text
);

-- suratjalanmasuk
create table if not exists "suratjalanmasuk" (
  id text primary key default gen_random_uuid()::text,
  "tanggal" text,
  "noSJ" text,
  "supplier" text,
  "status" text,
  "keterangan" text,
  "createdAt" text
);

-- suratjalanmasukdetail
create table if not exists "suratjalanmasukdetail" (
  id text primary key default gen_random_uuid()::text,
  "sjId" text,
  "noSJ" text,
  "sku" text,
  "nama" text,
  "satuan" text,
  "batch" text,
  "qty" numeric(14,2),
  "expDate" text,
  "createdAt" text
);

-- order
create table if not exists "order" (
  id text primary key default gen_random_uuid()::text,
  "noOrder" text,
  "noResi" text,
  "noresi" text,
  "tanggal" text,
  "pelanggan" text,
  "alamat" text,
  "kategori" text,
  "status" text,
  "totalItem" numeric(14,2),
  "total" numeric(14,2),
  "keterangan" text,
  "sentAt" text,
  "buktiPacking" text,
  "createdBy" text,
  "createdAt" text,
  "updatedAt" text
);

-- orderdetail
create table if not exists "orderdetail" (
  id text primary key default gen_random_uuid()::text,
  "orderId" text,
  "noOrder" text,
  "sku" text,
  "nama" text,
  "satuan" text,
  "batch" text,
  "lokasi" text,
  "qty" numeric(14,2),
  "expDate" text,
  "packedQty" text,
  "createdAt" text
);

-- klaim
create table if not exists "klaim" (
  id text primary key default gen_random_uuid()::text,
  "tanggal" text,
  "pic" text,
  "resi" text,
  "harga" numeric(14,2),
  "status" text,
  "keterangan" text,
  "createdAt" text
);

-- klaimdetail
create table if not exists "klaimdetail" (
  id text primary key default gen_random_uuid()::text,
  "klaimId" text,
  "sku" text,
  "harga" numeric(14,2),
  "createdAt" text
);

-- retur
create table if not exists "retur" (
  id text primary key default gen_random_uuid()::text,
  "tanggal" text,
  "noRetur" text,
  "sumber" text,
  "alasan" text,
  "keterangan" text,
  "createdAt" text
);

-- returdetail
create table if not exists "returdetail" (
  id text primary key default gen_random_uuid()::text,
  "returId" text,
  "noRetur" text,
  "sku" text,
  "nama" text,
  "satuan" text,
  "batch" text,
  "qty" numeric(14,2),
  "expDate" text,
  "createdAt" text
);

-- bookingmobil
create table if not exists "bookingmobil" (
  id text primary key default gen_random_uuid()::text,
  "tanggal" text,
  "pic" text,
  "jamBerangkat" text,
  "tujuan" text,
  "rute" text,
  "parkir" numeric(14,2),
  "tol" numeric(14,2),
  "bensin" numeric(14,2),
  "pkbm" numeric(14,2),
  "lainLain" numeric(14,2),
  "totalBiaya" numeric(14,2),
  "buktiPembayaranUrl" text,
  "jamMulaiPerjalanan" text,
  "jamTibaTujuan" text,
  "jamKembaliWarehouse" text,
  "jamSampaiWarehouse" text,
  "status" text,
  "driverNotes" text,
  "keterangan" text,
  "createdAt" text
);

-- bookingmobildetail
create table if not exists "bookingmobildetail" (
  id text primary key default gen_random_uuid()::text,
  "bookingId" text,
  "tanggal" text,
  "namaCustomer" text,
  "noPo" text,
  "totalCartoon" numeric(14,2),
  "parkir" text,
  "tol" text,
  "pkbm" text,
  "lainLain" text,
  "keterangan" text,
  "createdAt" text
);

-- kasgudang
create table if not exists "kasgudang" (
  id text primary key default gen_random_uuid()::text,
  "tanggal" text,
  "tipe" text,
  "keterangan" text,
  "nominal" numeric(14,2),
  "buktiUrl" text,
  "createdBy" text,
  "createdAt" text
);

-- teambuilding
create table if not exists "teambuilding" (
  id text primary key default gen_random_uuid()::text,
  "tanggal" text,
  "tipe" text,
  "keterangan" text,
  "nominal" numeric(14,2),
  "createdBy" text,
  "createdAt" text
);

-- expense
create table if not exists "expense" (
  id text primary key default gen_random_uuid()::text,
  "tanggal" text,
  "perusahaan" text,
  "kategori" text,
  "keterangan" text,
  "nominal" numeric(14,2),
  "bank" text,
  "rekening" text,
  "createdBy" text,
  "createdAt" text
);

-- pettycashperiod
create table if not exists "pettycashperiod" (
  id text primary key default gen_random_uuid()::text,
  "nama" text,
  "tanggalMulai" text,
  "tanggalSelesai" text,
  "saldoAwal" numeric(14,2),
  "keterangan" text,
  "status" text,
  "createdBy" text,
  "createdAt" text,
  "updatedAt" text
);

-- pettycash
create table if not exists "pettycash" (
  id text primary key default gen_random_uuid()::text,
  "periodId" text,
  "tanggal" text,
  "kategori" text,
  "keterangan" text,
  "nominal" numeric(14,2),
  "tipe" text,
  "buktiUrl" text,
  "statusBayar" text,
  "createdBy" text,
  "createdAt" text,
  "updatedAt" text
);

-- pettycashkategori
create table if not exists "pettycashkategori" (
  id text primary key default gen_random_uuid()::text,
  "kode" text,
  "nama" text,
  "urutan" integer,
  "aktif" text,
  "createdBy" text,
  "createdAt" text
);

-- assetwarehouse
create table if not exists "assetwarehouse" (
  id text primary key default gen_random_uuid()::text,
  "kategori" text,
  "code" text,
  "nama" text,
  "divisi" text,
  "zoneId" text,
  "qty" numeric(14,2),
  "status" text,
  "tanggalMasuk" text,
  "createdBy" text,
  "createdAt" text
);

-- assetopnamesession
create table if not exists "assetopnamesession" (
  id text primary key default gen_random_uuid()::text,
  "tanggal" text,
  "divisi" text,
  "status" text,
  "totalAsset" numeric(14,2),
  "terscan" numeric(14,2),
  "createdBy" text,
  "createdAt" text,
  "approvedAt" text
);

-- assetopnamelog
create table if not exists "assetopnamelog" (
  id text primary key default gen_random_uuid()::text,
  "sessionId" text,
  "assetId" text,
  "assetCode" text,
  "assetNama" text,
  "divisi" text,
  "qtyFisik" numeric(14,2),
  "qtySistem" numeric(14,2),
  "kondisi" text,
  "catatan" text,
  "scannedBy" text,
  "scannedAt" text,
  "createdAt" text
);

-- absensikaryawan
create table if not exists "absensikaryawan" (
  id text primary key default gen_random_uuid()::text,
  "karyawanId" text,
  "nama" text,
  "divisi" text,
  "jabatan" text,
  "tanggal" text,
  "jam" text,
  "tipe" text,
  "sumber" text,
  "status" text,
  "keterangan" text,
  "fingerprintId" text,
  "createdAt" text
);

-- suratperingatan
create table if not exists "suratperingatan" (
  id text primary key default gen_random_uuid()::text,
  "karyawanId" text,
  "karyawanNama" text,
  "jenisSP" text,
  "alasan" text,
  "tanggalSP" text,
  "masaBerlaku" text,
  "tanggalKadaluarsa" text,
  "status" text,
  "createdBy" text,
  "createdAt" text
);

-- laporankerja
create table if not exists "laporankerja" (
  id text primary key default gen_random_uuid()::text,
  "tanggal" text,
  "divisi" text,
  "pic" text,
  "shift" text,
  "totalOrang" numeric(14,2),
  "perbantuan" numeric(14,2),
  "perbantuanNames" text,
  "pengurangan" numeric(14,2),
  "jamLembur" numeric(14,2),
  "totalJamKerja" numeric(14,2),
  "kendala" text,
  "totalStaff" numeric(14,2),
  "totalAdmin" numeric(14,2),
  "totalOrder" numeric(14,2),
  "sisaOrder" numeric(14,2),
  "staffLemburNames" text,
  "totalPHL" numeric(14,2),
  "totalPO" numeric(14,2),
  "totalQty" numeric(14,2),
  "totalQtyInb" numeric(14,2),
  "totalInbound" numeric(14,2),
  "alasanPengurangan" text,
  "alasanLembur" text,
  "totalTeamLeader" numeric(14,2),
  "totalPelorporUtama" numeric(14,2),
  "jamPengurangan" numeric(14,2),
  "totalPekerjaLembur" numeric(14,2),
  "jamKerjaPHL" numeric(14,2),
  "jamPerbantuan" numeric(14,2),
  "noKartuTanpaKartu" text,
  "namaTanpaKartu" text,
  "namaTanpaSepatu" text,
  "kendalaPekerjaan" text,
  "autoHitung" text,
  "soreReload" text,
  "semuaMalamKartu" text,
  "semuaMemakaiSemuaKartu" text,
  "wpi" numeric(14,2),
  "totalKecukupanJamKerja" numeric(14,2),
  "totalKesesuaian" numeric(14,2),
  "createdBy" text,
  "createdAt" text
);

-- instruksilembur
create table if not exists "instruksilembur" (
  id text primary key default gen_random_uuid()::text,
  "tanggal" text,
  "judul" text,
  "keterangan" text,
  "createdBy" text,
  "createdAt" text
);

-- instruksilemburdetail
create table if not exists "instruksilemburdetail" (
  id text primary key default gen_random_uuid()::text,
  "instruksiId" text,
  "nama" text,
  "divisi" text,
  "divisiTujuan" text,
  "tipe" text,
  "shift" text,
  "jamKerja" numeric(14,2),
  "jamLembur" numeric(14,2),
  "totalJam" numeric(14,2),
  "createdAt" text
);

-- jadwalroster
create table if not exists "jadwalroster" (
  id text primary key default gen_random_uuid()::text,
  "Bulan" text,
  "Nama" text,
  "Jabatan" text,
  "Divisi" text,
  "1" text,
  "2" text,
  "3" text,
  "4" text,
  "5" text,
  "6" text,
  "7" text,
  "8" text,
  "9" text,
  "10" text,
  "11" text,
  "12" text,
  "13" text,
  "14" text,
  "15" text,
  "16" text,
  "17" text,
  "18" text,
  "19" text,
  "20" text,
  "21" text,
  "22" text,
  "23" text,
  "24" text,
  "25" text,
  "26" text,
  "27" text,
  "28" text,
  "29" text,
  "30" text,
  "31" text,
  "createdAt" text
);

-- jadwalshift
create table if not exists "jadwalshift" (
  id text primary key default gen_random_uuid()::text,
  "namaJadwal" text,
  "divisi" text,
  "shiftType" text,
  "jamMasuk" text,
  "jamPulang" text,
  "toleransiMenit" numeric(10,2),
  "createdAt" text
);

-- fcl_api_devices
create table if not exists "fcl_api_devices" (
  id text primary key default gen_random_uuid()::text,
  "nama" text,
  "merk" text,
  "ipaddress" text,
  "serialnumber" text,
  "aktif" text,
  "keterangan" text,
  "createdAt" text
);

-- ijin
create table if not exists "ijin" (
  id text primary key default gen_random_uuid()::text,
  "tanggal" text,
  "nama" text,
  "jenis" text,
  "status" text,
  "keterangan" text,
  "createdBy" text,
  "createdAt" text
);

-- lembur
create table if not exists "lembur" (
  id text primary key default gen_random_uuid()::text,
  "tanggal" text,
  "nama" text,
  "divisi" text,
  "jumlahJam" numeric(10,2),
  "status" text,
  "keterangan" text,
  "createdBy" text,
  "createdAt" text
);

-- sop
create table if not exists "sop" (
  id text primary key default gen_random_uuid()::text,
  "judul" text,
  "kategori" text,
  "konten" text,
  "updatedAt" text,
  "createdAt" text
);

-- Kolom tambahan untuk tabel yang sudah ada di database produksi
select fcl_add_column('users', 'username', 'text');
select fcl_add_column('users', 'password', 'text');
select fcl_add_column('users', 'nama', 'text');
select fcl_add_column('users', 'role', 'text');
select fcl_add_column('users', 'divisi', 'text');
select fcl_add_column('users', 'permissions', 'text');
select fcl_add_column('users', 'createdAt', 'text');
select fcl_add_column('users', 'email', 'text');
select fcl_add_column('users', 'status', 'text');
select fcl_add_column('users', 'updatedAt', 'text');
select fcl_add_column('karyawan', 'nama', 'text');
select fcl_add_column('karyawan', 'divisi', 'text');
select fcl_add_column('karyawan', 'jabatan', 'text');
select fcl_add_column('karyawan', 'cabang', 'text');
select fcl_add_column('karyawan', 'telepon', 'text');
select fcl_add_column('karyawan', 'email', 'text');
select fcl_add_column('karyawan', 'tanggalMasuk', 'text');
select fcl_add_column('karyawan', 'tanggalSelesai', 'text');
select fcl_add_column('karyawan', 'status', 'text');
select fcl_add_column('karyawan', 'sisaCuti', 'numeric(10,2)');
select fcl_add_column('karyawan', 'fingerprintId', 'text');
select fcl_add_column('karyawan', 'fingerprint_id', 'text');
select fcl_add_column('karyawan', 'fotoUrl', 'text');
select fcl_add_column('karyawan', 'foto', 'text');
select fcl_add_column('karyawan', 'alamat', 'text');
select fcl_add_column('karyawan', 'tanggalLahir', 'text');
select fcl_add_column('karyawan', 'jenisKelamin', 'text');
select fcl_add_column('karyawan', 'createdAt', 'text');
select fcl_add_column('karyawan', 'createdBy', 'text');
select fcl_add_column('karyawan', 'updatedAt', 'text');
select fcl_add_column('stock', 'sku', 'text');
select fcl_add_column('stock', 'nama', 'text');
select fcl_add_column('stock', 'barcode', 'text');
select fcl_add_column('stock', 'kategori', 'text');
select fcl_add_column('stock', 'satuan', 'text');
select fcl_add_column('stock', 'batch', 'text');
select fcl_add_column('stock', 'lokasi', 'text');
select fcl_add_column('stock', 'expDate', 'text');
select fcl_add_column('stock', 'stok', 'text');
select fcl_add_column('stock', 'stokMin', 'text');
select fcl_add_column('stock', 'stokAwal', 'numeric(14,2)');
select fcl_add_column('stock', 'harga', 'numeric(14,2)');
select fcl_add_column('stock', 'gudang', 'text');
select fcl_add_column('stock', 'rak', 'text');
select fcl_add_column('stock', 'bin', 'text');
select fcl_add_column('stock', 'createdAt', 'text');
select fcl_add_column('stock', 'createdBy', 'text');
select fcl_add_column('stock', 'updatedAt', 'text');
select fcl_add_column('stockopname', 'tanggal', 'text');
select fcl_add_column('stockopname', 'sku', 'text');
select fcl_add_column('stockopname', 'nama', 'text');
select fcl_add_column('stockopname', 'lokasi', 'text');
select fcl_add_column('stockopname', 'batch', 'text');
select fcl_add_column('stockopname', 'expDate', 'text');
select fcl_add_column('stockopname', 'stokSistem', 'numeric(14,2)');
select fcl_add_column('stockopname', 'stokFisik', 'numeric(14,2)');
select fcl_add_column('stockopname', 'selisih', 'numeric(14,2)');
select fcl_add_column('stockopname', 'status', 'text');
select fcl_add_column('stockopname', 'catatan', 'text');
select fcl_add_column('stockopname', 'createdAt', 'text');
select fcl_add_column('suratjalanmasuk', 'tanggal', 'text');
select fcl_add_column('suratjalanmasuk', 'noSJ', 'text');
select fcl_add_column('suratjalanmasuk', 'supplier', 'text');
select fcl_add_column('suratjalanmasuk', 'status', 'text');
select fcl_add_column('suratjalanmasuk', 'keterangan', 'text');
select fcl_add_column('suratjalanmasuk', 'createdAt', 'text');
select fcl_add_column('suratjalanmasukdetail', 'sjId', 'text');
select fcl_add_column('suratjalanmasukdetail', 'noSJ', 'text');
select fcl_add_column('suratjalanmasukdetail', 'sku', 'text');
select fcl_add_column('suratjalanmasukdetail', 'nama', 'text');
select fcl_add_column('suratjalanmasukdetail', 'satuan', 'text');
select fcl_add_column('suratjalanmasukdetail', 'batch', 'text');
select fcl_add_column('suratjalanmasukdetail', 'qty', 'numeric(14,2)');
select fcl_add_column('suratjalanmasukdetail', 'expDate', 'text');
select fcl_add_column('suratjalanmasukdetail', 'createdAt', 'text');
select fcl_add_column('order', 'noOrder', 'text');
select fcl_add_column('order', 'noResi', 'text');
select fcl_add_column('order', 'noresi', 'text');
select fcl_add_column('order', 'tanggal', 'text');
select fcl_add_column('order', 'pelanggan', 'text');
select fcl_add_column('order', 'alamat', 'text');
select fcl_add_column('order', 'kategori', 'text');
select fcl_add_column('order', 'status', 'text');
select fcl_add_column('order', 'totalItem', 'numeric(14,2)');
select fcl_add_column('order', 'total', 'numeric(14,2)');
select fcl_add_column('order', 'keterangan', 'text');
select fcl_add_column('order', 'sentAt', 'text');
select fcl_add_column('order', 'buktiPacking', 'text');
select fcl_add_column('order', 'createdBy', 'text');
select fcl_add_column('order', 'createdAt', 'text');
select fcl_add_column('order', 'updatedAt', 'text');
select fcl_add_column('orderdetail', 'orderId', 'text');
select fcl_add_column('orderdetail', 'noOrder', 'text');
select fcl_add_column('orderdetail', 'sku', 'text');
select fcl_add_column('orderdetail', 'nama', 'text');
select fcl_add_column('orderdetail', 'satuan', 'text');
select fcl_add_column('orderdetail', 'batch', 'text');
select fcl_add_column('orderdetail', 'lokasi', 'text');
select fcl_add_column('orderdetail', 'qty', 'numeric(14,2)');
select fcl_add_column('orderdetail', 'expDate', 'text');
select fcl_add_column('orderdetail', 'packedQty', 'text');
select fcl_add_column('orderdetail', 'createdAt', 'text');
select fcl_add_column('klaim', 'tanggal', 'text');
select fcl_add_column('klaim', 'pic', 'text');
select fcl_add_column('klaim', 'resi', 'text');
select fcl_add_column('klaim', 'harga', 'numeric(14,2)');
select fcl_add_column('klaim', 'status', 'text');
select fcl_add_column('klaim', 'keterangan', 'text');
select fcl_add_column('klaim', 'createdAt', 'text');
select fcl_add_column('klaimdetail', 'klaimId', 'text');
select fcl_add_column('klaimdetail', 'sku', 'text');
select fcl_add_column('klaimdetail', 'harga', 'numeric(14,2)');
select fcl_add_column('klaimdetail', 'createdAt', 'text');
select fcl_add_column('retur', 'tanggal', 'text');
select fcl_add_column('retur', 'noRetur', 'text');
select fcl_add_column('retur', 'sumber', 'text');
select fcl_add_column('retur', 'alasan', 'text');
select fcl_add_column('retur', 'keterangan', 'text');
select fcl_add_column('retur', 'createdAt', 'text');
select fcl_add_column('returdetail', 'returId', 'text');
select fcl_add_column('returdetail', 'noRetur', 'text');
select fcl_add_column('returdetail', 'sku', 'text');
select fcl_add_column('returdetail', 'nama', 'text');
select fcl_add_column('returdetail', 'satuan', 'text');
select fcl_add_column('returdetail', 'batch', 'text');
select fcl_add_column('returdetail', 'qty', 'numeric(14,2)');
select fcl_add_column('returdetail', 'expDate', 'text');
select fcl_add_column('returdetail', 'createdAt', 'text');
select fcl_add_column('bookingmobil', 'tanggal', 'text');
select fcl_add_column('bookingmobil', 'pic', 'text');
select fcl_add_column('bookingmobil', 'jamBerangkat', 'text');
select fcl_add_column('bookingmobil', 'tujuan', 'text');
select fcl_add_column('bookingmobil', 'rute', 'text');
select fcl_add_column('bookingmobil', 'parkir', 'numeric(14,2)');
select fcl_add_column('bookingmobil', 'tol', 'numeric(14,2)');
select fcl_add_column('bookingmobil', 'bensin', 'numeric(14,2)');
select fcl_add_column('bookingmobil', 'pkbm', 'numeric(14,2)');
select fcl_add_column('bookingmobil', 'lainLain', 'numeric(14,2)');
select fcl_add_column('bookingmobil', 'totalBiaya', 'numeric(14,2)');
select fcl_add_column('bookingmobil', 'buktiPembayaranUrl', 'text');
select fcl_add_column('bookingmobil', 'jamMulaiPerjalanan', 'text');
select fcl_add_column('bookingmobil', 'jamTibaTujuan', 'text');
select fcl_add_column('bookingmobil', 'jamKembaliWarehouse', 'text');
select fcl_add_column('bookingmobil', 'jamSampaiWarehouse', 'text');
select fcl_add_column('bookingmobil', 'status', 'text');
select fcl_add_column('bookingmobil', 'driverNotes', 'text');
select fcl_add_column('bookingmobil', 'keterangan', 'text');
select fcl_add_column('bookingmobil', 'createdAt', 'text');
select fcl_add_column('bookingmobildetail', 'bookingId', 'text');
select fcl_add_column('bookingmobildetail', 'tanggal', 'text');
select fcl_add_column('bookingmobildetail', 'namaCustomer', 'text');
select fcl_add_column('bookingmobildetail', 'noPo', 'text');
select fcl_add_column('bookingmobildetail', 'totalCartoon', 'numeric(14,2)');
select fcl_add_column('bookingmobildetail', 'parkir', 'text');
select fcl_add_column('bookingmobildetail', 'tol', 'text');
select fcl_add_column('bookingmobildetail', 'pkbm', 'text');
select fcl_add_column('bookingmobildetail', 'lainLain', 'text');
select fcl_add_column('bookingmobildetail', 'keterangan', 'text');
select fcl_add_column('bookingmobildetail', 'createdAt', 'text');
select fcl_add_column('kasgudang', 'tanggal', 'text');
select fcl_add_column('kasgudang', 'tipe', 'text');
select fcl_add_column('kasgudang', 'keterangan', 'text');
select fcl_add_column('kasgudang', 'nominal', 'numeric(14,2)');
select fcl_add_column('kasgudang', 'buktiUrl', 'text');
select fcl_add_column('kasgudang', 'createdBy', 'text');
select fcl_add_column('kasgudang', 'createdAt', 'text');
select fcl_add_column('teambuilding', 'tanggal', 'text');
select fcl_add_column('teambuilding', 'tipe', 'text');
select fcl_add_column('teambuilding', 'keterangan', 'text');
select fcl_add_column('teambuilding', 'nominal', 'numeric(14,2)');
select fcl_add_column('teambuilding', 'createdBy', 'text');
select fcl_add_column('teambuilding', 'createdAt', 'text');
select fcl_add_column('expense', 'tanggal', 'text');
select fcl_add_column('expense', 'perusahaan', 'text');
select fcl_add_column('expense', 'kategori', 'text');
select fcl_add_column('expense', 'keterangan', 'text');
select fcl_add_column('expense', 'nominal', 'numeric(14,2)');
select fcl_add_column('expense', 'bank', 'text');
select fcl_add_column('expense', 'rekening', 'text');
select fcl_add_column('expense', 'createdBy', 'text');
select fcl_add_column('expense', 'createdAt', 'text');
select fcl_add_column('pettycashperiod', 'nama', 'text');
select fcl_add_column('pettycashperiod', 'tanggalMulai', 'text');
select fcl_add_column('pettycashperiod', 'tanggalSelesai', 'text');
select fcl_add_column('pettycashperiod', 'saldoAwal', 'numeric(14,2)');
select fcl_add_column('pettycashperiod', 'keterangan', 'text');
select fcl_add_column('pettycashperiod', 'status', 'text');
select fcl_add_column('pettycashperiod', 'createdBy', 'text');
select fcl_add_column('pettycashperiod', 'createdAt', 'text');
select fcl_add_column('pettycashperiod', 'updatedAt', 'text');
select fcl_add_column('pettycash', 'periodId', 'text');
select fcl_add_column('pettycash', 'tanggal', 'text');
select fcl_add_column('pettycash', 'kategori', 'text');
select fcl_add_column('pettycash', 'keterangan', 'text');
select fcl_add_column('pettycash', 'nominal', 'numeric(14,2)');
select fcl_add_column('pettycash', 'tipe', 'text');
select fcl_add_column('pettycash', 'buktiUrl', 'text');
select fcl_add_column('pettycash', 'statusBayar', 'text');
select fcl_add_column('pettycash', 'createdBy', 'text');
select fcl_add_column('pettycash', 'createdAt', 'text');
select fcl_add_column('pettycash', 'updatedAt', 'text');
select fcl_add_column('pettycashkategori', 'kode', 'text');
select fcl_add_column('pettycashkategori', 'nama', 'text');
select fcl_add_column('pettycashkategori', 'urutan', 'integer');
select fcl_add_column('pettycashkategori', 'aktif', 'text');
select fcl_add_column('pettycashkategori', 'createdBy', 'text');
select fcl_add_column('pettycashkategori', 'createdAt', 'text');
select fcl_add_column('assetwarehouse', 'kategori', 'text');
select fcl_add_column('assetwarehouse', 'code', 'text');
select fcl_add_column('assetwarehouse', 'nama', 'text');
select fcl_add_column('assetwarehouse', 'divisi', 'text');
select fcl_add_column('assetwarehouse', 'zoneId', 'text');
select fcl_add_column('assetwarehouse', 'qty', 'numeric(14,2)');
select fcl_add_column('assetwarehouse', 'status', 'text');
select fcl_add_column('assetwarehouse', 'tanggalMasuk', 'text');
select fcl_add_column('assetwarehouse', 'createdBy', 'text');
select fcl_add_column('assetwarehouse', 'createdAt', 'text');
select fcl_add_column('assetopnamesession', 'tanggal', 'text');
select fcl_add_column('assetopnamesession', 'divisi', 'text');
select fcl_add_column('assetopnamesession', 'status', 'text');
select fcl_add_column('assetopnamesession', 'totalAsset', 'numeric(14,2)');
select fcl_add_column('assetopnamesession', 'terscan', 'numeric(14,2)');
select fcl_add_column('assetopnamesession', 'createdBy', 'text');
select fcl_add_column('assetopnamesession', 'createdAt', 'text');
select fcl_add_column('assetopnamesession', 'approvedAt', 'text');
select fcl_add_column('assetopnamelog', 'sessionId', 'text');
select fcl_add_column('assetopnamelog', 'assetId', 'text');
select fcl_add_column('assetopnamelog', 'assetCode', 'text');
select fcl_add_column('assetopnamelog', 'assetNama', 'text');
select fcl_add_column('assetopnamelog', 'divisi', 'text');
select fcl_add_column('assetopnamelog', 'qtyFisik', 'numeric(14,2)');
select fcl_add_column('assetopnamelog', 'qtySistem', 'numeric(14,2)');
select fcl_add_column('assetopnamelog', 'kondisi', 'text');
select fcl_add_column('assetopnamelog', 'catatan', 'text');
select fcl_add_column('assetopnamelog', 'scannedBy', 'text');
select fcl_add_column('assetopnamelog', 'scannedAt', 'text');
select fcl_add_column('assetopnamelog', 'createdAt', 'text');
select fcl_add_column('absensikaryawan', 'karyawanId', 'text');
select fcl_add_column('absensikaryawan', 'nama', 'text');
select fcl_add_column('absensikaryawan', 'divisi', 'text');
select fcl_add_column('absensikaryawan', 'jabatan', 'text');
select fcl_add_column('absensikaryawan', 'tanggal', 'text');
select fcl_add_column('absensikaryawan', 'jam', 'text');
select fcl_add_column('absensikaryawan', 'tipe', 'text');
select fcl_add_column('absensikaryawan', 'sumber', 'text');
select fcl_add_column('absensikaryawan', 'status', 'text');
select fcl_add_column('absensikaryawan', 'keterangan', 'text');
select fcl_add_column('absensikaryawan', 'fingerprintId', 'text');
select fcl_add_column('absensikaryawan', 'createdAt', 'text');
select fcl_add_column('suratperingatan', 'karyawanId', 'text');
select fcl_add_column('suratperingatan', 'karyawanNama', 'text');
select fcl_add_column('suratperingatan', 'jenisSP', 'text');
select fcl_add_column('suratperingatan', 'alasan', 'text');
select fcl_add_column('suratperingatan', 'tanggalSP', 'text');
select fcl_add_column('suratperingatan', 'masaBerlaku', 'text');
select fcl_add_column('suratperingatan', 'tanggalKadaluarsa', 'text');
select fcl_add_column('suratperingatan', 'status', 'text');
select fcl_add_column('suratperingatan', 'createdBy', 'text');
select fcl_add_column('suratperingatan', 'createdAt', 'text');
select fcl_add_column('laporankerja', 'tanggal', 'text');
select fcl_add_column('laporankerja', 'divisi', 'text');
select fcl_add_column('laporankerja', 'pic', 'text');
select fcl_add_column('laporankerja', 'shift', 'text');
select fcl_add_column('laporankerja', 'totalOrang', 'numeric(14,2)');
select fcl_add_column('laporankerja', 'perbantuan', 'numeric(14,2)');
select fcl_add_column('laporankerja', 'perbantuanNames', 'text');
select fcl_add_column('laporankerja', 'pengurangan', 'numeric(14,2)');
select fcl_add_column('laporankerja', 'jamLembur', 'numeric(14,2)');
select fcl_add_column('laporankerja', 'totalJamKerja', 'numeric(14,2)');
select fcl_add_column('laporankerja', 'kendala', 'text');
select fcl_add_column('laporankerja', 'totalStaff', 'numeric(14,2)');
select fcl_add_column('laporankerja', 'totalAdmin', 'numeric(14,2)');
select fcl_add_column('laporankerja', 'totalOrder', 'numeric(14,2)');
select fcl_add_column('laporankerja', 'sisaOrder', 'numeric(14,2)');
select fcl_add_column('laporankerja', 'staffLemburNames', 'text');
select fcl_add_column('laporankerja', 'totalPHL', 'numeric(14,2)');
select fcl_add_column('laporankerja', 'totalPO', 'numeric(14,2)');
select fcl_add_column('laporankerja', 'totalQty', 'numeric(14,2)');
select fcl_add_column('laporankerja', 'totalQtyInb', 'numeric(14,2)');
select fcl_add_column('laporankerja', 'totalInbound', 'numeric(14,2)');
select fcl_add_column('laporankerja', 'alasanPengurangan', 'text');
select fcl_add_column('laporankerja', 'alasanLembur', 'text');
select fcl_add_column('laporankerja', 'totalTeamLeader', 'numeric(14,2)');
select fcl_add_column('laporankerja', 'totalPelorporUtama', 'numeric(14,2)');
select fcl_add_column('laporankerja', 'jamPengurangan', 'numeric(14,2)');
select fcl_add_column('laporankerja', 'totalPekerjaLembur', 'numeric(14,2)');
select fcl_add_column('laporankerja', 'jamKerjaPHL', 'numeric(14,2)');
select fcl_add_column('laporankerja', 'jamPerbantuan', 'numeric(14,2)');
select fcl_add_column('laporankerja', 'noKartuTanpaKartu', 'text');
select fcl_add_column('laporankerja', 'namaTanpaKartu', 'text');
select fcl_add_column('laporankerja', 'namaTanpaSepatu', 'text');
select fcl_add_column('laporankerja', 'kendalaPekerjaan', 'text');
select fcl_add_column('laporankerja', 'autoHitung', 'text');
select fcl_add_column('laporankerja', 'soreReload', 'text');
select fcl_add_column('laporankerja', 'semuaMalamKartu', 'text');
select fcl_add_column('laporankerja', 'semuaMemakaiSemuaKartu', 'text');
select fcl_add_column('laporankerja', 'wpi', 'numeric(14,2)');
select fcl_add_column('laporankerja', 'totalKecukupanJamKerja', 'numeric(14,2)');
select fcl_add_column('laporankerja', 'totalKesesuaian', 'numeric(14,2)');
select fcl_add_column('laporankerja', 'createdBy', 'text');
select fcl_add_column('laporankerja', 'createdAt', 'text');
select fcl_add_column('instruksilembur', 'tanggal', 'text');
select fcl_add_column('instruksilembur', 'judul', 'text');
select fcl_add_column('instruksilembur', 'keterangan', 'text');
select fcl_add_column('instruksilembur', 'createdBy', 'text');
select fcl_add_column('instruksilembur', 'createdAt', 'text');
select fcl_add_column('instruksilemburdetail', 'instruksiId', 'text');
select fcl_add_column('instruksilemburdetail', 'nama', 'text');
select fcl_add_column('instruksilemburdetail', 'divisi', 'text');
select fcl_add_column('instruksilemburdetail', 'divisiTujuan', 'text');
select fcl_add_column('instruksilemburdetail', 'tipe', 'text');
select fcl_add_column('instruksilemburdetail', 'shift', 'text');
select fcl_add_column('instruksilemburdetail', 'jamKerja', 'numeric(14,2)');
select fcl_add_column('instruksilemburdetail', 'jamLembur', 'numeric(14,2)');
select fcl_add_column('instruksilemburdetail', 'totalJam', 'numeric(14,2)');
select fcl_add_column('instruksilemburdetail', 'createdAt', 'text');
select fcl_add_column('jadwalroster', 'Bulan', 'text');
select fcl_add_column('jadwalroster', 'Nama', 'text');
select fcl_add_column('jadwalroster', 'Jabatan', 'text');
select fcl_add_column('jadwalroster', 'Divisi', 'text');
select fcl_add_column('jadwalroster', '1', 'text');
select fcl_add_column('jadwalroster', '2', 'text');
select fcl_add_column('jadwalroster', '3', 'text');
select fcl_add_column('jadwalroster', '4', 'text');
select fcl_add_column('jadwalroster', '5', 'text');
select fcl_add_column('jadwalroster', '6', 'text');
select fcl_add_column('jadwalroster', '7', 'text');
select fcl_add_column('jadwalroster', '8', 'text');
select fcl_add_column('jadwalroster', '9', 'text');
select fcl_add_column('jadwalroster', '10', 'text');
select fcl_add_column('jadwalroster', '11', 'text');
select fcl_add_column('jadwalroster', '12', 'text');
select fcl_add_column('jadwalroster', '13', 'text');
select fcl_add_column('jadwalroster', '14', 'text');
select fcl_add_column('jadwalroster', '15', 'text');
select fcl_add_column('jadwalroster', '16', 'text');
select fcl_add_column('jadwalroster', '17', 'text');
select fcl_add_column('jadwalroster', '18', 'text');
select fcl_add_column('jadwalroster', '19', 'text');
select fcl_add_column('jadwalroster', '20', 'text');
select fcl_add_column('jadwalroster', '21', 'text');
select fcl_add_column('jadwalroster', '22', 'text');
select fcl_add_column('jadwalroster', '23', 'text');
select fcl_add_column('jadwalroster', '24', 'text');
select fcl_add_column('jadwalroster', '25', 'text');
select fcl_add_column('jadwalroster', '26', 'text');
select fcl_add_column('jadwalroster', '27', 'text');
select fcl_add_column('jadwalroster', '28', 'text');
select fcl_add_column('jadwalroster', '29', 'text');
select fcl_add_column('jadwalroster', '30', 'text');
select fcl_add_column('jadwalroster', '31', 'text');
select fcl_add_column('jadwalroster', 'createdAt', 'text');
select fcl_add_column('jadwalshift', 'namaJadwal', 'text');
select fcl_add_column('jadwalshift', 'divisi', 'text');
select fcl_add_column('jadwalshift', 'shiftType', 'text');
select fcl_add_column('jadwalshift', 'jamMasuk', 'text');
select fcl_add_column('jadwalshift', 'jamPulang', 'text');
select fcl_add_column('jadwalshift', 'toleransiMenit', 'numeric(10,2)');
select fcl_add_column('jadwalshift', 'createdAt', 'text');
select fcl_add_column('fcl_api_devices', 'nama', 'text');
select fcl_add_column('fcl_api_devices', 'merk', 'text');
select fcl_add_column('fcl_api_devices', 'ipaddress', 'text');
select fcl_add_column('fcl_api_devices', 'serialnumber', 'text');
select fcl_add_column('fcl_api_devices', 'aktif', 'text');
select fcl_add_column('fcl_api_devices', 'keterangan', 'text');
select fcl_add_column('fcl_api_devices', 'createdAt', 'text');
select fcl_add_column('ijin', 'tanggal', 'text');
select fcl_add_column('ijin', 'nama', 'text');
select fcl_add_column('ijin', 'jenis', 'text');
select fcl_add_column('ijin', 'status', 'text');
select fcl_add_column('ijin', 'keterangan', 'text');
select fcl_add_column('ijin', 'createdBy', 'text');
select fcl_add_column('ijin', 'createdAt', 'text');
select fcl_add_column('lembur', 'tanggal', 'text');
select fcl_add_column('lembur', 'nama', 'text');
select fcl_add_column('lembur', 'divisi', 'text');
select fcl_add_column('lembur', 'jumlahJam', 'numeric(10,2)');
select fcl_add_column('lembur', 'status', 'text');
select fcl_add_column('lembur', 'keterangan', 'text');
select fcl_add_column('lembur', 'createdBy', 'text');
select fcl_add_column('lembur', 'createdAt', 'text');
select fcl_add_column('sop', 'judul', 'text');
select fcl_add_column('sop', 'kategori', 'text');
select fcl_add_column('sop', 'konten', 'text');
select fcl_add_column('sop', 'updatedAt', 'text');
select fcl_add_column('sop', 'createdAt', 'text');

-- Pastikan tipe kolom stok sesuai kontrak backend (regex + cast ke numeric)
do $$ begin
  alter table stock alter column stok type text using stok::text;
exception when others then null; end $$;
do $$ begin
  alter table stock alter column "stokMin" type text using "stokMin"::text;
exception when others then null; end $$;
do $$ begin
  alter table stock alter column "expDate" type text using "expDate"::text;
exception when others then null; end $$;

-- Cleanup: fitur "Akun Gudang FCL" dihapus.
-- Hapus sisa menu/permission pada database yang pernah menjalankan fitur ini.
do $$ begin
  if to_regclass('public.sys_role_permissions') is not null
     and to_regclass('public.sys_permissions') is not null then
    delete from sys_role_permissions
     where permission_id in (select id from sys_permissions where code = 'fcl:akun');
  end if;
  if to_regclass('public.sys_permissions') is not null then
    delete from sys_permissions where code = 'fcl:akun';
  end if;
  if to_regclass('public.sys_menus') is not null then
    delete from sys_menus where path in ('/fcl/sdm/akun', '/opm/fcl-akun');
  end if;
exception when others then null;
end $$;
