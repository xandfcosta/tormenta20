/**
 * Domain types the backend speaks. Split out of `api.ts` on purpose: the React
 * app kept both in one 800-line file, over this project's 500-line ceiling.
 * Types live here, the client composes them there.
 *
 * Type-only imports from `@tormenta20/t20-data` erase at compile, so no catalog
 * DATA rides in the bundle — it is fetched from /catalog and cached.
 */
import type { AttributeKey } from '@tormenta20/t20-data'

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
