import type {
  settingservicev1_GenerateSortBasketsRequest,
  settingservicev1_GenerateSortBasketsResponse,
  settingservicev1_UpdateWaveSettingRequest,
  settingservicev1_WaveSetting,
} from '#/api/generated/admin/service/v1';

import {
  useMutation,
  type UseMutationOptions,
  useQuery,
  type UseQueryOptions,
} from '@tanstack/vue-query';

import { apiClient } from '#/api/client';
import { queryClient } from '#/plugins/vue-query';
import { type PaginationQuery } from '#/transport/rest';

// ==============================
// 设置（波次设置 + keranjang master）
// ==============================

// 波次设置（SINGLE / MULTI）

export function useGetWaveSetting(
  waveType: 'SINGLE' | 'MULTI',
  options?: UseQueryOptions<settingservicev1_WaveSetting, Error>,
) {
  return useQuery({
    queryKey: ['getWaveSetting', waveType],
    queryFn: () => apiClient.settingService.GetWaveSetting({ waveType }),
    ...options,
  });
}

export function useUpdateWaveSetting(
  options?: UseMutationOptions<object, Error, settingservicev1_UpdateWaveSettingRequest>,
) {
  return useMutation({
    mutationFn: (req) => apiClient.settingService.UpdateWaveSetting(req),
    ...options,
  });
}

// Keranjang picking master

export async function fetchListPickingBaskets(params: PaginationQuery) {
  return queryClient.fetchQuery({
    queryKey: ['listPickingBaskets', params],
    queryFn: () =>
      apiClient.settingService.ListPickingBaskets(params.toRawParams()),
    staleTime: 0,
    retry: 0,
  });
}

export function useDeletePickingBasket(
  options?: UseMutationOptions<object, Error, number>,
) {
  return useMutation({
    mutationFn: (id) => apiClient.settingService.DeletePickingBasket({ id }),
    ...options,
  });
}

// Keranjang sortir master

export async function fetchListSortBaskets(params: PaginationQuery) {
  return queryClient.fetchQuery({
    queryKey: ['listSortBaskets', params],
    queryFn: () =>
      apiClient.settingService.ListSortBaskets(params.toRawParams()),
    staleTime: 0,
    retry: 0,
  });
}

export function useGenerateSortBaskets(
  options?: UseMutationOptions<
    settingservicev1_GenerateSortBasketsResponse,
    Error,
    settingservicev1_GenerateSortBasketsRequest
  >,
) {
  return useMutation({
    mutationFn: (req) => apiClient.settingService.GenerateSortBaskets(req),
    ...options,
  });
}

export function useDeleteSortBasket(
  options?: UseMutationOptions<object, Error, number>,
) {
  return useMutation({
    mutationFn: (id) => apiClient.settingService.DeleteSortBasket({ id }),
    ...options,
  });
}
