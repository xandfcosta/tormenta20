/**
 * Loot resolver — compõe `rollLootForNd` (Tabela 8-1 kind outcomes)
 * com sub-roll resolvers de `loot-sub-rolls.ts` (Tabelas 8-3/8-4/8-5)
 * pra produzir loot final com nomes concretos de itens.
 *
 * PDF Cap 8, p328-332. Deterministic com seeded Rng.
 *
 * `magic` tier stays opaque — magic item catalog resolution é
 * follow-up separado (precisa Tabela 4-30 / 4-31 magic item lists).
 */
import { rollFormula, type Rng } from './loot-rng'
import { rollLootForNd, type ItemOutcome, type LootMagicTier } from './loot'
import type { RolledLoot } from './loot'
import {
  rollDiverso,
  rollEquipamento,
  rollSuperior,
  type EquipKind,
  type SuperiorProperty,
} from './loot-sub-rolls'

// ─── Types ──────────────────────────────────────────────────────────
export type ResolvedItem =
  | { kind: 'none' }
  | { kind: 'diverso'; name: string }
  | { kind: 'equipamento'; equipKind: EquipKind; name: string }
  | { kind: 'potion'; count: number }
  | {
      kind: 'superior'
      equipKind: EquipKind
      baseName: string
      properties: readonly SuperiorProperty[]
    }
  | { kind: 'magic'; tier: LootMagicTier }

export type ResolvedLoot = {
  nd: string
  money: RolledLoot['money']
  items: readonly ResolvedItem[]
  itemRoll: number
}

// ─── Resolvers ──────────────────────────────────────────────────────
function resolveSingle(rng: Rng, outcome: ItemOutcome): ResolvedItem {
  switch (outcome.kind) {
    case 'none':
      return { kind: 'none' }
    case 'diverso':
      return { kind: 'diverso', name: rollDiverso(rng) }
    case 'equipamento': {
      const eq = rollEquipamento(rng)
      return { kind: 'equipamento', equipKind: eq.kind, name: eq.name }
    }
    case 'potion': {
      const base = rollFormula(rng, outcome.countFormula)
      const count = outcome.bonusPct === 20 ? Math.floor(base * 1.2) : base
      return { kind: 'potion', count }
    }
    case 'superior': {
      const sup = rollSuperior(rng, outcome.improvements)
      // Base equip name usa mesmo kind rolado pela superior
      const base = rollEquipamento(rng, sup.kind)
      return {
        kind: 'superior',
        equipKind: sup.kind,
        baseName: base.name,
        properties: sup.properties,
      }
    }
    case 'magic':
      return { kind: 'magic', tier: outcome.tier }
  }
}

function resolveOutcome(
  rng: Rng,
  outcome: ItemOutcome,
): readonly ResolvedItem[] {
  if (outcome.kind === 'none') return []
  const double =
    (outcome.kind === 'equipamento' ||
      outcome.kind === 'superior' ||
      outcome.kind === 'magic') &&
    outcome.double === true
  const times = double ? 2 : 1
  const out: ResolvedItem[] = []
  for (let i = 0; i < times; i++) {
    out.push(resolveSingle(rng, outcome))
  }
  return out
}

/**
 * Full loot roll: rola Tabela 8-1 pra kind, então resolve nomes
 * via sub-tabelas 8-3/8-4/8-5. Retorna items[] (0-2 elementos —
 * 0 se "none", 2 se row marca "role duas vezes").
 */
export function resolveLootForNd(rng: Rng, nd: string): ResolvedLoot {
  const base = rollLootForNd(rng, nd)
  const items = resolveOutcome(rng, base.item.outcome)
  return {
    nd: base.nd,
    money: base.money,
    items,
    itemRoll: base.item.roll,
  }
}
