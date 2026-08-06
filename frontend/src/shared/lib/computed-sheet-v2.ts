import type { AttributeKey } from '@tormenta20/t20-data'

/**
 * ComputedSheetV2 — the rich derived sheet the Go engine (engine-go
 * `ComputeSheetV2`) returns over the WASM boundary: every `derived.ts` breakdown
 * in one payload. This TS type MIRRORS the Go struct field-for-field (verified
 * byte-equal by the parity oracle `sheetV2`); keep the two in lockstep. Consumed
 * by the sheet hooks once the UI swaps off the TS derive (task #7).
 */

/** A {source, amount, note?} contribution row (defense/expertise/pm/spell). */
export type BreakdownContribution = {
  source: string
  amount: number
  note?: string
}

/** A {source, amount} row (attribute/RD/tempHp — no note). */
export type SourceAmount = {
  source: string
  amount: number
}

export type DefenseBreakdown = {
  base: number
  itemBonus: number
  total: number
  dexApplied: boolean
  contributions: BreakdownContribution[]
}

/** Shared {base, itemBonus, total, contributions} shape (displacement, pmLimit). */
export type ValueBreakdown = {
  base: number
  itemBonus: number
  total: number
  contributions: BreakdownContribution[]
}

/** {total, contributions} shape (spellDCBonus, pmCostMod). */
export type TotalContribs = {
  total: number
  contributions: BreakdownContribution[]
}

export type AttributeBreakdown = {
  total: number
  contributions: SourceAmount[]
}

export type ExpertiseBreakdown = {
  name: string
  attribute: AttributeKey
  base: number
  itemBonus: number
  total: number
  halfLevel: number
  attrValue: number
  training: number
  itemContributions: BreakdownContribution[]
  armorPenaltyApplied: number
}

export type RdBreakdown = {
  total: number
  sources: SourceAmount[]
}

export type TempHpBreakdown = {
  total: number
  sources: SourceAmount[]
}

export type ComputedSheetV2 = {
  defense: DefenseBreakdown
  displacement: ValueBreakdown
  flySpeed: number
  inventorySlots: number
  attributes: Record<AttributeKey, AttributeBreakdown>
  pmLimit: ValueBreakdown
  bestBaseSpellCd: number | null
  /** Spell save CD per casting attribute (p171) — a spell row picks the CD for
   *  any of its applicable classes without re-deriving. */
  spellCdByAttribute: Record<AttributeKey, number>
  spellDCBonus: TotalContribs
  pmCostMod: TotalContribs
  /** {k:attack|damage, scope:all} globals — added onto every weapon/attack. */
  attackAll: TotalContribs
  damageAll: TotalContribs
  damageReduction: RdBreakdown
  /** tempHpFromPowers with furia active — the Alma de Bronze branch. */
  tempHpFuria: TempHpBreakdown
  expertises: ExpertiseBreakdown[]
}
