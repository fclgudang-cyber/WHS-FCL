import type { RouteRecordRaw } from 'vue-router';

import { BasicLayout } from '#/layouts';
import { $t } from '#/locales';

const opm: RouteRecordRaw[] = [
  {
    path: '/opm',
    name: 'OrganizationalPersonnelManagement',
    component: BasicLayout,
    redirect: '/opm/users',
    meta: {
      order: 2001,
      icon: 'lucide:users',
      title: $t('menu.opm.moduleName'),
      keepAlive: true,
      authority: ['sys:platform_admin', 'sys:tenant_manager'],
    },
    children: [
      // URL lama dialihkan ke Kelola Pengguna.
      {
        path: 'org-units',
        redirect: '/opm/users',
        meta: { hideInMenu: true, title: 'Redirect' },
      },

      {
        path: 'positions',
        name: 'PositionManagement',
        meta: {
          order: 2,
          icon: 'lucide:briefcase',
          title: $t('menu.opm.position'),
          authority: ['sys:platform_admin', 'sys:tenant_manager'],
        },
        component: () => import('#/views/app/opm/position/index.vue'),
      },

      {
        path: 'users',
        name: 'UserManagement',
        meta: {
          order: 3,
          icon: 'lucide:user',
          title: $t('menu.opm.user'),
          authority: ['sys:platform_admin', 'sys:tenant_manager'],
        },
        component: () => import('#/views/app/opm/user/index.vue'),
      },
      // Modul FCL yang terintegrasi ke Organisasi & SDM.
      {
        path: 'karyawan',
        name: 'FCLKaryawan',
        meta: { order: 5, icon: 'lucide:id-card', title: 'Karyawan', authority: ['sys:platform_admin', 'sys:tenant_manager'] },
        component: () => import('#/views/app/karyawan/index.vue'),
      },
      {
        path: 'absensiKaryawan',
        name: 'FCLAbsensiKaryawan',
        meta: { order: 6, icon: 'lucide:fingerprint', title: 'Absensi Karyawan', authority: ['sys:platform_admin', 'sys:tenant_manager'] },
        component: () => import('#/views/app/absensi-karyawan/index.vue'),
      },
      {
        path: 'jadwalShift',
        name: 'FCLJadwalShift',
        meta: { order: 7, icon: 'lucide:calendar-clock', title: 'Jadwal Shift', authority: ['sys:platform_admin', 'sys:tenant_manager'] },
        component: () => import('#/views/app/jadwal-shift/index.vue'),
      },
      {
        path: 'laporanKerja',
        name: 'FCLLaporanKerja',
        meta: { order: 8, icon: 'lucide:clipboard-pen', title: 'Laporan Kerja', authority: ['sys:platform_admin', 'sys:tenant_manager'] },
        component: () => import('#/views/app/laporan-kerja/index.vue'),
      },
      {
        path: 'grafikLaporan',
        name: 'FCLGrafikLaporan',
        meta: { order: 9, icon: 'lucide:chart-line', title: 'Grafik Laporan', authority: ['sys:platform_admin', 'sys:tenant_manager'] },
        component: () => import('#/views/app/grafik-laporan/index.vue'),
      },
      {
        path: 'instruksiLembur',
        name: 'FCLInstruksiLembur',
        meta: { order: 10, icon: 'lucide:alarm-clock', title: 'Instruksi Lembur', authority: ['sys:platform_admin', 'sys:tenant_manager'] },
        component: () => import('#/views/app/instruksi-lembur/index.vue'),
      },
      {
        path: 'ijin',
        name: 'FCLIjin',
        meta: { order: 11, icon: 'lucide:calendar-x', title: 'Ijin & Cuti', authority: ['sys:platform_admin', 'sys:tenant_manager'] },
        component: () => import('#/views/app/ijin/index.vue'),
      },
      {
        path: 'lembur',
        name: 'FCLLembur',
        meta: { order: 12, icon: 'lucide:clock', title: 'Lembur', authority: ['sys:platform_admin', 'sys:tenant_manager'] },
        component: () => import('#/views/app/lembur/index.vue'),
      },
      {
        path: 'users/detail/:id',
        name: 'UserDetail',
        meta: {
          hideInMenu: true,
          title: $t('menu.opm.userDetail'),
          authority: ['sys:platform_admin', 'sys:tenant_manager'],
        },
        component: () => import('#/views/app/opm/user/detail/index.vue'),
      },
    ],
  },
];

export default opm;
