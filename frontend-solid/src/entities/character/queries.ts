import { queryOptions } from '@tanstack/solid-query'
import { api } from '@/shared/api/api'

export const charactersQueryOptions = queryOptions({
  queryKey: ['characters'] as const,
  queryFn: api.characters.list,
})

/** Static pick lists for the creation wizard — never go stale in a session. */
export const characterOptionsQueryOptions = queryOptions({
  queryKey: ['characters', 'options'] as const,
  queryFn: api.characters.options,
  staleTime: Number.POSITIVE_INFINITY,
})

export const characterQueryOptions = (id: number) =>
  queryOptions({
    queryKey: ['characters', id] as const,
    queryFn: () => api.characters.get(id),
  })

/** The character plus its server-computed sheet (defense, attribute totals). */
export const characterSheetQueryOptions = (id: number) =>
  queryOptions({
    queryKey: ['characters', id, 'sheet'] as const,
    queryFn: () => api.characters.getSheet(id),
  })
