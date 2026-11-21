<script lang="ts" setup>
/**
 * Data Karyawan (FCL) — daftar aktif / riwayat resign / surat peringatan.
 * Layout mengikuti referensi: tab, toolbar (Template/Import/Ekspor/Cari/Tambah),
 * tabel dengan pagination, aksi Edit / Cetak ID Card QR / Beri SP.
 */
import { computed, onMounted, ref, watch } from 'vue';

import { Page } from '@vben/common-ui';
import { LucideDownload, LucidePlus } from '@vben/icons';

import { Modal, message, notification } from 'ant-design-vue';
import dayjs from 'dayjs';

import { fclDelete, fclInsert, fclPatch, fclQuery } from '#/api/fcl';
import FclPagination from '#/components/fcl/fcl-pagination.vue';

interface Karyawan {
  id: string;
  nama?: string;
  divisi?: string;
  jabatan?: string;
  cabang?: string;
  telepon?: string;
  email?: string;
  tanggalMasuk?: string;
  tanggalSelesai?: string;
  status?: string;
  sisaCuti?: number | string;
  fingerprintId?: string;
  createdAt?: string;
}
interface SP {
  id: string;
  karyawanId?: string;
  karyawanNama?: string;
  jenisSP?: string;
  alasan?: string;
  tanggalSP?: string;
  masaBerlaku?: string;
  tanggalKadaluarsa?: string;
  status?: string;
}

const loading = ref(true);
const all = ref<Karyawan[]>([]);
const spList = ref<SP[]>([]);
const tab = ref<'aktif' | 'resign' | 'sp'>('aktif');
const search = ref('');
const page = ref(1);
const pageSize = ref(20);

const modalOpen = ref(false);
const editing = ref<Karyawan | null>(null);
const saving = ref(false);
const form = ref({
  nama: '', divisi: '', jabatan: '', cabang: '', telepon: '', email: '',
  tanggalMasuk: '', tanggalSelesai: '', status: 'Kontrak', sisaCuti: 0 as number | undefined, fingerprintId: '',
});

const spOpen = ref(false);
const spTarget = ref<Karyawan | null>(null);
const spForm = ref({ jenisSP: 'SP 1', alasan: '', tanggalSP: dayjs().format('YYYY-MM-DD'), masaBerlaku: '3 bulan' });
const importInput = ref<HTMLInputElement | null>(null);

const rows = computed(() => all.value.filter((k) =>
  tab.value === 'resign'
    ? String(k.status ?? '').toLowerCase() === 'resign'
    : String(k.status ?? '').toLowerCase() !== 'resign',
));
const spRows = computed(() => spList.value.slice().sort((a, b) => String(b.tanggalSP ?? '').localeCompare(String(a.tanggalSP ?? ''))));

const filtered = computed(() => {
  const q = search.value.trim().toLowerCase();
  if (tab.value === 'sp') {
    if (!q) return spRows.value;
    return spRows.value.filter((s) => [s.karyawanNama, s.jenisSP, s.alasan].some((v) => String(v ?? '').toLowerCase().includes(q)));
  }
  if (!q) return rows.value;
  return rows.value.filter((k) => [k.nama, k.divisi, k.jabatan, k.cabang, k.fingerprintId]
    .some((v) => String(v ?? '').toLowerCase().includes(q)));
});
const paged = computed(() => filtered.value.slice((page.value - 1) * pageSize.value, page.value * pageSize.value));

const pagedKaryawan = computed(() => paged.value as Karyawan[]);
const pagedSp = computed(() => paged.value as SP[]);

function spCount(k: Karyawan): number {
  return spList.value.filter((s) => String(s.karyawanId) === String(k.id) || String(s.karyawanNama) === String(k.nama)).length;
}

function fmtDate(v?: string): string {
  const d = dayjs(v);
  return d.isValid() ? d.format('DD MMM\nYYYY') : '-';
}
function expired(v?: string): boolean {
  if (!v) return false;
  const d = dayjs(v);
  return d.isValid() && d.isBefore(dayjs(), 'day');
}

