import { beforeEach, describe, expect, it } from 'vitest'
import { useAuthStore } from './auth-store'

beforeEach(() => {
  useAuthStore.setState({ user: null })
})

describe('useAuthStore', () => {
  it('initial user is null', () => {
    expect(useAuthStore.getState().user).toBeNull()
  })

  it('setUser stores the AuthUser', () => {
    useAuthStore.getState().setUser({
      id: 1,
      email: 'a@b.com',
      name: 'Alice',
    })
    expect(useAuthStore.getState().user).toEqual({
      id: 1,
      email: 'a@b.com',
      name: 'Alice',
    })
  })

  it('setUser(null) clears the user (logout)', () => {
    useAuthStore.setState({ user: { id: 1, email: 'a', name: null } })
    useAuthStore.getState().setUser(null)
    expect(useAuthStore.getState().user).toBeNull()
  })

  it('does not persist to storage (in-memory only)', () => {
    useAuthStore.getState().setUser({ id: 1, email: 'a', name: null })
    // Auth state is intentionally not persisted — it rehydrates from
    // /auth/me on app boot. Storage shouldn't carry the 'auth' key.
    expect(localStorage.getItem('auth')).toBeNull()
  })
})
