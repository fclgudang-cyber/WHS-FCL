<script lang="ts" setup>
/**
 * API Management — webhook penerima absensi mesin fingerprint Solution X900 (ADMS).
 * Menampilkan URL endpoint, format payload, panduan setup mesin, tombol test kirim,
 * statistik sinkronisasi, dan daftar mesin yang terdaftar.
 */
import { computed, onMounted, ref } from 'vue';

import { Page } from '@vben/common-ui';
import { LucidePlus, LucideTrash2 } from '@vben/icons';

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

function saveDomain() {
  try { localStorage.setItem('fcl_webhook_domain', domain.value.trim()); } catch { /* abaikan */ }
  message.success('Domain webhook disimpan');
}
const webhookBase = computed(() => domain.value.trim().replace(/\/+$/, ''));
const admsUrl = computed(() => `${webhookBase.value}/iclock/cdata?SN=<SERIAL_NUMBER>`);
const jsonUrl = computed(() => `${webhookBase.value}/fp`);
const apiUrl = computed(() => `${webhookBase.value}/api/fingerprint-webhook`);
const port80Url = computed(() => {
  try {
    const u = new URL(webhookBase.value);
    return `${u.protocol}//${u.hostname}:80/fp`;
  } catch {
    return 'http://<IP-SERVER>:80/fp';
  }
});

const payloadExample = `{
  "records": [
    {
      "fingerprintId": "FP-001",
      "nama": "Ahmad Fauzi",
      "tanggal": "2026-11-21",
      "jam": "08:00:00",
      "tipe": "IN"
    }
  ]
}`;

const steps = [
  { title: 'Set domain / IP server', desc: 'Isi alamat server yang bisa dijangkau mesin, lalu klik Simpan. Contoh: http://192.168.1.10' },
  { title: 'Buka port 80', desc: 'Webhook & ADMS berjalan di port 80. Pastikan firewall/server mengizinkan TCP 80 dari jaringan mesin fingerprint.' },
  { title: 'Setting mesin Solution X900', desc: 'Masuk menu Network/ADMS → isi Server URL/IP, Port 80, endpoint /iclock/cdata, lalu aktifkan ADMS. Serial number (SN) akan otomatis terdaftar di tabel mesin.' },
  { title: 'Kirim data absensi', desc: 'Mesin mengirim GET/POST ke /iclock/cdata?SN=<serial>. Untuk integrasi manual/JSON, POST ke /fp atau /api/fingerprint-webhook dengan format di bawah.' },
  { title: 'Cek status & sinyal', desc: 'Klik Cek Sinyal untuk memastikan mesin online. Data absensi akan masuk ke Daftar Karyawan → Absensi Karyawan.' },
];

async function copyText(text: string, label = 'Teks') {
  try {
    await navigator.clipboard.writeText(text);
    message.success(`${label} disalin`);
  } catch {
    window.prompt(`Salin ${label}:`, text);
  }
}

async function testWebhook() {
  try {
    const res = await fetch('/fp', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ records: [{ fingerprintId: 'TEST', nama: 'Tes Webhook', tanggal: dayjs().format('YYYY-MM-DD'), jam: dayjs().format('HH:mm:ss'), tipe: 'IN' }] }),
    });
    const body = await res.json().catch(() => ({}));
    if (!res.ok) throw new Error(body?.message || `HTTP ${res.status}`);
    notification.success({ message: 'Tes Webhook', description: body?.message || 'Payload berhasil diterima' });
    await load(true);
  } catch (err: any) {
    notification.error({ message: 'Tes Webhook gagal', description: err?.message });
  }
}

const devices = ref<Device[]>([]);
interface Signal { online: boolean; latencyMs: number; port: number }
const signals = ref<Record<string, Signal>>({});
const checking = ref(false);
const loading = ref(false);
const page = ref(1);
const pageSize = ref(20);
const modalOpen = ref(false);
const saving = ref(false);
const editing = ref<Device | null>(null);
const form = ref({ nama: '', merk: 'Solution X900', ipaddress: '', serialnumber: '', aktif: 'Ya', keterangan: '' });

const paged = computed(() => devices.value.slice((page.value - 1) * pageSize.value, page.value * pageSize.value));

function signalOf(ip?: string): Signal | undefined {
  return ip ? signals.value[ip] : undefined;
}

