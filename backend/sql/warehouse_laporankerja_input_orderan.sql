-- Field tambahan untuk form "Input Laporan Pengerjaan Orderan" (Laporan Kerja).
-- Dijalankan sekali pada database ghost_wms.

alter table laporankerja
  add column if not exists "totalTeamLeader" numeric,
  add column if not exists "totalPelorporUtama" numeric,
  add column if not exists "jamPengurangan" numeric,
  add column if not exists "totalPekerjaLembur" numeric,
  add column if not exists "noKartuTanpaKartu" text,
  add column if not exists "namaTanpaKartu" text,
  add column if not exists "autoHitung" text,
  add column if not exists "soreReload" text,
  add column if not exists "semuaMalamKartu" text,
  add column if not exists "semuaMemakaiSemuaKartu" text,
  add column if not exists "wpi" numeric,
  add column if not exists "totalKecukupanJamKerja" numeric,
  add column if not exists "totalKesesuaian" numeric,
  add column if not exists "jamPerbantuan" numeric,
  add column if not exists "namaTanpaSepatu" text,
  add column if not exists "kendalaPekerjaan" text,
  add column if not exists "perbantuanNames" text;
