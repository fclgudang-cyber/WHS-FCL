<script lang="ts" setup>
/**
 * Manajemen Order (FCL) — layout & fitur mengikuti aplikasi Orderan asli:
 * header + dua pencarian, aksi Template/Impor/Ekspor/Buat Order,
 * tab kategori Marketplace | Distributor | Store | Riwayat (status Terkirim),
 * tabel order dengan baris detail (expand) dan aksi Validasi / Kirim / Edit /
 * Hapus / Bukti / Print, serta form order + item barang.
 * Seluruh warna memakai token tema vben (hsl(var(--...))).
 */
import { computed, onMounted, ref, watch } from 'vue';

import { Page } from '@vben/common-ui';
import {
  LucideCheck,
  LucideChevronDown,
  LucideDownload,
  LucideEye,
  LucideFilePenLine,
  LucidePencil,
  LucidePlus,
  LucidePrinter,
  LucideSend,
  LucideTrash2,
} from '@vben/icons';

import { Modal, message, notification } from 'ant-design-vue';
import dayjs from 'dayjs';

import { fclDelete, fclInsert, fclPatch, fclQuery } from '#/api/fcl';
import FclPagination from '#/components/fcl/fcl-pagination.vue';
import type { StockRow } from '#/utils/fcl-stock';

interface OrderRow {
  id: string;
  noOrder?: string;
  tanggal?: string;
  pelanggan?: string;
  alamat?: string;
  kategori?: string;
  noResi?: string;
  status?: string;
  totalItem?: string | number;
  keterangan?: string;
  sentAt?: string;
  buktiPacking?: string;
  createdBy?: string;
  createdAt?: string;
}
interface DetailRow {
  id: string;
  orderId?: string;
  noOrder?: string;
  sku?: string;
  nama?: string;
  qty?: string | number;
  satuan?: string;
  batch?: string;
  expDate?: string;
  lokasi?: string;
  packedQty?: string;
}

type KategoriTab = 'MP' | 'Dist' | 'Store' | 'History';

const loading = ref(true);
const orders = ref<OrderRow[]>([]);
const details = ref<DetailRow[]>([]);
const stockRows = ref<StockRow[]>([]);
const searchText = ref('');
const searchSku = ref('');
const activeTab = ref<KategoriTab>('MP');
const expanded = ref<Set<string>>(new Set());

function num(v: unknown): number {
  const n = Number(v);
  return Number.isFinite(n) ? n : 0;
}

function esc(s: unknown): string {
  return String(s ?? '')
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;');
}

function katOf(o?: OrderRow | null): KategoriTab {
  const s = String(o?.kategori ?? '').toLowerCase().trim();
  if (s.includes('marketplace') || s === 'mp' || s === 'market place') return 'MP';
  if (s === 'store' || s.includes('toko')) return 'Store';
  return 'Dist';
}

function isSent(o?: OrderRow | null): boolean {
  return String(o?.status ?? '').toLowerCase() === 'terkirim';
}

const counts = computed(() => {
  const c: Record<KategoriTab, number> = { MP: 0, Dist: 0, Store: 0, History: 0 };
  orders.value.forEach((o) => {
    if (isSent(o)) c.History = (c.History ?? 0) + 1;
    else {
      const tab = katOf(o);
      c[tab] = (c[tab] ?? 0) + 1;
    }
  });
  return c;
});

const filteredOrders = computed(() => {
  const q = searchText.value.trim().toLowerCase();
  const sku = searchSku.value.trim().toLowerCase();
  let list = orders.value.filter((o) => {
    if (activeTab.value === 'History') {
      if (!isSent(o)) return false;
    } else {
      if (isSent(o) || katOf(o) !== activeTab.value) return false;
    }
    if (q && ![o.noOrder, o.pelanggan, o.noResi, o.alamat]
      .some((v) => String(v ?? '').toLowerCase().includes(q))) return false;
    return true;
  });
  if (sku) {
    const matchedIds = new Set(
      details.value
        .filter((d) => String(d.sku ?? '').toLowerCase().includes(sku) || String(d.nama ?? '').toLowerCase().includes(sku))
        .map((d) => String(d.orderId)),
    );
    list = list.filter((o) => matchedIds.has(String(o.id)));
  }
  return list;
});

const page = ref(1);
const pageSize = ref(20);
const pagedOrders = computed(() =>
  filteredOrders.value.slice((page.value - 1) * pageSize.value, page.value * pageSize.value),
);

watch([searchText, searchSku, activeTab], () => { page.value = 1; });

function detailsOf(o: OrderRow): DetailRow[] {
  return details.value.filter((d) => String(d.orderId) === String(o.id) || String(d.noOrder) === String(o.noOrder));
}

function itemCount(o: OrderRow): number {
  const ds = detailsOf(o);
  return ds.length || num(o.totalItem);
}

function toggleExpand(id: string) {
  const next = new Set(expanded.value);
  if (next.has(id)) next.delete(id);
  else next.add(id);
  expanded.value = next;
}

