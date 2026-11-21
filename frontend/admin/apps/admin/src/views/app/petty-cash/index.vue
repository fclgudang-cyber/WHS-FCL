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
  LucideRefreshCw,
  LucideTrash2,
  LucideWallet,
} from '@vben/icons';

import { Form, Input, Modal, message, notification, Select, Textarea } from 'ant-design-vue';
import dayjs from 'dayjs';

import { fclDelete, fclInsert, fclPatch, fclQuery, formatRp } from '#/api/warehouse';

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

// Pindahkan transaksi ke periode lain.
const pindahOpen = ref(false);
const pindahTarget = ref<TxRow | null>(null);
const pindahPeriodId = ref('');
const pindahSaving = ref(false);

function openPindah(t: TxRow) {
  pindahTarget.value = t;
  pindahPeriodId.value = String(t.periodId ?? '');
  pindahOpen.value = true;
}

async function savePindah() {
  if (!pindahTarget.value) return;
  if (!pindahPeriodId.value) { message.warning('Pilih periode tujuan'); return; }
  if (String(pindahPeriodId.value) === String(pindahTarget.value.periodId ?? '')) {
    message.warning('Transaksi sudah berada di periode ini');
    return;
  }
  pindahSaving.value = true;
  try {
    await fclPatch('pettycash', pindahTarget.value.id, { periodId: pindahPeriodId.value });
    const tujuan = periods.value.find((p) => String(p.id) === String(pindahPeriodId.value));
    notification.success({
      message: 'Transaksi dipindahkan',
      description: `Sekarang berada di periode ${tujuan?.nama ?? '-'}`,
    });
    pindahOpen.value = false;
    await loadAll(true);
  } catch (err: any) {
    notification.error({ message: 'Pindah Transaksi', description: err?.message || 'Gagal memindahkan' });
  } finally {
    pindahSaving.value = false;
  }
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

// Unggah file ke Drive/penyimpanan sesuai periode transaksi.
async function uploadBuktiFile(file: File, target: TxRow): Promise<string> {
  const data = new FormData();
  data.append('file', file);
  const trxPeriodId = target.periodId ?? activePeriodId.value;
  const trxPeriod = periods.value.find((p) => String(p.id) === String(trxPeriodId));
  data.append('period', String(trxPeriod?.nama ?? activePeriod.value?.nama ?? 'Tanpa Periode'));
  const res = await fetch('/api/file/upload', { method: 'POST', body: data });
  const out = await res.json();
  if (!out?.success || !out?.url) throw new Error(out?.message || 'Upload gagal');
  return String(out.url);
}

async function saveBukti() {
  if (!buktiTarget.value) return;
  let url = buktiTarget.value.buktiUrl ?? '';
  if (buktiMode.value === 'upload') {
    if (!buktiFile.value && !url) { message.warning('Pilih file bukti terlebih dahulu'); return; }
    if (buktiFile.value) {
      buktiUploading.value = true;
      try {
        url = await uploadBuktiFile(buktiFile.value, buktiTarget.value);
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

// Upload langsung: tanpa modal — pilih foto lalu langsung tersimpan.
const directFileInput = ref<HTMLInputElement | null>(null);
let directTargetId: string | null = null;
const directUploadingId = ref<string | null>(null);
const isDirectUploading = (t: TxRow) => String(directUploadingId.value) === String(t.id);

function directUpload(t: TxRow) {
  directTargetId = t.id;
  if (directFileInput.value) directFileInput.value.value = '';
  directFileInput.value?.click();
}

async function onDirectFileChange(e: Event) {
  const file = (e.target as HTMLInputElement).files?.[0] ?? null;
  const target = txs.value.find((t) => String(t.id) === String(directTargetId));
  directTargetId = null;
  if (!file || !target) return;
  if (file.size > 12 * 1024 * 1024) { message.warning('Ukuran file maksimal 12 MB'); return; }
  directUploadingId.value = String(target.id);
  try {
    const url = await uploadBuktiFile(file, target);
    await fclPatch('pettycash', target.id, { buktiUrl: url });
    notification.success({ message: 'Foto bukti tersimpan ke Google Drive' });
    await loadAll(true);
  } catch (err: any) {
    notification.error({ message: 'Upload Bukti', description: err?.message || 'Gagal mengunggah file' });
  } finally {
    directUploadingId.value = null;
  }
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

// Export Petty Cash ke file Excel (.xls) — layout mengikuti format laporan gudang.
function exportExcel() {
  if (!activePeriod.value) { message.warning('Pilih periode dulu'); return; }
  const period = activePeriod.value;
  if (!txRows.value.length) { message.warning('Belum ada transaksi di periode ini'); return; }

  const katKode = new Map<string, string>();
  kategoris.value.forEach((k) => {
    const nama = String(k.nama ?? '').trim();
    if (nama) katKode.set(nama, String(k.kode ?? ''));
  });
  const absUrl = (u: string) => (/^https?:\/\//i.test(u) ? u : `${window.location.origin}${u}`);

  const dMulai = dayjs(period.tanggalMulai);
  const dSelesai = dayjs(period.tanggalSelesai);
  const periodeLabel = dMulai.isValid() && dSelesai.isValid()
    ? `${dMulai.format('D MMMM')}-${dSelesai.format('D MMMM YYYY')}`
    : String(period.nama || '-');
  const judul = `LAPORAN PETTY CASH - Periode ${periodeLabel}`;
  const subjudul = `Periode: ${fmtTanggalPendek(period.tanggalMulai)} s/d ${fmtTanggalPendek(period.tanggalSelesai)} | Saldo Awal: ${formatRp(toNum(period.saldoAwal))}`;

  const header = ['No', 'Tanggal', 'Tipe', 'Kategori', 'Keterangan', 'Nominal', 'Saldo Berjalan', 'Status Bayar', 'Link Bukti', 'Input By'];
  const dataRows = txRows.value.map((r) => {
    const namaKat = String(r.kategori ?? '').trim();
    const kode = katKode.get(namaKat) ?? '';
    const kat = kode ? `${kode} ${namaKat}` : (namaKat || '-');
    const tipe = r._tipe === 'Masuk' ? 'IN' : 'OUT';
    const link = r.buktiUrl ? String(r.buktiUrl) : '';
    const linkHtml = link
      ? `<a href="${esc(absUrl(link))}" target="_blank">${esc(link)}</a>`
      : '<span style="color:#b91c1c">tanpa bukti</span>';
    return [
      String(r._no),
      dayjs(r.tanggal).isValid() ? dayjs(r.tanggal).format('MM/DD/YYYY') : '',
      tipe,
      esc(kat),
      esc(r.keterangan ?? ''),
      String(r._nomSigned),
      String(r._saldo),
      r._lunas ? 'Lunas' : 'Belum',
      linkHtml,
      esc(r.createdBy ?? ''),
    ];
  });

  const headHtml = header.map((h) => `<th class="head">${esc(h)}</th>`).join('');
  const bodyHtml = dataRows
    .map((cols, i) => `<tr>${cols.map((c, ci) => {
      if (ci === 5 || ci === 6) return `<td class="num">${c}</td>`;
      if (ci === 8) return `<td class="link">${c}</td>`;
      if (ci === 0) return `<td class="center">${c}</td>`;
      return `<td>${c}</td>`;
    }).join('')}</tr>`).join('');

  const html = `<!doctype html><html xmlns:o="urn:schemas-microsoft-com:office:office" xmlns:x="urn:schemas-microsoft-com:office:excel" xmlns="http://www.w3.org/TR/REC-html40"><head><meta charset="utf-8"/>
<!--[if gte mso 9]><xml><x:ExcelWorkbook><x:ExcelWorksheets><x:ExcelWorksheet><x:Name>Petty Cash</x:Name><x:WorksheetOptions><x:DisplayGridlines/></x:WorksheetOptions></x:ExcelWorksheet></x:ExcelWorksheets></x:ExcelWorkbook></xml><![endif]-->
<style>
  table { border-collapse: collapse; width: 100%; font-family: Calibri, Arial, sans-serif; font-size: 11px; }
  th, td { border: 1px solid #bfbfbf; padding: 3px 6px; vertical-align: top; }
  .judul { font-size: 13px; font-weight: bold; }
  .subjudul { font-size: 11px; }
  .head { font-weight: bold; background: #f2f2f2; text-align: left; }
  .num { mso-number-format: "0"; text-align: right; }
  .link { mso-number-format: "\\@"; word-break: break-all; }
  .center { text-align: center; }
</style></head><body>
<table>
  <colgroup>
    <col width="38"/><col width="83"/><col width="48"/><col width="195"/>
    <col width="250"/><col width="90"/><col width="105"/><col width="95"/>
    <col width="430"/><col width="175"/>
  </colgroup>
  <tr><td colspan="10" class="judul">${esc(judul)}</td></tr>
  <tr><td colspan="10" class="subjudul">${esc(subjudul)}</td></tr>
  <tr><td colspan="10">&nbsp;</td></tr>
  <tr>${headHtml}</tr>
  ${bodyHtml}
</table>
</body></html>`;

  const blob = new Blob(['\uFEFF' + html], { type: 'application/vnd.ms-excel;charset=utf-8;' });
  const url = URL.createObjectURL(blob);
  const a = document.createElement('a');
  a.href = url;
  a.download = `laporan-petty-cash-${period.nama.replace(/\s+/g, '_')}.xls`;
  document.body.appendChild(a);
  a.click();
  document.body.removeChild(a);
  URL.revokeObjectURL(url);
  notification.success({ message: 'Export Petty Cash', description: 'File Excel (.xls) berhasil diunduh' });
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

.wh-pc-bukti-tabs { display: flex; gap: 6px; margin-bottom: 10px; }
.wh-pc-bukti-tabs button { border: 1px solid hsl(var(--border)); background: hsl(var(--muted)); color: hsl(var(--muted-foreground)); font-size: 11px; font-weight: 700; padding: 5px 12px; border-radius: 6px; cursor: pointer; }
.wh-pc-bukti-tabs button.active { background: hsl(var(--primary)); border-color: hsl(var(--primary)); color: hsl(var(--primary-foreground)); }
.wh-pc-file { font-size: 11px; width: 100%; color: hsl(var(--muted-foreground)); }
.wh-pc-bukti-preview { margin-top: 10px; max-height: 240px; overflow: hidden; border: 1px solid hsl(var(--border)); border-radius: 6px; }
.wh-pc-bukti-preview img { max-width: 100%; max-height: 240px; object-fit: contain; display: block; }
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

  const palette = ['#1e3a5f', '#0f6b4f', '#9a3412', '#6d28d9', '#0e7490', '#b91c1c'];
  const absUrl = (u: string) => (/^https?:\/\//i.test(u) ? u : `${window.location.origin}${u}`);

  // Link "view" Google Drive bukan gambar — harus diubah ke URL unduh/thumbnail langsung.
  const imageUrls = (u: string): string[] => {
    try {
      const parsed = new URL(u);
      if (/drive\.google\.com$/i.test(parsed.hostname)) {
        const m = parsed.pathname.match(/\/file\/d\/([^/]+)/);
        const id = m ? m[1] : parsed.searchParams.get('id') ?? '';
        if (id) {
          return [
            `https://lh3.googleusercontent.com/d/${id}`,
            `https://drive.google.com/uc?export=view&id=${id}`,
            u,
          ];
        }
      }
    } catch { /* bukan URL absolut */ }
    return [absUrl(u)];
  };

  // Urutan grup diambil dari master kategori (urutan/kode A, B, C, D, F, ...).
  const katIndex = new Map<string, { kode: string; urutan: number }>();
  kategoris.value.forEach((k) => {
    const nama = String(k.nama ?? '').trim();
    if (nama) {
      katIndex.set(nama, { kode: String(k.kode ?? ''), urutan: Number(k.urutan ?? 0) });
    }
  });
  const orderedGroups = [...grouped.entries()].sort((a, b) => {
    const ka = katIndex.get(a[0]);
    const kb = katIndex.get(b[0]);
    if (!ka && !kb) return a[0].localeCompare(b[0], 'id');
    if (!ka) return 1;
    if (!kb) return -1;
    return (ka.urutan - kb.urutan) || ka.kode.localeCompare(kb.kode);
  });

  const sections = orderedGroups.map(([kat, items], gi) => {
    const color = palette[gi % palette.length];
    const meta = katIndex.get(kat);
    const label = meta?.kode ? `${meta.kode} ${kat}` : kat;
    const fotoCount = items.filter((t) => !!t.buktiUrl).length;
    const cards = items.map((t, i) => {
      const by = t.createdBy ? `<div class="card-by">• ${esc(t.createdBy)}</div>` : '';
      const urls = imageUrls(String(t.buktiUrl));
      const image = t.buktiUrl
        ? `<img src="${esc(urls[0])}" data-urls="${esc(urls.join('|'))}" data-idx="0" alt="Bukti ${esc(t.keterangan || i + 1)}" onerror="imgFail(this)"/><div class="img-fallback" style="display:none">Foto tidak dapat dimuat</div>`
        : '<div class="img-fallback">Tanpa foto</div>';
      return '<div class="card">'
        + '<div class="card-top"><span class="num">#' + (i + 1) + '</span><span class="date">' + esc(fmtTanggalPendek(t.tanggal)) + '</span></div>'
        + '<div class="card-title">' + esc(t.keterangan || '-') + '</div>'
        + '<div class="card-amt">Rp ' + toNum(t.nominal).toLocaleString('id-ID') + '</div>'
        + by
        + '<div class="card-img">' + image + '</div>'
        + '</div>';
    }).join('');
    return '<section class="cat" style="--cat:' + color + '">'
      + '<div class="cat-head"><span class="cat-name">' + esc(label) + '</span>'
      + '<span class="pill">' + fotoCount + ' foto / ' + items.length + ' trx</span></div>'
      + '<div class="cat-body">' + cards + '</div></section>';
  }).join('');

  const html = `<!doctype html><html lang="id"><head><meta charset="utf-8"/>
<title>Bukti Foto Petty Cash - ${esc(period.nama)}</title>
<style>
  * { box-sizing: border-box; }
  body { margin: 0; padding: 22px; background: #f3f4f6; color: #111827;
         font-family: 'Segoe UI', Arial, sans-serif; font-size: 13px; }
  .page-head { display: flex; align-items: baseline; justify-content: space-between; gap: 16px;
               margin: 0 2px 16px; }
  .page-title { font-size: 18px; font-weight: 800; }
  .page-sub { color: #6b7280; font-size: 11px; }
  .cat { border: 1px solid #e5e7eb; border-radius: 9px; background: #fff; overflow: hidden;
         margin-bottom: 18px; box-shadow: 0 1px 2px rgba(0,0,0,.04); }
  .cat-head { display: flex; align-items: center; justify-content: space-between; gap: 12px;
              padding: 10px 16px; background: var(--cat); color: #fff; }
  .cat-name { font-weight: 800; letter-spacing: .02em; font-size: 13px; }
  .pill { background: rgba(255,255,255,.18); border: 1px solid rgba(255,255,255,.35);
          border-radius: 999px; padding: 3px 12px; font-size: 11px; font-weight: 700; white-space: nowrap; }
  .cat-body { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 14px; padding: 14px; }
  .card { border: 1px solid #e5e7eb; border-radius: 7px; background: #fff; padding: 12px;
          min-width: 0; break-inside: avoid; }
  .card-top { display: flex; align-items: center; justify-content: space-between; gap: 8px; margin-bottom: 8px; }
  .num { display: inline-block; border: 1px solid #bfdbfe; color: #2563eb; background: #eff6ff;
         border-radius: 3px; padding: 1px 6px; font-size: 10px; font-weight: 800; }
  .date { color: #9ca3af; font-size: 10px; }
  .card-title { font-weight: 700; font-size: 13px; margin-bottom: 3px; }
  .card-amt { color: #dc2626; font-weight: 800; font-size: 12px; }
  .card-by { color: #9ca3af; font-size: 10px; font-style: italic; margin-top: 2px; }
  .card-img { margin-top: 10px; border: 1px solid #e5e7eb; border-radius: 5px; background: #fafafa;
              padding: 6px; display: flex; align-items: center; justify-content: center;
              min-height: 120px; }
  .card-img img { display: block; width: 100%; max-height: 260px; object-fit: contain; }
  .img-fallback { width: 100%; min-height: 120px; align-items: center; justify-content: center;
                  color: #b91c1c; font-size: 11px; }
  .actions { position: fixed; right: 20px; bottom: 18px; display: flex; gap: 8px; z-index: 10; }
  .actions button { border: 0; border-radius: 6px; padding: 9px 18px; color: #fff;
                    font-size: 12px; font-weight: 800; cursor: pointer; box-shadow: 0 2px 6px rgba(0,0,0,.15); }
  .btn-print { background: #f59e0b; }
  .btn-close { background: #6b7280; }
  @media print {
    @page { size: A4; margin: 10mm; }
    body { background: #fff; padding: 0; }
    .actions { display: none !important; }
    .cat { break-inside: avoid; box-shadow: none; }
    .card-img img { max-height: 190px; }
    .cat-head { -webkit-print-color-adjust: exact; print-color-adjust: exact; }
  }
</style></head><body>
<script>
function imgFail(img) {
  var list = (img.getAttribute('data-urls') || '').split('|');
  var i = parseInt(img.getAttribute('data-idx') || '0', 10) + 1;
  if (i < list.length && list[i]) {
    img.setAttribute('data-idx', String(i));
    img.src = list[i];
    return;
  }
  img.style.display = 'none';
  var f = img.nextElementSibling;
  if (f) f.style.display = 'flex';
}
<\/script>
  <div class="page-head">
    <div><div class="page-title">BUKTI FOTO PETTY CASH</div>
      <div class="page-sub">${esc(period.nama)} &nbsp;•&nbsp; ${esc(fmtTanggalPendek(period.tanggalMulai))} s/d ${esc(fmtTanggalPendek(period.tanggalSelesai))}</div></div>
  </div>
  ${sections}
  <div class="actions">
    <button class="btn-print" onclick="window.print()">Print / Save PDF</button>
    <button class="btn-close" onclick="window.close()">Tutup</button>
  </div>
</body></html>`;

  const w = window.open('', '_blank', 'width=1240,height=860');
  if (!w) { message.warning('Popup diblokir browser. Izinkan popup untuk mencetak.'); return; }
  w.document.write(html);
  w.document.close();
  w.focus();
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
    <!-- Input tersembunyi untuk upload langsung (tanpa modal) -->
    <input
      ref="directFileInput"
      type="file"
      accept="image/*,application/pdf"
      style="display: none"
      :disabled="!!directUploadingId"
      @change="onDirectFileChange"
    />
    <div v-if="directUploadingId" class="wh-pc-upload-status">
      <span class="wh-pc-spin"></span> Mengunggah foto ke Google Drive...
    </div>
    <!-- ══════════ BELUM ADA PERIODE → hero create-first ══════════ -->
    <div v-if="!loading && periods.length === 0" class="wh-pc-hero">
      <div class="wh-pc-hero-icon"><LucideWallet :size="44" /></div>
      <div class="wh-pc-hero-title">Belum ada Periode Petty Cash</div>
      <div class="wh-pc-hero-desc">
        Alur kerja petty cash: <b>buat periode dulu</b> (mis. “16 November–22 November 2026” dengan saldo awal),
        baru setelah itu <b>tambah transaksi</b> ke periode tersebut.
      </div>
      <button class="wh-pc-btn wh-pc-btn-primary wh-pc-btn-lg" @click="openPeriodModal()">
        <LucidePlus :size="16" /> Buat Periode Baru
      </button>
    </div>

    <!-- ══════════ DASHBOARD PETTY CASH ══════════ -->
    <div v-else class="wh-pc">
      <!-- ── Bar statistik ── -->
      <div class="wh-pc-stats">
        <div class="wh-pc-stat wh-pc-stat-green">
          <div class="wh-pc-stat-label">SALDO AWAL</div>
          <div class="wh-pc-stat-value wh-pc-tone-green">{{ fmtRp(saldoAwal) }}</div>
        </div>
        <div class="wh-pc-stat wh-pc-stat-red">
          <div class="wh-pc-stat-label">PENGELUARAN</div>
          <div class="wh-pc-stat-value wh-pc-tone-red">{{ fmtRp(totalOut) }}</div>
        </div>
        <div class="wh-pc-stat wh-pc-stat-teal">
          <div class="wh-pc-stat-label">PEMASUKAN</div>
          <div class="wh-pc-stat-value wh-pc-tone-teal">{{ fmtRp(totalIn) }}</div>
        </div>
        <div class="wh-pc-stat wh-pc-stat-amber">
          <div class="wh-pc-stat-label">SISA SALDO</div>
          <div class="wh-pc-stat-value wh-pc-tone-amber">{{ fmtRp(sisaSaldo) }}</div>
        </div>
      </div>

      <!-- ── Seksi Periode ── -->
      <div class="wh-pc-card">
        <div class="wh-pc-card-head">
          <div class="wh-pc-card-title">Periode</div>
          <div class="wh-pc-card-actions">
            <button class="wh-pc-btn wh-pc-btn-ghost" @click="openPeriodModal()">
              <LucidePlus :size="13" /> Periode
            </button>
            <button class="wh-pc-btn wh-pc-btn-primary" @click="openTrxModal()">
              <LucidePlus :size="13" /> Transaksi
            </button>
          </div>
        </div>

        <!-- Tab filter -->
        <div class="wh-pc-tabs">
          <button
            v-for="tab in (['Semua', 'Aktif', 'Selesai'] as const)"
            :key="tab"
            class="wh-pc-tab"
            :class="{ 'wh-pc-tab-active': periodFilter === tab }"
            @click="periodFilter = tab"
          >
            {{ tab }}
          </button>
        </div>

        <!-- Kartu periode -->
        <div class="wh-pc-period-list">
          <div
            v-for="p in filteredPeriods"
            :key="p.id"
            class="wh-pc-period-card"
            :class="{ 'wh-pc-period-card-active': String(p.id) === String(activePeriodId) }"
            @click="selectPeriod(p.id)"
          >
            <div class="wh-pc-period-card-top">
              <div class="wh-pc-period-nama">{{ p.nama }}</div>
              <span class="wh-pc-badge" :class="isPeriodAktif(p) ? 'wh-pc-badge-green' : 'wh-pc-badge-gray'">
                {{ isPeriodAktif(p) ? 'Aktif' : 'Selesai' }}
              </span>
            </div>
            <div class="wh-pc-period-date">
              {{ fmtTanggalPendek(p.tanggalMulai) }} — {{ fmtTanggalPendek(p.tanggalSelesai) }}
            </div>
            <div class="wh-pc-period-saldo">Saldo Awal: <b>{{ fmtRp(toNum(p.saldoAwal)) }}</b></div>
            <div class="wh-pc-period-actions">
              <button class="wh-pc-mini-btn" @click.stop="openPeriodModal(p)"><LucidePencil :size="11" /> Edit</button>
              <button v-if="isPeriodAktif(p)" class="wh-pc-mini-btn" @click.stop="closePeriod(p)">
                <LucideCircleStop :size="11" /> Tutup
              </button>
              <button v-else class="wh-pc-mini-btn wh-pc-mini-btn-green" @click.stop="reopenPeriod(p)">
                <LucideCirclePlay :size="11" /> Buka
              </button>
              <button class="wh-pc-mini-btn wh-pc-mini-btn-red" @click.stop="deletePeriod(p)"><LucideTrash2 :size="11" /> Hapus</button>
            </div>
          </div>
          <div v-if="!filteredPeriods.length" class="wh-pc-empty-line">
            Tidak ada periode {{ periodFilter === 'Semua' ? '' : periodFilter === 'Aktif' ? 'aktif ' : 'selesai ' }}di filter ini.
          </div>
        </div>
      </div>

      <!-- ── Panel Transaksi ── -->
      <div class="wh-pc-card wh-pc-trx-card">
        <div class="wh-pc-card-head wh-pc-trx-head">
          <div class="wh-pc-trx-heading">
            <div class="wh-pc-card-title">Transaksi: {{ activePeriod?.nama ?? '-' }}</div>
            <div v-if="activePeriod" class="wh-pc-trx-sub">
              {{ fmtTanggalPendek(activePeriod.tanggalMulai) }} s/d {{ fmtTanggalPendek(activePeriod.tanggalSelesai) }}
              &nbsp;|&nbsp; Saldo Awal: <b>{{ fmtRp(saldoAwal) }}</b>
            </div>
          </div>
          <div class="wh-pc-trx-actions">
            <div class="wh-pc-search">
              
              <input v-model="search" type="text" placeholder="Cari..." class="wh-pc-search-input" />
            </div>
            <button
              class="wh-pc-btn wh-pc-btn-ghost wh-pc-btn-sm"
              :class="{ 'wh-pc-btn-toggled': onlyBelum }"
              @click="onlyBelum = !onlyBelum"
            >
              {{ onlyBelum ? 'Menampilkan: Belum Bayar' : 'Belum Bayar' }}
            </button>
            <button class="wh-pc-btn wh-pc-btn-ghost wh-pc-btn-sm" @click="router.push('/fcl/keuangan/pettyCashKategori')"><LucideList :size="13" /> Kategori</button>
            <button class="wh-pc-btn wh-pc-btn-ghost wh-pc-btn-sm" @click="exportExcel"><LucideDownload :size="13" /> Export</button>
            <button class="wh-pc-btn wh-pc-btn-info wh-pc-btn-sm" @click="openInvoice"><LucideFilePenLine :size="13" /> Invoice</button>
            <button class="wh-pc-btn wh-pc-btn-teal wh-pc-btn-sm" @click="printDokumen"><LucideNotebookPen :size="13" /> Dokumen</button>
            <button class="wh-pc-btn wh-pc-btn-orange wh-pc-btn-sm" @click="printFoto"><LucidePrinter :size="13" /> Foto</button>
          </div>
        </div>

        <div class="wh-pc-table-wrap">
          <table class="wh-pc-table">
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
                  <div class="wh-pc-tanggal">
                    <span>{{ fmtTanggalDuaBaris(row.tanggal)[0] }}</span>
                    <span class="wh-pc-tanggal-year">{{ fmtTanggalDuaBaris(row.tanggal)[1] }}</span>
                  </div>
                </td>
                <td>
                  <span class="wh-pc-tipe" :class="row._tipe === 'Masuk' ? 'wh-pc-tipe-in' : 'wh-pc-tipe-out'">{{ row._tipe }}</span>
                </td>
                <td>
                  <span class="wh-pc-kat-chip" :style="{ '--chip': catColor(row.kategori) }">{{ (row.kategori || 'LAINNYA').toUpperCase() }}</span>
                </td>
                <td class="wh-pc-ket">{{ row.keterangan || '-' }}</td>
                <td class="r">
                  <span class="wh-pc-nominal" :class="row._tipe === 'Masuk' ? 'wh-pc-tone-green' : 'wh-pc-tone-red'">
                    {{ fmtSignedRp(row._nomSigned) }}
                  </span>
                </td>
                <td class="r"><span class="wh-pc-saldo">{{ fmtRp(row._saldo) }}</span></td>
                <td class="c">
                  <span class="wh-pc-statuspill" :class="row._lunas ? 'wh-pc-pill-lunas' : 'wh-pc-pill-belum'">
                    {{ row._lunas ? 'LUNAS' : 'BELUM BAYAR' }}
                  </span>
                </td>
                <td>
                  <div class="wh-pc-bukti">
                    <template v-if="row.buktiUrl">
                      <button class="wh-pc-bukti-link" @click="viewBukti(row)"><LucideEye :size="12" /> Lihat Foto</button>
                      <button class="wh-pc-mini-btn" :disabled="isDirectUploading(row)" @click="directUpload(row)">
                        <span v-if="isDirectUploading(row)" class="wh-pc-spin"></span>
                        {{ isDirectUploading(row) ? 'Mengunggah...' : 'Ganti Foto' }}
                      </button>
                      <button class="wh-pc-mini-btn" :disabled="isDirectUploading(row)" @click="openBuktiModal(row)">Link</button>
                    </template>
                    <button v-else class="wh-pc-bukti-upload" :disabled="isDirectUploading(row)" @click="directUpload(row)">
                      <span v-if="isDirectUploading(row)" class="wh-pc-spin"></span>
                      <LucidePlus v-else :size="12" />
                      {{ isDirectUploading(row) ? 'Mengunggah...' : 'Upload Langsung' }}
                    </button>
                  </div>
                </td>
                <td class="wh-pc-by">{{ row.createdBy || '-' }}</td>
                <td class="c">
                  <div class="wh-pc-aksi">
                    <button v-if="!row._lunas" class="wh-pc-btn wh-pc-btn-orange wh-pc-btn-xs" @click="markLunas(row)">
                      <LucideCheck :size="12" /> Selesai Bayar
                    </button>
                    <button class="wh-pc-btn wh-pc-btn-ghost wh-pc-btn-xs" @click="openPindah(row)"><LucideRefreshCw :size="12" /> Pindah</button>
                    <button class="wh-pc-btn wh-pc-btn-ghost wh-pc-btn-xs" @click="openTrxModal(row)"><LucidePencil :size="12" /> Edit</button>
                    <button class="wh-pc-btn wh-pc-btn-red wh-pc-btn-xs" @click="deleteTrx(row)"><LucideTrash2 :size="12" /> Hapus</button>
                  </div>
                </td>
              </tr>
              <tr v-if="!txRows.length">
                <td colspan="11" class="wh-pc-empty-cell">
                  {{ search || onlyBelum ? 'Tidak ada transaksi cocok dengan filter.' : 'Belum ada transaksi di periode ini — klik “+ Transaksi”.' }}
                </td>
              </tr>
            </tbody>
            <tfoot>
              <tr class="wh-pc-total-row">
                <td colspan="5" class="r">TOTAL</td>
                <td class="r">
                  <div class="wh-pc-foot-line wh-pc-tone-red">OUT: {{ fmtRp(totalOut) }}</div>
                  <div class="wh-pc-foot-line wh-pc-tone-green">IN: {{ fmtRp(totalIn) }}</div>
                </td>
                <td class="r" colspan="4"><span class="wh-pc-foot-sisa">Sisa: {{ fmtRp(sisaSaldo) }}</span></td>
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
      <a-form layout="vertical" class="wh-pc-form">
        <a-form-item label="Nama Periode *" required>
          <a-input v-model:value="periodForm.nama" placeholder="Contoh: 16 November–22 November 2026" />
        </a-form-item>
        <div class="wh-pc-form-row">
          <a-form-item label="Tanggal Mulai">
            <input v-model="periodForm.tanggalMulai" type="date" class="wh-pc-input" />
          </a-form-item>
          <a-form-item label="Tanggal Selesai">
            <input v-model="periodForm.tanggalSelesai" type="date" class="wh-pc-input" />
          </a-form-item>
        </div>
        <div class="wh-pc-form-row">
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
      <a-form layout="vertical" class="wh-pc-form">
        <div class="wh-pc-form-row">
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
        <div class="wh-pc-form-row">
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
      <div class="wh-pc-bukti-tabs">
        <button :class="{ 'active': buktiMode === 'link' }" @click="buktiMode = 'link'">Link Google Drive</button>
        <button :class="{ 'active': buktiMode === 'upload' }" @click="buktiMode = 'upload'">Upload Langsung</button>
      </div>
      <template v-if="buktiMode === 'link'">
        <p class="wh-pc-bukti-hint">Masukkan link foto/struk dari Google Drive (kosongkan untuk menghapus).</p>
        <a-input v-model:value="buktiInput" placeholder="https://drive.google.com/..." allow-clear />
      </template>
      <template v-else>
        <p class="wh-pc-bukti-hint">Pilih foto/struk dari perangkat (maks. 12 MB). File langsung tersimpan ke folder Google Drive sesuai nama periode ini.</p>
        <input type="file" accept="image/*,application/pdf" class="wh-pc-file" @change="pickBuktiFile" />
        <div v-if="buktiPreview" class="wh-pc-bukti-preview"><img :src="buktiPreview" alt="Pratinjau bukti" /></div>
        <p v-if="buktiTarget?.buktiUrl && !buktiFile" class="wh-pc-bukti-hint">Biarkan kosong untuk menggunakan file yang sudah tersimpan.</p>
      </template>
    </a-modal>

    <!-- ══════════ Modal: Pindah Transaksi ke Periode Lain ══════════ -->
    <a-modal
      v-model:open="pindahOpen"
      title="Pindah Transaksi ke Periode Lain"
      ok-text="Pindahkan"
      cancel-text="Batal"
      :confirm-loading="pindahSaving"
      width="460px"
      @ok="savePindah"
    >
      <p class="wh-pc-bukti-hint">
        Transaksi: <b>{{ pindahTarget?.keterangan || pindahTarget?.kategori || '-' }}</b>
        &nbsp;•&nbsp; {{ fmtRp(toNum(pindahTarget?.nominal)) }}
      </p>
      <a-select
        v-model:value="pindahPeriodId"
        :options="periods.map((p) => ({ label: p.nama, value: String(p.id) }))"
        placeholder="Pilih periode tujuan..."
        style="width: 100%"
        show-search
        option-filter-prop="label"
      />
      <p class="wh-pc-bukti-hint">Saldo periode akan dihitung ulang otomatis setelah transaksi dipindahkan.</p>
    </a-modal>

    <!-- ══════════ Modal: Pratinjau Foto Bukti (inline) ══════════ -->
    <a-modal
      v-model:open="viewBuktiOpen"
      :title="viewBuktiLabel ? `Foto Bukti — ${viewBuktiLabel}` : 'Foto Bukti'"
      :footer="null"
      width="760px"
      centered
    >
      <div class="wh-pc-viewbukti">
        <template v-if="!viewBuktiFailed">
          <img
            v-if="!viewBuktiIsPdf"
            :src="viewBuktiSrc"
            alt="Foto bukti transaksi"
            class="wh-pc-viewbukti-img"
            @error="onViewBuktiError"
          />
          <iframe
            v-else
            :src="viewBuktiSrc"
            class="wh-pc-viewbukti-frame"
            title="Dokumen bukti transaksi"
          ></iframe>
        </template>
        <div v-else class="wh-pc-viewbukti-error">
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
      <a-form layout="vertical" class="wh-pc-form">
        <a-form-item label="Judul Invoice">
          <a-input v-model:value="invoiceTitle" />
        </a-form-item>
        <a-form-item label="Catatan (opsional)">
          <a-input v-model:value="invoiceNotes" placeholder="Catatan untuk invoice" />
        </a-form-item>
        <div class="wh-pc-invoice-sel-row">
          <span class="wh-pc-invoice-sel-count">{{ invoiceTxs.length }} transaksi dipilih</span>
          <div>
            <a-button size="small" @click="toggleInvoiceAll(true)">Pilih Semua</a-button>
            <a-button size="small" style="margin-left: 6px" @click="toggleInvoiceAll(false)">Kosongkan</a-button>
          </div>
        </div>
        <div class="wh-pc-invoice-list">
          <label v-for="t in periodTxs" :key="t.id" class="wh-pc-invoice-item">
            <input type="checkbox" :checked="invoiceSelected.has(String(t.id))" @change="toggleInvoiceTx(String(t.id))" />
            <span class="wh-pc-invoice-item-body">
              <span class="wh-pc-invoice-item-title">{{ (t.tanggal ?? '').slice(0, 10) }} · {{ t.kategori }}</span>
              <span class="wh-pc-invoice-item-ket">{{ t.keterangan }}</span>
              <span class="wh-pc-invoice-item-nom">Rp {{ toNum(t.nominal).toLocaleString('id-ID') }}</span>
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
.wh-pc {
  display: flex;
  flex-direction: column;
  gap: 12px;
  color: hsl(var(--foreground));
  font-size: 13px;
}

/* ── Hero (belum ada periode) ── */
.wh-pc-hero {
  background: hsl(var(--card));
  border: 1px dashed hsl(var(--border));
  border-radius: 10px;
  padding: 48px 24px;
  text-align: center;
  color: hsl(var(--foreground));
}
.wh-pc-hero-icon { color: hsl(var(--success)); display: flex; justify-content: center; margin-bottom: 12px; }
.wh-pc-hero-title { font-size: 18px; font-weight: 800; margin-bottom: 8px; }
.wh-pc-hero-desc { color: hsl(var(--muted-foreground)); max-width: 520px; margin: 0 auto 18px; line-height: 1.7; }

/* ── Bar statistik ── */
.wh-pc-stats { display: grid; grid-template-columns: repeat(4, minmax(0, 1fr)); gap: 10px; }
.wh-pc-stat {
  background: hsl(var(--card));
  border: 1px solid hsl(var(--border));
  border-top: 3px solid hsl(var(--muted-foreground));
  border-radius: 8px;
  padding: 12px 14px;
}
.wh-pc-stat-green { border-top-color: hsl(var(--success)); }
.wh-pc-stat-red { border-top-color: hsl(var(--destructive)); }
.wh-pc-stat-teal { border-top-color: hsl(var(--success)); }
.wh-pc-stat-amber { border-top-color: hsl(var(--warning)); }
.wh-pc-stat-label {
  font-size: 11px;
  font-weight: 700;
  letter-spacing: 0.04em;
  color: hsl(var(--muted-foreground));
  margin-bottom: 4px;
}
.wh-pc-stat-value { font-size: 20px; font-weight: 800; color: hsl(var(--foreground)); }
.wh-pc-tone-green { color: hsl(var(--success)) !important; }
.wh-pc-tone-red { color: hsl(var(--destructive)) !important; }
.wh-pc-tone-teal { color: hsl(var(--success)) !important; }
.wh-pc-tone-amber { color: hsl(var(--warning)) !important; }

/* ── Kartu / panel ── */
.wh-pc-card {
  background: hsl(var(--card));
  border: 1px solid hsl(var(--border));
  border-radius: 8px;
  padding: 12px;
}
.wh-pc-card-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 10px;
  flex-wrap: wrap;
  padding: 0 0 10px;
}
.wh-pc-card-title { font-size: 14px; font-weight: 800; color: hsl(var(--foreground)); }
.wh-pc-card-actions { display: flex; gap: 6px; flex-wrap: wrap; }

/* ── Tombol ── */
.wh-pc-btn {
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
.wh-pc-btn:hover { filter: brightness(0.96); }
.wh-pc-btn-lg { font-size: 14px; padding: 10px 20px; }
.wh-pc-btn-sm { padding: 4px 10px; font-size: 11px; }
.wh-pc-btn-xs { padding: 3px 8px; font-size: 11px; }
.wh-pc-btn-primary,
.wh-pc-btn-info,
.wh-pc-btn-orange { background: hsl(var(--primary)); border-color: hsl(var(--primary)); color: hsl(var(--primary-foreground)); }
.wh-pc-btn-ghost { background: hsl(var(--muted)); border-color: hsl(var(--border)); }
.wh-pc-btn-teal { background: hsl(var(--success)); border-color: hsl(var(--success)); color: hsl(var(--success-foreground)); }
.wh-pc-btn-red { background: hsl(var(--destructive)); border-color: hsl(var(--destructive)); color: hsl(var(--destructive-foreground)); }
.wh-pc-btn-toggled { background: hsl(var(--warning) / 0.18); border-color: hsl(var(--warning)); color: hsl(var(--warning)); }

/* ── Tab filter ── */
.wh-pc-tabs { display: flex; gap: 4px; padding: 0 0 10px; }
.wh-pc-tab {
  border: 1px solid transparent;
  background: hsl(var(--muted));
  color: hsl(var(--muted-foreground));
  font-size: 11px;
  font-weight: 700;
  padding: 4px 14px;
  border-radius: 6px;
  cursor: pointer;
}
.wh-pc-tab:hover { color: hsl(var(--foreground)); }
.wh-pc-tab-active { background: hsl(var(--primary) / 0.12); border-color: hsl(var(--primary) / 0.4); color: hsl(var(--primary)); }

/* ── Kartu periode ── */
.wh-pc-period-list { display: flex; flex-wrap: wrap; gap: 10px; }
.wh-pc-period-card {
  flex: 1 1 230px;
  min-width: 220px;
  max-width: 300px;
  background: hsl(var(--background));
  border: 1px solid hsl(var(--border));
  border-radius: 8px;
  padding: 10px 12px;
  cursor: pointer;
}
.wh-pc-period-card:hover { border-color: hsl(var(--primary) / 0.5); }
.wh-pc-period-card-active { border-color: hsl(var(--primary)); background: hsl(var(--primary) / 0.06); }
.wh-pc-period-card-top { display: flex; justify-content: space-between; align-items: flex-start; gap: 8px; }
.wh-pc-period-nama { font-size: 12px; font-weight: 800; color: hsl(var(--foreground)); line-height: 1.4; }
.wh-pc-period-card-active .wh-pc-period-nama { color: hsl(var(--primary)); }
.wh-pc-period-date { font-size: 11px; color: hsl(var(--muted-foreground)); margin-top: 4px; }
.wh-pc-period-saldo { font-size: 11px; color: hsl(var(--muted-foreground)); margin-top: 2px; }
.wh-pc-period-saldo b { color: hsl(var(--success)); }
.wh-pc-period-actions { display: flex; gap: 5px; margin-top: 10px; }
.wh-pc-badge { font-size: 10px; font-weight: 800; padding: 2px 8px; border-radius: 999px; white-space: nowrap; }
.wh-pc-badge-green { color: hsl(var(--success)); background: hsl(var(--success) / 0.14); }
.wh-pc-badge-gray { color: hsl(var(--muted-foreground)); background: hsl(var(--muted)); }
.wh-pc-mini-btn {
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
.wh-pc-mini-btn:hover { filter: brightness(0.96); }
.wh-pc-mini-btn-red { border-color: hsl(var(--destructive) / 0.5); color: hsl(var(--destructive)); }
.wh-pc-mini-btn-green { border-color: hsl(var(--success) / 0.5); color: hsl(var(--success)); }
.wh-pc-empty-line { color: hsl(var(--muted-foreground)); font-size: 12px; padding: 10px 0; }

/* ── Panel transaksi ── */
.wh-pc-trx-head { align-items: flex-start; }
.wh-pc-trx-heading { display: flex; flex-direction: column; gap: 2px; }
.wh-pc-trx-sub { font-size: 11px; color: hsl(var(--muted-foreground)); }
.wh-pc-trx-sub b { color: hsl(var(--success)); }
.wh-pc-trx-actions { display: flex; gap: 6px; flex-wrap: wrap; align-items: center; }
.wh-pc-search-input {
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
.wh-pc-search-input:focus { border-color: hsl(var(--primary)); }

/* ── Tabel (header mengikuti tema, tidak hitam) ── */
.wh-pc-table-wrap { overflow-x: auto; border-top: 1px solid hsl(var(--border)); }
.wh-pc-table { width: 100%; min-width: 1060px; border-collapse: collapse; font-size: 12px; }
.wh-pc-table th {
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
.wh-pc-table th.r, .wh-pc-table td.r { text-align: right; }
.wh-pc-table th.c, .wh-pc-table td.c { text-align: center; }
.wh-pc-table th.c-no, .wh-pc-table td.c-no { width: 36px; text-align: center; color: hsl(var(--muted-foreground)); }
.wh-pc-table td {
  padding: 7px 8px;
  border-bottom: 1px solid hsl(var(--border));
  vertical-align: middle;
  white-space: nowrap;
  color: hsl(var(--foreground));
}
.wh-pc-table tbody tr:nth-child(even) { background: hsl(var(--muted) / 0.45); }
.wh-pc-table tbody tr:hover { background: hsl(var(--accent)); }
.wh-pc-tanggal { display: flex; flex-direction: column; line-height: 1.25; font-weight: 700; }
.wh-pc-tanggal-year { color: hsl(var(--muted-foreground)); font-weight: 500; font-size: 10px; }
.wh-pc-tipe { font-weight: 800; }
.wh-pc-tipe-out { color: hsl(var(--destructive)); }
.wh-pc-tipe-in { color: hsl(var(--success)); }
.wh-pc-kat-chip {
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
.wh-pc-kat-chip::before { content: ''; width: 6px; height: 6px; border-radius: 99px; background: var(--chip, hsl(var(--primary))); }
.wh-pc-ket { max-width: 260px; overflow: hidden; text-overflow: ellipsis; }
.wh-pc-nominal { font-weight: 800; }
.wh-pc-saldo { color: hsl(var(--success)); font-weight: 700; }
.wh-pc-statuspill { display: inline-block; font-size: 10px; font-weight: 800; letter-spacing: 0.04em; padding: 3px 10px; border-radius: 5px; white-space: nowrap; }
.wh-pc-pill-belum { background: hsl(var(--warning) / 0.14); color: hsl(var(--warning)); border: 1px solid hsl(var(--warning) / 0.4); }
.wh-pc-pill-lunas { background: hsl(var(--success) / 0.14); color: hsl(var(--success)); border: 1px solid hsl(var(--success) / 0.4); }
.wh-pc-bukti { display: flex; align-items: center; gap: 6px; }
.wh-pc-bukti-link { display: inline-flex; align-items: center; gap: 5px; color: hsl(var(--primary)); font-size: 11px; font-weight: 700; background: none; border: none; cursor: pointer; padding: 0; }
.wh-pc-bukti-upload { display: inline-flex; align-items: center; gap: 5px; border: 1px dashed hsl(var(--border)); background: transparent; color: hsl(var(--muted-foreground)); font-size: 11px; font-weight: 700; border-radius: 6px; padding: 4px 10px; cursor: pointer; }
.wh-pc-bukti-upload:hover { color: hsl(var(--foreground)); border-color: hsl(var(--primary)); }
.wh-pc-bukti-hint { color: hsl(var(--muted-foreground)); font-size: 12px; margin-bottom: 10px; }
.wh-pc-viewbukti { display: flex; align-items: center; justify-content: center; min-height: 220px; background: hsl(var(--muted)); border: 1px solid hsl(var(--border)); border-radius: 8px; overflow: hidden; }
.wh-pc-viewbukti-img { display: block; max-width: 100%; max-height: 65vh; width: auto; height: auto; object-fit: contain; }
.wh-pc-viewbukti-frame { display: block; width: 100%; height: 65vh; border: 0; }
.wh-pc-viewbukti-error { display: flex; flex-direction: column; align-items: center; gap: 10px; padding: 32px 20px; text-align: center; color: hsl(var(--muted-foreground)); font-size: 13px; }
.wh-pc-viewbukti-error a { color: hsl(var(--primary)); font-weight: 700; }
.wh-pc-by { color: hsl(var(--muted-foreground)); max-width: 110px; overflow: hidden; text-overflow: ellipsis; }
.wh-pc-aksi { display: flex; gap: 4px; align-items: center; justify-content: center; }
.wh-pc-empty-cell { text-align: center; color: hsl(var(--muted-foreground)); padding: 22px 8px; }

/* ── Footer ── */
.wh-pc-total-row td { background: hsl(var(--muted)); border-top: 1px solid hsl(var(--border)); padding: 8px; }
.wh-pc-foot-line { font-size: 11px; font-weight: 800; line-height: 1.5; }
.wh-pc-foot-sisa { font-size: 11px; font-weight: 800; color: hsl(var(--warning)); }

/* ── Form & invoice ── */
.wh-pc-form { padding-top: 6px; }
.wh-pc-form-row { display: grid; grid-template-columns: 1fr 1fr; gap: 12px; }
.wh-pc-input {
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
.wh-pc-input:focus { border-color: hsl(var(--primary)); }
.wh-pc-invoice-sel-row { display: flex; align-items: center; justify-content: space-between; margin-bottom: 8px; }
.wh-pc-invoice-sel-count { font-size: 11px; color: hsl(var(--muted-foreground)); }
.wh-pc-invoice-list { max-height: 280px; overflow-y: auto; display: flex; flex-direction: column; gap: 6px; }
.wh-pc-invoice-item { display: flex; gap: 8px; align-items: flex-start; background: hsl(var(--muted)); border: 1px solid hsl(var(--border)); border-radius: 6px; padding: 6px 10px; cursor: pointer; }
.wh-pc-invoice-item input { margin-top: 3px; }
.wh-pc-invoice-item-body { display: flex; flex-direction: column; }
.wh-pc-invoice-item-title { font-weight: 700; font-size: 12px; }
.wh-pc-invoice-item-ket { color: hsl(var(--muted-foreground)); font-size: 11px; }
.wh-pc-invoice-item-nom { font-size: 11px; margin-top: 2px; }

@media (max-width: 1100px) {
  .wh-pc-stats { grid-template-columns: repeat(2, minmax(0, 1fr)); }
  .wh-pc-period-card { flex: 1 1 200px; }
}

.wh-pc-bukti-tabs { display: flex; gap: 6px; margin-bottom: 10px; }
.wh-pc-bukti-tabs button { border: 1px solid hsl(var(--border)); background: hsl(var(--muted)); color: hsl(var(--muted-foreground)); font-size: 11px; font-weight: 700; padding: 5px 12px; border-radius: 6px; cursor: pointer; }
.wh-pc-bukti-tabs button.active { background: hsl(var(--primary)); border-color: hsl(var(--primary)); color: hsl(var(--primary-foreground)); }
.wh-pc-file { font-size: 11px; width: 100%; color: hsl(var(--muted-foreground)); }
.wh-pc-bukti-preview { margin-top: 10px; max-height: 240px; overflow: hidden; border: 1px solid hsl(var(--border)); border-radius: 6px; }
.wh-pc-bukti-preview img { max-width: 100%; max-height: 240px; object-fit: contain; display: block; }
.wh-pc-spin { display: inline-block; width: 11px; height: 11px; flex: 0 0 auto; border: 2px solid hsl(var(--border)); border-top-color: hsl(var(--primary)); border-radius: 50%; animation: wh-pc-spin 0.7s linear infinite; }
@keyframes wh-pc-spin { to { transform: rotate(360deg); } }
.wh-pc-upload-status { position: fixed; right: 22px; bottom: 22px; z-index: 2000; display: flex; align-items: center; gap: 9px; padding: 9px 15px; border: 1px solid hsl(var(--border)); border-radius: 8px; background: hsl(var(--popover)); color: hsl(var(--popover-foreground)); box-shadow: 0 6px 20px rgba(0,0,0,.18); font-size: 12px; font-weight: 700; }
.wh-pc-upload-status .wh-pc-spin { width: 14px; height: 14px; }
button:disabled { opacity: 0.65; cursor: not-allowed; }
</style>
