<script lang="ts" setup>
/**
 * Stock Inventory (FCL) — layout sesuai referensi:
 * 4 kartu statistik, panel "Daftar Stock Barang" dengan pencarian & aksi
 * (Tambahkan ke Stok, Template/Import Excel, Export CSV, Pindah Stok, Tambah Barang),
 * tabel per SKU dengan baris detail batch/lokasi yang bisa di-expand.
 * Warna memakai token tema vben (hsl(var(--...))).
 */
import { computed, onMounted, ref, watch } from 'vue';

import { Page } from '@vben/common-ui';
import {
  LucideArrowLeftRight,
  LucideChevronDown,
  LucideDownload,
  LucideFilePenLine,
  LucidePencil,
  LucidePlus,
  LucideTrash2,
} from '@vben/icons';

import { Modal, message, notification } from 'ant-design-vue';
import dayjs from 'dayjs';

import { fclDelete, fclInsert, fclPatch, fclQuery } from '#/api/warehouse';
import FclPagination from '#/components/warehouse/warehouse-pagination.vue';
import {
  groupStockBySku,
  stockExpiringCount,
  stockLowCount,
  type StockRow,
} from '#/utils/warehouse-stock';

const loading = ref(true);
const rows = ref<StockRow[]>([]);
const search = ref('');
const expanded = ref<Set<string>>(new Set());

const grouped = computed(() => {
  const q = search.value.trim().toLowerCase();
  const positives = rows.value.filter((d) => Number(d.stok) > 0);
  const filtered = positives.filter((d) => {
    if (!q) return true;
    return [d.sku, d.nama, d.kategori, d.lokasi, d.batch, d.barcode]
      .map((v) => String(v ?? '').toLowerCase())
      .join(' ')
      .includes(q);
  });
  return groupStockBySku(filtered);
});

const stats = computed(() => {
  const g = grouped.value;
  return {
    jenis: g.length,
    total: g.reduce((s, x) => s + x.total, 0),
    rendah: stockLowCount(g),
    exp: stockExpiringCount(g, 30),
  };
});

const page = ref(1);
const pageSize = ref(20);
const pagedGroups = computed(() =>
  grouped.value.slice((page.value - 1) * pageSize.value, page.value * pageSize.value),
);

watch(search, () => { page.value = 1; });

function toggleExpand(key: string) {
  const next = new Set(expanded.value);
  if (next.has(key)) next.delete(key);
  else next.add(key);
  expanded.value = next;
}

async function loadAll(silent = false) {
  if (!silent) loading.value = true;
  try {
    rows.value = await fclQuery<StockRow>('stock', { select: '*', order: 'sku.asc', limit: 10000 });
  } catch (err: any) {
    console.error('[warehouse/stock] load gagal', err);
    notification.error({ message: 'Stock Inventory', description: err?.message || 'Gagal memuat data stok' });
  } finally {
    if (!silent) loading.value = false;
  }
}

function num(v: unknown): number {
  const n = Number(v);
  return Number.isFinite(n) ? n : 0;
}

// ── Modal Tambah Barang / Edit baris ─────────────────────────────────────────
const stockModalOpen = ref(false);
const stockSaving = ref(false);
const editingRow = ref<StockRow | null>(null);
const stockForm = ref({
  sku: '', nama: '', barcode: '', kategori: '', satuan: 'PCS',
  batch: '', lokasi: '', expDate: '', stok: undefined as number | undefined,
  stokMin: 0 as number | undefined,
});

function openStockModal(row?: StockRow | null) {
  editingRow.value = row ?? null;
  stockForm.value = row
    ? {
        sku: row.sku ?? '', nama: row.nama ?? '', barcode: row.barcode ?? '',
        kategori: row.kategori ?? '', satuan: row.satuan ?? 'PCS',
        batch: row.batch ?? '', lokasi: row.lokasi ?? '', expDate: (row.expDate ?? '').slice(0, 10),
        stok: num(row.stok), stokMin: num(row.stokMin),
      }
    : {
        sku: '', nama: '', barcode: '', kategori: '', satuan: 'PCS',
        batch: '', lokasi: '', expDate: '', stok: undefined, stokMin: 0,
      };
  stockModalOpen.value = true;
}

