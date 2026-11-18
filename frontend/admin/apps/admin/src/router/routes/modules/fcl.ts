import type { RouteRecordRaw } from 'vue-router';

import { BasicLayout } from '#/layouts';

const lazyLayout = () => BasicLayout();

const routes: RouteRecordRaw[] = [
  {
    path: '/fcl/dashboard',
    name: 'FCLDashboard',
    component: lazyLayout,
    meta: {
      order: -3,
      icon: 'lucide:layout-dashboard',
      title: 'Dashboard',
      ignoreAccess: true,
    },
    children: [
      {
        path: '',
        name: 'FCLDashboardHome',
        component: () => import('#/views/fcl/dashboard/index.vue'),
        meta: {
          affixTab: true,
          title: 'Dashboard',
          icon: 'lucide:layout-dashboard',
          ignoreAccess: true,
        },
      },
    ],
  },
  {
    path: '/fcl/warehouse',
    name: 'FCLWarehouse',
    component: lazyLayout,
    meta: {
      order: 1000,
      icon: 'lucide:warehouse',
      title: 'Inventaris & Logistik',
      ignoreAccess: true,
      // Halaman induk — arahkan ke stock sebagai halaman pertama
      redirect: '/fcl/warehouse/stock',
    },
    children: [
      {
        path: 'stock',
        name: 'FCLStock',
        component: () => import('#/views/fcl/stock/index.vue'),
        meta: {
          icon: 'lucide:package',
          order: 1,
          title: 'Stock Barang',
          ignoreAccess: true,
        },
      },
      {
        path: 'stockOpname',
        name: 'FCLStockOpname',
        component: () => import('#/views/fcl/stock-opname/index.vue'),
        meta: { icon: 'lucide:clipboard-check', order: 2, title: 'Stock Opname', ignoreAccess: true },
      },
      {
        path: 'inbound',
        name: 'FCLInbound',
        component: () => import('#/views/fcl/inbound/index.vue'),
        meta: { icon: 'lucide:package-plus', order: 4, title: 'Inbound', ignoreAccess: true },
      },
      {
        path: 'retur',
        name: 'FCLRetur',
        component: () => import('#/views/fcl/retur/index.vue'),
        meta: { icon: 'lucide:undo-2', order: 6, title: 'Retur', ignoreAccess: true },
      },
      {
        path: 'order',
        name: 'FCLOrder',
        component: () => import('#/views/fcl/order/index.vue'),
        meta: { icon: 'lucide:shopping-cart', order: 7, title: 'Order', ignoreAccess: true },
      },
      {
        path: 'bookingMobil',
        name: 'FCLBookingMobil',
        component: () => import('#/views/fcl/booking-mobil/index.vue'),
        meta: { icon: 'lucide:car', order: 9, title: 'Booking Mobil', ignoreAccess: true },
      },
      {
        path: 'assetWarehouse',
        name: 'FCLAssetWarehouse',
        component: () => import('#/views/fcl/asset-warehouse/index.vue'),
        meta: { icon: 'lucide:building-2', order: 10, title: 'Asset Warehouse', ignoreAccess: true },
      },
      {
        path: 'analisis',
        name: 'FCLAnalisis',
        component: () => import('#/views/fcl/analisis/index.vue'),
        meta: { icon: 'lucide:chart-bar', order: 11, title: 'Analisis Stock', ignoreAccess: true },
      },
    ],
  },
  {
    path: '/fcl/keuangan',
    name: 'FCLKeuangan',
    component: lazyLayout,
    meta: {
      order: 1100,
      icon: 'lucide:wallet',
      title: 'Keuangan',
      ignoreAccess: true,
      redirect: '/fcl/keuangan/kasGudang',
    },
    children: [
      {
        path: 'kasGudang',
        name: 'FCLKasGudang',
        component: () => import('#/views/fcl/kas-gudang/index.vue'),
        meta: { icon: 'lucide:vault', order: 1, title: 'Kas Gudang', ignoreAccess: true },
      },
      {
        path: 'teamBuilding',
        name: 'FCLTeamBuilding',
        component: () => import('#/views/fcl/team-building/index.vue'),
        meta: { icon: 'lucide:users', order: 2, title: 'Team Building', ignoreAccess: true },
      },
      {
        path: 'expense',
        name: 'FCLExpense',
        component: () => import('#/views/fcl/expense/index.vue'),
        meta: { icon: 'lucide:receipt', order: 3, title: 'Pengeluaran', ignoreAccess: true },
      },
      {
        path: 'pettyCash',
        name: 'FCLPettyCash',
        component: () => import('#/views/fcl/petty-cash/index.vue'),
        meta: { icon: 'lucide:coins', order: 4, title: 'Petty Cash', ignoreAccess: true },
      },
      {
        path: 'klaim',
        name: 'FCLKlaim',
        component: () => import('#/views/fcl/klaim/index.vue'),
        meta: { icon: 'lucide:badge-alert', order: 5, title: 'Klaim Paket', ignoreAccess: true },
      },
    ],
  },
  {
    path: '/fcl/operasional',
    name: 'FCLOperasional',
    component: lazyLayout,
    meta: {
      order: 1200,
      icon: 'lucide:settings-2',
      title: 'Operasional',
      ignoreAccess: true,
      redirect: '/fcl/operasional/laporanKerja',
    },
    children: [
      {
        path: 'laporanKerja',
        name: 'FCLLaporanKerja',
        component: () => import('#/views/fcl/laporan-kerja/index.vue'),
        meta: { icon: 'lucide:clipboard-pen', order: 1, title: 'Laporan Kerja', ignoreAccess: true },
      },
      {
        path: 'grafikLaporan',
        name: 'FCLGrafikLaporan',
        component: () => import('#/views/fcl/grafik-laporan/index.vue'),
        meta: { icon: 'lucide:chart-line', order: 2, title: 'Grafik Laporan', ignoreAccess: true },
      },
      {
        path: 'instruksiLembur',
        name: 'FCLInstruksiLembur',
        component: () => import('#/views/fcl/instruksi-lembur/index.vue'),
        meta: { icon: 'lucide:alarm-clock', order: 3, title: 'Instruksi Lembur', ignoreAccess: true },
      },
      {
        path: 'ijin',
        name: 'FCLIjin',
        component: () => import('#/views/fcl/ijin/index.vue'),
        meta: { icon: 'lucide:calendar-x', order: 4, title: 'Ijin & Cuti', ignoreAccess: true },
      },
      {
        path: 'lembur',
        name: 'FCLLembur',
        component: () => import('#/views/fcl/lembur/index.vue'),
        meta: { icon: 'lucide:clock', order: 5, title: 'Lembur', ignoreAccess: true },
      },
    ],
  },
  {
    path: '/fcl/sdm',
    name: 'FCLSDM',
    component: lazyLayout,
    meta: {
      order: 1300,
      icon: 'lucide:users-round',
      title: 'SDM',
      ignoreAccess: true,
      redirect: '/fcl/sdm/karyawan',
    },
    children: [
      {
        path: 'karyawan',
        name: 'FCLKaryawan',
        component: () => import('#/views/fcl/karyawan/index.vue'),
        meta: { icon: 'lucide:id-card', order: 1, title: 'Karyawan', ignoreAccess: true },
      },
      {
        path: 'absensiKaryawan',
        name: 'FCLAbsensiKaryawan',
        component: () => import('#/views/fcl/absensi-karyawan/index.vue'),
        meta: { icon: 'lucide:fingerprint', order: 3, title: 'Absensi Karyawan', ignoreAccess: true },
      },
      {
        path: 'jadwalShift',
        name: 'FCLJadwalShift',
        component: () => import('#/views/fcl/jadwal-shift/index.vue'),
        meta: { icon: 'lucide:calendar-clock', order: 4, title: 'Jadwal Shift', ignoreAccess: true },
      },
    ],
  },
  {
    path: '/fcl/dokumentasi',
    name: 'FCLDokumentasi',
    component: lazyLayout,
    meta: {
      order: 1400,
      icon: 'lucide:file-text',
      title: 'Dokumentasi',
      ignoreAccess: true,
      redirect: '/fcl/dokumentasi/sop',
    },
    children: [
      {
        path: 'sop',
        name: 'FCLSop',
        component: () => import('#/views/fcl/sop/index.vue'),
        meta: { icon: 'lucide:book-open', order: 1, title: 'SOP', ignoreAccess: true },
      },
    ],
  },
];

export default routes;
