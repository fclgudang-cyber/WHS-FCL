<script lang="ts" setup>
/**
 * API Management — webhook penerima absensi mesin fingerprint Solution X900 (ADMS).
 * Menampilkan URL endpoint, format payload, panduan setup mesin, tombol test kirim,
 * statistik sinkronisasi, dan daftar mesin yang terdaftar.
 */
import { computed, onMounted, ref } from 'vue';

import { Page } from '@vben/common-ui';
import { LucidePlus, LucideRefreshCw, LucideTrash2 } from '@vben/icons';

import { Modal, message, notification } from 'ant-design-vue';
import dayjs from 'dayjs';

import { fclDelete, fclInsert, fclPatch, fclQuery } from '#/api/fcl';
import FclPagination from '#/components/fcl/fcl-pagination.vue';

interface Device {
  id: string;
  nama?: string;
  merk?: string;
  ipaddress?: string;
  serialnumber?: string;
  aktif?: string;
  keterangan?: string;
  createdAt?: string;
}

const savedDomain = (() => {
  try { return localStorage.getItem('fcl_webhook_domain') || ''; } catch { return ''; }
})();
const domain = ref(savedDomain || window.location.origin);
const shortUrl = computed(() => `${String(domain.value).replace(/\/$/, '')}/fp`);
const fullUrl = computed(() => `${String(domain.value).replace(/\/$/, '')}/api/fingerprint-webhook`);

function saveDomain() {
  try { localStorage.setItem('fcl_webhook_domain', domain.value.trim()); } catch { /* abaikan */ }
  message.success('Domain webhook disimpan');
}
const payloadExample = JSON.stringify({
  func: 'syncFingerprintData',
  args: [[{
    fingerprintId: '001', karyawanId: '', nama: 'Budi', divisi: 'Gudang',
    jabatan: 'Staff', jam: '08:05:00', tipe: 'IN', tanggal: dayjs().format('YYYY-MM-DD'),
  }]],
}, null, 2);

const devices = ref<Device[]>([]);
const absen = ref<any[]>([]);
const karyawanCount = ref(0);
const loading = ref(false);
const testing = ref(false);
const testResult = ref('');
const page = ref(1);
const pageSize = ref(20);
const modalOpen = ref(false);
const saving = ref(false);
const editing = ref<Device | null>(null);
const form = ref({ nama: '', merk: 'Solution X900', ipaddress: '', serialnumber: '', aktif: 'Ya', keterangan: '' });

const fingerprintAbsen = computed(() => absen.value.filter((a) => String(a.sumber).toLowerCase() === 'fingerprint'));
const paged = computed(() => devices.value.slice((page.value - 1) * pageSize.value, page.value * pageSize.value));

async function load(silent = false) {
  if (!silent) loading.value = true;
  try {
    const [d, a, k] = await Promise.all([
      fclQuery<Device>('fcl_api_devices', { select: '*', order: 'createdAt.desc', limit: 1000 }),
      fclQuery<any>('absensikaryawan', { select: '*', order: 'createdAt.desc', limit: 5000 }),
      fclQuery<any>('karyawan', { select: 'id', limit: 5000 }),
    ]);
    devices.value = d;
    absen.value = a;
    karyawanCount.value = k.length;
  } catch (err: any) {
    console.error('[fcl/api-management]', err);
    notification.error({ message: 'API Management', description: err?.message || 'Gagal memuat data' });
  } finally {
    if (!silent) loading.value = false;
  }
}

async function copyUrl() {
  try { await navigator.clipboard.writeText(shortUrl.value); notification.success({ message: 'URL pendek disalin' }); }
  catch { message.info(shortUrl.value); }
}

async function testWebhook() {
  testing.value = true;
  testResult.value = '';
  try {
    const res = await fetch('/fp', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: payloadExample,
    });
    const body = await res.json();
    testResult.value = JSON.stringify(body);
    if (body?.success) notification.success({ message: 'Test Webhook', description: `Terkirim ${body.total} data · ditambah ${body.added} · duplikat ${body.skipped}` });
    else notification.error({ message: 'Test Webhook', description: body?.message || 'Gagal' });
    await load(true);
  } catch (err: any) {
    testResult.value = String(err?.message || err);
    notification.error({ message: 'Test Webhook', description: 'Gagal menghubungi endpoint' });
  } finally {
    testing.value = false;
  }
}

