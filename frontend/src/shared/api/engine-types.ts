// GERADO por engine-go — NÃO EDITE À MÃO.
//
// Regenere com:  cd engine-go && go generate ./engine
//
// As formas que atravessam a fronteira do motor (WASM + /sheet). O Go é a
// autoridade delas; os tipos de CATÁLOGO ficam à mão em rules-tables.ts e
// catalog.ts, porque o servidor os entrega como bytes crus e não tem struct
// para eles (ALE-108).

import type { AttributeKey } from './attribute-keys'

export type ActiveItem = {
  source: string
  equipped: string | null
  modifiers: Modifier[]
}

export type AggregatedStat = {
  total: number
  contributions: Contribution[]
}

export type AttributeBreakdown = {
  total: number
  contributions: SourceAmount[]
}

export type BreakdownContribution = {
  source: string
  amount: number
  note?: string
}

export type ClassChoiceSelections = {
  devoto: string
  caminho: string
}

export type ClassEntry = {
  className: string
  level: number
}

export type ComputedSheetV2 = {
  defense: DefenseBreakdown
  displacement: ValueBreakdown
  flySpeed: number
  inventorySlots: number
  attributes: Record<AttributeKey, AttributeBreakdown>
  pmLimit: ValueBreakdown
  bestBaseSpellCd: number | null
  spellCdByAttribute: Record<AttributeKey, number>
  spellDCBonus: TotalContribs
  pmCostMod: TotalContribs
  attackAll: TotalContribs
  damageAll: TotalContribs
  damageReduction: RdBreakdown
  tempHpFuria: TempHpBreakdown
  expertises: ExpertiseBreakdown[]
}

export type ConditionalDisplayInput = {
  target: ModifierTarget
  bonusType: string
  amount: number
}

export type ConditionalDisplayRow = {
  target: ModifierTarget
  amount: number
}

export type ConditionalEffect = {
  source: string
  bonusType: string
  amount: number
  note: string
  target: ModifierTarget
  flag?: string
}

export type Contribution = {
  source: string
  bonusType: string
  amount: number
  note?: string
}

export type DefenseBreakdown = {
  base: number
  itemBonus: number
  total: number
  dexApplied: boolean
  vsMelee: number
  vsRanged: number
  contributions: BreakdownContribution[]
}

export type EquippedFlag = {
  flag: string
  source: string
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

export type ItemEffects = {
  byTarget: Record<string, AggregatedStat>
  flags: string[]
  conditional: ConditionalEffect[]
}

export type Modifier = {
  target: ModifierTarget
  amount: number
  bonusType: string
  condition?: ModifierCondition | null
  note?: string
  scale?: VitalScale | null
}

export type ModifierCondition = {
  c: string
  type?: string
  trait?: string
  note?: string
  flag?: string
  label?: string
}

export type ModifierTarget = {
  k: string
  name?: string
  attribute?: string
  scope?: string
  school?: string
}

export type PointBuyStatus = {
  spent: number | null
  warnings: string[]
}

export type RdBreakdown = {
  total: number
  sources: SourceAmount[]
}

export type SourceAmount = {
  source: string
  amount: number
}

export type TempHpBreakdown = {
  total: number
  sources: SourceAmount[]
}

export type TotalContribs = {
  total: number
  contributions: BreakdownContribution[]
}

export type ValueBreakdown = {
  base: number
  itemBonus: number
  total: number
  contributions: BreakdownContribution[]
}

export type VitalContext = {
  level: number
  classes: ClassEntry[]
  raceId: string
  raceAbilityChoices: string[]
  powerIds: string[]
  classChoices: Record<string, ClassChoiceSelections>
  godPower: string
  origin: string
  originChoices: string[]
  attrTotals: Record<AttributeKey, number>
}

export type VitalPools = {
  pvMax: number
  pmMax: number
}

export type VitalScale = {
  per: string
  step?: number
  round?: string
  attribute?: string
}

export type WeaponCard = {
  name: string
  skill: string
  attribute: AttributeKey
  attack: number
  expertise: ExpertiseBreakdown
  attackAll: TotalContribs
  damage: string
  strDamage: number
  damageBonus: number
  damageAll: TotalContribs
  critRange: number
  critMult: number
}
