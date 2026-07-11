import { queryOptions } from '@tanstack/react-query'
import { ApiError, api } from '@/shared/api/api'

export const meQueryOptions = queryOptions({
  queryKey: ['auth', 'me'] as const,
  queryFn: async () => {
    try {
      return await api.auth.me()
    } catch (err) {
      if (err instanceof ApiError && err.status === 401) return null
      throw err
    }
  },
  staleTime: 60_000,
  retry: false,
})

export const usersQueryOptions = queryOptions({
  queryKey: ['users'] as const,
  queryFn: api.users.list,
})

export const charactersQueryOptions = queryOptions({
  queryKey: ['characters'] as const,
  queryFn: api.characters.list,
})

export const characterOptionsQueryOptions = queryOptions({
  queryKey: ['characters', 'options'] as const,
  queryFn: api.characters.options,
  staleTime: Infinity,
})

export const characterQueryOptions = (id: number) =>
  queryOptions({
    queryKey: ['characters', id] as const,
    queryFn: () => api.characters.get(id),
  })

export const characterSheetQueryOptions = (id: number) =>
  queryOptions({
    queryKey: ['characters', id, 'sheet'] as const,
    queryFn: () => api.characters.getSheet(id),
  })

export const characterCampaignsQueryOptions = (id: number) =>
  queryOptions({
    queryKey: ['characters', id, 'campaigns'] as const,
    queryFn: () => api.characters.campaigns(id),
  })

export const campaignsQueryOptions = queryOptions({
  queryKey: ['campaigns'] as const,
  queryFn: api.campaigns.list,
})

export const campaignQueryOptions = (id: number) =>
  queryOptions({
    queryKey: ['campaigns', id] as const,
    queryFn: () => api.campaigns.get(id),
  })

export const campaignSessionsQueryOptions = (campaignId: number) =>
  queryOptions({
    queryKey: ['campaigns', campaignId, 'sessions'] as const,
    queryFn: () => api.sessions.list(campaignId),
  })

export const campaignSessionQueryOptions = (
  campaignId: number,
  id: number,
) =>
  queryOptions({
    queryKey: ['campaigns', campaignId, 'sessions', id] as const,
    queryFn: () => api.sessions.get(campaignId, id),
  })

export const campaignMembersQueryOptions = (campaignId: number) =>
  queryOptions({
    queryKey: ['campaigns', campaignId, 'members'] as const,
    queryFn: () => api.members.list(campaignId),
  })
