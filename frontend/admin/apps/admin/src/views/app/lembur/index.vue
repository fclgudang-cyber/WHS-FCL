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
  { key: 'divisi', title: 'Divisi', width: 140 },
  { key: 'jumlahJam', title: 'Jam', width: 80, align: 'right' },
  { key: 'status', title: 'Status', width: 110, tag: statusTag },
  { key: 'keterangan', title: 'Keterangan', ellipsis: true },
];

const fields: any[] = [
  { name: 'tanggal', label: 'Tanggal', type: 'date', required: true },
  { name: 'nama', label: 'Nama', type: 'input', required: true },
  { name: 'divisi', label: 'Divisi', type: 'input', placeholder: 'Marketplace, Distributor...' },
  { name: 'jumlahJam', label: 'Jumlah Jam', type: 'number', min: 0, required: true },
  { name: 'status', label: 'Status', type: 'select', options: ['Menunggu', 'Disetujui', 'Ditolak'] },
  { name: 'keterangan', label: 'Keterangan', type: 'textarea', rows: 2 },
];
</script>

<template>
  <FclCrud
    title="Lembur"
    table="lembur"
    :columns="columns"
    :fields="fields"
    :search-keys="['nama', 'divisi', 'keterangan']"
    search-placeholder="Cari nama..."
    csv-name="lembur.csv"
    form-title-create="Pengajuan Lembur"
    form-title-edit="Edit Lembur"
    create-label="+ Ajukan"
  />
</template>
