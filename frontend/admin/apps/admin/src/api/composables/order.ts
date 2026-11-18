import type {
  orderservicev1_CountOrderResponse,
  orderservicev1_CreateOrderRequest,
  orderservicev1_DeleteOrderRequest,
  orderservicev1_FulfillOrderRequest,
  orderservicev1_GetOrderRequest,
  orderservicev1_ImportMarketplaceRequest,
  orderservicev1_ImportMarketplaceResponse,
  orderservicev1_ListOrderResponse,
  orderservicev1_UpdateOrderRequest,
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
// 订单管理
// ==============================

export function useListOrders(
  query: PaginationQuery,
  options?: UseQueryOptions<orderservicev1_ListOrderResponse, Error>,
) {
  return useQuery({
    queryKey: ['listOrders', query],
    queryFn: () => apiClient.orderService.List(query.toRawParams()),
    ...options,
  });
}

export async function fetchListOrders(params: PaginationQuery) {
  return queryClient.fetchQuery({
    queryKey: ['listOrders', params],
    queryFn: () => apiClient.orderService.List(params.toRawParams()),
    staleTime: 0,
    retry: 0,
  });
}

export async function fetchListOrdersRaw(params: PaginationQuery) {
  return apiClient.orderService.List(params.toRawParams());
}

export function useGetOrder(
  id: number,
  options?: UseQueryOptions<any, Error>,
) {
  return useQuery({
    queryKey: ['getOrder', id],
    queryFn: () => apiClient.orderService.Get({ id }),
    ...options,
  });
}

export async function fetchGetOrder(req: orderservicev1_GetOrderRequest) {
  return apiClient.orderService.Get(req);
}

export function useCreateOrder(
  options?: UseMutationOptions<object, Error, Record<string, any>>,
) {
  return useMutation({
    mutationFn: (values) =>
      apiClient.orderService.Create({
        data: { ...values },
      } as orderservicev1_CreateOrderRequest),
    ...options,
  });
}

export function useUpdateOrder(
  options?: UseMutationOptions<
    object,
    Error,
    { id: number; values: Record<string, any> }
  >,
) {
  return useMutation({
    mutationFn: ({ id, values }: { id: number; values: Record<string, any> }) =>
      apiClient.orderService.Update({
        id,
        data: { ...values },
        updateMask: makeUpdateMask(Object.keys(values ?? {})),
      } as orderservicev1_UpdateOrderRequest),
    ...options,
  });
}

export function useDeleteOrder(
  options?: UseMutationOptions<object, Error, { id: number }>,
) {
  return useMutation({
    mutationFn: ({ id }: { id: number }) =>
      apiClient.orderService.Delete({ id }) as any,
    ...options,
  });
}

export function useConfirmOrder(
  options?: UseMutationOptions<object, Error, { id: number }>,
) {
  return useMutation({
    mutationFn: ({ id }) => apiClient.orderService.Confirm({ id }) as any,
    ...options,
  });
}

export function useCancelOrder(
  options?: UseMutationOptions<object, Error, { id: number }>,
) {
  return useMutation({
    mutationFn: ({ id }) => apiClient.orderService.Cancel({ id }) as any,
    ...options,
  });
}

export function useFulfillOrder(
  options?: UseMutationOptions<object, Error, { id: number }>,
) {
  return useMutation({
    mutationFn: ({ id }: { id: number }) =>
      apiClient.orderService.Fulfill({ id } as orderservicev1_FulfillOrderRequest) as any,
    ...options,
  });
}

export function useCompleteOrder(
  options?: UseMutationOptions<object, Error, { id: number }>,
) {
  return useMutation({
    mutationFn: ({ id }) => apiClient.orderService.Complete({ id }) as any,
    ...options,
  });
}

export function useImportMarketplace(
  options?: UseMutationOptions<orderservicev1_ImportMarketplaceResponse, Error, orderservicev1_ImportMarketplaceRequest>,
) {
  return useMutation({
    mutationFn: (values) => apiClient.orderService.ImportMarketplace(values),
    ...options,
  });
}
