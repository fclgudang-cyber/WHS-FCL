<script lang="ts" setup>
import type { VxeGridListeners, VxeGridProps } from '#/adapter/vxe-table';

import { h, ref, watch } from 'vue';

import { useVbenDrawer, type VbenFormProps } from '@vben/common-ui';
import { LucideFilePenLine, LucideInfo, LucideTrash2 } from '@vben/icons';
import { isEqual } from '@vben/utils';

import { message, Modal, notification } from 'ant-design-vue';
import * as XLSX from 'xlsx';

import { useVbenVxeGrid } from '#/adapter/vxe-table';
import {
  apiClient,
  fetchListPositions,
  fetchListRoles,
  genderToColor,
  genderToName,
  PaginationQuery,
  type identityservicev1_User as User,
  userStatusList,
  userStatusToColor,
  userStatusToName,
} from '#/api';
import { $t } from '#/locales';
import { router } from '#/router';
import { getRandomColor } from '#/utils/color';
import { useUserViewStore } from '#/views/app/opm/user/user-view.state';

import UserDrawer from './user-drawer.vue';

const userViewStore = useUserViewStore();

const formOptions: VbenFormProps = {
  // 默认展开
  collapsed: true,
  // 控制表单是否显示折叠按钮
  showCollapseButton: true,
  // 按下回车时是否提交表单
  submitOnEnter: true,
  schema: [
    {
      component: 'Input',
      fieldName: 'username',
      label: $t('page.user.form.username'),
      componentProps: {
        placeholder: $t('ui.placeholder.input'),
        allowClear: true,
      },
    },
    {
      component: 'Input',
      fieldName: 'realname',
      label: $t('page.user.form.realname'),
      componentProps: {
        placeholder: $t('ui.placeholder.input'),
        allowClear: true,
      },
    },
    {
      component: 'Input',
      fieldName: 'mobile',
      label: $t('page.user.form.mobile'),
      componentProps: {
        placeholder: $t('ui.placeholder.input'),
        allowClear: true,
      },
    },
    {
      component: 'Select',
      fieldName: 'status',
      label: $t('page.user.form.status'),
      componentProps: {
        options: userStatusList,
        placeholder: $t('ui.placeholder.select'),
        filterOption: (input: string, option: any) =>
          option.label.toLowerCase().includes(input.toLowerCase()),
        allowClear: true,
        showSearch: true,
      },
    },
    {
      component: 'ApiSelect',
      fieldName: 'roleId',
      label: $t('page.user.form.role'),
      componentProps: {
        allowClear: true,
        showSearch: true,
        placeholder: $t('ui.placeholder.select'),
        filterOption: (input: string, option: any) =>
          option.label.toLowerCase().includes(input.toLowerCase()),
        afterFetch: (data: { name: string; path: string }[]) => {
          return data.map((item: any) => ({
            label: item.name,
            value: item.id,
          }));
        },
        api: async () => {
          const result = await fetchListRoles(
            new PaginationQuery({
              formValues: {
                status: 'ON',
                type__not: 'TEMPLATE',
                tenant_id: userViewStore.currentTenantId ?? 0,
              },
            }),
          );
          return result.items;
        },
      },
    },
    {
      component: 'ApiSelect',
      fieldName: 'positionId',
      label: $t('page.user.form.position'),
      componentProps: {
        placeholder: $t('ui.placeholder.select'),
        allowClear: true,
        showSearch: true,
        alwaysLoad: true,
        immediate: true,
        filterOption: (input: string, option: any) =>
          option.label.toLowerCase().includes(input.toLowerCase()),
        afterFetch: (data: { name: string; path: string }[]) => {
          return data.map((item: any) => ({
            label: item.name,
            value: item.id,
          }));
        },
        api: async () => {
          const result = await fetchListPositions(
            new PaginationQuery({
              formValues: {
                status: 'ON',
                org_unit_id: userViewStore.currentOrgUnitId,
                tenant_id: userViewStore.currentTenantId ?? 0,
              },
            }),
          );
          return result.items;
        },
      },
    },
  ],
};

