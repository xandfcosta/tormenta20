import { type AttributeKey, type ItemEffects, statFor } from '@tormenta20/t20-data'
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
import { useActiveConditionals } from '@/shared/stores/conditionals-store'
import {
  attributeContributions,
  attributeTotal,
  bestBaseSpellCd,
  characterDamageReduction,
  characterEffects,
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
export function computedSheetFor(
  char: Character,
  activeConditionals: ReadonlySet<string> = EMPTY_SET,
): ComputedSheetV2 {
  if (import.meta.env.MODE === 'test') {
    return assembleSheetV2(char, characterEffects(char, activeConditionals))
  }
  if (!areEngineCatalogsPrimed()) {
    throw new Error(
      'computed sheet: WASM engine not primed — ensureEngineCatalogs() must resolve before any sheet renders',
    )
  }
  return engineComputeSheetV2(char, [...activeConditionals])
}

/** Reactive `ComputedSheetV2` for a character, tracking its active conditionals. */
export function useComputedSheet(char: Character): ComputedSheetV2 {
  const active = useActiveConditionals(char.id)
  return computedSheetFor(char, active)
}

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
