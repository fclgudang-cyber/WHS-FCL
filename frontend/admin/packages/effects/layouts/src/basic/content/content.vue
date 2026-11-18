<script lang="ts" setup>
import type {
  RouteLocationNormalizedLoadedGeneric,
} from 'vue-router';

import { type VNode } from 'vue';
import { RouterView } from 'vue-router';

import { usePreferences } from '@vben/preferences';
import { storeToRefs, useTabbarStore } from '@vben/stores';

import { IFrameRouterView } from '../../iframe';

defineOptions({ name: 'LayoutContent' });

const tabbarStore = useTabbarStore();
const { keepAlive } = usePreferences();

const { getCachedTabs, getExcludeCachedTabs, renderRouteView } =
  storeToRefs(tabbarStore);

/**
 * 转换组件，自动添加 name
 * @param component
 */
function transformComponent(
  component: VNode,
  route: RouteLocationNormalizedLoadedGeneric,
) {
  // 组件视图未找到，如果有设置后备视图，则返回后备视图，如果没有，则抛出错误
  if (!component) {
    console.error(
      'Component view not found，please check the route configuration',
    );
    return undefined;
  }

  const routeName = route.name as string;
  // 如果组件没有 name，则直接返回
  if (!routeName) {
    return component;
  }
  const componentName = (component?.type as any)?.name;

  // 已经设置过 name，则直接返回
  if (componentName) {
    return component;
  }

  // componentName 与 routeName 一致，则直接返回
  if (componentName === routeName) {
    return component;
  }

  // 设置 name
  component.type ||= {};
  (component.type as any).name = routeName;

  return component;
}
</script>

<template>
  <div class="relative h-full">
    <IFrameRouterView />
    <!--
      NOTE: 不要给这里加回 <Transition mode="out-in"> —— out-in 会等待离开的
      子组件触发 transitionend，而 KeepAlive 的离开组件被 deactivate 后不触发
      leave，导致进入组件永远无法挂载（页面变空白）。
    -->
    <RouterView v-slot="{ Component, route }">
      <Transition>
        <KeepAlive
          v-if="keepAlive"
          :exclude="getExcludeCachedTabs"
          :include="getCachedTabs"
        >
          <component
            :is="transformComponent(Component, route)"
            v-if="renderRouteView"
            v-show="!route.meta.iframeSrc"
            :key="route.fullPath"
          />
        </KeepAlive>
        <component
          :is="Component"
          v-else-if="renderRouteView"
          :key="route.fullPath"
        />
      </Transition>
    </RouterView>
  </div>
</template>
