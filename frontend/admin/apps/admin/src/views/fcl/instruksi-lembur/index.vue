<script lang="ts" setup>
import { computed, onMounted, reactive, ref, watch } from 'vue';

import { Page } from '@vben/common-ui';
import { useUserStore } from '@vben/stores';
import { LucidePlus, LucideRefreshCw, LucideTrash2 } from '@vben/icons';

import { Modal, message, notification } from 'ant-design-vue';
import dayjs from 'dayjs';

import { fclDelete, fclDeleteWhere, fclInsert, fclPatch, fclQuery } from '#/api/fcl';

interface Karyawan { id: string; nama?: string | null; divisi?: string | null; jabatan?: string | null }
interface Roster { id: string; Bulan?: string; Nama?: string; Divisi?: string; Jabatan?: string; [key: string]: any }
interface Detail {
  key: string;
  nama: string;
  divisi: string;
  divisiTujuan: string;
  shift: string;
  jamKerja: number;
  jamLembur: number;
  totalJam: number;
  rosterId: string;
}

const userStore = useUserStore();
const namaAkun = () => {
  const u: any = userStore.userInfo ?? {};
  return String(u.realname || u.nama || u.username || u.nickname || '').trim();
};

const masters = ref<any[]>([]);
const details = ref<any[]>([]);
const karyawanList = ref<Karyawan[]>([]);
const rosterList = ref<Roster[]>([]);
const loading = ref(true);
const search = ref('');

const modalOpen = ref(false);
const editingId = ref<string | null>(null);
const saving = ref(false);
const detailOpen = ref(false);
const detailRows = ref<any[]>([]);
const detailTitle = ref('');

const form = reactive({
  tanggal: dayjs().format('YYYY-MM-DD'),
  judul: '',
  shiftLaporan: 'Pagi',
  divisiFilter: 'Semua Divisi',
  keterangan: '',
  terapkanKeJadwal: true,
});
const items = ref<Detail[]>([]);
const syncInfo = ref('');
const syncing = ref(false);

let seq = 0;
const newItem = (): Detail => ({
  key: `new-${++seq}`, nama: '', divisi: '', divisiTujuan: '', shift: '',
  jamKerja: 0, jamLembur: 0, totalJam: 0, rosterId: '',
});

const divisiOptions = computed(() => ['Semua Divisi', ...new Set(karyawanList.value
  .map((k) => String(k.divisi ?? '').trim()).filter(Boolean))].sort((a, b) => a === 'Semua Divisi' ? -1 : b === 'Semua Divisi' ? 1 : a.localeCompare(b, 'id')));
const namaOptions = computed(() => {
  const d = form.divisiFilter;
  return [...new Set(karyawanList.value
    .filter((k) => d === 'Semua Divisi' || String(k.divisi ?? '') === d)
    .map((k) => String(k.nama ?? '').trim()).filter(Boolean))].sort((a, b) => a.localeCompare(b, 'id'));
});

function karyawanMap(): Map<string, Karyawan> {
  const m = new Map<string, Karyawan>();
  karyawanList.value.forEach((k) => {
    const n = String(k.nama ?? '').trim().toLowerCase();
    if (n) m.set(n, k);
  });
  return m;
}
function rosterMap(): Map<string, Roster> {
  const m = new Map<string, Roster>();
  const monthKey = dayjs(form.tanggal).isValid() ? dayjs(form.tanggal).format('YYYY-MM') : '';
  rosterList.value.filter((r) => String(r.Bulan ?? '') === monthKey).forEach((r) => {
    const n = String(r.Nama ?? '').trim().toLowerCase();
    if (n) m.set(n, r);
  });
  return m;
}
function jamShiftDari(shift: string): number {
  const s = shift.toUpperCase();
  if (s.includes('SETENGAH') || s.includes('SPLIT')) return 4;
  if (s.includes('MALAM')) return 8;
  if (s.includes('PAGI')) return 8;
  return 0;
}

