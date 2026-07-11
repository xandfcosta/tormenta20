import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { ApiError } from '@/shared/api/api'
import {
  characterOptionsQueryOptions,
  characterQueryOptions,
  charactersQueryOptions,
  meQueryOptions,
  usersQueryOptions,
} from './queries'

/**
 * TanStack queryOptions wrappers — pin the queryKey shape (which the
 * mutation hooks invalidate against) and the meQuery's 401-to-null
 * fallthrough.
 */
type FetchFn = typeof globalThis.fetch
let fetchSpy: ReturnType<typeof vi.fn<FetchFn>>

beforeEach(() => {
  fetchSpy = vi.fn<FetchFn>()
  vi.stubGlobal('fetch', fetchSpy)
})
afterEach(() => {
  vi.unstubAllGlobals()
})

function jsonResponse(body: unknown, status = 200): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: { 'Content-Type': 'application/json' },
  })
}

describe('queryKey shapes', () => {
  it('meQuery uses ["auth", "me"]', () => {
    expect(meQueryOptions.queryKey).toEqual(['auth', 'me'])
  })

  it('usersQuery uses ["users"]', () => {
    expect(usersQueryOptions.queryKey).toEqual(['users'])
  })

  it('charactersQuery uses ["characters"]', () => {
    expect(charactersQueryOptions.queryKey).toEqual(['characters'])
  })

  it('characterOptionsQuery uses ["characters", "options"]', () => {
    expect(characterOptionsQueryOptions.queryKey).toEqual([
      'characters',
      'options',
    ])
  })

  it('characterQuery(id) uses ["characters", id]', () => {
    expect(characterQueryOptions(42).queryKey).toEqual(['characters', 42])
  })
})

describe('characterOptionsQuery — cache control', () => {
  it('has staleTime=Infinity (catalog is immutable in-session)', () => {
    expect(characterOptionsQueryOptions.staleTime).toBe(Infinity)
  })
})

describe('meQuery — 401 → null swallow', () => {
  it('returns null when the server responds 401 Unauthorized', async () => {
    fetchSpy.mockResolvedValueOnce(
      jsonResponse({ message: 'Unauthorized' }, 401),
    )
    const result = await meQueryOptions.queryFn!({} as never)
    expect(result).toBeNull()
  })

  it('rethrows ApiError when status is not 401', async () => {
    fetchSpy.mockResolvedValueOnce(jsonResponse({ message: 'Boom' }, 500))
    await expect(meQueryOptions.queryFn!({} as never)).rejects.toBeInstanceOf(
      ApiError,
    )
  })

  it('returns the AuthUser on 200 OK', async () => {
    fetchSpy.mockResolvedValueOnce(
      jsonResponse({ id: 1, email: 'a@b.com', name: 'A' }),
    )
    const result = await meQueryOptions.queryFn!({} as never)
    expect(result).toEqual({ id: 1, email: 'a@b.com', name: 'A' })
  })

  it('retry=false to avoid retrying unauthenticated probes', () => {
    expect(meQueryOptions.retry).toBe(false)
  })
})
