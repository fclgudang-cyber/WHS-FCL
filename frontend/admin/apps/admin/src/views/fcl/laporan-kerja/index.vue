<script lang="ts" setup>
import { computed, onMounted, reactive, ref } from 'vue';
import type { Dayjs } from 'dayjs';

import { Page } from '@vben/common-ui';
import { useUserStore } from '@vben/stores';
import { LucideDownload, LucidePlus, LucideRefreshCw } from '@vben/icons';

import { Modal, message, notification } from 'ant-design-vue';
import dayjs from 'dayjs';

import { fclDelete, fclInsert, fclPatch, fclQuery } from '#/api/fcl';

interface LaporanRow {
  id: string;
  tanggal?: string | null;
  divisi?: string | null;
  pic?: string | null;
  shift?: string | null;
  totalOrang?: number | string | null;
  perbantuan?: number | string | null;
  pengurangan?: number | string | null;
  jamLembur?: number | string | null;
  totalJamKerja?: number | string | null;
  kendala?: string | null;
  totalStaff?: number | string | null;
  totalAdmin?: number | string | null;
  totalOrder?: number | string | null;
  sisaOrder?: number | string | null;
  staffLemburNames?: string | null;
  totalPHL?: number | string | null;
  totalPO?: number | string | null;
  totalQty?: number | string | null;
  totalQtyInb?: number | string | null;
  totalInbound?: number | string | null;
  alasanPengurangan?: string | null;
  alasanLembur?: string | null;
  totalTeamLeader?: number | string | null;
  totalPelorporUtama?: number | string | null;
  jamPengurangan?: number | string | null;
  totalPekerjaLembur?: number | string | null;
  jamKerjaPHL?: number | string | null;
  noKartuTanpaKartu?: string | null;
  namaTanpaKartu?: string | null;
  autoHitung?: string | null;
  soreReload?: string | null;
  semuaMalamKartu?: string | null;
  semuaMemakaiSemuaKartu?: string | null;
  wpi?: number | string | null;
  totalKecukupanJamKerja?: number | string | null;
  totalKesesuaian?: number | string | null;
  createdBy?: string | null;
  createdAt?: string | null;
}

const userStore = useUserStore();
const namaAkun = () => {
  const u: any = userStore.userInfo ?? {};
  return String(u.realname || u.nama || u.username || u.nickname || '').trim();
};

const rows = ref<LaporanRow[]>([]);
const loading = ref(true);
const bulan = ref<Dayjs | null>(null);
const divisiFilter = ref<string>('Semua Divisi');
const search = ref('');

const num = (v: unknown) => {
  const n = Number(v ?? 0);
  return Number.isFinite(n) ? n : 0;
};
const fmtNum = (v: unknown, digits = 0) =>
  num(v).toLocaleString('id-ID', { minimumFractionDigits: digits, maximumFractionDigits: digits });
const tanggalDuaBaris = (v?: string | null): [string, string] => {
  const d = dayjs(v);
  return d.isValid() ? [d.format('DD MMM'), d.format('YYYY')] : ['-', ''];
};

// Daftar divisi diambil dari data Karyawan (master), dengan fallback dari laporan lama.
const divisiKaryawan = ref<string[]>([]);
const daftarDivisi = computed(() => {
  const set = new Set<string>(divisiKaryawan.value);
  rows.value.forEach((r) => {
    String(r.divisi ?? '')
      .split(/[,/]+/)
      .map((s) => s.trim())
      .filter(Boolean)
      .forEach((s) => set.add(s.toUpperCase()));
  });
  return ['Semua Divisi', ...[...set].sort((a, b) => a.localeCompare(b, 'id'))];
});
const divisiOptions = computed(() => daftarDivisi.value
  .filter((d) => d !== 'Semua Divisi')
  .map((d) => ({ label: d, value: d })));

const filtered = computed(() => {
  const q = search.value.trim().toLowerCase();
  return rows.value.filter((r) => {
    if (bulan.value && dayjs(r.tanggal).isValid() && !dayjs(r.tanggal).isSame(bulan.value, 'month')) return false;
    if (divisiFilter.value !== 'Semua Divisi' && !String(r.divisi ?? '').toLowerCase().includes(divisiFilter.value.toLowerCase())) return false;
    if (q && !`${r.divisi ?? ''} ${r.pic ?? ''} ${r.kendala ?? ''}`.toLowerCase().includes(q)) return false;
    return true;
  });
});

function alasanChips(r: LaporanRow): string[] {
  return [r.alasanLembur, r.alasanPengurangan, r.kendala]
    .flatMap((v) => String(v ?? '').split(/[,;]+/))
    .map((s) => s.trim())
    .filter((s) => !!s && s !== '-')
    .slice(0, 4);
}

