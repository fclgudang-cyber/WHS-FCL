<script lang="ts" setup>
/**
 * Petty Cash — dashboard per permohonan desain (mirip halaman asli FCL):
 *  - Bar statistik (Saldo Awal / Pengeluaran / Pemasukan / Sisa Saldo) dengan border atas berwarna
 *  - Seksi "Periode": tab [Semua|Aktif|Selesai], kartu periode (klik = pilih periode),
 *    tombol Edit/Tutup/Hapus per kartu, [+ Periode] dan [+ Transaksi]
 *  - Tabel "Transaksi: <periode>" dengan saldo berjalan, badge status bayar,
 *    aksi Selesai Bayar / Edit / Hapus, footer TOTAL (OUT/IN),
 *    plus Export (CSV), Invoice, Dokumen dan Foto (jendela cetak)
 * Alur: buat periode dulu, baru transaksi bisa dicatat (semua transaksi terikat ke periode).
 */
import { computed, onMounted, ref } from 'vue';
import { useRouter } from 'vue-router';

import { Page } from '@vben/common-ui';
import {
  LucideCheck,
  LucideCirclePlay,
  LucideCircleStop,
  LucideDownload,
  LucideEye,
  LucideFilePenLine,
  LucideInbox,
  LucideList,
  LucideNotebookPen,
  LucidePencil,
  LucidePlus,
  LucidePrinter,
  LucideTrash2,
  LucideWallet,
} from '@vben/icons';

import { Form, Input, Modal, message, notification, Select, Textarea } from 'ant-design-vue';
import dayjs from 'dayjs';

import { fclDelete, fclInsert, fclPatch, fclQuery, formatRp } from '#/api/fcl';

// ── tipe & konstanta ──────────────────────────────────────────────────────────
interface PeriodRow {
  id: string;
  nama: string;
  tanggalMulai?: string | null;
  tanggalSelesai?: string | null;
  saldoAwal?: number | string | null;
  keterangan?: string | null;
  status?: string | null;
  createdBy?: string | null;
  createdAt?: string | null;
}
interface TxRow {
  id: string;
  periodId?: string | null;
  tanggal?: string | null;
  kategori?: string | null;
  keterangan?: string | null;
  nominal?: number | string | null;
  tipe?: string | null;
  buktiUrl?: string | null;
  statusBayar?: string | null;
  createdBy?: string | null;
  createdAt?: string | null;
}

const CATEGORY_COLORS = ['#22d3ee', '#34d399', '#fbbf24', '#f472b6', '#a78bfa', '#fb923c', '#4ade80', '#60a5fa'];

interface KategoriRow {
  id: string;
  kode?: string | null;
  nama?: string | null;
  urutan?: number | string | null;
  aktif?: string | null;
}
const kategoris = ref<KategoriRow[]>([]);
const KATEGORI_LAIN = '__LAINNYA__';
const router = useRouter();

const loading = ref(true);

const periods = ref<PeriodRow[]>([]);
const txs = ref<TxRow[]>([]);

const periodFilter = ref<'Semua' | 'Aktif' | 'Selesai'>('Aktif');
const search = ref('');
const onlyBelum = ref(false);
const activePeriodId = ref<string>('');

// ── util ─────────────────────────────────────────────────────────────────────
function toNum(v: unknown): number {
  if (v === null || v === undefined) return 0;
  if (typeof v === 'number') return Number.isFinite(v) ? v : 0;
  const s = String(v).trim();
  if (!s) return 0;
  // hapus pemisah ribuan (id-ID pakai titik), koma = desimal
  const n = Number(s.replace(/\./g, '').replace(',', '.'));
  return Number.isFinite(n) ? n : 0;
}

function fmtRp(n: number): string {
  return formatRp(n);
}

function fmtSignedRp(n: number): string {
  const abs = Math.abs(n);
  return (n < 0 ? '-Rp ' : '+Rp ') + abs.toLocaleString('id-ID', { maximumFractionDigits: 0 });
}

function fmtTanggalPendek(v?: string | null): string {
  const d = dayjs(v);
  return d.isValid() ? d.format('DD MMM YYYY') : '-';
}

function fmtTanggalDuaBaris(v?: string | null): [string, string] {
  const d = dayjs(v);
  return d.isValid() ? [d.format('DD MMM'), d.format('YYYY')] : ['-', ''];
}

function normalizeTipe(t?: string | null): 'Masuk' | 'Keluar' {
  const s = String(t ?? '').toLowerCase();
  if (['in', 'masuk', 'pemasukan', 'kredit'].includes(s)) return 'Masuk';
  return 'Keluar';
}

function isPeriodAktif(p?: PeriodRow | null): boolean {
  const s = String(p?.status ?? '').trim().toLowerCase();
  return s === '' || s === 'aktif';
}

function isTxBelumLunas(t?: TxRow | null): boolean {
  const s = String(t?.statusBayar ?? 'Belum Bayar').toLowerCase();
  return s !== 'lunas';
}

function catColor(cat?: string | null): string {
  const s = String(cat ?? 'Lainnya').trim().toLowerCase() || 'lainnya';
  let hash = 0;
  for (let i = 0; i < s.length; i++) hash = (hash * 31 + s.charCodeAt(i)) >>> 0;
  return CATEGORY_COLORS[hash % CATEGORY_COLORS.length];
}

function esc(s: unknown): string {
  return String(s ?? '')
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;');
}

// ── data turunan ─────────────────────────────────────────────────────────────
const activePeriod = computed(
  () => periods.value.find((p) => String(p.id) === String(activePeriodId.value)) ?? null,
);

const filteredPeriods = computed(() => {
  let list = [...periods.value];
  if (periodFilter.value === 'Aktif') list = list.filter(isPeriodAktif);
  else if (periodFilter.value === 'Selesai') list = list.filter((p) => !isPeriodAktif(p));
  else list = list;
  // Aktif dulu, lalu tanggal mulai terbaru
  return list.sort((a, b) => {
    const oa = isPeriodAktif(a) ? 0 : 1;
    const ob = isPeriodAktif(b) ? 0 : 1;
    if (oa !== ob) return oa - ob;
    return new Date(b.tanggalMulai ?? 0).getTime() - new Date(a.tanggalMulai ?? 0).getTime();
  });
});

const periodTxs = computed(() => {
  if (!activePeriodId.value) return [] as TxRow[];
  return txs.value
    .filter((t) => String(t.periodId) === String(activePeriodId.value))
    .slice()
    .sort((a, b) => new Date(a.tanggal ?? 0).getTime() - new Date(b.tanggal ?? 0).getTime());
});

const visibleTxs = computed(() => {
  let list = periodTxs.value;
  const q = search.value.trim().toLowerCase();
  if (q) {
    list = list.filter((t) =>
      [t.keterangan, t.kategori, t.createdBy, t.tipe]
        .some((v) => String(v ?? '').toLowerCase().includes(q)),
    );
  }
  if (onlyBelum.value) list = list.filter(isTxBelumLunas);
  return list;
});

const saldoAwal = computed(() => toNum(activePeriod.value?.saldoAwal));
const totalOut = computed(() =>
  periodTxs.value.filter((t) => normalizeTipe(t.tipe) === 'Keluar').reduce((s, t) => s + toNum(t.nominal), 0),
);
const totalIn = computed(() =>
  periodTxs.value.filter((t) => normalizeTipe(t.tipe) === 'Masuk').reduce((s, t) => s + toNum(t.nominal), 0),
);
const sisaSaldo = computed(() => saldoAwal.value + totalIn.value - totalOut.value);

const txRows = computed(() => {
  let running = saldoAwal.value;
  return visibleTxs.value.map((t, i) => {
    const nom = toNum(t.nominal);
    const tipe = normalizeTipe(t.tipe);
    if (tipe === 'Masuk') running += nom;
    else running -= nom;
    return { ...t, _no: i + 1, _tipe: tipe, _nomSigned: tipe === 'Masuk' ? nom : -nom, _saldo: running, _lunas: !isTxBelumLunas(t) };
  });
});

// ── load ──────────────────────────────────────────────────────────────────────
async function loadAll(silent = false) {
  if (!silent) loading.value = true;
  try {
    const [pRows, tRows, kRows] = await Promise.all([
      fclQuery<PeriodRow>('pettycashperiod', { select: '*', order: 'createdAt.desc', limit: 500 }),
      fclQuery<TxRow>('pettycash', { select: '*', order: 'tanggal.asc,createdAt.asc', limit: 5000 }),
      fclQuery<KategoriRow>('pettycashkategori', { select: '*', order: 'urutan.asc', limit: 500 }),
    ]);
    periods.value = pRows;
    txs.value = tRows;
    kategoris.value = kRows;
    // pertahankan pilihan jika masih valid; jika tidak, pilih periode Aktif pertama
    const tetapValid = activePeriodId.value && pRows.some((p) => String(p.id) === String(activePeriodId.value));
    if (!tetapValid) {
      const firstAktif = pRows.find(isPeriodAktif) ?? pRows[0];
      activePeriodId.value = firstAktif ? String(firstAktif.id) : '';
    }
  } catch (err: any) {
    console.error('[petty-cash] load gagal', err);
    notification.error({ message: 'Petty Cash', description: err?.message || 'Gagal memuat data' });
  } finally {
    if (!silent) loading.value = false;
  }
}

