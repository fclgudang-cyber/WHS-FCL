<script lang="ts" setup>
/**
 * Kelola Akun FCL — buat akun login, import massal (CSV/XLSX), template,
 * reset password, ubah status, hapus. Password di-hash bcrypt di backend.
 */
import { computed, onMounted, reactive, ref } from 'vue';

import { Page } from '@vben/common-ui';
import { LucideDownload, LucidePlus, LucideTrash2 } from '@vben/icons';

import { message, Modal, notification } from 'ant-design-vue';
import * as XLSX from 'xlsx';

import { fclCount, fclDelete, fclPatch, fclQuery } from '#/api/fcl';
import { downloadCsv } from '#/utils/csv';

interface Akun {
  id: string;
  username?: string;
  nama?: string;
  role?: string;
  divisi?: string;
  email?: string;
  status?: string;
  createdAt?: string;
  [key: string]: any;
}

const loading = ref(true);
const rows = ref<Akun[]>([]);
const search = ref('');
const page = ref(1);
const pageSize = ref(15);

const filtered = computed(() => {
  const q = search.value.trim().toLowerCase();
  if (!q) return rows.value;
  return rows.value.filter((r) =>
    [r.username, r.nama, r.role, r.divisi, r.email, r.status]
      .map((v) => String(v ?? '').toLowerCase())
      .some((v) => v.includes(q)),
  );
});
const paged = computed(() =>
  filtered.value.slice((page.value - 1) * pageSize.value, page.value * pageSize.value),
);

async function load(silent = false) {
  if (!silent) loading.value = true;
  try {
    rows.value = await fclQuery<Akun>('users', { select: '*', order: 'createdAt.desc', limit: 10000 });
  } catch (e: any) {
    notification.error({ message: 'Kelola Akun', description: e?.message || 'Gagal memuat akun' });
  } finally {
    loading.value = false;
  }
}

// ---------------------------------------------------------------- create
const createOpen = ref(false);
const creating = ref(false);
const form = reactive({
  username: '', password: '', nama: '', role: 'OPERATOR', divisi: 'WAREHOUSE',
  email: '', status: 'AKTIF',
});

function openCreate() {
  Object.assign(form, {
    username: '', password: '', nama: '', role: 'OPERATOR', divisi: 'WAREHOUSE',
    email: '', status: 'AKTIF',
  });
  createOpen.value = true;
}

async function submitCreate() {
  if (!form.username.trim() || !form.password.trim()) {
    message.warning('Username dan password wajib diisi');
    return;
  }
  creating.value = true;
  try {
    const res = await postJson<{ success: boolean; message?: string }>('/api/auth/account', form);
    if (!res?.success) throw new Error(res?.message || 'Gagal membuat akun');
    notification.success({ message: 'Akun dibuat', description: `${form.username} dapat langsung login` });
    createOpen.value = false;
    await load(true);
  } catch (e: any) {
    notification.error({ message: 'Gagal membuat akun', description: e?.message });
  } finally {
    creating.value = false;
  }
}

// ---------------------------------------------------------------- status / password / hapus
async function setStatus(row: Akun, status: string) {
  try {
    await fclPatch('users', row.id, { status, updatedAt: new Date().toISOString() });
    notification.success({ message: 'Status diperbarui', description: `${row.username} → ${status}` });
    await load(true);
  } catch (e: any) { notification.error({ message: 'Gagal', description: e?.message }); }
}

const resetOpen = ref(false);
const resetTarget = ref<Akun | null>(null);
const resetPassword = ref('');
function openReset(row: Akun) {
  resetTarget.value = row;
  resetPassword.value = '';
  resetOpen.value = true;
}
async function submitReset() {
  if (!resetTarget.value || resetPassword.value.length < 6) {
    message.warning('Password minimal 6 karakter');
    return;
  }
  try {
    const res = await postJson<{ success: boolean; message?: string }>(
      `/api/auth/account/${encodeURIComponent(String(resetTarget.value.id))}/reset-password`,
      { password: resetPassword.value },
    );
    if (!res?.success) throw new Error(res?.message || 'Gagal reset password');
    notification.success({ message: 'Password direset', description: resetTarget.value.username });
    resetOpen.value = false;
  } catch (e: any) { notification.error({ message: 'Gagal reset password', description: e?.message }); }
}

