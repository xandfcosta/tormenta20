import { classExpertiseSlots, EXPERTISE_NAMES } from '@tormenta20/t20-data'

export type PericiaPlan = {
  /** Auto-trained perícias (always granted). */
  fixed: string[]
  /** Pick exactly one, if the class has an either/or slot. */
  eitherOr: [string, string] | null
  /** How many class-pool perícias to pick (base only — NO +INT folded in). */
  classCount: number
  /** The class list the `classCount` picks are drawn from. */
  classPool: string[]
  /** How many bonus perícias Inteligência grants (max(0, intMod)). */
  intCount: number
  /**
   * Pool the INT bonus picks draw from. Per the book, "perícias por
   * Inteligência não precisam ser da lista da classe" — so this is every
   * perícia the class list (and fixed/either-or) doesn't already offer, keeping
   * the perícia space partitioned so each name lives in exactly one band.
   */
  intPool: string[]
}

/**
 * The perícia-training plan a class exposes to the creation UI. The +INT bonus
 * is kept SEPARATE from the class picks (book: INT perícias can be any perícia,
 * not just the class list). Returns null for an unknown class.
 */
export function periciaPlan(
  className: string,
  intMod: number,
): PericiaPlan | null {
  // Pass 0 so the base class count is NOT inflated by INT — the INT bonus is a
  // distinct band drawn from a distinct pool.
  const slots = classExpertiseSlots(className, 0)
  if (!slots) return null
  const eitherOr = slots.eitherOr?.options ?? null
  const classPool = slots.choosePool
  const excluded = new Set<string>([
    ...slots.fixed,
    ...(eitherOr ?? []),
    ...classPool,
  ])
  return {
    fixed: slots.fixed,
    eitherOr,
    classCount: slots.chooseCount,
    classPool,
    intCount: Math.max(0, intMod),
    intPool: EXPERTISE_NAMES.filter((n) => !excluded.has(n)),
  }
}

/** How many picks remain in one band given the trained set so far. */
export function bandPicksRemaining(
  pool: string[],
  count: number,
  trained: string[],
): number {
  const set = new Set(trained)
  const picked = pool.filter((p) => set.has(p)).length
  return Math.max(0, count - picked)
}