function selectPeriod(id: string) {
  activePeriodId.value = String(id);
}

// ── modal periode ─────────────────────────────────────────────────────────────
const periodModalOpen = ref(false);
const periodSaving = ref(false);
const periodEditing = ref<PeriodRow | null>(null);
const periodForm = ref({
  nama: '',
  tanggalMulai: '',
  tanggalSelesai: '',
  saldoAwal: undefined as number | undefined,
  status: 'Aktif',
  keterangan: '',
});

function openPeriodModal(p?: PeriodRow | null) {
  periodEditing.value = p ?? null;
  periodForm.value = p
    ? {
        nama: p.nama ?? '',
        tanggalMulai: (p.tanggalMulai ?? '').slice(0, 10),
        tanggalSelesai: (p.tanggalSelesai ?? '').slice(0, 10),
        saldoAwal: toNum(p.saldoAwal),
        status: p.status === 'Selesai' ? 'Selesai' : 'Aktif',
        keterangan: p.keterangan ?? '',
      }
    : { nama: '', tanggalMulai: '', tanggalSelesai: '', saldoAwal: undefined, status: 'Aktif', keterangan: '' };
  periodModalOpen.value = true;
}

async function savePeriod() {
  if (!periodForm.value.nama.trim()) { message.warning('Nama periode wajib diisi'); return; }
  if (periodForm.value.saldoAwal === undefined || periodForm.value.saldoAwal === null) { message.warning('Saldo awal wajib diisi'); return; }
  periodSaving.value = true;
  const payload: Record<string, any> = {
    nama: periodForm.value.nama.trim(),
    tanggalMulai: periodForm.value.tanggalMulai || null,
    tanggalSelesai: periodForm.value.tanggalSelesai || null,
    saldoAwal: String(periodForm.value.saldoAwal),
    status: periodForm.value.status,
    keterangan: periodForm.value.keterangan,
  };
  try {
    if (periodEditing.value) {
      await fclPatch('pettycashperiod', periodEditing.value.id, payload);
      notification.success({ message: 'Periode', description: 'Periode berhasil diperbarui' });
    } else {
      const created = (await fclInsert('pettycashperiod', [payload])) as any;
      const newId = Array.isArray(created) ? created?.[0]?.id : created?.id;
      notification.success({ message: 'Periode', description: 'Periode berhasil ditambahkan' });
      // langsung pilih periode yang baru dibuat (mirip perilaku app asli)
      if (newId) {
        activePeriodId.value = String(newId);
        periodFilter.value = 'Semua';
      }
    }
    periodModalOpen.value = false;
    await loadAll(true);
  } catch (err: any) {
    notification.error({ message: 'Periode', description: err?.message || 'Gagal menyimpan periode' });
  } finally {
    periodSaving.value = false;
  }
}

async function closePeriod(p: PeriodRow) {
  Modal.confirm({
    title: 'Tutup periode?',
    content: `Status periode "${p.nama}" akan berubah menjadi Selesai. Transaksi tidak bisa ditambah lagi.`,
    okText: 'Tutup',
    okType: 'danger',
    onOk: async () => {
      await fclPatch('pettycashperiod', p.id, { status: 'Selesai' });
      notification.success({ message: 'Periode ditutup' });
      await loadAll(true);
    },
  });
}

async function reopenPeriod(p: PeriodRow) {
  await fclPatch('pettycashperiod', p.id, { status: 'Aktif' });
  notification.success({ message: 'Periode dibuka kembali' });
  await loadAll(true);
}

async function deletePeriod(p: PeriodRow) {
  const count = txs.value.filter((t) => String(t.periodId) === String(p.id)).length;
  Modal.confirm({
    title: 'Hapus periode?',
    content: `Hapus periode "${p.nama}" beserta ${count} transaksi di dalamnya? Data tidak bisa dikembalikan.`,
    okText: 'Hapus',
    okType: 'danger',
    onOk: async () => {
      await fclDelete('pettycashperiod', p.id);
      const orphanTx = txs.value.filter((t) => String(t.periodId) === String(p.id));
      for (const t of orphanTx) await fclDelete('pettycash', t.id);
      if (activePeriodId.value === String(p.id)) activePeriodId.value = '';
      notification.success({ message: 'Periode dihapus' });
      await loadAll(true);
    },
  });
}

// ── modal transaksi ───────────────────────────────────────────────────────────
const trxModalOpen = ref(false);
const trxSaving = ref(false);
const trxEditing = ref<TxRow | null>(null);
const trxForm = ref({
  tanggal: '',
  tipe: 'Keluar' as 'Masuk' | 'Keluar',
  kategori: '',
  keterangan: '',
  nominal: undefined as number | undefined,
  statusBayar: 'Belum Bayar',
  periodId: '',
  buktiUrl: '',
});

const kategoriOptions = computed(() => [
  ...kategoris.value
    .filter((k) => String(k.aktif ?? 'Ya') !== 'Tidak')
    .map((k) => {
      const nama = String(k.nama ?? '').trim();
      return { value: nama, label: k.kode ? `${String(k.kode)} — ${nama}` : nama };
    })
    .filter((o) => !!o.value),
  { value: KATEGORI_LAIN, label: 'Lainnya...' },
]);
const kategoriCustom = ref('');
const isKategoriLain = computed(() => trxForm.value.kategori === KATEGORI_LAIN);

function kategoriValueFor(v?: string | null): string {
  const s = String(v ?? '').trim();
  return s && kategoris.value.some((k) => String(k.nama ?? '').trim() === s) ? s : (s ? KATEGORI_LAIN : '');
}

const trxTanggalModel = computed({
  get: () => (trxForm.value.tanggal ? dayjs(trxForm.value.tanggal) : undefined),
  set: (v: any) => {
    // antd dengan value-format="YYYY-MM-DD" mengirim STRING, tanpa itu mengirim dayjs — tangani keduanya
    if (!v) {
      trxForm.value.tanggal = '';
      return;
    }
    const s = typeof v === 'string' ? v : (v.isValid?.() ? v.format('YYYY-MM-DD') : '');
    trxForm.value.tanggal = /^\d{4}-\d{2}-\d{2}/.test(s) ? s.slice(0, 10) : (s ? s : '');
  },
});

function openTrxModal(t?: TxRow | null) {
  if (!t && !activePeriod.value) {
    message.warning('Buat & pilih periode dulu sebelum menambah transaksi');
    return;
  }
  trxEditing.value = t ?? null;
  const katValue = kategoriValueFor(t?.kategori);
  kategoriCustom.value = katValue === KATEGORI_LAIN ? String(t?.kategori ?? '') : '';
  trxForm.value = t
    ? {
        tanggal: (t.tanggal ?? '').slice(0, 10),
        tipe: normalizeTipe(t.tipe),
        kategori: katValue,
        keterangan: t.keterangan ?? '',
        nominal: toNum(t.nominal),
        statusBayar: t.statusBayar ?? 'Belum Bayar',
        periodId: String(t.periodId ?? activePeriodId.value),
        buktiUrl: t.buktiUrl ?? '',
      }
    : {
        tanggal: dayjs().format('YYYY-MM-DD'),
        tipe: 'Keluar',
        kategori: '',
        keterangan: '',
        nominal: undefined,
        statusBayar: 'Belum Bayar',
        periodId: String(activePeriodId.value),
        buktiUrl: '',
      };
  trxModalOpen.value = true;
}

async function saveTrx() {
  if (!trxForm.value.tanggal) { message.warning('Tanggal wajib diisi'); return; }
  if (trxForm.value.nominal === undefined || trxForm.value.nominal === null) { message.warning('Nominal wajib diisi'); return; }
  if (!trxForm.value.periodId) { message.warning('Pilih periode untuk transaksi ini'); return; }
  const kategori = (isKategoriLain.value ? kategoriCustom.value : trxForm.value.kategori).trim();
  if (!kategori) { message.warning('Pilih kategori transaksi'); return; }
  trxSaving.value = true;
  const payload: Record<string, any> = {
    tanggal: trxForm.value.tanggal,
    tipe: trxForm.value.tipe,
    kategori,
    keterangan: trxForm.value.keterangan,
    nominal: String(trxForm.value.nominal),
    statusBayar: trxForm.value.statusBayar,
    periodId: trxForm.value.periodId,
    buktiUrl: trxForm.value.buktiUrl,
  };
  try {
    if (trxEditing.value) {
      await fclPatch('pettycash', trxEditing.value.id, payload);
      notification.success({ message: 'Petty Cash', description: 'Transaksi diperbarui' });
    } else {
      await fclInsert('pettycash', [payload]);
      notification.success({ message: 'Petty Cash', description: 'Transaksi ditambahkan' });
    }
    trxModalOpen.value = false;
    await loadAll(true);
  } catch (err: any) {
    notification.error({ message: 'Petty Cash', description: err?.message || 'Gagal menyimpan transaksi' });
  } finally {
    trxSaving.value = false;
  }
}