async function load() {
  loading.value = true;
  try {
    const [data, karyawan] = await Promise.all([
      fclQuery<LaporanRow>('laporankerja', { select: '*', order: 'tanggal.desc', limit: 2000 }),
      fclQuery<{ divisi?: string | null }>('karyawan', { select: 'divisi', order: 'divisi.asc', limit: 2000 }),
    ]);
    rows.value = data;
    divisiKaryawan.value = [...new Set(karyawan
      .map((k) => String(k.divisi ?? '').trim())
      .filter(Boolean))]
      .sort((a, b) => a.localeCompare(b, 'id'));
  } catch (err: any) {
    notification.error({ message: 'Laporan Kerja', description: err?.message || 'Gagal memuat data' });
  } finally {
    loading.value = false;
  }
}
onMounted(load);

// ── form buat / edit ───────────────────────────────────────────────────────────
const formOpen = ref(false);
const editingId = ref<string | null>(null);
const saving = ref(false);
const form = reactive<Record<string, any>>({});

const formFields: any[] = [
  { name: 'tanggal', label: 'TANGGAL LAPORAN', type: 'date' },
  { name: 'shift', label: 'SHIFT', type: 'select', options: ['Shift Pagi', 'Shift Siang', 'Shift Malam'] },
  { name: 'divisi', label: 'DIVISI', placeholder: 'Marketplace...' },
  { name: 'pic', label: 'NAMA PELOPOR (PIC) · AKUN LOGIN', placeholder: 'Nama akun...' },
  { name: 'totalTeamLeader', label: 'TOTAL TEAM LEADER', type: 'number' },
  { name: 'totalStaff', label: 'TOTAL STAFF', type: 'number' },
  { name: 'totalPelorporUtama', label: 'TOTAL PELOPOR UTAMA', type: 'number' },
  { name: 'totalAdmin', label: 'TOTAL ADMIN', type: 'number' },
  { name: 'totalPHL', label: 'TOTAL PHL', type: 'number' },
  { name: 'jamKerjaPHL', label: 'JAM KERJA PHL (DESKRIPSI)', type: 'number' },
  { name: 'perbantuan', label: 'ORANG PERBANTUAN', type: 'number' },
  { name: 'pengurangan', label: 'PENGURANGAN JAM KERJA', type: 'number' },
  { name: 'jamPengurangan', label: 'JAM PENGURANGAN', type: 'number' },
  { name: 'totalPekerjaLembur', label: 'TOTAL PEKERJA LEMBUR', type: 'number' },
  { name: 'jamLembur', label: 'LAMA LEMBUR (JAM)', type: 'number' },
  { name: 'alasanLembur', label: 'LAMA LEMBUR (JAM), KETERANGAN' },
  { name: 'totalOrder', label: 'Total Order Marketplace', type: 'number' },
  { name: 'noKartuTanpaKartu', label: 'NO KARTU KARYAWAN YANG TIDAK MENGGUNAKAN KARTU', type: 'textarea' },
  { name: 'namaTanpaKartu', label: 'NAMA KARYAWAN YANG TIDAK MENGGUNAKAN KARTU', type: 'textarea' },
  { name: 'kendala', label: 'KETERANGAN', type: 'textarea' },
  { name: 'totalJamKerja', label: 'TOTAL JAM KERJA (MANUAL)', type: 'number' },
  { name: 'totalQty', label: 'QTY', type: 'number' },
  { name: 'sisaOrder', label: 'KPI', type: 'number' },
];

const fieldLabel = (name: string) => formFields.find((f) => f.name === name)?.label ?? name;
const groupAngka = ['totalTeamLeader', 'totalStaff', 'totalPelorporUtama', 'totalAdmin', 'totalPHL', 'jamKerjaPHL', 'perbantuan', 'pengurangan', 'jamPengurangan', 'totalPekerjaLembur', 'jamLembur'];
const groupOrder = ['totalOrder', 'totalQty', 'sisaOrder'];

