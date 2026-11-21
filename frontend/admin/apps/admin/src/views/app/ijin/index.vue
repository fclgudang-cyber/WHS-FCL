<script lang="ts" setup>
import FclCrud from '#/components/warehouse/warehouse-crud.vue';

const statusTag: Record<string, { color: string; text: string }> = {
  Menunggu: { color: 'orange', text: 'Menunggu' },
  Disetujui: { color: 'green', text: 'Disetujui' },
  Ditolak: { color: 'red', text: 'Ditolak' },
};

const columns: any[] = [
  { key: 'tanggal', title: 'Tanggal', width: 110, fmt: 'date' },
  { key: 'nama', title: 'Nama', width: 160 },
  { key: 'jenis', title: 'Jenis', width: 110 },
  { key: 'status', title: 'Status', width: 110, tag: statusTag },
  { key: 'keterangan', title: 'Keterangan', ellipsis: true },
  { key: 'createdBy', title: 'Dibuat Oleh', width: 120 },
];

const fields: any[] = [
  { name: 'tanggal', label: 'Tanggal', type: 'date', required: true },
  { name: 'nama', label: 'Nama', type: 'input', required: true },
  { name: 'jenis', label: 'Jenis', type: 'select', required: true, options: ['Ijin', 'Cuti', 'Sakit', 'Lainnya'] },
  { name: 'status', label: 'Status', type: 'select', options: ['Menunggu', 'Disetujui', 'Ditolak'] },
  { name: 'keterangan', label: 'Keterangan', type: 'textarea', rows: 2 },
];
</script>

<template>
  <FclCrud
    title="Ijin & Cuti"
    table="ijin"
    :columns="columns"
    :fields="fields"
    :search-keys="['nama', 'keterangan', 'jenis']"
    search-placeholder="Cari nama..."
    csv-name="ijin.csv"
    form-title-create="Pengajuan Ijin/Cuti"
    form-title-edit="Edit Ijin/Cuti"
    create-label="+ Ajukan"
  />
</template>