async function deleteTrx(t: TxRow) {
  Modal.confirm({
    title: 'Hapus transaksi?',
    content: `Hapus "${t.keterangan || t.kategori || 'transaksi'}" sebesar ${fmtSignedRp(-toNum(t.nominal))}?`,
    okText: 'Hapus',
    okType: 'danger',
    onOk: async () => {
      await fclDelete('pettycash', t.id);
      notification.success({ message: 'Transaksi dihapus' });
      await loadAll(true);
    },
  });
}

async function markLunas(t: TxRow) {
  await fclPatch('pettycash', t.id, { statusBayar: 'Lunas' });
  notification.success({ message: 'Transaksi ditandai Lunas' });
  await loadAll(true);
}

// ── bukti & cetak ─────────────────────────────────────────────────────────────
const buktiModalOpen = ref(false);
const buktiTarget = ref<TxRow | null>(null);
const buktiInput = ref('');
const buktiMode = ref<'link' | 'upload'>('link');
const buktiFile = ref<File | null>(null);
const buktiPreview = ref('');
const buktiUploading = ref(false);

function openBuktiModal(t: TxRow) {
  buktiTarget.value = t;
  buktiInput.value = t.buktiUrl ?? '';
  buktiFile.value = null;
  buktiMode.value = String(t.buktiUrl ?? '').startsWith('/api/files/') ? 'upload' : 'link';
  buktiPreview.value = buktiMode.value === 'upload' ? String(t.buktiUrl ?? '') : '';
  buktiModalOpen.value = true;
}

function pickBuktiFile(e: Event) {
  const file = (e.target as HTMLInputElement).files?.[0] ?? null;
  buktiFile.value = file;
  buktiPreview.value = file ? URL.createObjectURL(file) : '';
  if (file && file.size > 12 * 1024 * 1024) {
    message.warning('Ukuran file maksimal 12 MB');
    buktiFile.value = null;
    buktiPreview.value = '';
  }
}

async function saveBukti() {
  if (!buktiTarget.value) return;
  let url = buktiTarget.value.buktiUrl ?? '';
  if (buktiMode.value === 'upload') {
    if (!buktiFile.value && !url) { message.warning('Pilih file bukti terlebih dahulu'); return; }
    if (buktiFile.value) {
      buktiUploading.value = true;
      try {
        const data = new FormData();
        data.append('file', buktiFile.value);
        const res = await fetch('/api/file/upload', { method: 'POST', body: data });
        const out = await res.json();
        if (!out?.success || !out?.url) throw new Error(out?.message || 'Upload gagal');
        url = String(out.url);
      } catch (err: any) {
        notification.error({ message: 'Upload Bukti', description: err?.message || 'Gagal mengunggah file' });
        return;
      } finally {
        buktiUploading.value = false;
      }
    }
  } else {
    url = buktiInput.value.trim();
  }
  await fclPatch('pettycash', buktiTarget.value.id, { buktiUrl: url });
  notification.success({ message: 'Bukti transaksi disimpan' });
  buktiModalOpen.value = false;
  await loadAll(true);
}

// Pratinjau foto bukti langsung di halaman (tanpa membuka tab/link baru).
// Untuk tautan Google Drive, URL "view" diubah ke URL gambar langsung agar <img> bisa merender.
const viewBuktiOpen = ref(false);
const viewBuktiSrc = ref('');
const viewBuktiCandidates = ref<string[]>([]);
const viewBuktiOriginal = ref('');
const viewBuktiLabel = ref('');
const viewBuktiIsPdf = ref(false);
const viewBuktiFailed = ref(false);

function driveFileId(url: string): string {
  try {
    const u = new URL(url);
    if (!/drive\.google\.com$/.test(u.hostname)) return '';
    const m = u.pathname.match(/\/file\/d\/([^/]+)/);
    return m ? m[1] : u.searchParams.get('id') ?? '';
  } catch {
    return '';
  }
}

function viewBukti(t: TxRow) {
  const url = String(t.buktiUrl ?? '').trim();
  if (!url) {
    message.warning('Transaksi ini belum memiliki foto bukti');
    return;
  }
  const id = driveFileId(url);
  const candidates = id
    ? [`https://lh3.googleusercontent.com/d/${id}`, `https://drive.google.com/uc?export=view&id=${id}`, url]
    : [url];
  viewBuktiCandidates.value = candidates;
  viewBuktiSrc.value = candidates[0];
  viewBuktiOriginal.value = url;
  viewBuktiLabel.value = String(t.keterangan || t.tanggal || '').slice(0, 48);
  viewBuktiIsPdf.value = /\.pdf(\?|$)/i.test(url);
  viewBuktiFailed.value = false;
  viewBuktiOpen.value = true;
}

function onViewBuktiError() {
  const i = viewBuktiCandidates.value.indexOf(viewBuktiSrc.value);
  if (i >= 0 && i < viewBuktiCandidates.value.length - 1) {
    viewBuktiSrc.value = viewBuktiCandidates.value[i + 1];
  } else {
    viewBuktiFailed.value = true;
  }
}

// Export CSV untuk periode aktif (termasuk saldo berjalan)
function exportCsv() {
  if (!activePeriod.value) { message.warning('Pilih periode dulu'); return; }
  const period = activePeriod.value;
  const header = ['No', 'Tanggal', 'Tipe', 'Kategori', 'Keterangan', 'Nominal', 'Saldo', 'Status', 'Bukti', 'Input By'];
  const rows = txRows.value.map((r) => [
    r._no,
    (r.tanggal ?? '').slice(0, 10),
    r._tipe,
    r.kategori ?? '',
    r.keterangan ?? '',
    String(r._nomSigned),
    String(r._saldo),
    r._lunas ? 'LUNAS' : 'BELUM BAYAR',
    r.buktiUrl ?? '',
    r.createdBy ?? '',
  ]);
  const total = [
    '', '', '', '', 'TOTAL',
    String(txRows.value.filter((r) => r._tipe === 'Keluar').reduce((s, r) => s + toNum(r.nominal), 0) * -1),
    String(sisaSaldo.value), '', '', '',
  ];
  const lines = [header, ...rows, ['', '', '', '', '', '', '', '', '', ''], total]
    .map((cols) => cols.map((c) => `"${String(c ?? '').replace(/"/g, '""')}"`).join(','))
    .join('\n');
  const blob = new Blob(['﻿' + lines], { type: 'text/csv;charset=utf-8;' });
  const url = URL.createObjectURL(blob);
  const a = document.createElement('a');
  a.href = url;
  a.download = `pettycash-${period.nama.replace(/\s+/g, '_')}.csv`;
  a.click();
  URL.revokeObjectURL(url);
}

// Jendela cetak generik
function openPrintWindow(title: string, bodyHtml: string) {
  const w = window.open('', '_blank', 'width=900,height=700');
  if (!w) { message.warning('Popup diblokir browser. Izinkan popup untuk mencetak.'); return; }
  w.document.write(`<!doctype html><html><head><meta charset="utf-8"><title>${esc(title)}</title>
<style>
  body { font-family: 'Segoe UI', Arial, sans-serif; color:#111; margin: 24px; font-size: 12px; }
  h1 { font-size: 18px; margin: 0 0 4px; }
  .sub { color:#555; font-size: 11px; margin-bottom: 16px; }
  table { width:100%; border-collapse: collapse; margin-bottom: 16px; }
  th, td { border: 1px solid #d1d5db; padding: 5px 8px; font-size: 11px; }
  th { background: #f3f4f6; text-align: left; }
  td.r, th.r { text-align: right; }
  .foot { display:flex; justify-content:space-between; margin-top: 28px; gap: 40px; }
  .sign { flex:1; text-align:center; font-size: 11px; }
  .sign .line { border-top: 1px solid #111; margin: 42px 8px 4px; }
  .total-row td { font-weight: 700; background:#fafafa; }
  @media print { @page { size: A4; margin: 14mm; } }

.fcl-pc-bukti-tabs { display: flex; gap: 6px; margin-bottom: 10px; }
.fcl-pc-bukti-tabs button { border: 1px solid hsl(var(--border)); background: hsl(var(--muted)); color: hsl(var(--muted-foreground)); font-size: 11px; font-weight: 700; padding: 5px 12px; border-radius: 6px; cursor: pointer; }
.fcl-pc-bukti-tabs button.active { background: hsl(var(--primary)); border-color: hsl(var(--primary)); color: hsl(var(--primary-foreground)); }
.fcl-pc-file { font-size: 11px; width: 100%; color: hsl(var(--muted-foreground)); }
.fcl-pc-bukti-preview { margin-top: 10px; max-height: 240px; overflow: hidden; border: 1px solid hsl(var(--border)); border-radius: 6px; }
.fcl-pc-bukti-preview img { max-width: 100%; max-height: 240px; object-fit: contain; display: block; }
</style></head><body>${bodyHtml}</body></html>`);
  w.document.close();
  w.focus();
  setTimeout(() => { try { w.print(); } catch { /* abaikan */ } }, 350);
}

