import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { ApiError, api } from './api'

/**
 * The `request` helper isn't exported, so cover it via the `api`
 * namespace's call sites. Mocks `globalThis.fetch` to inspect:
 *  - URL prefix '/api'
 *  - credentials + JSON content-type
 *  - 2xx body parsing
 *  - 204 → undefined
 *  - 4xx/5xx → ApiError with message + fieldErrors
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

describe('ApiError', () => {
  it('carries status, message, and fieldErrors', () => {
    const err = new ApiError(400, 'Bad', { name: ['required'] })
    expect(err.status).toBe(400)
    expect(err.message).toBe('Bad')
    expect(err.fieldErrors).toEqual({ name: ['required'] })
    expect(err.name).toBe('ApiError')
    expect(err).toBeInstanceOf(Error)
  })

  it('defaults fieldErrors to empty object', () => {
    const err = new ApiError(500, 'Oops')
    expect(err.fieldErrors).toEqual({})
  })
})

describe('request — happy path', () => {
  it('prefixes the URL with /api', async () => {
    fetchSpy.mockResolvedValueOnce(jsonResponse({ ok: true }))
    await api.users.list()
    expect(fetchSpy).toHaveBeenCalledWith('/api/users', expect.any(Object))
  })

  it('sends credentials and JSON content-type by default', async () => {
    fetchSpy.mockResolvedValueOnce(jsonResponse({ ok: true }))
    await api.users.list()
    const [, init] = fetchSpy.mock.calls[0]!
    expect(init?.credentials).toBe('include')
    const headers = init?.headers as Record<string, string>
    expect(headers['Content-Type']).toBe('application/json')
  })

  it('parses the JSON body for 2xx responses', async () => {
    fetchSpy.mockResolvedValueOnce(jsonResponse([{ id: 1, email: 'a@b.com' }]))
    const result = await api.users.list()
    expect(result).toEqual([{ id: 1, email: 'a@b.com' }])
  })

  it('returns undefined for 204 No Content', async () => {
    fetchSpy.mockResolvedValueOnce(new Response(null, { status: 204 }))
    await expect(api.auth.logout()).resolves.toBeUndefined()
  })
})

describe('request — POST body + custom headers', () => {
  it('serializes the body and forwards method', async () => {
    fetchSpy.mockResolvedValueOnce(jsonResponse({ id: 1, email: 'a@b.com', name: null }))
    await api.auth.register({ email: 'a@b.com', password: 'pw' })
    const [, init] = fetchSpy.mock.calls[0]!
    expect(init?.method).toBe('POST')
    expect(init?.body).toBe(JSON.stringify({ email: 'a@b.com', password: 'pw' }))
  })
})

describe('request — error path', () => {
  it('throws ApiError with status + message + fieldErrors', async () => {
    fetchSpy.mockResolvedValueOnce(
      jsonResponse(
        {
          message: 'Validation failed',
          fieldErrors: { name: ['Name is required'] },
        },
        400,
      ),
    )
    try {
      await api.users.list()
      throw new Error('should have thrown')
    } catch (err) {
      expect(err).toBeInstanceOf(ApiError)
      const e = err as ApiError
      expect(e.status).toBe(400)
      expect(e.message).toBe('Validation failed')
      expect(e.fieldErrors).toEqual({ name: ['Name is required'] })
    }
  })

  it('joins array messages with semicolons', async () => {
    fetchSpy.mockResolvedValueOnce(
      jsonResponse({ message: ['one', 'two'] }, 400),
    )
    try {
      await api.users.list()
      throw new Error('should have thrown')
    } catch (err) {
      expect((err as ApiError).message).toBe('one; two')
    }
  })

  it('falls back to "status statusText" when body has no message', async () => {
    fetchSpy.mockResolvedValueOnce(
      new Response('Internal Server Error', {
        status: 500,
        statusText: 'Internal Server Error',
      }),
    )
    try {
      await api.users.list()
      throw new Error('should have thrown')
    } catch (err) {
      expect((err as ApiError).status).toBe(500)
      expect((err as ApiError).message).toMatch(/500/)
    }
  })

  it('survives a non-JSON error body without crashing', async () => {
    fetchSpy.mockResolvedValueOnce(new Response('not json', { status: 502 }))
    try {
      await api.users.list()
      throw new Error('should have thrown')
    } catch (err) {
      expect(err).toBeInstanceOf(ApiError)
      expect((err as ApiError).status).toBe(502)
    }
  })
})

describe('api namespace — URL paths', () => {
  beforeEach(() => {
    // Fresh Response per call — Response bodies are single-read, so a
    // sticky mockResolvedValue handing back the same instance fails on
    // the second await.
    fetchSpy.mockImplementation(() => Promise.resolve(jsonResponse({})))
  })

  it('characters.get encodes the id in the URL', async () => {
    await api.characters.get(42)
    expect(fetchSpy).toHaveBeenCalledWith('/api/characters/42', expect.any(Object))
  })

  it('characters.deleteExpertise URL-encodes the name (spaces, accents)', async () => {
    await api.characters.deleteExpertise(1, 'Atuação Forte')
    expect(fetchSpy).toHaveBeenCalledWith(
      '/api/characters/1/expertises/Atua%C3%A7%C3%A3o%20Forte',
      expect.any(Object),
    )
  })

  it('characters.consumeItem POSTs to the correct path', async () => {
    await api.characters.consumeItem(1, 5)
    const [url, init] = fetchSpy.mock.calls[0]!
    expect(url).toBe('/api/characters/1/items/5/consume')
    expect(init?.method).toBe('POST')
  })

  it('characters.endScene/endDay POST without body', async () => {
    await api.characters.endScene(7)
    expect(fetchSpy).toHaveBeenLastCalledWith(
      '/api/characters/7/end-scene',
      expect.objectContaining({ method: 'POST' }),
    )
    await api.characters.endDay(7)
    expect(fetchSpy).toHaveBeenLastCalledWith(
      '/api/characters/7/end-day',
      expect.objectContaining({ method: 'POST' }),
    )
  })
})
