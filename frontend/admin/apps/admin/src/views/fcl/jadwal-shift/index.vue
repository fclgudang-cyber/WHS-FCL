<script lang="ts" setup>
/**
 * Jadwal Shift / Roster — layout seperti aplikasi FCL asli:
 * filter bulan, divisi, nama + aksi Refresh/Setelan Jam/Template/Export/Import,
 * legenda shift, ringkasan harian, grid karyawan × hari dengan sel berwarna,
 * klik sel untuk ganti shift. Semua warna memakai token tema + warna shift.
 */
import { computed, onMounted, ref, watch } from 'vue';

import { Page } from '@vben/common-ui';
import { LucideDownload, LucidePlus, LucideRefreshCw } from '@vben/icons';

import { Modal, message, notification } from 'ant-design-vue';
import dayjs from 'dayjs';

import { fclDelete, fclInsert, fclPatch, fclQuery } from '#/api/fcl';
import FclCrud from '#/components/fcl/fcl-crud.vue';

interface RosterRow {
  id: string;
  Bulan?: string;
  Nama?: string;
  Jabatan?: string;
  Divisi?: string;
  [key: string]: any;
}
interface KaryawanRow {
  id: string;
  nama?: string | null;
  divisi?: string | null;
  jabatan?: string | null;
  status?: string | null;
}

const month = ref(dayjs().format('YYYY-MM'));
const divisiFilter = ref('');
const nameFilter = ref('');
const summaryDay = ref(dayjs().date());
const loading = ref(true);
const rows = ref<RosterRow[]>([]);
const karyawanList = ref<KaryawanRow[]>([]);
const syncing = ref(false);
const addOpen = ref(false);
const settingsOpen = ref(false);
const newRow = ref({ Nama: '', Jabatan: '', Divisi: '' });
const importInput = ref<HTMLInputElement | null>(null);

const SHIFTS = ['', 'PAGI', 'MALAM', 'SETENGAH HARI', 'SPLIT', 'SD', 'STD', 'IJIN', 'CUTI', 'SAKIT', 'ALFA', 'OFF', 'LIBUR'];

const SHIFT_META: Record<string, { label: string; bg: string; fg: string }> = {
  PAGI: { label: 'PAGI', bg: 'hsl(152 60% 24%)', fg: 'hsl(152 80% 78%)' },
  MALAM: { label: 'MALAM', bg: 'hsl(200 60% 24%)', fg: 'hsl(198 90% 80%)' },
  SETENGAH_HARI: { label: 'SETENGAH HARI', bg: 'hsl(175 55% 22%)', fg: 'hsl(175 80% 75%)' },
  SPLIT: { label: 'SPLIT', bg: 'hsl(175 55% 22%)', fg: 'hsl(175 80% 75%)' },
  SD: { label: 'SD', bg: 'hsl(0 55% 30%)', fg: 'hsl(0 90% 80%)' },
  STD: { label: 'STD', bg: 'hsl(265 45% 30%)', fg: 'hsl(265 90% 85%)' },
  IJIN: { label: 'IJIN', bg: 'hsl(38 70% 26%)', fg: 'hsl(40 95% 75%)' },
  CUTI: { label: 'CUTI', bg: 'hsl(0 60% 34%)', fg: 'hsl(0 90% 82%)' },
  SAKIT: { label: 'SAKIT', bg: 'hsl(38 70% 26%)', fg: 'hsl(40 95% 75%)' },
  ALFA: { label: 'ALFA', bg: 'hsl(0 60% 34%)', fg: 'hsl(0 90% 82%)' },
  OFF: { label: 'OFF', bg: 'hsl(0 45% 26%)', fg: 'hsl(0 80% 80%)' },
  LIBUR: { label: 'LIBUR', bg: 'hsl(0 45% 26%)', fg: 'hsl(0 80% 80%)' },
};

function categoryOf(v?: string): string {
  const s = String(v ?? '').toUpperCase().trim();
  if (!s) return '';
  if (s.includes('SETENGAH HARI') || s.includes('SPLIT')) return 'SETENGAH_HARI';
  if (s.includes('MALAM')) return 'MALAM';
  if (s.includes('PAGI')) return 'PAGI';
  for (const k of ['SD', 'STD', 'IJIN', 'CUTI', 'SAKIT', 'ALFA', 'OFF', 'LIBUR']) {
    if (s.includes(k)) return k;
  }
  return 'OTHER';
}