function printDokumen() {
  const period = activePeriod.value;
  if (!period) { message.warning('Pilih periode dulu'); return; }
  if (!periodTxs.value.length) { message.warning('Belum ada transaksi di periode ini'); return; }
  let running = toNum(period.saldoAwal);
  let totalOut = 0, totalIn = 0;
  const rows = periodTxs.value.map((t, i) => {
    const nom = toNum(t.nominal);
    const tipe = normalizeTipe(t.tipe);
    if (tipe === 'Masuk') { running += nom; totalIn += nom; } else { running -= nom; totalOut += nom; }
    const nomStr = tipe === 'Masuk'
      ? `<span style="color:#16a34a;font-weight:700">+ Rp ${nom.toLocaleString('id-ID')}</span>`
      : `<span style="color:#dc2626;font-weight:700">- Rp ${nom.toLocaleString('id-ID')}</span>`;
    return `<tr><td>${i + 1}</td><td>${esc((t.tanggal ?? '').slice(0, 10))}</td><td>${esc(tipe)}</td><td>${esc(t.kategori)}</td><td>${esc(t.keterangan)}</td><td class="r">${nomStr}</td><td class="r">Rp ${running.toLocaleString('id-ID')}</td><td>${t.statusBayar === 'Lunas' ? 'LUNAS' : 'BELUM BAYAR'}</td></tr>`;
  }).join('');
  const body = `
    <h1>PERTANGGUNGJAWABAN PETTY CASH</h1>
    <div class="sub">${esc(period.nama)} &nbsp;•&nbsp; ${esc(fmtTanggalPendek(period.tanggalMulai))} s/d ${esc(fmtTanggalPendek(period.tanggalSelesai))} &nbsp;•&nbsp; Saldo Awal: <b>Rp ${toNum(period.saldoAwal).toLocaleString('id-ID')}</b></div>
    <table>
      <thead><tr><th>No</th><th>Tanggal</th><th>Tipe</th><th>Kategori</th><th>Keterangan</th><th class="r">Nominal</th><th class="r">Saldo</th><th>Status</th></tr></thead>
      <tbody>${rows}</tbody>
      <tfoot>
        <tr class="total-row"><td colspan="5" class="r">TOTAL PENGELUARAN</td><td class="r">- Rp ${totalOut.toLocaleString('id-ID')}</td><td class="r">Sisa: Rp ${(toNum(period.saldoAwal) + totalIn - totalOut).toLocaleString('id-ID')}</td><td></td></tr>
      </tfoot>
    </table>
    <div class="foot">
      <div class="sign"><div class="line"></div>Dibuat oleh (Gudang)</div>
      <div class="sign"><div class="line"></div>Diperiksa oleh</div>
      <div class="sign"><div class="line"></div>Disetujui oleh</div>
    </div>`;
  openPrintWindow(`Dokumen Petty Cash - ${period.nama}`, body);
}

function printFoto() {
  const period = activePeriod.value;
  if (!period) { message.warning('Pilih periode dulu'); return; }
  if (!periodTxs.value.length) { message.warning('Belum ada transaksi di periode ini'); return; }
  const grouped = new Map<string, TxRow[]>();
  periodTxs.value.forEach((t) => {
    const k = (t.kategori || 'Lainnya').trim() || 'Lainnya';
    if (!grouped.has(k)) grouped.set(k, []);
    grouped.get(k)!.push(t);
  });
  const blocks = [...grouped.entries()].map(([kat, items]) => `
    <h2 style="font-size:13px;margin:18px 0 8px;border-bottom:2px solid #111;padding-bottom:4px;">${esc(kat)}</h2>
    <table><thead><tr><th>No</th><th>Tanggal</th><th>Keterangan</th><th class="r">Nominal</th><th>Foto</th></tr></thead>
    <tbody>${items.map((t, i) => `<tr>
      <td>${i + 1}</td><td>${esc((t.tanggal ?? '').slice(0, 10))}</td><td>${esc(t.keterangan)}</td><td class="r">Rp ${toNum(t.nominal).toLocaleString('id-ID')}</td>
      <td>${t.buktiUrl ? `<img src="${esc(t.buktiUrl)}" alt="bukti" style="max-height:90px;max-width:120px;object-fit:contain;border:1px solid #ddd;border-radius:4px;" onerror="this.onerror=null;this.alt='(gagal dimuat)'"/>` : '<span style="color:#b91c1c">tanpa foto</span>'}</td>
    </tr>`).join('')}</tbody></table>`).join('');
  openPrintWindow(`Bukti Foto Petty Cash - ${period.nama}`, `
    <h1>BUKTI FOTO PETTY CASH</h1>
    <div class="sub">${esc(period.nama)} &nbsp;•&nbsp; ${esc(fmtTanggalPendek(period.tanggalMulai))} s/d ${esc(fmtTanggalPendek(period.tanggalSelesai))}</div>
    ${blocks}`);
}

// Invoice: modal pilih transaksi → cetak
const invoiceModalOpen = ref(false);
const invoiceTitle = ref('PT FCL Internasional Indonesia');
const invoiceNotes = ref('');
const invoiceSelected = ref<Set<string>>(new Set());

const invoiceCategories = computed(() =>
  [...new Set(periodTxs.value.map((t) => (t.kategori || 'Lainnya').trim() || 'Lainnya'))].sort(),
);

const invoiceTxs = computed(() => {
  const filtered = periodTxs.value.filter((t) => invoiceSelected.value.has(String(t.id)));
  return filtered;
});

const invoiceTotal = computed(() =>
  invoiceTxs.value.reduce((s, t) => s + (normalizeTipe(t.tipe) === 'Masuk' ? -toNum(t.nominal) : toNum(t.nominal)), 0),
);

function openInvoice() {
  if (!activePeriod.value) { message.warning('Pilih periode dulu'); return; }
  if (!periodTxs.value.length) { message.warning('Belum ada transaksi di periode ini'); return; }
  invoiceSelected.value = new Set(periodTxs.value.map((t) => String(t.id)));
  invoiceModalOpen.value = true;
}

function toggleInvoiceTx(id: string) {
  const next = new Set(invoiceSelected.value);
  if (next.has(id)) next.delete(id);
  else next.add(id);
  invoiceSelected.value = next;
}

function toggleInvoiceAll(check: boolean) {
  invoiceSelected.value = check ? new Set(periodTxs.value.map((t) => String(t.id))) : new Set();
}

function generateInvoice() {
  const period = activePeriod.value!;
  const chosen = periodTxs.value.filter((t) => invoiceSelected.value.has(String(t.id)));
  if (!chosen.length) { message.warning('Pilih minimal 1 transaksi'); return; }
  let total = 0;
  const rows = chosen.map((t, i) => {
    const nom = toNum(t.nominal);
    const tipe = normalizeTipe(t.tipe);
    if (tipe === 'Keluar') total += nom;
    return `<tr><td>${i + 1}</td><td>${esc((t.tanggal ?? '').slice(0, 10))}</td><td>${esc(t.kategori)}</td><td>${esc(t.keterangan)}</td><td class="r">Rp ${nom.toLocaleString('id-ID')}</td></tr>`;
  }).join('');
  const body = `
    <h1>${esc(invoiceTitle.value || 'INVOICE')}</h1>
    <div class="sub">Invoice Petty Cash • ${esc(period.nama)} &nbsp;•&nbsp; ${esc(fmtTanggalPendek(period.tanggalMulai))} s/d ${esc(fmtTanggalPendek(period.tanggalSelesai))}${invoiceNotes.value ? '<br/>' + esc(invoiceNotes.value) : ''}</div>
    <table>
      <thead><tr><th>No</th><th>Tanggal</th><th>Kategori</th><th>Keterangan</th><th class="r">Nominal</th></tr></thead>
      <tbody>${rows}</tbody>
      <tfoot><tr class="total-row"><td colspan="4" class="r">TOTAL</td><td class="r">Rp ${total.toLocaleString('id-ID')}</td></tr></tfoot>
    </table>`;
  openPrintWindow(`Invoice Petty Cash - ${period.nama}`, body);
  invoiceModalOpen.value = false;
}

onMounted(loadAll);
</script>

