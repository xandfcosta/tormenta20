/**
 * Magic armor/shield random tables — PDF Cap 8 Recompensas, p339-340.
 *
 * Tabela 8-10 (Armaduras e Escudos Mágicos) — d% pra encanto, ou
 * 91-100 rota pra item específico (Tabela 8-11).
 * Tabela 8-11 (Armaduras e Escudos Específicos) — d% pick de item nomeado.
 *
 * Footnotes 8-10:
 *  ¹ shieldOnly — apenas escudos; pra armaduras, role novamente.
 *  ² countsAsTwo — conta como dois encantos; pra menores, role novamente.
 *
 * Deterministic com seeded Rng.
 */
import { rollPercentile, type Rng } from './loot-rng'
import type { LootMagicTier } from './loot'

/** Row flag — nenhuma row do PDF combina os dois. */
export type ArmorEnchantFlag = 'shieldOnly' | 'countsAsTwo'

type ArmorEnchantRow = readonly [
  lo: number,
  hi: number,
  name: string,
  flag?: ArmorEnchantFlag,
]

type SpecificArmorRow = readonly [lo: number, hi: number, name: string]

// ─── Tabela 8-10: Armaduras & Escudos Mágicos (p339) ────────────────
export const ARMOR_ENCHANT_ROWS: readonly ArmorEnchantRow[] = Object.freeze([
  [1, 6, 'Abascanto'],
  [7, 10, 'Abençoado'],
  [11, 12, 'Acrobático'],
  [13, 14, 'Alado'],
  [15, 16, 'Animado', 'shieldOnly'],
  [17, 18, 'Assustador'],
  [19, 22, 'Cáustica'],
  [23, 32, 'Defensor'],
  [33, 34, 'Escorregadio'],
  [35, 36, 'Esmagador', 'shieldOnly'],
  [37, 38, 'Fantasmagórico'],
  [39, 40, 'Fortificado'],
  [41, 44, 'Gélido'],
  [45, 54, 'Guardião', 'countsAsTwo'],
  [55, 56, 'Hipnótico'],
  [57, 58, 'Ilusório'],
  [59, 62, 'Incandescente'],
  [63, 68, 'Invulnerável'],
  [69, 72, 'Opaco'],
  [73, 78, 'Protetor'],
  [79, 80, 'Refletor'],
  [81, 84, 'Relampejante'],
  [85, 86, 'Reluzente'],
  [87, 88, 'Sombrio'],
  [89, 90, 'Zeloso'],
  // 91-100: rota pra Tabela 8-11 (item específico) — sentinel
  [91, 100, 'Item específico'],
])

// ─── Tabela 8-11: Armaduras & Escudos Específicos (p340) ────────────
export const SPECIFIC_ARMOR_ROWS: readonly SpecificArmorRow[] = Object.freeze([
  [1, 10, 'Cota élfica'],
  [11, 20, 'Couro de monstro'],
  [21, 25, 'Escudo do conjurador'],
  [26, 32, 'Loriga do centurião'],
  [33, 42, 'Manto da noite'],
  [43, 49, 'Couraça do comando'],
  [50, 59, 'Baluarte anão'],
  [60, 66, 'Escudo espinhoso'],
  [67, 76, 'Escudo do leão'],
  [77, 83, 'Carapaça demoníaca'],
  [84, 88, 'Escudo do eclipse'],
  [89, 93, 'Escudo de Azgher'],
  [94, 100, 'Armadura da luz'],
])

// ─── Types ──────────────────────────────────────────────────────────
export type MagicArmorResult =
  | {
      kind: 'encanto'
      name: string
      countsAsTwo: boolean
      tier: LootMagicTier
      isShield: boolean
    }
  | { kind: 'specific'; name: string; tier: LootMagicTier; isShield: boolean }

// ─── Resolver ───────────────────────────────────────────────────────
function rollSpecificArmor(rng: Rng): string {
  const roll = rollPercentile(rng)
  const hit = SPECIFIC_ARMOR_ROWS.find((r) => roll >= r[0] && roll <= r[1])
  if (!hit) {
    throw new Error(`rollSpecificArmor: no row matches d% ${roll}`)
  }
  return hit[2]
}

function rollEnchantRow(rng: Rng): ArmorEnchantRow {
  const roll = rollPercentile(rng)
  const hit = ARMOR_ENCHANT_ROWS.find((r) => roll >= r[0] && roll <= r[1])
  if (!hit) {
    throw new Error(`rollMagicArmor: no row matches d% ${roll}`)
  }
  return hit
}

/**
 * Roll magic armor/shield via Tabela 8-10 (+ 8-11 se 91-100).
 * Footnotes:
 *  ¹ shieldOnly encantos rerollam se !isShield.
 *  ² countsAsTwo encantos rerollam se tier === 'menor'.
 * Safety cap 16 tentativas.
 */
export function rollMagicArmor(
  rng: Rng,
  tier: LootMagicTier,
  isShield: boolean,
): MagicArmorResult {
  let safety = 0
  while (safety < 16) {
    safety++
    const row = rollEnchantRow(rng)
    const flag = row[3]

    if (flag === 'shieldOnly' && !isShield) continue
    if (flag === 'countsAsTwo' && tier === 'menor') continue

    if (row[2] === 'Item específico') {
      return {
        kind: 'specific',
        name: rollSpecificArmor(rng),
        tier,
        isShield,
      }
    }
    return {
      kind: 'encanto',
      name: row[2],
      countsAsTwo: flag === 'countsAsTwo',
      tier,
      isShield,
    }
  }
  throw new Error(
    `rollMagicArmor: exceeded reroll safety (tier=${tier}, isShield=${isShield})`,
  )
}
