import { queryOptions } from '@tanstack/react-query'
import { api } from '@/shared/api/api'

export const campaignsQueryOptions = queryOptions({
  queryKey: ['campaigns'] as const,
  queryFn: api.campaigns.list,
})

export const campaignQueryOptions = (id: number) =>
  queryOptions({
    queryKey: ['campaigns', id] as const,
    queryFn: () => api.campaigns.get(id),
  })

export const campaignMembersQueryOptions = (campaignId: number) =>
  queryOptions({
    queryKey: ['campaigns', campaignId, 'members'] as const,
    queryFn: () => api.members.list(campaignId),
  })