async function remove(row: Akun) {
  try {
    await fclDelete('users', row.id);
    notification.success({ message: 'Akun dihapus', description: row.username });
    await load(true);
  } catch (e: any) { notification.error({ message: 'Gagal menghapus', description: e?.message }); }
}

// ---------------------------------------------------------------- template & import
const TEMPLATE_COLUMNS = [
  { title: 'username', key: 'username' },
  { title: 'password', key: 'password' },
  { title: 'nama', key: 'nama' },
  { title: 'role', key: 'role' },
  { title: 'divisi', key: 'divisi' },
  { title: 'email', key: 'email' },
  { title: 'status', key: 'status' },
];

function downloadTemplate() {
  downloadCsv('template-akun-fcl', TEMPLATE_COLUMNS, [{
    username: 'operator1',
    password: 'rahasia123',
    nama: 'Operator Gudang',
    role: 'OPERATOR',
    divisi: 'WAREHOUSE',
    email: 'operator@fcl.local',
    status: 'AKTIF',
  }]);
  message.success('Template berhasil diunduh');
}

const importOpen = ref(false);
const importing = ref(false);
const importRows = ref<Record<string, any>[]>([]);
const importFile = ref('');

async function pickFile(e: Event) {
  const input = e.target as HTMLInputElement;
  const file = input.files?.[0];
  if (!file) return;
  importFile.value = file.name;
  try {
    const isCsv = /\.csv$/i.test(file.name);
    const wb = isCsv
      ? XLSX.read(await file.text(), { type: 'string' })
      : XLSX.read(await file.arrayBuffer(), { type: 'array' });
    const sheetName = wb.SheetNames[0];
    if (!sheetName) throw new Error('Workbook tidak punya sheet');
    const ws = wb.Sheets[sheetName];
    if (!ws) throw new Error('Sheet tidak ditemukan');
    const matrix = XLSX.utils.sheet_to_json<string[]>(ws, { header: 1, defval: '' });
    if (!matrix.length) throw new Error('File kosong');
    const headers = (matrix[0] ?? []).map((h) => String(h ?? '').trim().toLowerCase());
    importRows.value = matrix.slice(1)
      .filter((r) => Array.isArray(r) && r.some((v) => String(v ?? '').trim()))
      .map((r) => {
        const obj: Record<string, any> = {};
        headers.forEach((h, i) => { if (h) obj[h] = String(r[i] ?? '').trim(); });
        return obj;
      });
    message.success(`${importRows.value.length} baris terbaca`);
  } catch (err: any) {
    importRows.value = [];
    notification.error({ message: 'Gagal membaca file', description: err?.message });
  }
}

async function submitImport() {
  if (!importRows.value.length) { message.warning('Tidak ada baris untuk diimpor'); return; }
  importing.value = true;
  try {
    const res = await postJson<{
      success: boolean; created: number; skipped: number;
      errors?: Array<{ row: number; message: string }>;
    }>('/api/auth/accounts/import', importRows.value);
    notification.success({
      message: 'Import selesai',
      description: `${res.created} akun dibuat, ${res.skipped} dilewati (username sudah ada)`,
    });
    if (res.errors?.length) {
      Modal.warning({
        title: `${res.errors.length} baris gagal`,
        content: res.errors.slice(0, 10).map((e) => `Baris ${e.row}: ${e.message}`).join('\n'),
        width: 480,
      });
    }
    importOpen.value = false;
    await load(true);
  } catch (e: any) { notification.error({ message: 'Gagal import', description: e?.message }); }
  finally { importing.value = false; }
}

async function postJson<T>(path: string, body: any): Promise<T> {
  const res = await fetch(path, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  });
  const text = await res.text();
  let data: any = null;
  try { data = text ? JSON.parse(text) : null; } catch { data = { message: text }; }
  if (!res.ok) throw new Error(data?.message || `HTTP ${res.status}`);
  return data as T;
}