// Deteksi otomatis divisi (dari Karyawan) dan shift/jam (dari Jadwal Shift).
function isiOtomatis(row: Detail) {
  const k = karyawanMap().get(row.nama.trim().toLowerCase());
  if (k) row.divisi = String(k.divisi ?? '').trim();
  else row.divisi = '';
  const r = rosterMap().get(row.nama.trim().toLowerCase());
  const dayKey = String(dayjs(form.tanggal).date());
  const shift = r ? String(r[dayKey] ?? '').trim() : '';
  row.rosterId = r?.id ?? '';
  row.shift = shift;
  row.jamKerja = jamShiftDari(shift);
  row.totalJam = Number(row.jamKerja) + Number(row.jamLembur || 0);
}
watch(() => items.value.map((i) => i.nama), () => items.value.forEach(isiOtomatis));
watch(() => [form.tanggal, form.divisiFilter], () => {
  items.value.forEach(isiOtomatis);
  updateSyncInfo();
});

function updateSyncInfo() {
  const d = dayjs(form.tanggal);
  syncInfo.value = d.isValid()
    ? `Jadwal Shift ${d.format('YYYY-MM')} (tgl ${d.date()}) tersinkron`
    : 'Belum ada tanggal';
}
async function syncJadwal() {
  syncing.value = true;
  try {
    const [k, r] = await Promise.all([
      fclQuery<Karyawan>('karyawan', { select: '*', order: 'nama.asc', limit: 3000 }),
      fclQuery<Roster>('jadwalroster', { select: '*', limit: 10000 }),
    ]);
    karyawanList.value = k;
    rosterList.value = r;
    items.value.forEach(isiOtomatis);
    updateSyncInfo();
    notification.success({ message: 'Sinkronkan Jadwal Shift', description: 'Data Karyawan & roster berhasil dimuat' });
  } catch (err: any) {
    notification.error({ message: 'Sinkronkan Jadwal Shift', description: err?.message || 'Gagal memuat data' });
  } finally {
    syncing.value = false;
  }
}

async function load() {
  loading.value = true;
  try {
    const [m, d, k, r] = await Promise.all([
      fclQuery<any>('instruksilembur', { select: '*', order: 'tanggal.desc', limit: 1000 }),
      fclQuery<any>('instruksilemburdetail', { select: '*', limit: 5000 }),
      fclQuery<Karyawan>('karyawan', { select: '*', order: 'nama.asc', limit: 3000 }),
      fclQuery<Roster>('jadwalroster', { select: '*', limit: 10000 }),
    ]);
    masters.value = m;
    details.value = d;
    karyawanList.value = k;
    rosterList.value = r;
  } catch (err: any) {
    notification.error({ message: 'Instruksi Lembur', description: err?.message || 'Gagal memuat data' });
  } finally {
    loading.value = false;
  }
}
onMounted(load);

function bukaBaru() {
  editingId.value = null;
  form.tanggal = dayjs().format('YYYY-MM-DD');
  form.judul = '';
  form.shiftLaporan = 'Pagi';
  form.divisiFilter = 'Semua Divisi';
  form.keterangan = '';
  form.terapkanKeJadwal = true;
  items.value = [newItem(), newItem(), newItem()];
  updateSyncInfo();
  modalOpen.value = true;
}

const filteredMasters = computed(() => {
  const q = search.value.trim().toLowerCase();
  return q ? masters.value.filter((m) => `${m.judul ?? ''} ${m.keterangan ?? ''}`.toLowerCase().includes(q)) : masters.value;
});

function lihatDetail(m: any) {
  detailTitle.value = `${m.judul || 'Instruksi'} — ${String(m.tanggal || '').slice(0, 10)}`;
  detailRows.value = details.value.filter((d) => String(d.instruksiId) === String(m.id));
  detailOpen.value = true;
}

function hapusInstruksi(m: any) {
  Modal.confirm({
    title: 'Hapus instruksi?',
    content: `Hapus "${m.judul || 'instruksi'}" beserta detail karyawannya?`,
    okText: 'Hapus', okType: 'danger',
    async onOk() {
      await fclDeleteWhere('instruksilemburdetail', [{ col: 'instruksiId', value: String(m.id) }]);
      await fclDelete('instruksilembur', m.id);
      notification.success({ message: 'Instruksi dihapus' });
      await load();
    },
  });
}

