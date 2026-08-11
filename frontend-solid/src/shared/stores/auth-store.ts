import { type Accessor, createSignal } from 'solid-js'
import type { AuthUser } from '@/shared/api/api'

export type AuthStore = {
  user: Accessor<AuthUser | null>
  setUser: (user: AuthUser | null) => void
  isAuthenticated: Accessor<boolean>
}

/**
 * Replaces the React app's zustand auth store with a plain signal — that's the
 * whole migration for this one: a signal IS the store, and `isAuthenticated`
 * is derived instead of mirrored.
 *
 * A factory, not a module singleton, so tests (and a future multi-account
 * shell) get isolated instances; the app's single instance is provided through
 * `AuthProvider` (shared/stores/auth-context).
 *
 * @example const auth = createAuthStore(); auth.setUser(user); auth.isAuthenticated()
 */
export function createAuthStore(initial: AuthUser | null = null): AuthStore {
  const [user, setUser] = createSignal<AuthUser | null>(initial)
  return {
    user,
    setUser: (next) => setUser(next),
    isAuthenticated: () => user() !== null,
  }
}
