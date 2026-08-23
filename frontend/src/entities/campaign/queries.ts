import { queryOptions } from '@tanstack/solid-query'
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


/**
 * The public preview behind an invite token. Disabled without a token, and
 * never retried: a dead token is an ANSWER ("peça um link novo"), not a
 * hiccup worth three round-trips.
 */
export const campaignMembersQueryOptions = (campaignId: number) =>
  queryOptions({
    queryKey: ['campaigns', campaignId, 'members'] as const,
    queryFn: () => api.members.list(campaignId),
  })