const totalAktif = computed(() => rows.value.filter((r) => String(r.status ?? '').toUpperCase() === 'AKTIF').length);

onMounted(async () => { await load(); await fclCount('users').catch(() => 0); });
</script>

<template>
  <Page auto-content-height>
    <div class="akun-page">
      <div class="akun-head">
        <div>
          <h2>👤 Kelola Akun</h2>
          <p>Akun login FCL · {{ rows.length }} akun · {{ totalAktif }} aktif</p>
        </div>
        <div class="akun-actions">
          <a-button size="small" @click="downloadTemplate"><LucideDownload :size="14" /> Template</a-button>
          <a-button size="small" @click="importOpen = true">⬆ Import</a-button>
          <a-button size="small" type="primary" @click="openCreate"><LucidePlus :size="14" /> Tambah Akun</a-button>
        </div>
      </div>

      <div class="akun-toolbar">
        <a-input v-model:value="search" size="small" placeholder="Cari username, nama, role, divisi..." allow-clear class="w-72" />
      </div>

      <div class="akun-table-wrap">
        <table class="akun-table">
          <thead>
            <tr>
              <th>Username</th><th>Nama</th><th>Role</th><th>Divisi</th>
              <th>Email</th><th>Status</th><th>Dibuat</th><th style="width:190px">Aksi</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="r in paged" :key="r.id">
              <td><b>{{ r.username }}</b></td>
              <td>{{ r.nama || '-' }}</td>
              <td>{{ r.role || '-' }}</td>
              <td>{{ r.divisi || '-' }}</td>
              <td>{{ r.email || '-' }}</td>
              <td>
                <a-tag :color="String(r.status ?? '').toUpperCase() === 'AKTIF' ? 'green' : 'default'">
                  {{ r.status || '-' }}
                </a-tag>
              </td>
              <td class="muted">{{ String(r.createdAt ?? '').slice(0, 10) }}</td>
              <td>
                <a-button v-if="String(r.status ?? '').toUpperCase() !== 'AKTIF'" size="small" link @click="setStatus(r, 'AKTIF')">Aktifkan</a-button>
                <a-button v-else size="small" link @click="setStatus(r, 'NONAKTIF')">Nonaktifkan</a-button>
                <a-button size="small" link @click="openReset(r)">🔑 Reset</a-button>
                <a-popconfirm title="Hapus akun ini?" ok-text="Ya" cancel-text="Batal" @confirm="remove(r)">
                  <a-button size="small" link danger><LucideTrash2 :size="13" /></a-button>
                </a-popconfirm>
              </td>
            </tr>
            <tr v-if="!paged.length"><td colspan="8" class="empty">Belum ada akun. Klik Tambah Akun atau Import.</td></tr>
          </tbody>
        </table>
      </div>

      <div class="akun-pager">
        <a-button size="small" :disabled="page <= 1" @click="page--">Sebelumnya</a-button>
        <span>Halaman {{ page }} / {{ Math.max(1, Math.ceil(filtered.length / pageSize)) }}</span>
        <a-button size="small" :disabled="page >= Math.ceil(filtered.length / pageSize)" @click="page++">Berikutnya</a-button>
      </div>
    </div>

    <!-- Tambah akun -->
    <a-modal v-model:open="createOpen" title="Tambah Akun" ok-text="Simpan" cancel-text="Batal" :confirm-loading="creating" @ok="submitCreate">
      <a-form layout="vertical">
        <div class="form-grid">
          <a-form-item label="Username" required><a-input v-model:value="form.username" placeholder="mis. operator1" /></a-form-item>
          <a-form-item label="Password" required><a-input-password v-model:value="form.password" placeholder="minimal 6 karakter" /></a-form-item>
          <a-form-item label="Nama"><a-input v-model:value="form.nama" /></a-form-item>
          <a-form-item label="Role"><a-input v-model:value="form.role" /></a-form-item>
          <a-form-item label="Divisi"><a-input v-model:value="form.divisi" /></a-form-item>
          <a-form-item label="Email"><a-input v-model:value="form.email" /></a-form-item>
        </div>
        <a-form-item label="Status">
          <a-select v-model:value="form.status">
            <a-select-option value="AKTIF">AKTIF</a-select-option>
            <a-select-option value="NONAKTIF">NONAKTIF</a-select-option>
          </a-select>
        </a-form-item>
      </a-form>
    </a-modal>

    <!-- Reset password -->
    <a-modal v-model:open="resetOpen" :title="`Reset Password — ${resetTarget?.username ?? ''}`" ok-text="Reset" cancel-text="Batal" @ok="submitReset">
      <a-form layout="vertical">
        <a-form-item label="Password baru" required>
          <a-input-password v-model:value="resetPassword" placeholder="minimal 6 karakter" />
        </a-form-item>
      </a-form>
    </a-modal>

    <!-- Import -->
    <a-modal v-model:open="importOpen" title="Import Akun" :width="720" ok-text="Import" cancel-text="Batal" :confirm-loading="importing" @ok="submitImport">
      <div class="import-info">
        <p>Upload file <b>.csv</b> atau <b>.xlsx</b> dengan kolom: username, password, nama, role, divisi, email, status.</p>
        <a-button size="small" @click="downloadTemplate"><LucideDownload :size="13" /> Download Template</a-button>
      </div>
      <input type="file" accept=".csv,.xlsx,.xls" @change="pickFile" />
      <div v-if="importFile" class="import-file">{{ importFile }} · {{ importRows.length }} baris</div>
      <div v-if="importRows.length" class="import-preview">
        <table>
          <thead><tr><th>username</th><th>password</th><th>nama</th><th>role</th><th>divisi</th><th>email</th><th>status</th></tr></thead>
          <tbody>
            <tr v-for="(r, i) in importRows.slice(0, 8)" :key="i">
              <td v-for="c in TEMPLATE_COLUMNS" :key="c.key">{{ r[c.key] || '-' }}</td>
            </tr>
          </tbody>
        </table>
      </div>
    </a-modal>
  </Page>
