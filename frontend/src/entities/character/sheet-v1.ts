import type { Character, ComputedSheet } from '@/shared/api/api'
import { computeSheet } from '@/shared/lib/engine-wasm'
import { characterToInput } from './to-character-input'

// Memo keyed by the character object: the sheet cards each ask for the same
// derivation in one render, and a WASM compute per card is wasteful. A WeakMap
// self-invalidates — any edit hands back a new object from the query cache.
const cache = new WeakMap<Character, ComputedSheet>()

/**
 * The v1 `ComputedSheet` (attributes, vitals, defense, saves, attacks, skills)
 * for a character — derived by the SAME Go rules engine the backend runs, via
 * WASM, so there's no round-trip and no payload shape to drift.
 *
 * Replaces `GET /characters/:id/sheet` for the read-only sheet screens (ALE-77):
 * that endpoint returns the flat `ComputedSheetV2`, never the
 * `Character & { computed }` those screens were written against — they crashed.
 *
 * Requires the engine + catalogs to be primed (the root route's
 * `ensureEngineCatalogs` gate).
 *
 * @example const computed = computedSheetV1For(character)
 */
export function computedSheetV1For(character: Character): ComputedSheet {
  const cached = cache.get(character)
  if (cached) return cached
  const sheet = computeSheet(characterToInput(character))
  cache.set(character, sheet)
  return sheet
}