function cellStyle(v?: string): Record<string, string> {
  const meta = SHIFT_META[categoryOf(v)];
  return meta
    ? { background: meta.bg, color: meta.fg, borderColor: meta.fg + '55' }
    : { background: 'hsl(var(--muted))', color: 'hsl(var(--muted-foreground))', borderColor: 'transparent' };
}

const DAYS_IN_MONTH = computed(() => dayjs(`${month.value}-01`).daysInMonth());
const DAYS = computed(() => Array.from({ length: DAYS_IN_MONTH.value }, (_, i) => i + 1));
const DAY_NAMES = ['MIN', 'SEN', 'SEL', 'RAB', 'KAM', 'JUM', 'SAB'];

function dayLabel(d: number): string {
  return DAY_NAMES[dayjs(`${month.value}-${String(d).padStart(2, '0')}`).day()];
}

const karyawanOptions = computed(() => [...new Set(karyawanList.value
  .map((k) => String(k.nama ?? '').trim())
  .filter(Boolean))].sort((a, b) => a.localeCompare(b, 'id')));

watch(() => newRow.value.Nama, (nama) => {
  const k = karyawanMap().get(String(nama ?? '').trim().toLowerCase());
  newRow.value.Jabatan = k ? String(k.jabatan ?? '') : '';
  newRow.value.Divisi = k ? String(k.divisi ?? '') : '';
});

const divisiOptions = computed(() =>
  [...new Set([
    ...karyawanList.value.map((k) => String(k.divisi ?? '').trim()),
    ...rows.value.map((r) => String(r.Divisi ?? '').trim()),
  ].filter(Boolean))].sort((a, b) => a.localeCompare(b, 'id')),
);

const visibleRows = computed(() => {
  const q = nameFilter.value.trim().toLowerCase();
  return rows.value.filter((r) => {
    if (divisiFilter.value && String(r.Divisi ?? '') !== divisiFilter.value) return false;
    if (q && !String(r.Nama ?? '').toLowerCase().includes(q)) return false;
    return true;
  });
});

const summary = computed(() => {
  const c: Record<string, number> = { PAGI: 0, MALAM: 0, OFF: 0, CUTI: 0, ALFA: 0, SAKIT: 0, IJIN: 0 };
  visibleRows.value.forEach((r) => {
    const cat = categoryOf(r[String(summaryDay.value)]);
    if (cat in c) c[cat] += 1;
  });
  return { total: visibleRows.value.length, ...c };
});

function karyawanMap(): Map<string, KaryawanRow> {
  const map = new Map<string, KaryawanRow>();
  karyawanList.value.forEach((k) => {
    const nama = String(k.nama ?? '').trim().toLowerCase();
    if (nama) map.set(nama, k);
  });
  return map;
}

// Samakan Divisi & Jabatan roster dengan data master Karyawan (berdasarkan nama).
async function syncDivisiJabatan(addMissing = false) {
  if (syncing.value) return { updated: 0, added: 0 };
  syncing.value = true;
  let updated = 0, added = 0;
  try {
    const map = karyawanMap();
    const existing = new Set(rows.value.map((r) => String(r.Nama ?? '').trim().toLowerCase()));
    for (const r of rows.value) {
      const k = map.get(String(r.Nama ?? '').trim().toLowerCase());
      if (!k) continue;
      const divisi = String(k.divisi ?? '').trim();
      const jabatan = String(k.jabatan ?? '').trim();
      if (String(r.Divisi ?? '').trim() !== divisi || String(r.Jabatan ?? '').trim() !== jabatan) {
        await fclPatch('jadwalroster', r.id, { Divisi: divisi, Jabatan: jabatan });
        r.Divisi = divisi;
        r.Jabatan = jabatan;
        updated++;
      }
    }
    if (addMissing) {
      const payload = karyawanList.value
        .filter((k) => !existing.has(String(k.nama ?? '').trim().toLowerCase()))
        .map((k) => ({
          Bulan: month.value,
          Nama: String(k.nama ?? '').trim(),
          Jabatan: String(k.jabatan ?? '').trim(),
          Divisi: String(k.divisi ?? '').trim(),
        }))
        .filter((p) => !!p.Nama);
      if (payload.length) {
        await fclInsert('jadwalroster', payload);
        added = payload.length;
      }
    }
  } catch (err: any) {
    notification.error({ message: 'Sinkronisasi Karyawan', description: err?.message || 'Gagal menyinkronkan data' });
  } finally {
    syncing.value = false;
  }
  return { updated, added };
}