<template>
  <Page auto-content-height>
    <!-- ══════════ BELUM ADA PERIODE → hero create-first ══════════ -->
    <div v-if="!loading && periods.length === 0" class="fcl-pc-hero">
      <div class="fcl-pc-hero-icon"><LucideWallet :size="44" /></div>
      <div class="fcl-pc-hero-title">Belum ada Periode Petty Cash</div>
      <div class="fcl-pc-hero-desc">
        Alur kerja petty cash: <b>buat periode dulu</b> (mis. “16 November–22 November 2026” dengan saldo awal),
        baru setelah itu <b>tambah transaksi</b> ke periode tersebut.
      </div>
      <button class="fcl-pc-btn fcl-pc-btn-primary fcl-pc-btn-lg" @click="openPeriodModal()">
        <LucidePlus :size="16" /> Buat Periode Baru
      </button>
    </div>

    <!-- ══════════ DASHBOARD PETTY CASH ══════════ -->
    <div v-else class="fcl-pc">
      <!-- ── Bar statistik ── -->
      <div class="fcl-pc-stats">
        <div class="fcl-pc-stat fcl-pc-stat-green">
          <div class="fcl-pc-stat-label">SALDO AWAL</div>
          <div class="fcl-pc-stat-value fcl-pc-tone-green">{{ fmtRp(saldoAwal) }}</div>
        </div>
        <div class="fcl-pc-stat fcl-pc-stat-red">
          <div class="fcl-pc-stat-label">PENGELUARAN</div>
          <div class="fcl-pc-stat-value fcl-pc-tone-red">{{ fmtRp(totalOut) }}</div>
        </div>
        <div class="fcl-pc-stat fcl-pc-stat-teal">
          <div class="fcl-pc-stat-label">PEMASUKAN</div>
          <div class="fcl-pc-stat-value fcl-pc-tone-teal">{{ fmtRp(totalIn) }}</div>
        </div>
        <div class="fcl-pc-stat fcl-pc-stat-amber">
          <div class="fcl-pc-stat-label">SISA SALDO</div>
          <div class="fcl-pc-stat-value fcl-pc-tone-amber">{{ fmtRp(sisaSaldo) }}</div>
        </div>
      </div>

      <!-- ── Seksi Periode ── -->
      <div class="fcl-pc-card">
        <div class="fcl-pc-card-head">
          <div class="fcl-pc-card-title">Periode</div>
          <div class="fcl-pc-card-actions">
            <button class="fcl-pc-btn fcl-pc-btn-ghost" @click="openPeriodModal()">
              <LucidePlus :size="13" /> Periode
            </button>
            <button class="fcl-pc-btn fcl-pc-btn-primary" @click="openTrxModal()">
              <LucidePlus :size="13" /> Transaksi
            </button>
          </div>
        </div>

        <!-- Tab filter -->
        <div class="fcl-pc-tabs">
          <button
            v-for="tab in (['Semua', 'Aktif', 'Selesai'] as const)"
            :key="tab"
            class="fcl-pc-tab"
            :class="{ 'fcl-pc-tab-active': periodFilter === tab }"
            @click="periodFilter = tab"
          >
            {{ tab }}
          </button>
        </div>

        <!-- Kartu periode -->
        <div class="fcl-pc-period-list">
          <div
            v-for="p in filteredPeriods"
            :key="p.id"
            class="fcl-pc-period-card"
            :class="{ 'fcl-pc-period-card-active': String(p.id) === String(activePeriodId) }"
            @click="selectPeriod(p.id)"
          >
            <div class="fcl-pc-period-card-top">
              <div class="fcl-pc-period-nama">{{ p.nama }}</div>
              <span class="fcl-pc-badge" :class="isPeriodAktif(p) ? 'fcl-pc-badge-green' : 'fcl-pc-badge-gray'">
                {{ isPeriodAktif(p) ? 'Aktif' : 'Selesai' }}
              </span>
            </div>
            <div class="fcl-pc-period-date">
              {{ fmtTanggalPendek(p.tanggalMulai) }} — {{ fmtTanggalPendek(p.tanggalSelesai) }}
            </div>
            <div class="fcl-pc-period-saldo">Saldo Awal: <b>{{ fmtRp(toNum(p.saldoAwal)) }}</b></div>
            <div class="fcl-pc-period-actions">
              <button class="fcl-pc-mini-btn" @click.stop="openPeriodModal(p)"><LucidePencil :size="11" /> Edit</button>
              <button v-if="isPeriodAktif(p)" class="fcl-pc-mini-btn" @click.stop="closePeriod(p)">
                <LucideCircleStop :size="11" /> Tutup
              </button>
              <button v-else class="fcl-pc-mini-btn fcl-pc-mini-btn-green" @click.stop="reopenPeriod(p)">
                <LucideCirclePlay :size="11" /> Buka
              </button>
              <button class="fcl-pc-mini-btn fcl-pc-mini-btn-red" @click.stop="deletePeriod(p)"><LucideTrash2 :size="11" /> Hapus</button>
            </div>
          </div>
          <div v-if="!filteredPeriods.length" class="fcl-pc-empty-line">
            Tidak ada periode {{ periodFilter === 'Semua' ? '' : periodFilter === 'Aktif' ? 'aktif ' : 'selesai ' }}di filter ini.
          </div>
        </div>
      </div>

      <!-- ── Panel Transaksi ── -->
      <div class="fcl-pc-card fcl-pc-trx-card">
        <div class="fcl-pc-card-head fcl-pc-trx-head">
          <div class="fcl-pc-trx-heading">
            <div class="fcl-pc-card-title">Transaksi: {{ activePeriod?.nama ?? '-' }}</div>
            <div v-if="activePeriod" class="fcl-pc-trx-sub">
              {{ fmtTanggalPendek(activePeriod.tanggalMulai) }} s/d {{ fmtTanggalPendek(activePeriod.tanggalSelesai) }}
              &nbsp;|&nbsp; Saldo Awal: <b>{{ fmtRp(saldoAwal) }}</b>
            </div>
          </div>
          <div class="fcl-pc-trx-actions">
            <div class="fcl-pc-search">
              
              <input v-model="search" type="text" placeholder="Cari..." class="fcl-pc-search-input" />
            </div>
            <button
              class="fcl-pc-btn fcl-pc-btn-ghost fcl-pc-btn-sm"
              :class="{ 'fcl-pc-btn-toggled': onlyBelum }"
              @click="onlyBelum = !onlyBelum"
            >
              {{ onlyBelum ? 'Menampilkan: Belum Bayar' : 'Belum Bayar' }}
            </button>
            <button class="fcl-pc-btn fcl-pc-btn-ghost fcl-pc-btn-sm" @click="router.push('/fcl/keuangan/pettyCashKategori')"><LucideList :size="13" /> Kategori</button>
            <button class="fcl-pc-btn fcl-pc-btn-ghost fcl-pc-btn-sm" @click="exportCsv"><LucideDownload :size="13" /> Export</button>
            <button class="fcl-pc-btn fcl-pc-btn-info fcl-pc-btn-sm" @click="openInvoice"><LucideFilePenLine :size="13" /> Invoice</button>
            <button class="fcl-pc-btn fcl-pc-btn-teal fcl-pc-btn-sm" @click="printDokumen"><LucideNotebookPen :size="13" /> Dokumen</button>
            <button class="fcl-pc-btn fcl-pc-btn-orange fcl-pc-btn-sm" @click="printFoto"><LucidePrinter :size="13" /> Foto</button>
          </div>
        </div>

        <div class="fcl-pc-table-wrap">
          <table class="fcl-pc-table">
            <thead>
              <tr>
                <th class="c-no">NO</th>
                <th>TANGGAL</th>
                <th>TIPE</th>
                <th>KATEGORI</th>
                <th>KETERANGAN</th>
                <th class="r">NOMINAL</th>
                <th class="r">SALDO</th>
                <th class="c">STATUS</th>
                <th>BUKTI</th>
                <th>BY</th>
                <th class="c">AKSI</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="row in txRows" :key="row.id">
                <td class="c-no">{{ row._no }}</td>
                <td>
                  <div class="fcl-pc-tanggal">
                    <span>{{ fmtTanggalDuaBaris(row.tanggal)[0] }}</span>
                    <span class="fcl-pc-tanggal-year">{{ fmtTanggalDuaBaris(row.tanggal)[1] }}</span>
                  </div>
                </td>
                <td>
                  <span class="fcl-pc-tipe" :class="row._tipe === 'Masuk' ? 'fcl-pc-tipe-in' : 'fcl-pc-tipe-out'">{{ row._tipe }}</span>
                </td>
                <td>
                  <span class="fcl-pc-kat-chip" :style="{ '--chip': catColor(row.kategori) }">{{ (row.kategori || 'LAINNYA').toUpperCase() }}</span>
                </td>
                <td class="fcl-pc-ket">{{ row.keterangan || '-' }}</td>
                <td class="r">
                  <span class="fcl-pc-nominal" :class="row._tipe === 'Masuk' ? 'fcl-pc-tone-green' : 'fcl-pc-tone-red'">
                    {{ fmtSignedRp(row._nomSigned) }}
                  </span>
                </td>
                <td class="r"><span class="fcl-pc-saldo">{{ fmtRp(row._saldo) }}</span></td>
                <td class="c">
                  <span class="fcl-pc-statuspill" :class="row._lunas ? 'fcl-pc-pill-lunas' : 'fcl-pc-pill-belum'">
                    {{ row._lunas ? 'LUNAS' : 'BELUM BAYAR' }}
                  </span>
                </td>
                <td>
                  <div class="fcl-pc-bukti">
                    <template v-if="row.buktiUrl">
                      <button class="fcl-pc-bukti-link" @click="viewBukti(row)"><LucideEye :size="12" /> Lihat Bukti</button>
                      <button class="fcl-pc-mini-btn" @click="openBuktiModal(row)">Ganti</button>
                    </template>
                    <button v-else class="fcl-pc-bukti-upload" @click="openBuktiModal(row)"><LucidePlus :size="12" /> Upload</button>
                  </div>
                </td>
                <td class="fcl-pc-by">{{ row.createdBy || '-' }}</td>
                <td class="c">
                  <div class="fcl-pc-aksi">
                    <button v-if="!row._lunas" class="fcl-pc-btn fcl-pc-btn-orange fcl-pc-btn-xs" @click="markLunas(row)">
                      <LucideCheck :size="12" /> Selesai Bayar
                    </button>
                    <button class="fcl-pc-btn fcl-pc-btn-ghost fcl-pc-btn-xs" @click="openTrxModal(row)"><LucidePencil :size="12" /> Edit</button>
                    <button class="fcl-pc-btn fcl-pc-btn-red fcl-pc-btn-xs" @click="deleteTrx(row)"><LucideTrash2 :size="12" /> Hapus</button>
                  </div>
                </td>
              </tr>
              <tr v-if="!txRows.length">
                <td colspan="11" class="fcl-pc-empty-cell">
                  {{ search || onlyBelum ? 'Tidak ada transaksi cocok dengan filter.' : 'Belum ada transaksi di periode ini — klik “+ Transaksi”.' }}
                </td>
              </tr>
            </tbody>
            <tfoot>
              <tr class="fcl-pc-total-row">
                <td colspan="5" class="r">TOTAL</td>
                <td class="r">
                  <div class="fcl-pc-foot-line fcl-pc-tone-red">OUT: {{ fmtRp(totalOut) }}</div>
                  <div class="fcl-pc-foot-line fcl-pc-tone-green">IN: {{ fmtRp(totalIn) }}</div>
                </td>
                <td class="r" colspan="4"><span class="fcl-pc-foot-sisa">Sisa: {{ fmtRp(sisaSaldo) }}</span></td>
                <td></td>
              </tr>
            </tfoot>
          </table>
        </div>
      </div>
    </div>

    <!-- ══════════ Modal: Periode ══════════ -->
    <a-modal
      v-model:open="periodModalOpen"
      :title="periodEditing ? 'Edit Periode' : 'Tambah Periode Petty Cash'"
      :confirm-loading="periodSaving"
      ok-text="Simpan"
      cancel-text="Batal"
      width="520px"
      @ok="savePeriod"
    >
      <a-form layout="vertical" class="fcl-pc-form">
        <a-form-item label="Nama Periode *" required>
          <a-input v-model:value="periodForm.nama" placeholder="Contoh: 16 November–22 November 2026" />
        </a-form-item>
        <div class="fcl-pc-form-row">
          <a-form-item label="Tanggal Mulai">
            <input v-model="periodForm.tanggalMulai" type="date" class="fcl-pc-input" />
          </a-form-item>
          <a-form-item label="Tanggal Selesai">
            <input v-model="periodForm.tanggalSelesai" type="date" class="fcl-pc-input" />
          </a-form-item>
        </div>
        <div class="fcl-pc-form-row">
          <a-form-item label="Saldo Awal (Rp) *" required>
            <a-input-number v-model:value="periodForm.saldoAwal" :min="0" :precision="0" style="width: 100%" placeholder="15000000" />
          </a-form-item>
          <a-form-item label="Status">
            <a-select v-model:value="periodForm.status" :options="[{ label: 'Aktif', value: 'Aktif' }, { label: 'Selesai', value: 'Selesai' }]" />
          </a-form-item>
        </div>
        <a-form-item label="Keterangan">
          <a-textarea v-model:value="periodForm.keterangan" :rows="2" placeholder="Opsional" />
        </a-form-item>
      </a-form>
    </a-modal>

    <!-- ══════════ Modal: Transaksi ══════════ -->
    <a-modal
      v-model:open="trxModalOpen"
      :title="trxEditing ? 'Edit Transaksi Petty Cash' : 'Tambah Transaksi Petty Cash'"
      :confirm-loading="trxSaving"
      ok-text="Simpan"
      cancel-text="Batal"
      width="560px"
      @ok="saveTrx"
    >
      <a-form layout="vertical" class="fcl-pc-form">
        <div class="fcl-pc-form-row">
          <a-form-item label="Tanggal *" required>
            <a-date-picker v-model:value="trxTanggalModel" value-format="YYYY-MM-DD" style="width: 100%" />
          </a-form-item>
          <a-form-item label="Periode *" required>
            <a-select
              v-model:value="trxForm.periodId"
              :options="periods.map((p) => ({ label: String(p.nama), value: String(p.id) }))"
              placeholder="Pilih periode"
            />
          </a-form-item>
        </div>
        <div class="fcl-pc-form-row">
          <a-form-item label="Tipe *" required>
            <a-select v-model:value="trxForm.tipe" :options="[{ label: 'Keluar (uang keluar)', value: 'Keluar' }, { label: 'Masuk (uang masuk)', value: 'Masuk' }]" />
          </a-form-item>
          <a-form-item label="Status Bayar">
            <a-select v-model:value="trxForm.statusBayar" :options="[{ label: 'Belum Bayar', value: 'Belum Bayar' }, { label: 'Lunas', value: 'Lunas' }]" />
          </a-form-item>
        </div>
        <a-form-item label="Kategori">
          <a-select
            v-model:value="trxForm.kategori"
            :options="kategoriOptions"
            placeholder="Pilih kategori..."
            show-search
            allow-clear
            option-filter-prop="label"
          />
          <a-input
            v-if="isKategoriLain"
            v-model:value="kategoriCustom"
            placeholder="Tulis kategori baru..."
            style="margin-top: 8px"
          />
        </a-form-item>
        <a-form-item label="Keterangan">
          <a-textarea v-model:value="trxForm.keterangan" :rows="2" placeholder="Keterangan transaksi" />
        </a-form-item>
        <a-form-item label="Nominal (Rp) *" required>
          <a-input-number v-model:value="trxForm.nominal" :min="0" :precision="0" style="width: 100%" placeholder="0" />
        </a-form-item>
        <a-form-item label="Bukti (link foto/struk)">
          <a-input v-model:value="trxForm.buktiUrl" placeholder="https://... (opsional)" />
        </a-form-item>
      </a-form>
    </a-modal>

    <!-- ══════════ Modal: Bukti ══════════ -->
    <a-modal
      v-model:open="buktiModalOpen"
      title="Bukti Transaksi"
      ok-text="Simpan"
      cancel-text="Batal"
      :confirm-loading="buktiUploading"
      width="520px"
      @ok="saveBukti"
    >
      <div class="fcl-pc-bukti-tabs">
        <button :class="{ 'active': buktiMode === 'link' }" @click="buktiMode = 'link'">Link Google Drive</button>
        <button :class="{ 'active': buktiMode === 'upload' }" @click="buktiMode = 'upload'">Upload Langsung</button>
      </div>
      <template v-if="buktiMode === 'link'">
        <p class="fcl-pc-bukti-hint">Masukkan link foto/struk dari Google Drive (kosongkan untuk menghapus).</p>
        <a-input v-model:value="buktiInput" placeholder="https://drive.google.com/..." allow-clear />
      </template>
      <template v-else>
        <p class="fcl-pc-bukti-hint">Pilih foto/struk dari perangkat (maks. 12 MB). File akan tersimpan di server.</p>
        <input type="file" accept="image/*,application/pdf" class="fcl-pc-file" @change="pickBuktiFile" />
        <div v-if="buktiPreview" class="fcl-pc-bukti-preview"><img :src="buktiPreview" alt="Pratinjau bukti" /></div>
        <p v-if="buktiTarget?.buktiUrl && !buktiFile" class="fcl-pc-bukti-hint">Biarkan kosong untuk menggunakan file yang sudah tersimpan.</p>
      </template>
    </a-modal>

    <!-- ══════════ Modal: Pratinjau Foto Bukti (inline) ══════════ -->
    <a-modal
      v-model:open="viewBuktiOpen"
      :title="viewBuktiLabel ? `Foto Bukti — ${viewBuktiLabel}` : 'Foto Bukti'"
      :footer="null"
      width="760px"
      centered
    >
      <div class="fcl-pc-viewbukti">
        <template v-if="!viewBuktiFailed">
          <img
            v-if="!viewBuktiIsPdf"
            :src="viewBuktiSrc"
            alt="Foto bukti transaksi"
            class="fcl-pc-viewbukti-img"
            @error="onViewBuktiError"
          />
          <iframe
            v-else
            :src="viewBuktiSrc"
            class="fcl-pc-viewbukti-frame"
            title="Dokumen bukti transaksi"
          ></iframe>
        </template>
        <div v-else class="fcl-pc-viewbukti-error">
          <LucideInbox :size="28" />
          <p>Foto tidak dapat ditampilkan langsung. Tautan asli tetap bisa dibuka jika diperlukan.</p>
          <a :href="viewBuktiOriginal" target="_blank" rel="noopener">Buka tautan asli ↗</a>
        </div>
      </div>
    </a-modal>

    <!-- ══════════ Modal: Invoice ══════════ -->
    <a-modal
      v-model:open="invoiceModalOpen"
      title="Cetak Invoice Petty Cash"
      ok-text="Cetak"
      cancel-text="Batal"
      width="620px"
      @ok="generateInvoice"
    >
      <a-form layout="vertical" class="fcl-pc-form">
        <a-form-item label="Judul Invoice">
          <a-input v-model:value="invoiceTitle" />
        </a-form-item>
        <a-form-item label="Catatan (opsional)">
          <a-input v-model:value="invoiceNotes" placeholder="Catatan untuk invoice" />
        </a-form-item>
        <div class="fcl-pc-invoice-sel-row">
          <span class="fcl-pc-invoice-sel-count">{{ invoiceTxs.length }} transaksi dipilih</span>
          <div>
            <a-button size="small" @click="toggleInvoiceAll(true)">Pilih Semua</a-button>
            <a-button size="small" style="margin-left: 6px" @click="toggleInvoiceAll(false)">Kosongkan</a-button>
          </div>
        </div>
        <div class="fcl-pc-invoice-list">
          <label v-for="t in periodTxs" :key="t.id" class="fcl-pc-invoice-item">
            <input type="checkbox" :checked="invoiceSelected.has(String(t.id))" @change="toggleInvoiceTx(String(t.id))" />
            <span class="fcl-pc-invoice-item-body">
              <span class="fcl-pc-invoice-item-title">{{ (t.tanggal ?? '').slice(0, 10) }} · {{ t.kategori }}</span>
              <span class="fcl-pc-invoice-item-ket">{{ t.keterangan }}</span>
              <span class="fcl-pc-invoice-item-nom">Rp {{ toNum(t.nominal).toLocaleString('id-ID') }}</span>
            </span>
          </label>
        </div>
      </a-form>
    </a-modal>
  </Page>
