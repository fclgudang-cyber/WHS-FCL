<script lang="ts" setup>
/**
 * Asset Warehouse (FCL) — layout sesuai referensi:
 * header aksi (Sinkron/Import/Export/Laporan/Print/Tambah Asset), tab Daftar Asset / Stock Opname /
 * Hasil Stock Opname, 6 kartu statistik, filter (search + divisi/kategori/status + chip status),
 * tabel group per kategori dengan baris detail aset. Tema memakai token vben hsl(var(--...)).
 */
import { computed, onMounted, onUnmounted, ref } from 'vue';

import { Page } from '@vben/common-ui';
import {
  LucideArrowLeftRight,
  LucideChevronDown,
  LucideDownload,
  LucideFilePenLine,
  LucidePencil,
  LucidePlus,
  LucidePrinter,
  LucideRefreshCw,
  LucideTrash2,
} from '@vben/icons';

import { Modal, message, notification } from 'ant-design-vue';
import dayjs from 'dayjs';

import { fclDelete, fclInsert, fclPatch, fclQuery } from '#/api/fcl';

interface AssetRow {
  id: string;
  kategori?: string;
  code?: string;
  nama?: string;
  divisi?: string;
  zoneId?: string;
  qty?: string | number;
  status?: string;
  tanggalMasuk?: string;
  createdBy?: string;
  createdAt?: string;
}
interface OpnameSession {
  id: string;
  tanggal?: string;
  divisi?: string;
  status?: string;
  totalAsset?: string | number;
  terscan?: string | number;
  createdBy?: string;
}
interface OpnameLog {
  id: string;
  sessionId?: string;
  assetId?: string;
  assetCode?: string;
  assetNama?: string;
  divisi?: string;
  qtyFisik?: string | number;
  qtySistem?: string | number;
  kondisi?: string;
  catatan?: string;
  scannedBy?: string;
  scannedAt?: string;
}

const loading = ref(true);
const activeTab = ref<'list' | 'opname' | 'hasil'>('list');
const assets = ref<AssetRow[]>([]);
const sessions = ref<OpnameSession[]>([]);
const opnameLogs = ref<OpnameLog[]>([]);

const search = ref('');
const divisiFilter = ref('');
const kategoriFilter = ref('');
const statusFilter = ref('');
const statusChip = ref<'semua' | 'perbaikan' | 'rusak'>('semua');
const expanded = ref<Set<string>>(new Set());

function num(v: unknown): number {
  const n = Number(v);
  return Number.isFinite(n) ? n : 0;
}

function norm(s: unknown): string {
  return String(s ?? '').trim().toLowerCase();
}

function qtyOf(a: AssetRow): number {
  return num(a.qty) > 0 ? num(a.qty) : 1;
}

const stats = computed(() => {
  const data = assets.value;
  return {
    total: data.reduce((s, a) => s + qtyOf(a), 0),
    mp: data.filter((a) => norm(a.divisi) === 'marketplace').reduce((s, a) => s + qtyOf(a), 0),
    dist: data.filter((a) => norm(a.divisi) === 'distributor').reduce((s, a) => s + qtyOf(a), 0),
    kol: data.filter((a) => norm(a.divisi) === 'kol').reduce((s, a) => s + qtyOf(a), 0),
    aktif: data.filter((a) => norm(a.status) === 'aktif').length,
    rusak: data.filter((a) => norm(a.status).includes('rusak')).length,
    perbaikan: data.filter((a) => norm(a.status).includes('perbaikan')).length,
  };
});

const divisiOptions = computed(() => [...new Set(assets.value.map((a) => String(a.divisi ?? '').trim()).filter(Boolean))].sort());
const kategoriOptions = computed(() => [...new Set(assets.value.map((a) => String(a.kategori ?? '').trim()).filter(Boolean))].sort());
const statusOptions = computed(() => [...new Set(assets.value.map((a) => String(a.status ?? '').trim()).filter(Boolean))].sort());

const filteredAssets = computed(() => {
  const q = norm(search.value);
  return assets.value.filter((a) => {
    if (q && ![a.nama, a.code, a.kategori].some((v) => norm(v).includes(q))) return false;
    if (divisiFilter.value && String(a.divisi ?? '') !== divisiFilter.value) return false;
    if (kategoriFilter.value && String(a.kategori ?? '') !== kategoriFilter.value) return false;
    if (statusFilter.value && String(a.status ?? '') !== statusFilter.value) return false;
    if (statusChip.value !== 'semua') {
      const s = norm(a.status);
      if (statusChip.value === 'rusak' && !s.includes('rusak')) return false;
      if (statusChip.value === 'perbaikan' && !s.includes('perbaikan')) return false;
    }
    return true;
  });
});

interface AssetGroup {
  key: string;
  identity: string;
  kategori: string;
  items: AssetRow[];
  total: number;
  divisi: string[];
}

const groups = computed<AssetGroup[]>(() => {
  const map = new Map<string, AssetGroup>();
  for (const a of filteredAssets.value) {
    const prefix = String(a.code ?? '').split('-')[0].trim();
    const key = String(a.kategori ?? '').trim() || prefix || a.nama || '-';
    if (!map.has(key)) map.set(key, { key, identity: key, kategori: '', items: [], total: 0, divisi: [] });
    const g = map.get(key)!;
    g.items.push(a);
    g.total += qtyOf(a);
    const d = String(a.divisi ?? '').trim();
    if (d && !g.divisi.includes(d)) g.divisi.push(d);
    const k = String(a.kategori ?? '').trim();
    if (!g.kategori) g.kategori = k;
    else if (g.kategori !== k) g.kategori = 'Beragam';
  }
  return [...map.values()].sort((a, b) => a.identity.localeCompare(b.identity));
});

function toggleExpand(key: string) {
  const next = new Set(expanded.value);
  if (next.has(key)) next.delete(key);
  else next.add(key);
  expanded.value = next;
}

function statusColor(s?: string): string {
  const v = norm(s);
  if (v.includes('rusak')) return 'hsl(var(--destructive))';
  if (v.includes('perbaikan')) return 'hsl(var(--warning))';
  return 'hsl(var(--success))';
}

async function loadAll(silent = false) {
  if (!silent) loading.value = true;
  try {
    const [a, s, l] = await Promise.all([
      fclQuery<AssetRow>('assetwarehouse', { select: '*', order: 'createdAt.desc', limit: 10000 }),
      fclQuery<OpnameSession>('assetopnamesession', { select: '*', order: 'createdAt.desc', limit: 1000 }),
      fclQuery<OpnameLog>('assetopnamelog', { select: '*', order: 'scannedAt.desc', limit: 10000 }),
    ]);
    assets.value = a;
    sessions.value = s;
    opnameLogs.value = l;
  } catch (err: any) {
    console.error('[fcl/asset-warehouse] load gagal', err);
    notification.error({ message: 'Asset Warehouse', description: err?.message || 'Gagal memuat data' });
  } finally {
    if (!silent) loading.value = false;
  }
}

// ── Modal Tambah/Edit asset ──────────────────────────────────────────────────
const assetModalOpen = ref(false);
const assetSaving = ref(false);
const editingAsset = ref<AssetRow | null>(null);
const form = ref({ kategori: '', code: '', nama: '', divisi: '', zoneId: '', qty: 1 as number | undefined, status: 'Aktif', tanggalMasuk: dayjs().format('YYYY-MM-DD') });

function openAssetModal(a?: AssetRow | null) {
  editingAsset.value = a ?? null;
  form.value = a
    ? {
        kategori: a.kategori ?? '', code: a.code ?? '', nama: a.nama ?? '', divisi: a.divisi ?? '',
        zoneId: a.zoneId ?? '', qty: num(a.qty), status: a.status ?? 'Aktif', tanggalMasuk: (a.tanggalMasuk ?? '').slice(0, 10),
      }
    : { kategori: '', code: '', nama: '', divisi: '', zoneId: '', qty: 1, status: 'Aktif', tanggalMasuk: dayjs().format('YYYY-MM-DD') };
  assetModalOpen.value = true;
}

async function saveAsset() {
  if (!form.value.code.trim() || !form.value.nama.trim()) { message.warning('Kode & nama asset wajib diisi'); return; }
  assetSaving.value = true;
  const payload: Record<string, any> = {
    kategori: form.value.kategori.trim(), code: form.value.code.trim(), nama: form.value.nama.trim(),
    divisi: form.value.divisi.trim(), zoneId: form.value.zoneId.trim(), qty: String(num(form.value.qty)),
    status: form.value.status, tanggalMasuk: form.value.tanggalMasuk,
  };
  try {
    if (editingAsset.value) {
      await fclPatch('assetwarehouse', editingAsset.value.id, payload);
      notification.success({ message: 'Asset', description: 'Asset diperbarui' });
    } else {
      await fclInsert('assetwarehouse', [{ ...payload, createdBy: 'admin', createdAt: dayjs().format('YYYY-MM-DD HH:mm:ss') }]);
      notification.success({ message: 'Asset', description: 'Asset ditambahkan' });
    }
    assetModalOpen.value = false;
    await loadAll(true);
  } catch (err: any) {
    notification.error({ message: 'Asset', description: err?.message || 'Gagal menyimpan' });
  } finally {
    assetSaving.value = false;
  }
}

function deleteAsset(a: AssetRow) {
  Modal.confirm({
    title: 'Hapus asset?', content: `${a.code} — ${a.nama}`, okText: 'Hapus', okType: 'danger',
    onOk: async () => { await fclDelete('assetwarehouse', a.id); notification.success({ message: 'Asset dihapus' }); await loadAll(true); },
  });
}

// ── Move / Print ─────────────────────────────────────────────────────────────
const moveOpen = ref(false);
const moveSaving = ref(false);
const moveTarget = ref<AssetRow | null>(null);
const moveForm = ref({ divisi: '', zoneId: '' });

function openMove(a: AssetRow) {
  moveTarget.value = a;
  moveForm.value = { divisi: a.divisi ?? '', zoneId: a.zoneId ?? '' };
  moveOpen.value = true;
}

async function saveMove() {
  if (!moveTarget.value) return;
  moveSaving.value = true;
  try {
    await fclPatch('assetwarehouse', moveTarget.value.id, { divisi: moveForm.value.divisi, zoneId: moveForm.value.zoneId });
    notification.success({ message: 'Asset dipindahkan' });
    moveOpen.value = false;
    await loadAll(true);
  } catch (err: any) {
    notification.error({ message: 'Pindah Asset', description: err?.message || 'Gagal memindahkan' });
  } finally {
    moveSaving.value = false;
  }
}

