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
      authority: ['fcl:dashboard', 'sys:platform_admin'],
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
          authority: ['fcl:dashboard', 'sys:platform_admin'],
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
      authority: ['fcl:warehouse', 'sys:platform_admin'],
      // Halaman induk — arahkan ke stock sebagai halaman pertama
      redirect: '/fcl/warehouse/stock',
    },
    children: [
      {
        path: 'warehouse3d',
        name: 'FCLWarehouse3D',
        component: () => import('#/views/fcl/warehouse-3d/index.vue'),
        meta: { icon: 'lucide:box', order: 3, title: 'Warehouse 3D', authority: ['fcl:warehouse3d', 'sys:platform_admin'] },
      },
      {
        path: 'stock',
        name: 'FCLStock',
        component: () => import('#/views/fcl/stock/index.vue'),
        meta: {
          icon: 'lucide:package',
          order: 1,
          title: 'Stock Barang',
          authority: ['fcl:stock', 'sys:platform_admin'],
        },
      },
      {
        path: 'stockOpname',
        name: 'FCLStockOpname',
        component: () => import('#/views/fcl/stock-opname/index.vue'),
        meta: { icon: 'lucide:clipboard-check', order: 2, title: 'Stock Opname', authority: ['fcl:stockOpname', 'sys:platform_admin'] },
      },
      {
        path: 'inbound',
        name: 'FCLInbound',
        component: () => import('#/views/fcl/inbound/index.vue'),
        meta: { icon: 'lucide:package-plus', order: 4, title: 'Inbound', authority: ['fcl:inbound', 'sys:platform_admin'] },
      },
      {
        path: 'retur',
        name: 'FCLRetur',
        component: () => import('#/views/fcl/retur/index.vue'),
        meta: { icon: 'lucide:undo-2', order: 6, title: 'Retur', authority: ['fcl:retur', 'sys:platform_admin'] },
      },
      {
        path: 'order',
        name: 'FCLOrder',
        component: () => import('#/views/fcl/order/index.vue'),
        meta: { icon: 'lucide:shopping-cart', order: 7, title: 'Order', authority: ['fcl:order', 'sys:platform_admin'] },
      },
      {
        path: 'bookingMobil',
        name: 'FCLBookingMobil',
        component: () => import('#/views/fcl/booking-mobil/index.vue'),
        meta: { icon: 'lucide:car', order: 9, title: 'Booking Mobil', authority: ['fcl:bookingMobil', 'sys:platform_admin'] },
      },
      {
        path: 'assetWarehouse',
        name: 'FCLAssetWarehouse',
        component: () => import('#/views/fcl/asset-warehouse/index.vue'),
        meta: { icon: 'lucide:building-2', order: 10, title: 'Asset Warehouse', authority: ['fcl:assetWarehouse', 'sys:platform_admin'] },
      },
      {
        path: 'analisis',
        name: 'FCLAnalisis',
        component: () => import('#/views/fcl/analisis/index.vue'),
        meta: { icon: 'lucide:chart-bar', order: 11, title: 'Analisis Stock', authority: ['fcl:analisis', 'sys:platform_admin'] },
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
      authority: ['fcl:keuangan', 'sys:platform_admin'],
      redirect: '/fcl/keuangan/kasGudang',
    },
    children: [
      {
        path: 'kasGudang',
        name: 'FCLKasGudang',
        component: () => import('#/views/fcl/kas-gudang/index.vue'),
        meta: { icon: 'lucide:vault', order: 1, title: 'Kas Gudang', authority: ['fcl:kasGudang', 'sys:platform_admin'] },
      },
      {
        path: 'teamBuilding',
        name: 'FCLTeamBuilding',
        component: () => import('#/views/fcl/team-building/index.vue'),
        meta: { icon: 'lucide:users', order: 2, title: 'Team Building', authority: ['fcl:teamBuilding', 'sys:platform_admin'] },
      },
      {
        path: 'expense',
        name: 'FCLExpense',
        component: () => import('#/views/fcl/expense/index.vue'),
        meta: { icon: 'lucide:receipt', order: 3, title: 'Pengeluaran', authority: ['fcl:expense', 'sys:platform_admin'] },
      },
      {
        path: 'pettyCash',
        name: 'FCLPettyCash',
        component: () => import('#/views/fcl/petty-cash/index.vue'),
        meta: { icon: 'lucide:coins', order: 4, title: 'Petty Cash', authority: ['fcl:pettyCash', 'sys:platform_admin'] },
      },
      {
        path: 'pettyCashKategori',
        name: 'FCLPettyCashKategori',
        component: () => import('#/views/fcl/petty-cash-kategori/index.vue'),
        meta: { icon: 'lucide:list', order: 5, title: 'Kategori Petty Cash', authority: ['fcl:pettyCashKategori', 'sys:platform_admin'] },
      },
      {
        path: 'klaim',
        name: 'FCLKlaim',
        component: () => import('#/views/fcl/klaim/index.vue'),
        meta: { icon: 'lucide:badge-alert', order: 6, title: 'Klaim Paket', authority: ['fcl:klaim', 'sys:platform_admin'] },
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
      authority: ['fcl:operasional', 'sys:platform_admin'],
      redirect: '/fcl/operasional/laporanKerja',
    },
    children: [
      {
        path: 'laporanKerja',
        name: 'FCLLaporanKerja',
        component: () => import('#/views/fcl/laporan-kerja/index.vue'),
        meta: { icon: 'lucide:clipboard-pen', order: 1, title: 'Laporan Kerja', authority: ['fcl:laporanKerja', 'sys:platform_admin'] },
      },
      {
        path: 'grafikLaporan',
        name: 'FCLGrafikLaporan',
        component: () => import('#/views/fcl/grafik-laporan/index.vue'),
        meta: { icon: 'lucide:chart-line', order: 2, title: 'Grafik Laporan', authority: ['fcl:grafikLaporan', 'sys:platform_admin'] },
      },
      {
        path: 'instruksiLembur',
        name: 'FCLInstruksiLembur',
        component: () => import('#/views/fcl/instruksi-lembur/index.vue'),
        meta: { icon: 'lucide:alarm-clock', order: 3, title: 'Instruksi Lembur', authority: ['fcl:instruksiLembur', 'sys:platform_admin'] },
      },
      {
        path: 'ijin',
        name: 'FCLIjin',
        component: () => import('#/views/fcl/ijin/index.vue'),
        meta: { icon: 'lucide:calendar-x', order: 4, title: 'Ijin & Cuti', authority: ['fcl:ijin', 'sys:platform_admin'] },
      },
      {
        path: 'lembur',
        name: 'FCLLembur',
        component: () => import('#/views/fcl/lembur/index.vue'),
        meta: { icon: 'lucide:clock', order: 5, title: 'Lembur', authority: ['fcl:lembur', 'sys:platform_admin'] },
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
      authority: ['fcl:sdm', 'sys:platform_admin'],
      redirect: '/fcl/sdm/karyawan',
    },
    children: [
      {
        path: 'karyawan',
        name: 'FCLKaryawan',
        component: () => import('#/views/fcl/karyawan/index.vue'),
        meta: { icon: 'lucide:id-card', order: 1, title: 'Karyawan', authority: ['fcl:karyawan', 'sys:platform_admin'] },
      },
      {
        path: 'akun',
        name: 'FCLAkun',
        component: () => import('#/views/fcl/akun/index.vue'),
        meta: { icon: 'lucide:users', order: 2, title: 'Kelola Akun', authority: ['fcl:akun', 'sys:platform_admin'] },
      },
      {
        path: 'absensiKaryawan',
        name: 'FCLAbsensiKaryawan',
        component: () => import('#/views/fcl/absensi-karyawan/index.vue'),
        meta: { icon: 'lucide:fingerprint', order: 3, title: 'Absensi Karyawan', authority: ['fcl:absensiKaryawan', 'sys:platform_admin'] },
      },
      {
        path: 'jadwalShift',
        name: 'FCLJadwalShift',
        component: () => import('#/views/fcl/jadwal-shift/index.vue'),
        meta: { icon: 'lucide:calendar-clock', order: 4, title: 'Jadwal Shift', authority: ['fcl:jadwalShift', 'sys:platform_admin'] },
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
      authority: ['fcl:dokumentasi', 'sys:platform_admin'],
      redirect: '/fcl/dokumentasi/sop',
    },
    children: [
      {
        path: 'sop',
        name: 'FCLSop',
        component: () => import('#/views/fcl/sop/index.vue'),
        meta: { icon: 'lucide:book-open', order: 1, title: 'SOP', authority: ['fcl:sop', 'sys:platform_admin'] },
      },
    ],
  },
];

export default routes;