async function loadAll(silent = false) {
  if (!silent) loading.value = true;
  try {
    const [o, d, st] = await Promise.all([
      fclQuery<OrderRow>('order', { select: '*', order: 'createdAt.desc', limit: 10000 }),
      fclQuery<DetailRow>('orderdetail', { select: '*', limit: 20000 }),
      fclQuery<StockRow>('stock', { select: '*', order: 'sku.asc', limit: 20000 }),
    ]);
    orders.value = o;
    details.value = d;
    stockRows.value = st;
  } catch (err: any) {
    console.error('[fcl/order] load gagal', err);
    notification.error({ message: 'Manajemen Order', description: err?.message || 'Gagal memuat data order' });
  } finally {
    if (!silent) loading.value = false;
  }
}

// ── Form order + item ────────────────────────────────────────────────────────
const orderModalOpen = ref(false);
const orderSaving = ref(false);
const editingOrder = ref<OrderRow | null>(null);
const form = ref({
  noOrder: '',
  tanggal: dayjs().format('YYYY-MM-DD'),
  pelanggan: '',
  alamat: '',
  kategori: 'Distributor',
  noResi: '',
  keterangan: '',
  status: 'Menunggu',
});
const items = ref<Array<{ sku: string; nama: string; qty: number | undefined; satuan: string; batch: string; expDate: string }>>([]);
const scanInput = ref('');

function newItem() {
  return { sku: '', nama: '', qty: undefined as number | undefined, satuan: 'pcs', batch: '', expDate: '' };
}

function genNoOrder(): string {
  return `ORD-${dayjs().format('YYYYMMDD')}-${String(Math.floor(Math.random() * 9000) + 1000)}`;
}

function openOrderModal(o?: OrderRow | null) {
  editingOrder.value = o ?? null;
  form.value = o
    ? {
        noOrder: o.noOrder ?? '',
        tanggal: (o.tanggal ?? '').slice(0, 10) || dayjs().format('YYYY-MM-DD'),
        pelanggan: o.pelanggan ?? '',
        alamat: o.alamat ?? '',
        kategori: o.kategori ?? 'Distributor',
        noResi: o.noResi ?? '',
        keterangan: o.keterangan ?? '',
        status: o.status ?? 'Menunggu',
      }
    : {
        noOrder: genNoOrder(),
        tanggal: dayjs().format('YYYY-MM-DD'),
        pelanggan: '',
        alamat: '',
        kategori: 'Distributor',
        noResi: '',
        keterangan: '',
        status: 'Menunggu',
      };
  items.value = o ? detailsOf(o).map((d) => ({
    sku: d.sku ?? '', nama: d.nama ?? '', qty: num(d.qty), satuan: d.satuan ?? 'pcs', batch: d.batch ?? '', expDate: (d.expDate ?? '').slice(0, 10),
  })) : [newItem()];
  if (!items.value.length) items.value = [newItem()];
  orderModalOpen.value = true;
}

function addScanItem() {
  const text = scanInput.value.trim().toLowerCase();
  if (!text) return;
  const found = stockRows.value.find((r) =>
    String(r.sku ?? '').toLowerCase() === text ||
    String(r.barcode ?? '').toLowerCase() === text,
  );
  if (!found) {
    message.warning('SKU / barcode tidak ditemukan di Stock Barang');
    return;
  }
  const item = { ...newItem(), sku: String(found.sku ?? '') };
  onSkuChange(item, item.sku);
  item.qty = 1;
  items.value.push(item);
  scanInput.value = '';
}

// ── Sinkronisasi dengan Stock Barang ────────────────────────────────────────
const stockSkuOptions = computed(() => {
  const map = new Map<string, { sku: string; nama: string; satuan: string; total: number }>();
  for (const r of stockRows.value) {
    const sku = String(r.sku ?? '').trim();
    const qty = num(r.stok);
    if (!sku || qty <= 0) continue;
    const cur = map.get(sku) ?? { sku, nama: String(r.nama ?? ''), satuan: String(r.satuan ?? 'pcs'), total: 0 };
    cur.total += qty;
    if (!cur.nama && r.nama) cur.nama = String(r.nama);
    map.set(sku, cur);
  }
  return [...map.values()].map((v) => ({
    label: `${v.sku} — ${v.nama || '-'} (stok ${v.total})`,
    value: v.sku,
  }));
});

function batchOptions(sku: string) {
  const seen = new Set<string>();
  const out: Array<{ label: string; value: string }> = [];
  for (const r of stockRows.value) {
    if (String(r.sku ?? '') !== sku) continue;
    const batch = String(r.batch ?? '').trim();
    if (!batch || seen.has(batch)) continue;
    seen.add(batch);
    out.push({ label: `${batch}${r.expDate ? ` · exp ${String(r.expDate).slice(0, 10)}` : ''}`, value: batch });
  }
  return out;
}

function stockKey(item: { sku: string; batch: string }): string {
  return `${String(item.sku ?? '').trim()}|${String(item.batch ?? '').trim()}`;
}

