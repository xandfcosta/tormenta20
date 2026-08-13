/**
 * Domain types the backend speaks. Split out of `api.ts` on purpose: the React
 * app kept both in one 800-line file, over this project's 500-line ceiling.
 * Types live here, the client composes them there.
 *
 * Type-only imports from `@tormenta20/t20-data` erase at compile, so no catalog
 * DATA rides in the bundle — it is fetched from /catalog and cached.
 */
import type { ClassChoices } from '@/shared/api/catalog-types'
import type { AttributeKey } from '@/shared/api/attribute-keys'

export type { AttributeKey }

// --- auth + users -------------------------------------------------------------

export type AuthUser = {
  id: number
  email: string
  name: string | null
}

export type User = AuthUser & { createdAt: string }

export type Credentials = { email: string; password: string }

// --- character ----------------------------------------------------------------

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

export type UpdateExpertiseInput = {
  name: string
  attribute?: AttributeKey
  trained?: boolean
}

import type { EquippedSlot } from './bonus-types'
export type { EquippedSlot }

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

/** Deformidade (Lefou p23): ≤2 perícias com +2, uma trocável por um poder. */
export type DeformidadeChoiceInput = {
  pericias: string[]
  tormentaPower?: string
}

/** Race attribute picks persisted so the sheet derives the racial mod ONCE,
 *  from the base attributes — baking it into the value would double-count. */
export type RaceAttributeChoicesInput = {
  floatingPicks?: string[]
  ascendencia?: string
  deformidade?: DeformidadeChoiceInput
}

/**
 * POST /characters — the whole character in one body. Verified field by field
 * against `api/character_create.go`: the handler seeds every perícia, derives
 * the class proficiencies itself, and re-heals PV/PM from the engine after the
 * insert, so the pools here are a starting point and not the last word.
 */
export type CreateCharacterInput = {
  name: string
  races: string[]
  origin: string
  classes: { className: string; level: number }[]
  god?: string
  godPower?: string
  /** Dinheiro inicial em T$ (Tabela 3-1 p140 / 4d6). */
  tibar?: number
  /** Itens iniciais (kit p140 + itens da origem + compras). */
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
  classPowers?: string[]
  originChoices?: string[]
  classChoices?: ClassChoices
  trainedExpertises?: string[]
  powerChoices?: Record<string, string[]>
  raceAttributeChoices?: RaceAttributeChoicesInput
  /** Opted-in secondary races (GM-negotiated); their mods apply too. */
  secondaryRaceChoices?: ({ race: string } & RaceAttributeChoicesInput)[]
}

export type UpdateItemInput = {
  name?: string
  quantity?: number
  slots?: number
  equipped?: EquippedSlot | null
  improvements?: string[]
  material?: string | null
}

export type ConsumeItemInput = {
  hpRolled?: number
  mpRolled?: number
}

export type ActiveEffect = {
  id: number
  catalogId: string
  scope: 'scene' | 'day'
  /** JSON-encoded Modifier[] copied from catalog at consume time */
  modifiers: string
  createdAt: string
}

export type CharacterSpell = {
  id: number
  catalogSpellId: string
  prepared: boolean
  learnedAt: string
}

export type CharacterClass = { className: string; level: number }

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
  /** JSON-encoded powerId -> option id[] sub-choices */
  powerChoices: string
  createdAt: string
  updatedAt: string
  races: { race: string }[]
  classes: CharacterClass[]
  expertises: CharacterExpertise[]
  items: CharacterItem[]
  activeEffects: ActiveEffect[]
  spells: CharacterSpell[]
}

export type CharacterOptions = {
  races: string[]
  classes: string[]
  origins: string[]
  gods: string[]
  sizes: string[]
  expertises: ExpertiseDef[]
}

// --- campaign -----------------------------------------------------------------

export type CampaignMemberRole = 'player' | 'gm'

