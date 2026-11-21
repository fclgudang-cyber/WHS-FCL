// Port dari helper roster di aplikasi FCL (app.js) untuk dipakai di halaman native vben.
// Klasifikasi kode shift -> kategori seragam yang dipakai dashboard & halaman SDM.

export type RosterCategory =
  | 'PAGI'
  | 'MALAM'
  | 'OFF'
  | 'CUTI'
  | 'SAKIT'
  | 'ALFA'
  | 'IJIN'
  | 'EMPTY'
  | 'OTHER';

export function classifyRosterCode(val: unknown): RosterCategory {
  const c = String(val == null ? '' : val).trim().toUpperCase();
  if (!c || c === '-') return 'EMPTY';
  if (c === 'PAGI' || c === 'SETENGAH HARI' || c === 'SD' || c === 'STD') return 'PAGI';
  if (c === 'MALAM' || c === 'SPLIT') return 'MALAM';
  if (c === 'OFF' || c === 'LIBUR' || c === 'L') return 'OFF';
  if (c === 'CUTI' || c === 'CG') return 'CUTI';
  if (c === 'SAKIT' || c === 'SICK' || c === 'S') return 'SAKIT';
  if (c === 'ALFA' || c === 'ALPA' || c === 'MANGKIR' || c === 'A') return 'ALFA';
  if (c === 'IJIN' || c === 'IZIN' || c === 'I') return 'IJIN';
  return 'OTHER';
}

// Samakan nama divisi karyawan ke kunci divisi dashboard.
export function matchRosterDivisiKeyword(divLower: string): string {
  const d = String(divLower || '');
  if (!d) return '';
  if (d.includes('marketplace') || d.includes('market place')) return 'Marketplace';
  if (d.includes('distributor')) return 'Distributor';
  if (d.includes('return')) return 'Return';
  if (d.includes('inbound')) return 'Inbound';
  if (d.includes('pinkpick') || d.includes('pink pick')) return 'Pinkpick';
  if (d.includes('kol')) return 'KOL';
  if (d.includes('consumable')) return 'Consumable';
  if (d.includes('inventory')) return 'Inventory Control';
  if (d.includes('maintenance')) return 'Maintenance';
  if (d.includes('cleaning')) return 'Cleaning Service';
  if (d.includes('security') || d.includes('satpam')) return 'Security';
  return '';
}

export interface RosterRow {
  nama: string;
  [day: string]: string | undefined;
}

// Normalisasi baris roster mentah -> [{nama,'1'..'31'}] + dedup per nama.
export function normalizeRosterRows(rows: any[], monthYear?: string): RosterRow[] {
  const map = new Map<string, RosterRow>();
  for (const row of Array.isArray(rows) ? rows : []) {
    if (!row) continue;
    const bulan = String(row.Bulan || row.bulan || '').substring(0, 7);
    if (monthYear && bulan && bulan !== monthYear) continue;
    const nama = String(row.Nama || row.nama || '').trim();
    if (!nama) continue;
    const key = nama.toLowerCase();
    let obj = map.get(key);
    if (!obj) {
      obj = { nama };
      for (let i = 1; i <= 31; i++) obj[String(i)] = '';
      map.set(key, obj);
    }
    for (let i = 1; i <= 31; i++) {
      const k = String(i);
      const vv = row[k];
      if (vv !== undefined && vv !== null && String(vv).trim() !== '') obj[k] = String(vv);
    }
  }
  return Array.from(map.values());
}

export interface RosterDaySummary {
  pagi: number;
  malam: number;
  off: number;
  cuti: number;
  sakit: number;
  alfa: number;
  ijin: number;
  total: number;
  perDivisi: Record<string, { pagi: number; malam: number; total: number }>;
}

export const ROSTER_DIVISI_KEYS = [
  'Marketplace',
  'Distributor',
  'Return',
  'Inbound',
  'Pinkpick',
  'KOL',
  'Consumable',
  'Inventory Control',
  'Maintenance',
  'Cleaning Service',
  'Security',
] as const;

export const DIVISI_COLORS: Record<string, string> = {
  Marketplace: '#f59e0b',
  Distributor: '#22c55e',
  Return: '#ef4444',
  Inbound: '#14b8a6',
  Pinkpick: '#3b82f6',
  KOL: '#f59e0b',
  Consumable: '#8b5cf6',
  'Inventory Control': '#06b6d4',
  Maintenance: '#64748b',
  'Cleaning Service': '#ec4899',
  Security: '#38bdf8',
};

export function summarizeRosterDay(
  rosterRows: RosterRow[],
  dayNum: number | string,
  karyawanMap: Record<string, string>,
): RosterDaySummary {
  const s: RosterDaySummary = {
    pagi: 0,
    malam: 0,
    off: 0,
    cuti: 0,
    sakit: 0,
    alfa: 0,
    ijin: 0,
    total: 0,
    perDivisi: {},
  };
  for (const d of ROSTER_DIVISI_KEYS) s.perDivisi[d] = { pagi: 0, malam: 0, total: 0 };

  const day = String(dayNum);
  for (const row of rosterRows) {
    const cat = classifyRosterCode(row[day]);
    if (cat === 'EMPTY' || cat === 'OTHER') continue;
    if (cat === 'OFF') { s.off++; continue; }
    if (cat === 'CUTI') { s.cuti++; continue; }
    if (cat === 'SAKIT') { s.sakit++; continue; }
    if (cat === 'ALFA') { s.alfa++; continue; }
    if (cat === 'IJIN') { s.ijin++; continue; }

    const isPagi = cat === 'PAGI';
    const isMalam = cat === 'MALAM';
    const nama = String(row.nama || '').trim().toLowerCase();
    const div = karyawanMap[nama] || '';
    const key = matchRosterDivisiKeyword(div.toLowerCase());
    if (key) {
      s.perDivisi[key].total++;
      if (isPagi) s.perDivisi[key].pagi++;
      else if (isMalam) s.perDivisi[key].malam++;
    }
    if (isPagi) s.pagi++;
    else if (isMalam) s.malam++;
    s.total++;
  }
  return s;
}