function availableStock(item: { sku: string; batch: string }): number {
  const sku = String(item.sku ?? '').trim();
  const batch = String(item.batch ?? '').trim();
  if (!sku) return 0;
  return stockRows.value
    .filter((r) => String(r.sku ?? '') === sku && (!batch || String(r.batch ?? '') === batch))
    .reduce((sum, r) => sum + num(r.stok), 0);
}

function onSkuChange(item: { sku: string; nama: string; satuan: string; batch: string; expDate: string }, sku: string) {
  const match = stockRows.value.find((r) => String(r.sku ?? '') === sku && num(r.stok) > 0)
    ?? stockRows.value.find((r) => String(r.sku ?? '') === sku);
  if (!match) {
    item.batch = '';
    item.expDate = '';
    return;
  }
  item.nama = String(match.nama ?? '');
  item.satuan = String(match.satuan ?? 'pcs');
  item.batch = String(match.batch ?? '');
  item.expDate = String(match.expDate ?? '').slice(0, 10);
}

async function saveOrder() {
  if (!form.value.noOrder.trim()) { message.warning('No. order wajib diisi'); return; }
  if (!form.value.tanggal) { message.warning('Tanggal wajib diisi'); return; }
  if (editingOrder.value && form.value.noOrder !== editingOrder.value.noOrder) {
    if (orders.value.some((o) => String(o.id) !== String(editingOrder.value!.id) && o.noOrder === form.value.noOrder)) {
      message.warning('No. order sudah dipakai order lain');
      return;
    }
  }
  const validItems = items.value.filter((i) => i.sku || i.nama || num(i.qty) > 0);
  const usage = new Map<string, number>();
  for (const i of validItems) {
    const sku = String(i.sku ?? '').trim();
    if (!sku) {
      message.warning('Pilih SKU dari Stock Barang untuk setiap item');
      return;
    }
    if (!stockRows.value.some((r) => String(r.sku ?? '') === sku)) {
      message.warning(`SKU ${sku} tidak ditemukan di Stock Barang`);
      return;
    }
    if (num(i.qty) <= 0) {
      message.warning(`Qty ${sku} harus lebih dari 0`);
      return;
    }
    const key = stockKey(i as any);
    usage.set(key, (usage.get(key) ?? 0) + num(i.qty));
  }
  for (const [key, qty] of usage) {
    const parts = key.split('|');
    const sku = parts[0] ?? '';
    const batch = parts[1] ?? '';
    const avail = availableStock({ sku, batch });
    if (qty > avail) {
      message.warning(`Stok ${sku}${batch ? ` batch ${batch}` : ''} tidak cukup. Tersedia ${avail}, diminta ${qty}`);
      return;
    }
  }

  orderSaving.value = true;
  const total = items.value.reduce((s, i) => s + num(i.qty), 0);
  const payload: Record<string, any> = {
    noOrder: form.value.noOrder.trim(),
    tanggal: form.value.tanggal,
    pelanggan: form.value.pelanggan,
    alamat: form.value.alamat,
    kategori: form.value.kategori,
    noResi: form.value.noResi,
    keterangan: form.value.keterangan,
    status: form.value.status,
    totalItem: String(total),
  };
  try {
    let orderId = editingOrder.value?.id;
    if (editingOrder.value) {
      await fclPatch('order', editingOrder.value.id, payload);
    } else {
      const created = (await fclInsert('order', [payload])) as any;
      orderId = Array.isArray(created) ? created?.[0]?.id : created?.id;
    }
    if (!orderId) throw new Error('Gagal mendapatkan id order');

    // Simpan item: edit = hapus lama lalu insert ulang
    if (editingOrder.value) {
      const oldItems = details.value.filter((d) => String(d.orderId) === String(orderId));
      for (const d of oldItems) await fclDelete('orderdetail', d.id);
    }
    const rows = items.value
      .filter((i) => i.sku || i.nama || num(i.qty) > 0)
      .map((i) => ({
        orderId: String(orderId),
        noOrder: form.value.noOrder.trim(),
        sku: i.sku,
        nama: i.nama,
        qty: String(num(i.qty)),
        satuan: i.satuan,
        batch: i.batch,
        expDate: i.expDate,
      }));
    if (rows.length) await fclInsert('orderdetail', rows);

    notification.success({ message: 'Order', description: editingOrder.value ? 'Order diperbarui' : 'Order dibuat' });
    orderModalOpen.value = false;
    await loadAll(true);
  } catch (err: any) {
    notification.error({ message: 'Order', description: err?.message || 'Gagal menyimpan order' });
  } finally {
    orderSaving.value = false;
  }
}

async function deleteOrder(o: OrderRow) {
  Modal.confirm({
    title: 'Hapus order?',
    content: `Hapus order "${o.noOrder}" beserta detail itemnya?`,
    okText: 'Hapus',
    okType: 'danger',
    onOk: async () => {
      const ds = detailsOf(o);
      for (const d of ds) await fclDelete('orderdetail', d.id);
      await fclDelete('order', o.id);
      notification.success({ message: 'Order dihapus' });
      await loadAll(true);
    },
  });
}

