/**
 * Magic weapon random tables — PDF Cap 8 Recompensas, p336-337.
 *
 * Tabela 8-8 (Armas Mágicas) — d% pra encanto, ou 91-100 rota pra
 * arma específica (Tabela 8-9).
 * Tabela 8-9 (Armas Específicas) — d% pick de arma nomeada.
 *
 * Footnote 8-8: `*` = conta como dois encantos; pra itens menores,
 * role novamente.
 *
 * Deterministic com seeded Rng.
 */
import { rollPercentile, type Rng } from './loot-rng'
import type { LootMagicTier } from './loot'

/** [lo, hi, name, countsAsTwo?] — countsAsTwo = footnote *. */
type WeaponEnchantRow = readonly [
  lo: number,
  hi: number,
  name: string,
  countsAsTwo?: true,
]

type SpecificWeaponRow = readonly [lo: number, hi: number, name: string]

// ─── Tabela 8-8: Armas Mágicas (p336) ───────────────────────────────
export const WEAPON_ENCHANT_ROWS: readonly WeaponEnchantRow[] = Object.freeze([
  [1, 5, 'Ameaçadora'],
  [6, 10, 'Anticriatura'],
  [11, 12, 'Arremesso'],
  [13, 14, 'Assassina'],
  [15, 16, 'Caçadora'],
  [17, 21, 'Congelante'],
  [22, 23, 'Conjuradora'],
  [24, 28, 'Corrosiva'],
  [29, 30, 'Dançarina'],
  [31, 34, 'Defensora'],
  [35, 36, 'Destruidora'],
  [37, 38, 'Dilacerante'],
  [39, 40, 'Drenante'],
  [41, 45, 'Elétrica'],
  [46, 46, 'Energética', true],
  [47, 48, 'Excruciante'],
  [49, 53, 'Flamejante'],
  [54, 63, 'Formidável'],
  [64, 64, 'Lancinante', true],
  [65, 72, 'Magnífica', true],
  [73, 74, 'Piedosa'],
  [75, 76, 'Profana'],
  [77, 78, 'Sagrada'],
  [79, 80, 'Sanguinária'],
  [81, 82, 'Trovejante'],
  [83, 84, 'Tumular'],
  [85, 88, 'Veloz'],
  [89, 90, 'Venenosa'],
  // 91-100: rota pra Tabela 8-9 (arma específica) — sentinel
  [91, 100, 'Arma específica'],
])

// ─── Tabela 8-9: Armas Específicas (p337) ───────────────────────────
export const SPECIFIC_WEAPON_ROWS: readonly SpecificWeaponRow[] = Object.freeze([
  [1, 5, 'Azagaia dos relâmpagos'],
  [6, 15, 'Espada baronial'],
  [16, 25, 'Lâmina da luz'],
  [26, 30, 'Lança animalesca'],
  [31, 35, 'Maça do terror'],
  [36, 40, 'Florete fugaz'],
  [41, 45, 'Cajado da destruição'],
  [46, 50, 'Cajado da vida'],
  [51, 55, 'Machado silvestre'],
  [56, 60, 'Martelo de Doherimm'],
  [61, 67, 'Arco do poder'],
  [68, 72, 'Língua do deserto'],
  [73, 77, 'Besta explosiva'],
  [78, 82, 'Punhal sszzaazita'],
  [83, 87, 'Espada sortuda'],
  [88, 92, 'Avalanche'],
  [93, 95, 'Cajado do poder'],
  [96, 100, 'Vingadora sagrada'],
])

// ─── Types ──────────────────────────────────────────────────────────
export type MagicWeaponResult =
  | { kind: 'encanto'; name: string; countsAsTwo: boolean; tier: LootMagicTier }
  | { kind: 'specific'; name: string; tier: LootMagicTier }

// ─── Resolver ───────────────────────────────────────────────────────
function rollSpecificWeapon(rng: Rng): string {
  const roll = rollPercentile(rng)
  const hit = SPECIFIC_WEAPON_ROWS.find((r) => roll >= r[0] && roll <= r[1])
  if (!hit) {
    throw new Error(`rollSpecificWeapon: no row matches d% ${roll}`)
  }
  return hit[2]
}

function rollEnchantRow(rng: Rng): WeaponEnchantRow {
  const roll = rollPercentile(rng)
  const hit = WEAPON_ENCHANT_ROWS.find((r) => roll >= r[0] && roll <= r[1])
  if (!hit) {
    throw new Error(`rollMagicWeapon: no row matches d% ${roll}`)
  }
  return hit
}

/**
 * Roll magic weapon result via Tabela 8-8 (+ 8-9 se 91-100).
 * Tier controls footnote: countsAsTwo encantos rerollam pra menor
 * (regra p336). Safety cap 16 tentativas.
 */
export function rollMagicWeapon(
  rng: Rng,
  tier: LootMagicTier,
): MagicWeaponResult {
  let safety = 0
  while (safety < 16) {
    safety++
    const row = rollEnchantRow(rng)
    const countsAsTwo = row[3] === true

    // Footnote *: menor + countsAsTwo → reroll
    if (countsAsTwo && tier === 'menor') continue

    if (row[2] === 'Arma específica') {
      return { kind: 'specific', name: rollSpecificWeapon(rng), tier }
    }
    return { kind: 'encanto', name: row[2], countsAsTwo, tier }
  }
  throw new Error(
    `rollMagicWeapon: exceeded reroll safety (tier=${tier}); table shape invalid`,
  )
}