function barLevel(s?: Signal): number {
  if (!s) return 0;
  if (!s.online) return 0;
  if (s.latencyMs <= 50) return 4;
  if (s.latencyMs <= 150) return 3;
  if (s.latencyMs <= 400) return 2;
  return 1;
}

async function checkSignals() {
  const ips = devices.value.filter((d) => String(d.aktif).toLowerCase() === 'ya' && d.ipaddress).map((d) => String(d.ipaddress));
  if (!ips.length) { message.info('Belum ada mesin aktif dengan IP address'); return; }
  checking.value = true;
  try {
    const res = await fetch('/api/fingerprint-check', {
      method: 'POST', headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ ips }),
    });
    const body = await res.json();
    const map: Record<string, Signal> = {};
    (body?.items ?? []).forEach((it: any) => { map[String(it.ip)] = { online: !!it.online, latencyMs: Number(it.latencyMs || 0), port: Number(it.port || 0) }; });
    signals.value = map;
    const online = Object.values(map).filter((v) => v.online).length;
    notification.success({ message: 'Cek Sinyal', description: `${online}/${ips.length} mesin terhubung` });
  } catch (err: any) {
    notification.error({ message: 'Cek Sinyal', description: err?.message || 'Gagal mengecek koneksi' });
  } finally {
    checking.value = false;
  }
}