function stockPayload(): Record<string, any> {
  return {
    sku: stockForm.value.sku.trim(),
    nama: stockForm.value.nama.trim(),
    barcode: stockForm.value.barcode.trim(),
    kategori: stockForm.value.kategori.trim(),
    satuan: stockForm.value.satuan.trim(),
    batch: stockForm.value.batch.trim(),
    lokasi: stockForm.value.lokasi.trim(),
    expDate: stockForm.value.expDate || null,
    stok: String(num(stockForm.value.stok)),
    stokMin: String(num(stockForm.value.stokMin)),
  };
}

async function saveStock() {
  if (!stockForm.value.sku.trim()) { message.warning('SKU wajib diisi'); return; }
  if (!stockForm.value.nama.trim()) { message.warning('Nama barang wajib diisi'); return; }
  stockSaving.value = true;
  try {
    if (editingRow.value) {
      await fclPatch('stock', editingRow.value.id!, stockPayload());
      notification.success({ message: 'Stock', description: 'Baris stok diperbarui' });
    } else {
      await fclInsert('stock', [{ ...stockPayload(), createdAt: dayjs().format('YYYY-MM-DD HH:mm:ss') }]);
      notification.success({ message: 'Stock', description: 'Barang ditambahkan' });
    }
    stockModalOpen.value = false;
    await loadAll(true);
  } catch (err: any) {
    notification.error({ message: 'Stock', description: err?.message || 'Gagal menyimpan' });
  } finally {
    stockSaving.value = false;
  }
}

function deleteRow(r: StockRow) {
  Modal.confirm({
    title: 'Hapus baris stok?',
    content: `${r.sku} • batch ${r.batch || '-'} • ${r.lokasi || '-'} (${num(r.stok)} ${r.satuan || ''})`,
    okText: 'Hapus',
    okType: 'danger',
    onOk: async () => {
      await fclDelete('stock', r.id!);
      notification.success({ message: 'Baris stok dihapus' });
      await loadAll(true);
    },
  });
}

// ── Modal Tambahkan ke Stok (tambah batch/lokasi baru untuk SKU existing) ────
const addStockOpen = ref(false);
const addSaving = ref(false);
const addForm = ref({ sku: '', batch: '', lokasi: '', expDate: '', stok: undefined as number | undefined });

function openAddStock() {
  addForm.value = { sku: '', batch: '', lokasi: '', expDate: '', stok: undefined };
  addStockOpen.value = true;
}

async function saveAddStock() {
  if (!addForm.value.sku) { message.warning('Pilih SKU dulu'); return; }
  if (!num(addForm.value.stok)) { message.warning('Jumlah stok harus lebih dari 0'); return; }
  addSaving.value = true;
  try {
    const source = grouped.value.find((g) => g.sku === addForm.value.sku);
    await fclInsert('stock', [{
      sku: addForm.value.sku,
      nama: source?.nama ?? '',
      barcode: source?.barcode ?? '',
      kategori: source?.kategori ?? '',
      satuan: source?.satuan ?? 'PCS',
      batch: addForm.value.batch.trim(),
      lokasi: addForm.value.lokasi.trim(),
      expDate: addForm.value.expDate || null,
      stok: String(num(addForm.value.stok)),
      stokMin: String(source?.minRef ?? 0),
      createdAt: dayjs().format('YYYY-MM-DD HH:mm:ss'),
    }]);
    notification.success({ message: 'Stock', description: 'Stok berhasil ditambahkan' });
    addStockOpen.value = false;
    await loadAll(true);
  } catch (err: any) {
    notification.error({ message: 'Stock', description: err?.message || 'Gagal menambah stok' });
  } finally {
    addSaving.value = false;
  }
}

// ── Modal Pindah Stok antar lokasi ───────────────────────────────────────────
const moveOpen = ref(false);
const moveSaving = ref(false);
const moveForm = ref({ rowId: '', lokasiTujuan: '', qty: undefined as number | undefined });

function openMove() {
  moveForm.value = { rowId: '', lokasiTujuan: '', qty: undefined };
  moveOpen.value = true;
}

