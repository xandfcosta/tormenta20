import { queryOptions } from '@tanstack/react-query'
import { api } from '@/shared/api/api'

export const campaignSessionsQueryOptions = (campaignId: number) =>
  queryOptions({
    queryKey: ['campaigns', campaignId, 'sessions'] as const,
    queryFn: () => api.sessions.list(campaignId),
  })

export const campaignSessionQueryOptions = (campaignId: number, id: number) =>
  queryOptions({
    queryKey: ['campaigns', campaignId, 'sessions', id] as const,
    queryFn: () => api.sessions.get(campaignId, id),
  })
