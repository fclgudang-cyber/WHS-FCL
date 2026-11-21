<script lang="ts" setup>
/**
 * Absensi Karyawan — daftar otomatis menampilkan SEMUA karyawan dari data master.
 * Setiap karyawan selalu muncul (meski belum absen), dengan data master tersinkron,
 * serta tombol Check In / Check Out untuk mencatat absensi pada tanggal pilihan.
 */
import { computed, onMounted, ref, watch } from 'vue';

import { Page } from '@vben/common-ui';
import { LucideDownload, LucideRefreshCw } from '@vben/icons';

import { Modal, message, notification } from 'ant-design-vue';
import dayjs from 'dayjs';

import { fclDelete, fclInsert, fclPatch, fclQuery } from '#/api/warehouse';
import FclPagination from '#/components/warehouse/warehouse-pagination.vue';

interface Karyawan {
  id: string;
  nama?: string;
  divisi?: string;
  jabatan?: string;
  fingerprintId?: string;
  status?: string;
}
interface Absen {
  id: string;
  tanggal?: string;
  jam?: string;
  karyawanId?: string;
  nama?: string;
  divisi?: string;
  jabatan?: string;
  tipe?: string;
  sumber?: string;
  fingerprintId?: string;
  status?: string;
  keterangan?: string;
}

const loading = ref(true);
const syncing = ref(false);
const karyawan = ref<Karyawan[]>([]);
const absen = ref<Absen[]>([]);
const tanggal = ref(dayjs().format('YYYY-MM-DD'));
const search = ref('');
const page = ref(1);
const pageSize = ref(20);

function norm(s: unknown): string { return String(s ?? '').trim().toLowerCase(); }

function masterOf(a: Absen): Karyawan | null {
  if (a.karyawanId) {
    const byId = karyawan.value.find((k) => String(k.id) === String(a.karyawanId));
    if (byId) return byId;
  }
  const fp = norm(a.fingerprintId);
  if (fp) {
    const byFp = karyawan.value.find((k) => norm(k.fingerprintId) === fp);
    if (byFp) return byFp;
  }
  return karyawan.value.find((k) => norm(k.nama) === norm(a.nama)) ?? null;
}

// Setiap karyawan selalu muncul, digabung dengan absen pada tanggal terpilih
const displayRows = computed(() => karyawan.value.map((k) => {
  const rows = absen.value.filter((a) => {
    const samePerson = String(a.karyawanId ?? '') === String(k.id)
      || (norm(a.fingerprintId) && norm(a.fingerprintId) === norm(k.fingerprintId))
      || norm(a.nama) === norm(k.nama);
    return samePerson && String(a.tanggal ?? '').slice(0, 10) === tanggal.value;
  }).sort((a, b) => String(a.jam ?? '').localeCompare(String(b.jam ?? '')));
  const masuk = rows.find((r) => String(r.tipe).toUpperCase() === 'IN');
  const pulang = [...rows].reverse().find((r) => String(r.tipe).toUpperCase() === 'OUT');
  return {
    id: String(k.id),
    nama: k.nama ?? '-',
    divisi: k.divisi ?? '-',
    jabatan: k.jabatan ?? '-',
    fingerprintId: k.fingerprintId ?? '-',
    statusKaryawan: k.status ?? '-',
    jamMasuk: masuk?.jam ?? '',
    jamPulang: pulang?.jam ?? '',
    sumber: masuk?.sumber ?? '',
    absenStatus: masuk && pulang ? 'Hadir' : masuk ? 'Belum Pulang' : 'Belum Absen',
    records: rows,
  };
}));

const filtered = computed(() => {
  const q = norm(search.value);
  if (!q) return displayRows.value;
  return displayRows.value.filter((r) =>
    [r.nama, r.divisi, r.jabatan, r.fingerprintId].some((v) => norm(v).includes(q)),
  );
});
const paged = computed(() => filtered.value.slice((page.value - 1) * pageSize.value, page.value * pageSize.value));
const hadirCount = computed(() => displayRows.value.filter((r) => r.absenStatus !== 'Belum Absen').length);

async function load(silent = false) {
  if (!silent) loading.value = true;
  try {
    const [k, a] = await Promise.all([
      fclQuery<Karyawan>('karyawan', { select: '*', order: 'nama.asc', limit: 5000 }),
      fclQuery<Absen>('absensikaryawan', { select: '*', order: 'tanggal.desc,jam.desc', limit: 30000 }),
    ]);
    karyawan.value = k;
    absen.value = a;
  } catch (err: any) {
    console.error('[warehouse/absensi]', err);
    notification.error({ message: 'Absensi Karyawan', description: err?.message || 'Gagal memuat data' });
  } finally {
    if (!silent) loading.value = false;
  }
}

