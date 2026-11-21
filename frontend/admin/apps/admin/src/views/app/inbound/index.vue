<script lang="ts" setup>
import FclHeaderDetail from '#/components/fcl/fcl-header-detail.vue';

const masterColumns: any[] = [
  { key: 'noSJ', title: 'No. SJ', width: 150 },
  { key: 'tanggal', title: 'Tanggal', width: 110, fmt: 'date' },
  { key: 'supplier', title: 'Supplier', width: 180, ellipsis: true },
  { key: 'status', title: 'Status', width: 120 },
  { key: 'keterangan', title: 'Keterangan', ellipsis: true },
];

const masterFields: any[] = [
  { name: 'tanggal', label: 'Tanggal', type: 'date', required: true },
  { name: 'noSJ', label: 'No. SJ', type: 'input', required: true, placeholder: 'SJ-2026-0001' },
  { name: 'supplier', label: 'Supplier', type: 'input', required: true },
  { name: 'status', label: 'Status', type: 'input', placeholder: 'Menunggu / Diterima' },
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
    title="Inbound (Barang Masuk)"
    table="suratjalanmasuk"
    :columns="masterColumns"
    :fields="masterFields"
    :master-search-keys="['noSJ', 'supplier', 'keterangan']"
    search-placeholder="Cari no SJ / supplier..."
    csv-name="inbound.csv"
    create-label="+ Inbound Barang"
    parent-key="sjId"
    child-title="Barang Masuk"
    child-table="suratjalanmasukdetail"
    :child-columns="childColumns"
    :child-fields="childFields"
    child-csv-name="inbound-detail.csv"
    :child-extra="(m: any) => ({ noSJ: m.noSJ })"
  />
</template>