async function load(silent = false) {
  if (!silent) loading.value = true;
  try {
    const [k, s] = await Promise.all([
      fclQuery<Karyawan>('karyawan', { select: '*', order: 'createdAt.desc', limit: 10000 }),
      fclQuery<SP>('suratperingatan', { select: '*', order: 'createdAt.desc', limit: 5000 }),
    ]);
    all.value = k;
    spList.value = s;
  } catch (err: any) {
    console.error('[fcl/karyawan]', err);
    notification.error({ message: 'Data Karyawan', description: err?.message || 'Gagal memuat data' });
  } finally {
    if (!silent) loading.value = false;
  }
}

function openModal(k?: Karyawan | null) {
  editing.value = k ?? null;
  form.value = k
    ? {
        nama: k.nama ?? '', divisi: k.divisi ?? '', jabatan: k.jabatan ?? '', cabang: k.cabang ?? '',
        telepon: k.telepon ?? '', email: k.email ?? '', tanggalMasuk: (k.tanggalMasuk ?? '').slice(0, 10),
        tanggalSelesai: (k.tanggalSelesai ?? '').slice(0, 10), status: k.status ?? 'Kontrak',
        sisaCuti: Number(k.sisaCuti ?? 0), fingerprintId: k.fingerprintId ?? '',
      }
    : {
        nama: '', divisi: '', jabatan: '', cabang: '', telepon: '', email: '',
        tanggalMasuk: dayjs().format('YYYY-MM-DD'), tanggalSelesai: '', status: 'Kontrak',
        sisaCuti: 0, fingerprintId: '',
      };
  modalOpen.value = true;
}

async function save() {
  if (!form.value.nama.trim()) { message.warning('Nama wajib diisi'); return; }
  saving.value = true;
  const payload: Record<string, any> = {
    nama: form.value.nama.trim(), divisi: form.value.divisi.trim(), jabatan: form.value.jabatan.trim(),
    cabang: form.value.cabang.trim(), telepon: form.value.telepon.trim(), email: form.value.email.trim(),
    tanggalMasuk: form.value.tanggalMasuk, tanggalSelesai: form.value.tanggalSelesai || null,
    status: form.value.status, sisaCuti: Number(form.value.sisaCuti ?? 0), fingerprintId: form.value.fingerprintId.trim(),
  };
  try {
    const oldName = editing.value?.nama ?? '';
    let id = editing.value?.id;
    if (editing.value) await fclPatch('karyawan', editing.value.id, payload);
    else {
      const created = (await fclInsert('karyawan', [{ ...payload, createdAt: dayjs().format('YYYY-MM-DD HH:mm:ss') }])) as any;
      id = Array.isArray(created) ? created?.[0]?.id : created?.id;
    }
    // Jaga agar nama/divisi/jabatan di Absensi Karyawan ikut berubah
    if (id) {
      const absen = await fclQuery<any>('absensikaryawan', { select: '*', limit: 20000 });
      for (const a of absen) {
        const linked = String(a.karyawanId ?? '') === String(id)
          || (oldName && String(a.nama ?? '') === String(oldName))
          || (!oldName && String(a.nama ?? '') === String(payload.nama));
        if (!linked) continue;
        await fclPatch('absensikaryawan', a.id, {
          karyawanId: String(id), nama: payload.nama, divisi: payload.divisi,
          jabatan: payload.jabatan, fingerprintId: payload.fingerprintId,
        });
      }
    }
    notification.success({ message: 'Karyawan', description: editing.value ? 'Data diperbarui & absensi tersinkron' : 'Karyawan ditambahkan' });
    modalOpen.value = false;
    await load(true);
  } catch (err: any) {
    notification.error({ message: 'Karyawan', description: err?.message || 'Gagal menyimpan' });
  } finally {
    saving.value = false;
  }
}

