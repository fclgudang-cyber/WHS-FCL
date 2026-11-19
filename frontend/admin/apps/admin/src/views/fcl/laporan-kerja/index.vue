<script lang="ts" setup>
import { computed, onMounted, reactive, ref } from 'vue';
import type { Dayjs } from 'dayjs';

import { Page } from '@vben/common-ui';
import { LucideDownload, LucidePlus } from '@vben/icons';

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
  createdBy?: string | null;
  createdAt?: string | null;
}

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

const daftarDivisi = computed(() => {
  const set = new Set<string>();
  rows.value.forEach((r) => {
    String(r.divisi ?? '')
      .split(/[,/]+/)
      .map((s) => s.trim())
      .filter(Boolean)
      .forEach((s) => set.add(s.toUpperCase()));
  });
  return ['Semua Divisi', ...[...set].sort()];
});

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
    rows.value = await fclQuery<LaporanRow>('laporankerja', { select: '*', order: 'tanggal.desc', limit: 2000 });
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
  { name: 'tanggal', label: 'Tanggal', type: 'date' },
  { name: 'divisi', label: 'Divisi', placeholder: 'Inbound, Marketplace, Consumable...' },
  { name: 'pic', label: 'PIC', placeholder: 'Nama-nama PIC' },
  { name: 'shift', label: 'Shift', type: 'select', options: ['Pagi', 'Siang', 'Malam'] },
  { name: 'totalJamKerja', label: 'Total Jam Kerja', type: 'number' },
  { name: 'jamLembur', label: 'Jam Lembur', type: 'number' },
  { name: 'perbantuan', label: 'Perbantuan (orang)', type: 'number' },
  { name: 'totalPHL', label: 'Total PHL', type: 'number' },
  { name: 'totalStaff', label: 'Total Staff', type: 'number' },
  { name: 'totalOrang', label: 'Total Masuk (orang)', type: 'number' },
  { name: 'totalOrder', label: 'Output', type: 'number' },
  { name: 'totalQty', label: 'QTY', type: 'number' },
  { name: 'sisaOrder', label: 'KPI', type: 'number' },
  { name: 'alasanLembur', label: 'Alasan Lembur' },
  { name: 'alasanPengurangan', label: 'Alasan Pengurangan' },
  { name: 'kendala', label: 'Kendala', type: 'textarea' },
];

function resetForm(row?: LaporanRow | null) {
  formFields.forEach((f) => {
    const v = row ? (row as any)[f.name] : undefined;
    form[f.name] = f.type === 'date' ? (v ? dayjs(String(v).slice(0, 10)) : null) : f.type === 'number' ? (v === undefined || v === null || v === '' ? undefined : Number(v)) : (v ?? '');
  });
}

function openCreate() {
  editingId.value = null;
  resetForm();
  form.tanggal = dayjs();
  formOpen.value = true;
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
      :title="editingId ? 'Edit Laporan Kerja' : 'Buat Laporan Kerja'"
      :confirm-loading="saving"
      width="640px"
      ok-text="Simpan"
      cancel-text="Batal"
      @ok="saveForm"
    >
      <a-form layout="vertical" class="lk-form">
        <a-form-item v-for="f in formFields" :key="f.name" :label="f.label">
          <a-date-picker v-if="f.type === 'date'" v-model:value="form[f.name]" value-format="YYYY-MM-DD" style="width: 100%" />
          <a-input-number v-else-if="f.type === 'number'" v-model:value="form[f.name]" :min="0" style="width: 100%" />
          <a-select v-else-if="f.type === 'select'" v-model:value="form[f.name]" :options="f.options?.map((o: string) => ({ label: o, value: o }))" allow-clear />
          <a-textarea v-else-if="f.type === 'textarea'" v-model:value="form[f.name]" :rows="2" />
          <a-input v-else v-model:value="form[f.name]" :placeholder="f.placeholder" />
        </a-form-item>
      </a-form>
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
@media (max-width: 1400px) {
  .lk-row { grid-template-columns: 58px minmax(160px, 2fr) 70px 76px minmax(120px, 1.2fr) 48px 48px 70px 76px 76px 48px 92px; }
}
</style>
