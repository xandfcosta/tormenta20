import type { Modifier } from '../items/types'
import type { OriginBenefit, OriginDefinition } from './types'

/**
 * Origin modifier assembly — pure logic over a RESOLVED origin definition, NO
 * catalog data. Split out of `./origins` (which holds ORIGINS_CATALOG) so the
 * frontend can call it against an origin from its fetched cache without
 * anchoring the catalog (project_front_decouple_catalog B.3). `./origins`
 * re-exports it.
 *
 * Sums the modifiers from the benefits the player actually chose (`choiceSet`),
 * including the poder único when picked.
 */
export function originModifiers(
  origin: OriginDefinition,
  choiceSet: ReadonlySet<string>,
): Modifier[] {
  const out: Modifier[] = []
  const all: OriginBenefit[] = [...origin.benefits, origin.poderUnico]
  for (const benefit of all) {
    if (!choiceSet.has(benefit.id)) continue
    if (benefit.modifiers) out.push(...benefit.modifiers)
  }
  return out
}
