import {
  createRouter,
  createWebHashHistory,
  createWebHistory,
} from 'vue-router';

import { resetStaticRoutes } from '@vben/utils';

import { createRouterGuard } from './guard';
import { routes } from './routes';

/**
 *  @zh_CN 创建vue-router实例
 */
const router = createRouter({
  history:
    import.meta.env.VITE_ROUTER_HISTORY === 'hash'
      ? createWebHashHistory(import.meta.env.VITE_BASE)
      : createWebHistory(import.meta.env.VITE_BASE),
  // 应该添加到路由的初始路由列表。
  routes,
  scrollBehavior: (to, _from, savedPosition) => {
    if (savedPosition) {
      return savedPosition;
    }
    return to.hash ? { behavior: 'smooth', el: to.hash } : { left: 0, top: 0 };
  },
  // 是否应该禁止尾部斜杠。
  // strict: true,
});

const resetRoutes = () => resetStaticRoutes(router, routes);

// Map off any stale-chunk failures from an open tab across deploys:
// force a full reload so the fresh index.html + chunks are picked up.
router.onError((error) => {
  const msg = String(error?.message ?? error ?? '');
  if (
    /chunk|loading chunk|failed to fetch|dynamically imported module|importing a module/i.test(
      msg,
    )
  ) {
    // eslint-disable-next-line no-console
    console.warn('[router] chunk load failed, hard reload', msg);
    window.location.reload();
    return;
  }
  throw error;
});

// 创建路由守卫
createRouterGuard(router);

export { resetRoutes, router };
