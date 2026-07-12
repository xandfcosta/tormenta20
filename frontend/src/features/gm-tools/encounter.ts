import { BESTIARY, type Monster } from '@tormenta20/t20-data'
import { computeGroupNd } from '@/shared/lib/encounter-math'

/**
 * Mirror of backend `INITIATIVE_MAX_ENTRIES` — the server rejects the 51st
 * add with a BadRequestException. The frontend clamps too so batch feedback
 * (encounter → initiative) shows in the UI instead of a silent WS error
 * mid-loop.
 */
export const INITIATIVE_MAX_ENTRIES = 50

/** One line of encounter composition: a monster and how many of it. */
export type EncounterEntry = { monsterId: string; quantity: number }

/** A composition line resolved to its monster + the group's book ND. */
export type EnrichedGroup = {
  monster: Monster
  quantity: number
  groupNd: number
}

/**
 * Resolves raw entries to their monsters and computes each group's ND
 * (Cap 7 p282 via {@link computeGroupNd}). Drops entries whose monster id
 * no longer exists in the bestiary.
 */
export function enrichEncounter(
  entries: readonly EncounterEntry[],
): EnrichedGroup[] {
  return entries
    .map((e) => {
      const monster = BESTIARY.find((m) => m.id === e.monsterId)
      if (!monster) return null
      return {
        monster,
        quantity: e.quantity,
        groupNd: computeGroupNd(monster.nd, e.quantity),
      }
    })
    .filter((g): g is EnrichedGroup => g !== null)
}

/**
 * Maps the encounter-vs-party ND gap to a difficulty band + badge variant.
 * Bands follow the book's threat scale (trivial → mortal).
 */
export function encounterDifficulty(gap: number): {
  label: string
  variant: 'default' | 'secondary' | 'outline' | 'destructive'
} {
  if (gap <= -3) return { label: 'Trivial', variant: 'secondary' }
  if (gap <= -1) return { label: 'Fácil', variant: 'outline' }
  if (gap === 0) return { label: 'Médio', variant: 'default' }
  if (gap <= 2) return { label: 'Difícil', variant: 'default' }
  return { label: 'Mortal', variant: 'destructive' }
}
