<script lang="ts" setup>
import FclCrud from '#/components/fcl/fcl-crud.vue';

const columns: any[] = [
  { key: 'judul', title: 'Judul', ellipsis: true },
  { key: 'kategori', title: 'Kategori', width: 140 },
  { key: 'konten', title: 'Konten', ellipsis: true },
  { key: 'updatedAt', title: 'Update', width: 150, fmt: 'datetime' },
];

const fields: any[] = [
  { name: 'judul', label: 'Judul', type: 'input', required: true },
  { name: 'kategori', label: 'Kategori', type: 'input', placeholder: 'SOP / Panduan / Kebijakan...' },
  { name: 'konten', label: 'Konten', type: 'textarea', rows: 10 },
];
</script>

<template>
  <FclCrud
    title="SOP & Dokumentasi"
    table="sop"
    :columns="columns"
    :fields="fields"
    :search-keys="['judul', 'kategori', 'konten']"
    search-placeholder="Cari judul..."
    order="updatedAt.desc"
    csv-name="sop.csv"
    form-title-create="Tambah Dokumen"
    form-title-edit="Edit Dokumen"
  />
</template>
