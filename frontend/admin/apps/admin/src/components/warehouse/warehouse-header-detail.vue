<script lang="ts" setup>
/**
 * Master–detail CRUD generik untuk halaman FCL (Surat Jalan Masuk/Keluar,
 * Order, Retur, Return Distributor, Booking Mobil).
 * - Master dikerjakan dengan modal (fields MASTER = form modal).
 * - Detail dikerjakan dengan komponen FclCrud di dalam baris yang di-expand
 *   (base-filters langsung membatasi anak ke parent yang sedang dibuka).
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

import { fclDelete, fclDeleteWhere, fclInsert, fclPatch, fclQuery, formatRp } from '#/api/warehouse';
import FclCrud from '#/components/warehouse/warehouse-crud.vue';
import type { FclDisplayColumn, FclField } from '#/components/warehouse/warehouse-crud-types';
import FclPagination from './warehouse-pagination.vue';

const props = withDefaults(
  defineProps<{
    title: string;
    table: string; // master table
    columns: FclDisplayColumn[]; // tampilan master
    fields: FclField[]; // form modal master
    masterSearchKeys?: string[];
    searchPlaceholder?: string;
    csvName?: string;
    createLabel?: string;
    masterLimit?: number;
    order?: string;
    // anak detail
    parentKey: string; // nama kolom FK di tabel anak (mis. 'orderId' / 'sjId')
    childTitle?: string;
    childTable: string;
    childColumns: FclDisplayColumn[];
    childFields: FclField[];
    childCsvName?: string;
    childLimit?: number;
    childOrder?: string;
    // salin field dari master ke anak saat create (tanpa FK id master)
    childExtra?: Record<string, any> | ((master: any) => Record<string, any>);
    childBaseFilters?: Array<{ col: string; op?: any; value: any }>;
    paginate?: boolean;
  }>(),
  {
    masterSearchKeys: () => [],
    searchPlaceholder: 'Cari...',
    csvName: 'export.csv',
    createLabel: '+ Tambah',
    masterLimit: 1000,
    order: 'createdAt.desc',
    childTitle: 'Detail',
    childLimit: 500,
    childOrder: 'id.asc',
    childCsvName: 'detail.csv',
    childExtra: undefined,
    paginate: false,
  },
);

const loading = ref(true);
const rows = ref<any[]>([]);
const search = ref('');
const page = ref(1);
const pageSize = ref(20);
const expandedKeys = ref<Array<string>>([]);

const modalVisible = ref(false);
const editingId = ref<string | undefined>(undefined);
const submitting = ref(false);
const formState = reactive<Record<string, any>>({});

function fmtDate(v: any) { return v ? String(v).substring(0, 10) : ''; }
function fmtDateTime(v: any) { return v ? String(v).replace('T', ' ').substring(0, 16) : ''; }

const filtered = computed(() => {
  const q = search.value.trim().toLowerCase();
  if (!q) return rows.value;
  return rows.value.filter((r) =>
    props.masterSearchKeys.some((k) => String(r?.[k] ?? '').toLowerCase().includes(q)),
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
    sorter: (a: any, b: any) =>
      String(a?.[c.key] ?? '').localeCompare(String(b?.[c.key] ?? ''), 'id', {
        numeric: true,
        sensitivity: 'base',
      }),
  }));
  cols.push({ title: '', key: '_action', width: 150, fixed: 'right' });
  return cols;
});

async function load() {
  loading.value = true;
  try {
    rows.value = await fclQuery(props.table, { select: '*', order: props.order, limit: props.masterLimit });
  } catch (err: any) {
    console.error('[wh-header-detail] load gagal', err);
    notification.error({ message: props.title, description: String(err?.message || err) });
  } finally {
    loading.value = false;
  }
}

function resetForm(source?: any) {
  for (const f of props.fields) delete (formState as any)[f.name];
  for (const f of props.fields) {
    let v: any = source ? source[f.name] : undefined;
    if (f.type === 'date' || f.type === 'datetime') {
      formState[f.name] = v ? dayjs(String(v)) : undefined;
    } else if (f.type === 'number') {
      formState[f.name] = v === undefined || v === null || v === '' ? undefined : Number(v);
    } else {
      formState[f.name] = v ?? '';
    }
  }
}

function openCreate() { editingId.value = undefined; resetForm(); modalVisible.value = true; }
function openEdit(row: any) { editingId.value = String(row.id); resetForm(row); modalVisible.value = true; }

function validate(): string | null {
  for (const f of props.fields) {
    if (!f.required) continue;
    const v = (formState as any)[f.name];
    const empty = v === undefined || v === null || (typeof v === 'string' && !v.trim());
    if (empty) return `${f.label} wajib diisi`;
  }
  return null;
}

function buildPayload() {
  const payload: Record<string, any> = {};
  for (const f of props.fields) {
    let v = (formState as any)[f.name];
    if (f.type === 'date') v = v && (v as Dayjs).isValid?.() ? (v as Dayjs).format('YYYY-MM-DD') : String(v ?? '');
    else if (f.type === 'datetime') v = v && (v as Dayjs).isValid?.() ? (v as Dayjs).format('YYYY-MM-DDTHH:mm:ss') : String(v ?? '');
    if (v === undefined) continue;
    payload[f.name] = v;
  }
  return payload;
}

async function submit() {
  const err = validate();
  if (err) { message.warning(err); return; }
  submitting.value = true;
  try {
    const payload = buildPayload();
    if (editingId.value) {
      await fclPatch(props.table, editingId.value, payload);
      notification.success({ message: props.title, description: 'Data master tersimpan' });
    } else {
      await fclInsert(props.table, [payload]);
      notification.success({ message: props.title, description: 'Data master tersimpan' });
    }
    modalVisible.value = false;
    await load();
  } catch (e: any) {
    notification.error({ message: 'Gagal menyimpan', description: String(e?.message || e) });
  } finally {
    submitting.value = false;
  }
}

function confirmDelete(row: any) {
  Modal.confirm({
    title: `Hapus ${props.title}?`,
    content: 'Data master dan seluruh detailnya akan ikut terhapus dari daftar.',
    okText: 'Hapus',
    okType: 'danger',
    async onOk() {
      try {
        // anak dulu, baru master
        await fclDeleteWhere(props.childTable, [{ col: props.parentKey, value: String(row.id), op: 'eq' }]);
        await fclDelete(props.table, row.id);
        notification.success({ message: props.title, description: 'Master & detail terhapus' });
        await load();
      } catch (e: any) {
        notification.error({ message: 'Gagal menghapus', description: String(e?.message || e) });
      }
    },
  });
}

function exportCsv() {
  const header = props.columns.map((c) => c.title);
  const lines = filtered.value.map((r) =>
    props.columns.map((c) => `"${String(r?.[c.key] ?? '').replace(/"/g, '""')}"`).join(','),
  );
  const blob = new Blob([[header.join(','), ...lines].join('\n')], { type: 'text/csv;charset=utf-8' });
  const url = URL.createObjectURL(blob);
  const a = document.createElement('a');
  a.href = url;
  a.download = props.csvName;
  a.click();
  URL.revokeObjectURL(url);
}

function fieldOptions(f: FclField): { label: string; value: any }[] {
  if (!f.options) return [];
  return (f.options as any[]).map((o) => (typeof o === 'string' ? { label: o, value: o } : o));
}

function childRowFilter(record: any) {
  const base = [{ col: props.parentKey, op: 'eq' as const, value: record.id }];
  for (const f of props.childBaseFilters ?? []) base.push(f as any);
  // salinan business key selain parentKey bila perlu di filter tetap (mis. noSJ)
  return base;
}

function childExtraPayload(record: any): Record<string, any> {
  const copy: Record<string, any> = {};
  const extra = typeof props.childExtra === 'function' ? props.childExtra(record) : props.childExtra;
  if (extra) Object.assign(copy, extra);
  return { [props.parentKey]: record.id, ...copy };
}

function onExpand(expanded: boolean, record: any) {
  expandedKeys.value = expanded
    ? [...expandedKeys.value, String(record.id)]
    : expandedKeys.value.filter((k) => k !== String(record.id));
}

load();
</script>

<template>
  <Page auto-content-height>
    <template #extra>
      <input v-model="search" type="text" class="wh-search" :placeholder="searchPlaceholder" />
      <a-button @click="load" :loading="loading">
        <template #icon><LucideRefreshCw :size="14" /></template>
        Refresh
      </a-button>
      <a-button @click="exportCsv">
        <template #icon><LucideDownload :size="14" /></template>
        Export CSV
      </a-button>
      <a-button type="primary" @click="openCreate">{{ createLabel }}</a-button>
    </template>

    <a-card :bordered="false" :title="title">
      <a-table
        :columns="tableColumns"
        :data-source="pagedFiltered"
        :row-key="(r: any) => r.id"
        :loading="loading"
        :expanded-row-keys="expandedKeys"
        @expand="onExpand"
        :pagination="paginate ? false : { pageSize: 25, showSizeChanger: true, pageSizeOptions: ['25', '50', '100'] }"
        size="small"
      >
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === '_action'">
            <div class="wh-actions">
              <a-button type="link" size="small" @click="openEdit(record)">Edit</a-button>
              <a-button danger type="link" size="small" @click="confirmDelete(record)">Hapus</a-button>
            </div>
          </template>
          <template v-else>
            <template v-for="c in columns.filter((x) => x.key === column.dataIndex)" :key="c.key">
              <span v-if="c.fmt === 'rp'">{{ formatRp(record[c.key]) }}</span>
              <span v-else-if="c.fmt === 'date'">{{ fmtDate(record[c.key]) }}</span>
              <span v-else-if="c.fmt === 'datetime'">{{ fmtDateTime(record[c.key]) }}</span>
              <a-tag v-else-if="c.tag && c.tag[String(record[c.key] ?? '')]" :color="c.tag[String(record[c.key] ?? '')].color">
                {{ c.tag[String(record[c.key] ?? '')].text }}
              </a-tag>
              <span v-else>{{ record[c.key] ?? '—' }}</span>
            </template>
          </template>
        </template>

        <template #expandedRowRender="{ record }">
          <div class="wh-child-wrap">
            <FclCrud
              embedded
              :title="childTitle + ' — ' + (record.noSJ || record.noOrder || record.noRetur || record.noReturn || record.id)"
              :table="childTable"
              :columns="childColumns"
              :fields="childFields"
              :base-filters="childRowFilter(record)"
              :extra-payload="childExtraPayload(record)"
              :search-keys="[]"
              :csv-name="childCsvName"
              :limit="childLimit"
              :order="childOrder"
              form-title-create="Tambah Detail"
              form-title-edit="Edit Detail"
              create-label="+ Tambah Detail"
            />
          </div>
        </template>
      </a-table>
      <FclPagination v-if="paginate" v-model:page="page" v-model:page-size="pageSize" :total="filtered.length" />
    </a-card>

    <a-modal
      v-model:open="modalVisible"
      :title="editingId ? 'Edit ' + title : 'Tambah ' + title"
      :confirm-loading="submitting"
      width="560px"
      ok-text="Simpan"
      cancel-text="Batal"
      @ok="submit"
    >
      <a-form layout="vertical" class="wh-form">
        <a-form-item v-for="f in fields" :key="f.name" :label="f.label + (f.required ? ' *' : '')">
          <a-input v-if="(f.type ?? 'input') === 'input'" v-model:value="formState[f.name]" :placeholder="f.placeholder" />
          <a-textarea v-else-if="f.type === 'textarea'" v-model:value="formState[f.name]" :placeholder="f.placeholder" :rows="f.rows ?? 3" />
          <a-input-number v-else-if="f.type === 'number'" v-model:value="formState[f.name]" :min="f.min" :max="f.max" style="width: 100%" />
          <a-select v-else-if="f.type === 'select'" v-model:value="formState[f.name]" :placeholder="f.placeholder ?? 'Pilih...'" allow-clear style="width: 100%">
            <a-select-option v-for="o in fieldOptions(f)" :key="String(o.value)" :value="o.value">{{ o.label }}</a-select-option>
          </a-select>
          <a-date-picker v-else-if="f.type === 'date'" v-model:value="formState[f.name]" value-format="YYYY-MM-DD" style="width: 100%" />
          <a-date-picker v-else-if="f.type === 'datetime'" v-model:value="formState[f.name]" show-time value-format="YYYY-MM-DDTHH:mm:ss" style="width: 100%" />
        </a-form-item>
      </a-form>
    </a-modal>
  </Page>
</template>

<style scoped>
.wh-search {
  width: 240px;
  height: 32px;
  padding: 0 12px;
  border: 1px solid hsl(var(--border));
  border-radius: 6px;
  background: transparent;
  color: inherit;
}
.wh-actions { display: flex; gap: 4px; justify-content: flex-end; }
.wh-child-wrap { padding: 10px 12px; background: rgba(255, 255, 255, 0.03); border-radius: 8px; }
.wh-form :deep(.ant-form-item) { margin-bottom: 12px; }
</style>