const gridOptions: VxeGridProps<User> = {
  height: 'auto',
  stripe: true,
  autoResize: true,
  toolbarConfig: {
    custom: true,
    export: true,
    import: false,
    refresh: true,
    zoom: true,
  },
  exportConfig: {},
  pagerConfig: {},
  rowConfig: {
    isHover: true,
    resizable: true,
  },
  resizableConfig: {},
  tooltipConfig: {
    showAll: true,
    enterable: true,
    contentMethod: ({ column, row }) => {
      const { field } = column;
      if (field === 'roleNames') {
        return `${row[field]}`;
      }
      // 其余的单元格使用默认行为
      return null;
    },
  },

  proxyConfig: {
    ajax: {
      query: async ({ page }, formValues) => {
        // console.log('query:', filters, form, formValues);
        return userViewStore.fetchUserList(
          page.currentPage,
          page.pageSize,
          formValues,
        );
      },
    },
  },

  columns: [
    { title: $t('ui.table.seq'), type: 'seq', width: 50 },
    { title: $t('page.user.table.username'), field: 'username', width: 120 },
    { title: $t('page.user.table.realname'), field: 'realname', width: 100 },
    { title: $t('page.user.table.nickname'), field: 'nickname', width: 100 },
    { title: $t('page.user.table.email'), field: 'email', width: 160 },
    { title: $t('page.user.table.mobile'), field: 'mobile', width: 130 },
    {
      title: $t('page.user.table.orgUnitId'),
      field: 'orgUnitNames',
      slots: { default: 'orgUnit' },
      width: 130,
    },
    {
      title: $t('page.user.table.positionId'),
      field: 'positionNames',
      slots: { default: 'position' },
      width: 130,
    },
    {
      title: $t('page.user.table.roleId'),
      field: 'roleNames',
      slots: { default: 'role' },
      width: 100,
      showOverflow: 'tooltip',
    },
    {
      title: $t('page.user.table.status'),
      field: 'status',
      width: 95,
      slots: { default: 'status' },
    },
    {
      title: $t('page.user.table.lastLoginAt'),
      field: 'lastLoginAt',
      formatter: 'formatDateTime',
      width: 160,
    },
    {
      title: $t('ui.table.createdAt'),
      field: 'createdAt',
      formatter: 'formatDateTime',
      width: 160,
    },
    { title: $t('ui.table.remark'), field: 'remark', width: 250 },

    {
      title: $t('ui.table.action'),
      field: 'action',
      fixed: 'right',
      slots: { default: 'action' },
      width: 120,
    },
  ],
};

const gridEvents: VxeGridListeners<User> = {
  cellDblclick: ({ row }) => {
    // console.log(`cell-click: ${row.id}`);
    handleDetail(row);
  },
};

const [Grid, gridApi] = useVbenVxeGrid({
  gridOptions,
  formOptions,
  gridEvents,
});

const [Drawer, drawerApi] = useVbenDrawer({
  // 连接抽离的组件
  connectedComponent: UserDrawer,

  onOpenChange(isOpen: boolean) {
    if (!isOpen) {
      // 关闭时，重载表格数据
      gridApi.reload();
    }
  },
});

/* 打开模态窗口 */
function openDrawer(create: boolean, row?: any) {
  drawerApi.setData({
    create,
    row,
  });

  drawerApi.open();
}

// ==================== Import akun dari Excel ====================
const importOpen = ref(false);
const importing = ref(false);
const importRows = ref<Record<string, string>[]>([]);
const importFileName = ref('');
const importColumns = ['username', 'password', 'realname', 'nickname', 'email', 'mobile', 'status'];

function downloadTemplate() {
  const ws = XLSX.utils.aoa_to_sheet([
    importColumns,
    ['operator1', 'rahasia123', 'Operator Gudang', 'Operator', 'operator@fcl.local', '08123456789', 'ON'],
  ]);
  const wb = XLSX.utils.book_new();
  XLSX.utils.book_append_sheet(wb, ws, 'Users');
  XLSX.writeFile(wb, 'template-akun-user.xlsx');
  message.success('Template berhasil diunduh');
}