</template>

<style scoped>
/* Petty Cash — memakai token tema vben (hsl(var(--...))) supaya otomatis
   mengikuti tema terang/gelap aplikasi. Tanpa warna hard-coded. */
.fcl-pc {
  display: flex;
  flex-direction: column;
  gap: 12px;
  color: hsl(var(--foreground));
  font-size: 13px;
}

/* ── Hero (belum ada periode) ── */
.fcl-pc-hero {
  background: hsl(var(--card));
  border: 1px dashed hsl(var(--border));
  border-radius: 10px;
  padding: 48px 24px;
  text-align: center;
  color: hsl(var(--foreground));
}
.fcl-pc-hero-icon { color: hsl(var(--success)); display: flex; justify-content: center; margin-bottom: 12px; }
.fcl-pc-hero-title { font-size: 18px; font-weight: 800; margin-bottom: 8px; }
.fcl-pc-hero-desc { color: hsl(var(--muted-foreground)); max-width: 520px; margin: 0 auto 18px; line-height: 1.7; }

/* ── Bar statistik ── */
.fcl-pc-stats { display: grid; grid-template-columns: repeat(4, minmax(0, 1fr)); gap: 10px; }
.fcl-pc-stat {
  background: hsl(var(--card));
  border: 1px solid hsl(var(--border));
  border-top: 3px solid hsl(var(--muted-foreground));
  border-radius: 8px;
  padding: 12px 14px;
}
.fcl-pc-stat-green { border-top-color: hsl(var(--success)); }
.fcl-pc-stat-red { border-top-color: hsl(var(--destructive)); }
.fcl-pc-stat-teal { border-top-color: hsl(var(--success)); }
.fcl-pc-stat-amber { border-top-color: hsl(var(--warning)); }
.fcl-pc-stat-label {
  font-size: 11px;
  font-weight: 700;
  letter-spacing: 0.04em;
  color: hsl(var(--muted-foreground));
  margin-bottom: 4px;
}
.fcl-pc-stat-value { font-size: 20px; font-weight: 800; color: hsl(var(--foreground)); }
.fcl-pc-tone-green { color: hsl(var(--success)) !important; }
.fcl-pc-tone-red { color: hsl(var(--destructive)) !important; }
.fcl-pc-tone-teal { color: hsl(var(--success)) !important; }
.fcl-pc-tone-amber { color: hsl(var(--warning)) !important; }

