import { queryOptions } from '@tanstack/solid-query'
import { type ApiClient, ApiError, type AuthUser, api } from '@/shared/api/api'

/**
 * Reads the session, mapping 401 to "logged out".
 *
 * A 401 is an ANSWER here, not a failure — every route guard reads this, so it
 * must resolve rather than throw. Split out of the queryOptions (and taking the
 * client as a parameter) so this rule is testable without patching a global:
 * the module-level `api` captures `globalThis.fetch` at import time.
 *
 * @example await fetchSessionUser(createApiClient(fake.fetch)) // null on 401
 */
export async function fetchSessionUser(client: ApiClient = api): Promise<AuthUser | null> {
  try {
    return await client.auth.me()
  } catch (error) {
    if (error instanceof ApiError && error.status === 401) return null
    throw error
  }
}

export const meQueryOptions = queryOptions({
  queryKey: ['auth', 'me'] as const,
  queryFn: () => fetchSessionUser(),
  staleTime: 60_000,
  retry: false,
})