async function pickImportFile(e: Event) {
  const input = e.target as HTMLInputElement;
  const file = input.files?.[0];
  if (!file) return;
  importFileName.value = file.name;
  try {
    const isCsv = /\.csv$/i.test(file.name);
    const wb = isCsv
      ? XLSX.read(await file.text(), { type: 'string' })
      : XLSX.read(await file.arrayBuffer(), { type: 'array' });
    const name = wb.SheetNames[0];
    if (!name || !wb.Sheets[name]) throw new Error('File tidak punya sheet');
    const matrix = XLSX.utils.sheet_to_json<string[]>(wb.Sheets[name], { header: 1, defval: '' });
    if (matrix.length < 2) throw new Error('File kosong atau tidak ada baris data');
    const headers = (matrix[0] ?? []).map((v) => String(v ?? '').trim().toLowerCase());
    importRows.value = matrix.slice(1)
      .filter((r) => Array.isArray(r) && r.some((v) => String(v ?? '').trim()))
      .map((r) => {
        const row: Record<string, string> = {};
        headers.forEach((key, i) => {
          if (key) row[key] = String(r[i] ?? '').trim();
        });
        return row;
      });
    message.success(`${importRows.value.length} baris terbaca`);
  } catch (err: any) {
    importRows.value = [];
    notification.error({ message: 'Gagal membaca file', description: err?.message });
  }
}

async function submitImport() {
  const rows = importRows.value.filter((r) => {
    const username = String(r.username ?? '').trim();
    const password = String(r.password ?? '').trim();
    return username.length > 0 && password.length > 0;
  });
  if (!rows.length) {
    message.warning('Tidak ada baris valid (username & password wajib)');
    return;
  }
  importing.value = true;
  let ok = 0;
  const failed: string[] = [];
  try {
    for (const r of rows) {
      try {
        await apiClient.userService.Create({
          data: {
            username: String(r.username).trim(),
            realname: String(r.realname ?? '').trim(),
            nickname: String(r.nickname ?? '').trim(),
            email: String(r.email ?? '').trim(),
            mobile: String(r.mobile ?? '').trim(),
            status: String(r.status ?? 'ON').trim() || 'ON',
            tenantId: userViewStore.currentTenantId ?? 0,
          } as any,
          password: String(r.password).trim(),
        } as any);
        ok++;
      } catch (err: any) {
        failed.push(`${r.username}: ${err?.message ?? 'gagal'}`);
      }
    }
    notification.success({
      message: 'Import selesai',
      description: `${ok} akun dibuat, ${failed.length} gagal`,
    });
    if (failed.length) {
      Modal.warning({ title: `${failed.length} akun gagal`, content: failed.slice(0, 10).join('\n'), width: 480 });
    }
    if (ok > 0) {
      importOpen.value = false;
      importRows.value = [];
      await gridApi.reload();
    }
  } finally {
    importing.value = false;
  }
}

/* 创建 */
function handleCreate() {
  console.log('创建');
  openDrawer(true);
}

/* 编辑 */
function handleEdit(row: any) {
  console.log('编辑', row);
  openDrawer(false, row);
}

/* 删除 */
async function handleDelete(row: any) {
  console.log('删除', row);

  try {
    await apiClient.userService.Delete({ id: row.id });

    notification.success({
      message: $t('ui.notification.delete_success'),
    });

    await gridApi.reload();
  } catch {
    notification.error({
      message: $t('ui.notification.delete_failed'),
    });
  }
}

/* 详情 */
function handleDetail(row: any) {
  router.push(`/opm/users/detail/${row.id}`);
}

watch(
  () => [userViewStore.currentOrgUnitId, userViewStore.currentTenantId],
  (newValues, oldValue) => {
    if (isEqual(newValues, oldValue)) {
      return;
    }
    console.log(newValues, oldValue);
    gridApi.reload();
  },
);
</script>

