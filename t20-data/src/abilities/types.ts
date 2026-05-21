import type { AttributeKey } from '../attributes'
import type { ExpertiseName } from '../expertises'
import type { Modifier } from '../items/types'

/**
 * Typed prerequisite for class powers and general powers. Covers the four
 * common gates plus a `note` escape hatch for prereqs not yet machine-checked
 * (Ofício sub-crafts, spell knowledge, etc.) — UI displays note verbatim.
 *
 *  - power: must own a specific other power (by id).
 *  - anyPower: must own at least one of the listed ids (used for "any
 *    armadilha", "any missa" rules).
 *  - trained: must be trained in the named expertise.
 *  - attribute: attribute (post-race-mods, raw character.X) must meet min.
 *  - classChoice: a per-class choice (devoto/caminho) stored in
 *    Character.classChoices must satisfy `allowed`/`forbidden`. When neither
 *    is set, any non-empty value satisfies (i.e., "must be devoto").
 *  - note: free-form description shown in UI; not auto-resolved.
 */
export type Prerequisite =
  | { kind: 'power'; id: string }
  | { kind: 'anyPower'; ids: string[] }
  | { kind: 'trained'; expertise: ExpertiseName }
  | { kind: 'attribute'; attr: AttributeKey; min: number }
  | {
      kind: 'classChoice'
      class: string
      field: 'devoto' | 'caminho'
      allowed?: string[]
      forbidden?: string[]
      /** Human-readable hint shown in UI ("Devoto, exceto Lena/Marah"). */
      label: string
    }
  | { kind: 'note'; description: string }

/**
 * Catalog entry for a racial ability. Mirrors how PDF Chapter 1 (Races)
 * presents each trait — a fixed grant per race, sometimes with sub-options
 * the player picks at creation (e.g., elf Linhagem).
 */
export type RaceAbility = {
  id: string
  raceId: string
  name: string
  description: string
  /** True when player must pick one variant from `options`. */
  variants?: RaceAbilityVariant[]
  /** Numeric modifiers folded into the engine when this ability is owned. */
  modifiers?: Modifier[]
}

export type RaceAbilityVariant = {
  id: string
  name: string
  description: string
  modifiers?: Modifier[]
}

export type RaceDefinition = {
  id: string
  name: string
  /** Attribute deltas applied at creation (T20 racial bonuses). */
  attributeBonuses: Partial<Record<AttributeKey, number>>
  /** Innate abilities granted by this race. */
  abilities: RaceAbility[]
}

/**
 * Origens (PDF Chapter 2). Each origin lists 4 perícias and 2 poderes the
 * player picks 2 benefits from, plus one exclusive poder único.
 */
export type OriginBenefit = {
  id: string
  name: string
  kind: 'pericia' | 'poder'
  description: string
  /** If kind='pericia', the expertise it trains. */
  expertise?: ExpertiseName
  modifiers?: Modifier[]
}

export type OriginDefinition = {
  id: string
  name: string
  /** Full benefit pool (typically 4 perícias + 2 poderes). */
  benefits: OriginBenefit[]
  /** Exclusive poder único — chosen as one of the two slots. */
  poderUnico: OriginBenefit
}

/**
 * Class powers (PDF Chapter 1 — class chapters). Includes both fixed
 * habilidades granted at specific levels and the elective pool the player
 * draws from when leveling ("Poder de Bárbaro" etc.).
 */
export type ClassPower = {
  id: string
  className: string
  name: string
  description: string
  /** When non-null, this power is automatically granted at this class level. */
  grantedAtLevel?: number
  /** Typed prerequisites — power refs, trained perícias, attribute mins, or
   *  free-form notes for gates not yet machine-checked. */
  prerequisites?: Prerequisite[]
  /** Minimum class level required to pick (separate from `grantedAtLevel`). */
  minLevel?: number
  modifiers?: Modifier[]
}