// Ambil data PIC, shift, jumlah peran & lembur dari roster Jadwal Shift.
const syncingShift = ref(false);
async function syncFromJadwalShift() {
  if (!form.tanggal) { message.warning('Pilih tanggal laporan dulu'); return; }
  const tgl = dayjs(form.tanggal);
  if (!tgl.isValid()) return;
  const monthKey = tgl.format('YYYY-MM');
  const dayKey = String(tgl.date());
  syncingShift.value = true;
  try {
    const roster = await fclQuery<Record<string, any>>('jadwalroster', { select: '*', limit: 10000 });
    const divisi = String(form.divisi ?? '').trim().toLowerCase();
    const list = roster.filter((r) => String(r.Bulan ?? '') === monthKey
      && (!divisi || String(r.Divisi ?? '').toLowerCase().includes(divisi)));
    const shiftOf = (r: Record<string, any>) => String(r[dayKey] ?? '').toUpperCase().trim();
    const isOff = (s: string) => ['', 'OFF', 'LIBUR', 'CUTI', 'IJIN', 'SAKIT', 'ALFA'].includes(s);
    const aktif = list.filter((r) => !isOff(shiftOf(r)));
    const jml = (teks: string) => aktif.filter((r) => String(r.Jabatan ?? '').toLowerCase().includes(teks.toLowerCase())).length;
    // Orang lembur & jam lembur berasal dari sel LEMBUR yang sama di Jadwal Shift.
    const lemburRows = aktif.filter((r) => /LEMBUR/i.test(shiftOf(r)));
    const jamPerOrang = (s: string) => {
      const m = s.match(/LEMBUR\s*(\d+(?:[.,]\d+)?)\s*J/i);
      return m ? Number(m[1].replace(',', '.')) : 2;
    };
    const totalJamLembur = lemburRows.reduce((s, r) => s + jamPerOrang(shiftOf(r)), 0);

    const hitung: Record<string, number> = {};
    aktif.forEach((r) => {
      const s = /MALAM/i.test(shiftOf(r)) ? 'Shift Malam' : /PAGI/i.test(shiftOf(r)) ? 'Shift Pagi' : '';
      if (s) hitung[s] = (hitung[s] ?? 0) + 1;
    });
    const shiftDominan = Object.entries(hitung).sort((a, b) => b[1] - a[1])[0]?.[0] ?? '';
    // PIC memakai nama akun yang sedang login (bukan nama dari roster).
    const pic = namaAkun();

    if (shiftDominan) form.shift = shiftDominan;
    if (pic) form.pic = pic;
    form.totalTeamLeader = jml('team leader');
    form.totalStaff = jml('staff');
    form.totalPelorporUtama = jml('pelopor');
    form.totalAdmin = jml('admin');
    form.totalPHL = jml('phl');
    form.totalPekerjaLembur = lemburRows.length;
    form.jamLembur = totalJamLembur;
    form.totalOrang = aktif.length;
    notification.success({
      message: 'Sync Jadwal Shift',
      description: `${aktif.length} karyawan aktif dari ${list.length} data roster`,
    });
  } catch (err: any) {
    notification.error({ message: 'Sync Jadwal Shift', description: err?.message || 'Gagal memuat roster' });
  } finally {
    syncingShift.value = false;
  }
}

const autoHitung = ref(true);
const soreReload = ref(false);
const semuaMalamKartu = ref(false);
const semuaMemakaiSemuaKartu = ref(false);

const totalCrew = computed(() => ['totalTeamLeader', 'totalStaff', 'totalPelorporUtama', 'totalAdmin', 'totalPHL', 'totalPekerjaLembur']
  .reduce((s, k) => s + num(form[k]), 0));
const jamKerjaAuto = computed(() => {
  const crew = ['totalTeamLeader', 'totalStaff', 'totalPelorporUtama', 'totalAdmin']
    .reduce((s, k) => s + num(form[k]), 0);
  return crew * 1.5 + num(form.totalPHL) * num(form.jamKerjaPHL)
    + num(form.totalPekerjaLembur) * num(form.jamLembur)
    + num(form.perbantuan) - num(form.pengurangan);
});
const jamKerjaFinal = computed(() => (autoHitung.value ? jamKerjaAuto.value : num(form.totalJamKerja)));
const wpi = computed(() => (jamKerjaFinal.value > 0 ? num(form.totalOrder) / jamKerjaFinal.value : 0));
const totalKecukupan = computed(() => totalCrew.value * 16);
const totalKesesuaian = computed(() => Math.round((jamKerjaFinal.value - totalKecukupan.value) * 10) / 10);

function resetForm(row?: LaporanRow | null) {
  formFields.forEach((f) => {
    const v = row ? (row as any)[f.name] : undefined;
    form[f.name] = f.type === 'date' ? (v ? dayjs(String(v).slice(0, 10)) : null) : f.type === 'number' ? (v === undefined || v === null || v === '' ? undefined : Number(v)) : (v ?? '');
  });
  autoHitung.value = row ? String((row as any).autoHitung ?? 'Ya') !== 'Tidak' : true;
  soreReload.value = row ? String((row as any).soreReload ?? 'Tidak') === 'Ya' : false;
  semuaMalamKartu.value = row ? String((row as any).semuaMalamKartu ?? 'Tidak') === 'Ya' : false;
  semuaMemakaiSemuaKartu.value = row ? String((row as any).semuaMemakaiSemuaKartu ?? 'Tidak') === 'Ya' : false;
}