function openModal(d?: Device | null) {
  editing.value = d ?? null;
  form.value = d
    ? { nama: d.nama ?? '', merk: d.merk ?? 'Solution X900', ipaddress: d.ipaddress ?? '', serialnumber: d.serialnumber ?? '', aktif: d.aktif ?? 'Ya', keterangan: d.keterangan ?? '' }
    : { nama: '', merk: 'Solution X900', ipaddress: '', serialnumber: '', aktif: 'Ya', keterangan: '' };
  modalOpen.value = true;
}

async function save() {
  if (!form.value.nama.trim()) { message.warning('Nama mesin wajib diisi'); return; }
  saving.value = true;
  const payload = { ...form.value };
  try {
    if (editing.value) await fclPatch('fcl_api_devices', editing.value.id, payload);
    else await fclInsert('fcl_api_devices', [{ ...payload, createdAt: dayjs().format('YYYY-MM-DD HH:mm:ss') }]);
    notification.success({ message: 'Mesin', description: editing.value ? 'Diperbarui' : 'Ditambahkan' });
    modalOpen.value = false;
    await load(true);
  } catch (err: any) {
    notification.error({ message: 'Mesin', description: err?.message || 'Gagal menyimpan' });
  } finally {
    saving.value = false;
  }
}

function remove(d: Device) {
  Modal.confirm({
    title: `Hapus mesin ${d.nama}?`, content: 'Data mesin akan dihapus (catatan absensi tetap tersimpan).',
    okText: 'Hapus', okType: 'danger',
    onOk: async () => { await fclDelete('fcl_api_devices', d.id); await load(true); },
  });
}

onMounted(() => load());
</script>

