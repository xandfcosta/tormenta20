import type { AttributeKey } from '@/shared/api/attribute-keys'
import {
  areEngineCatalogsPrimed,
  computeSheetV2 as engineComputeSheetV2,
} from '@/shared/lib/engine-wasm'
import type { ComputedSheetV2, ExpertiseBreakdown } from '@/shared/lib/computed-sheet-v2'
import type { Character } from '@/shared/api/api'

const EMPTY_SET: ReadonlySet<string> = new Set()



/**
 * The sheet-breakdown CHOKE POINT (Fase A): the rich `ComputedSheetV2` via the
 * Go/WASM engine — the single source of truth for every derived field. Não há
 * mais um assembler TS por trás (ele morreu com o `t20-data`, ALE-109) nem gate
 * por `import.meta.env.MODE`: o motor calcula em produção, em dev e no vitest,
 * que carrega o mesmo `.wasm`. O oráculo `sheetV2` fixa os números.
 */
// Memo: computing a sheet crosses into the WASM engine (serialize char → Go →
// compute → deserialize) — non-trivial, and ~10 sheet components call this with
// the same query-stable `char` reference in a single render. Uncached, each
// call re-ran the engine, so a mount/tab-switch fired ~10 WASM computes and
// felt laggy. Cache is keyed by the character object (WeakMap → the entry is
// dropped for free when the query hands back a new object on any edit, so it
// self-invalidates) then by the active-conditionals signature. The compute is a
// pure function of (char, conditionals), so a hit is always correct.
const sheetCache = new WeakMap<Character, Map<string, ComputedSheetV2>>()

export function computedSheetFor(
  char: Character,
  activeConditionals: ReadonlySet<string> = EMPTY_SET,
): ComputedSheetV2 {
  const key =
    activeConditionals.size === 0 ? '' : [...activeConditionals].sort().join('|')
  let byCond = sheetCache.get(char)
  const cached = byCond?.get(key)
  if (cached) return cached

  const result = computeSheetUncached(char, activeConditionals)
  if (!byCond) {
    byCond = new Map()
    sheetCache.set(char, byCond)
  }
  byCond.set(key, result)
  return result
}

function computeSheetUncached(
  char: Character,
  activeConditionals: ReadonlySet<string>,
): ComputedSheetV2 {
  if (!areEngineCatalogsPrimed()) {
    throw new Error(
      'computed sheet: WASM engine not primed — ensureEngineCatalogs() must resolve before any sheet renders',
    )
  }
  return engineComputeSheetV2(char, [...activeConditionals])
}

// The React file also exported a `useComputedSheet` hook whose whole body was
// "read the toggled conditionals from a Zustand store, then call
// `computedSheetFor`". On the Solid side the store is a signal the caller
// already holds, so the hook has no reason to exist — `computedSheetFor` is
// the seam, and it was pure all along.

/**
 * The computed breakdown for a single perícia by name — every standard + custom
 * expertise is persisted on the character, so the entry is present for anything
 * the UI can render (`undefined` only for an unknown name).
 */
export function expertiseFromSheet(
  sheet: ComputedSheetV2,
  name: string,
): ExpertiseBreakdown | undefined {
  return sheet.expertises.find((e) => e.name === name)
}

/**
 * Like {@link expertiseFromSheet} but total-safe for the always-present combat
 * perícias (Luta/Pontaria/resistências): returns a zeroed breakdown rather than
 * `undefined` if a character somehow lacks the entry, so display code stays
 * branch-free.
 */
export function requireExpertise(
  sheet: ComputedSheetV2,
  name: string,
  attribute: AttributeKey,
): ExpertiseBreakdown {
  return (
    expertiseFromSheet(sheet, name) ?? {
      name,
      attribute,
      base: 0,
      itemBonus: 0,
      total: 0,
      halfLevel: 0,
      attrValue: 0,
      training: 0,
      itemContributions: [],
      armorPenaltyApplied: 0,
    }
  )
}
