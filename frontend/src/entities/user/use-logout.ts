import { useMutation, useQueryClient } from '@tanstack/react-query'
import { api } from '@/shared/api/api'
import { useAuthStore } from '@/shared/stores/auth-store'
import { meQueryOptions } from './queries'

/**
 * Logs the user out and tears down auth state: clears the `me` cache, drops
 * the users list, and resets the auth store. Navigation is left to the caller
 * via `onLoggedOut` so the hook stays router-free (and unit-testable) — both
 * the app shell and the Hub footer pass `() => navigate({ to: '/login' })`.
 *
 * @example
 *   const logout = useLogout(() => navigate({ to: '/login' }))
 *   <button onClick={() => logout.mutate()} disabled={logout.isPending} />
 */
export function useLogout(onLoggedOut?: () => void) {
  const qc = useQueryClient()
  const setUser = useAuthStore((s) => s.setUser)
  return useMutation({
    mutationFn: api.auth.logout,
    onSuccess: () => {
      qc.setQueryData(meQueryOptions.queryKey, null)
      qc.removeQueries({ queryKey: ['users'] })
      setUser(null)
      onLoggedOut?.()
    },
  })
}
