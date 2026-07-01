/**
 * Magic accessory random tables — PDF Cap 8 Recompensas, p342-343.
 *
 * Tabelas 8-13 (Acessórios Menores), 8-14 (Acessórios Médios),
 * 8-15 (Acessórios Maiores). d% pick de acessório específico por
 * tier. Todas cobertura contígua [1, 100].
 *
 * Kind routing (p330): quando Tabela 8-1 produz `magic` outcome,
 * role 1d6 — 1-2 arma (Tabela 8-8, TODO), 3 armadura/escudo
 * (Tabela 8-10, TODO), 4-6 acessório (esta tabela, casada por tier).
 */
import { rollPercentile, type Rng } from './loot-rng'
import type { LootMagicTier } from './loot'

type AccessoryRow = readonly [lo: number, hi: number, name: string]

// ─── Tabela 8-13: Acessórios Menores (p342) ─────────────────────────
export const ACCESSORIES_MENOR_ROWS: readonly AccessoryRow[] = Object.freeze([
  [1, 2, 'Anel do sustento'],
  [3, 7, 'Bainha mágica'],
  [8, 12, 'Corda da escalada'],
  [13, 14, 'Ferraduras da velocidade'],
  [15, 19, 'Garrafa da fumaça eterna'],
  [20, 24, 'Gema da luminosidade'],
  [25, 29, 'Manto élfico'],
  [30, 34, 'Mochila de carga'],
  [35, 40, 'Brincos da sagacidade'],
  [41, 46, 'Luvas da delicadeza'],
  [47, 52, 'Manoplas da força do ogro'],
  [53, 59, 'Manto da resistência'],
  [60, 65, 'Manto do fascínio'],
  [66, 71, 'Pingente da sensatez'],
  [72, 77, 'Torque do vigor'],
  [78, 82, 'Chapéu do disfarce'],
  [83, 84, 'Flauta fantasma'],
  [85, 89, 'Lanterna da revelação'],
  [90, 96, 'Anel da proteção'],
  [97, 98, 'Anel do escudo mental'],
  [99, 100, 'Pingente da saúde'],
])

// ─── Tabela 8-14: Acessórios Médios (p343) ──────────────────────────
export const ACCESSORIES_MEDIO_ROWS: readonly AccessoryRow[] = Object.freeze([
  [1, 4, 'Anel de telecinesia'],
  [5, 8, 'Bola de cristal'],
  [9, 10, 'Caveira maldita'],
  [11, 14, 'Botas aladas'],
  [15, 18, 'Braceletes de bronze'],
  [19, 24, 'Anel da energia'],
  [25, 30, 'Anel da vitalidade'],
  [31, 34, 'Anel de invisibilidade'],
  [35, 38, 'Braçadeiras do arqueiro'],
  [39, 42, 'Brincos de Marah'],
  [43, 46, 'Faixas do pugilista'],
  [47, 50, 'Manto da aranha'],
  [51, 54, 'Vassoura voadora'],
  [55, 58, 'Símbolo abençoado'],
  [59, 64, 'Amuleto da robustez'],
  [65, 68, 'Botas velozes'],
  [69, 74, 'Cinto da força do gigante'],
  [75, 80, 'Coroa majestosa'],
  [81, 86, 'Estola da serenidade'],
  [87, 88, 'Manto do morcego'],
  [89, 94, 'Pulseiras da celeridade'],
  [95, 100, 'Tiara da sapiência'],
])

// ─── Tabela 8-15: Acessórios Maiores (p343) ─────────────────────────
export const ACCESSORIES_MAIOR_ROWS: readonly AccessoryRow[] = Object.freeze([
  [1, 2, 'Elmo do teletransporte'],
  [3, 4, 'Gema da telepatia'],
  [5, 9, 'Gema elemental'],
  [10, 15, 'Manual da saúde corporal'],
  [16, 21, 'Manual do bom exercício'],
  [22, 27, 'Manual dos movimentos precisos'],
  [28, 34, 'Medalhão de Lena'],
  [35, 40, 'Tomo da compreensão'],
  [41, 46, 'Tomo da liderança e influência'],
  [47, 52, 'Tomo dos grandes pensamentos'],
  [53, 57, 'Anel refletor'],
  [58, 60, 'Cinto do campeão'],
  [61, 67, 'Colar guardião'],
  [68, 72, 'Estatueta animista'],
  [73, 77, 'Anel da liberdade'],
  [78, 82, 'Tapete voador'],
  [83, 87, 'Braceletes de ouro'],
  [88, 89, 'Espelho da oposição'],
  [90, 94, 'Robe do arquimago'],
  [95, 96, 'Orbe das tempestades'],
  [97, 98, 'Anel da regeneração'],
  [99, 100, 'Espelho do aprisionamento'],
])

// ─── Resolver ───────────────────────────────────────────────────────
function rowsForTier(tier: LootMagicTier): readonly AccessoryRow[] {
  if (tier === 'menor') return ACCESSORIES_MENOR_ROWS
  if (tier === 'medio') return ACCESSORIES_MEDIO_ROWS
  return ACCESSORIES_MAIOR_ROWS
}

/**
 * Pick a specific magic accessory name by tier via d% roll on
 * Tabela 8-13/14/15. Assumes caller already decided this is an
 * "acessório" outcome (via 1d6 kind roll, p330).
 */
export function rollMagicAccessory(
  rng: Rng,
  tier: LootMagicTier,
): { tier: LootMagicTier; name: string } {
  const roll = rollPercentile(rng)
  const rows = rowsForTier(tier)
  const hit = rows.find((r) => roll >= r[0] && roll <= r[1])
  if (!hit) {
    throw new Error(
      `rollMagicAccessory: no row matches d% ${roll} for tier ${tier}`,
    )
  }
  return { tier, name: hit[2] }
}