</template>

<style scoped>
.akun-page { padding: 14px; }
.akun-head { display:flex;justify-content:space-between;align-items:center;gap:12px;margin-bottom:12px; }
.akun-head h2 { display:flex;gap:8px;align-items:center;font-size:17px;font-weight:800;margin:0; }
.akun-head p { margin:2px 0 0;font-size:11px;color:hsl(var(--muted-foreground)); }
.akun-actions { display:flex;gap:6px; }
.akun-toolbar { margin-bottom:10px; }
.akun-table-wrap { overflow:auto;border:1px solid hsl(var(--border));border-radius:10px; }
.akun-table { width:100%;border-collapse:collapse;font-size:11px;white-space:nowrap; }
.akun-table th { background:hsl(var(--muted)/.6);text-align:left;padding:8px 10px;font-weight:800;color:hsl(var(--muted-foreground)); }
.akun-table td { padding:7px 10px;border-top:1px solid hsl(var(--border)); }
.akun-table tr:hover td { background:hsl(var(--muted)/.35); }
.muted { color:hsl(var(--muted-foreground)); }
.empty { text-align:center;padding:26px;color:hsl(var(--muted-foreground)); }
.akun-pager { display:flex;gap:10px;align-items:center;justify-content:flex-end;margin-top:10px;font-size:11px; }
.form-grid { display:grid;grid-template-columns:1fr 1fr;gap:10px; }
.import-info { background:hsl(var(--muted)/.5);padding:10px;border-radius:8px;margin-bottom:10px;font-size:11px;display:flex;justify-content:space-between;align-items:center;gap:10px; }
.import-info p { margin:0; }
.import-file { margin-top:8px;font-size:11px;font-weight:700; }
.import-preview { margin-top:10px;max-height:260px;overflow:auto;border:1px solid hsl(var(--border));border-radius:8px; }
.import-preview table { width:100%;border-collapse:collapse;font-size:10px; }
.import-preview th,.import-preview td { padding:5px 7px;text-align:left;border-bottom:1px solid hsl(var(--border)); }
.import-preview th { background:hsl(var(--muted)/.6); }
.w-72 { width:288px; }
</style>
