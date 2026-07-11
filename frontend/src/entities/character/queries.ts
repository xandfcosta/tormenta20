import { queryOptions } from '@tanstack/react-query'
import { api } from '@/shared/api/api'

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
