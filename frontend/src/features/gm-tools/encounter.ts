import type { Monster } from '@/shared/api/catalog-types'
import { computeGroupNd } from '@/shared/lib/encounter-math'

/**
 * Mirror of the backend's `INITIATIVE_MAX_ENTRIES` — the server rejects the
 * 51st add. The client clamps too, so sending a big encounter to the tracker
 * reports the truncation in the UI instead of dying as a silent WS error
 * halfway through the loop.
 */
export const INITIATIVE_MAX_ENTRIES = 50

/** One line of the composition: a monster and how many of it. */
export type EncounterEntry = { monsterId: string; quantity: number }

/** A composition line resolved to its monster plus the group's book ND. */
export type EnrichedGroup = {
  monster: Monster
  quantity: number
  groupNd: number
}

/**
 * Resolves raw entries to their monsters and computes each group's ND (Cap 7
 * p282). Entries whose monster id is no longer in the bestiary are DROPPED —
 * a stale id would otherwise render an empty row with a live quantity.
 *
 * @example enrichEncounter([{ monsterId: 'goblin', quantity: 4 }], bestiary)
 */
export function enrichEncounter(
  entries: readonly EncounterEntry[],
  monsters: readonly Monster[],
): EnrichedGroup[] {
  return entries.flatMap((entry) => {
    const monster = monsters.find((m) => m.id === entry.monsterId)
    if (!monster) return []
    return [
      {
        monster,
        quantity: entry.quantity,
        groupNd: computeGroupNd(monster.nd, entry.quantity),
      },
    ]
  })
}

/** The encounter's ND: the sum of its groups. The book is silent on mixed
 *  composition, so summing is the permissive default and the GM eyeballs. */
export function encounterNd(groups: readonly EnrichedGroup[]): number {
  return groups.reduce((sum, group) => sum + group.groupNd, 0)
}

export type EncounterDifficulty = {
  label: string
  /** Which end of the scale it sits on — drives the colour, not the text. */
  tone: 'calm' | 'even' | 'hard' | 'deadly'
}

/**
 * Maps the encounter-vs-party ND gap to a difficulty band. ND igual ao nível do
 * grupo é um combate justo ("Médio", p281); abaixo é mais fácil, acima é pior.
 *
 * The gap is often FRACTIONAL — monsters below ND 1 give fractional group NDs
 * and the doubling rule uses `log2` — while the bands are whole ND steps. So it
 * is rounded to the nearest step first. Without that, a small negative gap like
 * −0.75 (one ND 1/4 against a level-1 party) slipped past both `<= -1` and
 * `=== 0` and landed on "Difícil" (ALE-25).
 */
export function encounterDifficulty(gap: number): EncounterDifficulty {
  const step = Math.round(gap)
  if (step <= -3) return { label: 'Trivial', tone: 'calm' }
  if (step <= -1) return { label: 'Fácil', tone: 'calm' }
  if (step === 0) return { label: 'Médio', tone: 'even' }
  if (step <= 2) return { label: 'Difícil', tone: 'hard' }
  return { label: 'Mortal', tone: 'deadly' }
}

/** One creature as the tracker will hold it — named apart and carrying its own
 *  PV, which is the whole reason the copies are numbered. */
export type EncounterInitiativeEntry = {
  label: string
  hpCurrent: number
  hpMax: number
}

/**
 * The encounter as tracker entries, one per creature — four goblins become
 * "Goblin 1".."Goblin 4", each with its own PV pool, because the GM damages
 * them one at a time.
 *
 * Capped at what the server accepts, counting what is ALREADY in the tracker;
 * the caller reports what was left out instead of letting the surplus die as a
 * silent socket error halfway through the loop.
 *
 * @example encounterInitiativeEntries(groups, 48) // { entries: [...2], dropped: 3 }
 */
export function encounterInitiativeEntries(
  groups: readonly EnrichedGroup[],
  alreadyInTracker = 0,
): { entries: EncounterInitiativeEntry[]; dropped: number } {
  const all = groups.flatMap((group) =>
    Array.from({ length: group.quantity }, (_, i) => ({
      label: group.quantity === 1 ? group.monster.name : `${group.monster.name} ${i + 1}`,
      hpCurrent: group.monster.hp,
      hpMax: group.monster.hp,
    })),
  )
  const room = Math.max(0, INITIATIVE_MAX_ENTRIES - alreadyInTracker)
  return { entries: all.slice(0, room), dropped: Math.max(0, all.length - room) }
}
