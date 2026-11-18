<script lang="ts" setup>
import FclCrud from '#/components/fcl/fcl-crud.vue';

const columns: any[] = [
  { key: 'tanggal', title: 'Tanggal', width: 110, fmt: 'date' },
  { key: 'perusahaan', title: 'Perusahaan', width: 160 },
  { key: 'kategori', title: 'Kategori', width: 140 },
  { key: 'keterangan', title: 'Keterangan', ellipsis: true },
  { key: 'nominal', title: 'Nominal', width: 160, align: 'right', fmt: 'rp' },
  { key: 'bank', title: 'Bank', width: 110 },
  { key: 'createdBy', title: 'Dibuat Oleh', width: 120 },
];

const fields: any[] = [
  { name: 'tanggal', label: 'Tanggal', type: 'date', required: true },
  { name: 'perusahaan', label: 'Perusahaan', type: 'input', required: true },
  { name: 'kategori', label: 'Kategori', type: 'input', placeholder: 'Contoh: Operasional, Logistik...' },
  { name: 'keterangan', label: 'Keterangan', type: 'textarea' },
  { name: 'nominal', label: 'Nominal', type: 'number', required: true, min: 0 },
  { name: 'bank', label: 'Bank', type: 'input' },
  { name: 'rekening', label: 'Rekening', type: 'input' },
];
</script>

<template>
  <FclCrud
    title="Pengeluaran (Expense)"
    table="expense"
    :columns="columns"
    :fields="fields"
    :search-keys="['keterangan', 'perusahaan', 'kategori', 'bank']"
    search-placeholder="Cari perusahaan / keterangan..."
    csv-name="expense.csv"
    form-title-create="Catat Pengeluaran"
    form-title-edit="Edit Pengeluaran"
  />
</template>
