import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { api } from './api'

/**
 * Coverage for the campaigns / sessions / members namespaces added in
 * Fase D1. Each spec asserts URL, method, and body — the `request`
 * helper is already covered generically in `api.test.ts`.
 */

type FetchFn = typeof globalThis.fetch

function jsonResponse(body: unknown, status = 200): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: { 'Content-Type': 'application/json' },
  })
}

let fetchSpy: ReturnType<typeof vi.fn<FetchFn>>

beforeEach(() => {
  fetchSpy = vi.fn<FetchFn>()
  vi.stubGlobal('fetch', fetchSpy)
})

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('api.characters.getSheet', () => {
  it('GET /api/characters/:id/sheet', async () => {
    fetchSpy.mockResolvedValueOnce(jsonResponse({ id: 1, computed: {} }))
    await api.characters.getSheet(7)
    expect(fetchSpy).toHaveBeenCalledWith(
      '/api/characters/7/sheet',
      expect.any(Object),
    )
  })
})

describe('api.characters.campaigns', () => {
  it('GET /api/characters/:id/campaigns', async () => {
    fetchSpy.mockResolvedValueOnce(jsonResponse([]))
    await api.characters.campaigns(7)
    expect(fetchSpy).toHaveBeenCalledWith(
      '/api/characters/7/campaigns',
      expect.any(Object),
    )
  })
})

describe('api.campaigns', () => {
  it('list → GET /api/campaigns', async () => {
    fetchSpy.mockResolvedValueOnce(jsonResponse([]))
    await api.campaigns.list()
    expect(fetchSpy).toHaveBeenCalledWith('/api/campaigns', expect.any(Object))
  })

  it('get(id) → GET /api/campaigns/:id', async () => {
    fetchSpy.mockResolvedValueOnce(jsonResponse({ id: 1 }))
    await api.campaigns.get(1)
    expect(fetchSpy).toHaveBeenCalledWith('/api/campaigns/1', expect.any(Object))
  })

  it('create → POST /api/campaigns with body', async () => {
    fetchSpy.mockResolvedValueOnce(jsonResponse({ id: 1 }))
    await api.campaigns.create({ name: 'Nova' })
    const [, init] = fetchSpy.mock.calls[0]!
    expect(init?.method).toBe('POST')
    expect(init?.body).toBe(JSON.stringify({ name: 'Nova' }))
  })

  it('update → PATCH /api/campaigns/:id with body', async () => {
    fetchSpy.mockResolvedValueOnce(jsonResponse({ id: 1 }))
    await api.campaigns.update(7, { description: 'x' })
    const [url, init] = fetchSpy.mock.calls[0]!
    expect(url).toBe('/api/campaigns/7')
    expect(init?.method).toBe('PATCH')
  })

  it('delete → DELETE /api/campaigns/:id', async () => {
    fetchSpy.mockResolvedValueOnce(jsonResponse({ id: 7 }))
    await api.campaigns.delete(7)
    const [url, init] = fetchSpy.mock.calls[0]!
    expect(url).toBe('/api/campaigns/7')
    expect(init?.method).toBe('DELETE')
  })
})

describe('api.sessions', () => {
  it('list → GET /api/campaigns/:cid/sessions', async () => {
    fetchSpy.mockResolvedValueOnce(jsonResponse([]))
    await api.sessions.list(3)
    expect(fetchSpy).toHaveBeenCalledWith(
      '/api/campaigns/3/sessions',
      expect.any(Object),
    )
  })

  it('get → GET nested id', async () => {
    fetchSpy.mockResolvedValueOnce(jsonResponse({ id: 1 }))
    await api.sessions.get(3, 5)
    expect(fetchSpy).toHaveBeenCalledWith(
      '/api/campaigns/3/sessions/5',
      expect.any(Object),
    )
  })

  it('create → POST with { sessionNumber, title, notes }', async () => {
    fetchSpy.mockResolvedValueOnce(jsonResponse({ id: 1 }))
    await api.sessions.create(3, { sessionNumber: 2, title: 'Sessão 2' })
    const [url, init] = fetchSpy.mock.calls[0]!
    expect(url).toBe('/api/campaigns/3/sessions')
    expect(init?.method).toBe('POST')
    expect(init?.body).toContain('"sessionNumber":2')
  })

  it('start → POST /:id/start', async () => {
    fetchSpy.mockResolvedValueOnce(jsonResponse({ id: 1 }))
    await api.sessions.start(3, 5)
    const [url, init] = fetchSpy.mock.calls[0]!
    expect(url).toBe('/api/campaigns/3/sessions/5/start')
    expect(init?.method).toBe('POST')
  })

  it('end → POST /:id/end', async () => {
    fetchSpy.mockResolvedValueOnce(jsonResponse({ id: 1 }))
    await api.sessions.end(3, 5)
    const [url, init] = fetchSpy.mock.calls[0]!
    expect(url).toBe('/api/campaigns/3/sessions/5/end')
    expect(init?.method).toBe('POST')
  })

  it('clearTracker → POST /:id/clear-tracker', async () => {
    fetchSpy.mockResolvedValueOnce(jsonResponse({ id: 5 }))
    await api.sessions.clearTracker(3, 5)
    const [url, init] = fetchSpy.mock.calls[0]!
    expect(url).toBe('/api/campaigns/3/sessions/5/clear-tracker')
    expect(init?.method).toBe('POST')
  })

  it('delete → DELETE nested id', async () => {
    fetchSpy.mockResolvedValueOnce(jsonResponse({ id: 5 }))
    await api.sessions.delete(3, 5)
    const [url, init] = fetchSpy.mock.calls[0]!
    expect(url).toBe('/api/campaigns/3/sessions/5')
    expect(init?.method).toBe('DELETE')
  })
})

describe('api.members', () => {
  it('list → GET /:cid/members', async () => {
    fetchSpy.mockResolvedValueOnce(jsonResponse([]))
    await api.members.list(3)
    expect(fetchSpy).toHaveBeenCalledWith(
      '/api/campaigns/3/members',
      expect.any(Object),
    )
  })

  it('add → POST with { characterId, role }', async () => {
    fetchSpy.mockResolvedValueOnce(jsonResponse({ id: 1 }))
    await api.members.add(3, { characterId: 10, role: 'gm' })
    const [url, init] = fetchSpy.mock.calls[0]!
    expect(url).toBe('/api/campaigns/3/members')
    expect(init?.method).toBe('POST')
    expect(init?.body).toBe(
      JSON.stringify({ characterId: 10, role: 'gm' }),
    )
  })

  it('updateRole → PATCH /:cid/members/:id', async () => {
    fetchSpy.mockResolvedValueOnce(jsonResponse({ id: 1 }))
    await api.members.updateRole(3, 5, { role: 'player' })
    const [url, init] = fetchSpy.mock.calls[0]!
    expect(url).toBe('/api/campaigns/3/members/5')
    expect(init?.method).toBe('PATCH')
  })

  it('remove → DELETE /:cid/members/:id', async () => {
    fetchSpy.mockResolvedValueOnce(jsonResponse({ id: 5 }))
    await api.members.remove(3, 5)
    const [url, init] = fetchSpy.mock.calls[0]!
    expect(url).toBe('/api/campaigns/3/members/5')
    expect(init?.method).toBe('DELETE')
  })
})
