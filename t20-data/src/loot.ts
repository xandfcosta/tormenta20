/**
 * Loot generator — PDF Cap 8 (Recompensas, p328-329) Tabela 8-1
 * (Tesouro por Nível de Desafio) + Tabela 8-2 (Riquezas, p330).
 *
 * **Out of scope this module**: sub-rolls for specific item names from
 * Tabelas 8-3 (Itens Diversos), 8-4 (Equipamento), 8-5 (Itens
 * Superiores). Esses preenchem o NOME do item; este módulo decide o
 * KIND (e.g. "superior com 3 melhorias", "mágico menor") + monetário.
 *
 * Deterministic when given a seeded `Rng` (`loot-rng.ts`).
 *
 * Coin abbreviations (PDF p140):
 *  - TC = Tibar de Cobre (T$ 0.1)
 *  - TS = Tibar de prata (default T$, "T$ N")
 *  - TO = Tibar de Ouro (T$ 10)
 */
import { rollFormula, rollPercentile, type Rng } from './loot-rng'

// ─── Types ──────────────────────────────────────────────────────────
export type CoinCurrency = 'TC' | 'TS' | 'TO'
export type WealthTier = 'menor' | 'media' | 'maior'
export type LootMagicTier = 'menor' | 'medio' | 'maior'

export type MoneyOutcome =
  | { kind: 'none' }
  | { kind: 'coins'; formula: string; currency: CoinCurrency; bonusPct?: 20 }
  | {
      kind: 'wealth'
      countFormula: string
      tier: WealthTier
      bonusPct?: 20
    }

export type ItemOutcome =
  | { kind: 'none' }
  | { kind: 'diverso' }
  | { kind: 'equipamento'; double?: true }
  | { kind: 'potion'; countFormula: string; bonusPct?: 20 }
  | { kind: 'superior'; improvements: 1 | 2 | 3 | 4; double?: true }
  | { kind: 'magic'; tier: LootMagicTier; double?: true }

type LootRow<T> = readonly [lo: number, hi: number, outcome: T]

export type LootNdEntry = {
  nd: string
  moneyRows: readonly LootRow<MoneyOutcome>[]
  itemRows: readonly LootRow<ItemOutcome>[]
}

export type RolledLoot = {
  nd: string
  money: { outcome: MoneyOutcome; tibarValue: number; roll: number }
  item: { outcome: ItemOutcome; roll: number }
}

// ─── Tabela 8-2: Riquezas (p330) ────────────────────────────────────
/** Valor médio em T$ por riqueza (selling price). */
export const WEALTH_VALUE_TIBAR: Record<WealthTier, number> = Object.freeze({
  menor: 50,
  media: 700,
  maior: 27000,
})

// ─── Tabela 8-1: Tesouro por ND (p328-329) ──────────────────────────
/** Coin formula → tibar value. TC = ×0.1, TS = ×1, TO = ×10. */
function coinsToTibar(amount: number, currency: CoinCurrency): number {
  if (currency === 'TC') return amount * 0.1
  if (currency === 'TO') return amount * 10
  return amount
}

const T = (
  nd: string,
  moneyRows: readonly LootRow<MoneyOutcome>[],
  itemRows: readonly LootRow<ItemOutcome>[],
): LootNdEntry => ({ nd, moneyRows, itemRows })