async function saveMove() {
  const src = rows.value.find((r) => String(r.id) === String(moveForm.value.rowId));
  if (!src) { message.warning('Pilih baris stok sumber'); return; }
  const qty = num(moveForm.value.qty);
  if (qty <= 0 || qty > num(src.stok)) { message.warning('Jumlah tidak valid'); return; }
  const dest = moveForm.value.lokasiTujuan.trim();
  if (!dest || dest === String(src.lokasi ?? '')) { message.warning('Lokasi tujuan berbeda dengan lokasi sumber'); return; }
  moveSaving.value = true;
  try {
    // kurangi sumber
    await fclPatch('stock', src.id!, { stok: String(num(src.stok) - qty) });
    // cari baris tujuan yang sama (sku + batch + lokasi tujuan)
    const target = rows.value.find((r) =>
      String(r.id) !== String(src.id) && r.sku === src.sku && r.batch === src.batch && r.lokasi === dest,
    );
    if (target) {
      await fclPatch('stock', target.id!, { stok: String(num(target.stok) + qty) });
    } else {
      await fclInsert('stock', [{
        ...src,
        id: undefined,
        lokasi: dest,
        stok: String(qty),
        createdAt: dayjs().format('YYYY-MM-DD HH:mm:ss'),
      }]);
    }
    notification.success({ message: 'Pindah Stok', description: `${qty} ${src.satuan || ''} dipindah ke ${dest}` });
    moveOpen.value = false;
    await loadAll(true);
  } catch (err: any) {
    notification.error({ message: 'Pindah Stok', description: err?.message || 'Gagal memindahkan stok' });
  } finally {
    moveSaving.value = false;
  }
}

// ── CSV ──────────────────────────────────────────────────────────────────────
function downloadCsv(filename: string, lines: string[][]) {
  const csv = lines.map((cols) => cols.map((c) => `"${String(c ?? '').replace(/"/g, '""')}"`).join(',')).join('\n');
  const url = URL.createObjectURL(new Blob(['﻿' + csv], { type: 'text/csv;charset=utf-8;' }));
  const a = document.createElement('a');
  a.href = url; a.download = filename; a.click();
  URL.revokeObjectURL(url);
}

function downloadTemplate() {
  downloadCsv('template-stock.csv', [['SKU', 'Nama', 'Barcode', 'Batch', 'Lokasi', 'ExpDate', 'Stok', 'Min', 'Satuan', 'Kategori']]);
}

function exportCsv() {
  const lines = [['SKU', 'Nama', 'Barcode', 'Batch', 'Lokasi', 'ExpDate', 'Stok', 'Min', 'Satuan', 'Kategori']];
  grouped.value.forEach((g) => g.rows.forEach((r) => lines.push([
    r.sku ?? '', r.nama ?? '', r.barcode ?? '', r.batch ?? '', r.lokasi ?? '',
    (r.expDate ?? '').slice(0, 10), String(r.stok ?? ''), String(r.stokMin ?? ''), r.satuan ?? '', r.kategori ?? '',
  ])));
  downloadCsv('stock-barang.csv', lines);
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
    if (!parsed.length || !parsed[0][0]?.toLowerCase().includes('sku')) throw new Error('Baris pertama harus header template');
    let inserted = 0;
    const payload = parsed.slice(1).map((c) => ({
      sku: c[0] || '-', nama: c[1] || '-', barcode: c[2] || '', batch: c[3] || '', lokasi: c[4] || '',
      expDate: c[5] || null, stok: String(num(c[6])), stokMin: String(num(c[7])), satuan: c[8] || 'PCS',
      kategori: c[9] || '-', createdAt: dayjs().format('YYYY-MM-DD HH:mm:ss'),
    })).filter((r) => r.sku && r.nama);
    if (payload.length) { await fclInsert('stock', payload); inserted = payload.length; }
    notification.success({ message: 'Import Excel', description: `${inserted} baris stok ditambahkan` });
    await loadAll(true);
  } catch (err: any) {
    notification.error({ message: 'Import Excel', description: err?.message || 'Gagal membaca file CSV' });
  } finally {
    if (importInput.value) importInput.value.value = '';
  }
}

onMounted(loadAll);
</script>