async function markStatus(o: OrderRow, status: string, extra: Record<string, any> = {}) {
  await fclPatch('order', o.id, { status, ...extra });
  notification.success({ message: 'Order', description: `Status diubah menjadi ${status}` });
  await loadAll(true);
}

function viewBukti(o: OrderRow) {
  if (o.buktiPacking) window.open(o.buktiPacking, '_blank', 'noopener');
}

// ── Bukti packing ────────────────────────────────────────────────────────────
const buktiModalOpen = ref(false);
const buktiTarget = ref<OrderRow | null>(null);
const buktiInput = ref('');

function openBukti(o: OrderRow) {
  buktiTarget.value = o;
  buktiInput.value = o.buktiPacking ?? '';
  buktiModalOpen.value = true;
}

async function saveBukti() {
  if (!buktiTarget.value) return;
  await fclPatch('order', buktiTarget.value.id, { buktiPacking: buktiInput.value.trim() });
  notification.success({ message: 'Bukti packing disimpan' });
  buktiModalOpen.value = false;
  await loadAll(true);
}

// ── Template / Impor / Ekspor ────────────────────────────────────────────────
function downloadCsv(filename: string, lines: string[][]) {
  const csv = lines.map((cols) => cols.map((c) => `"${String(c ?? '').replace(/"/g, '""')}"`).join(',')).join('\n');
  const url = URL.createObjectURL(new Blob(['﻿' + csv], { type: 'text/csv;charset=utf-8;' }));
  const a = document.createElement('a');
  a.href = url;
  a.download = filename;
  a.click();
  URL.revokeObjectURL(url);
}

function downloadTemplate() {
  downloadCsv('template-order.csv', [[
    'No.Order', 'Tanggal', 'Kategori', 'Pelanggan', 'Alamat', 'No.Resi', 'Keterangan',
    'SKU', 'Nama', 'Qty', 'Satuan', 'Batch', 'ExpDate',
  ]]);
}

function exportOrders() {
  const rows: string[][] = [[
    'No.Order', 'Tanggal', 'Kategori', 'Pelanggan', 'Alamat', 'No.Resi', 'Status', 'Item', 'Keterangan',
  ]];
  filteredOrders.value.forEach((o) => rows.push([
    o.noOrder ?? '', o.tanggal ?? '', o.kategori ?? '', o.pelanggan ?? '', o.alamat ?? '',
    o.noResi ?? '', o.status ?? '', String(itemCount(o)), o.keterangan ?? '',
  ]));
  downloadCsv(`order-${activeTab.value.toLowerCase()}.csv`, rows);
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
    const header = parsed[0] ?? [];
    if (!header[0]?.toLowerCase().includes('order')) throw new Error('Baris pertama harus header template');
    let created = 0, skipped = 0;
    for (let i = 1; i < parsed.length; i++) {
      const c = parsed[i] ?? [];
      const noOrder = c[0];
      if (!noOrder) continue;
      if (orders.value.some((o) => o.noOrder === noOrder)) { skipped++; continue; }
      await fclInsert('order', [{
        noOrder,
        tanggal: c[1] || dayjs().format('YYYY-MM-DD'),
        kategori: (c[2] || 'Distributor'),
        pelanggan: (c[3] || ''),
        alamat: (c[4] || ''),
        noResi: (c[5] || ''),
        keterangan: (c[6] || ''),
        status: 'Menunggu',
        totalItem: '1',
      }]);
      created++;
    }
    notification.success({ message: 'Impor Order', description: `${created} order ditambahkan, ${skipped} dilewati (duplikat)` });
    await loadAll(true);
  } catch (err: any) {
    notification.error({ message: 'Impor Order', description: err?.message || 'Gagal membaca file CSV' });
  } finally {
    if (importInput.value) importInput.value.value = '';
  }
}

