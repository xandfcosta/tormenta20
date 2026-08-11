/**
 * HTTP client for the Go backend. Framework-agnostic on purpose — this file
 * carries no Solid import and ports ~1:1 from the React app. It grows one
 * domain at a time as scenes are ported (ALE-63).
 *
 * Only the READ endpoints the query layer needs are here; each scene brings its
 * own mutations when it lands, so the client never gets ahead of its consumers.
 */
import type {
  ActivationSpec,
  CatalogItem,
  CatalogSpell,
  ClassPower,
  Condition,
  ConditionId,
  Deus,
  GeneralPower,
  GrantedPower,
  GrantedPowerOption,
  Origem,
  OriginDefinition,
  Raca,
  TormentaPower,
  TormentaPowerId,
} from '@tormenta20/t20-data'
import type {
  AddMemberInput,
  ApplyDamageResult,
  ApplyEffectInput,
  ApplyEffectResult,
  AuthUser,
  Campaign,
  CampaignInvitePreview,
  CampaignInviteToken,
  CampaignMember,
  Character,
  CharacterExpertise,
  CharacterItem,
  CharacterOptions,
  ClassLevelResult,
  ConsumeItemInput,
  ConsumeItemResult,
  ComputedSheetV2,
  CreateCampaignInput,
  CreateExpertiseInput,
  CreateItemInput,
  CreateSessionInput,
  ProficienciesResult,
  Credentials,
  RaceDefinition,
  Session,
  UpdateCampaignInput,
  UpdateExpertiseInput,
  UpdateClassLevelInput,
  UpdateItemInput,
  UpdateProficienciesInput,
  UpdateVitalsInput,
  User,
  VitalsResult,
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
  const patch = (body: unknown): RequestInit => ({
    method: 'PATCH',
    body: JSON.stringify(body),
  })
  const del: RequestInit = { method: 'DELETE' }

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
      /** Trains/untrains a perícia or rekeys it to another attribute. */
      updateExpertise: (id: number, input: UpdateExpertiseInput) =>
        request<CharacterExpertise>(`/characters/${id}/expertises`, patch(input)),
      /** A custom "ofício" the player invented. */
      addExpertise: (id: number, input: CreateExpertiseInput) =>
        request<CharacterExpertise>(`/characters/${id}/expertises`, json(input)),
      deleteExpertise: (id: number, name: string) =>
        request<{ name: string }>(
          `/characters/${id}/expertises/${encodeURIComponent(name)}`,
          del,
        ),
      addItem: (id: number, input: CreateItemInput) =>
        request<CharacterItem>(`/characters/${id}/items`, json(input)),
      updateItem: (id: number, itemId: number, input: UpdateItemInput) =>
        request<CharacterItem>(`/characters/${id}/items/${itemId}`, patch(input)),
      deleteItem: (id: number, itemId: number) =>
        request<{ id: number }>(`/characters/${id}/items/${itemId}`, del),
      /** Spends one unit; the server answers a DELTA, not the whole character. */
      consumeItem: (id: number, itemId: number, input?: ConsumeItemInput) =>
        request<ConsumeItemResult>(`/characters/${id}/items/${itemId}/consume`, json(input ?? {})),
      /** PV/PM edit. Answers the CLAMPED pair, so the client takes the server's word. */
      updateVitals: (id: number, input: UpdateVitalsInput) =>
        request<VitalsResult>(`/characters/${id}/vitals`, patch(input)),
      /**
       * Damage in ONE request: the server routes it temp-PV-first and answers
       * what it drained. Two requests (drain, then subtract) could interleave
       * with another client's hit and lose a pool.
       */
      applyDamage: (id: number, amount: number) =>
        request<ApplyDamageResult>(`/characters/${id}/damage`, json({ amount })),
      /** Replaces the proficiency set (weapons, armor, shields). */
      updateProficiencies: (id: number, input: UpdateProficienciesInput) =>
        request<ProficienciesResult>(`/characters/${id}/proficiencies`, patch(input)),
      /** Level one class up or down; answers the new totals AND the resynced
       *  PV/PM pools, which are derived from class levels. */
      updateClassLevel: (id: number, input: UpdateClassLevelInput) =>
        request<ClassLevelResult>(`/characters/${id}/classes/level`, patch(input)),
      /** Spell buff, power grant, or the GM's manual temp-PV pool. */
      applyEffect: (id: number, input: ApplyEffectInput) =>
        request<ApplyEffectResult>(`/characters/${id}/active-effects`, json(input)),
    },
    catalog: {
      // Static rulebook reference; cached hard (staleTime ∞) and fetched instead
      // of bundled — the front ships no catalog DATA
      // (project_front_decouple_catalog).
      spells: () => request<Record<string, CatalogSpell>>('/catalog/spells'),
      items: () => request<CatalogItem[]>('/catalog/items'),
      /** RaceDefinition catalog (innate abilities) — DISTINCT from `races`. */
      raceDefs: () => request<RaceDefinition[]>('/catalog/race-defs'),
      origins: () => request<OriginDefinition[]>('/catalog/origins'),
      classPowers: () => request<ClassPower[]>('/catalog/class-powers'),
      generalPowers: () => request<GeneralPower[]>('/catalog/general-powers'),
      deuses: () => request<Deus[]>('/catalog/deuses'),
      grantedPowers: () => request<GrantedPower[]>('/catalog/granted-powers'),
      /** racas.ts RACAS (movement/size/attr-mod) — DISTINCT from `raceDefs`. */
      races: () => request<Record<string, Raca>>('/catalog/races'),
      origens: () => request<Record<string, Origem>>('/catalog/origens'),
      conditions: () => request<Record<ConditionId, Condition>>('/catalog/conditions'),
      tormentaPowers: () =>
        request<Record<TormentaPowerId, TormentaPower>>('/catalog/tormenta-powers'),
      divinePowers: () => request<GrantedPowerOption[]>('/catalog/divine-powers'),
      activations: () => request<ActivationSpec[]>('/catalog/activations'),
    },
    campaigns: {
      list: () => request<Campaign[]>('/campaigns'),
      get: (id: number) => request<Campaign>(`/campaigns/${id}`),
      create: (input: CreateCampaignInput) => request<Campaign>('/campaigns', json(input)),
      update: (id: number, input: UpdateCampaignInput) =>
        request<Campaign>(`/campaigns/${id}`, patch(input)),
      delete: (id: number) => request<{ id: number }>(`/campaigns/${id}`, del),
      /** Mints a fresh invite token — any link shared before this 404s. */
      rotateInvite: (id: number) =>
        request<CampaignInviteToken>(`/campaigns/${id}/invite`, { method: 'POST' }),
    },
    invites: {
      /** Public: resolves a shared token to the campaign it invites into. */
      resolve: (token: string) =>
        request<CampaignInvitePreview>(`/invites/${encodeURIComponent(token)}`),
    },
    members: {
      list: (campaignId: number) => request<CampaignMember[]>(`/campaigns/${campaignId}/members`),
      /** Self-join: the caller must own `characterId`. */
      add: (campaignId: number, input: AddMemberInput) =>
        request<CampaignMember>(`/campaigns/${campaignId}/members`, json(input)),
      /** `id` is the MEMBER's id, not the character's. */
      remove: (campaignId: number, id: number) =>
        request<{ id: number }>(`/campaigns/${campaignId}/members/${id}`, del),
    },
    sessions: {
      list: (campaignId: number) => request<Session[]>(`/campaigns/${campaignId}/sessions`),
      get: (campaignId: number, id: number) =>
        request<Session>(`/campaigns/${campaignId}/sessions/${id}`),
      create: (campaignId: number, input: CreateSessionInput) =>
        request<Session>(`/campaigns/${campaignId}/sessions`, json(input)),
    },
  }
}

export type ApiClient = ReturnType<typeof createApiClient>

/** The app-wide client. Tests build their own with `createApiClient`. */
export const api: ApiClient = createApiClient()