function removeK(k: Karyawan) {
  Modal.confirm({
    title: `Hapus ${k.nama}?`, content: 'Data karyawan akan dihapus permanen.',
    okText: 'Hapus', okType: 'danger',
    onOk: async () => { await fclDelete('karyawan', k.id); notification.success({ message: 'Karyawan dihapus' }); await load(true); },
  });
}

async function resign(k: Karyawan) {
  await fclPatch('karyawan', k.id, { status: 'Resign' });
  notification.success({ message: 'Karyawan', description: 'Ditandai Resign — pindah ke Riwayat Resign' });
  await load(true);
}

async function reaktif(k: Karyawan) {
  await fclPatch('karyawan', k.id, { status: 'Kontrak' });
  await load(true);
}

function openSP(k: Karyawan) {
  spTarget.value = k;
  spForm.value = { jenisSP: 'SP 1', alasan: '', tanggalSP: dayjs().format('YYYY-MM-DD'), masaBerlaku: '3 bulan' };
  spOpen.value = true;
}

async function saveSP() {
  if (!spTarget.value) return;
  if (!spForm.value.alasan.trim()) { message.warning('Alasan wajib diisi'); return; }
  await fclInsert('suratperingatan', [{
    karyawanId: spTarget.value.id, karyawanNama: spTarget.value.nama,
    ...spForm.value, status: 'Aktif', createdBy: 'admin', createdAt: dayjs().format('YYYY-MM-DD HH:mm:ss'),
  }]);
  notification.success({ message: 'Surat Peringatan', description: 'SP berhasil dicatat' });
  spOpen.value = false;
  await load(true);
}

function printIdCard(k: Karyawan) {
  const w = window.open('', '_blank', 'width=420,height=300');
  if (!w) { message.warning('Popup diblokir browser'); return; }
  w.document.write(`<!doctype html><html><head><meta charset="utf-8"><title>ID Card ${String(k.nama)}</title><style>
  body{font-family:'Segoe UI',Arial,sans-serif;margin:18px}
  .card{border:2px solid #111;border-radius:12px;padding:16px;max-width:360px}
  .name{font-size:17px;font-weight:800}.meta{font-size:11px;color:#444;margin-top:6px;line-height:1.6}
  .code{font-family:monospace;font-size:18px;font-weight:800;margin-top:12px;letter-spacing:2px}
  </style></head><body><div class="card"><div class="name">${String(k.nama ?? '').toUpperCase()}</div>
  <div class="meta">${String(jabatanOf(k))} · ${String(k.divisi ?? '-')}<br/>${String(k.cabang ?? '-')}<br/>Status: ${String(k.status ?? '-')}</div>
  <div class="code">${String(k.fingerprintId || '-')}</div><div class="meta">Fingerprint ID / QR</div></div></body></html>`);
  w.document.close(); w.focus();
  setTimeout(() => { try { w.print(); } catch { /* abaikan */ } }, 300);
}

function jabatanOf(k: Karyawan) { return k.jabatan || '-'; }

