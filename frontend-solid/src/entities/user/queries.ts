import { queryOptions } from '@tanstack/solid-query'
import { ApiError, api } from '@/shared/api/api'

/**
 * The signed-in user, or null when logged out. A 401 is an ANSWER here, not a
 * failure — every route guard reads this, so it must resolve rather than throw.
 */
export const meQueryOptions = queryOptions({
  queryKey: ['auth', 'me'] as const,
  queryFn: async () => {
    try {
      return await api.auth.me()
    } catch (error) {
      if (error instanceof ApiError && error.status === 401) return null
      throw error
    }
  },
  staleTime: 60_000,
  retry: false,
})
