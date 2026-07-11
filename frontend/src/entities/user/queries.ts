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
