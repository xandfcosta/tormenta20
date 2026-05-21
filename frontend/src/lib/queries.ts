import { queryOptions } from '@tanstack/react-query'
import { ApiError, api } from './api'

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
