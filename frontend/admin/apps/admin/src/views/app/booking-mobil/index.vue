<script lang="ts" setup>
import FclHeaderDetail from '#/components/warehouse/warehouse-header-detail.vue';

const masterColumns: any[] = [
  { key: 'tanggal', title: 'Tanggal', width: 110, fmt: 'date' },
  { key: 'pic', title: 'PIC', width: 140 },
  { key: 'jamBerangkat', title: 'Jam', width: 80 },
  { key: 'tujuan', title: 'Tujuan', ellipsis: true },
  { key: 'totalBiaya', title: 'Total Biaya', width: 140, align: 'right', fmt: 'rp' },
  { key: 'status', title: 'Status', width: 120 },
];

const masterFields: any[] = [
  { name: 'tanggal', label: 'Tanggal', type: 'date', required: true },
  { name: 'pic', label: 'PIC / Driver', type: 'input', required: true },
  { name: 'jamBerangkat', label: 'Jam Berangkat', type: 'input', placeholder: '08:00' },
  { name: 'tujuan', label: 'Tujuan', type: 'input', required: true },
  { name: 'rute', label: 'Rute', type: 'input', placeholder: 'Rute perjalanan' },
  { name: 'parkir', label: 'Parkir', type: 'input', placeholder: '0' },
  { name: 'tol', label: 'Tol', type: 'input', placeholder: '0' },
  { name: 'bensin', label: 'Bensin', type: 'input', placeholder: '0' },
  { name: 'pkbm', label: 'PKBM', type: 'input', placeholder: '0' },
  { name: 'lainLain', label: 'Lain-lain', type: 'input', placeholder: '0' },
  { name: 'totalBiaya', label: 'Total Biaya', type: 'input', placeholder: '0' },
  { name: 'buktiPembayaranUrl', label: 'Bukti Pembayaran (URL)', type: 'input', placeholder: 'https://...' },
  { name: 'jamMulaiPerjalanan', label: 'Jam Mulai', type: 'input' },
  { name: 'jamTibaTujuan', label: 'Jam Tiba Tujuan', type: 'input' },
  { name: 'jamKembaliWarehouse', label: 'Jam Kembali', type: 'input' },
  { name: 'jamSampaiWarehouse', label: 'Jam Sampai', type: 'input' },
  { name: 'status', label: 'Status', type: 'input', placeholder: 'Menunggu / Jalan / Selesai' },
  { name: 'driverNotes', label: 'Catatan Driver', type: 'textarea', rows: 2 },
  { name: 'keterangan', label: 'Keterangan', type: 'textarea', rows: 2 },
];

const childColumns: any[] = [
  { key: 'tanggal', title: 'Tanggal', width: 110, fmt: 'date' },
  { key: 'namaCustomer', title: 'Customer', ellipsis: true },
  { key: 'noPo', title: 'No. PO', width: 130 },
  { key: 'totalCartoon', title: 'Carton', width: 90, align: 'right' },
  { key: 'keterangan', title: 'Keterangan', ellipsis: true },
];

const childFields: any[] = [
  { name: 'tanggal', label: 'Tanggal', type: 'date', required: true },
  { name: 'namaCustomer', label: 'Nama Customer', type: 'input', required: true },
  { name: 'noPo', label: 'No. PO', type: 'input' },
  { name: 'totalCartoon', label: 'Total Carton', type: 'number', min: 0 },
  { name: 'parkir', label: 'Parkir', type: 'input' },
  { name: 'tol', label: 'Tol', type: 'input' },
  { name: 'pkbm', label: 'PKBM', type: 'input' },
  { name: 'lainLain', label: 'Lain-lain', type: 'input' },
  { name: 'keterangan', label: 'Keterangan', type: 'textarea', rows: 2 },
];
</script>

<template>
  <FclHeaderDetail
    title="Booking Mobil"
    table="bookingmobil"
    :columns="masterColumns"
    :fields="masterFields"
    :master-search-keys="['pic', 'tujuan', 'rute', 'status']"
    search-placeholder="Cari PIC / tujuan..."
    csv-name="booking-mobil.csv"
    create-label="+ Booking Mobil"
    parent-key="bookingId"
    child-title="Stop / Customer"
    child-table="bookingmobildetail"
    :child-columns="childColumns"
    :child-fields="childFields"
    child-csv-name="booking-mobil-detail.csv"
    child-order="id.asc"
    :child-extra="(m: any) => ({ tanggal: m.tanggal })"
  />
</template>