function openCreate() {
  editingId.value = null;
  resetForm();
  form.tanggal = dayjs();
  form.pic = namaAkun();
  formOpen.value = true;
  void syncFromJadwalShift();
}
function openEdit(r: LaporanRow) {
  editingId.value = r.id;
  resetForm(r);
  detailOpen.value = false;
  formOpen.value = true;
}

async function saveForm() {
  if (!form.tanggal) { message.warning('Tanggal wajib diisi'); return; }
  saving.value = true;
  try {
    const payload: Record<string, any> = {};
    formFields.forEach((f) => {
      let v = form[f.name];
      if (f.type === 'date') v = v && dayjs(v).isValid() ? dayjs(v).format('YYYY-MM-DD') : '';
      if (v === undefined || v === null || v === '') return;
      payload[f.name] = v;
    });
    // Nilai turunan & opsi form ikut tersimpan.
    payload.totalJamKerja = String(Math.round(jamKerjaFinal.value * 10) / 10);
    payload.totalOrang = String(totalCrew.value);
    payload.wpi = String(Math.round(wpi.value * 1000) / 1000);
    payload.totalKecukupanJamKerja = String(totalKecukupan.value);
    payload.totalKesesuaian = String(totalKesesuaian.value);
    payload.autoHitung = autoHitung.value ? 'Ya' : 'Tidak';
    payload.soreReload = soreReload.value ? 'Ya' : 'Tidak';
    payload.semuaMalamKartu = semuaMalamKartu.value ? 'Ya' : 'Tidak';
    payload.semuaMemakaiSemuaKartu = semuaMemakaiSemuaKartu.value ? 'Ya' : 'Tidak';
    if (editingId.value) {
      await fclPatch('laporankerja', editingId.value, payload);
      notification.success({ message: 'Laporan Kerja diperbarui' });
    } else {
      await fclInsert('laporankerja', [payload]);
      notification.success({ message: 'Laporan Kerja dibuat' });
    }
    formOpen.value = false;
    await load();
  } catch (err: any) {
    notification.error({ message: 'Laporan Kerja', description: err?.message || 'Gagal menyimpan' });
  } finally {
    saving.value = false;
  }
}

function hapus(r: LaporanRow) {
  Modal.confirm({
    title: 'Hapus laporan?',
    content: `Hapus laporan tanggal ${String(r.tanggal ?? '').slice(0, 10)}?`,
    okText: 'Hapus',
    okType: 'danger',
    async onOk() {
      await fclDelete('laporankerja', r.id);
      notification.success({ message: 'Laporan dihapus' });
      detailOpen.value = false;
      await load();
    },
  });
}

// ── detail harian ──────────────────────────────────────────────────────────────
const detailOpen = ref(false);
const detailRow = ref<LaporanRow | null>(null);
function bukaDetail(r: LaporanRow) {
  detailRow.value = r;
  detailOpen.value = true;
}

// ── set KPI ────────────────────────────────────────────────────────────────────
const kpiOpen = ref(false);
const kpiValues = reactive<Record<string, number>>({});
function openKpi() {
  Object.keys(kpiValues).forEach((k) => delete kpiValues[k]);
  filtered.value.forEach((r) => { kpiValues[r.id] = num(r.sisaOrder); });
  kpiOpen.value = true;
}
async function saveKpi() {
  saving.value = true;
  try {
    for (const r of filtered.value) {
      if (num(kpiValues[r.id]) !== num(r.sisaOrder)) {
        await fclPatch('laporankerja', r.id, { sisaOrder: String(num(kpiValues[r.id])) });
      }
    }
    notification.success({ message: 'KPI tersimpan' });
    kpiOpen.value = false;
    await load();
  } catch (err: any) {
    notification.error({ message: 'Set KPI', description: err?.message || 'Gagal menyimpan' });
  } finally {
    saving.value = false;
  }
}

