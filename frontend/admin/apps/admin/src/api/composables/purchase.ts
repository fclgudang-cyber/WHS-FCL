import type {
  purchaseservicev1_ClosePurchaseRequest,
  purchaseservicev1_CreatePurchaseRequest,
  purchaseservicev1_DeletePurchaseRequest,
  purchaseservicev1_ListPurchaseResponse,
  purchaseservicev1_PutawayPurchaseRequest,
  purchaseservicev1_ReceivePurchaseRequest,
  purchaseservicev1_UpdatePurchaseRequest,
} from '#/api/generated/admin/service/v1';

import {
  useMutation,
  type UseMutationOptions,
  useQuery,
  type UseQueryOptions,
} from '@tanstack/vue-query';

import { apiClient } from '#/api/client';
import { queryClient } from '#/plugins/vue-query';
import { makeUpdateMask, type PaginationQuery } from '#/transport/rest';

// ==============================
// 到货通知单（Purchase）
// ==============================

export function useListPurchases(
  query: PaginationQuery,
  options?: UseQueryOptions<purchaseservicev1_ListPurchaseResponse, Error>,
) {
  return useQuery({
    queryKey: ['listPurchases', query],
    queryFn: () => apiClient.purchaseService.List(query.toRawParams()),
    ...options,
  });
}

export async function fetchListPurchases(params: PaginationQuery) {
  return queryClient.fetchQuery({
    queryKey: ['listPurchases', params],
    queryFn: () => apiClient.purchaseService.List(params.toRawParams()),
    staleTime: 0,
    retry: 0,
  });
}

export function useGetPurchase(
  id: number,
  options?: UseQueryOptions<any, Error>,
) {
  return useQuery({
    queryKey: ['getPurchase', id],
    queryFn: () => apiClient.purchaseService.Get({ id }),
    ...options,
  });
}

export function useCreatePurchase(
  options?: UseMutationOptions<object, Error, Record<string, any>>,
) {
  return useMutation({
    mutationFn: (values) =>
      apiClient.purchaseService.Create({
        data: { ...values },
      } as purchaseservicev1_CreatePurchaseRequest),
    ...options,
  });
}

export function useUpdatePurchase(
  options?: UseMutationOptions<
    object,
    Error,
    { id: number; values: Record<string, any> }
  >,
) {
  return useMutation({
    mutationFn: ({ id, values }: { id: number; values: Record<string, any> }) =>
      apiClient.purchaseService.Update({
        id,
        data: { ...values },
        updateMask: makeUpdateMask(Object.keys(values ?? {})),
      } as purchaseservicev1_UpdatePurchaseRequest),
    ...options,
  });
}

export function useDeletePurchase(
  options?: UseMutationOptions<
    object,
    Error,
    purchaseservicev1_DeletePurchaseRequest
  >,
) {
  return useMutation({
    mutationFn: (req) => apiClient.purchaseService.Delete(req),
    ...options,
  });
}

export function useAnnouncePurchase(
  options?: UseMutationOptions<object, Error, { id: number }>,
) {
  return useMutation({
    mutationFn: (req) => apiClient.purchaseService.Announce(req),
    ...options,
  });
}

export function useCancelPurchase(
  options?: UseMutationOptions<object, Error, { id: number }>,
) {
  return useMutation({
    mutationFn: (req) => apiClient.purchaseService.Cancel(req),
    ...options,
  });
}

export function useClosePurchase(
  options?: UseMutationOptions<
    object,
    Error,
    purchaseservicev1_ClosePurchaseRequest
  >,
) {
  return useMutation({
    mutationFn: (req) => apiClient.purchaseService.Close(req),
    ...options,
  });
}

export function useReceivePurchase(
  options?: UseMutationOptions<
    object,
    Error,
    purchaseservicev1_ReceivePurchaseRequest
  >,
) {
  return useMutation({
    mutationFn: (req) => apiClient.purchaseService.Receive(req),
    ...options,
  });
}

export function usePutawayPurchase(
  options?: UseMutationOptions<
    object,
    Error,
    purchaseservicev1_PutawayPurchaseRequest
  >,
) {
  return useMutation({
    mutationFn: (req) => apiClient.purchaseService.Putaway(req),
    ...options,
  });
}
