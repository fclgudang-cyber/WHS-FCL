<script lang="ts" setup>
import { computed, onMounted, ref, shallowRef, watch } from 'vue';

import type { EchartsUIType } from '@vben/plugins/echarts';

import { Page } from '@vben/common-ui';
import { EchartsUI, useEcharts } from '@vben/plugins/echarts';

import { Card, Table, Tag, notification } from 'ant-design-vue';

import { fclQuery } from '#/api/warehouse';
import { groupStockBySku, type StockGroup, type StockRow } from '#/utils/warehouse-stock';

const chartRef = shallowRef<EchartsUIType>();
const { renderEcharts } = useEcharts(chartRef);

const loading = ref(true);
const groups = ref<StockGroup[]>([]);
const topRows = computed(() =>
  [...groups.value].sort((a, b) => b.total - a.total).slice(0, 10),
);
const lowRows = computed(() =>
  groups.value.filter((g) => g.total <= g.minRef && g.minRef > 0),
);

const lowColumns = [
  { title: 'SKU', dataIndex: 'sku' },
  { title: 'Nama', dataIndex: 'nama', ellipsis: true },
  { title: 'Total', dataIndex: 'total', width: 90, align: 'right' as const },
  { title: 'Min', dataIndex: 'minRef', width: 70, align: 'right' as const },
  { title: 'Status', key: 'status', width: 100 },
];

async function load() {
  loading.value = true;
  try {
    const rows = await fclQuery<StockRow>('stock', {
      select: 'id,sku,nama,satuan,stok,stokMin,kategori,lokasi,batch,expDate',
      order: 'sku.asc',
      limit: 10000,
    });
    groups.value = groupStockBySku(rows.filter((d) => Number(d.stok) > 0));
  } catch (err: any) {
    console.error('[warehouse/analisis] load gagal', err);
    notification.error({ message: 'Analisis Stock', description: String(err?.message || err) });
  } finally {
    loading.value = false;
  }
}

watch(groups, (vals) => {
  if (!vals.length) return;
  const top = [...vals].sort((a, b) => b.total - a.total).slice(0, 10);
  renderEcharts({
    grid: { top: 20, right: 20, bottom: 40, left: 90 },
    tooltip: { trigger: 'axis' },
    xAxis: { type: 'value' },
    yAxis: {
      type: 'category',
      inverse: true,
      data: top.map((v) => v.sku),
    },
    series: [{ name: 'Total Stok', type: 'bar', data: top.map((v) => v.total) }],
  });
});

onMounted(load);
</script>

<template>
  <Page auto-content-height>
    <Card :bordered="false" title="Top 10 SKU berdasarkan Stok" :loading="loading">
      <EchartsUI ref="chartRef" height="360px" />
    </Card>

    <Card :bordered="false" class="mt-3" :title="'Stok Rendah (' + lowRows.length + ')'">
      <Table :columns="lowColumns" :data-source="lowRows" :row-key="(r: any) => r.key" size="small" :pagination="{ pageSize: 10 }">
        <template #bodyCell="{ column, record }">
          <span v-if="column.key === 'status'">
            <Tag :color="record.total <= 0 ? 'red' : 'orange'">{{ record.total <= 0 ? 'Habis' : 'Rendah' }}</Tag>
          </span>
        </template>
      </Table>
    </Card>
  </Page>
</template>
