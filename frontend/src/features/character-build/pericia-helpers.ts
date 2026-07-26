import { classExpertiseSlots } from '@tormenta20/t20-data'

export type PericiaPlan = {
  /** Auto-trained perícias (always granted). */
  fixed: string[]
  /** Pick exactly one, if the class has an either/or slot. */
  eitherOr: [string, string] | null
  /** How many pool perícias to pick (includes the +INT bonus). */
  chooseCount: number
  /** The pool the player draws the `chooseCount` picks from. */
  choosePool: string[]
}

/**
 * The perícia-training plan a class exposes to the creation UI at a given
 * Inteligência modifier. Returns null for an unknown class.
 */
export function periciaPlan(
  className: string,
  intMod: number,
): PericiaPlan | null {
  const slots = classExpertiseSlots(className, intMod)
  if (!slots) return null
  return {
    fixed: slots.fixed,
    eitherOr: slots.eitherOr?.options ?? null,
    chooseCount: slots.chooseCount,
    choosePool: slots.choosePool,
  }
}

/** How many pool picks remain for this plan given the trained set so far. */
export function poolPicksRemaining(
  plan: PericiaPlan,
  trained: string[],
): number {
  const set = new Set(trained)
  const picked = plan.choosePool.filter((p) => set.has(p)).length
  return Math.max(0, plan.chooseCount - picked)
}
