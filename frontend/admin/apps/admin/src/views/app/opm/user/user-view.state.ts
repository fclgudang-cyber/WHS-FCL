import { useUserStore } from '@vben/stores';

import { defineStore } from 'pinia';

import {
  fetchListTenants,
  fetchListUsers,
  type identityservicev1_ListTenantResponse as ListTenantResponse,
  type identityservicev1_ListUserResponse as ListUserResponse,
  PaginationQuery,
} from '#/api';

const userStore = useUserStore();

interface UserViewState {
  loading: boolean;
  currentTenantId: number | undefined;
  userList: ListUserResponse;
  tenantList: ListTenantResponse;
}

export const useUserViewStore = defineStore('user-view', {
  state: (): UserViewState => ({
    currentTenantId: undefined,
    loading: false,
    userList: { items: [], total: 0 } as unknown as ListUserResponse,
    tenantList: { items: [], total: 0 } as unknown as ListTenantResponse,
  }),

  actions: {
    async fetchTenantList(formValues?: any): Promise<ListTenantResponse> {
      this.loading = true;
      try {
        this.tenantList = await fetchListTenants(
          new PaginationQuery({
            formValues: { ...formValues, status: 'ON' },
          }),
        );
        return this.tenantList;
      } catch (error) {
        console.error('获取租户列表失败:', error);
        this.resetTenantList();
      } finally {
        this.loading = false;
      }
      return this.tenantList;
    },

    async fetchUserList(
      currentPage: number,
      pageSize: number,
      formValues: any,
    ): Promise<ListUserResponse> {
      this.loading = true;
      try {
        // Tidak ada filter organisasi: daftar pengguna diambil langsung.
        const values: Record<string, any> = { ...formValues };
        if (this.currentTenantId !== undefined) {
          values.tenant_id = this.currentTenantId;
        }
        this.userList = await fetchListUsers(
          new PaginationQuery({
            paging: { page: currentPage, pageSize },
            formValues: values,
          }),
        );
      } catch (error) {
        console.error(`获取租户[${this.currentTenantId}]的用户失败:`, error);
        this.resetUserList();
      } finally {
        this.loading = false;
      }
      return this.userList;
    },

    resetTenantList() {
      this.tenantList = { items: [], total: 0 } as unknown as ListTenantResponse;
    },

    resetUserList() {
      this.userList = { items: [], total: 0 } as unknown as ListUserResponse;
    },

    setCurrentTenantId(tenantId: number | undefined) {
      this.currentTenantId = tenantId;
    },

    getCurrentTenantId(): number {
      if (userStore.isTenantUser()) {
        return userStore.tenantId as number;
      }
      return this.currentTenantId ?? 0;
    },

    isTenantUser(): boolean {
      return userStore.isTenantUser();
    },
  },
});
