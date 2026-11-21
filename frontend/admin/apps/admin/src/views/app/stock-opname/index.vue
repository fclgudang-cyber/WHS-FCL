<script lang="ts" setup>
import FclCrud from '#/components/warehouse/warehouse-crud.vue';

const statusTag: Record<string, { color: string; text: string }> = {
  Menunggu: { color: 'orange', text: 'Menunggu' },
  Selesai: { color: 'green', text: 'Selesai' },
  Approved: { color: 'blue', text: 'Approved' },
};

const columns: any[] = [
  { key: 'tanggal', title: 'Tanggal', width: 110, fmt: 'date' },
  { key: 'sku', title: 'SKU', width: 120 },
  { key: 'nama', title: 'Nama', ellipsis: true },
  { key: 'lokasi', title: 'Lokasi', width: 100 },
  { key: 'stokSistem', title: 'Sistem', width: 80, align: 'right' },
  { key: 'stokFisik', title: 'Fisik', width: 80, align: 'right' },
  { key: 'selisih', title: 'Selisih', width: 80, align: 'right' },
  { key: 'status', title: 'Status', width: 100, tag: statusTag },
];

const fields: any[] = [
  { name: 'tanggal', label: 'Tanggal', type: 'date', required: true },
  { name: 'sku', label: 'SKU', type: 'input', required: true },
  { name: 'nama', label: 'Nama Produk', type: 'input', required: true },
  { name: 'lokasi', label: 'Lokasi', type: 'input' },
  { name: 'batch', label: 'Batch', type: 'input' },
  { name: 'expDate', label: 'Exp Date', type: 'date' },
  { name: 'stokSistem', label: 'Stok Sistem', type: 'number', min: 0 },
  { name: 'stokFisik', label: 'Stok Fisik', type: 'number', min: 0 },
  { name: 'selisih', label: 'Selisih', type: 'number', min: 0 },
  { name: 'status', label: 'Status', type: 'select', options: ['Menunggu', 'Selesai', 'Approved'] },
  { name: 'catatan', label: 'Catatan', type: 'textarea', rows: 2 },
];
</script>

<template>
  <FclCrud
    title="Stock Opname"
    :paginate="true"
    table="stockopname"
    :columns="columns"
    :fields="fields"
    :search-keys="['sku', 'nama', 'lokasi', 'batch']"
    search-placeholder="Cari SKU..."
    csv-name="stock-opname.csv"
    form-title-create="Tambah Stok Opname"
    form-title-edit="Edit Stok Opname"
  />
</template>
