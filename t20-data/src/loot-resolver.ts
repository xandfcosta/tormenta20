/**
 * Loot resolver — compõe `rollLootForNd` (Tabela 8-1 kind outcomes)
 * com sub-roll resolvers de:
 *  - `loot-sub-rolls.ts` (Tabelas 8-3/8-4/8-5)
 *  - `magic-weapon-tables.ts` (Tabelas 8-8/8-9)
 *  - `magic-armor-tables.ts` (Tabelas 8-10/8-11)
 *  - `magic-potion-tables.ts` (Tabela 8-12)
 *  - `magic-accessory-tables.ts` (Tabelas 8-13/14/15)
 *
 * Full loot resolution — todos outcomes retornam nomes concretos.
 *
 * PDF Cap 8, p328-343. Deterministic com seeded Rng.
 *
 * Magic kind router (p330): 1d6 → 1-2 arma, 3 armadura/escudo,
 * 4-6 acessório. isShield derivado do nome base equip (prefixo "Escudo").
 */
import { randInt, rollFormula, type Rng } from './loot-rng'
import { rollLootForNd, type ItemOutcome, type LootMagicTier } from './loot'
import type { RolledLoot } from './loot'
import {
  rollDiverso,
  rollEquipamento,
  rollSuperior,
  type EquipKind,
  type SuperiorProperty,
} from './loot-sub-rolls'
import {
  rollMagicWeapon,
  type MagicWeaponResult,
} from './magic-weapon-tables'
import {
  rollMagicArmor,
  type MagicArmorResult,
} from './magic-armor-tables'
import { rollMagicAccessory } from './magic-accessory-tables'
import {
  rollMagicPotion,
  type PotionRoll,
} from './magic-potion-tables'

// ─── Types ──────────────────────────────────────────────────────────
export type LootMagicItemKind = 'arma' | 'armadura' | 'acessorio'

export type ResolvedItem =
  | { kind: 'none' }
  | { kind: 'diverso'; name: string }
  | { kind: 'equipamento'; equipKind: EquipKind; name: string }
  | { kind: 'potion'; count: number; potions: readonly PotionRoll[] }
  | {
      kind: 'superior'
      equipKind: EquipKind
      baseName: string
      properties: readonly SuperiorProperty[]
    }
  | {
      kind: 'magic'
      magicKind: 'arma'
      tier: LootMagicTier
      baseName: string
      weapon: MagicWeaponResult
    }
  | {
      kind: 'magic'
      magicKind: 'armadura'
      tier: LootMagicTier
      baseName: string
      isShield: boolean
      armor: MagicArmorResult
    }
  | {
      kind: 'magic'
      magicKind: 'acessorio'
      tier: LootMagicTier
      name: string
    }

export type ResolvedLoot = {
  nd: string
  money: RolledLoot['money']
  items: readonly ResolvedItem[]
  itemRoll: number
}

// ─── Magic kind router (p330 1d6) ───────────────────────────────────
function resolveMagic(rng: Rng, tier: LootMagicTier): ResolvedItem {
  const d6 = randInt(rng, 1, 6)
  if (d6 <= 2) {
    // 1-2 → arma. Roll base weapon name + magic weapon result.
    const base = rollEquipamento(rng, 'arma')
    const weapon = rollMagicWeapon(rng, tier)
    return {
      kind: 'magic',
      magicKind: 'arma',
      tier,
      baseName: base.name,
      weapon,
    }
  }
  if (d6 === 3) {
    // 3 → armadura/escudo. Roll base first pra saber isShield.
    const base = rollEquipamento(rng, 'armadura')
    const isShield = base.name.startsWith('Escudo')
    const armor = rollMagicArmor(rng, tier, isShield)
    return {
      kind: 'magic',
      magicKind: 'armadura',
      tier,
      baseName: base.name,
      isShield,
      armor,
    }
  }
  // 4-6 → acessório.
  const acc = rollMagicAccessory(rng, tier)
  return { kind: 'magic', magicKind: 'acessorio', tier, name: acc.name }
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
      const potions: PotionRoll[] = []
      for (let i = 0; i < count; i++) potions.push(rollMagicPotion(rng))
      return { kind: 'potion', count, potions }
    }
    case 'superior': {
      const sup = rollSuperior(rng, outcome.improvements)
      const base = rollEquipamento(rng, sup.kind)
      return {
        kind: 'superior',
        equipKind: sup.kind,
        baseName: base.name,
        properties: sup.properties,
      }
    }
    case 'magic':
      return resolveMagic(rng, outcome.tier)
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
 * via sub-tabelas 8-3/8-4/8-5 + magic tabelas 8-8..8-15. Retorna
 * items[] (0-2 elementos — 0 se "none", 2 se row marca "role duas
 * vezes").
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
