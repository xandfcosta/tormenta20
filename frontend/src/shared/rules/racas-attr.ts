import type { AttributeKey } from '@/shared/api/attribute-keys'
import type { Raca } from '@/shared/api/catalog-types'
/**
 * Resolve a raça's atributo modifiers into a flat map — PURE logic over a
 * RESOLVED `raca`, NO catalog data. Split out of `./racas` (which holds the
 * RACAS record) so the frontend can call it against a raça from its fetched
 * cache without anchoring the racas catalog into the bundle
 * (project_front_decouple_catalog). `./racas` re-exports it.
 *
 * - `fixed`: returns the static map.
 * - `floating`: requires `floatingPicks` — exactly `count` distinct
 *   attributes, each given `value`. Validates exclusion.
 * - `subraca-gated`: requires `ascendencia` matching a variant key.
 */
export function resolveAtributoMod(
  raca: Raca,
  opts: {
    floatingPicks?: readonly AttributeKey[]
    ascendencia?: string
  } = {},
): Partial<Record<AttributeKey, number>> {
  const mod = raca.atributoMod
  if (mod.kind === 'fixed') return { ...mod.mods }

  if (mod.kind === 'floating') {
    const picks = opts.floatingPicks ?? []
    if (picks.length !== mod.count) {
      throw new Error(
        `resolveAtributoMod: ${raca.name} requires exactly ${mod.count} floating picks, got ${picks.length}`,
      )
    }
    if (new Set(picks).size !== picks.length) {
      throw new Error(`resolveAtributoMod: ${raca.name} floating picks must be distinct`)
    }
    if (mod.exclude && picks.includes(mod.exclude)) {
      throw new Error(
        `resolveAtributoMod: ${raca.name} cannot place +${mod.value} in ${mod.exclude}`,
      )
    }
    const result: Partial<Record<AttributeKey, number>> = {}
    for (const a of picks) result[a] = mod.value
    if (mod.penalty) result[mod.penalty.attribute] = mod.penalty.value
    return result
  }

  // subraca-gated
  if (!opts.ascendencia || !mod.variants[opts.ascendencia]) {
    const keys = Object.keys(mod.variants).join(', ')
    throw new Error(
      `resolveAtributoMod: ${raca.name} requires ascendência in [${keys}], got ${opts.ascendencia}`,
    )
  }
  return { ...mod.variants[opts.ascendencia] }
}