// ── export Excel ───────────────────────────────────────────────────────────────
function exportExcel() {
  if (!filtered.value.length) { message.warning('Tidak ada data untuk diekspor'); return; }
  const header = ['Tanggal', 'Divisi', 'PIC', 'Jam Kerja', 'Lembur', 'Alasan', 'PHL', 'Staff', 'Total Masuk', 'Output', 'QTY', 'KPI'];
  const esc = (s: unknown) => String(s ?? '').replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;');
  const head = header.map((h) => `<th>${h}</th>`).join('');
  const body = filtered.value.map((r) => {
    const [d1, d2] = tanggalDuaBaris(r.tanggal);
    return `<tr>
      <td>${esc(d1)} ${esc(d2)}</td><td>${esc(r.divisi)}</td><td>${esc(r.pic)}</td>
      <td>${fmtNum(r.totalJamKerja, 1)}</td><td>${fmtNum(r.jamLembur, 1)}</td>
      <td>${esc(alasanChips(r).join(', '))}</td><td>${fmtNum(r.totalPHL)}</td>
      <td>${fmtNum(r.totalStaff)}</td><td>${fmtNum(r.totalOrang)}</td>
      <td>${fmtNum(r.totalOrder)}</td><td>${fmtNum(r.totalQty)}</td><td>${fmtNum(r.sisaOrder)}</td></tr>`;
  }).join('');
  const html = `<!doctype html><html><head><meta charset="utf-8"><style>
    table{border-collapse:collapse;width:100%;font-family:Calibri,Arial;font-size:11px}
    th,td{border:1px solid #bfbfbf;padding:4px 6px}
    th{background:#f2f2f2;font-weight:bold}
  </style></head><body><table><tr>${head}</tr>${body}</table></body></html>`;
  const blob = new Blob(['\uFEFF' + html], { type: 'application/vnd.ms-excel;charset=utf-8;' });
  const url = URL.createObjectURL(blob);
  const a = document.createElement('a');
  a.href = url;
  a.download = 'laporan-kerja.xls';
  document.body.appendChild(a);
  a.click();
  document.body.removeChild(a);
  URL.revokeObjectURL(url);
}
</script>