async function simpan() {
  if (!form.tanggal) { message.warning('Tanggal lembur wajib diisi'); return; }
  if (!form.judul.trim()) { message.warning('Judul / pekerjaan wajib diisi'); return; }
  const valid = items.value.filter((i) => i.nama.trim());
  if (!valid.length) { message.warning('Pilih minimal satu nama karyawan'); return; }
  saving.value = true;
  try {
    const tanggal = dayjs(form.tanggal).format('YYYY-MM-DD');
    const inserted = await fclInsert<any>('instruksilembur', [{
      tanggal,
      judul: form.judul.trim(),
      keterangan: form.keterangan,
      createdBy: namaAkun(),
      createdAt: dayjs().format('YYYY-MM-DDTHH:mm:ss'),
    }]);
    const masterId = Array.isArray(inserted) ? inserted[0]?.id : (inserted as any)?.id;
    await fclInsert('instruksilemburdetail', valid.map((i) => ({
      instruksiId: masterId,
      nama: i.nama.trim(),
      divisi: i.divisi,
      divisiTujuan: i.divisiTujuan,
      tipe: 'Lembur',
      shift: i.shift,
      jamKerja: String(i.jamKerja),
      jamLembur: String(i.jamLembur),
      totalJam: String(i.totalJam),
    })));
    // Terapkan total jam ke kolom hari pada Jadwal Shift.
    if (form.terapkanKeJadwal) {
      const monthKey = dayjs(form.tanggal).format('YYYY-MM');
      const dayKey = String(dayjs(form.tanggal).date());
      const rmap = rosterMap();
      for (const i of valid) {
        const r = rmap.get(i.nama.trim().toLowerCase());
        if (r) {
          const nilai = i.shift
            ? `${i.shift} + LEMBUR ${i.jamLembur}J`
            : `LEMBUR ${i.totalJam}J`;
          await fclPatch('jadwalroster', r.id, { [dayKey]: nilai });
        }
      }
    }
    notification.success({ message: 'Instruksi lembur disimpan', description: `${valid.length} karyawan tercatat` });
    modalOpen.value = false;
    await load();
  } catch (err: any) {
    notification.error({ message: 'Instruksi Lembur', description: err?.message || 'Gagal menyimpan' });
  } finally {
    saving.value = false;
  }
}
</script>