/* ── Kartu / panel ── */
.fcl-pc-card {
  background: hsl(var(--card));
  border: 1px solid hsl(var(--border));
  border-radius: 8px;
  padding: 12px;
}
.fcl-pc-card-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 10px;
  flex-wrap: wrap;
  padding: 0 0 10px;
}
.fcl-pc-card-title { font-size: 14px; font-weight: 800; color: hsl(var(--foreground)); }
.fcl-pc-card-actions { display: flex; gap: 6px; flex-wrap: wrap; }

/* ── Tombol ── */
.fcl-pc-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 5px;
  border-radius: 6px;
  border: 1px solid hsl(var(--border));
  background: hsl(var(--muted));
  color: hsl(var(--foreground));
  font-size: 12px;
  font-weight: 700;
  padding: 5px 12px;
  cursor: pointer;
  white-space: nowrap;
  transition: filter 0.15s ease;
}
.fcl-pc-btn:hover { filter: brightness(0.96); }
.fcl-pc-btn-lg { font-size: 14px; padding: 10px 20px; }
.fcl-pc-btn-sm { padding: 4px 10px; font-size: 11px; }
.fcl-pc-btn-xs { padding: 3px 8px; font-size: 11px; }
.fcl-pc-btn-primary,
.fcl-pc-btn-info,
.fcl-pc-btn-orange { background: hsl(var(--primary)); border-color: hsl(var(--primary)); color: hsl(var(--primary-foreground)); }
.fcl-pc-btn-ghost { background: hsl(var(--muted)); border-color: hsl(var(--border)); }
.fcl-pc-btn-teal { background: hsl(var(--success)); border-color: hsl(var(--success)); color: hsl(var(--success-foreground)); }
.fcl-pc-btn-red { background: hsl(var(--destructive)); border-color: hsl(var(--destructive)); color: hsl(var(--destructive-foreground)); }
.fcl-pc-btn-toggled { background: hsl(var(--warning) / 0.18); border-color: hsl(var(--warning)); color: hsl(var(--warning)); }

/* ── Tab filter ── */
.fcl-pc-tabs { display: flex; gap: 4px; padding: 0 0 10px; }
.fcl-pc-tab {
  border: 1px solid transparent;
  background: hsl(var(--muted));
  color: hsl(var(--muted-foreground));
  font-size: 11px;
  font-weight: 700;
  padding: 4px 14px;
  border-radius: 6px;
  cursor: pointer;
}
.fcl-pc-tab:hover { color: hsl(var(--foreground)); }
.fcl-pc-tab-active { background: hsl(var(--primary) / 0.12); border-color: hsl(var(--primary) / 0.4); color: hsl(var(--primary)); }

/* ── Kartu periode ── */
.fcl-pc-period-list { display: flex; flex-wrap: wrap; gap: 10px; }
.fcl-pc-period-card {
  flex: 1 1 230px;
  min-width: 220px;
  max-width: 300px;
  background: hsl(var(--background));
  border: 1px solid hsl(var(--border));
  border-radius: 8px;
  padding: 10px 12px;
  cursor: pointer;
}
.fcl-pc-period-card:hover { border-color: hsl(var(--primary) / 0.5); }
.fcl-pc-period-card-active { border-color: hsl(var(--primary)); background: hsl(var(--primary) / 0.06); }
.fcl-pc-period-card-top { display: flex; justify-content: space-between; align-items: flex-start; gap: 8px; }
.fcl-pc-period-nama { font-size: 12px; font-weight: 800; color: hsl(var(--foreground)); line-height: 1.4; }
.fcl-pc-period-card-active .fcl-pc-period-nama { color: hsl(var(--primary)); }
.fcl-pc-period-date { font-size: 11px; color: hsl(var(--muted-foreground)); margin-top: 4px; }
.fcl-pc-period-saldo { font-size: 11px; color: hsl(var(--muted-foreground)); margin-top: 2px; }
.fcl-pc-period-saldo b { color: hsl(var(--success)); }
.fcl-pc-period-actions { display: flex; gap: 5px; margin-top: 10px; }
.fcl-pc-badge { font-size: 10px; font-weight: 800; padding: 2px 8px; border-radius: 999px; white-space: nowrap; }
.fcl-pc-badge-green { color: hsl(var(--success)); background: hsl(var(--success) / 0.14); }
.fcl-pc-badge-gray { color: hsl(var(--muted-foreground)); background: hsl(var(--muted)); }
.fcl-pc-mini-btn {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  font-size: 11px;
  font-weight: 700;
  padding: 3px 9px;
  border-radius: 5px;
  border: 1px solid hsl(var(--border));
  background: hsl(var(--muted));
  color: hsl(var(--foreground));
  cursor: pointer;
}
.fcl-pc-mini-btn:hover { filter: brightness(0.96); }
.fcl-pc-mini-btn-red { border-color: hsl(var(--destructive) / 0.5); color: hsl(var(--destructive)); }
.fcl-pc-mini-btn-green { border-color: hsl(var(--success) / 0.5); color: hsl(var(--success)); }
.fcl-pc-empty-line { color: hsl(var(--muted-foreground)); font-size: 12px; padding: 10px 0; }