function downloadCsv(filename: string, lines: string[][]) {
  const csv = lines.map((cols) => cols.map((c) => `"${String(c ?? '').replace(/"/g, '""')}"`).join(',')).join('\n');
  const url = URL.createObjectURL(new Blob(['﻿' + csv], { type: 'text/csv;charset=utf-8;' }));
  const a = document.createElement('a'); a.href = url; a.download = filename; a.click(); URL.revokeObjectURL(url);
}
function template() {
  downloadCsv('template-karyawan.csv', [['Nama', 'Divisi', 'Jabatan', 'Lokasi', 'Status', 'Fingerprint ID', 'Tgl Masuk', 'Selesai Kontrak', 'Sisa Cuti']]);
}
function exportCsv() {
  const lines = [['Nama', 'Divisi', 'Jabatan', 'Lokasi', 'Status', 'Fingerprint ID', 'Tgl Masuk', 'Selesai Kontrak', 'Sisa Cuti']];
  rows.value.forEach((k) => lines.push([k.nama ?? '', k.divisi ?? '', k.jabatan ?? '', k.cabang ?? '', k.status ?? '', k.fingerprintId ?? '', k.tanggalMasuk ?? '', k.tanggalSelesai ?? '', String(k.sisaCuti ?? 0)]));
  downloadCsv('data-karyawan.csv', lines);
}
async function importCsv(event: Event) {
  const file = (event.target as HTMLInputElement).files?.[0];
  if (!file) return;
  try {
    const parsed = (await file.text()).replace(/^\uFEFF/, '').split(/\r?\n/).filter(Boolean)
      .map((l) => (l.match(/("([^"]|"")*"|[^,])+/g) ?? []).map((c) => c.replace(/^"|"$/g, '').replace(/""/g, '"').trim()));
    const payload = parsed.slice(1).filter((c) => c[0]).map((c) => ({
      nama: c[0], divisi: c[1] ?? '', jabatan: c[2] ?? '', cabang: c[3] ?? '', status: c[4] || 'Kontrak',
      fingerprintId: c[5] ?? '', tanggalMasuk: c[6] ?? '', tanggalSelesai: c[7] ?? '', sisaCuti: Number(c[8] ?? 0),
      createdAt: dayjs().format('YYYY-MM-DD HH:mm:ss'),
    }));
    if (!payload.length) throw new Error('Tidak ada baris valid');
    await fclInsert('karyawan', payload);
    notification.success({ message: 'Import', description: `${payload.length} karyawan ditambahkan` });
    await load(true);
  } catch (err: any) {
    notification.error({ message: 'Import', description: err?.message || 'Gagal membaca CSV' });
  } finally {
    if (importInput.value) importInput.value.value = '';
  }
}

watch([tab, search], () => { page.value = 1; });
onMounted(() => load());
</script>

<template>
  <Page auto-content-height>
    <!-- Tab -->
    <div class="kw-tabs">
      <button class="kw-tab" :class="{ 'kw-tab-active': tab === 'aktif' }" @click="tab = 'aktif'">
        Karyawan Aktif <span>{{ rows.length }}</span>
      </button>
      <button class="kw-tab" :class="{ 'kw-tab-active': tab === 'resign' }" @click="tab = 'resign'">
        Riwayat Resign <span>{{ all.filter((k) => String(k.status).toLowerCase() === 'resign').length }}</span>
      </button>
      <button class="kw-tab" :class="{ 'kw-tab-active': tab === 'sp' }" @click="tab = 'sp'">
        Surat Peringatan <span>{{ spRows.length }}</span>
      </button>
    </div>

    <div class="kw-panel">
      <div class="kw-head">
        <div class="kw-title">{{ tab === 'sp' ? 'Surat Peringatan' : tab === 'resign' ? 'Riwayat Karyawan Resign' : 'Data Karyawan Aktif' }}</div>
        <div class="kw-actions">
          <template v-if="tab !== 'sp'">
            <button class="kw-btn kw-btn-ghost" @click="template"><LucideDownload :size="13" /> Template</button>
            <input ref="importInput" type="file" accept=".csv,text/csv" class="kw-hidden" @change="importCsv" />
            <button class="kw-btn kw-btn-accent" @click="importInput?.click()"><LucidePlus :size="13" /> Import Excel</button>
            <button class="kw-btn kw-btn-success" @click="exportCsv"><LucideDownload :size="13" /> Ekspor</button>
          </template>
          <input v-model="search" class="kw-search" placeholder="Cari..." />
          <button class="kw-btn kw-btn-primary" @click="openModal()"><LucidePlus :size="13" /> Tambah</button>
        </div>
      </div>

      <div class="kw-table-wrap">
        <table v-if="tab !== 'sp'" class="kw-table">
          <thead>
            <tr><th>NAMA</th><th>DIVISI</th><th>JABATAN</th><th>LOKASI</th><th>STATUS</th><th>FINGERPRINT ID</th><th>TGL MASUK</th><th>SELESAI KONTRAK</th><th>SISA CUTI</th><th>SP</th><th>AKSI</th></tr>
          </thead>
          <tbody>
            <tr v-for="k in pagedKaryawan" :key="k.id">
              <td class="kw-name">{{ k.nama }}</td>
              <td><span class="kw-badge kw-badge-div">{{ k.divisi || '-' }}</span></td>
              <td>{{ k.jabatan || '-' }}</td>
              <td><span class="kw-badge kw-badge-lok">{{ k.cabang || '-' }}</span></td>
              <td><span class="kw-status" :class="String(k.status).toLowerCase() === 'tetap' ? 'kw-status-tetap' : String(k.status).toLowerCase() === 'resign' ? 'kw-status-resign' : 'kw-status-kontrak'">{{ k.status || '-' }}</span></td>
              <td><span class="kw-badge kw-badge-fp">{{ k.fingerprintId || '-' }}</span></td>
              <td class="kw-date">{{ fmtDate(k.tanggalMasuk) }}</td>
              <td class="kw-date" :class="{ 'kw-danger': expired(k.tanggalSelesai) }">
                <template v-if="k.tanggalSelesai">{{ fmtDate(k.tanggalSelesai) }}<span v-if="expired(k.tanggalSelesai)" class="kw-late">({{ dayjs().diff(dayjs(k.tanggalSelesai), 'day') }} hr)</span></template>
                <template v-else>-</template>
              </td>
              <td>{{ k.sisaCuti ?? 0 }} hr</td>
              <td>{{ spCount(k) || '-' }}</td>
              <td>
                <div class="kw-row-actions">
                  <button class="kw-btn kw-btn-ghost kw-btn-xs" @click="openModal(k)">Edit Data</button>
                  <button class="kw-btn kw-btn-ghost kw-btn-xs" @click="printIdCard(k)">Cetak ID Card QR</button>
                  <button v-if="tab === 'aktif'" class="kw-btn kw-btn-accent kw-btn-xs" @click="openSP(k)">Beri SP</button>
                  <button v-if="tab === 'aktif'" class="kw-btn kw-btn-danger kw-btn-xs" @click="resign(k)">Resign</button>
                  <button v-if="tab === 'resign'" class="kw-btn kw-btn-success kw-btn-xs" @click="reaktif(k)">Aktifkan</button>
                  <button class="kw-btn kw-btn-danger kw-btn-xs" @click="removeK(k)">Hapus</button>
                </div>
              </td>
            </tr>
            <tr v-if="!paged.length"><td colspan="11" class="kw-empty">Belum ada data karyawan.</td></tr>
          </tbody>
        </table>

        <table v-else class="kw-table">
          <thead><tr><th>NAMA</th><th>JENIS SP</th><th>ALASAN</th><th>TANGGAL SP</th><th>MASA BERLAKU</th><th>STATUS</th></tr></thead>
          <tbody>
            <tr v-for="s in pagedSp" :key="s.id">
              <td class="kw-name">{{ s.karyawanNama || '-' }}</td><td>{{ s.jenisSP || '-' }}</td><td>{{ s.alasan || '-' }}</td>
              <td>{{ (s.tanggalSP || '').slice(0, 10) }}</td><td>{{ s.masaBerlaku || '-' }}</td><td>{{ s.status || '-' }}</td>
            </tr>
            <tr v-if="!paged.length"><td colspan="6" class="kw-empty">Belum ada surat peringatan.</td></tr>
          </tbody>
        </table>
      </div>
      <FclPagination v-model:page="page" v-model:page-size="pageSize" :total="filtered.length" />
    </div>

    <!-- Modal tambah/edit -->
    <a-modal v-model:open="modalOpen" :title="editing ? 'Edit Data Karyawan' : 'Tambah Karyawan'" :confirm-loading="saving" ok-text="Simpan" cancel-text="Batal" width="640px" @ok="save">
      <div class="kw-form">
        <div class="kw-form-row"><div class="kw-field"><label>Nama *</label><input v-model="form.nama" class="kw-input" /></div><div class="kw-field"><label>Divisi</label><input v-model="form.divisi" class="kw-input" /></div></div>
        <div class="kw-form-row"><div class="kw-field"><label>Jabatan</label><input v-model="form.jabatan" class="kw-input" /></div><div class="kw-field"><label>Lokasi / Cabang</label><input v-model="form.cabang" class="kw-input" /></div></div>
        <div class="kw-form-row"><div class="kw-field"><label>Status</label><select v-model="form.status" class="kw-input"><option>Kontrak</option><option>Tetap</option><option>Resign</option></select></div><div class="kw-field"><label>Fingerprint ID</label><input v-model="form.fingerprintId" class="kw-input" /></div></div>
        <div class="kw-form-row"><div class="kw-field"><label>Tanggal Masuk</label><input v-model="form.tanggalMasuk" type="date" class="kw-input" /></div><div class="kw-field"><label>Selesai Kontrak</label><input v-model="form.tanggalSelesai" type="date" class="kw-input" /></div></div>
        <div class="kw-form-row"><div class="kw-field"><label>Sisa Cuti</label><input v-model.number="form.sisaCuti" type="number" min="0" class="kw-input" /></div><div class="kw-field"><label>Telepon</label><input v-model="form.telepon" class="kw-input" /></div></div>
        <div class="kw-field"><label>Email</label><input v-model="form.email" class="kw-input" /></div>
      </div>
    </a-modal>

    <!-- Modal SP -->
    <a-modal v-model:open="spOpen" title="Beri Surat Peringatan" ok-text="Simpan" cancel-text="Batal" width="480px" @ok="saveSP">
      <div class="kw-form">
        <div class="kw-field"><label>Karyawan</label><input :value="spTarget?.nama || ''" class="kw-input" disabled /></div>
        <div class="kw-form-row"><div class="kw-field"><label>Jenis SP</label><select v-model="spForm.jenisSP" class="kw-input"><option>SP 1</option><option>SP 2</option><option>SP 3</option></select></div><div class="kw-field"><label>Tanggal</label><input v-model="spForm.tanggalSP" type="date" class="kw-input" /></div></div>
        <div class="kw-field"><label>Alasan *</label><input v-model="spForm.alasan" class="kw-input" /></div>
        <div class="kw-field"><label>Masa Berlaku</label><input v-model="spForm.masaBerlaku" class="kw-input" /></div>
      </div>
    </a-modal>
  </Page>
</template>

<style scoped>
/* Data Karyawan — token tema */
.kw-tabs { display: flex; gap: 6px; margin-bottom: 10px; flex-wrap: wrap; }
.kw-tab { border: 1px solid hsl(var(--border)); background: hsl(var(--muted)); color: hsl(var(--muted-foreground)); font-size: 11px; font-weight: 800; padding: 6px 12px; border-radius: 999px; cursor: pointer; }
.kw-tab span { font-size: 10px; opacity: 0.8; margin-left: 4px; }
.kw-tab-active { background: hsl(var(--warning)); border-color: hsl(var(--warning)); color: hsl(var(--warning-foreground)); }

.kw-panel { background: hsl(var(--card)); border: 1px solid hsl(var(--border)); border-radius: 8px; }
.kw-head { display: flex; justify-content: space-between; align-items: center; gap: 10px; flex-wrap: wrap; padding: 12px; }
.kw-title { font-size: 12px; font-weight: 800; color: hsl(var(--foreground)); }
.kw-actions { display: flex; gap: 6px; align-items: center; flex-wrap: wrap; }
.kw-hidden { display: none; }
.kw-search { width: 160px; height: 30px; padding: 2px 9px; border-radius: 6px; border: 1px solid hsl(var(--border)); background: hsl(var(--background)); color: hsl(var(--foreground)); font-size: 11px; outline: none; }
.kw-search:focus { border-color: hsl(var(--primary)); }

.kw-btn { display: inline-flex; align-items: center; justify-content: center; gap: 5px; border: 1px solid hsl(var(--border)); background: hsl(var(--muted)); color: hsl(var(--foreground)); font-size: 11px; font-weight: 700; padding: 6px 11px; border-radius: 6px; cursor: pointer; white-space: nowrap; height: 30px; }
.kw-btn:hover { filter: brightness(0.96); }
.kw-btn-xs { height: 24px; padding: 2px 8px; font-size: 10px; }
.kw-btn-primary { background: hsl(var(--primary)); border-color: hsl(var(--primary)); color: hsl(var(--primary-foreground)); }
.kw-btn-accent { background: hsl(var(--warning)); border-color: hsl(var(--warning)); color: hsl(var(--warning-foreground)); }
.kw-btn-success { background: hsl(var(--success)); border-color: hsl(var(--success)); color: hsl(var(--success-foreground)); }
.kw-btn-danger { background: hsl(var(--destructive)); border-color: hsl(var(--destructive)); color: hsl(var(--destructive-foreground)); }

.kw-table-wrap { overflow-x: auto; border-top: 1px solid hsl(var(--border)); }
.kw-table { width: 100%; min-width: 1180px; border-collapse: collapse; font-size: 11px; }
.kw-table th { background: hsl(var(--muted)); color: hsl(var(--muted-foreground)); font-size: 10px; font-weight: 800; letter-spacing: 0.04em; text-align: left; padding: 8px 10px; border-bottom: 1px solid hsl(var(--border)); white-space: nowrap; }
.kw-table td { padding: 8px 10px; border-bottom: 1px solid hsl(var(--border)); color: hsl(var(--foreground)); vertical-align: middle; }
.kw-table tbody tr:nth-child(even) { background: hsl(var(--muted) / 0.4); }
.kw-table tbody tr:hover { background: hsl(var(--accent)); }
.kw-name { font-weight: 800; white-space: nowrap; }
.kw-badge { display: inline-block; font-size: 10px; font-weight: 700; padding: 2px 7px; border-radius: 5px; background: hsl(var(--muted)); color: hsl(var(--foreground)); white-space: nowrap; }
.kw-badge-div { color: hsl(var(--warning)); background: hsl(var(--warning) / 0.12); }
.kw-badge-lok { background: hsl(var(--muted)); }
.kw-badge-fp { font-family: monospace; }
.kw-status { font-size: 10px; font-weight: 800; padding: 3px 10px; border-radius: 999px; white-space: nowrap; }
.kw-status-tetap { background: hsl(var(--success) / 0.14); color: hsl(var(--success)); }
.kw-status-kontrak { background: hsl(var(--warning) / 0.16); color: hsl(var(--warning)); }
.kw-status-resign { background: hsl(var(--destructive) / 0.14); color: hsl(var(--destructive)); }
.kw-date { white-space: pre-line; line-height: 1.3; }
.kw-danger { color: hsl(var(--destructive)); font-weight: 700; }
.kw-late { display: block; font-size: 9px; }
.kw-row-actions { display: flex; gap: 4px; flex-wrap: wrap; }
.kw-empty { text-align: center; color: hsl(var(--muted-foreground)); padding: 22px 8px; }

.kw-form { display: flex; flex-direction: column; gap: 10px; padding-top: 4px; }
.kw-form-row { display: grid; grid-template-columns: 1fr 1fr; gap: 10px; }
.kw-field label { display: block; font-size: 11px; font-weight: 700; color: hsl(var(--muted-foreground)); margin-bottom: 4px; }
.kw-input { width: 100%; height: 32px; padding: 4px 9px; border-radius: 6px; border: 1px solid hsl(var(--border)); background: hsl(var(--background)); color: hsl(var(--foreground)); font-size: 12px; outline: none; }
.kw-input:focus { border-color: hsl(var(--primary)); }
</style>
