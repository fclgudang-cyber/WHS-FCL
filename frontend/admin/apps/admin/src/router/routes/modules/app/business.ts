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
      authority: ['sys:platform_admin', 'sys:tenant_manager'],
    },
    children: [
      {
        path: '',
        name: 'FCLDashboardHome',
        component: () => import('#/views/app/dashboard/index.vue'),
        meta: {
          affixTab: true,
          title: 'Dashboard',
          icon: 'lucide:layout-dashboard',
          authority: ['sys:platform_admin', 'sys:tenant_manager'],
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
      authority: ['sys:platform_admin', 'sys:tenant_manager'],
      // Halaman induk — arahkan ke stock sebagai halaman pertama
      redirect: '/fcl/warehouse/stock',
    },
    children: [
      {
        path: 'warehouse3d',
        name: 'FCLWarehouse3D',
        component: () => import('#/views/app/warehouse-3d/index.vue'),
        meta: { icon: 'lucide:box', order: 3, title: 'Warehouse 3D', authority: ['sys:platform_admin', 'sys:tenant_manager'] },
      },
      {
        path: 'stock',
        name: 'FCLStock',
        component: () => import('#/views/app/stock/index.vue'),
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
        component: () => import('#/views/app/stock-opname/index.vue'),
        meta: { icon: 'lucide:clipboard-check', order: 2, title: 'Stock Opname', authority: ['fcl:stockOpname', 'sys:platform_admin'] },
      },
      {
        path: 'inbound',
        name: 'FCLInbound',
        component: () => import('#/views/app/inbound/index.vue'),
        meta: { icon: 'lucide:package-plus', order: 4, title: 'Inbound', authority: ['fcl:inbound', 'sys:platform_admin'] },
      },
      {
        path: 'retur',
        name: 'FCLRetur',
        component: () => import('#/views/app/retur/index.vue'),
        meta: { icon: 'lucide:undo-2', order: 6, title: 'Retur', authority: ['fcl:retur', 'sys:platform_admin'] },
      },
      {
        path: 'order',
        name: 'FCLOrder',
        component: () => import('#/views/app/order/index.vue'),
        meta: { icon: 'lucide:shopping-cart', order: 7, title: 'Order', authority: ['fcl:order', 'sys:platform_admin'] },
      },
      {
        path: 'bookingMobil',
        name: 'FCLBookingMobil',
        component: () => import('#/views/app/booking-mobil/index.vue'),
        meta: { icon: 'lucide:car', order: 9, title: 'Booking Mobil', authority: ['fcl:bookingMobil', 'sys:platform_admin'] },
      },
      {
        path: 'assetWarehouse',
        name: 'FCLAssetWarehouse',
        component: () => import('#/views/app/asset-warehouse/index.vue'),
        meta: { icon: 'lucide:building-2', order: 10, title: 'Asset Warehouse', authority: ['fcl:assetWarehouse', 'sys:platform_admin'] },
      },
      {
        path: 'analisis',
        name: 'FCLAnalisis',
        component: () => import('#/views/app/analisis/index.vue'),
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
      authority: ['sys:platform_admin', 'sys:tenant_manager'],
      redirect: '/fcl/keuangan/kasGudang',
    },
    children: [
      {
        path: 'kasGudang',
        name: 'FCLKasGudang',
        component: () => import('#/views/app/kas-gudang/index.vue'),
        meta: { icon: 'lucide:vault', order: 1, title: 'Kas Gudang', authority: ['fcl:kasGudang', 'sys:platform_admin'] },
      },
      {
        path: 'teamBuilding',
        name: 'FCLTeamBuilding',
        component: () => import('#/views/app/team-building/index.vue'),
        meta: { icon: 'lucide:users', order: 2, title: 'Team Building', authority: ['fcl:teamBuilding', 'sys:platform_admin'] },
      },
      {
        path: 'expense',
        name: 'FCLExpense',
        component: () => import('#/views/app/expense/index.vue'),
        meta: { icon: 'lucide:receipt', order: 3, title: 'Pengeluaran', authority: ['fcl:expense', 'sys:platform_admin'] },
      },
      {
        path: 'pettyCash',
        name: 'FCLPettyCash',
        component: () => import('#/views/app/petty-cash/index.vue'),
        meta: { icon: 'lucide:coins', order: 4, title: 'Petty Cash', authority: ['fcl:pettyCash', 'sys:platform_admin'] },
      },
      {
        path: 'pettyCashKategori',
        name: 'FCLPettyCashKategori',
        component: () => import('#/views/app/petty-cash-kategori/index.vue'),
        meta: { icon: 'lucide:list', order: 5, title: 'Kategori Petty Cash', authority: ['fcl:pettyCashKategori', 'sys:platform_admin'] },
      },
      {
        path: 'klaim',
        name: 'FCLKlaim',
        component: () => import('#/views/app/klaim/index.vue'),
        meta: { icon: 'lucide:badge-alert', order: 6, title: 'Klaim Paket', authority: ['fcl:klaim', 'sys:platform_admin'] },
      },
    ],
  },
  // Kompatibilitas: URL lama dialihkan ke modul utama.
  {
    path: '/fcl/operasional',
    redirect: '/opm',
    meta: { hideInMenu: true, title: 'Redirect' },
  },
  {
    path: '/fcl/operasional/:child',
    redirect: (to) => `/opm/${String(to.params.child ?? '')}`,
    meta: { hideInMenu: true, title: 'Redirect' },
  },
  {
    path: '/fcl/sdm',
    redirect: '/opm/karyawan',
    meta: { hideInMenu: true, title: 'Redirect' },
  },
  {
    path: '/fcl/sdm/:child',
    redirect: (to) => `/opm/${String(to.params.child ?? '')}`,
    meta: { hideInMenu: true, title: 'Redirect' },
  },
  {
    path: '/fcl/dokumentasi',
    redirect: '/system/sop',
    meta: { hideInMenu: true, title: 'Redirect' },
  },
  {
    path: '/fcl/dokumentasi/:child',
    redirect: '/system/sop',
    meta: { hideInMenu: true, title: 'Redirect' },
  },
];

export default routes;
