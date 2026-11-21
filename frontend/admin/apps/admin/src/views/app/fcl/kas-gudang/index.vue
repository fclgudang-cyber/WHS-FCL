<script lang="ts" setup>
import FclCrud from '#/components/fcl/fcl-crud.vue';

const columns: any[] = [
  { key: 'tanggal', title: 'Tanggal', width: 110, fmt: 'date' },
  { key: 'tipe', title: 'Tipe', width: 90, tag: { IN: { color: 'green', text: 'Masuk' }, OUT: { color: 'red', text: 'Keluar' } } },
  { key: 'keterangan', title: 'Keterangan', ellipsis: true },
  { key: 'nominal', title: 'Nominal', width: 160, align: 'right', fmt: 'rp' },
  { key: 'buktiUrl', title: 'Bukti', width: 90 },
  { key: 'createdBy', title: 'Dibuat Oleh', width: 130 },
  { key: 'createdAt', title: 'Dibuat', width: 150, fmt: 'datetime' },
];

const fields: any[] = [
  { name: 'tanggal', label: 'Tanggal', type: 'date', required: true },
  { name: 'tipe', label: 'Tipe', type: 'select', required: true, options: [
    { label: 'Kas Masuk', value: 'IN' },
    { label: 'Kas Keluar', value: 'OUT' },
  ] },
  { name: 'nominal', label: 'Nominal', type: 'number', required: true, min: 0 },
  { name: 'keterangan', label: 'Keterangan', type: 'textarea', placeholder: 'Keterangan transaksi' },
  { name: 'buktiUrl', label: 'Bukti (URL)', type: 'input', placeholder: 'Link bukti (opsional)' },
];
</script>

<template>
  <FclCrud
    title="Kas Gudang"
    table="kasgudang"
    :columns="columns"
    :fields="fields"
    :search-keys="['keterangan', 'tipe', 'createdBy']"
    search-placeholder="Cari keterangan / tipe..."
    csv-name="kas-gudang.csv"
    form-title-create="Catat Kas Masuk/Keluar"
    form-title-edit="Edit Kas"
  />
</template>