async function syncKaryawan() {
  const res = await syncDivisiJabatan(true);
  notification.success({
    message: 'Sinkronisasi Karyawan',
    description: `${res.updated} baris diperbarui, ${res.added} karyawan ditambahkan`,
  });
  await load(false, false);
}

async function load(silent = false, doSync = true) {
  if (!silent) loading.value = true;
  try {
    const [all, kars] = await Promise.all([
      fclQuery<RosterRow>('jadwalroster', { select: '*', limit: 10000 }),
      fclQuery<KaryawanRow>('karyawan', { select: '*', order: 'nama.asc', limit: 2000 }),
    ]);
    karyawanList.value = kars || [];
    rows.value = (all || []).filter((r) => String(r.Bulan ?? '').startsWith(month.value));
    if (doSync && rows.value.length) await syncDivisiJabatan(false);
  } catch (err: any) {
    console.error('[fcl/jadwal-shift]', err);
    notification.error({ message: 'Jadwal Shift', description: err?.message || 'Gagal memuat roster' });
  } finally {
    if (!silent) loading.value = false;
  }
}

async function updateCell(row: RosterRow, day: number, value: string) {
  try {
    await fclPatch('jadwalroster', row.id, { [String(day)]: value });
    row[String(day)] = value;
  } catch (err: any) {
    notification.error({ message: 'Gagal simpan', description: err?.message || String(err) });
  }
}

async function addRow() {
  if (!newRow.value.Nama.trim()) { message.warning('Nama wajib diisi'); return; }
  await fclInsert('jadwalroster', [{
    Bulan: month.value,
    Nama: newRow.value.Nama.trim(),
    Jabatan: newRow.value.Jabatan.trim(),
    Divisi: newRow.value.Divisi.trim(),
  }]);
  notification.success({ message: 'Roster', description: 'Karyawan ditambahkan' });
  addOpen.value = false;
  newRow.value = { Nama: '', Jabatan: '', Divisi: '' };
  await load(true);
}

function removeRow(row: RosterRow) {
  Modal.confirm({
    title: `Hapus ${row.Nama}?`,
    content: 'Baris roster ini akan dihapus permanen.',
    okText: 'Hapus', okType: 'danger',
    onOk: async () => { await fclDelete('jadwalroster', row.id); await load(true); },
  });
}

function downloadCsv(filename: string, lines: string[][]) {
  const csv = lines.map((cols) => cols.map((c) => `"${String(c ?? '').replace(/"/g, '""')}"`).join(',')).join('\n');
  const url = URL.createObjectURL(new Blob(['﻿' + csv], { type: 'text/csv;charset=utf-8;' }));
  const a = document.createElement('a'); a.href = url; a.download = filename; a.click(); URL.revokeObjectURL(url);
}

function exportRoster() {
  const header = ['Nama Karyawan', 'Jabatan', 'Divisi', ...DAYS.value.map((d) => `${dayLabel(d)} ${d}`)];
  const lines = [header, ...visibleRows.value.map((r) => [r.Nama ?? '', r.Jabatan ?? '', r.Divisi ?? '', ...DAYS.value.map((d) => String(r[String(d)] ?? ''))])];
  downloadCsv(`jadwal-shift-${month.value}.csv`, lines);
}

function downloadTemplate() {
  downloadCsv('template-jadwal-shift.csv', [['Nama Karyawan', 'Jabatan', 'Divisi', ...DAYS.value.map((d) => `${dayLabel(d)} ${d}`)]]);
}