<template>
  <Page auto-content-height>
    <!-- ── Statistik ── -->
    <div class="stk-stats">
      <div class="stk-stat stk-border-orange">
        <div class="stk-stat-label">TOTAL JENIS BARANG</div>
        <div class="stk-stat-value">{{ stats.jenis }}</div>
      </div>
      <div class="stk-stat stk-border-green">
        <div class="stk-stat-label">TOTAL STOK (UNIT)</div>
        <div class="stk-stat-value stk-green">{{ stats.total.toLocaleString('id-ID') }}</div>
      </div>
      <div class="stk-stat stk-border-amber">
        <div class="stk-stat-label">STOK RENDAH</div>
        <div class="stk-stat-value stk-amber">{{ stats.rendah }}</div>
      </div>
      <div class="stk-stat stk-border-red">
        <div class="stk-stat-label">HAMPIR EXP (30 HR)</div>
        <div class="stk-stat-value stk-red">{{ stats.exp }}</div>
      </div>
    </div>

    <!-- ── Panel daftar ── -->
    <div class="stk-panel">
      <div class="stk-panel-head">
        <div class="stk-panel-title">Daftar Stock Barang</div>
        <div class="stk-panel-actions">
          <input v-model="search" type="text" class="stk-search" placeholder="Cari SKU / Nama / Lokasi..." />
          <button class="stk-btn stk-btn-ghost" @click="openAddStock"><LucidePlus :size="14" /> Tambahkan ke Stok</button>
          <button class="stk-btn stk-btn-ghost" @click="downloadTemplate"><LucideFilePenLine :size="14" /> Template Excel</button>
          <button class="stk-btn stk-btn-ghost" @click="importInput?.click()"><LucidePlus :size="14" /> Import Excel</button>
          <input ref="importInput" type="file" accept=".csv,text/csv" class="stk-hidden-file" @change="handleImport" />
          <button class="stk-btn stk-btn-ghost" @click="exportCsv"><LucideDownload :size="14" /> Export CSV</button>
          <button class="stk-btn stk-btn-move" @click="openMove"><LucideArrowLeftRight :size="14" /> Pindah Stok</button>
          <button class="stk-btn stk-btn-primary" @click="openStockModal()"><LucidePlus :size="14" /> Tambah Barang</button>
        </div>
      </div>

      <div class="stk-table-wrap">
        <table class="stk-table">
          <thead>
            <tr>
              <th>SKU</th>
              <th>Nama Barang</th>
              <th>Barcode</th>
              <th class="r">Total Stok</th>
              <th class="r">Min</th>
              <th>Satuan</th>
              <th>Kategori</th>
              <th>Status</th>
              <th>Aksi</th>
            </tr>
          </thead>
          <tbody>
            <template v-for="g in pagedGroups" :key="g.key">
              <tr>
                <td>
                  <button class="stk-expand" @click="toggleExpand(g.key)">
                    <LucideChevronDown :size="14" :class="{ 'stk-rotate': expanded.has(g.key) }" />
                  </button>
                  <div class="stk-sku-cell">
                    <b>{{ g.sku }}</b>
                    <span class="stk-hint">{{ g.hint }}</span>
                  </div>
                </td>
                <td class="stk-nama">{{ g.nama }}</td>
                <td>{{ g.barcode || '-' }}</td>
                <td class="r"><b>{{ g.total.toLocaleString('id-ID') }}</b></td>
                <td class="r">{{ g.minRef }}</td>
                <td>{{ g.satuan || '-' }}</td>
                <td><span class="stk-kat">{{ (g.kategori || '-').toUpperCase() }}</span></td>
                <td><span class="stk-status" :class="'stk-status-' + g.statusClass">{{ g.status.toUpperCase() }}</span></td>
                <td>
                  <button class="stk-btn stk-btn-ghost stk-btn-xs" @click="toggleExpand(g.key)">Detail</button>
                </td>
              </tr>
              <tr v-if="expanded.has(g.key)" class="stk-detail-row">
                <td colspan="9">
                  <div class="stk-detail-head">
                    <b>DETAIL BATCH / LOKASI — {{ g.sku }}</b>
                    <span>{{ g.rows.length }} batch</span>
                  </div>
                  <table class="stk-detail-table">
                    <thead><tr><th>Lokasi</th><th>Batch</th><th>Exp Date</th><th class="r">Qty</th><th>Status</th><th>Aksi</th></tr></thead>
                    <tbody>
                      <tr v-for="r in g.rows" :key="r.id">
                        <td>{{ r.lokasi || '-' }}</td>
                        <td>{{ r.batch || '-' }}</td>
                        <td>{{ (r.expDate || '').slice(0, 10) || '-' }}</td>
                        <td class="r"><b>{{ num(r.stok).toLocaleString('id-ID') }}</b></td>
                        <td>
                          <span class="stk-status" :class="num(r.stok) <= 0 ? 'stk-status-habis' : num(r.stok) <= num(r.stokMin) && num(r.stokMin) > 0 ? 'stk-status-rendah' : 'stk-status-aman'">
                            {{ num(r.stok) <= 0 ? 'HABIS' : num(r.stok) <= num(r.stokMin) && num(r.stokMin) > 0 ? 'RENDAH' : 'AMAN' }}
                          </span>
                        </td>
                        <td>
                          <div class="stk-row-actions">
                            <button class="stk-btn stk-btn-ghost stk-btn-xs" @click="openStockModal(r)"><LucidePencil :size="11" /> Edit</button>
                            <button class="stk-btn stk-btn-danger stk-btn-xs" @click="deleteRow(r)"><LucideTrash2 :size="11" /> Hapus</button>
                          </div>
                        </td>
                      </tr>
                    </tbody>
                  </table>
                </td>
              </tr>
            </template>
            <tr v-if="!grouped.length">
              <td colspan="9" class="stk-empty">Belum ada data stok barang.</td>
            </tr>
          </tbody>
        </table>
      </div>
      <FclPagination v-model:page="page" v-model:page-size="pageSize" :total="grouped.length" />
    </div>

    <!-- ── Modal Tambah / Edit baris stok ── -->
    <a-modal
      v-model:open="stockModalOpen"
      :title="editingRow ? 'Edit Barang' : 'Tambah Barang'"
      :confirm-loading="stockSaving"
      ok-text="Simpan"
      cancel-text="Batal"
      width="640px"
      @ok="saveStock"
    >
      <div class="stk-form">
        <div class="stk-form-row">
          <div class="stk-field"><label>SKU *</label><input v-model="stockForm.sku" class="stk-input" placeholder="DR-PR06" /></div>
          <div class="stk-field"><label>Nama Barang *</label><input v-model="stockForm.nama" class="stk-input" placeholder="Nama barang" /></div>
        </div>
        <div class="stk-form-row">
          <div class="stk-field"><label>Barcode</label><input v-model="stockForm.barcode" class="stk-input" /></div>
          <div class="stk-field"><label>Kategori</label><input v-model="stockForm.kategori" class="stk-input" placeholder="KOSMETIK" /></div>
        </div>
        <div class="stk-form-row">
          <div class="stk-field"><label>Batch</label><input v-model="stockForm.batch" class="stk-input" /></div>
          <div class="stk-field"><label>Lokasi</label><input v-model="stockForm.lokasi" class="stk-input" /></div>
        </div>
        <div class="stk-form-row">
          <div class="stk-field"><label>Exp Date</label><input v-model="stockForm.expDate" type="date" class="stk-input" /></div>
          <div class="stk-field"><label>Satuan</label><input v-model="stockForm.satuan" class="stk-input" /></div>
        </div>
        <div class="stk-form-row">
          <div class="stk-field"><label>Stok *</label><input v-model.number="stockForm.stok" type="number" min="0" class="stk-input" /></div>
          <div class="stk-field"><label>Minimal Stok</label><input v-model.number="stockForm.stokMin" type="number" min="0" class="stk-input" /></div>
        </div>
      </div>
    </a-modal>

    <!-- ── Modal Tambahkan ke Stok ── -->
    <a-modal v-model:open="addStockOpen" title="Tambahkan ke Stok" :confirm-loading="addSaving" ok-text="Tambah" cancel-text="Batal" width="520px" @ok="saveAddStock">
      <div class="stk-form">
        <div class="stk-field"><label>Pilih SKU *</label>
          <select v-model="addForm.sku" class="stk-input">
            <option value="">-- pilih SKU --</option>
            <option v-for="g in grouped" :key="g.key" :value="g.sku">{{ g.sku }} — {{ g.nama }}</option>
          </select>
        </div>
        <div class="stk-form-row">
          <div class="stk-field"><label>Batch</label><input v-model="addForm.batch" class="stk-input" /></div>
          <div class="stk-field"><label>Lokasi</label><input v-model="addForm.lokasi" class="stk-input" /></div>
        </div>
        <div class="stk-form-row">
          <div class="stk-field"><label>Exp Date</label><input v-model="addForm.expDate" type="date" class="stk-input" /></div>
          <div class="stk-field"><label>Jumlah *</label><input v-model.number="addForm.stok" type="number" min="0" class="stk-input" /></div>
        </div>
      </div>
    </a-modal>

    <!-- ── Modal Pindah Stok ── -->
    <a-modal v-model:open="moveOpen" title="Pindah Stok" :confirm-loading="moveSaving" ok-text="Pindahkan" cancel-text="Batal" width="560px" @ok="saveMove">
      <div class="stk-form">
        <div class="stk-field"><label>Baris stok sumber *</label>
          <select v-model="moveForm.rowId" class="stk-input">
            <option value="">-- pilih batch / lokasi --</option>
            <option v-for="r in rows.filter((x) => num(x.stok) > 0)" :key="r.id" :value="String(r.id)">
              {{ r.sku }} • {{ r.batch || '-' }} • {{ r.lokasi || '-' }} ({{ num(r.stok) }})
            </option>
          </select>
        </div>
        <div class="stk-form-row">
          <div class="stk-field"><label>Lokasi tujuan *</label><input v-model="moveForm.lokasiTujuan" class="stk-input" placeholder="Nama lokasi/gudang tujuan" /></div>
          <div class="stk-field"><label>Jumlah *</label><input v-model.number="moveForm.qty" type="number" min="0" class="stk-input" /></div>
        </div>
      </div>
    </a-modal>
  </Page>
