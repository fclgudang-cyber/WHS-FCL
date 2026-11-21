<script lang="ts" setup>
import FclCrud from '#/components/warehouse/warehouse-crud.vue';

const columns: any[] = [
  { key: 'tanggal', title: 'Tanggal', width: 110, fmt: 'date' },
  { key: 'tipe', title: 'Tipe', width: 120, tag: { Pemasukan: { color: 'green', text: 'Pemasukan' }, Pengeluaran: { color: 'red', text: 'Pengeluaran' } } },
  { key: 'keterangan', title: 'Keterangan', ellipsis: true },
  { key: 'nominal', title: 'Nominal', width: 160, align: 'right', fmt: 'rp' },
  { key: 'createdBy', title: 'Dibuat Oleh', width: 130 },
  { key: 'createdAt', title: 'Dibuat', width: 150, fmt: 'datetime' },
];

const fields: any[] = [
  { name: 'tanggal', label: 'Tanggal', type: 'date', required: true },
  { name: 'tipe', label: 'Tipe', type: 'select', required: true, options: [
    { label: 'Pemasukan', value: 'Pemasukan' },
    { label: 'Pengeluaran', value: 'Pengeluaran' },
  ] },
  { name: 'nominal', label: 'Nominal', type: 'number', required: true, min: 0 },
  { name: 'keterangan', label: 'Keterangan', type: 'textarea', placeholder: 'Keperluan dana' },
];
</script>

<template>
  <FclCrud
    title="Team Building"
    table="teambuilding"
    :columns="columns"
    :fields="fields"
    :search-keys="['keterangan', 'tipe', 'createdBy']"
    search-placeholder="Cari keterangan..."
    csv-name="team-building.csv"
    form-title-create="Catat Team Building"
    form-title-edit="Edit Team Building"
  />
</template>