<template>
  <Page auto-content-height>
    <div class="lk-page">
      <div class="lk-head">
        <div class="lk-title">Riwayat Laporan Kerja (Harian)</div>
        <div class="lk-tools">
          <a-date-picker v-model:value="bulan" picker="month" placeholder="Semua Bulan" class="lk-month" />
          <a-select v-model:value="divisiFilter" :options="daftarDivisi.map((d) => ({ label: d, value: d }))" class="lk-divisi" />
          <input v-model="search" type="text" class="lk-search" placeholder="Cari..." />
          <button class="lk-btn lk-btn-orange" @click="exportExcel"><LucideDownload :size="13" /> Export</button>
          <button class="lk-btn lk-btn-orange" @click="openKpi">Set KPI</button>
          <button class="lk-btn lk-btn-orange" @click="openCreate"><LucidePlus :size="13" /> Buat</button>
        </div>
      </div>

      <div class="lk-table">
        <div class="lk-row lk-row-head">
          <div>TANGGAL</div><div>DIVISI/PIC</div><div>JAM KERJA</div><div>LEMBUR</div>
          <div>ALASAN</div><div>PHL</div><div>STAFF</div><div>TOTAL MASUK</div>
          <div>OUTPUT</div><div>QTY</div><div>KPI</div><div>AKSI</div>
        </div>
        <div v-if="loading" class="lk-empty">Memuat data...</div>
        <div v-else-if="!filtered.length" class="lk-empty">Belum ada laporan kerja.</div>
        <div
          v-for="(r, i) in filtered"
          :key="r.id"
          class="lk-row"
          :class="{ 'lk-row-alt': i % 2 === 1 }"
        >
          <div class="lk-tanggal"><b>{{ tanggalDuaBaris(r.tanggal)[0] }}</b><span>{{ tanggalDuaBaris(r.tanggal)[1] }}</span></div>
          <div class="lk-divisi-pic">
            <b>{{ (r.divisi || '-').split(/[,/]+/).map((s) => s.trim().toUpperCase()).filter(Boolean).join(', ') }}</b>
            <span>{{ r.pic || '-' }}</span>
          </div>
          <div class="lk-num">{{ fmtNum(r.totalJamKerja, 1) }}</div>
          <div class="lk-lembur"><b>{{ fmtNum(r.jamLembur, 1) }}</b><span>({{ fmtNum(r.perbantuan) }} orang)</span></div>
          <div class="lk-alasan">
            <span v-for="(a, ai) in alasanChips(r)" :key="ai" class="lk-chip">{{ a }}</span>
            <span v-if="!alasanChips(r).length">-</span>
          </div>
          <div class="lk-num"><b>{{ fmtNum(r.totalPHL) }}</b><span>Org</span></div>
          <div class="lk-num"><b>{{ fmtNum(r.totalStaff) }}</b><span>Org</span></div>
          <div class="lk-num lk-hijau"><b>{{ fmtNum(r.totalOrang) }}</b><span>Org</span></div>
          <div class="lk-num"><b>{{ fmtNum(r.totalOrder) }}</b><span>PO {{ fmtNum(r.totalPO) }}</span></div>
          <div class="lk-num"><b>{{ fmtNum(r.totalQty) }}</b></div>
          <div><span class="lk-kpi">{{ fmtNum(r.sisaOrder) }}</span></div>
          <div><button class="lk-detail" @click="bukaDetail(r)">Detail Harian</button></div>
        </div>
      </div>
    </div>

    <!-- Modal: form buat/edit -->
    <a-modal
      v-model:open="formOpen"
      title="Input Laporan Pengerjaan Orderan"
      :confirm-loading="saving"
      width="720px"
      ok-text="Simpan"
      cancel-text="Batal"
      @ok="saveForm"
    >
      <div class="lk-input-form">
        <div class="lk-sync-bar">
          <button type="button" class="lk-btn lk-btn-ghost" :disabled="syncingShift" @click="syncFromJadwalShift">
            <LucideRefreshCw :size="13" /> {{ syncingShift ? 'Sinkron...' : 'Sync dari Jadwal Shift' }}
          </button>
          <span>PIC memakai nama akun login; shift, jumlah peran &amp; lembur diambil dari roster bulan ini.</span>
        </div>
        <div class="lk-grid2">
          <label class="lk-f"><span>TANGGAL LAPORAN</span>
            <a-date-picker v-model:value="form.tanggal" value-format="YYYY-MM-DD" style="width: 100%" />
          </label>
          <label class="lk-f"><span>SHIFT</span>
            <a-select v-model:value="form.shift" :options="[{ label: 'Shift Pagi', value: 'Shift Pagi' }, { label: 'Shift Siang', value: 'Shift Siang' }, { label: 'Shift Malam', value: 'Shift Malam' }]" style="width: 100%" allow-clear />
          </label>
          <label class="lk-f"><span>DIVISI</span>
            <a-select
              v-model:value="form.divisi"
              :options="divisiOptions"
              placeholder="Pilih divisi..."
              show-search
              allow-clear
              option-filter-prop="label"
              style="width: 100%"
            />
          </label>
          <label class="lk-f"><span>NAMA PELOPOR (PIC) · AKUN LOGIN</span>
            <a-input v-model:value="form.pic" placeholder="M Mixed Nursi..." />
          </label>
        </div>

        <div class="lk-checks">
          <label><input type="checkbox" v-model="autoHitung" /> Auto-hitung dari jumlah PIC &amp; Lembur</label>
          <label><input type="checkbox" v-model="soreReload" /> Sore/Reload</label>
          <span class="lk-lock-hint">TOTAL TEAM LEADER / TOTAL STAFF / TOTAL ADMIN terkunci mengikuti Jadwal Shift — field lain bisa diedit manual.</span>
        </div>

        <div class="lk-info">
          (Shift Pagi = 1,5 jam + 1x istirahat + 1 jam lembur. Sementara 1 jam = 1 orang + 1 jam + 1 orang +
          dst.) Auto: jam kerja = {{ fmtNum(jamKerjaAuto, 1) }} jam. Total crew = {{ fmtNum(totalCrew) }} orang.
          Perhitungan (non manual) = {{ fmtNum(totalCrew) }} orang x 16 jam = {{ fmtNum(totalKecukupan) }} orang jam.
        </div>

        <div class="lk-grid2">
          <label v-for="name in groupAngka" :key="name" class="lk-f">
            <span>{{ fieldLabel(name) }} <i v-if="['totalTeamLeader', 'totalStaff', 'totalAdmin'].includes(name)">🔒</i></span>
            <a-input-number
              v-model:value="form[name]"
              :min="0"
              :disabled="['totalTeamLeader', 'totalStaff', 'totalAdmin'].includes(name)"
              style="width: 100%"
            />
          </label>
        </div>

        <div class="lk-warning" v-if="num(form.jamLembur) > 10">
          ⚠ Jam lembur melebihi batas normal (maksimal 10 jam)
        </div>

        <div class="lk-grid2">
          <label v-for="name in groupOrder" :key="name" class="lk-f">
            <span>{{ fieldLabel(name) }}</span>
            <a-input-number v-model:value="form[name]" :min="0" style="width: 100%" />
          </label>
        </div>

        <div class="lk-checks lk-checks-kartu">
          <label><input type="checkbox" v-model="semuaMalamKartu" /> Semua malam Kartu</label>
          <label><input type="checkbox" v-model="semuaMemakaiSemuaKartu" /> Semua memakai Semua Kartu</label>
        </div>

        <div class="lk-grid2">
          <label class="lk-f lk-full"><span>NO KARTU KARYAWAN YANG TIDAK MENGGUNAKAN KARTU</span>
            <a-textarea v-model:value="form.noKartuTanpaKartu" :rows="2" placeholder="Posisi dengan format..." />
          </label>
          <label class="lk-f lk-full"><span>NAMA KARYAWAN YANG TIDAK MENGGUNAKAN KARTU</span>
            <a-textarea v-model:value="form.namaTanpaKartu" :rows="2" placeholder="Posisi dengan format..." />
          </label>
          <label class="lk-f lk-full"><span>KETERANGAN</span>
            <a-textarea v-model:value="form.kendala" :rows="2" />
          </label>
        </div>

        <div class="lk-summary">
          <div><span>WPI (total Order / total jam Kerja)</span><b>{{ fmtNum(wpi, 3) }}</b></div>
          <div><span>Total Kecukupan Jam Kerja</span><b class="lk-merah">{{ fmtNum(totalKecukupan) }}</b></div>
          <div><span>TOTAL KESESUAIAN</span><b>{{ fmtNum(totalKesesuaian, 1) }}</b></div>
        </div>
        <div class="lk-catatan">* Jam Kerja Normal {{ fmtNum(totalCrew) }} orang ({{ fmtNum(jamKerjaFinal, 1) }} jam) &nbsp;•&nbsp; Total jam kerja dihitung {{ autoHitung ? 'otomatis' : 'manual' }}</div>
      </div>
    </a-modal>

    <!-- Modal: detail harian -->
    <a-modal v-model:open="detailOpen" title="Detail Laporan Kerja" :footer="null" width="640px">
      <div v-if="detailRow" class="lk-detail-grid">
        <div v-for="f in formFields" :key="f.name" class="lk-detail-item">
          <span class="lk-detail-label">{{ f.label }}</span>
          <span class="lk-detail-value">{{ f.type === 'number' ? fmtNum(detailRow[f.name as keyof LaporanRow], f.name === 'totalJamKerja' || f.name === 'jamLembur' ? 1 : 0) : (detailRow[f.name as keyof LaporanRow] || '-') }}</span>
        </div>
      </div>
      <div class="lk-detail-actions">
        <button class="lk-btn lk-btn-orange" @click="openEdit(detailRow!)">Edit</button>
        <button class="lk-btn lk-btn-red" @click="hapus(detailRow!)">Hapus</button>
      </div>
    </a-modal>

    <!-- Modal: set KPI -->
    <a-modal v-model:open="kpiOpen" title="Set KPI" :confirm-loading="saving" width="560px" ok-text="Simpan" cancel-text="Batal" @ok="saveKpi">
      <div class="lk-kpi-list">
        <div v-for="r in filtered" :key="r.id" class="lk-kpi-row">
          <span>{{ String(r.tanggal || '').slice(0, 10) }} · {{ r.divisi || '-' }}</span>
          <a-input-number v-model:value="kpiValues[r.id]" :min="0" style="width: 140px" />
        </div>
        <div v-if="!filtered.length" class="lk-empty">Tidak ada data.</div>
      </div>
    </a-modal>
  </Page>