</template>

<style scoped>
/* Stock Inventory — token tema */
.stk-stats { display: grid; grid-template-columns: repeat(4, minmax(0, 1fr)); gap: 10px; margin-bottom: 12px; }
.stk-stat {
  background: hsl(var(--card)); border: 1px solid hsl(var(--border));
  border-top: 3px solid hsl(var(--muted-foreground)); border-radius: 8px; padding: 14px 16px;
}
.stk-border-orange { border-top-color: hsl(var(--warning)); }
.stk-border-green { border-top-color: hsl(var(--success)); }
.stk-border-amber { border-top-color: hsl(var(--warning)); }
.stk-border-red { border-top-color: hsl(var(--destructive)); }
.stk-stat-label { font-size: 11px; font-weight: 800; letter-spacing: 0.04em; color: hsl(var(--muted-foreground)); }
.stk-stat-value { font-size: 24px; font-weight: 800; color: hsl(var(--foreground)); margin-top: 6px; }
.stk-green { color: hsl(var(--success)) !important; }
.stk-amber { color: hsl(var(--warning)) !important; }
.stk-red { color: hsl(var(--destructive)) !important; }

.stk-panel { background: hsl(var(--card)); border: 1px solid hsl(var(--border)); border-radius: 8px; }
.stk-panel-head { display: flex; justify-content: space-between; align-items: center; gap: 10px; flex-wrap: wrap; padding: 12px; }
.stk-panel-title { font-size: 14px; font-weight: 800; color: hsl(var(--foreground)); }
.stk-panel-actions { display: flex; gap: 6px; flex-wrap: wrap; align-items: center; }
.stk-hidden-file { display: none; }
.stk-search {
  height: 32px; width: 210px; padding: 2px 10px; border-radius: 6px;
  border: 1px solid hsl(var(--border)); background: hsl(var(--background));
  color: hsl(var(--foreground)); font-size: 12px; outline: none;
}
.stk-search:focus { border-color: hsl(var(--primary)); }
.stk-search::placeholder { color: hsl(var(--muted-foreground)); }