<template>
  <Page auto-content-height>
    <div class="il-page">
      <div class="il-head">
        <div class="il-title">Instruksi Lembur</div>
        <div class="il-tools">
          <input v-model="search" type="text" class="il-input" placeholder="Cari instruksi..." />
          <button class="il-btn il-btn-primary" @click="bukaBaru"><LucidePlus :size="13" /> Instruksi Baru</button>
        </div>
      </div>
      <div class="il-card">
        <table class="il-table">
          <thead><tr><th>TANGGAL</th><th>JUDUL INSTRUKSI</th><th>KETERANGAN</th><th>DIBUAT OLEH</th><th>KARYAWAN</th><th>AKSI</th></tr></thead>
          <tbody>
            <tr v-for="m in filteredMasters" :key="m.id">
              <td>{{ String(m.tanggal || '').slice(0, 10) }}</td>
              <td><b>{{ m.judul || '-' }}</b></td>
              <td>{{ m.keterangan || '-' }}</td>
              <td>{{ m.createdBy || '-' }}</td>
              <td>{{ details.filter((d) => String(d.instruksiId) === String(m.id)).length }}</td>
              <td>
                <button class="il-btn il-btn-ghost" @click="lihatDetail(m)">Detail</button>
                <button class="il-btn il-btn-red" @click="hapusInstruksi(m)">Hapus</button>
              </td>
            </tr>
            <tr v-if="!filteredMasters.length"><td colspan="6" class="il-empty">{{ loading ? 'Memuat...' : 'Belum ada instruksi lembur.' }}</td></tr>
          </tbody>
        </table>
      </div>
    </div>

    <!-- Modal: tambah instruksi -->
    <a-modal v-model:open="modalOpen" title="Tambah Instruksi Lembur" width="1080px" :confirm-loading="saving" ok-text="Simpan Instruksi" cancel-text="Batal" @ok="simpan">
      <div class="il-form">
        <div class="il-note">
          Shift &amp; Jam Shift per baris <b>TERKUNCI</b> 🔒 berasal dari Jadwal Shift sesuai karyawan yang dipilih — tidak bisa diubah manual.
          Total = jam shift + jam lembur (mis. 8 + 3 = 11 jam). Khusus jam merah, jam lembur boleh dikosongkan (&lt; 8 jam normal).
        </div>
        <div class="il-grid4">
          <label class="il-f"><span>TANGGAL LEMBUR *</span><input v-model="form.tanggal" type="date" class="il-input" /></label>
          <label class="il-f"><span class="il-merah">JUDUL / PEKERJAAN * (WAJIB)</span><input v-model="form.judul" class="il-input" placeholder="Contoh: Lembur packing marketplace" /></label>
          <label class="il-f"><span>SHIFT LAPORAN (PAGI / MALAM)</span>
            <select v-model="form.shiftLaporan" class="il-input"><option>Pagi</option><option>Malam</option></select>
            <small>Laporan memakai shift {{ form.shiftLaporan }} — kolom Shift baris terkunci ikuti Jadwal Shift karyawan.</small>
          </label>
          <label class="il-f"><span>FILTER DIVISI</span>
            <select v-model="form.divisiFilter" class="il-input"><option v-for="d in divisiOptions" :key="d" :value="d">{{ d }}</option></select>
          </label>
        </div>
        <label class="il-f"><span>KETERANGAN</span><input v-model="form.keterangan" class="il-input" placeholder="Catatan tambahan (opsional)" /></label>

        <div class="il-syncbar">
          <span>✓ {{ syncInfo }}</span>
          <button type="button" class="il-btn il-btn-ghost" :disabled="syncing" @click="syncJadwal"><LucideRefreshCw :size="13" /> {{ syncing ? 'Sinkron...' : 'Sinkronkan Jadwal Shift' }}</button>
        </div>

        <div class="il-detail-table">
          <div class="il-detail-head">
            <span>NAMA KARYAWAN *</span><span>DIVISI</span><span>DIVISI TUJUAN (OPSIONAL)</span>
            <span>SHIFT 🔒</span><span>JAM SHIFT 🔒</span><span>JAM LEMBUR *</span><span>TOTAL JAM</span><span></span>
          </div>
          <div v-for="(row, i) in items" :key="row.key" class="il-detail-row">
            <select v-model="row.nama" class="il-input"><option value="">Ketik nama...</option><option v-for="n in namaOptions" :key="n" :value="n">{{ n }}</option></select>
            <span class="il-readonly">{{ row.divisi || '-' }}</span>
            <input v-model="row.divisiTujuan" class="il-input" placeholder="Divisi tujuan (opsional)" />
            <span class="il-readonly">{{ row.shift || '— belum simpan' }}</span>
            <span class="il-readonly">{{ row.jamKerja ? `${row.jamKerja} jam` : 'Auto' }}</span>
            <input v-model.number="row.jamLembur" type="number" min="0" class="il-input" @input="row.totalJam = Number(row.jamKerja) + Number(row.jamLembur || 0)" />
            <span class="il-readonly">{{ row.totalJam || '-' }}</span>
            <button type="button" class="il-del" @click="items.splice(i, 1)"><LucideTrash2 :size="13" /></button>
          </div>
        </div>
        <div class="il-detail-actions">
          <button type="button" class="il-btn il-btn-ghost" @click="items.push(newItem())"><LucidePlus :size="13" /> Tambah Baris</button>
          <label class="il-check"><input v-model="form.terapkanKeJadwal" type="checkbox" /> TERAPKAN TOTAL JAM (SHIFT + LEMBUR) KE JADWAL SHIFT</label>
        </div>
      </div>
    </a-modal>

    <!-- Modal: detail instruksi -->
    <a-modal v-model:open="detailOpen" :title="detailTitle" :footer="null" width="900px">
      <table class="il-table">
        <thead><tr><th>NAMA</th><th>DIVISI</th><th>DIVISI TUJUAN</th><th>SHIFT</th><th>JAM KERJA</th><th>JAM LEMBUR</th><th>TOTAL</th></tr></thead>
        <tbody>
          <tr v-for="d in detailRows" :key="d.id"><td>{{ d.nama }}</td><td>{{ d.divisi || '-' }}</td><td>{{ d.divisiTujuan || '-' }}</td><td>{{ d.shift || '-' }}</td><td>{{ d.jamKerja }}</td><td>{{ d.jamLembur }}</td><td>{{ d.totalJam }}</td></tr>
          <tr v-if="!detailRows.length"><td colspan="7" class="il-empty">Tidak ada detail.</td></tr>
        </tbody>
      </table>
    </a-modal>
  </Page>
