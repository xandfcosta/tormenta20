import type { CatalogSpell } from '@/shared/api/catalog-types'
import type { SpellCircle } from '@/shared/api/spell-types'
import type { SpellAugmentPick } from '@/shared/api/api'

type Augment = CatalogSpell['augments'][number]

/**
 * The dialog's per-augment stacks as the payload the server expects. Zero means
 * "not taken" and drops out; the source being a Map is what guarantees one
 * entry per augment — a duplicate index is a 400.
 *
 * @example augmentPicksFrom(new Map([[0, 2]])) // [{ augmentIndex: 0, stacks: 2 }]
 */
export function augmentPicksFrom(stacksByIndex: ReadonlyMap<number, number>): SpellAugmentPick[] {
  return [...stacksByIndex]
    .filter(([, stacks]) => stacks > 0)
    .map(([augmentIndex, stacks]) => ({ augmentIndex, stacks }))
}

/**
 * PM the chosen augments add. An unknown index contributes 0 rather than
 * throwing — the server is the authority on what is valid, and a preview that
 * crashes is worse than one that under-counts.
 */
export function augmentPmFor(
  augments: readonly Augment[],
  picks: readonly SpellAugmentPick[],
): number {
  return picks.reduce(
    (total, pick) => total + (augments[pick.augmentIndex]?.pmCost ?? 0) * pick.stacks,
    0,
  )
}

/**
 * An augment gated behind a circle the character cannot reach (p42/p171): a
 * power-granted spell on a non-caster casts at its own circle and nothing
 * above it.
 */
export function isAugmentLocked(augment: Augment, castableCircle: SpellCircle | number): boolean {
  return augment.requiresCircle !== undefined && augment.requiresCircle > castableCircle
}