.stk-btn {
  display: inline-flex; align-items: center; justify-content: center; gap: 5px;
  border: 1px solid hsl(var(--border)); background: hsl(var(--muted)); color: hsl(var(--foreground));
  font-size: 12px; font-weight: 700; padding: 6px 12px; border-radius: 6px;
  cursor: pointer; white-space: nowrap; transition: filter 0.15s;
}
.stk-btn:hover { filter: brightness(0.96); }
.stk-btn-xs { padding: 3px 8px; font-size: 10px; }
.stk-btn-primary { background: hsl(var(--primary)); border-color: hsl(var(--primary)); color: hsl(var(--primary-foreground)); }
.stk-btn-move { background: hsl(var(--info)); border-color: hsl(var(--primary) / 0.4); color: hsl(var(--foreground)); }
.stk-btn-danger { background: hsl(var(--destructive)); border-color: hsl(var(--destructive)); color: hsl(var(--destructive-foreground)); }

.stk-table-wrap { overflow-x: auto; border-top: 1px solid hsl(var(--border)); }
.stk-table { width: 100%; min-width: 1100px; border-collapse: collapse; font-size: 12px; }
.stk-table th {
  background: hsl(var(--muted)); color: hsl(var(--muted-foreground)); font-size: 10px;
  font-weight: 800; letter-spacing: 0.04em; text-align: left;
  padding: 10px 12px; border-bottom: 1px solid hsl(var(--border)); white-space: nowrap;
}
.stk-table th.r, .stk-table td.r { text-align: right; }
.stk-table td { padding: 9px 12px; border-bottom: 1px solid hsl(var(--border)); color: hsl(var(--foreground)); vertical-align: middle; }
.stk-table tbody tr:hover { background: hsl(var(--accent)); }
.stk-expand { border: none; background: transparent; color: hsl(var(--muted-foreground)); cursor: pointer; padding: 0 8px 0 0; vertical-align: middle; }
.stk-rotate { transform: rotate(180deg); }
.stk-sku-cell { display: inline-flex; flex-direction: column; vertical-align: middle; }
.stk-hint { font-size: 10px; color: hsl(var(--muted-foreground)); margin-top: 2px; }
.stk-nama { max-width: 320px; white-space: normal !important; }
.stk-kat { font-size: 10px; font-weight: 800; background: hsl(var(--muted)); border: 1px solid hsl(var(--border)); border-radius: 5px; padding: 3px 8px; }
.stk-status { display: inline-block; font-size: 10px; font-weight: 800; padding: 3px 10px; border-radius: 999px; white-space: nowrap; }
.stk-status-aman { background: hsl(var(--success) / 0.15); color: hsl(var(--success)); }
.stk-status-rendah { background: hsl(var(--warning) / 0.16); color: hsl(var(--warning)); }
.stk-status-habis { background: hsl(var(--destructive) / 0.15); color: hsl(var(--destructive)); }
.stk-empty { text-align: center; color: hsl(var(--muted-foreground)); padding: 24px 8px; }
.stk-footer { margin: 10px 12px 12px; padding: 8px 12px; border: 1px solid hsl(var(--border)); border-radius: 6px; color: hsl(var(--muted-foreground)); font-size: 11px; }
.stk-detail-row td { background: hsl(var(--muted) / 0.45); }
.stk-detail-head { display: flex; justify-content: space-between; gap: 10px; padding: 8px 0 10px; border-bottom: 1px solid hsl(var(--border)); font-size: 11px; }
.stk-detail-head b { color: hsl(var(--primary)); }
.stk-detail-head span { color: hsl(var(--muted-foreground)); }
.stk-detail-table { width: 100%; border-collapse: collapse; font-size: 11px; }
.stk-detail-table th { padding: 6px 8px; text-align: left; background: hsl(var(--muted)); }
.stk-detail-table td { padding: 6px 8px; border: 1px solid hsl(var(--border)); }
.stk-row-actions { display: flex; gap: 4px; }

.stk-form { display: flex; flex-direction: column; gap: 12px; padding-top: 4px; }
.stk-form-row { display: grid; grid-template-columns: 1fr 1fr; gap: 10px; }
.stk-field label { display: block; font-size: 11px; font-weight: 700; color: hsl(var(--muted-foreground)); margin-bottom: 4px; }
.stk-input {
  height: 32px; width: 100%; padding: 4px 9px; border-radius: 6px;
  border: 1px solid hsl(var(--border)); background: hsl(var(--background));
  color: hsl(var(--foreground)); font-size: 12px; outline: none;
}
.stk-input:focus { border-color: hsl(var(--primary)); }

@media (max-width: 1100px) {
  .stk-stats { grid-template-columns: repeat(2, minmax(0, 1fr)); }
}
</style>