</template>

<style scoped>
.lk-page { display: flex; flex-direction: column; gap: 12px; }
.lk-head { display: flex; align-items: center; justify-content: space-between; gap: 12px; flex-wrap: wrap; padding: 12px 14px; background: hsl(var(--card)); border: 1px solid hsl(var(--border)); border-radius: 8px; }
.lk-title { font-size: 14px; font-weight: 800; color: hsl(var(--card-foreground)); }
.lk-tools { display: flex; align-items: center; gap: 8px; flex-wrap: wrap; }
.lk-month { width: 170px; }
.lk-divisi { width: 180px; }
.lk-search { width: 160px; height: 32px; padding: 0 10px; border: 1px solid hsl(var(--border)); border-radius: 6px; background: transparent; color: inherit; font-size: 12px; }
.lk-btn { display: inline-flex; align-items: center; gap: 5px; border: 0; border-radius: 5px; padding: 6px 12px; color: #fff; font-size: 11px; font-weight: 800; cursor: pointer; }
.lk-btn-orange { background: #f59e0b; }
.lk-btn-red { background: #dc2626; }
.lk-table { border: 1px solid hsl(var(--border)); border-radius: 8px; overflow: hidden; background: hsl(var(--card)); }
.lk-row { display: grid; grid-template-columns: 62px minmax(200px, 2fr) 82px 88px minmax(150px, 1.4fr) 58px 58px 80px 88px 88px 58px 100px; gap: 8px; align-items: center; padding: 10px 12px; border-bottom: 1px solid hsl(var(--border)); font-size: 11px; color: hsl(var(--card-foreground)); }
.lk-row-alt { background: hsl(var(--muted) / 0.45); }
.lk-row-head { background: hsl(var(--muted)); font-size: 10px; font-weight: 800; letter-spacing: .04em; color: hsl(var(--muted-foreground)); }
.lk-row > div { min-width: 0; }
.lk-tanggal, .lk-num, .lk-lembur { display: flex; flex-direction: column; line-height: 1.35; }
.lk-tanggal b, .lk-num b, .lk-lembur b { font-weight: 800; }
.lk-tanggal span, .lk-num span, .lk-lembur span { font-size: 10px; color: hsl(var(--muted-foreground)); }
.lk-divisi-pic { display: flex; flex-direction: column; gap: 2px; }
.lk-divisi-pic b { font-weight: 800; }
.lk-divisi-pic span { font-size: 10px; color: hsl(var(--muted-foreground)); white-space: normal; }
.lk-hijau b { color: #16a34a; }
.lk-lembur span { color: #f97316; }
.lk-alasan { display: flex; flex-wrap: wrap; gap: 4px; }
.lk-chip { border: 1px solid hsl(var(--border)); border-radius: 999px; padding: 2px 8px; font-size: 10px; font-weight: 700; background: hsl(var(--muted)); white-space: nowrap; }
.lk-kpi { border: 1px solid hsl(var(--border)); border-radius: 999px; padding: 3px 10px; font-size: 11px; font-weight: 800; }
.lk-detail { border: 0; border-radius: 5px; padding: 6px 10px; background: hsl(var(--muted)); color: hsl(var(--foreground)); font-size: 11px; font-weight: 800; cursor: pointer; }
.lk-empty { padding: 24px; text-align: center; color: hsl(var(--muted-foreground)); font-size: 12px; }
.lk-form :deep(.ant-form-item) { margin-bottom: 10px; }
.lk-detail-grid { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 10px; }
.lk-detail-item { display: flex; flex-direction: column; gap: 2px; padding: 8px; border: 1px solid hsl(var(--border)); border-radius: 6px; }
.lk-detail-label { font-size: 10px; font-weight: 800; color: hsl(var(--muted-foreground)); }
.lk-detail-value { font-size: 12px; }
.lk-detail-actions { display: flex; justify-content: flex-end; gap: 8px; margin-top: 14px; }
.lk-kpi-list { max-height: 420px; overflow: auto; }
.lk-kpi-row { display: flex; align-items: center; justify-content: space-between; gap: 12px; padding: 7px 0; border-bottom: 1px solid hsl(var(--border)); font-size: 12px; }
.lk-input-form { display: flex; flex-direction: column; gap: 12px; }
.lk-sync-bar { display: flex; align-items: center; gap: 12px; flex-wrap: wrap; padding: 9px 11px; border: 1px solid hsl(var(--border)); border-radius: 6px; background: hsl(var(--muted)); }
.lk-sync-bar > span { font-size: 10px; color: hsl(var(--muted-foreground)); }
.lk-btn-ghost { background: hsl(var(--muted)); color: hsl(var(--foreground)); border: 1px solid hsl(var(--border)); }
.lk-grid2 { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 10px 14px; }
.lk-f { display: flex; flex-direction: column; gap: 4px; min-width: 0; }
.lk-f > span { font-size: 10px; font-weight: 800; letter-spacing: .03em; color: hsl(var(--muted-foreground)); text-transform: uppercase; }
.lk-f > span i { font-style: normal; font-size: 9px; opacity: .7; }
.lk-full { grid-column: 1 / -1; }
.lk-checks { display: flex; gap: 20px; flex-wrap: wrap; padding: 6px 0; font-size: 12px; }
.lk-lock-hint { font-size: 10px; color: hsl(var(--muted-foreground)); }
.lk-checks label { display: inline-flex; align-items: center; gap: 6px; cursor: pointer; }
.lk-info { padding: 10px 12px; border: 1px solid hsl(var(--border)); border-radius: 6px; background: hsl(var(--muted)); font-size: 10px; line-height: 1.5; color: hsl(var(--muted-foreground)); }
.lk-warning { padding: 8px 10px; border: 1px solid #f59e0b; border-radius: 6px; color: #f59e0b; font-size: 11px; font-weight: 700; }
.lk-checks-kartu { border-top: 1px solid hsl(var(--border)); padding-top: 10px; }
.lk-summary { display: flex; flex-direction: column; gap: 6px; padding: 10px 12px; border: 1px solid hsl(var(--border)); border-radius: 6px; }
.lk-summary > div { display: flex; align-items: center; justify-content: space-between; gap: 12px; font-size: 11px; }
.lk-summary b { font-size: 13px; }
.lk-merah { color: #dc2626; }
.lk-catatan { font-size: 10px; color: #16a34a; }
@media (max-width: 1400px) {
  .lk-row { grid-template-columns: 58px minmax(160px, 2fr) 70px 76px minmax(120px, 1.2fr) 48px 48px 70px 76px 76px 48px 92px; }
}
</style>