/* eslint-disable prettier/prettier */
export const TREASURE_BY_ND: readonly LootNdEntry[] = Object.freeze([
  T('1/4',
    [
      [1, 30, { kind: 'none' }],
      [31, 70, { kind: 'coins', formula: '1d6×10', currency: 'TC' }],
      [71, 95, { kind: 'coins', formula: '1d4×100', currency: 'TC' }],
      [96, 100, { kind: 'coins', formula: '1d6×10', currency: 'TS' }],
    ],
    [
      [1, 50, { kind: 'none' }],
      [51, 75, { kind: 'diverso' }],
      [76, 100, { kind: 'equipamento' }],
    ],
  ),
  T('1/2',
    [
      [1, 25, { kind: 'none' }],
      [26, 70, { kind: 'coins', formula: '2d6×10', currency: 'TC' }],
      [71, 95, { kind: 'coins', formula: '2d8×10', currency: 'TS' }],
      [96, 100, { kind: 'coins', formula: '1d4×100', currency: 'TS' }],
    ],
    [
      [1, 45, { kind: 'none' }],
      [46, 70, { kind: 'diverso' }],
      [71, 100, { kind: 'equipamento' }],
    ],
  ),
  T('1',
    [
      [1, 20, { kind: 'none' }],
      [21, 70, { kind: 'coins', formula: '3d8×10', currency: 'TS' }],
      [71, 95, { kind: 'coins', formula: '4d12×10', currency: 'TS' }],
      [96, 100, { kind: 'wealth', countFormula: '1', tier: 'menor' }],
    ],
    [
      [1, 40, { kind: 'none' }],
      [41, 65, { kind: 'diverso' }],
      [66, 90, { kind: 'equipamento' }],
      [91, 100, { kind: 'potion', countFormula: '1' }],
    ],
  ),
  T('2',
    [
      [1, 15, { kind: 'none' }],
      [16, 55, { kind: 'coins', formula: '3d10×10', currency: 'TS' }],
      [56, 85, { kind: 'coins', formula: '2d4×100', currency: 'TS' }],
      [86, 95, { kind: 'coins', formula: '2d6+1×100', currency: 'TS' }],
      [96, 100, { kind: 'wealth', countFormula: '1', tier: 'menor' }],
    ],
    [
      [1, 30, { kind: 'none' }],
      [31, 40, { kind: 'diverso' }],
      [41, 70, { kind: 'equipamento' }],
      [71, 90, { kind: 'potion', countFormula: '1' }],
      [91, 100, { kind: 'superior', improvements: 1 }],
    ],
  ),
  T('3',
    [
      [1, 10, { kind: 'none' }],
      [11, 20, { kind: 'coins', formula: '4d12×10', currency: 'TS' }],
      [21, 60, { kind: 'coins', formula: '1d4×100', currency: 'TS' }],
      [61, 90, { kind: 'coins', formula: '1d8×10', currency: 'TO' }],
      [91, 100, { kind: 'wealth', countFormula: '1d3', tier: 'menor' }],
    ],
    [
      [1, 25, { kind: 'none' }],
      [26, 35, { kind: 'diverso' }],
      [36, 60, { kind: 'equipamento' }],
      [61, 85, { kind: 'potion', countFormula: '1' }],
      [86, 100, { kind: 'superior', improvements: 1 }],
    ],
  ),
  T('4',
    [
      [1, 10, { kind: 'none' }],
      [11, 50, { kind: 'coins', formula: '1d6×100', currency: 'TS' }],
      [51, 80, { kind: 'coins', formula: '1d12×100', currency: 'TS' }],
      [81, 90, { kind: 'wealth', countFormula: '1', tier: 'menor', bonusPct: 20 }],
      [91, 100, { kind: 'wealth', countFormula: '1d3', tier: 'menor', bonusPct: 20 }],
    ],
    [
      [1, 20, { kind: 'none' }],
      [21, 30, { kind: 'diverso' }],
      [31, 55, { kind: 'equipamento', double: true }],
      [56, 80, { kind: 'potion', countFormula: '1', bonusPct: 20 }],
      [81, 100, { kind: 'superior', improvements: 1, double: true }],
    ],
  ),
  T('5',
    [
      [1, 15, { kind: 'none' }],
      [16, 65, { kind: 'coins', formula: '1d8×100', currency: 'TS' }],
      [66, 95, { kind: 'coins', formula: '3d4×10', currency: 'TO' }],
      [96, 100, { kind: 'wealth', countFormula: '1', tier: 'media' }],
    ],
    [
      [1, 20, { kind: 'none' }],
      [21, 70, { kind: 'potion', countFormula: '1' }],
      [71, 90, { kind: 'superior', improvements: 1 }],
      [91, 100, { kind: 'superior', improvements: 2 }],
    ],
  ),
  T('6',
    [
      [1, 15, { kind: 'none' }],
      [16, 60, { kind: 'coins', formula: '2d6×100', currency: 'TS' }],
      [61, 90, { kind: 'coins', formula: '2d10×100', currency: 'TS' }],
      [91, 100, { kind: 'wealth', countFormula: '1d3+1', tier: 'menor' }],
    ],
    [
      [1, 20, { kind: 'none' }],
      [21, 65, { kind: 'potion', countFormula: '1', bonusPct: 20 }],
      [66, 95, { kind: 'superior', improvements: 1 }],
      [96, 100, { kind: 'superior', improvements: 2, double: true }],
    ],
  ),
  T('7',
    [
      [1, 10, { kind: 'none' }],
      [11, 60, { kind: 'coins', formula: '2d8×100', currency: 'TS' }],
      [61, 90, { kind: 'coins', formula: '2d12×10', currency: 'TO' }],
      [91, 100, { kind: 'wealth', countFormula: '1d4+1', tier: 'menor' }],
    ],
    [
      [1, 20, { kind: 'none' }],
      [21, 60, { kind: 'potion', countFormula: '1d3' }],
      [61, 90, { kind: 'superior', improvements: 2 }],
      [91, 100, { kind: 'superior', improvements: 3 }],
    ],
  ),
  T('8',
    [
      [1, 10, { kind: 'none' }],
      [11, 55, { kind: 'coins', formula: '2d10×100', currency: 'TS' }],
      [56, 95, { kind: 'wealth', countFormula: '1d4+1', tier: 'menor' }],
      [96, 100, { kind: 'wealth', countFormula: '1', tier: 'media', bonusPct: 20 }],
    ],
    [
      [1, 20, { kind: 'none' }],
      [21, 75, { kind: 'potion', countFormula: '1d3' }],
      [76, 95, { kind: 'superior', improvements: 2 }],
      [96, 100, { kind: 'superior', improvements: 3, double: true }],
    ],
  ),
  T('9',
    [
      [1, 10, { kind: 'none' }],
      [11, 35, { kind: 'wealth', countFormula: '1', tier: 'media' }],
      [36, 85, { kind: 'coins', formula: '4d6×100', currency: 'TS' }],
      [86, 100, { kind: 'wealth', countFormula: '1d3', tier: 'media' }],
    ],
    [
      [1, 20, { kind: 'none' }],
      [21, 70, { kind: 'potion', countFormula: '1', bonusPct: 20 }],
      [71, 95, { kind: 'superior', improvements: 3 }],
      [96, 100, { kind: 'magic', tier: 'menor' }],
    ],
  ),
  T('10',
    [
      [1, 10, { kind: 'none' }],
      [11, 30, { kind: 'coins', formula: '4d6×100', currency: 'TS' }],
      [31, 85, { kind: 'coins', formula: '4d10×10', currency: 'TO' }],
      [86, 100, { kind: 'wealth', countFormula: '1d3+1', tier: 'media' }],
    ],
    [
      [1, 50, { kind: 'none' }],
      [51, 75, { kind: 'potion', countFormula: '1d3+1' }],
      [76, 90, { kind: 'superior', improvements: 3 }],
      [91, 100, { kind: 'magic', tier: 'menor' }],
    ],
  ),
  T('11',
    [
      [1, 10, { kind: 'none' }],
      [11, 45, { kind: 'coins', formula: '2d4×1.000', currency: 'TS' }],
      [46, 85, { kind: 'wealth', countFormula: '1d3', tier: 'media' }],
      [86, 100, { kind: 'coins', formula: '2d6×100', currency: 'TO' }],
    ],
    [
      [1, 45, { kind: 'none' }],
      [46, 70, { kind: 'potion', countFormula: '1d4+1' }],
      [71, 90, { kind: 'superior', improvements: 3 }],
      [91, 100, { kind: 'magic', tier: 'menor', double: true }],
    ],
  ),
  T('12',
    [
      [1, 10, { kind: 'none' }],
      [11, 45, { kind: 'wealth', countFormula: '1', tier: 'media', bonusPct: 20 }],
      [46, 80, { kind: 'coins', formula: '2d6×1.000', currency: 'TS' }],
      [81, 100, { kind: 'wealth', countFormula: '1d4+1', tier: 'media' }],
    ],
    [
      [1, 45, { kind: 'none' }],
      [46, 70, { kind: 'potion', countFormula: '1d3+1', bonusPct: 20 }],
      [71, 85, { kind: 'superior', improvements: 4 }],
      [86, 100, { kind: 'magic', tier: 'menor' }],
    ],
  ),
  T('13',
    [
      [1, 10, { kind: 'none' }],
      [11, 45, { kind: 'coins', formula: '4d4×1.000', currency: 'TS' }],
      [46, 80, { kind: 'wealth', countFormula: '1d3+1', tier: 'media' }],
      [81, 100, { kind: 'coins', formula: '4d6×100', currency: 'TO' }],
    ],
    [
      [1, 40, { kind: 'none' }],
      [41, 65, { kind: 'potion', countFormula: '1d4+1', bonusPct: 20 }],
      [66, 95, { kind: 'superior', improvements: 4 }],
      [96, 100, { kind: 'magic', tier: 'medio' }],
    ],
  ),
  T('14',
    [
      [1, 10, { kind: 'none' }],
      [11, 45, { kind: 'wealth', countFormula: '1d3+1', tier: 'media' }],
      [46, 80, { kind: 'coins', formula: '3d6×1.000', currency: 'TS' }],
      [81, 100, { kind: 'wealth', countFormula: '1', tier: 'maior' }],
    ],
    [
      [1, 40, { kind: 'none' }],
      [41, 65, { kind: 'potion', countFormula: '1d4+1', bonusPct: 20 }],
      [66, 90, { kind: 'superior', improvements: 4 }],
      [91, 100, { kind: 'magic', tier: 'medio' }],
    ],
  ),
  T('15',
    [
      [1, 10, { kind: 'none' }],
      [11, 45, { kind: 'wealth', countFormula: '1', tier: 'media', bonusPct: 20 }],
      [46, 80, { kind: 'coins', formula: '2d10×1.000', currency: 'TS' }],
      [81, 100, { kind: 'coins', formula: '1d4×1.000', currency: 'TO' }],
    ],
    [
      [1, 35, { kind: 'none' }],
      [36, 45, { kind: 'potion', countFormula: '1d6+1' }],
      [46, 85, { kind: 'superior', improvements: 4, double: true }],
      [86, 100, { kind: 'magic', tier: 'medio' }],
    ],
  ),
  T('16',
    [
      [1, 10, { kind: 'none' }],
      [11, 40, { kind: 'coins', formula: '3d6×1.000', currency: 'TS' }],
      [41, 75, { kind: 'coins', formula: '3d10×100', currency: 'TO' }],
      [76, 100, { kind: 'wealth', countFormula: '1d3', tier: 'maior' }],
    ],
    [
      [1, 35, { kind: 'none' }],
      [36, 45, { kind: 'potion', countFormula: '1d6+1', bonusPct: 20 }],
      [46, 80, { kind: 'superior', improvements: 4, double: true }],
      [81, 100, { kind: 'magic', tier: 'medio' }],
    ],
  ),
  T('17',
    [
      [1, 5, { kind: 'none' }],
      [6, 40, { kind: 'coins', formula: '4d6×1.000', currency: 'TS' }],
      [41, 75, { kind: 'wealth', countFormula: '1d3', tier: 'media', bonusPct: 20 }],
      [76, 100, { kind: 'coins', formula: '2d4×1.000', currency: 'TO' }],
    ],
    [
      [1, 20, { kind: 'none' }],
      [21, 40, { kind: 'magic', tier: 'menor' }],
      [41, 80, { kind: 'magic', tier: 'medio' }],
      [81, 100, { kind: 'magic', tier: 'maior' }],
    ],
  ),
  T('18',
    [
      [1, 5, { kind: 'none' }],
      [6, 40, { kind: 'coins', formula: '4d10×1.000', currency: 'TS' }],
      [41, 75, { kind: 'wealth', countFormula: '1', tier: 'maior' }],
      [76, 100, { kind: 'wealth', countFormula: '1d3+1', tier: 'maior' }],
    ],
    [
      [1, 15, { kind: 'none' }],
      [16, 40, { kind: 'magic', tier: 'menor', double: true }],
      [41, 70, { kind: 'magic', tier: 'medio' }],
      [71, 100, { kind: 'magic', tier: 'maior' }],
    ],
  ),
  T('19',
    [
      [1, 5, { kind: 'none' }],
      [6, 40, { kind: 'coins', formula: '4d12×1.000', currency: 'TS' }],
      [41, 75, { kind: 'wealth', countFormula: '1', tier: 'maior', bonusPct: 20 }],
      [76, 100, { kind: 'coins', formula: '1d12×1.000', currency: 'TO' }],
    ],
    [
      [1, 10, { kind: 'none' }],
      [11, 40, { kind: 'magic', tier: 'menor', double: true }],
      [41, 60, { kind: 'magic', tier: 'medio', double: true }],
      [61, 100, { kind: 'magic', tier: 'maior' }],
    ],
  ),
  T('20',
    [
      [1, 5, { kind: 'none' }],
      [6, 40, { kind: 'coins', formula: '2d4×1.000', currency: 'TO' }],
      [41, 75, { kind: 'wealth', countFormula: '1d3', tier: 'maior' }],
      [76, 100, { kind: 'wealth', countFormula: '1d3+1', tier: 'maior', bonusPct: 20 }],
    ],
    [
      [1, 5, { kind: 'none' }],
      [6, 40, { kind: 'magic', tier: 'menor', double: true }],
      [41, 50, { kind: 'magic', tier: 'medio', double: true }],
      [51, 100, { kind: 'magic', tier: 'maior', double: true }],
    ],
  ),
])
/* eslint-enable prettier/prettier */

