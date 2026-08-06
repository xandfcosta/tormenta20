// Type-only — erases at compile, so the spell catalog DATA no longer rides in
// the bundle; it is fetched from GET /catalog/spells and cached instead.
import type { CatalogSpell } from '@tormenta20/t20-data'

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
  /** Poder concedido escolhido ao se tornar devoto (p96); '' = não devoto. */
  godPower: string
  /** Tibares (T$). */
  tibar: number
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
  /** JSON ConditionId[] — condições do livro ativas (p394-395). */
  activeConditions: string
  /** JSON-encoded { floatingPicks?, ascendencia? } — primary-race attribute
   *  choices; stored attributes are BASE and race is derived from these. */
  raceAttributeChoices: string
  /** JSON-encoded { race, floatingPicks?, ascendencia? }[] — opted-in secondary
   *  races (GM-negotiated); their attribute mods also apply. */
  secondaryRaceChoices: string
  /** JSON-encoded string[] of origin benefit ids the player picked */
  originChoices: string
  /** JSON-encoded string[] of class power ids the character owns */
  classPowers: string
  /** JSON-encoded ClassChoices keyed by className (devoto, caminho, ...) */
  classChoices: string
  /** JSON-encoded powerId -> option id[] sub-choices (totem/school/companion/weapon) */
  powerChoices: string
  createdAt: string
  updatedAt: string
  races: { race: string }[]
  classes: { className: string; level: number }[]
  expertises: CharacterExpertise[]
  items: CharacterItem[]
  activeEffects: ActiveEffect[]
  spells: CharacterSpell[]
}

export type CharacterSpell = {
  id: number
  catalogSpellId: string
  prepared: boolean
  learnedAt: string
}

export type UpdateProficienciesInput = {
  proficiencies: string[]
}

export type UpdateLevelInput = {
  level: number
}

import type { ClassChoices } from '@tormenta20/t20-data'

