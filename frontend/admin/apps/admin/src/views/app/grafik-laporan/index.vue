<script lang="ts" setup>
import { computed, onMounted, ref, watch, shallowRef } from 'vue';

import type { EchartsUIType } from '@vben/plugins/echarts';

import { Page } from '@vben/common-ui';
import { EchartsUI, useEcharts } from '@vben/plugins/echarts';

import { Card, DatePicker, Table, notification } from 'ant-design-vue';
import dayjs from 'dayjs';

import { fclQuery } from '#/api/fcl';
import { matchRosterDivisiKeyword, ROSTER_DIVISI_KEYS } from '#/utils/fcl-roster';

const chartRef = shallowRef<EchartsUIType>();
const { renderEcharts } = useEcharts(chartRef);

const month = ref(dayjs().format('YYYY-MM'));
const loading = ref(false);

interface DivisiRow {
  divisi: string;
  total: number;
  totalPO: number;
  totalOrder: number;
  totalInbound: number;
  totalQtyInb: number;
}

const rows = ref<DivisiRow[]>([]);
const columns = [
  { title: 'Divisi', dataIndex: 'divisi' },
  { title: 'Laporan', dataIndex: 'total', width: 90, align: 'right' as const },
  { title: 'PO', dataIndex: 'totalPO', width: 80, align: 'right' as const },
  { title: 'Order', dataIndex: 'totalOrder', width: 80, align: 'right' as const },
  { title: 'Inbound', dataIndex: 'totalInbound', width: 90, align: 'right' as const },
  { title: 'Qty Inb', dataIndex: 'totalQtyInb', width: 90, align: 'right' as const },
];

const total = computed(() => rows.value.reduce((s, r) => s + r.total, 0));

async function load() {
  loading.value = true;
  try {
    // Ambil seluruh data 1 bulan terpilih
    const all = await fclQuery<any>('laporankerja', {
      select: 'tanggal,divisi,pic,totalPO,totalOrder,totalInbound,totalQtyInb',
      filters: [{ col: 'tanggal', op: 'gte', value: `${month.value}-01` }],
      limit: 5000,
    });
    const inMonth = (all || []).filter((r) =>
      String(r?.tanggal || '').startsWith(month.value),
    );
    const map: Record<string, DivisiRow> = {};
    inMonth.forEach((l) => {
      const key = matchRosterDivisiKeyword(String(l.divisi || '').toLowerCase()) || String(l.divisi || 'Lainnya');
      if (!map[key]) {
        map[key] = { divisi: key, total: 0, totalPO: 0, totalOrder: 0, totalInbound: 0, totalQtyInb: 0 };
      }
      map[key].total += 1;
      map[key].totalPO += Number(l.totalPO) || 0;
      map[key].totalOrder += Number(l.totalOrder) || 0;
      map[key].totalInbound += Number(l.totalInbound) || 0;
      map[key].totalQtyInb += Number(l.totalQtyInb) || 0;
    });
    rows.value = Object.values(map).sort((a, b) => b.total - a.total);
  } catch (err: any) {
    console.error('[fcl/grafik-laporan] load gagal', err);
    notification.error({ message: 'Grafik Laporan', description: String(err?.message || err) });
  } finally {
    loading.value = false;
  }
}

watch(rows, (vals) => {
  if (!vals.length) return;
  renderEcharts({
    grid: { top: 20, right: 20, bottom: 40, left: 60 },
    tooltip: { trigger: 'axis' },
    legend: { bottom: 0 },
    xAxis: {
      type: 'category',
      data: vals.map((v) => v.divisi),
      axisLabel: { rotate: 25, fontSize: 11 },
    },
    yAxis: { type: 'value', minInterval: 1 },
    series: [
      { name: 'Laporan', type: 'bar', data: vals.map((v) => v.total) },
      { name: 'Order', type: 'bar', data: vals.map((v) => v.totalOrder) },
      { name: 'Qty Inb', type: 'bar', data: vals.map((v) => v.totalQtyInb) },
    ],
  });
});

watch(month, load);
onMounted(load);
</script>

<template>
  <Page auto-content-height>
    <template #extra>
      <DatePicker.MonthPicker v-model:value="month" value-format="YYYY-MM" size="small" :allow-clear="false" />
    </template>

    <Card :bordered="false" title="Grafik Laporan per Divisi" :loading="loading">
      <EchartsUI ref="chartRef" height="360px" />
      <div class="fcl-summary">Total laporan bulan ini: <b>{{ total }}</b> entri</div>
    </Card>

    <Card :bordered="false" class="mt-3" title="Ringkasan per Divisi">
      <Table :columns="columns" :data-source="rows" :row-key="(r: any) => r.divisi" :pagination="false" size="small" />
    </Card>
  </Page>
</template>

<style scoped>
.fcl-summary { margin-top: 8px; font-size: 12px; color: hsl(var(--muted-foreground)); text-align: right; }
</style>
