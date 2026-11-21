<script lang="ts" setup>
/**
 * Komponen generik untuk halaman CRUD FCL di Ghost-main.
 * Setiap config halaman hanya berisi: columns (tampilan), fields (form drawer),
 * searchKeys, labels — sisanya (list, search, create, edit, delete, export CSV)
 * ditangani di sini, memakai backend FCL (wh-service Go) via /api/tables/*.
 */
import type { Dayjs } from 'dayjs';

import { computed, reactive, ref, watch } from 'vue';

import { Page } from '@vben/common-ui';
import { LucideDownload, LucideRefreshCw } from '@vben/icons';

import {
  Modal,
  message,
  notification,
} from 'ant-design-vue';

import dayjs from 'dayjs';

import { fclDelete, fclInsert, fclPatch, fclQuery, formatRp, type FclFilter } from '#/api/warehouse';

import type { FclDisplayColumn, FclField } from './warehouse-crud-types';
import FclPagination from './warehouse-pagination.vue';

const props = withDefaults(
  defineProps<{
    title: string;
    table: string;
    columns: FclDisplayColumn[];
    fields: FclField[];
    searchKeys?: string[];
    searchPlaceholder?: string;
    csvName?: string;
    csvEnabled?: boolean;
    createEnabled?: boolean;
    actions?: 'delete' | 'edit' | 'edit+delete' | 'none';
    emptyText?: string;
    limit?: number;
    order?: string;
    select?: string;
    createLabel?: string;
    formTitleCreate?: string;
    formTitleEdit?: string;
    embedded?: boolean;
    baseFilters?: FclFilter[];
    extraPayload?: Record<string, any>;
    paginate?: boolean;
  }>(),
  {
    searchKeys: () => [],
    searchPlaceholder: 'Cari...',
    csvName: 'export.csv',
    csvEnabled: true,
    createEnabled: true,
    actions: 'edit+delete',
    emptyText: 'Kosong',
    limit: 1000,
    order: 'createdAt.desc',
    select: '*',
    createLabel: '+ Tambah',
    formTitleCreate: 'Tambah',
    formTitleEdit: 'Edit',
    embedded: false,
    baseFilters: () => [],
    extraPayload: undefined,
    paginate: false,
  },
);

const loading = ref(true);
const rows = ref<any[]>([]);
const search = ref('');
const page = ref(1);
const pageSize = ref(20);

const modalVisible = ref(false);
const editingId = ref<null | string>(null);
const submitting = ref(false);
const formState = reactive<Record<string, any>>({});

function fmtDate(v: any) {
  if (!v) return '';
  return String(v).substring(0, 10);
}
function fmtDateTime(v: any) {
  if (!v) return '';
  return String(v).replace('T', ' ').substring(0, 16);
}

const filtered = computed(() => {
  const q = search.value.trim().toLowerCase();
  if (!q) return rows.value;
  return rows.value.filter((r) =>
    props.searchKeys.some((k) => String(r?.[k] ?? '').toLowerCase().includes(q)),
  );
});

const pagedFiltered = computed(() =>
  props.paginate
    ? filtered.value.slice((page.value - 1) * pageSize.value, page.value * pageSize.value)
    : filtered.value,
);

watch(search, () => { page.value = 1; });

const tableColumns = computed(() => {
  const cols: any[] = props.columns.map((c) => ({
    title: c.title,
    dataIndex: c.key,
    key: c.key,
    width: c.width,
    align: c.align,
    ellipsis: c.ellipsis ?? true,
    fixed: c.fixed,
    sorter: (a: any, b: any) =>
      String(a?.[c.key] ?? '').localeCompare(String(b?.[c.key] ?? ''), 'id', {
        numeric: true,
        sensitivity: 'base',
      }),
  }));
  if (props.actions !== 'none') {
    cols.push({ title: '', key: '_action', width: props.actions === 'edit+delete' ? 150 : 90, fixed: 'right' });
  }
  return cols;
});