export type UpdateAbilityChoicesInput = {
  raceAbilityChoices?: string[]
  originChoices?: string[]
  classPowers?: string[]
  classChoices?: ClassChoices
  powerChoices?: Record<string, string[]>
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

/** Delta returned by consumeItem (not the whole Character) — the client merges
 *  it into the cached character. `item.quantity` is the new count (0 when
 *  removed); `effect` is a scene/day ActiveEffect a non-instant consumable
 *  created; vitals are the clamped post-consume values. */
export type ConsumeItemResult = {
  item: { id: number; quantity: number; removed: boolean }
  effect: ActiveEffect | null
  hpCurrent: number
  mpCurrent: number
}

/** One of `spellId` (spell buff), `powerId` (power grant, Fase 4) or
 *  `manualTempHp` (GM ad-hoc temp-PV pool, F3 — 0 clears it). */
export type ApplyEffectInput = {
  spellId?: string
  powerId?: string
  manualTempHp?: number
  steps?: number
  scope?: 'scene' | 'day'
}

/** A pool displaced under vale-o-maior (p256): `removed` rows were deleted;
 *  kept rows (mixed buffs like Heroísmo) had their tempHp amount zeroed. */
export type DisplacedPool = { effectId: number; removed: boolean }

/** Temp-PV apply outcome: the upserted pool row + what it displaced. */
export type PoolApplied = { effect: ActiveEffect; displaced: DisplacedPool[] }

/** Vale-o-maior no-op: an existing pool was bigger or equal; nothing written. */
export type PoolSuperseded = {
  superseded: true
  keptEffectId: number
  keptAmount: number
}

/** `manualTempHp: 0` cleared the manual pool. */
export type ManualPoolCleared = { cleared: true; removedEffectIds: number[] }

/** POST :id/active-effects — spell buffs / verbatim power grants return the
 *  plain ActiveEffect row; temp-PV pool paths (temp-hp grants, manualTempHp)
 *  return one of the pool outcomes. */
export type ApplyEffectResult =
  | ActiveEffect
  | PoolApplied
  | PoolSuperseded
  | ManualPoolCleared

/** One drained pool from POST :id/damage. `removed` rows are gone; kept rows
 *  carry `newAmount` (0 for an emptied mixed buff, partial otherwise). */
export type DamageDrainStep = {
  effectId: number
  newAmount: number
  removed: boolean
}

/** POST :id/damage — atomic temp-first routing result (F2). */
export type ApplyDamageResult = {
  hpCurrent: number
  tempHpRemaining: number
  drained: DamageDrainStep[]
}

/** Delta from castSpell: new PM + catalyst effect ids the cast consumed. */
export type CastSpellResult = {
  mpCurrent: number
  removedEffectIds: number[]
}

/** Delta from endScene/endDay: which effect scopes to drop from the cache. */
export type EffectsClearedResult = {
  clearedScopes: ('scene' | 'day')[]
}

/** Deltas from the sheet-edit mutations — each carries only the fields the
 *  write touched; the client merges them into the cached Character. */
export type VitalsResult = { hpCurrent: number; mpCurrent: number }
/** Engine-derived PV/PM pools shipped with any delta whose write changes
 *  computeSheet inputs — merging them keeps hpMax/mpMax fresh without a
 *  refetch (regression 2026-08: level up/down left the pools stale). */
export type VitalsSync = {
  hpMax: number
  hpCurrent: number
  mpMax: number
  mpCurrent: number
}
export type LevelResult = { level: number; vitals: VitalsSync }
export type ClassLevelResult = {
  level: number
  classes: { className: string; level: number }[]
  vitals: VitalsSync
}
export type ProficienciesResult = { proficiencies: string }
export type AbilityChoicesResult = {
  raceAbilityChoices?: string
  originChoices?: string
  classPowers?: string
  classChoices?: string
}

export type CreateCharacterInput = {
  name: string
  races: string[]
  origin: string
  classes: { className: string; level: number }[]
  god?: string
  godPower?: string
  /** Dinheiro inicial em T$ (Tabela 3-1 p140 / 4d6). */
  tibar?: number
  /** Itens iniciais (kit p140 + itens da origem). */
  items?: {
    catalogId?: string
    name: string
    quantity?: number
    slots?: number
    equipped?: string
  }[]
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
  // Optional creation-time ability choices (mirror UpdateAbilityChoicesInput +
  // trained perícias). Absent = character starts with empty picks.
  classPowers?: string[]
  originChoices?: string[]
  classChoices?: ClassChoices
  trainedExpertises?: string[]
  powerChoices?: Record<string, string[]>
  /** Primary-race attribute choices — persisted so the racial mod is derived
   *  once from the BASE attributes (no baking). */
  raceAttributeChoices?: {
    floatingPicks?: string[]
    ascendencia?: string
    deformidade?: DeformidadeChoiceInput
  }
  /** Opted-in secondary races (GM-negotiated); their attribute mods also apply. */
  secondaryRaceChoices?: {
    race: string
    floatingPicks?: string[]
    ascendencia?: string
    deformidade?: DeformidadeChoiceInput
  }[]
}

/** Deformidade (Lefou p23): ≤2 perícias com +2, uma trocável por um poder. */
export type DeformidadeChoiceInput = {
  pericias: string[]
  tormentaPower?: string
}

import type { ComputedSheet } from '@tormenta20/t20-data'

export type { ComputedSheet }

export type CharacterWithComputed = Character & { computed: ComputedSheet }

export type Campaign = {
  id: number
  ownerId: number
  name: string
  description: string | null
  createdAt: string
  updatedAt: string
  /** Caller's role in this campaign — server resolves owner=gm vs
   * member=player on GET /campaigns and GET /campaigns/:id. Optional
   * because older cached payloads may predate it. */
  role?: CampaignMemberRole
  /** The caller's own member character in this campaign (null when they're
   * the GM with no PC of their own). Only populated by GET /campaigns. */
  character?: {
    id: number
    name: string
    level: number
    classes: { className: string; level: number }[]
  } | null
}

export type CreateCampaignInput = {
  name: string
  description?: string
}

export type UpdateCampaignInput = {
  name?: string
  description?: string
}

export type SessionStatus = 'planned' | 'active' | 'ended'

export type Session = {
  id: number
  campaignId: number
  title: string | null
  sessionNumber: number
  notes: string | null
  status: SessionStatus
  startedAt: string | null
  endedAt: string | null
  createdAt: string
  updatedAt: string
}

export type CreateSessionInput = {
  sessionNumber: number
  title?: string
  notes?: string
}

export type UpdateSessionInput = {
  sessionNumber?: number
  title?: string
  notes?: string
}

export type CampaignMemberRole = 'player' | 'gm'

export type CampaignMember = {
  id: number
  campaignId: number
  characterId: number
  role: CampaignMemberRole
  addedAt: string
  character?: {
    id: number
    name: string
    level: number
    hpCurrent: number
    hpMax: number
    mpCurrent: number
    mpMax: number
    classes: { className: string; level: number }[]
  }
}

export type AddMemberInput = {
  characterId: number
  role?: CampaignMemberRole
  inviteToken?: string
}

export type CampaignInviteToken = {
  campaignId: number
  token: string
}

export type CampaignInvitePreview = {
  campaignId: number
  campaignName: string
}

export type UpdateMemberInput = {
  role: CampaignMemberRole
}

export type CampaignMembershipWithCampaign = {
  id: number
  campaignId: number
  characterId: number
  role: CampaignMemberRole
  addedAt: string
  campaign: {
    id: number
    name: string
    description: string | null
    updatedAt: string
  }
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
  catalog: {
    // Static rulebook reference; served id-keyed, cached hard (staleTime ∞).
    spells: () =>
      request<Record<string, CatalogSpell>>('/catalog/spells'),
  },
  characters: {
    options: () => request<CharacterOptions>('/characters/options'),
    list: () => request<Character[]>('/characters'),
    get: (id: number) => request<Character>(`/characters/${id}`),
    create: (input: CreateCharacterInput) =>
      request<Character>('/characters', { method: 'POST', body: JSON.stringify(input) }),
    updateVitals: (id: number, input: UpdateVitalsInput) =>
      request<VitalsResult>(`/characters/${id}/vitals`, {
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
      request<ConsumeItemResult>(`/characters/${id}/items/${itemId}/consume`, {
        method: 'POST',
        body: JSON.stringify(input),
      }),
    applyEffect: (id: number, input: ApplyEffectInput) =>
      request<ApplyEffectResult>(`/characters/${id}/active-effects`, {
        method: 'POST',
        body: JSON.stringify(input),
      }),
    applyDamage: (id: number, amount: number) =>
      request<ApplyDamageResult>(`/characters/${id}/damage`, {
        method: 'POST',
        body: JSON.stringify({ amount }),
      }),
    removeActiveEffect: (id: number, effectId: number) =>
      request<{ id: number }>(`/characters/${id}/active-effects/${effectId}`, {
        method: 'DELETE',
      }),
    endScene: (id: number) =>
      request<EffectsClearedResult>(`/characters/${id}/end-scene`, {
        method: 'POST',
      }),
    endDay: (id: number) =>
      request<EffectsClearedResult>(`/characters/${id}/end-day`, {
        method: 'POST',
      }),
    updateProficiencies: (id: number, input: UpdateProficienciesInput) =>
      request<ProficienciesResult>(`/characters/${id}/proficiencies`, {
        method: 'PATCH',
        body: JSON.stringify(input),
      }),
    updateLevel: (id: number, input: UpdateLevelInput) =>
      request<LevelResult>(`/characters/${id}/level`, {
        method: 'PATCH',
        body: JSON.stringify(input),
      }),
    updateAbilityChoices: (id: number, input: UpdateAbilityChoicesInput) =>
      request<AbilityChoicesResult>(`/characters/${id}/abilities`, {
        method: 'PATCH',
        body: JSON.stringify(input),
      }),
    updateConditions: (id: number, activeConditions: string[]) =>
      request<{ activeConditions: string }>(`/characters/${id}/conditions`, {
        method: 'PATCH',
        body: JSON.stringify({ activeConditions }),
      }),
    updateClassLevel: (id: number, input: UpdateClassLevelInput) =>
      request<ClassLevelResult>(`/characters/${id}/classes/level`, {
        method: 'PATCH',
        body: JSON.stringify(input),
      }),
    getSheet: (id: number) =>
      request<CharacterWithComputed>(`/characters/${id}/sheet`),
    campaigns: (id: number) =>
      request<CampaignMembershipWithCampaign[]>(
        `/characters/${id}/campaigns`,
      ),
    learnSpell: (id: number, catalogSpellId: string) =>
      request<CharacterSpell>(`/characters/${id}/spells`, {
        method: 'POST',
        body: JSON.stringify({ catalogSpellId }),
      }),
    unlearnSpell: (id: number, catalogSpellId: string) =>
      request<{ catalogSpellId: string; removed: number }>(
        `/characters/${id}/spells/${encodeURIComponent(catalogSpellId)}`,
        { method: 'DELETE' },
      ),
    setSpellPrepared: (
      id: number,
      catalogSpellId: string,
      prepared: boolean,
    ) =>
      request<CharacterSpell>(
        `/characters/${id}/spells/${encodeURIComponent(catalogSpellId)}/prepared`,
        {
          method: 'PATCH',
          body: JSON.stringify({ prepared }),
        },
      ),
    castSpell: (
      id: number,
      catalogSpellId: string,
      augments: { augmentIndex: number; stacks: number }[] = [],
    ) =>
      request<CastSpellResult>(
        `/characters/${id}/spells/${encodeURIComponent(catalogSpellId)}/cast`,
        {
          method: 'POST',
          body: JSON.stringify({ augments }),
        },
      ),
  },
  campaigns: {
    list: () => request<Campaign[]>('/campaigns'),
    get: (id: number) => request<Campaign>(`/campaigns/${id}`),
    create: (input: CreateCampaignInput) =>
      request<Campaign>('/campaigns', {
        method: 'POST',
        body: JSON.stringify(input),
      }),
    update: (id: number, input: UpdateCampaignInput) =>
      request<Campaign>(`/campaigns/${id}`, {
        method: 'PATCH',
        body: JSON.stringify(input),
      }),
    delete: (id: number) =>
      request<{ id: number }>(`/campaigns/${id}`, { method: 'DELETE' }),
    rotateInvite: (id: number) =>
      request<CampaignInviteToken>(`/campaigns/${id}/invite`, {
        method: 'POST',
      }),
  },
  invites: {
    resolve: (token: string) =>
      request<CampaignInvitePreview>(
        `/invites/${encodeURIComponent(token)}`,
      ),
  },
  sessions: {
    list: (campaignId: number) =>
      request<Session[]>(`/campaigns/${campaignId}/sessions`),
    get: (campaignId: number, id: number) =>
      request<Session>(`/campaigns/${campaignId}/sessions/${id}`),
    create: (campaignId: number, input: CreateSessionInput) =>
      request<Session>(`/campaigns/${campaignId}/sessions`, {
        method: 'POST',
        body: JSON.stringify(input),
      }),
    update: (campaignId: number, id: number, input: UpdateSessionInput) =>
      request<Session>(`/campaigns/${campaignId}/sessions/${id}`, {
        method: 'PATCH',
        body: JSON.stringify(input),
      }),
    delete: (campaignId: number, id: number) =>
      request<{ id: number }>(`/campaigns/${campaignId}/sessions/${id}`, {
        method: 'DELETE',
      }),
    start: (campaignId: number, id: number) =>
      request<Session>(`/campaigns/${campaignId}/sessions/${id}/start`, {
        method: 'POST',
      }),
    end: (campaignId: number, id: number) =>
      request<Session>(`/campaigns/${campaignId}/sessions/${id}/end`, {
        method: 'POST',
      }),
    clearTracker: (campaignId: number, id: number) =>
      request<{ id: number }>(
        `/campaigns/${campaignId}/sessions/${id}/clear-tracker`,
        { method: 'POST' },
      ),
  },
  members: {
    list: (campaignId: number) =>
      request<CampaignMember[]>(`/campaigns/${campaignId}/members`),
    add: (campaignId: number, input: AddMemberInput) =>
      request<CampaignMember>(`/campaigns/${campaignId}/members`, {
        method: 'POST',
        body: JSON.stringify(input),
      }),
    updateRole: (campaignId: number, id: number, input: UpdateMemberInput) =>
      request<CampaignMember>(`/campaigns/${campaignId}/members/${id}`, {
        method: 'PATCH',
        body: JSON.stringify(input),
      }),
    remove: (campaignId: number, id: number) =>
      request<{ id: number }>(`/campaigns/${campaignId}/members/${id}`, {
        method: 'DELETE',
      }),
  },
}