const byNd = new Map(TREASURE_BY_ND.map((e) => [e.nd, e]))

export function lootEntryForNd(nd: string): LootNdEntry | undefined {
  return byNd.get(nd)
}

/** Resolve d% roll → outcome row. */
function findRow<T>(
  rows: readonly LootRow<T>[],
  roll: number,
): LootRow<T> {
  for (const row of rows) {
    if (roll >= row[0] && roll <= row[1]) return row
  }
  throw new Error(`findRow: no row matches roll ${roll}`)
}

function applyBonus(value: number, bonusPct?: 20): number {
  if (!bonusPct) return value
  return Math.floor(value * (1 + bonusPct / 100))
}

function rollMoneyValue(rng: Rng, outcome: MoneyOutcome): number {
  if (outcome.kind === 'none') return 0
  if (outcome.kind === 'coins') {
    const amount = rollFormula(rng, outcome.formula)
    return applyBonus(coinsToTibar(amount, outcome.currency), outcome.bonusPct)
  }
  const count = rollCount(rng, outcome.countFormula)
  const unit = WEALTH_VALUE_TIBAR[outcome.tier]
  return applyBonus(count * unit, outcome.bonusPct)
}

/** Convert "1d3" / "1d4+1" → formula consumable; "1" → fixed integer 1. */
function rollCount(rng: Rng, s: string): number {
  if (/^\d+$/.test(s)) return Number(s)
  return rollFormula(rng, s)
}

/**
 * Roll a single ND's loot. Returns money outcome + item outcome + the
 * d% rolls used (for debug / snapshot).
 *
 * Throws if `nd` is not a known T20 ND string ('1/4'..'20').
 */
export function rollLootForNd(rng: Rng, nd: string): RolledLoot {
  const entry = byNd.get(nd)
  if (!entry) throw new Error(`rollLootForNd: unknown nd "${nd}"`)
  const moneyRoll = rollPercentile(rng)
  const moneyRow = findRow(entry.moneyRows, moneyRoll)
  const tibarValue = rollMoneyValue(rng, moneyRow[2])
  const itemRoll = rollPercentile(rng)
  const itemRow = findRow(entry.itemRows, itemRoll)
  return {
    nd,
    money: { outcome: moneyRow[2], tibarValue, roll: moneyRoll },
    item: { outcome: itemRow[2], roll: itemRoll },
  }
}
