-- Kategori transaksi Petty Cash (daftar custom yang bisa ditambah/diubah).
-- Dijalankan sekali pada database ghost_wms.

create table if not exists pettycashkategori (
  id       text primary key default gen_random_uuid()::text,
  kode     text,
  nama     text,
  urutan   integer default 0,
  aktif    text default 'Ya',
  "createdBy" text,
  "createdAt" text
);

create unique index if not exists pettycashkategori_kode_key on pettycashkategori (kode);

-- Kategori bawaan sesuai daftar yang digunakan gudang.
insert into pettycashkategori (kode, nama, urutan, aktif) values
  ('A', 'WATER SUPPLY', 1, 'Ya'),
  ('B', 'DRINKING WATER', 2, 'Ya'),
  ('C', 'MAINTENANCE', 3, 'Ya'),
  ('D', 'PEMBELIAN OFFLINE', 4, 'Ya'),
  ('E', 'PEMBELIAN ONLINE', 5, 'Ya'),
  ('F', 'CONSUMABLE', 6, 'Ya'),
  ('G', 'GOSEND/LALAMOVE', 7, 'Ya'),
  ('H', 'ONGKIR DISTRIBUTOR', 8, 'Ya'),
  ('I', 'RETUR', 9, 'Ya'),
  ('J', 'ADMIN FEE', 10, 'Ya'),
  ('K', 'PORTER', 11, 'Ya'),
  ('L', 'SEWA ALAT BERAT', 12, 'Ya'),
  ('M', 'BENSIN', 13, 'Ya'),
  ('N', 'PARKIR', 14, 'Ya'),
  ('O', 'TOL', 15, 'Ya'),
  ('P', 'GOJEK/GRAB/TRANSPORTASI MOTOR/MOBIL', 16, 'Ya'),
  ('Q', 'EVENT MAKAN-MAKAN', 17, 'Ya'),
  ('R', 'PANCA KOBRA', 18, 'Ya')
on conflict (kode) do nothing;