</template>

<style scoped>
.il-page { display: flex; flex-direction: column; gap: 12px; }
.il-head { display: flex; align-items: center; justify-content: space-between; gap: 12px; }
.il-title { font-size: 16px; font-weight: 800; }
.il-tools { display: flex; gap: 8px; }
.il-input { width: 100%; height: 34px; padding: 0 9px; border: 1px solid hsl(var(--border)); border-radius: 6px; background: hsl(var(--background)); color: hsl(var(--foreground)); font-size: 12px; }
select.il-input { padding-right: 4px; }
.il-btn { display: inline-flex; align-items: center; gap: 5px; border: 0; border-radius: 6px; padding: 7px 12px; font-size: 11px; font-weight: 800; cursor: pointer; color: #fff; }
.il-btn-primary { background: #f59e0b; }
.il-btn-ghost { background: hsl(var(--muted)); color: hsl(var(--foreground)); border: 1px solid hsl(var(--border)); }
.il-btn-red { background: #dc2626; }
.il-card { border: 1px solid hsl(var(--border)); border-radius: 8px; overflow: hidden; }
.il-table { width: 100%; border-collapse: collapse; font-size: 12px; }
.il-table th { background: hsl(var(--muted)); text-align: left; padding: 9px 10px; font-size: 10px; font-weight: 800; color: hsl(var(--muted-foreground)); }
.il-table td { padding: 9px 10px; border-top: 1px solid hsl(var(--border)); }
.il-empty { text-align: center; color: hsl(var(--muted-foreground)); padding: 24px; }
.il-form { display: flex; flex-direction: column; gap: 12px; }
.il-note { padding: 10px 12px; border: 1px solid hsl(var(--border)); border-radius: 6px; background: hsl(var(--muted)); font-size: 10px; line-height: 1.5; color: hsl(var(--muted-foreground)); }
.il-grid4 { display: grid; grid-template-columns: repeat(4, minmax(0, 1fr)); gap: 12px; }
.il-f { display: flex; flex-direction: column; gap: 4px; min-width: 0; }
.il-f > span { font-size: 10px; font-weight: 800; color: hsl(var(--muted-foreground)); }
.il-f small { font-size: 9px; color: hsl(var(--muted-foreground)); line-height: 1.3; }
.il-merah { color: #dc2626 !important; }
.il-syncbar { display: flex; align-items: center; justify-content: space-between; gap: 10px; padding: 8px 10px; border: 1px solid hsl(var(--border)); border-radius: 6px; font-size: 11px; }
.il-detail-table { border: 1px solid hsl(var(--border)); border-radius: 8px; overflow: hidden; }
.il-detail-head, .il-detail-row { display: grid; grid-template-columns: 1.4fr 1fr 1.2fr 0.9fr 0.9fr 0.8fr 0.7fr 34px; gap: 7px; align-items: center; padding: 8px; }
.il-detail-head { background: hsl(var(--muted)); font-size: 9px; font-weight: 800; color: hsl(var(--muted-foreground)); }
.il-detail-row { border-top: 1px solid hsl(var(--border)); }
.il-readonly { font-size: 11px; padding: 7px 4px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.il-del { display: inline-flex; align-items: center; justify-content: center; width: 26px; height: 26px; border: 0; border-radius: 5px; background: #dc2626; color: #fff; cursor: pointer; }
.il-detail-actions { display: flex; align-items: center; gap: 16px; flex-wrap: wrap; }
.il-check { display: inline-flex; align-items: center; gap: 6px; font-size: 10px; font-weight: 700; }
@media (max-width: 1200px) { .il-grid4 { grid-template-columns: repeat(2, minmax(0, 1fr)); } }
</style>
