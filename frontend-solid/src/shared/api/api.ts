/**
 * HTTP client for the Go backend. Framework-agnostic on purpose — this file
 * carries no Solid import and ports ~1:1 from the React app. It grows one
 * domain at a time as scenes are ported (ALE-63).
 *
 * Only the READ endpoints the query layer needs are here; each scene brings its
 * own mutations when it lands, so the client never gets ahead of its consumers.
 */
import type {
  AuthUser,
  Campaign,
  CampaignMember,
  Character,
  CharacterOptions,
  ComputedSheetV2,
  Credentials,
  RaceDefinition,
  Session,
  User,
} from './types'

export * from './types'

export type FieldErrorMap = Record<string, string[]>

export class ApiError extends Error {
  status: number
  fieldErrors: FieldErrorMap

  constructor(status: number, message: string, fieldErrors: FieldErrorMap = {}) {
    super(message)
    this.name = 'ApiError'
    this.status = status
    this.fieldErrors = fieldErrors
  }
}

// Vite proxies /api to the Go server (:3001), stripping the prefix.
const API_BASE = '/api'

type ErrorBody = {
  message?: string | string[]
  fieldErrors?: FieldErrorMap
}

/** Reads the backend's error shape, falling back to the bare HTTP status. */
async function toApiError(res: Response): Promise<ApiError> {
  const body = (await res.json().catch(() => null)) as ErrorBody | null
  const raw = body?.message
  const message = Array.isArray(raw) ? raw.join('; ') : (raw ?? `${res.status} ${res.statusText}`)
  return new ApiError(res.status, message, body?.fieldErrors ?? {})
}

function createRequest(fetchImpl: typeof globalThis.fetch) {
  return async function request<T>(path: string, init?: RequestInit): Promise<T> {
    const res = await fetchImpl(`${API_BASE}${path}`, {
      ...init,
      // The session is an httpOnly cookie — without this every call is a 401.
      credentials: 'include',
      headers: { 'Content-Type': 'application/json', ...(init?.headers ?? {}) },
    })
    if (!res.ok) throw await toApiError(res)
    if (res.status === 204) return undefined as T
    return res.json() as Promise<T>
  }
}

/**
 * Builds a client over an injected `fetch`, so tests drive it with a FakeFetch
 * instead of patching a global.
 *
 * @example const api = createApiClient(new FakeFetch([...]).fetch)
 */
export function createApiClient(fetchImpl: typeof globalThis.fetch = globalThis.fetch) {
  const request = createRequest(fetchImpl)
  const json = (body: unknown): RequestInit => ({
    method: 'POST',
    body: JSON.stringify(body),
  })

  return {
    auth: {
      register: (input: Credentials & { name?: string }) =>
        request<AuthUser>('/auth/register', json(input)),
      login: (input: Credentials) => request<AuthUser>('/auth/login', json(input)),
      logout: () => request<void>('/auth/logout', { method: 'POST' }),
      me: () => request<AuthUser>('/auth/me'),
    },
    users: {
      list: () => request<User[]>('/users'),
    },
    characters: {
      list: () => request<Character[]>('/characters'),
      get: (id: number) => request<Character>(`/characters/${id}`),
      /** The creation wizard's pick lists (races, classes, origins…). Static. */
      options: () => request<CharacterOptions>('/characters/options'),
      /** The server-computed sheet (defense, attribute totals, perícias…). */
      getSheet: (id: number) => request<ComputedSheetV2>(`/characters/${id}/sheet`),
    },
    catalog: {
      /** Race definitions (innate abilities). Static rulebook reference. */
      raceDefs: () => request<RaceDefinition[]>('/catalog/race-defs'),
    },
    campaigns: {
      list: () => request<Campaign[]>('/campaigns'),
      get: (id: number) => request<Campaign>(`/campaigns/${id}`),
    },
    members: {
      list: (campaignId: number) => request<CampaignMember[]>(`/campaigns/${campaignId}/members`),
    },
    sessions: {
      list: (campaignId: number) => request<Session[]>(`/campaigns/${campaignId}/sessions`),
      get: (campaignId: number, id: number) =>
        request<Session>(`/campaigns/${campaignId}/sessions/${id}`),
    },
  }
}

export type ApiClient = ReturnType<typeof createApiClient>

/** The app-wide client. Tests build their own with `createApiClient`. */
export const api: ApiClient = createApiClient()
