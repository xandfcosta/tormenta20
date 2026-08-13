import { statFor } from '@/shared/rules/items-engine'
import { spellSaveDc } from '@/shared/rules/spells'
import type { ItemEffects } from '@/shared/api/item-types'
import { type AttributeKey } from '@/shared/api/attribute-keys'
import {
  areEngineCatalogsPrimed,
  computeSheetV2 as engineComputeSheetV2,
} from '@/shared/lib/engine-wasm'
import type {
  AttributeBreakdown,
  ComputedSheetV2,
  ExpertiseBreakdown,
  TotalContribs,
} from '@/shared/lib/computed-sheet-v2'
import type { Character } from '@/shared/api/api'
import {
  attributeContributions,
  attributeTotal,
  bestBaseSpellCd,
  characterDamageReduction,
  defenseTotal,
  displacementTotal,
  expertiseTotalWithItems,
  flySpeedTotal,
  inventorySlotsTotal,
  pmCostMod,
  pmLimitTotal,
  spellDCBonus,
  tempHpFromPowers,
} from './derived'

const EMPTY_SET: ReadonlySet<string> = new Set()

const ATTRIBUTE_KEYS: readonly AttributeKey[] = [
  'strength',
  'dexterity',
  'constitution',
  'intelligence',
  'wisdom',
  'charisma',
]

/** statFor's {total, contributions} → the ComputedSheetV2 TotalContribs shape. */
function toTotalContribs(stat: ReturnType<typeof statFor>): TotalContribs {
  return {
    total: stat.total,
    contributions: stat.contributions.map((c) => ({
      source: c.source,
      amount: c.amount,
      ...(c.note ? { note: c.note } : {}),
    })),
  }
}

/**
 * The pure TS assembly of `ComputedSheetV2` from the `derived.ts` breakdowns —
 * the SAME payload the Go `ComputeSheetV2` reproduces byte-equal (proven by the
 * `sheetV2` parity oracle). Kept as the single source of truth so both the oracle
 * generator and this hook's test branch reuse it. `tempHpFuria` uses
 * furiaActive=true (the base sheet with furia off is always {0, []}).
 */
export function assembleSheetV2(char: Character, effects: ItemEffects): ComputedSheetV2 {
  const attributes = Object.fromEntries(
    ATTRIBUTE_KEYS.map((a) => [
      a,
      { total: attributeTotal(char, a, effects), contributions: attributeContributions(a, effects) },
    ]),
  ) as Record<AttributeKey, AttributeBreakdown>
  return {
    defense: defenseTotal(char, effects),
    displacement: displacementTotal(char, effects),
    flySpeed: flySpeedTotal(effects),
    inventorySlots: inventorySlotsTotal(char, effects),
    attributes,
    pmLimit: pmLimitTotal(char, effects),
    bestBaseSpellCd: bestBaseSpellCd(char, effects),
    spellCdByAttribute: Object.fromEntries(
      ATTRIBUTE_KEYS.map((a) => [
        a,
        spellSaveDc(char.level, attributeTotal(char, a, effects)),
      ]),
    ) as Record<AttributeKey, number>,
    spellDCBonus: spellDCBonus(effects),
    pmCostMod: pmCostMod(effects),
    attackAll: toTotalContribs(statFor(effects, { k: 'attack', scope: 'all' })),
    damageAll: toTotalContribs(statFor(effects, { k: 'damage', scope: 'all' })),
    damageReduction: characterDamageReduction(char, effects),
    tempHpFuria: tempHpFromPowers(char, effects, true),
    expertises: char.expertises.map((e) => ({
      name: e.name,
      attribute: e.attribute,
      ...expertiseTotalWithItems(char, e, effects),
    })),
  }
}

/**
 * The sheet-breakdown CHOKE POINT (Fase A): the rich `ComputedSheetV2` via the
 * Go/WASM engine — the single source of truth for every derived field. Same
 * MODE-gate as `resolveEffects`/`computeVitalPools`: in production/dev the engine
 * computes everything; the TS branch (`assembleSheetV2` over the TS derive) is
 * TEST-ONLY, so it stays the parity oracle without wasm and is dead-code-
 * eliminated from the app bundle (`import.meta.env.MODE` is statically
 * `'production'` there). Parity is proven byte-equal by the `sheetV2` oracle.
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
