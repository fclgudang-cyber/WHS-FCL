<script lang="ts" setup>
import FclHeaderDetail from '#/components/fcl/fcl-header-detail.vue';

const masterColumns: any[] = [
  { key: 'noRetur', title: 'No. Retur', width: 150 },
  { key: 'tanggal', title: 'Tanggal', width: 110, fmt: 'date' },
  { key: 'sumber', title: 'Sumber', width: 160 },
  { key: 'alasan', title: 'Alasan', width: 180, ellipsis: true },
  { key: 'keterangan', title: 'Keterangan', ellipsis: true },
];

const masterFields: any[] = [
  { name: 'tanggal', label: 'Tanggal', type: 'date', required: true },
  { name: 'noRetur', label: 'No. Retur', type: 'input', required: true, placeholder: 'RT-2026-0001' },
  { name: 'sumber', label: 'Sumber', type: 'input', placeholder: 'Marketplace / Distributor ...' },
  { name: 'alasan', label: 'Alasan', type: 'input', required: true },
  { name: 'keterangan', label: 'Keterangan', type: 'textarea', rows: 2 },
];

const childColumns: any[] = [
  { key: 'sku', title: 'SKU', width: 130 },
  { key: 'nama', title: 'Nama', ellipsis: true },
  { key: 'qty', title: 'Qty', width: 80, align: 'right' },
  { key: 'satuan', title: 'Satuan', width: 80 },
  { key: 'batch', title: 'Batch', width: 120 },
  { key: 'expDate', title: 'Exp', width: 110, fmt: 'date' },
];

const childFields: any[] = [
  { name: 'sku', label: 'SKU', type: 'input', required: true },
  { name: 'nama', label: 'Nama Barang', type: 'input', required: true },
  { name: 'qty', label: 'Qty', type: 'number', required: true, min: 0 },
  { name: 'satuan', label: 'Satuan', type: 'input', placeholder: 'pcs / karton / dus' },
  { name: 'batch', label: 'Batch', type: 'input' },
  { name: 'expDate', label: 'Exp Date', type: 'date' },
];
</script>

<template>
  <FclHeaderDetail
    title="Retur"
    :paginate="true"
    table="retur"
    :columns="masterColumns"
    :fields="masterFields"
    :master-search-keys="['noRetur', 'sumber', 'alasan']"
    search-placeholder="Cari no retur / alasan..."
    csv-name="retur.csv"
    create-label="+ Retur Baru"
    parent-key="returId"
    child-title="Barang Retur"
    child-table="returdetail"
    :child-columns="childColumns"
    :child-fields="childFields"
    child-csv-name="retur-detail.csv"
    :child-extra="(m: any) => ({ noRetur: m.noRetur })"
  />
</template>
