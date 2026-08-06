import type { Modifier } from '../items/types'
import type { RaceDefinition } from './types'

/**
 * Race modifier assembly — pure logic over a RESOLVED race definition, NO
 * catalog data. Split out of `./catalog` (which imports RACES_CATALOG et al.)
 * so the frontend can call it against a race it looked up in its fetched cache
 * without anchoring the abilities catalog into the bundle
 * (project_front_decouple_catalog B.3). `./catalog` re-exports it.
 *
 * Build the list of Modifier[] contributed by a race definition + variant
 * choices for the player. Race attribute bonuses are emitted as `attribute`
 * target modifiers so they flow through the standard engine. Abilities with
 * variants only contribute modifiers from the chosen variant id.
 */
export function raceModifiers(
  race: RaceDefinition,
  variantChoices: ReadonlySet<string>,
): Modifier[] {
  const out: Modifier[] = []
  for (const [attr, amount] of Object.entries(race.attributeBonuses)) {
    if (typeof amount !== 'number' || amount === 0) continue
    out.push({
      target: { k: 'attribute', name: attr as never },
      amount,
      bonusType: 'untyped',
      note: race.name,
    })
  }
  for (const ability of race.abilities) {
    if (ability.modifiers) out.push(...ability.modifiers)
    if (ability.variants) {
      const chosen = ability.variants.find((v) => variantChoices.has(v.id))
      if (chosen?.modifiers) out.push(...chosen.modifiers)
    }
  }
  return out
}