// Samakan data master ke seluruh baris absen yang cocok
async function syncNames() {
  const targets = absen.value.filter((a) => !!masterOf(a));
  if (!targets.length) { message.info('Belum ada baris absensi yang bisa disinkronkan'); return; }
  syncing.value = true;
  let ok = 0;
  try {
    for (const a of targets) {
      const m = masterOf(a)!;
      const patch: Record<string, any> = {};
      if (a.nama !== m.nama) patch.nama = m.nama;
      if (a.divisi !== m.divisi) patch.divisi = m.divisi;
      if (a.jabatan !== m.jabatan) patch.jabatan = m.jabatan;
      if (a.fingerprintId !== m.fingerprintId) patch.fingerprintId = m.fingerprintId ?? '';
      if (String(a.karyawanId ?? '') !== String(m.id)) patch.karyawanId = m.id;
      if (Object.keys(patch).length) { await fclPatch('absensikaryawan', a.id, patch); ok++; }
    }
    notification.success({ message: 'Sinkron', description: `${ok} baris absensi disesuaikan dengan data master` });
    await load(true);
  } catch (err: any) {
    notification.error({ message: 'Sinkron', description: err?.message || 'Gagal menyinkronkan' });
  } finally {
    syncing.value = false;
  }
}

async function catat(k: Karyawan & Record<string, any>, tipe: 'IN' | 'OUT') {
  await fclInsert('absensikaryawan', [{
    karyawanId: String(k.id), nama: k.nama, divisi: k.divisi, jabatan: k.jabatan,
    fingerprintId: k.fingerprintId ?? '', tanggal: tanggal.value, jam: dayjs().format('HH:mm'),
    tipe, sumber: 'Manual', status: 'Hadir', keterangan: '',
    createdAt: dayjs().format('YYYY-MM-DD HH:mm:ss'),
  }]);
  notification.success({ message: 'Absensi', description: `${k.nama} — ${tipe} ${dayjs().format('HH:mm')}` });
  await load(true);
}

function hapusRecord(a: Absen) {
  Modal.confirm({
    title: 'Hapus catatan absen?', content: `${a.nama} · ${a.tanggal} ${a.jam} ${a.tipe}`,
    okText: 'Hapus', okType: 'danger',
    onOk: async () => { await fclDelete('absensikaryawan', a.id); await load(true); },
  });
}

function downloadCsv(filename: string, lines: string[][]) {
  const csv = lines.map((cols) => cols.map((c) => `"${String(c ?? '').replace(/"/g, '""')}"`).join(',')).join('\n');
  const url = URL.createObjectURL(new Blob(['﻿' + csv], { type: 'text/csv;charset=utf-8;' }));
  const a = document.createElement('a'); a.href = url; a.download = filename; a.click(); URL.revokeObjectURL(url);
}
function exportCsv() {
  const lines = [['Tanggal', 'Nama', 'Divisi', 'Jabatan', 'Fingerprint ID', 'Jam Masuk', 'Jam Pulang', 'Status']];
  filtered.value.forEach((r) => lines.push([
    tanggal.value, r.nama, r.divisi, r.jabatan, r.fingerprintId,
    r.jamMasuk, r.jamPulang, r.absenStatus,
  ]));
  downloadCsv(`absensi-${tanggal.value}.csv`, lines);
}

watch([search, tanggal], () => { page.value = 1; });
onMounted(() => load());
</script>

<template>
  <Page auto-content-height>
    <div class="ab-root">
      <div class="ab-head">
        <div class="ab-title">Absensi Karyawan</div>
        <div class="ab-actions">
          <input v-model="tanggal" type="date" class="ab-date" />
          <button class="ab-btn ab-btn-ghost" @click="load()"><LucideRefreshCw :size="13" /> Refresh</button>
          <button class="ab-btn ab-btn-ghost" :loading="syncing" @click="syncNames"><LucideRefreshCw :size="13" /> Sinkron Nama</button>
          <button class="ab-btn ab-btn-ghost" @click="exportCsv"><LucideDownload :size="13" /> Export CSV</button>
          <input v-model="search" class="ab-search" placeholder="Cari nama / divisi..." />
        </div>
      </div>
      <div class="ab-hint">
        Menampilkan <b>{{ displayRows.length }} karyawan</b> dari data master ·
        {{ hadirCount }} sudah tercatat absennya pada {{ tanggal }}. Centang otomatis mengikuti perubahan data Karyawan.
      </div>

      <div class="ab-table-wrap">
        <table class="ab-table">
          <thead><tr><th>NAMA</th><th>DIVISI</th><th>JABATAN</th><th>FINGERPRINT ID</th><th>JAM MASUK</th><th>JAM PULANG</th><th>STATUS</th><th>CATATAN ABSEN</th></tr></thead>
          <tbody>
            <tr v-for="r in (paged as any[])" :key="r.id">
              <td class="ab-name">{{ r.nama }}</td>
              <td><span class="ab-badge">{{ r.divisi }}</span></td>
              <td>{{ r.jabatan }}</td>
              <td><span class="ab-badge ab-fp">{{ r.fingerprintId }}</span></td>
              <td>{{ r.jamMasuk || '-' }}</td>
              <td>{{ r.jamPulang || '-' }}</td>
              <td>
                <span class="ab-status" :class="r.absenStatus === 'Hadir' ? 'ab-hadir' : r.absenStatus === 'Belum Pulang' ? 'ab-setengah' : 'ab-belum'">
                  {{ r.absenStatus }}
                </span>
              </td>
              <td>
                <div class="ab-row-actions">
                  <button v-if="!r.jamMasuk" class="ab-btn ab-btn-success ab-btn-xs" @click="catat(r, 'IN')">Check In</button>
                  <button v-if="r.jamMasuk && !r.jamPulang" class="ab-btn ab-btn-accent ab-btn-xs" @click="catat(r, 'OUT')">Check Out</button>
                  <button v-for="a in r.records" :key="a.id" class="ab-btn ab-btn-ghost ab-btn-xs" @click="hapusRecord(a)">Hapus {{ a.tipe }} {{ a.jam }}</button>
                </div>
              </td>
            </tr>
            <tr v-if="!paged.length"><td colspan="8" class="ab-empty">Belum ada data karyawan. Tambahkan karyawan di menu SDM → Karyawan.</td></tr>
          </tbody>
        </table>
      </div>
      <FclPagination v-model:page="page" v-model:page-size="pageSize" :total="filtered.length" />
    </div>
  </Page>