function printLabel(a: AssetRow) {
  const w = window.open('', '_blank', 'width=420,height=300');
  if (!w) { message.warning('Popup diblokir browser'); return; }
  w.document.write(`<!doctype html><html><head><meta charset="utf-8"><title>Label ${a.code}</title><style>
  body{font-family:'Segoe UI',Arial,sans-serif;margin:16px;text-align:center}
  .box{border:2px solid #111;border-radius:8px;padding:18px;max-width:320px;margin:auto}
  .kode{font-family:monospace;font-size:22px;font-weight:800;margin:8px 0}
  .nama{font-size:14px;font-weight:700}.meta{font-size:11px;color:#444;margin-top:8px}
  
/* ── Stock Opname QR scanner ── */
.so-label { color: hsl(var(--warning)); font-size: 12px; font-weight: 800; margin: 14px 16px 6px; border-bottom: 2px solid hsl(var(--warning)); display: inline-block; padding-bottom: 2px; }
.so-filter { display: flex; gap: 10px; align-items: end; flex-wrap: wrap; margin: 12px 16px; padding: 12px; background: hsl(var(--muted)); border: 1px solid hsl(var(--border)); border-radius: 6px; }
.so-filter-item { display: flex; flex-direction: column; gap: 4px; }
.so-filter-item label { font-size: 10px; font-weight: 800; color: hsl(var(--muted-foreground)); }
.so-filter-item .aw-input { width: 220px; height: 34px; }
.so-laporan { margin-left: auto; }
.so-progress-head { display: flex; justify-content: space-between; gap: 10px; font-size: 11px; padding: 0 16px 6px; color: hsl(var(--foreground)); }
.so-progress-head span { color: hsl(var(--muted-foreground)); }
.so-progress-track { height: 6px; margin: 0 16px 12px; background: hsl(var(--muted)); border-radius: 999px; overflow: hidden; }
.so-progress-fill { height: 100%; background: hsl(var(--success)); transition: width 0.3s; }

.so-grid { display: grid; grid-template-columns: 340px 1fr; gap: 12px; padding: 0 16px 16px; }
.so-scanner { background: hsl(var(--card)); border: 1px solid hsl(var(--border)); border-radius: 6px; padding: 12px; }
.so-card-title { font-size: 12px; font-weight: 800; color: hsl(var(--foreground)); margin-bottom: 10px; }
.so-camera { position: relative; background: #000; border: 1px solid hsl(var(--border)); border-radius: 6px; overflow: hidden; aspect-ratio: 4/3; }
.so-video { width: 100%; height: 100%; object-fit: cover; }
.so-camera-overlay { position: absolute; inset: 0; display: flex; flex-direction: column; align-items: center; justify-content: center; gap: 8px; background: rgb(0 0 0 / 0.6); }
.so-link { background: hsl(var(--muted)); color: hsl(var(--foreground)); border: 1px solid hsl(var(--border)); font-size: 11px; font-weight: 700; padding: 6px 10px; border-radius: 4px; cursor: pointer; }
.so-upload { margin-top: 10px; padding: 8px; border: 1px dashed hsl(var(--warning)); border-radius: 6px; }
.so-upload-label { font-size: 10px; font-weight: 800; color: hsl(var(--warning)); margin-bottom: 6px; }
.so-file { font-size: 11px; width: 100%; color: hsl(var(--muted-foreground)); }
.so-manual { margin-top: 10px; }
.so-manual label { display: block; font-size: 10px; font-weight: 800; color: hsl(var(--muted-foreground)); margin-bottom: 4px; }
.so-manual-row { display: flex; gap: 6px; }

.so-sudah, .so-belum { background: hsl(var(--card)); border: 1px solid hsl(var(--success)); border-radius: 6px; padding: 10px 12px; }
.so-scanner { grid-column: 1; grid-row: 1 / span 2; }
.so-sudah { grid-column: 2; grid-row: 1; }
.so-belum { grid-column: 2; grid-row: 2; }
.so-belum { border-color: hsl(var(--warning)); }
.so-sudah .so-card-title { color: hsl(var(--success)); }
.so-belum .so-card-title { color: hsl(var(--warning)); }
.so-belum-wrap { max-height: 420px; overflow-y: auto; }
.so-table { width: 100%; border-collapse: collapse; font-size: 11px; }
.so-table th { background: hsl(var(--muted)); color: hsl(var(--muted-foreground)); font-size: 10px; font-weight: 800; text-align: left; padding: 6px 8px; border-bottom: 1px solid hsl(var(--border)); }
.so-table td { padding: 6px 8px; border-bottom: 1px solid hsl(var(--border)); color: hsl(var(--foreground)); white-space: nowrap; }
.so-code { font-family: monospace; color: hsl(var(--primary)); font-weight: 700; }
.so-status { background: hsl(var(--destructive) / 0.14); color: hsl(var(--destructive)); font-size: 9px; font-weight: 800; padding: 2px 8px; border-radius: 999px; }
.so-undo { border: none; background: transparent; color: hsl(var(--destructive)); font-size: 16px; cursor: pointer; }
@media (max-width: 1100px) { .so-grid { grid-template-columns: 1fr; } }


/* ── Hasil Stock Opname ── */
.hasil-panel { margin: 0 16px 12px; padding: 12px; background: hsl(var(--muted)); border: 1px solid hsl(var(--border)); border-radius: 6px; }
.hasil-head { display: flex; justify-content: space-between; align-items: center; gap: 10px; margin-bottom: 10px; }
.hasil-title { font-size: 12px; font-weight: 800; color: hsl(var(--foreground)); }
.hasil-chips { display: flex; gap: 6px; }
.hasil-table th, .hasil-table td { padding: 9px 12px; }
.hasil-selisih, .hasil-status { display: inline-block; font-size: 10px; font-weight: 800; padding: 3px 9px; border-radius: 999px; white-space: nowrap; }
.hasil-ok { background: hsl(var(--success) / 0.14); color: hsl(var(--success)); }
.hasil-bad { background: hsl(var(--destructive) / 0.14); color: hsl(var(--destructive)); }
.hasil-selesai { background: hsl(var(--success) / 0.14); color: hsl(var(--success)); }
.hasil-draft { background: hsl(var(--muted)); color: hsl(var(--muted-foreground)); border: 1px solid hsl(var(--border)); }
.hasil-aksi { display: flex; gap: 4px; }
.hasil-detail-meta { font-size: 11px; color: hsl(var(--muted-foreground)); margin-bottom: 10px; }

.hasil-detail-cards { display: grid; grid-template-columns: repeat(4, 1fr); gap: 8px; margin-bottom: 10px; }
.hasil-detail-cards > div { background: hsl(var(--muted)); border: 1px solid hsl(var(--border)); border-radius: 6px; padding: 8px 10px; display: flex; flex-direction: column; gap: 3px; }
.hasil-detail-cards span { font-size: 9px; font-weight: 800; color: hsl(var(--muted-foreground)); }
.hasil-detail-cards b { font-size: 15px; color: hsl(var(--foreground)); }
.hasil-detail-note { font-size: 11px; color: hsl(var(--warning)); background: hsl(var(--warning) / 0.1); border: 1px solid hsl(var(--warning) / 0.3); border-radius: 6px; padding: 8px 10px; margin-bottom: 10px; }

.so-start { margin: 0 16px 16px; padding: 40px 16px; text-align: center; background: hsl(var(--card)); border: 1px dashed hsl(var(--warning)); border-radius: 8px; }
.so-start-icon { font-size: 34px; margin-bottom: 8px; }
.so-start-title { font-size: 14px; font-weight: 800; color: hsl(var(--foreground)); }
.so-start-desc { font-size: 11px; color: hsl(var(--muted-foreground)); margin: 6px 0 14px; }

.hasil-detail-note-green { background: hsl(var(--success) / 0.1); border-color: hsl(var(--success) / 0.3); color: hsl(var(--success)); }
.hasil-row-selisih { background: hsl(var(--destructive) / 0.08) !important; }
.aw-muted { color: hsl(var(--muted-foreground)); }

.hasil-row-new { background: hsl(var(--primary) / 0.08) !important; }
.hasil-baru { font-size: 9px; font-weight: 800; background: hsl(var(--primary)); color: hsl(var(--primary-foreground)); border-radius: 4px; padding: 2px 6px; margin-left: 6px; }
.hasil-detail-row td { background: hsl(var(--muted) / 0.4); }
.hasil-detail-title { font-size: 10px; font-weight: 800; color: hsl(var(--muted-foreground)); text-transform: uppercase; margin-bottom: 8px; }

.hasil-missing { margin-top: 12px; }
.hasil-missing-title { font-size: 11px; font-weight: 800; color: hsl(var(--destructive)); margin-bottom: 6px; }
.hasil-missing-wrap { max-height: 240px; overflow-y: auto; border: 1px solid hsl(var(--border)); border-radius: 6px; }
.hasil-missing-inline { margin-top: 10px; }

.hasil-head-actions { display: flex; gap: 6px; }
.hasil-check { accent-color: hsl(var(--primary)); width: 15px; height: 15px; cursor: pointer; }
.hasil-row-picked { background: hsl(var(--primary) / 0.06) !important; }
</style></head><body><div class="box"><div class="nama">${String(a.nama ?? '').toUpperCase()}</div>
  <div class="kode">${String(a.code ?? '-')}</div><div class="meta">${String(a.kategori ?? '-')} · ${String(a.divisi ?? '-')}<br/>Qty ${num(a.qty)} · ${String(a.tanggalMasuk ?? '')}</div></div></body></html>`);
  w.document.close(); w.focus();
  setTimeout(() => { try { w.print(); } catch { /* abaikan */ } }, 300);
}

// ── CSV ──────────────────────────────────────────────────────────────────────
function downloadCsv(filename: string, lines: string[][]) {
  const csv = lines.map((cols) => cols.map((c) => `"${String(c ?? '').replace(/"/g, '""')}"`).join(',')).join('\n');
  const url = URL.createObjectURL(new Blob(['﻿' + csv], { type: 'text/csv;charset=utf-8;' }));
  const a = document.createElement('a'); a.href = url; a.download = filename; a.click(); URL.revokeObjectURL(url);
}