// ── Cetak ────────────────────────────────────────────────────────────────────
function printOrder(o: OrderRow) {
  const ds = detailsOf(o);
  const rows = ds.map((d, i) => `<tr><td>${i + 1}</td><td>${esc(d.sku)}</td><td>${esc(d.nama)}</td><td class="r">${num(d.qty)}</td><td>${esc(d.satuan)}</td><td>${esc(d.batch)}</td><td>${esc((d.expDate ?? '').slice(0, 10))}</td></tr>`).join('');
  const w = window.open('', '_blank', 'width=900,height=700');
  if (!w) { message.warning('Popup diblokir browser'); return; }
  w.document.write(`<!doctype html><html><head><meta charset="utf-8"><title>Order ${esc(o.noOrder)}</title><style>
  body{font-family:'Segoe UI',Arial,sans-serif;color:#111;margin:22px;font-size:12px}
  h1{font-size:18px;margin:0 0 4px}.sub{color:#555;margin-bottom:16px}
  table{width:100%;border-collapse:collapse;margin-top:12px}th,td{border:1px solid #ccc;padding:5px 8px;text-align:left}
  th{background:#f3f4f6}.r{text-align:right}.sign{display:flex;justify-content:space-between;margin-top:40px;gap:40px}
  .sign div{flex:1;text-align:center;font-size:11px}.line{border-top:1px solid #111;margin:40px 8px 4px}
  </style></head><body>
  <h1>ORDER / SURAT PESANAN</h1>
  <div class="sub"><b>${esc(o.noOrder)}</b> &nbsp;•&nbsp; ${esc(o.tanggal)} &nbsp;•&nbsp; ${esc(o.kategori)}<br/>
  Pelanggan: <b>${esc(o.pelanggan || '-')}</b> &nbsp;•&nbsp; Alamat: ${esc(o.alamat || '-')} &nbsp;•&nbsp; No. Resi: ${esc(o.noResi || '-')}</div>
  <table><thead><tr><th>No</th><th>SKU</th><th>Nama Barang</th><th class="r">Qty</th><th>Satuan</th><th>Batch</th><th>Exp</th></tr></thead>
  <tbody>${rows || '<tr><td colspan="7" style="text-align:center">Tidak ada item</td></tr>'}</tbody></table>
  <div class="sign"><div><div class="line"></div>Dibuat oleh</div><div><div class="line"></div>Diperiksa oleh</div><div><div class="line"></div>Penerima</div></div>
  </body></html>`);
  w.document.close();
  w.focus();
  setTimeout(() => { try { w.print(); } catch { /* abaikan */ } }, 350);
}

onMounted(loadAll);
</script>