</template>

<style scoped>
.ab-root { display: flex; flex-direction: column; gap: 10px; }
.ab-head { display: flex; justify-content: space-between; align-items: center; gap: 10px; flex-wrap: wrap; padding: 12px; background: hsl(var(--card)); border: 1px solid hsl(var(--border)); border-radius: 8px; }
.ab-title { font-size: 14px; font-weight: 800; color: hsl(var(--foreground)); }
.ab-actions { display: flex; gap: 6px; flex-wrap: wrap; }
.ab-btn { display: inline-flex; align-items: center; justify-content: center; gap: 5px; border: 1px solid hsl(var(--border)); background: hsl(var(--muted)); color: hsl(var(--foreground)); font-size: 11px; font-weight: 700; padding: 6px 11px; border-radius: 6px; cursor: pointer; height: 32px; white-space: nowrap; }
.ab-btn:hover { filter: brightness(0.96); }
.ab-btn-xs { height: 24px; padding: 2px 8px; font-size: 10px; }
.ab-btn-primary { background: hsl(var(--primary)); border-color: hsl(var(--primary)); color: hsl(var(--primary-foreground)); }
.ab-btn-success { background: hsl(var(--success)); border-color: hsl(var(--success)); color: hsl(var(--success-foreground)); }
.ab-btn-accent { background: hsl(var(--warning)); border-color: hsl(var(--warning)); color: hsl(var(--warning-foreground)); }
.ab-btn-danger { background: hsl(var(--destructive)); border-color: hsl(var(--destructive)); color: hsl(var(--destructive-foreground)); }
.ab-date, .ab-search { height: 32px; padding: 2px 9px; border-radius: 6px; border: 1px solid hsl(var(--border)); background: hsl(var(--background)); color: hsl(var(--foreground)); font-size: 11px; outline: none; }
.ab-search { width: 220px; }
.ab-date:focus, .ab-search:focus { border-color: hsl(var(--primary)); }
.ab-hint { font-size: 11px; color: hsl(var(--muted-foreground)); }
.ab-hint b { color: hsl(var(--primary)); }
.ab-table-wrap { overflow-x: auto; background: hsl(var(--card)); border: 1px solid hsl(var(--border)); border-radius: 8px; }
.ab-table { width: 100%; min-width: 980px; border-collapse: collapse; font-size: 11px; }
.ab-table th { background: hsl(var(--muted)); color: hsl(var(--muted-foreground)); font-size: 10px; font-weight: 800; text-align: left; padding: 8px 10px; border-bottom: 1px solid hsl(var(--border)); }
.ab-table td { padding: 8px 10px; border-bottom: 1px solid hsl(var(--border)); color: hsl(var(--foreground)); }
.ab-table tbody tr:nth-child(even) { background: hsl(var(--muted) / 0.4); }
.ab-name { font-weight: 800; }
.ab-badge { display: inline-block; font-size: 10px; font-weight: 700; padding: 2px 7px; border-radius: 5px; background: hsl(var(--muted)); white-space: nowrap; }
.ab-fp { font-family: monospace; }
.ab-status { display: inline-block; font-size: 10px; font-weight: 800; padding: 3px 10px; border-radius: 999px; white-space: nowrap; }
.ab-hadir { background: hsl(var(--success) / 0.14); color: hsl(var(--success)); }
.ab-setengah { background: hsl(var(--warning) / 0.16); color: hsl(var(--warning)); }
.ab-belum { background: hsl(var(--muted)); color: hsl(var(--muted-foreground)); }
.ab-row-actions { display: flex; gap: 4px; flex-wrap: wrap; }
.ab-empty { text-align: center; color: hsl(var(--muted-foreground)); padding: 22px 8px; }
</style>