function exportCsv() {
  const lines = [['Kategori', 'Kode', 'Nama', 'Divisi', 'Zone', 'Qty', 'Status', 'Tanggal Masuk']];
  filteredAssets.value.forEach((a) => lines.push([a.kategori ?? '', a.code ?? '', a.nama ?? '', a.divisi ?? '', a.zoneId ?? '', String(a.qty ?? ''), a.status ?? '', a.tanggalMasuk ?? '']));
  downloadCsv('asset-warehouse.csv', lines);
}

const importInput = ref<HTMLInputElement | null>(null);
async function handleImport(event: Event) {
  const file = (event.target as HTMLInputElement).files?.[0];
  if (!file) return;
  try {
    const text = await file.text();
    const lines = text.replace(/^\uFEFF/, '').split(/\r?\n/).filter(Boolean);
    const parsed = lines.map((line) => (line.match(/("([^"]|"")*"|[^,])+/g) ?? [])
      .map((c) => c.replace(/^"|"$/g, '').replace(/""/g, '"').trim()));
    const rows = parsed.slice(1).map((c) => ({
      kategori: c[0] || '-', code: c[1] || '', nama: c[2] || '', divisi: c[3] || '',
      zoneId: c[4] || '', qty: String(num(c[5]) || 1), status: c[6] || 'Aktif', tanggalMasuk: c[7] || dayjs().format('YYYY-MM-DD'),
      createdBy: 'admin', createdAt: dayjs().format('YYYY-MM-DD HH:mm:ss'),
    })).filter((r) => r.code && r.nama);
    if (!rows.length) throw new Error('Tidak ada baris valid');
    await fclInsert('assetwarehouse', rows);
    notification.success({ message: 'Impor Asset', description: `${rows.length} asset ditambahkan` });
    await loadAll(true);
  } catch (err: any) {
    notification.error({ message: 'Impor Asset', description: err?.message || 'Gagal membaca CSV' });
  } finally {
    if (importInput.value) importInput.value.value = '';
  }
}

async function syncCodes() {
  try {
    for (const a of assets.value) {
      const code = String(a.code ?? '').trim();
      const nama = String(a.nama ?? '').trim();
      if (code !== a.code || nama !== a.nama) await fclPatch('assetwarehouse', a.id, { code, nama });
    }
    notification.success({ message: 'Sinkron', description: 'Kode & nama asset sudah disinkronkan' });
    await loadAll(true);
  } catch (err: any) {
    notification.error({ message: 'Sinkron', description: err?.message || 'Gagal sinkron' });
  }
}

// ── Stock Opname (QR scanner + manual) ─────────────────────────────────────
const soTanggal = ref(dayjs().format('YYYY-MM-DD'));
const soDivisi = ref('');
const manualCode = ref('');
const imageInput = ref<HTMLInputElement | null>(null);
const videoRef = ref<HTMLVideoElement | null>(null);
const cameraStream = ref<MediaStream | null>(null);
let scanTimer: ReturnType<typeof setInterval> | undefined;

const soDivisiOptions = computed(() =>
  [...new Set(assets.value.map((a) => String(a.divisi ?? '').trim()).filter(Boolean))].sort(),
);
const soScannedCodes = computed(() =>
  new Set(opnameLogs.value.map((l) => String(l.assetCode ?? '').trim().toLowerCase())),
);
const soAssets = computed(() =>
  assets.value.filter((a) => !soDivisi.value || String(a.divisi ?? '') === soDivisi.value),
);
const soSudah = computed(() =>
  opnameLogs.value.filter((l) => !soDivisi.value || String(l.divisi ?? '') === soDivisi.value),
);
const soBelum = computed(() =>
  soAssets.value.filter((a) => !soScannedCodes.value.has(String(a.code ?? '').trim().toLowerCase())),
);
const soProgress = computed(() =>
  soAssets.value.length ? Math.min(100, Math.round((soSudah.value.length / soAssets.value.length) * 100)) : 0,
);

// Sesi SO yang sedang berjalan (hari ini). Menentukan apakah daftar scan ditampilkan.
const activeSoSession = computed(() =>
  sessions.value.find((x) => norm(x.status) === 'berjalan' && String(x.tanggal) === soTanggal.value)
  ?? sessions.value.find((x) => norm(x.status) === 'berjalan')
  ?? null,
);

async function mulaiSesi() {
  if (activeSoSession.value) {
    message.info('Sesi SO sudah berjalan — langsung scan asetnya');
    return;
  }
  await fclInsert('assetopnamesession', [{
    tanggal: soTanggal.value,
    divisi: soDivisi.value || 'Semua Divisi',
    status: 'Berjalan',
    totalAsset: String(soAssets.value.length),
    terscan: '0',
    createdBy: 'admin',
    createdAt: dayjs().format('YYYY-MM-DD HH:mm:ss'),
  }]);
  notification.success({ message: 'Mulai Sesi SO', description: 'Sesi dimulai. Daftar aset siap di-scan.' });
  await loadAll(true);
}

async function recordScan(code: string, kondisi = 'Baik', qtyFisik?: number) {
  const key = String(code ?? '').trim().toLowerCase();
  if (!key) return false;
  if (soScannedCodes.value.has(key)) { message.info(`Aset ${code} sudah terscan`); return false; }
  const a = assets.value.find((x) => String(x.code ?? '').trim().toLowerCase() === key);
  // Aset yang tidak ada di master tetap dilampirkan dengan nama "Tidak ada" (qty sistem 0)
  if (!a) message.info(`Kode "${code}" tidak ada di master — tetap dicatat sebagai “Tidak ada”`);
  // Pilih session aktif ( Berjalan / tanggal hari ini ), jika tidak ada yang terbaru
  const today = dayjs().format('YYYY-MM-DD');
  const active = sessions.value.find((x) => norm(x.status) === 'berjalan' && String(x.tanggal) === today)
    ?? sessions.value.find((x) => norm(x.status) === 'berjalan')
    ?? sessions.value.slice().sort((x, y) => String(y.tanggal ?? '').localeCompare(String(x.tanggal ?? '')))[0];
  const sessionId = active ? String(active.id) : '';
  await fclInsert('assetopnamelog', [{
    sessionId, assetId: a?.id ?? '', assetCode: a?.code ?? String(code).trim(), assetNama: a?.nama ?? 'Tidak ada',
    divisi: a?.divisi ?? soDivisi.value ?? '-', qtyFisik: String(qtyFisik ?? num(a?.qty ?? 1)), qtySistem: String(num(a?.qty)),
    kondisi, catatan: a ? '' : 'Aset tidak ditemukan di master', scannedBy: 'admin', scannedAt: dayjs().format('YYYY-MM-DD HH:mm:ss'),
  }]);
  if (active) {
    await fclPatch('assetopnamesession', active.id, { terscan: String(num(active.terscan) + 1) });
  }
  await loadAll(true);
  return true;
}

async function manualScan() {
  if (await recordScan(manualCode.value)) manualCode.value = '';
}

async function scanRow(a: { code?: string; qty?: string | number }) {
  await recordScan(String(a.code ?? ''), 'Baik', num(a.qty));
}

async function undoScan(logId: string) {
  await fclDelete('assetopnamelog', logId);
  await loadAll(true);
}

// Scan dari file gambar (BarcodeDetector bila didukung browser)
async function handleImageScan(event: Event) {
  const file = (event.target as HTMLInputElement).files?.[0];
  if (!file) return;
  const Detector = (window as any).BarcodeDetector;
  if (!Detector) { message.warning('Browser tidak mendukung deteksi barcode dari gambar — gunakan Input Manual'); return; }
  try {
    const detector = new Detector({ formats: ['code_128', 'code_39', 'ean_13', 'qr_code'] });
    const bitmap = await createImageBitmap(file);
    const found = await detector.detect(bitmap);
    if (!found.length) { message.warning('Barcode tidak terdeteksi pada gambar'); return; }
    for (const f of found) await recordScan(String(f.rawValue ?? ''));
  } catch (err: any) {
    notification.error({ message: 'Scan Gambar', description: err?.message || 'Gagal memproses gambar' });
  } finally {
    if (imageInput.value) imageInput.value.value = '';
  }
}

// Scan dari kamera
async function startCamera() {
  try {
    cameraStream.value = await navigator.mediaDevices.getUserMedia({ video: { facingMode: 'environment' } });
    if (videoRef.value) {
      videoRef.value.srcObject = cameraStream.value;
      await videoRef.value.play();
    }
    message.success('Kamera aktif');
    const Detector = (window as any).BarcodeDetector;
    if (Detector && !scanTimer) {
      const detector = new Detector({ formats: ['code_128', 'code_39', 'ean_13', 'qr_code'] });
      scanTimer = setInterval(async () => {
        if (!videoRef.value || videoRef.value.readyState < 2) return;
        try {
          const found = await detector.detect(videoRef.value);
          if (found.length) await recordScan(String(found[0].rawValue ?? ''));
        } catch { /* abaikan frame gagal */ }
      }, 800);
    }
  } catch {
    message.warning('Kamera terblokir. Gunakan upload foto atau Input Manual.');
  }
}

function stopCamera() {
  if (scanTimer) { clearInterval(scanTimer); scanTimer = undefined; }
  cameraStream.value?.getTracks().forEach((t) => t.stop());
  cameraStream.value = null;
}

async function resetSo() {
  const session = activeSoSession.value;
  if (!session) {
    soTanggal.value = dayjs().format('YYYY-MM-DD');
    soDivisi.value = '';
    manualCode.value = '';
    message.info('Filter & input scan dikembalikan ke awal');
    return;
  }
  Modal.confirm({
    title: 'Reset sesi SO?',
    content: `Sesi ${session.tanggal} (${session.divisi || 'Semua Divisi'}) dan seluruh data scan yang belum diselesaikan akan dihapus. Sesi selesai tidak terpengaruh.`,
    okText: 'Reset',
    okType: 'danger',
    onOk: async () => {
      try {
        const logs = opnameLogs.value.filter((l) => String(l.sessionId) === String(session.id));
        for (const l of logs) await fclDelete('assetopnamelog', l.id);
        await fclDelete('assetopnamesession', session.id);
        stopCamera();
        manualCode.value = '';
        soTanggal.value = dayjs().format('YYYY-MM-DD');
        soDivisi.value = '';
        notification.success({ message: 'Reset', description: 'Sesi SO direset — siap memulai sesi baru' });
        await loadAll(true);
      } catch (err: any) {
        notification.error({ message: 'Reset', description: err?.message || 'Gagal mereset sesi' });
      }
    },
  });
}

async function selesaikanSo() {
  if (!activeSoSession.value) {
    message.warning('Mulai Sesi SO terlebih dahulu');
    return;
  }
  const session = activeSoSession.value;
  const scanned = opnameLogs.value.filter((l) => String(l.sessionId) === String(session.id)).length;
  await fclPatch('assetopnamesession', session.id, {
    status: 'Selesai',
    totalAsset: String(soAssets.value.length),
    terscan: String(scanned || soSudah.value.length),
    approvedAt: dayjs().format('YYYY-MM-DD HH:mm:ss'),
  });
  stopCamera();
  notification.success({ message: 'Selesai', description: 'Sesi SO ditutup — data langsung tampil di Hasil SO' });
  await loadAll(true);
  hasilFilter.value = 'semua';
  highlightSessionId.value = String(session.id);
  expandedHasil.value = new Set([String(session.id)]);
  activeTab.value = 'hasil';
}

// ── Hasil Stock Opname (riwayat session) ────────────────────────────────────
const hasilFilter = ref<'semua' | 'sesuai' | 'tidak'>('semua');
const hasilDetailOpen = ref(false);
const hasilDetailSession = ref<OpnameSession | null>(null);
const highlightSessionId = ref('');
const expandedHasil = ref<Set<string>>(new Set());

function selisihOf(s: OpnameSession): number {
  return num(s.totalAsset) - num(s.terscan);
}

const hasilSessions = computed(() =>
  sessions.value
    .filter((s) => {
      const d = selisihOf(s);
      if (hasilFilter.value === 'sesuai') return d === 0;
      if (hasilFilter.value === 'tidak') return d !== 0;
      return true;
    })
    .slice()
    .sort((a, b) => String(b.tanggal ?? '').localeCompare(String(a.tanggal ?? ''))),
);

function logsOfSession(s: OpnameSession | null): OpnameLog[] {
  if (!s) return [];
  // Data baru: log tersimpan dengan sessionId. Data lama: sessionId kosong → cocokkan tanggal.
  const direct = opnameLogs.value.filter((l) => String(l.sessionId) === String(s.id));
  if (direct.length) return direct;
  const tanggal = String(s.tanggal ?? '');
  const legacy = opnameLogs.value.filter((l) =>
    (!l.sessionId || String(l.sessionId).trim() === '') && String(l.scannedAt ?? '').slice(0, 10) === tanggal,
  );
  if (legacy.length) return legacy;
  return opnameLogs.value.filter((l) => String(l.scannedAt ?? '').slice(0, 10) === tanggal);
}

const hasilDetailLogs = computed(() => logsOfSession(hasilDetailSession.value));

function missingOfSession(s: OpnameSession | null): AssetRow[] {
  if (!s) return [];
  const div = String(s.divisi ?? '').trim();
  let list = assets.value;
  if (div && norm(div) !== 'semua divisi') list = list.filter((a) => String(a.divisi ?? '').trim() === div);
  const scanned = new Set(logsOfSession(s).map((l) => String(l.assetCode ?? '').trim().toLowerCase()));
  return list.filter((a) => !scanned.has(String(a.code ?? '').trim().toLowerCase()));
}

function toggleHasil(id: string) {
  const next = new Set(expandedHasil.value);
  if (next.has(id)) next.delete(id);
  else next.add(id);
  expandedHasil.value = next;
}

// ── Checklist & export hasil SO ─────────────────────────────────────────────
const selectedHasilIds = ref<Set<string>>(new Set());

function toggleHasilCheck(id: string) {
  const next = new Set(selectedHasilIds.value);
  if (next.has(id)) next.delete(id);
  else next.add(id);
  selectedHasilIds.value = next;
}

const allHasilChecked = computed(() =>
  hasilSessions.value.length > 0 && hasilSessions.value.every((s) => selectedHasilIds.value.has(String(s.id))),
);

function toggleAllHasil() {
  selectedHasilIds.value = allHasilChecked.value
    ? new Set()
    : new Set(hasilSessions.value.map((s) => String(s.id)));
}

function exportHasilTerpilih() {
  const chosen = hasilSessions.value.filter((s) => selectedHasilIds.value.has(String(s.id)));
  if (!chosen.length) { message.warning('Centang minimal 1 sesi untuk diekspor'); return; }
  const header = ['TANGGAL', 'DIVISI', 'STATUS SESI', 'TOTAL ASET', 'TER-SCAN', 'SELISIH SESI', 'DIBUAT OLEH',
    'KODE ASET', 'NAMA ASET', 'JENIS DATA', 'QTY SISTEM', 'QTY FISIK', 'SELISIH ASET', 'KONDISI', 'WAKTU SCAN'];
  const lines: string[][] = [header];
  for (const ses of chosen) {
    const common = [
      String(ses.tanggal ?? '').slice(0, 10), ses.divisi || 'Semua Divisi', ses.status || '',
      String(ses.totalAsset ?? ''), String(ses.terscan ?? ''),
      String(selisihOf(ses) === 0 ? 'Sesuai' : 'Selisih ' + Math.abs(selisihOf(ses))),
      ses.createdBy || '',
    ];
    for (const l of logsOfSession(ses)) {
      lines.push([...common, String(l.assetCode ?? ''), String(l.assetNama ?? ''), 'Ter-scan',
        String(l.qtySistem ?? ''), String(l.qtyFisik ?? ''), String(selisihFisik(l)), String(l.kondisi ?? ''), String(l.scannedAt ?? '')]);
    }
    for (const a of missingOfSession(ses)) {
      lines.push([...common, String(a.code ?? ''), String(a.nama ?? ''), 'Belum di-scan',
        String(num(a.qty)), '', '', '', '']);
    }
  }
  downloadCsv(`hasil-so-${dayjs().format('YYYYMMDD_HHmm')}.csv`, lines);
  notification.success({ message: 'Export Hasil SO', description: `${chosen.length} sesi diekspor (${lines.length - 1} baris data)` });
}

// Referensi dari data master bila detail scan legacy tidak tersimpan
const hasilDetailReferensi = computed(() => {
  const s = hasilDetailSession.value;
  if (!s || hasilDetailLogs.value.length) return [] as AssetRow[];
  const div = String(s.divisi ?? '');
  let list = assets.value;
  if (div && norm(div) !== 'semua divisi') list = list.filter((a) => String(a.divisi ?? '') === div);
  return list.slice(0, Math.max(1, num(s.terscan)));
});

function selisihFisik(l: OpnameLog): number {
  return num(l.qtyFisik) - num(l.qtySistem);
}

function openHasilDetail(s: OpnameSession) {
  hasilDetailSession.value = s;
  hasilDetailOpen.value = true;
}

function deleteHasil(s: OpnameSession) {
  Modal.confirm({
    title: 'Hapus laporan?',
    content: `Hapus session opname ${s.tanggal} (${s.divisi || 'Semua Divisi'})?`,
    okText: 'Hapus',
    okType: 'danger',
    onOk: async () => {
      await fclDelete('assetopnamesession', s.id);
      notification.success({ message: 'Laporan dihapus' });
      await loadAll(true);
    },
  });
}

onUnmounted(stopCamera);
onMounted(loadAll);
</script>

<template>
  <Page auto-content-height>
    <div class="aw-panel">
      <!-- ── Header aksi ── -->
      <div class="aw-head">
        <div class="aw-title">Asset Warehouse</div>
        <div class="aw-actions">
          <button class="aw-btn aw-btn-ghost" @click="syncCodes"><LucideRefreshCw :size="13" /> Sinkron</button>
          <input ref="importInput" type="file" accept=".csv,text/csv" class="aw-hidden-file" @change="handleImport" />
          <button class="aw-btn aw-btn-ghost" @click="importInput?.click()"><LucidePlus :size="13" /> Impor</button>
          <button class="aw-btn aw-btn-ghost" @click="exportCsv"><LucideDownload :size="13" /> Export</button>
          <button class="aw-btn aw-btn-ghost" @click="reportOpen = true"><LucideFilePenLine :size="13" /> Laporan</button>
          <button class="aw-btn aw-btn-accent" @click="assets[0] && printLabel(assets[0])"><LucidePrinter :size="13" /> Print</button>
          <button class="aw-btn aw-btn-primary" @click="openAssetModal()"><LucidePlus :size="13" /> Tambah Asset</button>
        </div>
      </div>

      <!-- ── Tabs ── -->
      <div class="aw-tabs">
        <button class="aw-tab" :class="{ 'aw-tab-active': activeTab === 'list' }" @click="activeTab = 'list'">Daftar Asset</button>
        <button class="aw-tab" :class="{ 'aw-tab-active': activeTab === 'opname' }" @click="activeTab = 'opname'">Stock Opname</button>
        <button class="aw-tab" :class="{ 'aw-tab-active': activeTab === 'hasil' }" @click="activeTab = 'hasil'">Hasil Stock Opname</button>
      </div>

      <!-- ── Daftar Asset ── -->
      <template v-if="activeTab === 'list'">
        <div class="aw-section-label">Daftar Asset</div>
        <div class="aw-section-sub">Stock Opname</div>
        <div class="aw-section-sub">Hasil Stock Opname</div>

        <div class="aw-stats">
          <div class="aw-stat"><div class="aw-stat-label">TOTAL</div><div class="aw-stat-value aw-primary">{{ stats.total }}</div></div>
          <div class="aw-stat"><div class="aw-stat-label">MARKETPLACE</div><div class="aw-stat-value">{{ stats.mp }}</div></div>
          <div class="aw-stat"><div class="aw-stat-label">DISTRIBUTOR</div><div class="aw-stat-value aw-orange">{{ stats.dist }}</div></div>
          <div class="aw-stat"><div class="aw-stat-label">KOL</div><div class="aw-stat-value aw-purple">{{ stats.kol }}</div></div>
          <div class="aw-stat"><div class="aw-stat-label">AKTIF</div><div class="aw-stat-value aw-green">{{ stats.aktif }}</div></div>
          <div class="aw-stat"><div class="aw-stat-label">RUSAK / PERBAIKAN</div><div class="aw-stat-value aw-red">{{ stats.rusak }} / {{ stats.perbaikan }}</div></div>
        </div>

        <div class="aw-filters">
          <input v-model="search" type="text" class="aw-search" placeholder="Cari Nama / Kode Asset..." />
          <select v-model="divisiFilter" class="aw-select"><option value="">Semua Divisi</option><option v-for="d in divisiOptions" :key="d" :value="d">{{ d }}</option></select>
          <select v-model="kategoriFilter" class="aw-select"><option value="">-- Kategori --</option><option v-for="k in kategoriOptions" :key="k" :value="k">{{ k }}</option></select>
          <select v-model="statusFilter" class="aw-select"><option value="">-- Status --</option><option v-for="s in statusOptions" :key="s" :value="s">{{ s }}</option></select>
        </div>

        <div class="aw-chips">
          <button class="aw-chip" :class="{ 'aw-chip-active': statusChip === 'perbaikan' }" @click="statusChip = statusChip === 'perbaikan' ? 'semua' : 'perbaikan'">Perbaikan</button>
          <button class="aw-chip" :class="{ 'aw-chip-active': statusChip === 'rusak' }" @click="statusChip = statusChip === 'rusak' ? 'semua' : 'rusak'">Rusak</button>
          <button class="aw-chip" :class="{ 'aw-chip-dark': statusChip === 'semua' }" @click="statusChip = 'semua'">Semua</button>
        </div>

        <div class="aw-table-wrap">
          <table class="aw-table">
            <thead><tr><th>DTL</th><th>KATEGORI</th><th>TOTAL UNIT</th><th>DISTRIBUSI DIVISI</th><th class="r">STATUS RECORD</th></tr></thead>
            <tbody>
              <template v-for="g in groups" :key="g.key">
                <tr class="aw-group-row" @click="toggleExpand(g.key)">
                  <td class="c"><button class="aw-expand" @click.stop="toggleExpand(g.key)"><LucideChevronDown :size="14" :class="{ 'aw-rotate': expanded.has(g.key) }" /></button></td>
                  <td>
                    <div class="aw-group-name">{{ g.kategori || g.identity }}</div>
                    <div class="aw-group-sub">{{ g.identity }} — {{ g.items.length }} unit</div>
                  </td>
                  <td><div class="aw-qty">{{ g.total }}</div><div class="aw-unit">UNIT</div></td>
                  <td><div class="aw-sebar">Tersebar di: <b>{{ g.divisi.join(', ') || 'Tanpa Divisi' }}</b></div></td>
                  <td class="r"><span class="aw-records">{{ g.items.length }} RECORDS ›</span></td>
                </tr>
                <tr v-if="expanded.has(g.key)" class="aw-detail-row">
                  <td colspan="5">
                    <div class="aw-detail-title">RINCIAN DISTRIBUSI &amp; LOKASI RACK</div>
                    <div class="aw-detail-list">
                      <div v-for="a in g.items" :key="a.id" class="aw-asset-card" :style="{ borderLeftColor: statusColor(a.status) }">
                        <div class="aw-asset-fields">
                          <div><span class="aw-lbl">TANGGAL</span><span class="aw-val">{{ (a.tanggalMasuk || '').slice(0, 10) || '-' }}</span></div>
                          <div><span class="aw-lbl">KODE ASSET</span><span class="aw-val aw-code">{{ a.code }}</span></div>
                          <div><span class="aw-lbl">KATEGORI</span><span class="aw-val aw-teal">{{ a.kategori || '-' }}</span></div>
                          <div><span class="aw-lbl">NAMA BARANG</span><span class="aw-val">{{ a.nama }}</span></div>
                          <div><span class="aw-lbl">DIVISI / LOKASI</span><span class="aw-val aw-teal">{{ a.divisi || '-' }} {{ a.zoneId ? '· ' + a.zoneId : '' }}</span></div>
                          <div><span class="aw-lbl">STATUS ASSET</span><span class="aw-val" :style="{ color: statusColor(a.status) }">{{ a.status || '-' }}</span></div>
                          <div><span class="aw-lbl">QTY</span><span class="aw-val">{{ num(a.qty) }}</span></div>
                        </div>
                        <div class="aw-asset-actions">
                          <button class="aw-btn aw-btn-accent aw-btn-xs" @click="printLabel(a)">Cetak</button>
                          <button class="aw-btn aw-btn-ghost aw-btn-xs" @click="openMove(a)"><LucideArrowLeftRight :size="11" /> Move</button>
                          <button class="aw-btn aw-btn-ghost aw-btn-xs" @click="openAssetModal(a)"><LucidePencil :size="11" /> Edit</button>
                          <button class="aw-btn aw-btn-danger aw-btn-xs" @click="deleteAsset(a)"><LucideTrash2 :size="11" /> Hapus</button>
                        </div>
                      </div>
                    </div>
                  </td>
                </tr>
              </template>
              <tr v-if="!groups.length"><td colspan="5" class="aw-empty">Data tidak ditemukan</td></tr>
            </tbody>
          </table>
        </div>
      </template>

      <!-- ── Stock Opname ── -->
      <template v-else-if="activeTab === 'opname'">
        <!-- Hasil Stock Opname label -->
        <div class="so-label">Hasil Stock Opname</div>

        <!-- Filter -->
        <div class="so-filter">
          <div class="so-filter-item">
            <label>TANGGAL SO</label>
            <input v-model="soTanggal" type="date" class="aw-input" :disabled="!!activeSoSession" />
          </div>
          <div class="so-filter-item">
            <label>FILTER DIVISI</label>
            <select v-model="soDivisi" class="aw-input" :disabled="!!activeSoSession">
              <option value="">-- Semua Divisi --</option>
              <option v-for="d in soDivisiOptions" :key="d" :value="d">{{ d }}</option>
            </select>
          </div>
          <button class="aw-btn aw-btn-accent" @click="mulaiSesi"><LucidePlus :size="12" /> Mulai Sesi SO</button>
          <button class="aw-btn aw-btn-danger" @click="resetSo">Reset</button>
          <button class="aw-btn aw-btn-ghost so-laporan" @click="selesaikanSo">Selesai</button>
        </div>

        <!-- Progress -->
        <div class="so-progress-head">
          <b>Progress Scan:</b>
          <span>{{ soSudah.length }} / {{ soAssets.length }} Aset Ter-scan</span>
        </div>
        <div class="so-progress-track"><div class="so-progress-fill" :style="{ width: soProgress + '%' }"></div></div>

        <!-- Sebelum sesi dimulai -->
        <div v-if="!activeSoSession" class="so-start">
          <div class="so-start-icon">📋</div>
          <div class="so-start-title">Belum ada Sesi Stock Opname</div>
          <div class="so-start-desc">Klik <b>Mulai Sesi SO</b> di atas untuk membuka daftar aset yang harus di-scan.</div>
          <button class="aw-btn aw-btn-accent" @click="mulaiSesi"><LucidePlus :size="13" /> Mulai Sesi SO</button>
        </div>

        <!-- Scanner + hasil -->
        <div v-else class="so-grid">
          <!-- QR Scanner -->
          <div class="so-scanner">
            <div class="so-card-title">QR Scanner</div>
            <div class="so-camera">
              <video ref="videoRef" class="so-video" playsinline muted></video>
              <div v-if="!cameraStream" class="so-camera-overlay">
                <button class="so-link" @click="startCamera">Request Camera Permissions</button>
                <button class="so-link" @click="imageInput?.click()">Scan an Image File</button>
              </div>
              <input ref="imageInput" type="file" accept="image/*" class="aw-hidden-file" @change="handleImageScan" />
            </div>
            <div class="so-upload">
              <div class="so-upload-label">Kamera Terblokir? Upload foto:</div>
              <input type="file" accept="image/*" class="so-file" @change="handleImageScan" />
            </div>
            <div class="so-manual">
              <label>Input Manual</label>
              <div class="so-manual-row">
                <input v-model="manualCode" class="aw-input" placeholder="AST-001" @keydown.enter.prevent="manualScan" />
                <button class="aw-btn aw-btn-accent" @click="manualScan">Scan</button>
              </div>
            </div>
          </div>

          <!-- Sudah scan -->
          <div class="so-sudah">
            <div class="so-card-title">Sudah Scan ({{ soSudah.length }})</div>
            <table class="so-table">
              <thead><tr><th>#</th><th>KODE ASET</th><th>NAMA</th><th>DIVISI</th><th>QTY</th><th>KONDISI</th><th>JAM</th><th></th></tr></thead>
              <tbody>
                <tr v-for="(l, i) in soSudah" :key="l.id">
                  <td>{{ i + 1 }}</td><td class="so-code">{{ l.assetCode }}</td><td>{{ l.assetNama }}</td>
                  <td>{{ l.divisi || '-' }}</td><td>{{ l.qtyFisik }}</td><td>{{ l.kondisi }}</td>
                  <td>{{ (l.scannedAt || '').slice(11, 16) }}</td>
                  <td><button class="so-undo" title="Batalkan scan" @click="undoScan(l.id)">×</button></td>
                </tr>
                <tr v-if="!soSudah.length"><td colspan="8" class="aw-empty">Belum ada asset terscan</td></tr>
              </tbody>
            </table>
          </div>

          <!-- Belum scan -->
          <div class="so-belum">
            <div class="so-card-title">Belum Scan ({{ soBelum.length }})</div>
            <div class="so-belum-wrap">
              <table class="so-table">
                <thead><tr><th>KODE ASET</th><th>NAMA</th><th>DIVISI</th><th>QTY SISTEM</th><th>STATUS</th><th></th></tr></thead>
                <tbody>
                  <tr v-for="a in soBelum" :key="a.id">
                    <td class="so-code">{{ a.code }}</td><td>{{ a.nama }}</td><td>{{ a.divisi || '-' }}</td>
                    <td>{{ num(a.qty) }}</td>
                    <td><span class="so-status">{{ String(a.status || 'AKTIF').toUpperCase() }}</span></td>
                    <td><button class="aw-btn aw-btn-ghost aw-btn-xs" @click="scanRow(a)">Scan</button></td>
                  </tr>
                  <tr v-if="!soBelum.length"><td colspan="6" class="aw-empty">Semua asset sudah terscan</td></tr>
                </tbody>
              </table>
            </div>
          </div>
        </div>
      </template>

      <!-- ── Hasil Stock Opname ── -->
      <template v-else>
        <div class="so-label">Hasil Stock Opname</div>
        <div class="hasil-panel">
          <div class="hasil-head">
            <div class="hasil-title">Riwayat Hasil Stock Opname · {{ selectedHasilIds.size }} dipilih</div>
            <div class="hasil-head-actions">
              <button class="aw-btn aw-btn-success aw-btn-xs" :disabled="!selectedHasilIds.size" @click="exportHasilTerpilih"><LucideDownload :size="12" /> Export Hasil SO ({{ selectedHasilIds.size }})</button>
              <button class="aw-btn aw-btn-ghost aw-btn-xs" @click="loadAll()">Refresh</button>
            </div>
          </div>
          <div class="hasil-chips">
            <button class="aw-chip" :class="{ 'aw-chip-active': hasilFilter === 'semua' }" @click="hasilFilter = 'semua'">Semua</button>
            <button class="aw-chip" :class="{ 'aw-chip-active': hasilFilter === 'sesuai' }" @click="hasilFilter = 'sesuai'">Stock Sesuai</button>
            <button class="aw-chip" :class="{ 'aw-chip-active': hasilFilter === 'tidak' }" @click="hasilFilter = 'tidak'">Stock Tidak Sesuai</button>
          </div>
        </div>
        <div class="aw-table-wrap">
          <table class="so-table hasil-table">
            <thead>
              <tr><th class="c"><input type="checkbox" class="hasil-check" :checked="allHasilChecked" @change="toggleAllHasil" title="Pilih semua" /></th><th>DTL</th><th>TANGGAL</th><th>DIVISI</th><th>TOTAL ASET</th><th>TER-SCAN</th><th>SELISIH</th><th>STATUS</th><th>DIBUAT OLEH</th><th>AKSI</th></tr>
            </thead>
            <tbody>
              <template v-for="s in hasilSessions" :key="s.id">
              <tr :class="{ 'hasil-row-new': highlightSessionId === String(s.id), 'hasil-row-picked': selectedHasilIds.has(String(s.id)) }">
                <td class="c"><input type="checkbox" class="hasil-check" :checked="selectedHasilIds.has(String(s.id))" @change="toggleHasilCheck(String(s.id))" /></td>
                <td class="c"><button class="aw-expand" @click="toggleHasil(String(s.id))"><LucideChevronDown :size="14" :class="{ 'aw-rotate': expandedHasil.has(String(s.id)) }" /></button></td>
                <td>{{ String(s.tanggal ?? '').slice(0, 10) }} <span v-if="highlightSessionId === String(s.id)" class="hasil-baru">BARU</span></td>
                <td>{{ s.divisi || '-' }}</td>
                <td>{{ s.totalAsset }}</td>
                <td>{{ s.terscan }}</td>
                <td>
                  <span class="hasil-selisih" :class="selisihOf(s) === 0 ? 'hasil-ok' : 'hasil-bad'">
                    {{ selisihOf(s) === 0 ? 'Sesuai' : 'Selisih ' + Math.abs(selisihOf(s)) }}
                  </span>
                </td>
                <td>
                  <span class="hasil-status" :class="String(s.status).toLowerCase() === 'selesai' ? 'hasil-selesai' : 'hasil-draft'">
                    {{ s.status || 'Draft' }}
                  </span>
                </td>
                <td>{{ s.createdBy || '-' }}</td>
                <td>
                  <div class="hasil-aksi">
                    <button class="aw-btn aw-btn-accent aw-btn-xs" @click="openHasilDetail(s)">Lihat</button>
                    <button class="aw-btn aw-btn-danger aw-btn-xs" @click="deleteHasil(s)">Hapus</button>
                  </div>
                </td>
              </tr>
              <tr v-if="expandedHasil.has(String(s.id))" class="hasil-detail-row">
                <td colspan="10">
                  <div class="hasil-detail-title">DATA HASIL SCAN — {{ s.divisi || 'Semua Divisi' }}</div>
                  <table class="so-table">
                    <thead><tr><th>KODE ASET</th><th>NAMA</th><th>DIVISI</th><th class="r">FISIK</th><th class="r">SISTEM</th><th class="r">SELISIH</th><th>KONDISI</th><th>JAM</th></tr></thead>
                    <tbody>
                      <tr v-for="l in logsOfSession(s)" :key="l.id" :class="{ 'hasil-row-selisih': selisihFisik(l) !== 0 }">
                        <td class="so-code">{{ l.assetCode }}</td><td>{{ l.assetNama }}</td><td>{{ l.divisi || '-' }}</td>
                        <td class="r">{{ l.qtyFisik }}</td><td class="r">{{ l.qtySistem }}</td>
                        <td class="r"><span class="hasil-selisih" :class="selisihFisik(l) === 0 ? 'hasil-ok' : 'hasil-bad'">{{ selisihFisik(l) === 0 ? 'Sesuai' : (selisihFisik(l) > 0 ? '+' : '') + selisihFisik(l) }}</span></td>
                        <td>{{ l.kondisi }}</td><td>{{ (l.scannedAt || '').slice(11,16) }}</td>
                      </tr>
                      <tr v-if="!logsOfSession(s).length"><td colspan="8" class="aw-empty">Detail scan belum tersimpan untuk sesi lama ini — klik Lihat untuk referensi aset.</td></tr>
                    </tbody>
                  </table>
                  <div class="hasil-missing-inline">
                    <div class="hasil-missing-title">Aset Belum Ter-scan / Selisih ({{ missingOfSession(s).length }})</div>
                    <div class="hasil-missing-wrap">
                      <table class="so-table">
                        <thead><tr><th>KODE ASET</th><th>NAMA</th><th>DIVISI</th><th class="r">QTY SISTEM</th><th>STATUS</th><th class="r">KETERANGAN</th></tr></thead>
                        <tbody>
                          <tr v-for="a in missingOfSession(s)" :key="a.id"><td class="so-code">{{ a.code }}</td><td>{{ a.nama }}</td><td>{{ a.divisi || '-' }}</td><td class="r">{{ num(a.qty) }}</td><td><span class="so-status">{{ String(a.status || 'AKTIF').toUpperCase() }}</span></td><td class="r"><span class="hasil-selisih hasil-bad">Belum di-scan</span></td></tr>
                          <tr v-if="!missingOfSession(s).length"><td colspan="6" class="aw-empty">Semua aset sudah ter-scan.</td></tr>
                        </tbody>
                      </table>
                    </div>
                  </div>
                </td>
              </tr>
              </template>
              <tr v-if="!hasilSessions.length"><td colspan="10" class="aw-empty">Belum ada riwayat hasil stock opname.</td></tr>
            </tbody>
          </table>
        </div>
      </template>
    </div>

    <!-- ── Modal asset ── -->
    <a-modal v-model:open="assetModalOpen" :title="editingAsset ? 'Edit Asset' : 'Tambah Asset'" :confirm-loading="assetSaving" ok-text="Simpan" cancel-text="Batal" width="620px" @ok="saveAsset">
      <div class="aw-form">
        <div class="aw-form-row"><div class="aw-field"><label>Kategori</label><input v-model="form.kategori" class="aw-input" /></div><div class="aw-field"><label>Kode Asset *</label><input v-model="form.code" class="aw-input" /></div></div>
        <div class="aw-form-row"><div class="aw-field"><label>Nama Asset *</label><input v-model="form.nama" class="aw-input" /></div><div class="aw-field"><label>Divisi</label><input v-model="form.divisi" class="aw-input" /></div></div>
        <div class="aw-form-row"><div class="aw-field"><label>Zona / Lokasi</label><input v-model="form.zoneId" class="aw-input" /></div><div class="aw-field"><label>Qty</label><input v-model.number="form.qty" type="number" min="0" class="aw-input" /></div></div>
        <div class="aw-form-row"><div class="aw-field"><label>Status</label><input v-model="form.status" class="aw-input" placeholder="Aktif / Rusak / Perbaikan" /></div><div class="aw-field"><label>Tanggal Masuk</label><input v-model="form.tanggalMasuk" type="date" class="aw-input" /></div></div>
      </div>
    </a-modal>

    <!-- ── Modal move ── -->
    <a-modal v-model:open="moveOpen" title="Pindah Asset" :confirm-loading="moveSaving" ok-text="Pindahkan" cancel-text="Batal" width="460px" @ok="saveMove">
      <div class="aw-form">
        <div class="aw-field"><label>Divisi Baru</label><input v-model="moveForm.divisi" class="aw-input" /></div>
        <div class="aw-field"><label>Zona / Lokasi Baru</label><input v-model="moveForm.zoneId" class="aw-input" /></div>
      </div>
    </a-modal>

    <!-- ── Modal laporan ── -->
    <a-modal v-model:open="reportOpen" title="Laporan Stock Opname" ok-text="Tutup" :footer="null" cancel-text="Batal" width="640px" @ok="reportOpen = false">
      <div class="aw-report">
        <div v-for="s in sessions" :key="s.id" class="aw-report-item">
          <b>{{ s.tanggal }}</b> — {{ s.divisi || 'Semua Divisi' }} · status: {{ s.status }} · total {{ s.totalAsset }} · terscan {{ s.terscan }}
        </div>
        <div v-if="!sessions.length" class="aw-empty">Belum ada laporan.</div>
      </div>
    </a-modal>

    <!-- ── Modal detail hasil opname ── -->
    <a-modal v-model:open="hasilDetailOpen" :title="'Detail Opname — ' + (hasilDetailSession?.tanggal || '')" ok-text="Tutup" :footer="null" cancel-text="Batal" width="760px" @ok="hasilDetailOpen = false">
      <div class="hasil-detail-meta">
        Divisi: <b>{{ hasilDetailSession?.divisi || 'Semua Divisi' }}</b> · Total: <b>{{ hasilDetailSession?.totalAsset }}</b> · Ter-scan: <b>{{ hasilDetailSession?.terscan }}</b>
      </div>
      <div class="hasil-detail-cards">
        <div><span>TOTAL ASET</span><b>{{ hasilDetailSession?.totalAsset }}</b></div>
        <div><span>TER-SCAN</span><b class="aw-green">{{ hasilDetailSession?.terscan }}</b></div>
        <div><span>SELISIH</span><b :class="num(hasilDetailSession?.totalAsset) - num(hasilDetailSession?.terscan) === 0 ? 'aw-green' : 'aw-red'">
          {{ num(hasilDetailSession?.totalAsset) - num(hasilDetailSession?.terscan) === 0 ? 'Sesuai' : 'Selisih ' + Math.abs(num(hasilDetailSession?.totalAsset) - num(hasilDetailSession?.terscan)) }}
        </b></div>
        <div><span>STATUS</span><b>{{ hasilDetailSession?.status || '-' }}</b></div>
      </div>
      <template v-if="hasilDetailLogs.length">
        <div class="hasil-detail-note hasil-detail-note-green">
          Selisih per aset = <b>Qty Fisik − Qty Sistem</b>. Baris dengan selisih ditandai merah.
        </div>
        <table class="so-table">
          <thead><tr><th>KODE ASET</th><th>NAMA</th><th>DIVISI</th><th class="r">FISIK</th><th class="r">SISTEM</th><th class="r">SELISIH</th><th>KONDISI</th><th>JAM</th></tr></thead>
          <tbody>
            <tr v-for="l in hasilDetailLogs" :key="l.id" :class="{ 'hasil-row-selisih': selisihFisik(l) !== 0 }">
              <td class="so-code">{{ l.assetCode }}</td><td>{{ l.assetNama }}</td><td>{{ l.divisi || '-' }}</td>
              <td class="r"><b>{{ l.qtyFisik }}</b></td><td class="r">{{ l.qtySistem }}</td>
              <td class="r">
                <span class="hasil-selisih" :class="selisihFisik(l) === 0 ? 'hasil-ok' : 'hasil-bad'">
                  {{ selisihFisik(l) === 0 ? 'Sesuai' : (selisihFisik(l) > 0 ? '+' : '') + selisihFisik(l) }}
                </span>
              </td>
              <td>{{ l.kondisi }}</td><td>{{ (l.scannedAt || '').slice(11,16) }}</td>
            </tr>
          </tbody>
        </table>
      <!-- Daftar aset yang belum ter-scan = sumber selisih -->
      <div class="hasil-missing">
        <div class="hasil-missing-title">Aset Belum Ter-scan / Selisih ({{ missingOfSession(hasilDetailSession).length }})</div>
        <div class="hasil-detail-note">
          Selisih <b>{{ Math.abs(num(hasilDetailSession?.totalAsset) - num(hasilDetailSession?.terscan)) }}</b>
          = {{ hasilDetailSession?.totalAsset }} total − {{ hasilDetailSession?.terscan }} ter-scan. Berikut daftar aset yang belum di-scan:
        </div>
        <div class="hasil-missing-wrap">
          <table class="so-table">
            <thead><tr><th>KODE ASET</th><th>NAMA</th><th>DIVISI</th><th class="r">QTY SISTEM</th><th>STATUS</th><th class="r">KETERANGAN</th></tr></thead>
            <tbody>
              <tr v-for="a in missingOfSession(hasilDetailSession)" :key="a.id">
                <td class="so-code">{{ a.code }}</td><td>{{ a.nama }}</td><td>{{ a.divisi || '-' }}</td>
                <td class="r">{{ num(a.qty) }}</td><td><span class="so-status">{{ String(a.status || 'AKTIF').toUpperCase() }}</span></td>
                <td class="r"><span class="hasil-selisih hasil-bad">Belum di-scan</span></td>
              </tr>
              <tr v-if="!missingOfSession(hasilDetailSession).length"><td colspan="6" class="aw-empty">Semua aset sudah ter-scan — tidak ada selisih.</td></tr>
            </tbody>
          </table>
        </div>
      </div>

      </template>
      <template v-else>
        <div class="hasil-detail-note">Detail scan per aset belum tersimpan untuk sesi lama ini. Berikut referensi aset dari data master (divisi: {{ hasilDetailSession?.divisi || 'Semua Divisi' }}):</div>
        <table class="so-table">
          <thead><tr><th>KODE ASET</th><th>NAMA</th><th>DIVISI</th><th class="r">QTY SISTEM</th><th class="r">SELISIH</th><th>STATUS</th></tr></thead>
          <tbody>
            <tr v-for="a in hasilDetailReferensi" :key="a.id"><td class="so-code">{{ a.code }}</td><td>{{ a.nama }}</td><td>{{ a.divisi || '-' }}</td><td class="r">{{ num(a.qty) }}</td><td class="r aw-muted">—</td><td><span class="so-status">{{ String(a.status || 'AKTIF').toUpperCase() }}</span></td></tr>
            <tr v-if="!hasilDetailReferensi.length"><td colspan="6" class="aw-empty">Tidak ada data aset pada divisi ini.</td></tr>
          </tbody>
        </table>
      </template>
    </a-modal>

  </Page>
</template>

<style scoped>
/* Asset Warehouse — token tema vben */
.aw-panel { background: hsl(var(--card)); border: 1px solid hsl(var(--border)); border-radius: 8px; }
.aw-head { display: flex; justify-content: space-between; align-items: center; gap: 10px; flex-wrap: wrap; padding: 14px 16px; }
.aw-title { font-size: 14px; font-weight: 800; color: hsl(var(--foreground)); }
.aw-actions { display: flex; gap: 6px; flex-wrap: wrap; }
.aw-hidden-file { display: none; }
.aw-btn {
  display: inline-flex; align-items: center; justify-content: center; gap: 5px;
  border: 1px solid hsl(var(--border)); background: hsl(var(--muted)); color: hsl(var(--foreground));
  font-size: 12px; font-weight: 700; padding: 6px 12px; border-radius: 6px; cursor: pointer; white-space: nowrap;
}
.aw-btn:hover { filter: brightness(0.96); }
.aw-btn-xs { padding: 3px 8px; font-size: 10px; }
.aw-btn-primary { background: hsl(var(--primary)); border-color: hsl(var(--primary)); color: hsl(var(--primary-foreground)); }
.aw-btn-accent { background: hsl(var(--warning)); border-color: hsl(var(--warning)); color: hsl(var(--warning-foreground)); }
.aw-btn-danger { background: hsl(var(--destructive)); border-color: hsl(var(--destructive)); color: hsl(var(--destructive-foreground)); }

.aw-tabs { display: flex; gap: 18px; border-bottom: 1px solid hsl(var(--border)); padding: 0 16px; }
.aw-tab { border: none; background: transparent; color: hsl(var(--muted-foreground)); font-size: 12px; font-weight: 700; padding: 8px 0; cursor: pointer; border-bottom: 2px solid transparent; }
.aw-tab-active { color: hsl(var(--foreground)); border-bottom-color: hsl(var(--border)); }
.aw-section-label { color: hsl(var(--warning)); font-size: 12px; font-weight: 800; margin: 14px 16px 6px; border-bottom: 2px solid hsl(var(--warning)); display: inline-block; }
.aw-section-sub { color: hsl(var(--muted-foreground)); font-size: 11px; margin: 4px 16px; }

.aw-stats { display: grid; grid-template-columns: repeat(6, minmax(0, 1fr)); gap: 8px; padding: 10px 16px; }
.aw-stat { background: hsl(var(--muted)); border: 1px solid hsl(var(--border)); border-radius: 6px; padding: 10px 12px; }
.aw-stat-label { font-size: 10px; font-weight: 800; color: hsl(var(--muted-foreground)); }
.aw-stat-value { font-size: 20px; font-weight: 800; margin-top: 4px; color: hsl(var(--foreground)); }
.aw-primary { color: hsl(var(--warning)); }
.aw-orange { color: hsl(var(--warning)); }
.aw-purple { color: #a78bfa; }
.aw-green { color: hsl(var(--success)); }
.aw-red { color: hsl(var(--destructive)); }

.aw-filters { display: flex; gap: 8px; flex-wrap: wrap; padding: 0 16px 10px; }
.aw-search { flex: 1; min-width: 220px; height: 34px; padding: 2px 10px; border-radius: 6px; border: 1px solid hsl(var(--border)); background: hsl(var(--background)); color: hsl(var(--foreground)); font-size: 12px; outline: none; }
.aw-search:focus { border-color: hsl(var(--primary)); }
.aw-select { height: 34px; min-width: 140px; padding: 2px 8px; border-radius: 6px; border: 1px solid hsl(var(--border)); background: hsl(var(--background)); color: hsl(var(--foreground)); font-size: 12px; outline: none; }
.aw-chips { display: flex; gap: 6px; padding: 0 16px 10px; }
.aw-chip { border: 1px solid hsl(var(--border)); background: hsl(var(--muted)); color: hsl(var(--muted-foreground)); font-size: 11px; font-weight: 700; padding: 4px 12px; border-radius: 6px; cursor: pointer; }
.aw-chip-active { background: hsl(var(--warning) / 0.16); color: hsl(var(--warning)); border-color: hsl(var(--warning)); }
.aw-chip-dark { background: hsl(var(--accent)); color: hsl(var(--foreground)); }

.aw-table-wrap { overflow-x: auto; border-top: 1px solid hsl(var(--border)); }
.aw-table { width: 100%; min-width: 900px; border-collapse: collapse; font-size: 12px; }
.aw-table th { background: hsl(var(--muted)); color: hsl(var(--muted-foreground)); font-size: 10px; font-weight: 800; letter-spacing: 0.04em; text-align: left; padding: 10px 12px; border-bottom: 1px solid hsl(var(--border)); }
.aw-table th.r, .aw-table td.r { text-align: right; }
.aw-table th.c, .aw-table td.c { text-align: center; }
.aw-table td { padding: 10px 12px; border-bottom: 1px solid hsl(var(--border)); color: hsl(var(--foreground)); vertical-align: middle; }
.aw-group-row { cursor: pointer; }
.aw-group-row:hover { background: hsl(var(--accent)); }
.aw-expand { width: 24px; height: 24px; border-radius: 999px; border: 1px solid hsl(var(--border)); background: hsl(var(--muted)); color: hsl(var(--primary)); display: inline-flex; align-items: center; justify-content: center; cursor: pointer; }
.aw-rotate { transform: rotate(180deg); }
.aw-group-name { font-size: 13px; font-weight: 800; color: hsl(var(--warning)); text-transform: uppercase; letter-spacing: 0.04em; }
.aw-group-sub { font-size: 10px; color: hsl(var(--muted-foreground)); margin-top: 2px; }
.aw-qty { font-size: 16px; font-weight: 800; color: hsl(var(--success)); }
.aw-unit { font-size: 9px; font-weight: 800; color: hsl(var(--muted-foreground)); }
.aw-sebar { font-size: 11px; color: hsl(var(--muted-foreground)); }
.aw-sebar b { color: hsl(var(--success)); }
.aw-records { font-size: 10px; font-weight: 800; background: hsl(var(--success) / 0.14); color: hsl(var(--success)); border-radius: 6px; padding: 4px 10px; }
.aw-empty { text-align: center; color: hsl(var(--muted-foreground)); padding: 24px 8px; }
.aw-table-simple td { white-space: nowrap; }

.aw-detail-row td { background: hsl(var(--muted) / 0.4); }
.aw-detail-title { font-size: 10px; font-weight: 800; color: hsl(var(--muted-foreground)); text-transform: uppercase; border-left: 3px solid hsl(var(--success)); padding-left: 8px; margin-bottom: 10px; }
.aw-detail-list { display: flex; flex-direction: column; gap: 8px; }
.aw-asset-card { border-left: 3px solid hsl(var(--success)); background: hsl(var(--card)); border-radius: 4px; padding: 10px; display: flex; justify-content: space-between; gap: 10px; flex-wrap: wrap; }
.aw-asset-fields { display: flex; flex-wrap: wrap; gap: 14px; }
.aw-asset-fields > div { display: flex; flex-direction: column; gap: 2px; }
.aw-lbl { font-size: 9px; font-weight: 800; color: hsl(var(--muted-foreground)); }
.aw-val { font-size: 11px; color: hsl(var(--foreground)); }
.aw-code { font-family: monospace; color: hsl(var(--primary)); font-weight: 700; }
.aw-teal { color: hsl(var(--success)); }
.aw-asset-actions { display: flex; gap: 4px; align-items: center; }

.aw-subhead { display: flex; justify-content: space-between; align-items: center; gap: 10px; padding: 14px 16px; color: hsl(var(--foreground)); font-size: 12px; }
.aw-report { display: flex; flex-direction: column; gap: 6px; }
.aw-report-item { background: hsl(var(--muted)); border: 1px solid hsl(var(--border)); border-radius: 6px; padding: 8px 10px; font-size: 12px; }

.aw-form { display: flex; flex-direction: column; gap: 12px; padding-top: 4px; }
.aw-form-row { display: grid; grid-template-columns: 1fr 1fr; gap: 10px; }
.aw-field label { display: block; font-size: 11px; font-weight: 700; color: hsl(var(--muted-foreground)); margin-bottom: 4px; }
.aw-input { height: 32px; width: 100%; padding: 4px 9px; border-radius: 6px; border: 1px solid hsl(var(--border)); background: hsl(var(--background)); color: hsl(var(--foreground)); font-size: 12px; outline: none; }
.aw-input:focus { border-color: hsl(var(--primary)); }

@media (max-width: 1100px) {
  .aw-stats { grid-template-columns: repeat(3, minmax(0, 1fr)); }
}

/* ── Stock Opname QR scanner ── */
.so-label { color: hsl(var(--warning)); font-size: 12px; font-weight: 800; margin: 14px 16px 6px; border-bottom: 2px solid hsl(var(--warning)); display: inline-block; padding-bottom: 2px; }
.so-filter { display: flex; gap: 10px; align-items: end; flex-wrap: wrap; margin: 12px 16px; padding: 12px; background: hsl(var(--muted)); border: 1px solid hsl(var(--border)); border-radius: 6px; }
.so-filter-item { display: flex; flex-direction: column; gap: 4px; }
.so-filter-item label { font-size: 10px; font-weight: 800; color: hsl(var(--muted-foreground)); }
.so-filter-item .aw-input { width: 220px; height: 34px; }
.so-laporan { margin-left: auto; }
.so-progress-head { display: flex; justify-content: space-between; gap: 10px; font-size: 11px; padding: 0 16px 6px; color: hsl(var(--foreground)); }
.so-progress-head span { color: hsl(var(--muted-foreground)); }
.so-progress-track { height: 6px; margin: 0 16px 12px; background: hsl(var(--muted)); border-radius: 999px; overflow: hidden; }
.so-progress-fill { height: 100%; background: hsl(var(--success)); transition: width 0.3s; }

.so-grid { display: grid; grid-template-columns: 340px 1fr; gap: 12px; padding: 0 16px 16px; }
.so-scanner { background: hsl(var(--card)); border: 1px solid hsl(var(--border)); border-radius: 6px; padding: 12px; }
.so-card-title { font-size: 12px; font-weight: 800; color: hsl(var(--foreground)); margin-bottom: 10px; }
.so-camera { position: relative; background: #000; border: 1px solid hsl(var(--border)); border-radius: 6px; overflow: hidden; aspect-ratio: 4/3; }
.so-video { width: 100%; height: 100%; object-fit: cover; }
.so-camera-overlay { position: absolute; inset: 0; display: flex; flex-direction: column; align-items: center; justify-content: center; gap: 8px; background: rgb(0 0 0 / 0.6); }
.so-link { background: hsl(var(--muted)); color: hsl(var(--foreground)); border: 1px solid hsl(var(--border)); font-size: 11px; font-weight: 700; padding: 6px 10px; border-radius: 4px; cursor: pointer; }
.so-upload { margin-top: 10px; padding: 8px; border: 1px dashed hsl(var(--warning)); border-radius: 6px; }
.so-upload-label { font-size: 10px; font-weight: 800; color: hsl(var(--warning)); margin-bottom: 6px; }
.so-file { font-size: 11px; width: 100%; color: hsl(var(--muted-foreground)); }
.so-manual { margin-top: 10px; }
.so-manual label { display: block; font-size: 10px; font-weight: 800; color: hsl(var(--muted-foreground)); margin-bottom: 4px; }
.so-manual-row { display: flex; gap: 6px; }

.so-sudah, .so-belum { background: hsl(var(--card)); border: 1px solid hsl(var(--success)); border-radius: 6px; padding: 10px 12px; }
.so-scanner { grid-column: 1; grid-row: 1 / span 2; }
.so-sudah { grid-column: 2; grid-row: 1; }
.so-belum { grid-column: 2; grid-row: 2; }
.so-belum { border-color: hsl(var(--warning)); }
.so-sudah .so-card-title { color: hsl(var(--success)); }
.so-belum .so-card-title { color: hsl(var(--warning)); }
.so-belum-wrap { max-height: 420px; overflow-y: auto; }
.so-table { width: 100%; border-collapse: collapse; font-size: 11px; }
.so-table th { background: hsl(var(--muted)); color: hsl(var(--muted-foreground)); font-size: 10px; font-weight: 800; text-align: left; padding: 6px 8px; border-bottom: 1px solid hsl(var(--border)); }
.so-table td { padding: 6px 8px; border-bottom: 1px solid hsl(var(--border)); color: hsl(var(--foreground)); white-space: nowrap; }
.so-code { font-family: monospace; color: hsl(var(--primary)); font-weight: 700; }
.so-status { background: hsl(var(--destructive) / 0.14); color: hsl(var(--destructive)); font-size: 9px; font-weight: 800; padding: 2px 8px; border-radius: 999px; }
.so-undo { border: none; background: transparent; color: hsl(var(--destructive)); font-size: 16px; cursor: pointer; }
@media (max-width: 1100px) { .so-grid { grid-template-columns: 1fr; } }


/* ── Hasil Stock Opname ── */
.hasil-panel { margin: 0 16px 12px; padding: 12px; background: hsl(var(--muted)); border: 1px solid hsl(var(--border)); border-radius: 6px; }
.hasil-head { display: flex; justify-content: space-between; align-items: center; gap: 10px; margin-bottom: 10px; }
.hasil-title { font-size: 12px; font-weight: 800; color: hsl(var(--foreground)); }
.hasil-chips { display: flex; gap: 6px; }
.hasil-table th, .hasil-table td { padding: 9px 12px; }
.hasil-selisih, .hasil-status { display: inline-block; font-size: 10px; font-weight: 800; padding: 3px 9px; border-radius: 999px; white-space: nowrap; }
.hasil-ok { background: hsl(var(--success) / 0.14); color: hsl(var(--success)); }
.hasil-bad { background: hsl(var(--destructive) / 0.14); color: hsl(var(--destructive)); }
.hasil-selesai { background: hsl(var(--success) / 0.14); color: hsl(var(--success)); }
.hasil-draft { background: hsl(var(--muted)); color: hsl(var(--muted-foreground)); border: 1px solid hsl(var(--border)); }
.hasil-aksi { display: flex; gap: 4px; }
.hasil-detail-meta { font-size: 11px; color: hsl(var(--muted-foreground)); margin-bottom: 10px; }

.hasil-detail-cards { display: grid; grid-template-columns: repeat(4, 1fr); gap: 8px; margin-bottom: 10px; }
.hasil-detail-cards > div { background: hsl(var(--muted)); border: 1px solid hsl(var(--border)); border-radius: 6px; padding: 8px 10px; display: flex; flex-direction: column; gap: 3px; }
.hasil-detail-cards span { font-size: 9px; font-weight: 800; color: hsl(var(--muted-foreground)); }
.hasil-detail-cards b { font-size: 15px; color: hsl(var(--foreground)); }
.hasil-detail-note { font-size: 11px; color: hsl(var(--warning)); background: hsl(var(--warning) / 0.1); border: 1px solid hsl(var(--warning) / 0.3); border-radius: 6px; padding: 8px 10px; margin-bottom: 10px; }

.so-start { margin: 0 16px 16px; padding: 40px 16px; text-align: center; background: hsl(var(--card)); border: 1px dashed hsl(var(--warning)); border-radius: 8px; }
.so-start-icon { font-size: 34px; margin-bottom: 8px; }
.so-start-title { font-size: 14px; font-weight: 800; color: hsl(var(--foreground)); }
.so-start-desc { font-size: 11px; color: hsl(var(--muted-foreground)); margin: 6px 0 14px; }

.hasil-detail-note-green { background: hsl(var(--success) / 0.1); border-color: hsl(var(--success) / 0.3); color: hsl(var(--success)); }
.hasil-row-selisih { background: hsl(var(--destructive) / 0.08) !important; }
.aw-muted { color: hsl(var(--muted-foreground)); }

.hasil-row-new { background: hsl(var(--primary) / 0.08) !important; }
.hasil-baru { font-size: 9px; font-weight: 800; background: hsl(var(--primary)); color: hsl(var(--primary-foreground)); border-radius: 4px; padding: 2px 6px; margin-left: 6px; }
.hasil-detail-row td { background: hsl(var(--muted) / 0.4); }
.hasil-detail-title { font-size: 10px; font-weight: 800; color: hsl(var(--muted-foreground)); text-transform: uppercase; margin-bottom: 8px; }

.hasil-missing { margin-top: 12px; }
.hasil-missing-title { font-size: 11px; font-weight: 800; color: hsl(var(--destructive)); margin-bottom: 6px; }
.hasil-missing-wrap { max-height: 240px; overflow-y: auto; border: 1px solid hsl(var(--border)); border-radius: 6px; }
.hasil-missing-inline { margin-top: 10px; }

.hasil-head-actions { display: flex; gap: 6px; }
.hasil-check { accent-color: hsl(var(--primary)); width: 15px; height: 15px; cursor: pointer; }
.hasil-row-picked { background: hsl(var(--primary) / 0.06) !important; }
</style>