async function importRoster(event: Event) {
  const file = (event.target as HTMLInputElement).files?.[0];
  if (!file) return;
  try {
    const text = await file.text();
    const parsed = text.replace(/^\uFEFF/, '').split(/\r?\n/).filter(Boolean)
      .map((line) => (line.match(/("([^"]|"")*"|[^,])+/g) ?? []).map((c) => c.replace(/^"|"$/g, '').replace(/""/g, '"').trim()));
    const payload = parsed.slice(1).filter((c) => c[0]).map((c) => {
      const row: Record<string, string> = { Bulan: month.value, Nama: c[0], Jabatan: c[1] ?? '', Divisi: c[2] ?? '' };
      DAYS.value.forEach((d, i) => { row[String(d)] = c[3 + i] ?? ''; });
      return row;
    });
    if (!payload.length) throw new Error('Tidak ada baris valid');
    await fclInsert('jadwalroster', payload);
    notification.success({ message: 'Import Roster', description: `${payload.length} baris ditambahkan` });
    await load(true);
  } catch (err: any) {
    notification.error({ message: 'Import Roster', description: err?.message || 'Gagal membaca CSV' });
  } finally {
    if (importInput.value) importInput.value.value = '';
  }
}

const shiftColumns: any[] = [
  { key: 'namaJadwal', title: 'Nama Jadwal', ellipsis: true },
  { key: 'divisi', title: 'Divisi', width: 140 },
  { key: 'shiftType', title: 'Tipe', width: 110 },
  { key: 'jamMasuk', title: 'Masuk', width: 90 },
  { key: 'jamPulang', title: 'Pulang', width: 90 },
  { key: 'toleransiMenit', title: 'Toleransi (mnt)', width: 120, align: 'right' },
];
const shiftFields: any[] = [
  { name: 'namaJadwal', label: 'Nama Jadwal', type: 'input', required: true },
  { name: 'divisi', label: 'Divisi', type: 'input' },
  { name: 'shiftType', label: 'Tipe', type: 'select', options: ['Pagi', 'Malam', 'SPLIT', 'Custom'] },
  { name: 'jamMasuk', label: 'Jam Masuk', type: 'input', placeholder: '08:00' },
  { name: 'jamPulang', label: 'Jam Pulang', type: 'input', placeholder: '17:00' },
  { name: 'toleransiMenit', label: 'Toleransi (menit)', type: 'number', min: 0 },
];

watch(month, () => load());
onMounted(() => load());
</script>

<template>
  <Page auto-content-height>
    <div class="js-root">
      <!-- ── Filter & aksi ── -->
      <div class="js-filter">
        <div class="js-field"><label>BULAN &amp; TAHUN</label><input v-model="month" type="month" class="js-input" /></div>
        <div class="js-field"><label>DIVISI</label>
          <select v-model="divisiFilter" class="js-input">
            <option value="">Semua Divisi</option>
            <option v-for="d in divisiOptions" :key="d" :value="d">{{ d }}</option>
          </select>
        </div>
        <div class="js-field"><label>CARI NAMA</label><input v-model="nameFilter" class="js-input" placeholder="Nama karyawan..." /></div>
        <button class="js-btn js-btn-ghost" @click="load()"><LucideRefreshCw :size="13" /> Refresh</button>
        <button class="js-btn js-btn-ghost" :disabled="syncing" @click="syncKaryawan"><LucideRefreshCw :size="13" /> {{ syncing ? 'Sinkron...' : 'Sync Karyawan' }}</button>
        <button class="js-btn js-btn-ghost" @click="settingsOpen = true">Setelan Jam</button>
        <div class="js-spacer"></div>
        <input ref="importInput" type="file" accept=".csv,text/csv" class="js-hidden" @change="importRoster" />
        <button class="js-btn js-btn-ghost" @click="downloadTemplate"><LucideDownload :size="13" /> Template Excel</button>
        <button class="js-btn js-btn-accent" @click="exportRoster"><LucideDownload :size="13" /> Ekspor Excel</button>
        <button class="js-btn js-btn-accent" @click="importInput?.click()"><LucidePlus :size="13" /> Import Excel</button>
      </div>

      <!-- ── Legenda ── -->
      <div class="js-legend">
        <span v-for="(meta, key) in SHIFT_META" :key="key" class="js-shift-chip" :style="{ background: meta.bg, color: meta.fg, borderColor: meta.fg + '66' }">{{ meta.label }}</span>
        <span class="js-hint">Klik sel untuk ganti shift</span>
      </div>

      <!-- ── Ringkasan ── -->
      <div class="js-summary">
        <b>Hari ini ({{ summaryDay }}):</b>
        <span>Total <b>{{ summary.total }}</b> Orang</span>
        <span>• <b class="js-pagi">Pagi: {{ summary.PAGI }}</b></span>
        <span>• <b class="js-malam">Malam: {{ summary.MALAM }}</b></span>
        <span>• <b class="js-off">OFF: {{ summary.OFF }}</b></span>
        <span>• CUTI: {{ summary.CUTI }}</span>
        <span>• ALFA: {{ summary.ALFA }}</span>
        <span>• SAKIT: {{ summary.SAKIT }}</span>
        <span>• IJIN: {{ summary.IJIN }}</span>
        <span class="js-muted">{{ divisiFilter || 'Semua Divisi' }}</span>
      </div>

      <!-- ── Grid roster ── -->
      <div class="js-grid-wrap">
        <table class="js-grid">
          <thead>
            <tr>
              <th class="js-col-name">NAMA KARYAWAN</th><th class="js-col-jab">JABATAN</th><th class="js-col-div">DIVISI</th>
              <th v-for="d in DAYS" :key="d" :class="{ 'js-today': d === summaryDay }" @click="summaryDay = d">
                {{ dayLabel(d) }} <b>{{ d }}</b>
              </th>
              <th class="js-col-act"></th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="r in visibleRows" :key="r.id">
              <td class="js-name">{{ r.Nama || '-' }}</td>
              <td><span class="js-badge">{{ r.Jabatan || '-' }}</span></td>
              <td><span class="js-badge js-badge-div">{{ r.Divisi || '-' }}</span></td>
              <td v-for="d in DAYS" :key="d" class="js-cell-td" :class="{ 'js-today': d === summaryDay }">
                <select
                  class="js-cell"
                  :style="cellStyle(r[String(d)])"
                  :value="r[String(d)] || ''"
                  @change="updateCell(r, d, ($event.target as HTMLSelectElement).value)"
                >
                  <option v-for="o in SHIFTS" :key="o" :value="o">{{ o || '—' }}</option>
                </select>
              </td>
              <td class="js-col-act"><button class="js-btn js-btn-danger js-btn-xs" @click="removeRow(r)">Hapus</button></td>
            </tr>
            <tr v-if="!visibleRows.length"><td :colspan="DAYS.length + 4" class="js-empty">Belum ada data roster bulan ini.</td></tr>
          </tbody>
        </table>
      </div>
      <button class="js-btn js-btn-primary js-add" @click="addOpen = true"><LucidePlus :size="13" /> Tambah Karyawan</button>
    </div>

    <!-- ── Modal tambah baris ── -->
    <a-modal v-model:open="addOpen" title="Tambah Karyawan ke Roster" ok-text="Tambah" cancel-text="Batal" width="460px" @ok="addRow">
      <div class="js-form">
        <div class="js-field"><label>Nama Karyawan *</label>
          <select v-model="newRow.Nama" class="js-input">
            <option value="">Pilih karyawan...</option>
            <option v-for="k in karyawanOptions" :key="k" :value="k">{{ k }}</option>
          </select>
        </div>
        <div class="js-field"><label>Jabatan (dari data Karyawan)</label><input v-model="newRow.Jabatan" class="js-input" readonly /></div>
        <div class="js-field"><label>Divisi (dari data Karyawan)</label><input v-model="newRow.Divisi" class="js-input" readonly /></div>
      </div>
    </a-modal>

    <!-- ── Modal setelan jam ── -->
    <a-modal v-model:open="settingsOpen" title="Setelan Jam & Shift" ok-text="Tutup" :footer="null" cancel-text="Batal" width="820px" @ok="settingsOpen = false">
      <FclCrud
        embedded
        title="Setelan Jam"
        table="jadwalshift"
        :columns="shiftColumns"
        :fields="shiftFields"
        :search-keys="['namaJadwal', 'divisi']"
        search-placeholder="Cari jadwal..."
        csv-name="jadwal-shift-setting.csv"
        form-title-create="Tambah Jadwal"
        form-title-edit="Edit Jadwal"
      />
    </a-modal>
  </Page>
</template>

<style scoped>
/* Jadwal Shift — token tema */
.js-root { display: flex; flex-direction: column; gap: 10px; }
.js-filter { display: flex; gap: 8px; align-items: end; flex-wrap: wrap; padding: 12px; background: hsl(var(--card)); border: 1px solid hsl(var(--border)); border-radius: 8px; }
.js-field { display: flex; flex-direction: column; gap: 4px; }
.js-field label { font-size: 10px; font-weight: 800; color: hsl(var(--muted-foreground)); letter-spacing: 0.04em; }
.js-input { height: 32px; padding: 2px 9px; border-radius: 6px; border: 1px solid hsl(var(--border)); background: hsl(var(--background)); color: hsl(var(--foreground)); font-size: 12px; outline: none; min-width: 150px; }
.js-input:focus { border-color: hsl(var(--primary)); }
.js-spacer { flex: 1; }
.js-hidden { display: none; }
.js-btn { display: inline-flex; align-items: center; justify-content: center; gap: 5px; border: 1px solid hsl(var(--border)); background: hsl(var(--muted)); color: hsl(var(--foreground)); font-size: 11px; font-weight: 700; padding: 6px 11px; border-radius: 6px; cursor: pointer; height: 32px; white-space: nowrap; }
.js-btn:hover { filter: brightness(0.96); }
.js-btn-xs { height: 24px; padding: 2px 8px; font-size: 10px; }
.js-btn-primary { background: hsl(var(--primary)); border-color: hsl(var(--primary)); color: hsl(var(--primary-foreground)); }
.js-btn-accent { background: hsl(var(--warning)); border-color: hsl(var(--warning)); color: hsl(var(--warning-foreground)); }
.js-btn-danger { background: hsl(var(--destructive)); border-color: hsl(var(--destructive)); color: hsl(var(--destructive-foreground)); }

.js-legend { display: flex; gap: 6px; align-items: center; flex-wrap: wrap; padding: 8px 12px; background: hsl(var(--card)); border: 1px solid hsl(var(--border)); border-radius: 8px; }
.js-shift-chip { font-size: 10px; font-weight: 800; padding: 3px 9px; border-radius: 999px; border: 1px solid; }
.js-hint { font-size: 10px; color: hsl(var(--muted-foreground)); margin-left: 8px; }
.js-summary { display: flex; gap: 8px; align-items: center; flex-wrap: wrap; padding: 8px 12px; background: hsl(var(--card)); border: 1px solid hsl(var(--border)); border-radius: 8px; font-size: 11px; color: hsl(var(--foreground)); }
.js-summary b { font-weight: 800; }
.js-pagi { color: hsl(var(--success)); }
.js-malam { color: hsl(198 90% 70%); }
.js-off { color: hsl(var(--destructive)); }
.js-muted { color: hsl(var(--muted-foreground)); margin-left: auto; }

.js-grid-wrap { overflow: auto; max-height: 620px; background: hsl(var(--card)); border: 1px solid hsl(var(--border)); border-radius: 8px; }
.js-grid { border-collapse: separate; border-spacing: 0; width: max-content; min-width: 100%; font-size: 11px; }
.js-grid th { position: sticky; top: 0; z-index: 2; background: hsl(var(--muted)); color: hsl(var(--muted-foreground)); font-size: 10px; font-weight: 800; padding: 7px 8px; border-bottom: 1px solid hsl(var(--border)); white-space: nowrap; cursor: pointer; }
.js-grid th b { color: hsl(var(--foreground)); }
.js-grid td { padding: 4px; border-bottom: 1px solid hsl(var(--border)); border-right: 1px solid hsl(var(--border)); }
.js-col-name { left: 0; z-index: 3 !important; min-width: 190px; }
.js-col-jab { left: 190px; z-index: 3 !important; min-width: 130px; }
.js-col-div { left: 320px; z-index: 3 !important; min-width: 100px; }
.js-col-act { min-width: 60px; }
.js-grid th.js-col-name, .js-grid th.js-col-jab, .js-grid th.js-col-div { position: sticky; z-index: 4 !important; }
.js-grid td.js-name { position: sticky; left: 0; z-index: 1; background: hsl(var(--card)); font-weight: 700; min-width: 190px; }
.js-grid td:nth-child(2) { position: sticky; left: 190px; z-index: 1; background: hsl(var(--card)); }
.js-grid td:nth-child(3) { position: sticky; left: 320px; z-index: 1; background: hsl(var(--card)); }
.js-today { box-shadow: inset 0 0 0 2px hsl(var(--primary) / 0.35); }
.js-badge { display: inline-block; font-size: 10px; font-weight: 700; padding: 2px 7px; border-radius: 999px; background: hsl(var(--primary) / 0.12); color: hsl(var(--primary)); white-space: nowrap; }
.js-badge-div { background: hsl(var(--muted)); color: hsl(var(--muted-foreground)); }
.js-cell-td { width: 92px; min-width: 92px; }
.js-cell { width: 100%; min-width: 84px; height: 46px; border: 1px solid transparent; border-radius: 4px; font-size: 9px; font-weight: 800; text-align: center; cursor: pointer; padding: 2px; appearance: none; }
.js-cell option { background: hsl(var(--card)); color: hsl(var(--foreground)); font-size: 11px; }
.js-empty { text-align: center; color: hsl(var(--muted-foreground)); padding: 24px 8px; }
.js-add { align-self: flex-start; }
.js-form { display: flex; flex-direction: column; gap: 10px; padding-top: 4px; }
</style>