async function load() {
  loading.value = true;
  try {
    rows.value = await fclQuery(props.table, {
      select: props.select,
      order: props.order,
      limit: props.limit,
      filters: [...props.baseFilters],
    });
  } catch (err: any) {
    console.error('[wh-crud] load gagal', err);
    notification.error({
      message: props.title,
      description: err?.message || 'Gagal memuat data',
    });
  } finally {
    loading.value = false;
  }
}

function fieldOptions(f: FclField): { label: string; value: any }[] {
  if (!f.options) return [];
  return (f.options as any[]).map((o) =>
    typeof o === 'string' ? { label: o, value: o } : o,
  );
}

function resetForm(source?: any) {
  for (const f of props.fields) delete (formState as any)[f.name];
  for (const f of props.fields) {
    let v: any =
      f.type === 'number' ? undefined : f.type === 'date' || f.type === 'datetime' ? undefined : '';
    if (source) v = source[f.name];
    if (f.type === 'date' || f.type === 'datetime') {
      formState[f.name] = v ? dayjs(String(v)) : undefined;
    } else if (f.type === 'number') {
      formState[f.name] = v === undefined || v === '' ? undefined : Number(v);
    } else {
      formState[f.name] = v ?? '';
    }
  }
}

function openCreate(preset?: Record<string, any>) {
  editingId.value = null;
  resetForm();
  if (preset) {
    for (const [k, v] of Object.entries(preset)) {
      if (v === undefined) continue;
      (formState as any)[k] = v;
    }
  }
  modalVisible.value = true;
}

function openEdit(row: any) {
  editingId.value = String(row.id);
  resetForm(row);
  modalVisible.value = true;
}

function validate(): string | null {
  for (const f of props.fields) {
    if (!f.required) continue;
    const v = (formState as any)[f.name];
    const empty =
      v === undefined || v === null || (typeof v === 'string' && !v.trim()) || (dayjs.isDayjs(v) && !v.isValid());
    if (empty) return `${f.label} wajib diisi`;
  }
  return null;
}

function buildPayload() {
  const payload: Record<string, any> = { ...(props.extraPayload ?? {}) };
  for (const f of props.fields) {
    let v = (formState as any)[f.name];
    if (f.type === 'date') v = v && (v as Dayjs).isValid?.() ? (v as Dayjs).format('YYYY-MM-DD') : undefined;
    else if (f.type === 'datetime') v = v && (v as Dayjs).isValid?.() ? (v as Dayjs).format('YYYY-MM-DDTHH:mm:ss') : undefined;
    // Kolom numerik: string kosong dari form harus diabaikan, bukan dikirim ke DB.
    if (v === undefined || (f.type === 'number' && v === '')) continue;
    payload[f.name] = v;
  }
  return payload;
}

async function submit() {
  const err = validate();
  if (err) {
    message.warning(err);
    return;
  }
  submitting.value = true;
  try {
    const payload = buildPayload();
    if (editingId.value) {
      await fclPatch(props.table, editingId.value, payload);
      notification.success({ message: props.formTitleEdit, description: 'Data tersimpan' });
    } else {
      await fclInsert(props.table, [payload]);
      notification.success({ message: props.formTitleCreate, description: 'Data tersimpan' });
    }
    modalVisible.value = false;
    await load();
  } catch (e: any) {
    notification.error({ message: 'Gagal menyimpan', description: e?.message || String(e) });
  } finally {
    submitting.value = false;
  }
}

function confirmDelete(row: any) {
  Modal.confirm({
    title: 'Hapus data?',
    content: 'Data yang dihapus tidak dapat dikembalikan.',
    okText: 'Hapus',
    okType: 'danger',
    cancelText: 'Batal',
    async onOk() {
      try {
        await fclDelete(props.table, row.id);
        notification.success({ message: 'Hapus', description: 'Data terhapus' });
        await load();
      } catch (e: any) {
        notification.error({ message: 'Gagal menghapus', description: e?.message || String(e) });
      }
    },
  });
}