/* ── Panel transaksi ── */
.fcl-pc-trx-head { align-items: flex-start; }
.fcl-pc-trx-heading { display: flex; flex-direction: column; gap: 2px; }
.fcl-pc-trx-sub { font-size: 11px; color: hsl(var(--muted-foreground)); }
.fcl-pc-trx-sub b { color: hsl(var(--success)); }
.fcl-pc-trx-actions { display: flex; gap: 6px; flex-wrap: wrap; align-items: center; }
.fcl-pc-search-input {
  background: hsl(var(--background));
  border: 1px solid hsl(var(--border));
  border-radius: 6px;
  color: hsl(var(--foreground));
  font-size: 12px;
  height: 30px;
  padding: 2px 10px;
  width: 160px;
  outline: none;
}
.fcl-pc-search-input:focus { border-color: hsl(var(--primary)); }

/* ── Tabel (header mengikuti tema, tidak hitam) ── */
.fcl-pc-table-wrap { overflow-x: auto; border-top: 1px solid hsl(var(--border)); }
.fcl-pc-table { width: 100%; min-width: 1060px; border-collapse: collapse; font-size: 12px; }
.fcl-pc-table th {
  background: hsl(var(--muted));
  color: hsl(var(--muted-foreground));
  font-size: 11px;
  font-weight: 800;
  letter-spacing: 0.04em;
  text-align: left;
  padding: 8px;
  border-bottom: 1px solid hsl(var(--border));
  white-space: nowrap;
}
.fcl-pc-table th.r, .fcl-pc-table td.r { text-align: right; }
.fcl-pc-table th.c, .fcl-pc-table td.c { text-align: center; }
.fcl-pc-table th.c-no, .fcl-pc-table td.c-no { width: 36px; text-align: center; color: hsl(var(--muted-foreground)); }
.fcl-pc-table td {
  padding: 7px 8px;
  border-bottom: 1px solid hsl(var(--border));
  vertical-align: middle;
  white-space: nowrap;
  color: hsl(var(--foreground));
}
.fcl-pc-table tbody tr:nth-child(even) { background: hsl(var(--muted) / 0.45); }
.fcl-pc-table tbody tr:hover { background: hsl(var(--accent)); }
.fcl-pc-tanggal { display: flex; flex-direction: column; line-height: 1.25; font-weight: 700; }
.fcl-pc-tanggal-year { color: hsl(var(--muted-foreground)); font-weight: 500; font-size: 10px; }
.fcl-pc-tipe { font-weight: 800; }
.fcl-pc-tipe-out { color: hsl(var(--destructive)); }
.fcl-pc-tipe-in { color: hsl(var(--success)); }
.fcl-pc-kat-chip {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-size: 10px;
  font-weight: 800;
  background: hsl(var(--muted));
  border: 1px solid hsl(var(--border));
  border-radius: 5px;
  padding: 3px 8px;
  color: hsl(var(--foreground));
}
.fcl-pc-kat-chip::before { content: ''; width: 6px; height: 6px; border-radius: 99px; background: var(--chip, hsl(var(--primary))); }
.fcl-pc-ket { max-width: 260px; overflow: hidden; text-overflow: ellipsis; }
.fcl-pc-nominal { font-weight: 800; }
.fcl-pc-saldo { color: hsl(var(--success)); font-weight: 700; }
.fcl-pc-statuspill { display: inline-block; font-size: 10px; font-weight: 800; letter-spacing: 0.04em; padding: 3px 10px; border-radius: 5px; white-space: nowrap; }
.fcl-pc-pill-belum { background: hsl(var(--warning) / 0.14); color: hsl(var(--warning)); border: 1px solid hsl(var(--warning) / 0.4); }
.fcl-pc-pill-lunas { background: hsl(var(--success) / 0.14); color: hsl(var(--success)); border: 1px solid hsl(var(--success) / 0.4); }
.fcl-pc-bukti { display: flex; align-items: center; gap: 6px; }
.fcl-pc-bukti-link { display: inline-flex; align-items: center; gap: 5px; color: hsl(var(--primary)); font-size: 11px; font-weight: 700; background: none; border: none; cursor: pointer; padding: 0; }
.fcl-pc-bukti-upload { display: inline-flex; align-items: center; gap: 5px; border: 1px dashed hsl(var(--border)); background: transparent; color: hsl(var(--muted-foreground)); font-size: 11px; font-weight: 700; border-radius: 6px; padding: 4px 10px; cursor: pointer; }
.fcl-pc-bukti-upload:hover { color: hsl(var(--foreground)); border-color: hsl(var(--primary)); }
.fcl-pc-bukti-hint { color: hsl(var(--muted-foreground)); font-size: 12px; margin-bottom: 10px; }
.fcl-pc-viewbukti { display: flex; align-items: center; justify-content: center; min-height: 220px; background: hsl(var(--muted)); border: 1px solid hsl(var(--border)); border-radius: 8px; overflow: hidden; }
.fcl-pc-viewbukti-img { display: block; max-width: 100%; max-height: 65vh; width: auto; height: auto; object-fit: contain; }
.fcl-pc-viewbukti-frame { display: block; width: 100%; height: 65vh; border: 0; }
.fcl-pc-viewbukti-error { display: flex; flex-direction: column; align-items: center; gap: 10px; padding: 32px 20px; text-align: center; color: hsl(var(--muted-foreground)); font-size: 13px; }
.fcl-pc-viewbukti-error a { color: hsl(var(--primary)); font-weight: 700; }
.fcl-pc-by { color: hsl(var(--muted-foreground)); max-width: 110px; overflow: hidden; text-overflow: ellipsis; }
.fcl-pc-aksi { display: flex; gap: 4px; align-items: center; justify-content: center; }
.fcl-pc-empty-cell { text-align: center; color: hsl(var(--muted-foreground)); padding: 22px 8px; }

/* ── Footer ── */
.fcl-pc-total-row td { background: hsl(var(--muted)); border-top: 1px solid hsl(var(--border)); padding: 8px; }
.fcl-pc-foot-line { font-size: 11px; font-weight: 800; line-height: 1.5; }
.fcl-pc-foot-sisa { font-size: 11px; font-weight: 800; color: hsl(var(--warning)); }

/* ── Form & invoice ── */
.fcl-pc-form { padding-top: 6px; }
.fcl-pc-form-row { display: grid; grid-template-columns: 1fr 1fr; gap: 12px; }
.fcl-pc-input {
  background: hsl(var(--background));
  border: 1px solid hsl(var(--border));
  color: hsl(var(--foreground));
  border-radius: 6px;
  padding: 6px 10px;
  font-size: 12px;
  height: 32px;
  width: 100%;
  outline: none;
}
.fcl-pc-input:focus { border-color: hsl(var(--primary)); }
.fcl-pc-invoice-sel-row { display: flex; align-items: center; justify-content: space-between; margin-bottom: 8px; }
.fcl-pc-invoice-sel-count { font-size: 11px; color: hsl(var(--muted-foreground)); }
.fcl-pc-invoice-list { max-height: 280px; overflow-y: auto; display: flex; flex-direction: column; gap: 6px; }
.fcl-pc-invoice-item { display: flex; gap: 8px; align-items: flex-start; background: hsl(var(--muted)); border: 1px solid hsl(var(--border)); border-radius: 6px; padding: 6px 10px; cursor: pointer; }
.fcl-pc-invoice-item input { margin-top: 3px; }
.fcl-pc-invoice-item-body { display: flex; flex-direction: column; }
.fcl-pc-invoice-item-title { font-weight: 700; font-size: 12px; }
.fcl-pc-invoice-item-ket { color: hsl(var(--muted-foreground)); font-size: 11px; }
.fcl-pc-invoice-item-nom { font-size: 11px; margin-top: 2px; }

@media (max-width: 1100px) {
  .fcl-pc-stats { grid-template-columns: repeat(2, minmax(0, 1fr)); }
  .fcl-pc-period-card { flex: 1 1 200px; }
}

.fcl-pc-bukti-tabs { display: flex; gap: 6px; margin-bottom: 10px; }
.fcl-pc-bukti-tabs button { border: 1px solid hsl(var(--border)); background: hsl(var(--muted)); color: hsl(var(--muted-foreground)); font-size: 11px; font-weight: 700; padding: 5px 12px; border-radius: 6px; cursor: pointer; }
.fcl-pc-bukti-tabs button.active { background: hsl(var(--primary)); border-color: hsl(var(--primary)); color: hsl(var(--primary-foreground)); }
.fcl-pc-file { font-size: 11px; width: 100%; color: hsl(var(--muted-foreground)); }
.fcl-pc-bukti-preview { margin-top: 10px; max-height: 240px; overflow: hidden; border: 1px solid hsl(var(--border)); border-radius: 6px; }
.fcl-pc-bukti-preview img { max-width: 100%; max-height: 240px; object-fit: contain; display: block; }
</style>
