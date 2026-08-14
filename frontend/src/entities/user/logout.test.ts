import { QueryClient } from '@tanstack/solid-query'
import { describe, expect, it } from 'vitest'
import { createApiClient } from '@/shared/api/api'
import { FakeFetch } from '@/shared/api/fake-fetch'
import { createAuthStore } from '@/shared/stores/auth-store'
import { logout } from './logout'
import { meQueryOptions } from './queries'

const ALICE = { id: 1, email: 'mestre@t20.local', name: 'Mestre', isAdmin: false }

function loggedInWorld() {
  const http = new FakeFetch([FakeFetch.empty()])
  const queryClient = new QueryClient()
  queryClient.setQueryData(meQueryOptions.queryKey, ALICE)
  queryClient.setQueryData(['users'], [ALICE])
  return { http, queryClient, auth: createAuthStore(ALICE), client: createApiClient(http.fetch) }
}

describe('logout', () => {
  it('chama POST /auth/logout', async () => {
    const world = loggedInWorld()
    await logout(world)
    expect(world.http.onlyCall.url).toBe('/api/auth/logout')
    expect(world.http.onlyCall.init?.method).toBe('POST')
  })

  // Regression: clearing only the auth store left the `me` query cached, so the
  // /login guard read a live user and bounced straight back to the Hub — the
  // user appeared to be unable to log out.
  it('zera o cache do `me`, senão o guard devolve o usuário pra dentro', async () => {
    const world = loggedInWorld()
    await logout(world)
    expect(world.queryClient.getQueryData(meQueryOptions.queryKey)).toBeNull()
  })

  it('descarta a lista de usuários (dado de outra conta)', async () => {
    const world = loggedInWorld()
    await logout(world)
    expect(world.queryClient.getQueryData(['users'])).toBeUndefined()
  })

  it('esvazia o auth store', async () => {
    const world = loggedInWorld()
    await logout(world)
    expect(world.auth.user()).toBeNull()
    expect(world.auth.isAuthenticated()).toBe(false)
  })

  it('não apaga a sessão local se o servidor recusar o logout', async () => {
    const world = loggedInWorld()
    const failing = createApiClient(new FakeFetch([FakeFetch.error(500, { message: 'boom' })]).fetch)

    await expect(logout({ ...world, client: failing })).rejects.toThrow('boom')

    expect(world.auth.user()).toEqual(ALICE)
    expect(world.queryClient.getQueryData(meQueryOptions.queryKey)).toEqual(ALICE)
  })
})