<template>
  <Page auto-content-height>
    <!-- ── Header ── -->
    <div class="ord-head">
      <div class="ord-head-title">Manajemen Order</div>
      <div class="ord-head-sub">Kelola pesanan pelanggan dan pengiriman barang</div>
    </div>

    <!-- ── Toolbar ── -->
    <div class="ord-toolbar">
      <div class="ord-search-wrap">
        <input v-model="searchText" type="text" class="ord-search" placeholder="Cari No. Order / Pelanggan / Resi..." />
        <input v-model="searchSku" type="text" class="ord-search" placeholder="Cari SKU dalam pesanan..." />
      </div>
      <div class="ord-actions">
        <input ref="importInput" type="file" accept=".csv,text/csv" class="ord-hidden-file" @change="handleImport" />
        <button class="ord-btn ord-btn-ghost" @click="downloadTemplate"><LucideFilePenLine :size="14" /> Template</button>
        <button class="ord-btn ord-btn-ghost" @click="importInput?.click()"><LucidePlus :size="14" /> Impor</button>
        <button class="ord-btn ord-btn-success" @click="exportOrders"><LucideDownload :size="14" /> Ekspor</button>
        <button class="ord-btn ord-btn-primary" @click="openOrderModal()"><LucidePlus :size="14" /> Buat Order</button>
      </div>
    </div>

    <!-- ── Tab kategori ── -->
    <div class="ord-tabs">
      <button
        v-for="tab in (['MP', 'Dist', 'Store', 'History'] as KategoriTab[])"
        :key="tab"
        class="ord-tab"
        :class="{ 'ord-tab-active': activeTab === tab, 'ord-tab-danger': tab === 'History' }"
        @click="activeTab = tab"
      >
        {{ tab === 'MP' ? 'Marketplace' : tab === 'Dist' ? 'Distributor' : tab === 'Store' ? 'Store' : 'Riwayat' }}
        <span class="ord-tab-count">{{ counts[tab] }}</span>
      </button>
    </div>

    <!-- ── Tabel ── -->
    <div class="ord-table-wrap">
      <table class="ord-table">
        <thead>
          <tr>
            <th>No. Order</th>
            <th>Tanggal</th>
            <th>{{ activeTab === 'Dist' ? 'Pelanggan' : 'Toko/Pelanggan' }}</th>
            <th>No. Resi</th>
            <th class="c">Item</th>
            <th>Status</th>
            <th>Aksi</th>
          </tr>
        </thead>
        <tbody>
          <template v-for="o in pagedOrders" :key="o.id">
            <tr>
              <td>
                <button class="ord-expand" @click="toggleExpand(o.id)">
                  <LucideChevronDown :size="14" :class="{ 'ord-rotate': expanded.has(o.id) }" />
                </button>
                <b>{{ o.noOrder }}</b>
              </td>
              <td>{{ (o.tanggal || '').slice(0, 10) }}</td>
              <td><b>{{ o.pelanggan || '-' }}</b></td>
              <td>{{ o.noResi || (o.alamat || '-') }}</td>
              <td class="c"><b>{{ itemCount(o) }}</b></td>
              <td>
                <span class="ord-status" :class="isSent(o) ? 'ord-status-sent' : 'ord-status-pending'">{{ o.status || 'Menunggu' }}</span>
              </td>
              <td>
                <div class="ord-row-actions">
                  <template v-if="!isSent(o)">
                    <button class="ord-btn ord-btn-primary ord-btn-xs" @click="markStatus(o, 'Divalidasi')"><LucideCheck :size="12" /> Validasi Fisik Pesanan</button>
                    <button class="ord-btn ord-btn-success ord-btn-xs" @click="markStatus(o, 'Terkirim', { sentAt: dayjs().format('YYYY-MM-DD HH:mm:ss') })"><LucideSend :size="12" /> Kirim</button>
                    <button class="ord-btn ord-btn-ghost ord-btn-xs" @click="openOrderModal(o)"><LucidePencil :size="12" /> Edit</button>
                    <button class="ord-btn ord-btn-danger ord-btn-xs" @click="deleteOrder(o)"><LucideTrash2 :size="12" /> Hapus</button>
                  </template>
                  <button class="ord-btn ord-btn-ghost ord-btn-xs" @click="openBukti(o)"><LucidePlus :size="12" /> {{ o.buktiPacking ? 'Ganti Bukti' : 'Upload' }}</button>
                  <button v-if="o.buktiPacking" class="ord-btn ord-btn-ghost ord-btn-xs" @click="viewBukti(o)"><LucideEye :size="12" /> Foto</button>
                  <button class="ord-btn ord-btn-ghost ord-btn-xs ord-btn-print" @click="printOrder(o)"><LucidePrinter :size="12" /> Print / Cetak Order</button>
                </div>
              </td>
            </tr>
            <tr v-if="expanded.has(o.id)" class="ord-detail-row">
              <td colspan="7">
                <div class="ord-detail-head">
                  <b>ITEM ORDER: {{ o.noOrder }}</b>
                  <span>Alamat: {{ o.alamat || '-' }} | Pelanggan: {{ o.pelanggan || '-' }}</span>
                </div>
                <table class="ord-detail-table">
                  <thead><tr><th>SKU</th><th>Nama</th><th class="r">Qty</th><th>Satuan</th><th>Batch</th><th>Exp</th></tr></thead>
                  <tbody>
                    <tr v-for="d in detailsOf(o)" :key="d.id">
                      <td>{{ d.sku || '-' }}</td><td>{{ d.nama || '-' }}</td>
                      <td class="r">{{ num(d.qty) }}</td><td>{{ d.satuan || '-' }}</td>
                      <td>{{ d.batch || '-' }}</td><td>{{ (d.expDate || '').slice(0, 10) }}</td>
                    </tr>
                    <tr v-if="!detailsOf(o).length"><td colspan="6" class="c ord-muted">Belum ada item.</td></tr>
                  </tbody>
                </table>
              </td>
            </tr>
          </template>
          <tr v-if="!filteredOrders.length">
            <td colspan="7" class="ord-empty">Belum ada orderan di kategori ini</td>
          </tr>
        </tbody>
      </table>
    </div>
    <FclPagination v-model:page="page" v-model:page-size="pageSize" :total="filteredOrders.length" />

    <!-- ── Modal Buat/Edit Order ── -->
    <a-modal
      v-model:open="orderModalOpen"
      :title="editingOrder ? 'Edit Order' : 'Buat Orderan Baru'"
      :confirm-loading="orderSaving"
      ok-text="Simpan Order"
      cancel-text="Batal"
      width="880px"
      @ok="saveOrder"
    >
      <div class="ord-form">
        <div class="ord-form-row">
          <div class="ord-field"><label>Kategori Pengiriman</label>
            <select v-model="form.kategori" class="ord-input">
              <option value="Distributor">Distributor</option>
              <option value="Marketplace">Marketplace</option>
              <option value="Store">Store</option>
            </select>
          </div>
          <div class="ord-field"><label>Tanggal Order</label><input v-model="form.tanggal" type="date" class="ord-input" /></div>
          <div class="ord-field"><label>No. Order</label><input v-model="form.noOrder" class="ord-input" placeholder="ORD-2026-0001" /></div>
        </div>
        <div class="ord-form-row">
          <div class="ord-field"><label>Nama Pelanggan / Toko</label><input v-model="form.pelanggan" class="ord-input" placeholder="Nama pelanggan / customer" /></div>
          <div class="ord-field"><label>No. Resi</label><input v-model="form.noResi" class="ord-input" placeholder="No. resi pengiriman" /></div>
        </div>
        <div class="ord-form-row">
          <div class="ord-field"><label>Alamat Pengiriman</label><input v-model="form.alamat" class="ord-input" placeholder="Alamat lengkap" /></div>
          <div class="ord-field"><label>Keterangan</label><input v-model="form.keterangan" class="ord-input" placeholder="Catatan order (opsional)" /></div>
        </div>

        <div class="ord-scan">
          <label>SCAN BARCODE / KODE SKU</label>
          <div class="ord-scan-row">
            <input v-model="scanInput" class="ord-input" placeholder="Ketik/scan SKU lalu Enter..." @keydown.enter.prevent="addScanItem" />
            <button class="ord-btn ord-btn-ghost" @click="addScanItem">+ Tambah</button>
          </div>
        </div>

        <div class="ord-items-head">
          <b>Daftar Barang Order <span class="ord-muted">(sinkron dengan Stock Barang)</span></b>
          <div class="ord-actions">
            <button class="ord-btn ord-btn-ghost ord-btn-xs" @click="loadAll(true)">⟳ Refresh Stok</button>
            <button class="ord-btn ord-btn-ghost ord-btn-xs" @click="items.push(newItem())"><LucidePlus :size="12" /> Tambah Baris</button>
          </div>
        </div>
        <div class="ord-items-grid">
          <div v-for="(item, idx) in items" :key="idx" class="ord-item-row">
            <a-select
              v-model:value="item.sku"
              class="ord-input"
              show-search
              option-filter-prop="label"
              :options="stockSkuOptions"
              placeholder="Cari SKU / barang"
              @change="onSkuChange(item, $event)"
            />
            <input v-model="item.nama" class="ord-input" placeholder="Nama barang (dari stock)" readonly />
            <a-select
              v-model:value="item.batch"
              class="ord-input"
              :options="batchOptions(item.sku)"
              placeholder="Batch"
              allow-clear
            />
            <input v-model.number="item.qty" type="number" min="0" class="ord-input" placeholder="Qty" />
            <input v-model="item.satuan" class="ord-input" placeholder="Satuan" />
            <input v-model="item.expDate" type="date" class="ord-input" />
            <span class="ord-stock" :class="{ low: num(item.qty) > availableStock(item) }">
              Stok {{ availableStock(item) }}
            </span>
            <button class="ord-btn ord-btn-danger ord-btn-xs" title="Hapus baris" @click="items.splice(idx, 1)"><LucideTrash2 :size="12" /></button>
          </div>
        </div>
      </div>
    </a-modal>

    <!-- ── Modal bukti packing ── -->
    <a-modal v-model:open="buktiModalOpen" title="Bukti Packing" ok-text="Simpan" cancel-text="Batal" width="460px" @ok="saveBukti">
      <p class="ord-muted">Masukkan link foto bukti packing (kosongkan untuk menghapus).</p>
      <input v-model="buktiInput" class="ord-input" placeholder="https://..." />
    </a-modal>
  </Page>