<template>
  <Page auto-content-height>
    <div class="api-root">
      <!-- Webhook info -->
      <div class="api-card api-hero">
        <div class="api-title">Webhook Fingerprint Solution X900 (ADMS)</div>
        <div class="api-sub">Terima absensi dari mesin fingerprint dan simpan langsung ke <b>Absensi Karyawan</b> (berdasarkan fingerprint ID / karyawanId).</div>
        <div class="api-domain-row">
          <label>Domain / alamat server</label>
          <input v-model="domain" class="api-input" placeholder="http://192.168.3.100:8082" @change="saveDomain" />
        </div>
        <div class="api-url-row">
          <code class="api-url api-url-short">{{ shortUrl }}</code>
          <button class="api-btn api-btn-primary" @click="copyUrl">Copy URL</button>
        </div>
        <div class="api-full-url">URL lengkap (alternatif): <code>{{ fullUrl }}</code></div>
        <div class="api-grid">
          <div class="api-box">
            <b>Payload yang diterima</b>
            <pre class="api-code">{{ payloadExample }}</pre>
            <div class="api-muted">Format array langsung dan <code>{"records":[...]}</code> juga didukung.</div>
          </div>
          <div class="api-box">
            <b>Cara setting di mesin X900</b>
            <ol class="api-steps">
              <li>Buka browser, ketik IP mesin X900, login sebagai admin.</li>
              <li>Masuk menu <b>Communication</b> / <b>HTTP Push</b> / <b>ADMS</b>.</li>
              <li>Isi URL server dengan <code>{{ shortUrl }}</code>, method <b>POST</b>.</li>
              <li>Pilih format JSON, aktifkan push, lalu simpan.</li>
              <li>Klik <b>Test Kirim Data</b> di bawah untuk memastikan data masuk.</li>
            </ol>
            <div class="api-test-row">
              <button class="api-btn api-btn-accent" :loading="testing" @click="testWebhook">Test Kirim Data</button>
              <button class="api-btn api-btn-ghost" @click="load()"><LucideRefreshCw :size="12" /> Refresh</button>
            </div>
            <div v-if="testResult" class="api-result">{{ testResult }}</div>
          </div>
        </div>
      </div>

      <!-- Statistik -->
      <div class="api-stats">
        <div class="api-stat"><span>DATA ABSEN FINGERPRINT</span><b>{{ fingerprintAbsen.length }}</b></div>
        <div class="api-stat"><span>TOTAL KARYAWAN (MASTER)</span><b>{{ karyawanCount }}</b></div>
        <div class="api-stat"><span>MESIN TERDAFTAR</span><b>{{ devices.length }}</b></div>
        <div class="api-stat"><span>STATUS WEBHOOK</span><b class="api-ok">Aktif</b></div>
      </div>

      <!-- Daftar mesin -->
      <div class="api-card">
        <div class="api-head">
          <div class="api-title">Daftar Mesin Fingerprint</div>
          <button class="api-btn api-btn-primary" @click="openModal()"><LucidePlus :size="12" /> Tambah Mesin</button>
        </div>
        <div class="api-table-wrap">
          <table class="api-table">
            <thead><tr><th>NAMA MESIN</th><th>MERK</th><th>IP ADDRESS</th><th>SERIAL NUMBER</th><th>AKTIF</th><th>KETERANGAN</th><th>AKSI</th></tr></thead>
            <tbody>
              <tr v-for="d in paged" :key="d.id">
                <td class="api-name">{{ d.nama }}</td><td>{{ d.merk || '-' }}</td><td><code>{{ d.ipaddress || '-' }}</code></td>
                <td><code>{{ d.serialnumber || '-' }}</code></td>
                <td><span class="api-status" :class="String(d.aktif).toLowerCase() === 'ya' ? 'api-ok' : 'api-off'">{{ d.aktif }}</span></td>
                <td>{{ d.keterangan || '-' }}</td>
                <td>
                  <div class="api-row-actions">
                    <button class="api-btn api-btn-ghost api-btn-xs" @click="openModal(d)">Edit</button>
                    <button class="api-btn api-btn-danger api-btn-xs" @click="remove(d)"><LucideTrash2 :size="11" /> Hapus</button>
                  </div>
                </td>
              </tr>
              <tr v-if="!paged.length"><td colspan="7" class="api-empty">Belum ada mesin terdaftar.</td></tr>
            </tbody>
          </table>
        </div>
        <FclPagination v-model:page="page" v-model:page-size="pageSize" :total="devices.length" />
      </div>
    </div>

    <a-modal v-model:open="modalOpen" :title="editing ? 'Edit Mesin' : 'Tambah Mesin Fingerprint'" :confirm-loading="saving" ok-text="Simpan" cancel-text="Batal" width="480px" @ok="save">
      <div class="api-form">
        <div class="api-field"><label>Nama Mesin *</label><input v-model="form.nama" class="api-input" placeholder="Contoh: Fingerprint Gate 1" /></div>
        <div class="api-form-row">
          <div class="api-field"><label>Merk</label><input v-model="form.merk" class="api-input" /></div>
          <div class="api-field"><label>IP Address</label><input v-model="form.ipaddress" class="api-input" placeholder="192.168.1.201" /></div>
        </div>
        <div class="api-form-row">
          <div class="api-field"><label>Serial Number</label><input v-model="form.serialnumber" class="api-input" /></div>
          <div class="api-field"><label>Aktif</label><select v-model="form.aktif" class="api-input"><option>Ya</option><option>Tidak</option></select></div>
        </div>
        <div class="api-field"><label>Keterangan</label><input v-model="form.keterangan" class="api-input" /></div>
      </div>
    </a-modal>
  </Page>
</template>

