import { apiClient } from '#/api/client';
import { queryClient } from '#/plugins/vue-query';
import { type PaginationQuery } from '#/transport/rest';

export async function fetchListPickingTasks(params: PaginationQuery) {
  return queryClient.fetchQuery({
    queryKey: ['listPickingTasks', params],
    queryFn: () => apiClient.pickingTaskService.List(params.toRawParams()),
    staleTime: 0,
    retry: 0,
  });
}
