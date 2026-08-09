import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { act, renderHook, waitFor } from '@testing-library/react'
import type { ReactNode } from 'react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { useAuthStore } from '@/shared/stores/auth-store'
import { meQueryOptions } from './queries'
import { useLogout } from './use-logout'

// Fake the API module so logout resolves without a server. Hoisted so the
// mock factory can reference the spy (vitest hoists vi.mock above imports).
const { logoutSpy } = vi.hoisted(() => ({ logoutSpy: vi.fn() }))
vi.mock('@/shared/api/api', () => ({
  api: { auth: { logout: logoutSpy }, users: { list: vi.fn() } },
  ApiError: class ApiError extends Error {},
}))

function wrap(qc: QueryClient) {
  return ({ children }: { children: ReactNode }) => (
    <QueryClientProvider client={qc}>{children}</QueryClientProvider>
  )
}

afterEach(() => {
  vi.clearAllMocks()
  useAuthStore.setState({ user: null })
})

describe('useLogout', () => {
  it('clears the me + users caches, resets auth, and calls onLoggedOut', async () => {
    logoutSpy.mockResolvedValue(undefined)
    const qc = new QueryClient()
    qc.setQueryData(meQueryOptions.queryKey, { id: 1, email: 'a@b.c', name: null })
    qc.setQueryData(['users'], [{ id: 1 }])
    useAuthStore.setState({ user: { id: 1, email: 'a@b.c', name: null } })
    const onLoggedOut = vi.fn()

    const { result } = renderHook(() => useLogout(onLoggedOut), {
      wrapper: wrap(qc),
    })
    await act(async () => {
      await result.current.mutateAsync()
    })

    expect(logoutSpy).toHaveBeenCalledOnce()
    expect(qc.getQueryData(meQueryOptions.queryKey)).toBeNull()
    expect(qc.getQueryData(['users'])).toBeUndefined()
    expect(useAuthStore.getState().user).toBeNull()
    await waitFor(() => expect(onLoggedOut).toHaveBeenCalledOnce())
  })
})