async function load(silent = false) {
  if (!silent) loading.value = true;
  try {
    devices.value = await fclQuery<Device>('fcl_api_devices', { select: '*', order: 'createdAt.desc', limit: 1000 });
    if (!silent && devices.value.length) await checkSignals();
  } catch (err: any) {
    console.error('[fcl/api-management]', err);
    notification.error({ message: 'API Management', description: err?.message || 'Gagal memuat data' });
  } finally {
    if (!silent) loading.value = false;
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
    await checkSignals();
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
      <!-- Webhook + panduan penggunaan -->
      <div class="api-card api-hero">
        <div class="api-hero-head">
          <div>
            <div class="api-title">Webhook Fingerprint Solution X900 (ADMS)</div>
            <div class="api-subtitle">Terima absensi mesin langsung ke Master Karyawan · berjalan pada <b>port 80</b></div>
          </div>
          <span class="api-port-badge">PORT 80</span>
        </div>

        <div class="api-domain-row">
          <label>Domain / alamat server</label>
          <input v-model="domain" class="api-input" placeholder="http://192.168.3.100" @change="saveDomain" />
          <button class="api-btn api-btn-ghost" @click="saveDomain">Simpan</button>
          <button class="api-btn api-btn-accent" :loading="checking" @click="testWebhook">Tes Webhook</button>
        </div>

        <div class="api-url-grid">
          <div class="api-url-item">
            <span>ADMS / auto-register mesin (GET/POST)</span>
            <div><code>{{ admsUrl }}</code><button class="api-copy" @click="copyText(admsUrl, 'URL ADMS')">Salin</button></div>
          </div>
          <div class="api-url-item">
            <span>Webhook JSON (POST) — direkomendasikan</span>
            <div><code>{{ jsonUrl }}</code><button class="api-copy" @click="copyText(jsonUrl, 'URL webhook')">Salin</button></div>
          </div>
          <div class="api-url-item">
            <span>Endpoint alternatif</span>
            <div><code>{{ apiUrl }}</code><button class="api-copy" @click="copyText(apiUrl, 'Endpoint')">Salin</button></div>
          </div>
          <div class="api-url-item">
            <span>Akses langsung via port 80</span>
            <div><code>{{ port80Url }}</code><button class="api-copy" @click="copyText(port80Url, 'URL port 80')">Salin</button></div>
          </div>
        </div>
      </div>

      <div class="api-card api-guide">
        <div class="api-title">Tata Cara Penggunaan</div>
        <div class="api-steps">
          <div v-for="(step, i) in steps" :key="i" class="api-step">
            <span class="api-step-no">{{ i + 1 }}</span>
            <div><b>{{ step.title }}</b><p>{{ step.desc }}</p></div>
          </div>
        </div>
        <div class="api-payload-head">
          <b>Format payload JSON untuk <code>/fp</code></b>
          <button class="api-copy" @click="copyText(payloadExample, 'Contoh payload')">Salin</button>
        </div>
        <pre class="api-code">{{ payloadExample }}</pre>
      </div>

      <!-- Daftar mesin -->
      <div class="api-card">
        <div class="api-head">
          <div class="api-title">Daftar Mesin Fingerprint</div>
          <button class="api-btn api-btn-ghost" :loading="checking" @click="checkSignals">Cek Sinyal</button>
          <button class="api-btn api-btn-primary" @click="openModal()"><LucidePlus :size="12" /> Tambah Mesin</button>
        </div>
        <div class="api-table-wrap">
          <table class="api-table">
            <thead><tr><th>NAMA MESIN</th><th>MERK</th><th>IP ADDRESS</th><th>SERIAL NUMBER</th><th>SINYAL</th><th>AKTIF</th><th>KETERANGAN</th><th>AKSI</th></tr></thead>
            <tbody>
              <tr v-for="d in paged" :key="d.id">
                <td class="api-name">{{ d.nama }}</td><td>{{ d.merk || '-' }}</td><td><code>{{ d.ipaddress || '-' }}</code></td>
                <td><code>{{ d.serialnumber || '-' }}</code></td>
                <td>
                  <span class="api-signal" :title="signalOf(d.ipaddress) ? (signalOf(d.ipaddress)?.online ? `Terhubung · ${signalOf(d.ipaddress)?.latencyMs} ms (port ${signalOf(d.ipaddress)?.port})` : 'Tidak terhubung') : 'Belum dicek'">
                    <i v-for="n in 4" :key="n" :class="{ 'on': n <= barLevel(signalOf(d.ipaddress)), 'off': n > barLevel(signalOf(d.ipaddress)) }" :style="{ height: 4 + n * 3 + 'px' }"></i>
                  </span>
                  <span class="api-signal-text" :class="signalOf(d.ipaddress)?.online ? 'api-ok' : 'api-off'">{{ signalOf(d.ipaddress) ? (signalOf(d.ipaddress)?.online ? signalOf(d.ipaddress)?.latencyMs + ' ms' : 'Offline') : '-' }}</span>
                </td>
                <td><span class="api-status" :class="String(d.aktif).toLowerCase() === 'ya' ? 'api-ok' : 'api-off'">{{ d.aktif }}</span></td>
                <td>{{ d.keterangan || '-' }}</td>
                <td>
                  <div class="api-row-actions">
                    <button class="api-btn api-btn-ghost api-btn-xs" @click="openModal(d)">Edit</button>
                    <button class="api-btn api-btn-danger api-btn-xs" @click="remove(d)"><LucideTrash2 :size="11" /> Hapus</button>
                  </div>
                </td>
              </tr>
              <tr v-if="!paged.length"><td colspan="8" class="api-empty">Belum ada mesin terdaftar.</td></tr>
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
.api-btn { display: inline-flex; align-items: center; gap: 5px; border: 1px solid hsl(var(--border)); background: hsl(var(--muted)); color: hsl(var(--foreground)); font-size: 11px; font-weight: 700; padding: 6px 11px; border-radius: 6px; cursor: pointer; height: 30px; }
.api-btn:hover { filter: brightness(0.96); }
.api-btn-xs { height: 24px; padding: 2px 8px; font-size: 10px; }
.api-btn-primary { background: hsl(var(--primary)); border-color: hsl(var(--primary)); color: hsl(var(--primary-foreground)); }
.api-btn-accent { background: hsl(var(--warning)); border-color: hsl(var(--warning)); color: hsl(var(--warning-foreground)); }
.api-btn-danger { background: hsl(var(--destructive)); border-color: hsl(var(--destructive)); color: hsl(var(--destructive-foreground)); }
.api-head { display: flex; justify-content: space-between; align-items: center; gap: 10px; margin-bottom: 10px; }
.api-table-wrap { overflow-x: auto; }
.api-table { width: 100%; min-width: 900px; border-collapse: collapse; font-size: 11px; }
.api-table th { background: hsl(var(--muted)); color: hsl(var(--muted-foreground)); font-size: 10px; font-weight: 800; text-align: left; padding: 8px 10px; border-bottom: 1px solid hsl(var(--border)); }
.api-table td { padding: 8px 10px; border-bottom: 1px solid hsl(var(--border)); color: hsl(var(--foreground)); }
.api-name { font-weight: 800; }
.api-status { font-size: 10px; font-weight: 800; padding: 3px 9px; border-radius: 999px; background: hsl(var(--muted)); }
.api-ok { color: hsl(var(--success)) !important; }
.api-off { color: hsl(var(--muted-foreground)) !important; }
.api-row-actions { display: flex; gap: 4px; }
.api-empty { text-align: center; color: hsl(var(--muted-foreground)); padding: 20px 8px; }
.api-form { display: flex; flex-direction: column; gap: 10px; padding-top: 4px; }
.api-form-row { display: grid; grid-template-columns: 1fr 1fr; gap: 10px; }
.api-field label { display: block; font-size: 11px; font-weight: 700; color: hsl(var(--muted-foreground)); margin-bottom: 4px; }
.api-input { width: 100%; height: 32px; padding: 4px 9px; border-radius: 6px; border: 1px solid hsl(var(--border)); background: hsl(var(--background)); color: hsl(var(--foreground)); font-size: 12px; outline: none; }
.api-input:focus { border-color: hsl(var(--primary)); }

.api-domain-row { display: flex; align-items: center; gap: 8px; margin-bottom: 8px; }
.api-domain-row label { font-size: 10px; font-weight: 800; color: hsl(var(--muted-foreground)); white-space: nowrap; }
.api-domain-row .api-input { max-width: 360px; }

.api-signal { display: inline-flex; align-items: flex-end; gap: 2px; vertical-align: middle; }
.api-signal i { display: inline-block; width: 4px; border-radius: 2px; background: hsl(var(--border)); }
.api-signal i.on { background: hsl(var(--success)); }
.api-signal i.off { opacity: 0.35; }
.api-signal-text { font-size: 10px; font-weight: 800; margin-left: 6px; white-space: nowrap; }

.api-hero-head { display: flex; justify-content: space-between; align-items: flex-start; gap: 10px; margin-bottom: 10px; }
.api-subtitle { font-size: 11px; color: hsl(var(--muted-foreground)); margin-top: 3px; }
.api-port-badge { flex-shrink: 0; background: hsl(var(--success) / 0.14); color: hsl(var(--success)); border: 1px solid hsl(var(--success) / 0.4); border-radius: 999px; padding: 4px 10px; font-size: 10px; font-weight: 800; }
.api-url-grid { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 8px; }
.api-url-item { background: hsl(var(--muted) / 0.55); border: 1px solid hsl(var(--border)); border-radius: 8px; padding: 8px 10px; min-width: 0; }
.api-url-item > span { display: block; font-size: 10px; font-weight: 700; color: hsl(var(--muted-foreground)); }
.api-url-item > div { display: flex; align-items: center; gap: 6px; margin-top: 5px; }
.api-url-item code { flex: 1; min-width: 0; overflow-x: auto; white-space: nowrap; font-size: 11px; color: hsl(var(--foreground)); }
.api-copy { flex-shrink: 0; border: 1px solid hsl(var(--border)); background: hsl(var(--card)); color: hsl(var(--primary)); border-radius: 5px; padding: 2px 7px; font-size: 10px; font-weight: 800; cursor: pointer; }
.api-steps { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 10px; margin: 10px 0 12px; }
.api-step { display: flex; gap: 9px; align-items: flex-start; }
.api-step-no { flex-shrink: 0; width: 22px; height: 22px; border-radius: 50%; background: hsl(var(--primary) / 0.12); color: hsl(var(--primary)); display: grid; place-items: center; font-size: 11px; font-weight: 800; }
.api-step b { font-size: 11px; }
.api-step p { margin: 2px 0 0; font-size: 10px; line-height: 1.5; color: hsl(var(--muted-foreground)); }
.api-payload-head { display: flex; justify-content: space-between; align-items: center; gap: 8px; font-size: 11px; margin-bottom: 5px; }
.api-code { background: hsl(var(--muted) / 0.65); border: 1px solid hsl(var(--border)); border-radius: 8px; padding: 10px; font-size: 11px; line-height: 1.5; overflow-x: auto; margin: 0; }

@media (max-width: 900px) {
  .api-hero-head, .api-domain-row { flex-wrap: wrap; }
  .api-domain-row .api-input { max-width: none; flex: 1 1 100%; }
  .api-url-grid, .api-steps { grid-template-columns: 1fr; }
  .api-head { flex-wrap: wrap; }
  .api-table { min-width: 760px; }
}
</style>
