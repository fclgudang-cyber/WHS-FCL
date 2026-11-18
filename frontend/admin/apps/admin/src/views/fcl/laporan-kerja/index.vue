<script lang="ts" setup>
import FclCrud from '#/components/fcl/fcl-crud.vue';

const columns: any[] = [
  { key: 'tanggal', title: 'Tanggal', width: 110, fmt: 'date' },
  { key: 'pic', title: 'PIC', width: 140 },
  { key: 'divisi', title: 'Divisi', width: 150 },
  { key: 'shift', title: 'Shift', width: 100 },
  { key: 'totalPO', title: 'PO', width: 80, align: 'right' },
  { key: 'totalOrder', title: 'Order', width: 80, align: 'right' },
  { key: 'totalQtyInb', title: 'Qty Inb', width: 90, align: 'right' },
  { key: 'totalJamKerja', title: 'Jam Kerja', width: 100, align: 'right' },
  { key: 'jamLembur', title: 'Lembur', width: 80, align: 'right' },
  { key: 'kendala', title: 'Kendala', ellipsis: true },
];

const fields: any[] = [
  { name: 'tanggal', label: 'Tanggal', type: 'date', required: true },
  { name: 'pic', label: 'PIC', type: 'input', required: true },
  { name: 'divisi', label: 'Divisi', type: 'input', placeholder: 'Marketplace, Distributor, Return...' },
  { name: 'shift', label: 'Shift', type: 'select', options: ['Pagi', 'Siang', 'Malam'] },
  { name: 'totalPO', label: 'Total PO', type: 'number', min: 0 },
  { name: 'totalOrder', label: 'Total Order', type: 'number', min: 0 },
  { name: 'totalQtyInb', label: 'Qty Inbound', type: 'number', min: 0 },
  { name: 'totalJamKerja', label: 'Total Jam Kerja', type: 'number', min: 0 },
  { name: 'jamLembur', label: 'Jam Lembur', type: 'number', min: 0 },
  { name: 'perbantuan', label: 'Perbantuan', type: 'input' },
  { name: 'kendala', label: 'Kendala', type: 'textarea', rows: 2 },
];
</script>

<template>
  <FclCrud
    title="Riwayat Laporan Kerja (Harian)"
    table="laporankerja"
    :columns="columns"
    :fields="fields"
    :search-keys="['pic', 'divisi', 'kendala', 'perbantuan']"
    search-placeholder="Cari PIC / divisi..."
    csv-name="laporan-kerja.csv"
    form-title-create="Form Laporan Kerja"
    form-title-edit="Edit Laporan Kerja"
    create-label="+ Buat"
  />
</template>
