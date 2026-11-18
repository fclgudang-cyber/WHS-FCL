<script lang="ts" setup>
import FclHeaderDetail from '#/components/fcl/fcl-header-detail.vue';

const statusTag: Record<string, { color: string; text: string }> = {
  Pending: { color: 'orange', text: 'Pending' },
  Disetujui: { color: 'green', text: 'Disetujui' },
  Ditolak: { color: 'red', text: 'Ditolak' },
  Lunas: { color: 'blue', text: 'Lunas' },
};

const masterColumns: any[] = [
  { key: 'tanggal', title: 'Tanggal', width: 110, fmt: 'date' },
  { key: 'pic', title: 'PIC', width: 150 },
  { key: 'resi', title: 'No. Resi', width: 160 },
  { key: 'harga', title: 'Harga', width: 150, align: 'right', fmt: 'rp' },
  { key: 'status', title: 'Status', width: 120, tag: statusTag },
  { key: 'keterangan', title: 'Keterangan', ellipsis: true },
];

const masterFields: any[] = [
  { name: 'tanggal', label: 'Tanggal', type: 'date', required: true },
  { name: 'pic', label: 'PIC', type: 'input', required: true },
  { name: 'resi', label: 'No. Resi', type: 'input', required: true },
  { name: 'harga', label: 'Harga', type: 'number', min: 0 },
  { name: 'status', label: 'Status', type: 'select', options: ['Pending', 'Disetujui', 'Ditolak', 'Lunas'] },
  { name: 'keterangan', label: 'Keterangan', type: 'textarea', rows: 2 },
];

const childColumns: any[] = [
  { key: 'sku', title: 'SKU', width: 140 },
  { key: 'harga', title: 'Harga', width: 160, align: 'right', fmt: 'rp' },
];

const childFields: any[] = [
  { name: 'sku', label: 'SKU', type: 'input', required: true },
  { name: 'harga', label: 'Harga Item', type: 'number', min: 0, required: true },
];
</script>

<template>
  <FclHeaderDetail
    title="Klaim Paket"
    table="klaim"
    :columns="masterColumns"
    :fields="masterFields"
    :master-search-keys="['pic', 'resi', 'keterangan']"
    search-placeholder="Cari PIC / resi..."
    csv-name="klaim.csv"
    create-label="+ Klaim Baru"
    parent-key="klaimId"
    child-title="Detail Klaim"
    child-table="klaimdetail"
    :child-columns="childColumns"
    :child-fields="childFields"
    child-csv-name="klaim-detail.csv"
    child-order="id.asc"
  />
</template>
