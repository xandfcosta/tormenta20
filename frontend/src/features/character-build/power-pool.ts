import {
  type ClassEntry,
  type PowerOption,
  classPowerCandidates,
  tormentaPowerOptions,
  totalSlots,
  usedSlots,
} from '@/entities/character/class-powers'
import { racesGrantTormenta } from '@/shared/lib/abilities-cache'
import { matchesQuery } from '@/shared/lib/fuzzy-filter'
import { type RaceChoiceState, appliedRaces } from './grant-helpers'

export type PowerFacet = 'all' | 'class' | 'general' | 'tormenta'

export type PowerLedger = { total: number; used: number; remaining: number }

type PoolValues = { classes: ClassEntry[]; races: string[] }

/**
 * Every power a creation slot may be spent on: the primary class's electives
 * plus all general powers — they are alternatives for the SAME slot (p33) —
 * plus poderes da Tormenta when an APPLIED race grants access (Lefou).
 *
 * "Applied" matters: a secondary race is flavor until the player opts in, so a
 * Lefou listed second must not silently open the Tormenta list.
 *
 * @example draftPowerPool({ classes, races }, raceChoices).length // 84
 */
export function draftPowerPool(
  values: PoolValues,
  raceChoices: RaceChoiceState,
): PowerOption[] {
  const primary = values.classes[0]?.className
  if (!primary) return []
  const { classPowers, generalPowers } = classPowerCandidates(primary)
  const tormenta = racesGrantTormenta(appliedRaces(values.races, raceChoices))
    ? tormentaPowerOptions()
    : []
  return [...classPowers, ...generalPowers, ...tormenta]
}

/** How many powers sit behind each facet chip, for the live counts. */
export function facetTally(pool: PowerOption[]): Record<PowerFacet, number> {
  return {
    all: pool.length,
    class: pool.filter((p) => p.source === 'class').length,
    general: pool.filter((p) => p.source === 'general').length,
    tormenta: pool.filter((p) => p.source === 'tormenta').length,
  }
}

/** The pool narrowed by the active facet and the search box, in that order. */
export function filterPowers(
  pool: PowerOption[],
  facet: PowerFacet,
  query: string,
): PowerOption[] {
  const bySource = facet === 'all' ? pool : pool.filter((p) => p.source === facet)
  return query.trim() ? bySource.filter((p) => matchesQuery([p.name], query)) : bySource
}

/**
 * Slots earned, slots spent and slots left. Repeatable powers eat one slot per
 * sub-choice, which `usedSlots` already knows — this only adds the "never below
 * zero" reading the counter shows.
 */
export function powerLedger(
  classes: ClassEntry[],
  chosenIds: string[],
  powerChoices: Record<string, string[]>,
  pool: PowerOption[],
): PowerLedger {
  const total = totalSlots(classes)
  const used = usedSlots(chosenIds, powerChoices, new Map(pool.map((p) => [p.id, p])))
  return { total, used, remaining: Math.max(0, total - used) }
}
