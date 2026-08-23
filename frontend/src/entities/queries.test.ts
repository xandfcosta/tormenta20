import { QueryClient } from '@tanstack/solid-query'
import { describe, expect, it } from 'vitest'
import { createApiClient } from '@/shared/api/api'
import { FakeFetch } from '@/shared/api/fake-fetch'
import {
  campaignMembersQueryOptions,
  campaignQueryOptions,
  campaignsQueryOptions,
} from './campaign/queries'
import {
  characterOptionsQueryOptions,
  characterQueryOptions,
  charactersQueryOptions,
} from './character/queries'
import { campaignSessionQueryOptions, campaignSessionsQueryOptions } from './session/queries'
import { fetchSessionUser, meQueryOptions } from './user/queries'

/**
 * The query keys ARE the cache contract: a mutation invalidating
 * `['campaigns', 1]` only reaches the detail if the detail registered exactly
 * that key. Nesting also matters — `['campaigns', 1, 'sessions']` must sit
 * under the campaign so invalidating the campaign sweeps its children.
 */
describe('query keys', () => {
  it('personagens: lista, opções e detalhe', () => {
    expect(charactersQueryOptions.queryKey).toEqual(['characters'])
    expect(characterOptionsQueryOptions.queryKey).toEqual(['characters', 'options'])
    expect(characterQueryOptions(7).queryKey).toEqual(['characters', 7])
  })

  it('campanhas: lista, detalhe e membros aninhados', () => {
    expect(campaignsQueryOptions.queryKey).toEqual(['campaigns'])
    expect(campaignQueryOptions(1).queryKey).toEqual(['campaigns', 1])
    expect(campaignMembersQueryOptions(1).queryKey).toEqual(['campaigns', 1, 'members'])
  })

  it('sessões ficam aninhadas na campanha', () => {
    expect(campaignSessionsQueryOptions(1).queryKey).toEqual(['campaigns', 1, 'sessions'])
    expect(campaignSessionQueryOptions(1, 4).queryKey).toEqual(['campaigns', 1, 'sessions', 4])
  })

  // As opções da criação são estáticas: refetch nelas é desperdício puro.
  it('as opções de criação nunca ficam stale', () => {
    expect(characterOptionsQueryOptions.staleTime).toBe(Number.POSITIVE_INFINITY)
  })

  it('invalidar a campanha alcança membros e sessões', () => {
    const client = new QueryClient()
    client.setQueryData(campaignQueryOptions(1).queryKey, {
      id: 1,
      ownerId: 1,
      name: 'Snapshot Test',
      description: null,
      createdAt: '2026-01-01T00:00:00.000Z',
      updatedAt: '2026-01-01T00:00:00.000Z',
    })
    client.setQueryData(campaignMembersQueryOptions(1).queryKey, [])
    client.setQueryData(campaignSessionsQueryOptions(1).queryKey, [])
    client.setQueryData(campaignsQueryOptions.queryKey, [])

    const swept = client
      .getQueryCache()
      .findAll({ queryKey: ['campaigns', 1] })
      .map((q) => q.queryKey)

    expect(swept).toHaveLength(3)
    expect(swept).toContainEqual(['campaigns', 1, 'members'])
    expect(swept).toContainEqual(['campaigns', 1, 'sessions'])
  })
})

describe('queryFn → endpoint', () => {
  it('cada query bate no caminho certo do backend', async () => {
    const http = new FakeFetch([
      FakeFetch.json([]),
      FakeFetch.json({ id: 7 }),
      FakeFetch.json([]),
      FakeFetch.json([]),
      FakeFetch.json({ id: 4 }),
    ])
    const api = createApiClient(http.fetch)

    await api.characters.list()
    await api.characters.get(7)
    await api.campaigns.list()
    await api.members.list(1)
    await api.sessions.get(1, 4)

    expect(http.calls.map((c) => c.url)).toEqual([
      '/api/characters',
      '/api/characters/7',
      '/api/campaigns',
      '/api/campaigns/1/members',
      '/api/campaigns/1/sessions/4',
    ])
  })
})

describe('fetchSessionUser', () => {
  const ALICE = { id: 1, email: 'mestre@t20.local', name: 'Mestre' }

  it('devolve o usuário da sessão', async () => {
    const client = createApiClient(new FakeFetch([FakeFetch.json(ALICE)]).fetch)
    await expect(fetchSessionUser(client)).resolves.toEqual(ALICE)
  })

  // 401 é RESPOSTA aqui, não falha: todo guard de rota lê isso, e um throw
  // deixaria o app preso numa tela de erro em vez de mandar pro /login.
  it('trata 401 como deslogado em vez de erro', async () => {
    const client = createApiClient(new FakeFetch([FakeFetch.error(401, { message: 'nope' })]).fetch)
    await expect(fetchSessionUser(client)).resolves.toBeNull()
  })

  // Um 500 NÃO é "deslogado" — engolir viraria logout silencioso a cada
  // soluço do backend.
  it('propaga falhas que não são 401', async () => {
    const client = createApiClient(new FakeFetch([FakeFetch.error(500, { message: 'boom' })]).fetch)
    await expect(fetchSessionUser(client)).rejects.toThrow('boom')
  })

  it('meQueryOptions registra a chave da sessão sem retry', () => {
    expect(meQueryOptions.queryKey).toEqual(['auth', 'me'])
    expect(meQueryOptions.retry).toBe(false)
  })
})
