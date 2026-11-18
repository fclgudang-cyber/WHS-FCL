import type {
  sortingservicev1_ListSortSessionsResponse,
  sortingservicev1_LockBasketRequest,
  sortingservicev1_LockBasketResponse,
  sortingservicev1_ScanSkuSortingRequest,
  sortingservicev1_ScanSkuSortingResponse,
  sortingservicev1_ScanSortSlotItemRequest,
  sortingservicev1_ScanSortSlotItemResponse,
  sortingservicev1_SortSession,
  sortingservicev1_StartSortSessionRequest,
  sortingservicev1_StartSortSessionResponse,
} from '#/api/generated/admin/service/v1';

import {
  useMutation,
  type UseMutationOptions,
  useQuery,
  type UseQueryOptions,
} from '@tanstack/vue-query';

import { apiClient } from '#/api/client';
import { type PaginationQuery } from '#/transport/rest';

export function useListSortSessions(
  query: PaginationQuery,
  options?: UseQueryOptions<sortingservicev1_ListSortSessionsResponse, Error>,
) {
  return useQuery({
    queryKey: ['listSortSessions', query],
    queryFn: () =>
      apiClient.sortingService.ListSessions(query.toRawParams()),
    ...options,
  });
}

export function useGetSortSession(
  id: number,
  options?: UseQueryOptions<sortingservicev1_SortSession, Error>,
) {
  return useQuery({
    queryKey: ['getSortSession', id],
    queryFn: () => apiClient.sortingService.GetSession({ id }),
    ...options,
  });
}

export function useStartSortSession(
  options?: UseMutationOptions<
    sortingservicev1_StartSortSessionResponse,
    Error,
    sortingservicev1_StartSortSessionRequest
  >,
) {
  return useMutation({
    mutationFn: (req) => apiClient.sortingService.StartSession(req),
    ...options,
  });
}

export function useScanSortSku(
  options?: UseMutationOptions<
    sortingservicev1_ScanSkuSortingResponse,
    Error,
    sortingservicev1_ScanSkuSortingRequest
  >,
) {
  return useMutation({
    mutationFn: (req) => apiClient.sortingService.ScanSku(req),
    ...options,
  });
}

export function useScanSortSlotItem(
  options?: UseMutationOptions<
    sortingservicev1_ScanSortSlotItemResponse,
    Error,
    sortingservicev1_ScanSortSlotItemRequest
  >,
) {
  return useMutation({
    mutationFn: (req) => apiClient.sortingService.ScanSlotItem(req),
    ...options,
  });
}

export function useCancelSortSession(
  options?: UseMutationOptions<object, Error, number>,
) {
  return useMutation({
    mutationFn: (id) => apiClient.sortingService.CancelSession({ sessionId: id }),
    ...options,
  });
}

export function useLockBasket(
  options?: UseMutationOptions<
    sortingservicev1_LockBasketResponse,
    Error,
    sortingservicev1_LockBasketRequest
  >,
) {
  return useMutation({
    mutationFn: (req) => apiClient.sortingService.LockBasket(req),
    ...options,
  });
}

export function useUnlockBasket(
  options?: UseMutationOptions<object, Error, { basketType: string; code: string }>,
) {
  return useMutation({
    mutationFn: (req) =>
      apiClient.sortingService.UnlockBasket(
        req as import('#/api/generated/admin/service/v1').sortingservicev1_UnlockBasketRequest,
      ),
    ...options,
  });
}