export type Campaign = {
  id: number
  ownerId: number
  name: string
  description: string | null
  createdAt: string
  updatedAt: string
  /** Caller's role — the server resolves owner=gm vs member=player. Optional
   *  because older cached payloads may predate it. */
  role?: CampaignMemberRole
  /** The caller's own member character here (null when they're the GM with no
   *  PC of their own). Only populated by GET /campaigns. */
  character?: {
    id: number
    name: string
    level: number
    classes: CharacterClass[]
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

/** What a player sees BEFORE joining — public, resolved from the token alone. */
export type CampaignInvitePreview = {
  campaignId: number
  campaignName: string
}

export type AddMemberInput = {
  characterId: number
  role?: CampaignMemberRole
  /** Present when joining through an invite link; the server verifies it. */
  inviteToken?: string
}

/** POST /campaigns/:id/invite mints a fresh token, invalidating the previous. */
export type CampaignInviteToken = {
  campaignId: number
  token: string
}

export type CampaignMember = {
  id: number
  campaignId: number
  characterId: number
  role: CampaignMemberRole
  addedAt: string
  character?: {
    id: number
    ownerId: number
    name: string
    level: number
    hpCurrent: number
    hpMax: number
    mpCurrent: number
    mpMax: number
    classes: CharacterClass[]
  }
}

// --- session ------------------------------------------------------------------

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

/**
 * Delta returned by consumeItem (not the whole Character) — the client merges
 * it into the cached character. `item.quantity` is the new count (0 when
 * removed); `effect` is the scene/day ActiveEffect a non-instant consumable
 * created; vitals are the clamped post-consume values.
 */
export type ConsumeItemResult = {
  item: { id: number; quantity: number; removed: boolean }
  effect: ActiveEffect | null
  hpCurrent: number
  mpCurrent: number
}

// --- PV temporários (contrato do POST :id/damage e :id/active-effects) ---

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

/** Pools the server recomputed after a write that changes them (level up/down).
 *  Merging these keeps hpMax/mpMax fresh without a refetch — a 2026-08 bug left
 *  the bars on the pre-level pools. Verified against `api/character_level.go`. */
export type VitalsSync = {
  hpMax: number
  hpCurrent: number
  mpMax: number
  mpCurrent: number
}

/** PATCH :id/classes/level — one class's new level, plus the resynced pools. */
export type UpdateClassLevelInput = { className: string; level: number }

export type ClassLevelResult = {
  level: number
  classes: { className: string; level: number }[]
  vitals: VitalsSync
}

/** POST :id/damage — atomic temp-first routing result (F2). */
export type ApplyDamageResult = {
  hpCurrent: number
  tempHpRemaining: number
  drained: DamageDrainStep[]
}

/** PATCH :id/proficiencies — the full category set; the server dedups and
 *  rejects unknown ids, then echoes the stored blob. */
export type UpdateProficienciesInput = { proficiencies: string[] }
export type ProficienciesResult = { proficiencies: string }

/** PATCH :id/vitals — either field alone is a valid write. */
export type UpdateVitalsInput = {
  hpCurrent?: number
  mpCurrent?: number
}

/** PATCH :id/vitals answers with the CLAMPED pair, never the whole character. */
export type VitalsResult = { hpCurrent: number; mpCurrent: number }

/** POST :id/active-effects — one of `spellId` (spell buff), `powerId` (power
 *  grant) or `manualTempHp` (GM ad-hoc temp-PV pool; 0 clears it). Verified
 *  against the Go handler (`api/apply_effect.go`). */
export type ApplyEffectInput = {
  spellId?: string
  powerId?: string
  manualTempHp?: number
  scope?: 'scene' | 'day'
}

/**
 * POST :id/end-scene and :id/end-day — the scopes the client must drop from its
 * cached character. Ending the day also ends the running scene (book rest
 * semantics), so it answers BOTH scopes; trusting the endpoint's name instead
 * would leave cleared scene buffs painted on the sheet. Verified against
 * `api/character_effects.go` (routes added for the Solid port — the React app
 * called them before they existed and got a 404).
 */
export type EffectsClearedResult = {
  clearedScopes: ('scene' | 'day')[]
}

/** PATCH :id/conditions — replaces the whole set; the server validates every id
 *  against the book catalog (p394-395) and echoes the stored JSON blob. */
export type ConditionsResult = { activeConditions: string }

/** PATCH /campaigns/:cid/sessions/:id — any subset; the server echoes the
 *  whole session back. Verified against `api/sessions.go`. */
export type UpdateSessionInput = {
  sessionNumber?: number
  title?: string
  notes?: string
}

/** POST :id/spells/:catalogSpellId/cast — the augment picks, one entry per
 *  aprimoramento (stacks combined; a duplicate index is a 400). */
export type SpellAugmentPick = { augmentIndex: number; stacks: number }

/**
 * Delta from castSpell: the new PM plus the catalyst effects the cast consumed.
 * `removedEffectIds` is always empty today — the catalisador scene-discount is
 * deferred in the Go handler — but it is part of the contract and the client
 * already drops what it names.
 */
export type CastSpellResult = {
  mpCurrent: number
  removedEffectIds: number[]
}

/** DELETE :id/spells/:catalogSpellId — `removed` is 0 when it wasn't known
 *  (still a 200, so "esquecer" is idempotent). */
export type UnlearnSpellResult = { catalogSpellId: string; removed: number }

/**
 * PATCH :id/abilities — patches ANY SUBSET of the five ability-choice blobs;
 * omitted fields are left alone, and sending none is a 400. Verified against
 * `api/character_abilities.go`.
 */
export type UpdateAbilityChoicesInput = {
  raceAbilityChoices?: string[]
  originChoices?: string[]
  classPowers?: string[]
  classChoices?: ClassChoices
  powerChoices?: Record<string, string[]>
}

/** The server echoes back ONLY the blobs it wrote, each already JSON-encoded. */
export type AbilityChoicesResult = {
  raceAbilityChoices?: string
  originChoices?: string
  classPowers?: string
  classChoices?: string
  powerChoices?: string
}

// --- computed sheet -----------------------------------------------------------

import type { RaceDefinition } from '@/shared/api/catalog-types'
import type { ComputedSheetV2 } from '@/shared/lib/computed-sheet-v2'

export type { RaceDefinition, ComputedSheetV2 }

/**
 * GET /characters/:id/sheet returns the **flat `ComputedSheetV2`** the Go
 * engine produces — NOT `Character & { computed }`, which is the pre-cutover
 * Nest shape the React app still types it as (and crashes on: see ALE-77).
 * Verified against the running backend.
 */
