import { EXPERTISE_NAMES, raceFreeExpertiseCount } from '@tormenta20/t20-data'
import { classExpertiseSlots } from '@/entities/character/class-expertise-rules'

export type PericiaPlan = {
  /** Auto-trained perícias (always granted). */
  fixed: string[]
  /** Pick exactly one, if the class has an either/or slot. */
  eitherOr: [string, string] | null
  /** How many class-pool perícias to pick (base only — NO +INT folded in). */
  classCount: number
  /** The class list the class picks are drawn from. */
  classPool: string[]
  /** Total FREE (any-perícia) picks = Inteligência bonus + race grants. */
  freeCount: number
  /** Free picks from Inteligência (for the label breakdown). */
  intCount: number
  /** Free picks from race grants — Versátil etc. (for the label breakdown). */
  raceCount: number
  /**
   * Pool the free picks are shown in: every perícia OUTSIDE the class list
   * (and not fixed/either-or). Keeping it disjoint from `classPool` avoids
   * rendering the same perícia in two bands; a free pick that should be a
   * class perícia is made by over-picking the class band (the overflow spills
   * into the free budget — see `periciaBudget`).
   */
  freePool: string[]
}

/**
 * The perícia-training plan a class + races expose to the creation UI. Class
 * picks come from the class list; the FREE picks (Inteligência + race grants
 * like Humano's Versátil) can be any perícia, per the book ("perícias por
 * Inteligência não precisam ser da lista da classe"). Returns null for an
 * unknown class.
 */
export function periciaPlan(
  className: string,
  intMod: number,
  raceNames: readonly string[],
): PericiaPlan | null {
  // Pass 0 so the base class count is NOT inflated by INT — the free picks are
  // a distinct budget over a distinct pool.
  const slots = classExpertiseSlots(className, 0)
  if (!slots) return null
  const eitherOr = slots.eitherOr?.options ?? null
  const classPool = slots.choosePool
  const excluded = new Set<string>([
    ...slots.fixed,
    ...(eitherOr ?? []),
    ...classPool,
  ])
  const intCount = Math.max(0, intMod)
  const raceCount = raceFreeExpertiseCount(raceNames)
  return {
    fixed: slots.fixed,
    eitherOr,
    classCount: slots.chooseCount,
    classPool,
    freeCount: intCount + raceCount,
    intCount,
    raceCount,
    freePool: EXPERTISE_NAMES.filter((n) => !excluded.has(n)),
  }
}

export type PericiaBudget = {
  classSpent: number
  freeSpent: number
  classRemaining: number
  freeRemaining: number
  /** True once the class list has more picks than its own cap — the excess is
   *  drawing from the free budget. */
  classOverflow: boolean
}

/**
 * Split the flat trained set across the two caps. Class-pool picks fill the
 * class cap first; any excess (plus every free-pool pick) draws the free
 * budget — so a free slot can land on a class perícia by over-picking the
 * class band.
 */
export function periciaBudget(
  plan: PericiaPlan,
  trained: string[],
): PericiaBudget {
  const set = new Set(trained)
  const classPoolPicks = plan.classPool.filter((p) => set.has(p)).length
  const freePoolPicks = plan.freePool.filter((p) => set.has(p)).length
  const classSpent = Math.min(classPoolPicks, plan.classCount)
  const freeSpent = classPoolPicks - classSpent + freePoolPicks
  return {
    classSpent,
    freeSpent,
    classRemaining: Math.max(0, plan.classCount - classSpent),
    freeRemaining: Math.max(0, plan.freeCount - freeSpent),
    classOverflow: classPoolPicks > plan.classCount,
  }
}