</template>

<style scoped>
/* Manajemen Order — token tema vben, aman di terang & gelap */
.ord-head { margin: 2px 0 14px; }
.ord-head-title { font-size: 20px; font-weight: 800; color: hsl(var(--foreground)); }
.ord-head-sub { font-size: 12px; color: hsl(var(--muted-foreground)); margin-top: 2px; }

.ord-toolbar { display: flex; justify-content: space-between; align-items: center; gap: 10px; flex-wrap: wrap; margin-bottom: 12px; }
.ord-search-wrap { display: flex; gap: 8px; flex-wrap: wrap; }
.ord-search {
  width: 250px; height: 34px; padding: 2px 10px; border-radius: 6px;
  border: 1px solid hsl(var(--border)); background: hsl(var(--background));
  color: hsl(var(--foreground)); font-size: 12px; outline: none;
}
.ord-search:focus { border-color: hsl(var(--primary)); }
.ord-search::placeholder { color: hsl(var(--muted-foreground)); }
.ord-actions { display: flex; gap: 6px; flex-wrap: wrap; }
.ord-hidden-file { display: none; }

.ord-btn {
  display: inline-flex; align-items: center; justify-content: center; gap: 5px;
  border-radius: 6px; border: 1px solid hsl(var(--border)); background: hsl(var(--muted));
  color: hsl(var(--foreground)); font-size: 12px; font-weight: 700; padding: 6px 12px;
  cursor: pointer; white-space: nowrap; transition: filter 0.15s;
}
.ord-btn:hover { filter: brightness(0.96); }
.ord-btn-xs { padding: 3px 8px; font-size: 10px; }
.ord-btn-ghost { background: hsl(var(--muted)); }
.ord-btn-primary { background: hsl(var(--primary)); border-color: hsl(var(--primary)); color: hsl(var(--primary-foreground)); }
.ord-btn-success { background: hsl(var(--success)); border-color: hsl(var(--success)); color: hsl(var(--success-foreground)); }
.ord-btn-danger { background: hsl(var(--destructive)); border-color: hsl(var(--destructive)); color: hsl(var(--destructive-foreground)); }
.ord-btn-print { color: hsl(var(--primary)); }

.ord-tabs { display: flex; gap: 8px; flex-wrap: wrap; margin-bottom: 12px; }
.ord-tab {
  display: inline-flex; align-items: center; gap: 8px;
  border: 1px solid hsl(var(--border)); background: hsl(var(--card));
  color: hsl(var(--muted-foreground)); font-size: 12px; font-weight: 800;
  padding: 7px 14px; border-radius: 8px; cursor: pointer;
}
.ord-tab:hover { color: hsl(var(--foreground)); }
.ord-tab-active { background: hsl(var(--primary)); border-color: hsl(var(--primary)); color: hsl(var(--primary-foreground)); }
.ord-tab-active .ord-tab-count { background: hsl(var(--primary-foreground) / 0.2); color: hsl(var(--primary-foreground)); }
.ord-tab-danger { color: hsl(var(--destructive)); }
.ord-tab-count {
  background: hsl(var(--muted)); border-radius: 999px; padding: 1px 8px;
  font-size: 11px; color: hsl(var(--muted-foreground));
}

