<script lang="ts" setup>
import FclCrud from '#/components/fcl/fcl-crud.vue';

const columns: any[] = [
  { key: 'kode', title: 'KODE', width: 90, align: 'center' },
  { key: 'nama', title: 'NAMA KATEGORI', ellipsis: true },
  { key: 'urutan', title: 'URUTAN', width: 90, align: 'center' },
  {
    key: 'aktif',
    title: 'STATUS',
    width: 110,
    align: 'center',
    tag: {
      Ya: { color: 'green', text: 'Aktif' },
      Tidak: { color: 'default', text: 'Nonaktif' },
    },
  },
];

const fields: any[] = [
  { name: 'kode', label: 'Kode', type: 'input', required: true, placeholder: 'A, B, C, ...' },
  { name: 'nama', label: 'Nama Kategori', type: 'input', required: true, placeholder: 'Contoh: WATER SUPPLY' },
  { name: 'urutan', label: 'Urutan', type: 'number', min: 1, max: 999, placeholder: 'Nomor urut tampilan' },
  {
    name: 'aktif',
    label: 'Status',
    type: 'select',
    options: [
      { label: 'Aktif', value: 'Ya' },
      { label: 'Nonaktif', value: 'Tidak' },
    ],
  },
];
</script>

<template>
  <FclCrud
    title="Kategori Transaksi Petty Cash"
    table="pettycashkategori"
    :columns="columns"
    :fields="fields"
    :search-keys="['kode', 'nama']"
    search-placeholder="Cari kode atau nama kategori..."
    order="urutan.asc"
    csv-name="kategori-petty-cash.csv"
    create-label="+ Tambah Kategori"
    form-title-create="Tambah Kategori"
    form-title-edit="Edit Kategori"
  />
</template>
