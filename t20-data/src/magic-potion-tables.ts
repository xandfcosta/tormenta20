/**
 * Magic potion random table — PDF Cap 8 Recompensas, p341 (Tabela 8-12).
 *
 * Cobertura d% [1, 100] contígua. Cada linha tem nome + preço T$.
 * Tiers derivados pelas bandas de preço explicitadas no texto p341:
 *  - Menor: rows 1-36 (T$ 30, círculos 1º/2º)
 *  - Médio: rows 37-80 (T$ 120-270, círculos 3º/4º)
 *  - Maior: rows 81-100 (T$ 750-3.000, círculo 5º)
 *
 * Deterministic com seeded Rng.
 */
import { rollPercentile, type Rng } from './loot-rng'
import type { LootMagicTier } from './loot'

type PotionRow = readonly [lo: number, hi: number, name: string, priceTs: number]

export const POTION_ROWS: readonly PotionRow[] = Object.freeze([
  [1, 1, 'Abençoar Alimentos (óleo)', 30],
  [2, 3, 'Área Escorregadia (granada)', 30],
  [4, 6, 'Arma Mágica (óleo)', 30],
  [7, 7, 'Compreensão', 30],
  [8, 15, 'Curar Ferimentos (2d8+2 PV)', 30],
  [16, 18, 'Disfarce Ilusório', 30],
  [19, 20, 'Escuridão (óleo)', 30],
  [21, 22, 'Luz (óleo)', 30],
  [23, 24, 'Névoa (granada)', 30],
  [25, 26, 'Primor Atlético', 30],
  [27, 28, 'Proteção Divina', 30],
  [29, 30, 'Resistência a Energia', 30],
  [31, 32, 'Sono', 30],
  [33, 33, 'Suporte Ambiental', 30],
  [34, 34, 'Tranca Arcana (óleo)', 30],
  [35, 35, 'Visão Mística', 30],
  [36, 36, 'Vitalidade Fantasma', 30],
  [37, 38, 'Escudo da Fé (aprimoramento para duração cena)', 120],
  [39, 40, 'Alterar Tamanho', 270],
  [41, 42, 'Aparência Perfeita', 270],
  [43, 43, 'Armamento da Natureza (óleo)', 270],
  [44, 49, 'Bola de Fogo (granada)', 270],
  [50, 51, 'Camuflagem Ilusória', 270],
  [52, 53, 'Concentração de Combate (aprimoramento para duração cena)', 270],
  [54, 62, 'Curar Ferimentos (4d8+4 PV)', 270],
  [63, 66, 'Físico Divino', 270],
  [67, 68, 'Mente Divina', 270],
  [69, 70, 'Metamorfose', 270],
  [71, 75, 'Purificação', 270],
  [76, 77, 'Velocidade', 270],
  [78, 79, 'Vestimenta da Fé (óleo)', 270],
  [80, 80, 'Voz Divina', 270],
  [81, 82, 'Arma Mágica (óleo; aprimoramento para bônus +3)', 750],
  [83, 88, 'Curar Ferimentos (7d8+7 PV)', 1080],
  [89, 89, 'Físico Divino (aprimoramento para três atributos)', 1080],
  [90, 92, 'Invisibilidade (aprimoramento para duração cena)', 1080],
  [93, 96, 'Bola de Fogo (granada; aprimoramento para 10d6 de dano)', 1470],
  [97, 100, 'Curar Ferimentos (11d8+11 PV)', 3000],
])

/** Derive tier from row hi index — bands per texto p341. */
export function tierForPotionRoll(roll: number): LootMagicTier {
  if (roll <= 36) return 'menor'
  if (roll <= 80) return 'medio'
  return 'maior'
}

export type PotionRoll = {
  name: string
  priceTs: number
  tier: LootMagicTier
  roll: number
}

/**
 * Roll one poção via Tabela 8-12. Tabela 8-1 rola count separado
 * (countFormula) — caller invoca este N vezes.
 */
export function rollMagicPotion(rng: Rng): PotionRoll {
  const roll = rollPercentile(rng)
  const hit = POTION_ROWS.find((r) => roll >= r[0] && roll <= r[1])
  if (!hit) {
    throw new Error(`rollMagicPotion: no row matches d% ${roll}`)
  }
  return { name: hit[2], priceTs: hit[3], tier: tierForPotionRoll(roll), roll }
}
