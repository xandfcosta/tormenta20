export type User = {
  id: number
  email: string
  name: string | null
  createdAt: string
}

export type AuthUser = {
  id: number
  email: string
  name: string | null
}

import type { AttributeKey } from '@tormenta20/t20-data'

export type { AttributeKey }

export type ExpertiseDef = { name: string; attribute: AttributeKey }

export type CharacterExpertise = {
  name: string
  attribute: AttributeKey
  trained: boolean
  custom: boolean
}

export type CreateExpertiseInput = {
  name: string
  attribute: AttributeKey
}

export type EquippedSlot = 'vested' | 'wielded' | 'wielded2'

export type CharacterItem = {
  id: number
  catalogId: string | null
  name: string
  quantity: number
  slots: number
  equipped: EquippedSlot | null
  /** JSON-encoded string[] of improvement catalog ids */
  improvements: string
  material: string | null
}

export type CreateItemInput = {
  catalogId?: string
  name?: string
  quantity: number
  slots?: number
  equipped?: EquippedSlot
  improvements?: string[]
  material?: string
}

export type UpdateItemInput = {
  name?: string
  quantity?: number
  slots?: number
  equipped?: EquippedSlot | null
  improvements?: string[]
  material?: string | null
}

export type ActiveEffect = {
  id: number
  catalogId: string
  scope: 'scene' | 'day'
  /** JSON-encoded Modifier[] copied from catalog at consume time */
  modifiers: string
  createdAt: string
}

export type Character = {
  id: number
  ownerId: number
  name: string
  origin: string
  god: string | null
  level: number
  hpMax: number
  hpCurrent: number
  mpMax: number
  mpCurrent: number
  strength: number
  dexterity: number
  constitution: number
  intelligence: number
  wisdom: number
  charisma: number
  size: string
  displacement: number
  /** JSON-encoded string[] of proficiency category ids */
  proficiencies: string
  /** JSON-encoded string[] of race ability variant ids the player picked */
  raceAbilityChoices: string
  /** JSON-encoded string[] of origin benefit ids the player picked */
  originChoices: string
  /** JSON-encoded string[] of class power ids the character owns */
  classPowers: string
  createdAt: string
  updatedAt: string
  races: { race: string }[]
  classes: { className: string; level: number }[]
  expertises: CharacterExpertise[]
  items: CharacterItem[]
  activeEffects: ActiveEffect[]
}

export type UpdateProficienciesInput = {
  proficiencies: string[]
}

export type UpdateLevelInput = {
  level: number
}

export type UpdateAbilityChoicesInput = {
  raceAbilityChoices?: string[]
  originChoices?: string[]
  classPowers?: string[]
}

export type UpdateClassLevelInput = {
  className: string
  level: number
}

export type CharacterOptions = {
  races: string[]
  classes: string[]
  origins: string[]
  gods: string[]
  sizes: string[]
  expertises: ExpertiseDef[]
}

export type UpdateExpertiseInput = {
  name: string
  attribute?: AttributeKey
  trained?: boolean
}

export type UpdateVitalsInput = {
  hpCurrent?: number
  mpCurrent?: number
}

export type ConsumeItemInput = {
  hpRolled?: number
  mpRolled?: number
}

export type CreateCharacterInput = {
  name: string
  races: string[]
  origin: string
  classes: { className: string; level: number }[]
  god?: string
  hpMax: number
  hpCurrent: number
  mpMax: number
  mpCurrent: number
  strength: number
  dexterity: number
  constitution: number
  intelligence: number
  wisdom: number
  charisma: number
  size: string
  displacement: number
}

const API_BASE = '/api'

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

type ErrorBody = {
  message?: string | string[]
  fieldErrors?: FieldErrorMap
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(`${API_BASE}${path}`, {
    ...init,
    credentials: 'include',
    headers: { 'Content-Type': 'application/json', ...(init?.headers ?? {}) },
  })
  if (!res.ok) {
    const body = (await res.json().catch(() => null)) as ErrorBody | null
    const raw = body?.message
    const message = Array.isArray(raw) ? raw.join('; ') : raw ?? `${res.status} ${res.statusText}`
    throw new ApiError(res.status, message, body?.fieldErrors ?? {})
  }
  if (res.status === 204) return undefined as T
  return res.json() as Promise<T>
}