.ord-table-wrap { background: hsl(var(--card)); border: 1px solid hsl(var(--border)); border-radius: 8px; overflow-x: auto; }
.ord-table { width: 100%; min-width: 1020px; border-collapse: collapse; font-size: 12px; }
.ord-table th {
  background: hsl(var(--muted)); color: hsl(var(--muted-foreground));
  font-size: 10px; font-weight: 800; letter-spacing: 0.04em;
  text-align: left; padding: 10px 12px; border-bottom: 1px solid hsl(var(--border));
  white-space: nowrap;
}
.ord-table th.c, .ord-table td.c { text-align: center; }
.ord-table td { padding: 9px 12px; border-bottom: 1px solid hsl(var(--border)); color: hsl(var(--foreground)); vertical-align: middle; }
.ord-table tbody tr:hover { background: hsl(var(--accent)); }
.ord-expand { border: none; background: transparent; color: hsl(var(--muted-foreground)); cursor: pointer; padding: 0 6px 0 0; vertical-align: middle; }
.ord-rotate { transform: rotate(180deg); }
.ord-status { display: inline-block; padding: 3px 10px; border-radius: 999px; font-size: 10px; font-weight: 800; white-space: nowrap; }
.ord-status-pending { background: hsl(var(--warning) / 0.15); color: hsl(var(--warning)); }
.ord-status-sent { background: hsl(var(--success) / 0.15); color: hsl(var(--success)); }
.ord-row-actions { display: flex; gap: 4px; flex-wrap: wrap; align-items: center; }
.ord-empty { text-align: center; color: hsl(var(--muted-foreground)); padding: 22px 8px; }
.ord-muted { color: hsl(var(--muted-foreground)); font-size: 11px; }
.ord-footer {
  margin-top: 10px; padding: 10px 12px; border: 1px solid hsl(var(--border));
  border-radius: 8px; background: hsl(var(--card)); color: hsl(var(--muted-foreground)); font-size: 11px;
}

.ord-detail-row td { background: hsl(var(--muted) / 0.4); }
.ord-detail-head { display: flex; justify-content: space-between; gap: 12px; flex-wrap: wrap; padding: 8px 0 10px; margin-bottom: 8px; border-bottom: 1px solid hsl(var(--border)); font-size: 11px; }
.ord-detail-head b { color: hsl(var(--primary)); text-transform: uppercase; }
.ord-detail-head span { color: hsl(var(--muted-foreground)); }
.ord-detail-table { width: 100%; border-collapse: collapse; font-size: 11px; }
.ord-detail-table th { background: hsl(var(--muted)); color: hsl(var(--muted-foreground)); padding: 6px 8px; text-align: left; }
.ord-detail-table td { padding: 6px 8px; border: 1px solid hsl(var(--border)); }
.ord-detail-table td.r { text-align: right; }

/* Form */
.ord-form { display: flex; flex-direction: column; gap: 12px; padding-top: 4px; }
.ord-form-row { display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); gap: 10px; }
.ord-field label, .ord-scan label { display: block; font-size: 11px; font-weight: 700; color: hsl(var(--muted-foreground)); margin-bottom: 4px; }
.ord-input {
  height: 32px; width: 100%; padding: 4px 9px; border-radius: 6px;
  border: 1px solid hsl(var(--border)); background: hsl(var(--background));
  color: hsl(var(--foreground)); font-size: 12px; outline: none;
}
.ord-input:focus { border-color: hsl(var(--primary)); }
.ord-scan { background: hsl(var(--primary) / 0.06); border: 1px solid hsl(var(--primary) / 0.3); border-radius: 8px; padding: 10px; }
.ord-scan label { color: hsl(var(--primary)); font-weight: 800; }
.ord-scan-row { display: flex; gap: 6px; }
.ord-items-head { display: flex; justify-content: space-between; align-items: center; }
.ord-items-grid { display: flex; flex-direction: column; gap: 6px; max-height: 260px; overflow-y: auto; }
.ord-item-row { display: grid; grid-template-columns: 1.5fr 1.4fr 0.9fr 0.5fr 0.7fr 1fr 0.8fr 32px; gap: 5px; align-items: center; }
.ord-stock { font-size: 10px; font-weight: 700; color: hsl(var(--success)); text-align: center; white-space: nowrap; }
.ord-stock.low { color: hsl(var(--destructive)); }

@media (max-width: 900px) {
  .ord-form-row { grid-template-columns: 1fr 1fr; }
  .ord-item-row { grid-template-columns: 1fr 1fr 0.7fr 32px; }
.ord-item-row .ord-stock { grid-column: 1 / -1; text-align: left; }
}
</style>