<template>
  <Grid :table-title="$t('menu.opm.user')">
    <template #toolbar-tools>
      <a-button size="small" @click="downloadTemplate">Template Excel</a-button>
      <a-button size="small" @click="importOpen = true">Import Excel</a-button>
      <a-button type="primary" size="small" @click="handleCreate">
        {{ $t('page.user.button.create') }}
      </a-button>
    </template>
    <template #status="{ row }">
      <a-tag :color="userStatusToColor(row.status)">
        {{ userStatusToName(row.status) }}
      </a-tag>
    </template>
    <template #gender="{ row }">
      <a-tag :color="genderToColor(row.gender)">
        {{ genderToName(row.gender) }}
      </a-tag>
    </template>
    <template #role="{ row }">
      <div>
        <a-tag
          v-for="role in row.roleNames"
          :key="role"
          class="mb-1 mr-1"
          :style="{
            backgroundColor: getRandomColor(role), // 随机背景色
            color: '#333', // 深色文字（适配浅色背景）
            border: 'none', // 可选：去掉边框更美观
          }"
        >
          {{ role }}
        </a-tag>
      </div>
    </template>
    <template #orgUnit="{ row }">
      <div>
        <a-tag
          v-for="orgUnit in row.orgUnitNames"
          :key="orgUnit"
          class="mb-1 mr-1"
          :style="{
            backgroundColor: getRandomColor(orgUnit), // 随机背景色
            color: '#333', // 深色文字（适配浅色背景）
            border: 'none', // 可选：去掉边框更美观
          }"
        >
          {{ orgUnit }}
        </a-tag>
      </div>
    </template>
    <template #position="{ row }">
      <div>
        <a-tag
          v-for="position in row.positionNames"
          :key="position"
          class="mb-1 mr-1"
          :style="{
            backgroundColor: getRandomColor(position), // 随机背景色
            color: '#333', // 深色文字（适配浅色背景）
            border: 'none', // 可选：去掉边框更美观
          }"
        >
          {{ position }}
        </a-tag>
      </div>
    </template>
    <template #action="{ row }">
      <a-button
        type="link"
        :icon="h(LucideInfo)"
        @click.stop="handleDetail(row)"
      />

      <a-button
        type="link"
        :icon="h(LucideFilePenLine)"
        @click.stop="handleEdit(row)"
      />
      <a-popconfirm
        :cancel-text="$t('ui.button.cancel')"
        :ok-text="$t('ui.button.ok')"
        :title="
          $t('ui.text.do_you_want_delete', {
            moduleName: $t('page.user.moduleName'),
          })
        "
        @confirm="handleDelete(row)"
      >
        <a-button danger type="link" :icon="h(LucideTrash2)" />
      </a-popconfirm>
    </template>
  </Grid>
  <Drawer />

  <a-modal
    v-model:open="importOpen"
    title="Import Akun dari Excel"
    :width="720"
    ok-text="Import"
    cancel-text="Batal"
    :confirm-loading="importing"
    @ok="submitImport"
  >
    <div class="import-info">
      <p>
        Kolom: <b>username</b>, <b>password</b>, <b>realname</b>,
        <b>nickname</b>, <b>email</b>, <b>mobile</b>, <b>status</b> (ON/OFF).
        Username &amp; password wajib diisi.
      </p>
      <a-button size="small" @click="downloadTemplate">Download Template</a-button>
    </div>
    <input type="file" accept=".xlsx,.xls,.csv" @change="pickImportFile" />
    <div v-if="importFileName" class="import-file">
      {{ importFileName }} · {{ importRows.length }} baris
    </div>
    <div v-if="importRows.length" class="import-preview">
      <table>
        <thead>
          <tr><th v-for="c in importColumns" :key="c">{{ c }}</th></tr>
        </thead>
        <tbody>
          <tr v-for="(r, i) in importRows.slice(0, 8)" :key="i">
            <td v-for="c in importColumns" :key="c">{{ r[c] || '-' }}</td>
          </tr>
        </tbody>
      </table>
    </div>
  </a-modal>
</template>

<style scoped>
.import-info {
  background: hsl(var(--muted) / 0.5);
  padding: 10px;
  border-radius: 8px;
  margin-bottom: 10px;
  font-size: 12px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
}
.import-info p { margin: 0; }
.import-file { margin-top: 8px; font-size: 12px; font-weight: 700; }
.import-preview { margin-top: 10px; max-height: 260px; overflow: auto; border: 1px solid hsl(var(--border)); border-radius: 8px; }
.import-preview table { width: 100%; border-collapse: collapse; font-size: 11px; }
.import-preview th,
.import-preview td { padding: 5px 7px; text-align: left; border-bottom: 1px solid hsl(var(--border)); }
.import-preview th { background: hsl(var(--muted) / 0.6); }
</style>