<style scoped>
.api-root { display: flex; flex-direction: column; gap: 10px; }
.api-card { background: hsl(var(--card)); border: 1px solid hsl(var(--border)); border-radius: 8px; padding: 14px; }
.api-title { font-size: 14px; font-weight: 800; color: hsl(var(--foreground)); }
.api-sub { font-size: 11px; color: hsl(var(--muted-foreground)); margin: 4px 0 10px; }
.api-url-row { display: flex; gap: 6px; align-items: center; flex-wrap: wrap; margin-bottom: 12px; }
.api-url { display: inline-block; padding: 7px 12px; border-radius: 6px; background: hsl(var(--muted)); color: hsl(var(--primary)); font-size: 12px; word-break: break-all; }
.api-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 12px; }
.api-box { background: hsl(var(--muted)); border: 1px solid hsl(var(--border)); border-radius: 6px; padding: 10px; font-size: 11px; color: hsl(var(--foreground)); }
.api-code { background: hsl(var(--background)); border: 1px solid hsl(var(--border)); border-radius: 6px; padding: 10px; font-size: 10px; overflow: auto; max-height: 220px; color: hsl(var(--foreground)); }
.api-muted { font-size: 10px; color: hsl(var(--muted-foreground)); margin-top: 6px; }
.api-steps { margin: 8px 0 8px 18px; line-height: 1.7; }
.api-steps code { color: hsl(var(--primary)); }
.api-test-row { display: flex; gap: 6px; }
.api-result { margin-top: 8px; font-family: monospace; font-size: 10px; background: hsl(var(--background)); border-radius: 5px; padding: 8px; word-break: break-all; }
.api-btn { display: inline-flex; align-items: center; gap: 5px; border: 1px solid hsl(var(--border)); background: hsl(var(--muted)); color: hsl(var(--foreground)); font-size: 11px; font-weight: 700; padding: 6px 11px; border-radius: 6px; cursor: pointer; height: 30px; }
.api-btn:hover { filter: brightness(0.96); }
.api-btn-xs { height: 24px; padding: 2px 8px; font-size: 10px; }
.api-btn-primary { background: hsl(var(--primary)); border-color: hsl(var(--primary)); color: hsl(var(--primary-foreground)); }
.api-btn-accent { background: hsl(var(--warning)); border-color: hsl(var(--warning)); color: hsl(var(--warning-foreground)); }
.api-btn-danger { background: hsl(var(--destructive)); border-color: hsl(var(--destructive)); color: hsl(var(--destructive-foreground)); }
.api-stats { display: grid; grid-template-columns: repeat(4, minmax(0, 1fr)); gap: 8px; }
.api-stat { background: hsl(var(--card)); border: 1px solid hsl(var(--border)); border-radius: 8px; padding: 10px 12px; display: flex; flex-direction: column; gap: 4px; }
.api-stat span { font-size: 10px; font-weight: 800; color: hsl(var(--muted-foreground)); }
.api-stat b { font-size: 20px; color: hsl(var(--foreground)); }
.api-ok { color: hsl(var(--success)) !important; }
.api-off { color: hsl(var(--muted-foreground)) !important; }
.api-head { display: flex; justify-content: space-between; align-items: center; gap: 10px; margin-bottom: 10px; }
.api-table-wrap { overflow-x: auto; }
.api-table { width: 100%; min-width: 900px; border-collapse: collapse; font-size: 11px; }
.api-table th { background: hsl(var(--muted)); color: hsl(var(--muted-foreground)); font-size: 10px; font-weight: 800; text-align: left; padding: 8px 10px; border-bottom: 1px solid hsl(var(--border)); }
.api-table td { padding: 8px 10px; border-bottom: 1px solid hsl(var(--border)); color: hsl(var(--foreground)); }
.api-name { font-weight: 800; }
.api-status { font-size: 10px; font-weight: 800; padding: 3px 9px; border-radius: 999px; background: hsl(var(--muted)); }
.api-row-actions { display: flex; gap: 4px; }
.api-empty { text-align: center; color: hsl(var(--muted-foreground)); padding: 20px 8px; }
.api-form { display: flex; flex-direction: column; gap: 10px; padding-top: 4px; }
.api-form-row { display: grid; grid-template-columns: 1fr 1fr; gap: 10px; }
.api-field label { display: block; font-size: 11px; font-weight: 700; color: hsl(var(--muted-foreground)); margin-bottom: 4px; }
.api-input { width: 100%; height: 32px; padding: 4px 9px; border-radius: 6px; border: 1px solid hsl(var(--border)); background: hsl(var(--background)); color: hsl(var(--foreground)); font-size: 12px; outline: none; }
.api-input:focus { border-color: hsl(var(--primary)); }
@media (max-width: 1100px) { .api-grid { grid-template-columns: 1fr; } .api-stats { grid-template-columns: repeat(2, 1fr); } }

.api-domain-row { display: flex; align-items: center; gap: 8px; margin-bottom: 8px; }
.api-domain-row label { font-size: 10px; font-weight: 800; color: hsl(var(--muted-foreground)); white-space: nowrap; }
.api-domain-row .api-input { max-width: 320px; }
.api-url-short { font-size: 15px; font-weight: 800; }
.api-full-url { font-size: 10px; color: hsl(var(--muted-foreground)); margin: 0 0 12px; }
</style>