export const api = {
  auth: {
    register: (input: { email: string; password: string; name?: string }) =>
      request<AuthUser>('/auth/register', { method: 'POST', body: JSON.stringify(input) }),
    login: (input: { email: string; password: string }) =>
      request<AuthUser>('/auth/login', { method: 'POST', body: JSON.stringify(input) }),
    logout: () => request<void>('/auth/logout', { method: 'POST' }),
    me: () => request<AuthUser>('/auth/me'),
  },
  users: {
    list: () => request<User[]>('/users'),
  },
  characters: {
    options: () => request<CharacterOptions>('/characters/options'),
    list: () => request<Character[]>('/characters'),
    get: (id: number) => request<Character>(`/characters/${id}`),
    create: (input: CreateCharacterInput) =>
      request<Character>('/characters', { method: 'POST', body: JSON.stringify(input) }),
    updateVitals: (id: number, input: UpdateVitalsInput) =>
      request<Character>(`/characters/${id}/vitals`, {
        method: 'PATCH',
        body: JSON.stringify(input),
      }),
    updateExpertise: (id: number, input: UpdateExpertiseInput) =>
      request<CharacterExpertise>(`/characters/${id}/expertises`, {
        method: 'PATCH',
        body: JSON.stringify(input),
      }),
    addExpertise: (id: number, input: CreateExpertiseInput) =>
      request<CharacterExpertise>(`/characters/${id}/expertises`, {
        method: 'POST',
        body: JSON.stringify(input),
      }),
    deleteExpertise: (id: number, name: string) =>
      request<{ name: string }>(
        `/characters/${id}/expertises/${encodeURIComponent(name)}`,
        { method: 'DELETE' },
      ),
    addItem: (id: number, input: CreateItemInput) =>
      request<CharacterItem>(`/characters/${id}/items`, {
        method: 'POST',
        body: JSON.stringify(input),
      }),
    updateItem: (id: number, itemId: number, input: UpdateItemInput) =>
      request<CharacterItem>(`/characters/${id}/items/${itemId}`, {
        method: 'PATCH',
        body: JSON.stringify(input),
      }),
    deleteItem: (id: number, itemId: number) =>
      request<{ id: number }>(`/characters/${id}/items/${itemId}`, {
        method: 'DELETE',
      }),
    consumeItem: (id: number, itemId: number, input: ConsumeItemInput = {}) =>
      request<Character>(`/characters/${id}/items/${itemId}/consume`, {
        method: 'POST',
        body: JSON.stringify(input),
      }),
    removeActiveEffect: (id: number, effectId: number) =>
      request<{ id: number }>(`/characters/${id}/active-effects/${effectId}`, {
        method: 'DELETE',
      }),
    endScene: (id: number) =>
      request<Character>(`/characters/${id}/end-scene`, { method: 'POST' }),
    endDay: (id: number) =>
      request<Character>(`/characters/${id}/end-day`, { method: 'POST' }),
    updateProficiencies: (id: number, input: UpdateProficienciesInput) =>
      request<Character>(`/characters/${id}/proficiencies`, {
        method: 'PATCH',
        body: JSON.stringify(input),
      }),
    updateLevel: (id: number, input: UpdateLevelInput) =>
      request<Character>(`/characters/${id}/level`, {
        method: 'PATCH',
        body: JSON.stringify(input),
      }),
    updateAbilityChoices: (id: number, input: UpdateAbilityChoicesInput) =>
      request<Character>(`/characters/${id}/abilities`, {
        method: 'PATCH',
        body: JSON.stringify(input),
      }),
    updateClassLevel: (id: number, input: UpdateClassLevelInput) =>
      request<Character>(`/characters/${id}/classes/level`, {
        method: 'PATCH',
        body: JSON.stringify(input),
      }),
  },
}
