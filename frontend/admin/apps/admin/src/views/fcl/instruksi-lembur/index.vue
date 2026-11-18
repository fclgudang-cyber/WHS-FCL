<script lang="ts" setup>
import FclHeaderDetail from '#/components/fcl/fcl-header-detail.vue';

const masterColumns: any[] = [
  { key: 'tanggal', title: 'Tanggal', width: 110, fmt: 'date' },
  { key: 'judul', title: 'Judul Instruksi', ellipsis: true },
  { key: 'keterangan', title: 'Keterangan', ellipsis: true },
  { key: 'createdBy', title: 'Dibuat Oleh', width: 120 },
];

const masterFields: any[] = [
  { name: 'tanggal', label: 'Tanggal', type: 'date', required: true },
  { name: 'judul', label: 'Judul Instruksi', type: 'input', required: true, placeholder: 'Instruksi lembur 13 Nov 2026' },
  { name: 'keterangan', label: 'Keterangan', type: 'textarea', rows: 2 },
];

const childColumns: any[] = [
  { key: 'nama', title: 'Nama', ellipsis: true },
  { key: 'divisi', title: 'Divisi', width: 130 },
  { key: 'shift', title: 'Shift', width: 90 },
  { key: 'tipe', title: 'Tipe', width: 90 },
  { key: 'jamKerja', title: 'Jam Kerja', width: 80, align: 'right' },
  { key: 'jamLembur', title: 'Jam Lembur', width: 80, align: 'right' },
  { key: 'totalJam', title: 'Total', width: 80, align: 'right' },
];

const childFields: any[] = [
  { name: 'nama', label: 'Nama', type: 'input', required: true },
  { name: 'divisi', label: 'Divisi', type: 'input' },
  { name: 'divisiTujuan', label: 'Divisi Tujuan', type: 'input' },
  { name: 'shift', label: 'Shift', type: 'select', options: ['Pagi', 'Malam', 'SPLIT'] },
  { name: 'tipe', label: 'Tipe', type: 'select', options: ['Lembur', 'Perbantuan'] },
  { name: 'jamKerja', label: 'Jam Kerja', type: 'number', min: 0 },
  { name: 'jamLembur', label: 'Jam Lembur', type: 'number', min: 0 },
  { name: 'totalJam', label: 'Total Jam', type: 'number', min: 0 },
  { name: 'keterangan', label: 'Keterangan', type: 'textarea', rows: 2 },
];
</script>

<template>
  <FclHeaderDetail
    title="Instruksi Lembur"
    table="instruksilembur"
    :columns="masterColumns"
    :fields="masterFields"
    :master-search-keys="['judul', 'keterangan', 'createdBy']"
    search-placeholder="Cari judul..."
    csv-name="instruksi-lembur.csv"
    create-label="+ Instruksi Baru"
    parent-key="instruksiId"
    child-title="Daftar Karyawan"
    child-table="instruksilemburdetail"
    :child-columns="childColumns"
    :child-fields="childFields"
    child-csv-name="instruksi-lembur-detail.csv"
    child-order="id.asc"
  />
</template>
