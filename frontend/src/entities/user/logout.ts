import type { QueryClient } from '@tanstack/solid-query'
import { type ApiClient, api } from '@/shared/api/api'
import type { AuthStore } from '@/shared/stores/auth-store'
import { meQueryOptions } from './queries'

export type LogoutDeps = {
  queryClient: QueryClient
  auth: AuthStore
  client?: ApiClient
}

/**
 * Logs out and tears down the auth state. Clearing the `me` cache is not
 * housekeeping — the route guards read it, so a stale hit there bounces the
 * just-logged-out user straight back into the app.
 *
 * Navigation is the caller's job, so this stays router-free and testable.
 *
 * @example await logout({ queryClient, auth }); navigate({ to: '/login' })
 */
export async function logout({ queryClient, auth, client = api }: LogoutDeps): Promise<void> {
  await client.auth.logout()
  queryClient.setQueryData(meQueryOptions.queryKey, null)
  queryClient.removeQueries({ queryKey: ['users'] })
  auth.setUser(null)
}