function exportCsv() {
  const header = props.columns.map((c) => c.title);
  const lines = filtered.value.map((r) =>
    props.columns
      .map((c) => `"${String(r?.[c.key] ?? '').replace(/"/g, '""')}"`)
      .join(','),
  );
  const csv = [header.join(','), ...lines].join('\n');
  const blob = new Blob([csv], { type: 'text/csv;charset=utf-8' });
  const url = URL.createObjectURL(blob);
  const a = document.createElement('a');
  a.href = url;
  a.download = props.csvName;
  a.click();
  URL.revokeObjectURL(url);
}

load();

// expose agar halaman induk bisa membuka modal "Tambah" dengan preset (mis. dari kartu dashboard)
defineExpose({ openCreate });
</script>

<template>
  <template v-if="embedded">
    <div class="wh-embedded">
      <div class="wh-embedded-bar">
        <span class="wh-chip-title">{{ title }}</span>
        <div class="wh-embedded-actions">
          <input v-model="search" type="text" class="wh-search" :placeholder="searchPlaceholder" />
          <a-button size="small" @click="load" :loading="loading">
            <template #icon><LucideRefreshCw :size="12" /></template>
          </a-button>
          <a-button v-if="csvEnabled" size="small" @click="exportCsv">CSV</a-button>
          <a-button v-if="createEnabled" size="small" type="primary" @click="openCreate">{{ createLabel }}</a-button>
        </div>
      </div>
      <a-table
        :columns="tableColumns"
        :data-source="pagedFiltered"
        :row-key="(r: any) => r.id"
        :loading="loading"
        :pagination="paginate ? false : { pageSize: 25, showSizeChanger: true }"
        size="small"
      >
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === '_action'">
            <div class="wh-actions">
              <a-button v-if="actions !== 'delete' && actions !== 'none'" type="link" size="small" @click="openEdit(record)">Edit</a-button>
              <a-button v-if="actions !== 'edit' && actions !== 'none'" danger type="link" size="small" @click="confirmDelete(record)">Hapus</a-button>
            </div>
          </template>
          <template v-else>
            <template v-for="c in columns.filter((x) => x.key === column.dataIndex)" :key="c.key">
              <span v-if="c.fmt === 'rp'">{{ formatRp(record[c.key]) }}</span>
              <span v-else-if="c.fmt === 'date'">{{ fmtDate(record[c.key]) }}</span>
              <span v-else-if="c.fmt === 'datetime'">{{ fmtDateTime(record[c.key]) }}</span>
              <span v-else-if="c.fmt === 'bool'">{{ record[c.key] ? 'Ya' : 'Tidak' }}</span>
              <a-tag v-else-if="c.tag && c.tag[String(record[c.key] ?? '')]" :color="c.tag[String(record[c.key] ?? '')].color">
                {{ c.tag[String(record[c.key] ?? '')].text }}
              </a-tag>
              <span v-else>{{ record[c.key] ?? '—' }}</span>
            </template>
          </template>
        </template>
      </a-table>
      <FclPagination v-if="paginate" v-model:page="page" v-model:page-size="pageSize" :total="filtered.length" />
    </div>
  </template>

  <Page v-else auto-content-height>
    <template #extra>
      <input
        v-model="search"
        type="text"
        class="wh-search"
        :placeholder="searchPlaceholder"
      />
      <a-button @click="load" :loading="loading">
        <template #icon><LucideRefreshCw :size="14" /></template>
        Refresh
      </a-button>
      <a-button v-if="csvEnabled" @click="exportCsv">
        <template #icon><LucideDownload :size="14" /></template>
        Export CSV
      </a-button>
      <a-button v-if="createEnabled" type="primary" @click="openCreate">
        {{ createLabel }}
      </a-button>
    </template>

    <a-card :bordered="false" :title="title">
      <a-table
        :columns="tableColumns"
        :data-source="pagedFiltered"
        :row-key="(r: any) => r.id"
        :loading="loading"
        :pagination="paginate ? false : { pageSize: 25, showSizeChanger: true, pageSizeOptions: ['25', '50', '100'] }"
        size="small"
      >
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === '_action'">
            <div class="wh-actions">
              <a-button v-if="actions !== 'delete' && actions !== 'none'" type="link" size="small" @click="openEdit(record)">Edit</a-button>
              <a-button v-if="actions !== 'edit' && actions !== 'none'" danger type="link" size="small" @click="confirmDelete(record)">Hapus</a-button>
            </div>
          </template>
          <template v-else>
            <!-- render cell sesuai fmt -->
            <template v-if="column">
              <template v-for="c in columns.filter((x) => x.key === column.dataIndex)" :key="c.key">
                <span v-if="c.fmt === 'rp'">{{ formatRp(record[c.key]) }}</span>
                <span v-else-if="c.fmt === 'date'">{{ fmtDate(record[c.key]) }}</span>
                <span v-else-if="c.fmt === 'datetime'">{{ fmtDateTime(record[c.key]) }}</span>
                <span v-else-if="c.fmt === 'bool'">{{ record[c.key] ? 'Ya' : 'Tidak' }}</span>
                <a-tag v-else-if="c.tag && c.tag[String(record[c.key] ?? '')]" :color="c.tag[String(record[c.key] ?? '')].color">
                  {{ c.tag[String(record[c.key] ?? '')].text }}
                </a-tag>
                <span v-else>{{ record[c.key] ?? '—' }}</span>
              </template>
            </template>
          </template>
        </template>
      </a-table>
      <FclPagination v-if="paginate" v-model:page="page" v-model:page-size="pageSize" :total="filtered.length" />
    </a-card>

  </Page>
    <a-modal
      v-model:open="modalVisible"
      :title="editingId ? formTitleEdit : formTitleCreate"
      :confirm-loading="submitting"
      width="560px"
      :ok-text="editingId ? 'Simpan' : 'Tambah'"
      cancel-text="Batal"
      @ok="submit"
    >
      <a-form layout="vertical" class="wh-form">
        <a-form-item
          v-for="f in fields.filter((x) => !(x.createOnly && editingId))"
          :key="f.name"
          :label="f.label + (f.required ? ' *' : '')"
        >
          <a-input
            v-if="(f.type ?? 'input') === 'input'"
            v-model:value="formState[f.name]"
            :placeholder="f.placeholder"
          />
          <a-textarea
            v-else-if="f.type === 'textarea'"
            v-model:value="formState[f.name]"
            :placeholder="f.placeholder"
            :rows="f.rows ?? 3"
          />
          <a-input-number
            v-else-if="f.type === 'number'"
            v-model:value="formState[f.name]"
            :placeholder="f.placeholder"
            :min="f.min"
            :max="f.max"
            style="width: 100%"
          />
          <a-select
            v-else-if="f.type === 'select'"
            v-model:value="formState[f.name]"
            :placeholder="f.placeholder ?? 'Pilih...'"
            allow-clear
            style="width: 100%"
          >
            <a-select-option v-for="o in fieldOptions(f)" :key="String(o.value)" :value="o.value">
              {{ o.label }}
            </a-select-option>
          </a-select>
          <a-date-picker
            v-else-if="f.type === 'date'"
            v-model:value="formState[f.name]"
            value-format="YYYY-MM-DD"
            style="width: 100%"
          />
          <a-date-picker
            v-else-if="f.type === 'datetime'"
            v-model:value="formState[f.name]"
            show-time
            value-format="YYYY-MM-DDTHH:mm:ss"
            style="width: 100%"
          />
        </a-form-item>
      </a-form>
    </a-modal>

</template>

<style scoped>
.wh-embedded { display: flex; flex-direction: column; gap: 12px; }
.wh-embedded-bar { display: flex; align-items: center; justify-content: space-between; gap: 8px; flex-wrap: wrap; }
.wh-embedded-actions { display: flex; gap: 8px; align-items: center; }
.wh-chip-title { font-weight: 700; }
.wh-search {
  width: 240px;
  height: 32px;
  padding: 0 12px;
  border: 1px solid hsl(var(--border));
  border-radius: 6px;
  font-size: 13px;
  background: transparent;
  color: inherit;
}
.wh-actions { display: flex; gap: 4px; justify-content: flex-end; }
.wh-form :deep(.ant-form-item) { margin-bottom: 12px; }
</style>
