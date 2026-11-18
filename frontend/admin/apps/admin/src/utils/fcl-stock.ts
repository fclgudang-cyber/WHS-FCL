// Port dari groupStockBySku() di app.js — kelompokkan baris stock per SKU.
export interface StockRow {
  id?: number | string;
  sku?: string;
  nama?: string;
  barcode?: string;
  satuan?: string;
  kategori?: string;
  stok?: number | string;
  stokMin?: number | string;
  lokasi?: string;
  batch?: string;
  expDate?: string | null;
  [key: string]: any;
}

export interface StockGroup {
  key: string;
  sku: string;
  nama: string;
  barcode: string;
  satuan: string;
  kategori: string;
  rows: StockRow[];
  total: number;
  minRef: number;
  hint: string;
  status: 'Aman' | 'Rendah' | 'Habis';
  statusClass: 'aman' | 'rendah' | 'habis';
}

export function groupStockBySku(data: StockRow[]): StockGroup[] {
  const map = new Map<string, StockGroup>();
  for (const d of data || []) {
    const skuRaw = String(d.sku || '').trim();
    const key = skuRaw ? skuRaw.toLowerCase() : '__tanpa_sku__' + d.id;
    if (!map.has(key)) {
      map.set(key, {
        key,
        sku: skuRaw || '-',
        nama: d.nama || '-',
        barcode: d.barcode || '',
        satuan: d.satuan || '',
        kategori: d.kategori || '-',
        rows: [],
        total: 0,
        minRef: 0,
        hint: '',
        status: 'Aman',
        statusClass: 'aman',
      });
    }
    const g = map.get(key)!;
    if ((!g.nama || g.nama === '-') && d.nama) g.nama = String(d.nama);
    if (!g.barcode && d.barcode) g.barcode = String(d.barcode);
    if (!g.satuan && d.satuan) g.satuan = String(d.satuan);
    if ((!g.kategori || g.kategori === '-') && d.kategori) g.kategori = String(d.kategori);
    g.rows.push(d);
    g.total += Number(d.stok) || 0;
    g.minRef = Math.max(g.minRef, Number(d.stokMin) || 0);
  }
  const groups = [...map.values()];
  for (const g of groups) {
    g.rows.sort(
      (a, b) =>
        String(a.lokasi || '').localeCompare(String(b.lokasi || '')) ||
        String(a.batch || '').localeCompare(String(b.batch || '')),
    );
    const locCount = new Set(g.rows.map((r) => String(r.lokasi || '-').trim().toLowerCase())).size;
    g.hint = `${g.rows.length} batch - ${locCount} lokasi`;
    if (g.total <= 0) {
      g.status = 'Habis';
      g.statusClass = 'habis';
    } else if (g.total <= g.minRef) {
      g.status = 'Rendah';
      g.statusClass = 'rendah';
    }
  }
  return groups.sort((a, b) => a.sku.localeCompare(b.sku));
}

export function stockExpiringCount(groups: StockGroup[], inDays = 30): number {
  const now = Date.now();
  const span = inDays * 86_400_000;
  return groups.filter((g) =>
    g.rows.some((d) => {
      if (!d.expDate) return false;
      const t = new Date(d.expDate).getTime();
      return Number.isFinite(t) && t >= now && t - now <= span;
    }),
  ).length;
}

export function stockLowCount(groups: StockGroup[]): number {
  return groups.filter((g) => g.total <= g.minRef && g.minRef > 0).length;
}
