/**
 * Crafting rules — regras de fabricação de itens.
 *
 * PDF refs:
 *  - Ofício Fabricar (p121)
 *  - Escrever Pergaminho + Preparar Poção (p131, Cap 2 magic pool)
 *  - Preço pergaminho/poção + Fabricação de itens mágicos (p333-334)
 *  - Tabela 8-7 encantos (p334)
 *
 * Constantes complementares de tier CD/PM já em `reward-tables.ts`:
 *  - MAGIC_ITEM_CRAFT_CD, MAGIC_ITEM_CRAFT_PM_SACRIFICE
 */

// ─── Ofício Fabricar — regras base p121 ─────────────────────────────
/** CD de itens simples (equipamento aventura, armas simples, armaduras leves). */
export const OFICIO_CD_SIMPLE = 15

/** CD de itens complexos (armas marciais/exóticas/fogo, armaduras pesadas). */
export const OFICIO_CD_COMPLEX = 20

/** Fração do preço final gasta em matéria-prima. */
export const MATERIAL_COST_FRACTION = 1 / 3

/** Custo em matéria-prima para consertar item (1/10 preço original). */
export const REPAIR_MATERIAL_FRACTION = 1 / 10

/** Penalidade no teste ao fabricar 2 consumíveis no mesmo tempo. */
export const CONSUMABLE_DOUBLE_PENALTY = -5

/** Tempo de fabricação em dias por categoria. */
export type CraftTimeCategory =
  | 'consumable'
  | 'non-consumable-common'
  | 'non-consumable-superior'

export const CRAFT_TIME_DAYS: Readonly<Record<CraftTimeCategory, number>> =
  Object.freeze({
    consumable: 1,
    'non-consumable-common': 7,
    'non-consumable-superior': 30,
  })

/**
 * Custo em matéria-prima para fabricar item de preço `finalPrice`.
 * Regra p121: 1/3 do preço. Retorna arredondado pra cima.
 */
export function materialCost(finalPrice: number): number {
  if (finalPrice < 0) {
    throw new Error(
      `materialCost: finalPrice must be ≥ 0, got ${finalPrice}`,
    )
  }
  return Math.ceil(finalPrice * MATERIAL_COST_FRACTION)
}

/**
 * Custo em matéria-prima para consertar item (1/10 preço original).
 * Consumido por tentativa (1 hora de trabalho).
 */
export function repairCost(originalPrice: number): number {
  if (originalPrice < 0) {
    throw new Error(
      `repairCost: originalPrice must be ≥ 0, got ${originalPrice}`,
    )
  }
  return Math.ceil(originalPrice * REPAIR_MATERIAL_FRACTION)
}

// ─── Pergaminho + Poção — regras p333 ───────────────────────────────
/**
 * Preço de pergaminho ou poção baseado no custo em PM da magia base.
 *  price = T$ 30 × PM² (mínimo T$ 1)
 */
export function scrollPotionPrice(pmCost: number): number {
  if (pmCost < 0) {
    throw new Error(
      `scrollPotionPrice: pmCost must be ≥ 0, got ${pmCost}`,
    )
  }
  const price = 30 * pmCost * pmCost
  return Math.max(1, price)
}

/**
 * CD do teste de Ofício para fabricar pergaminho ou poção.
 *  CD = 20 + PM da magia
 */
export function scrollPotionCd(pmCost: number): number {
  if (pmCost < 0) {
    throw new Error(
      `scrollPotionCd: pmCost must be ≥ 0, got ${pmCost}`,
    )
  }
  return 20 + pmCost
}

// ─── Tabela 8-7 encantos mágicos — p334 ─────────────────────────────
export type EncantoCount = 1 | 2 | 3

export type EncantoTierInfo = {
  priceBonus: number
  cdBonus: number
}

/**
 * Tabela 8-7 (p334): preço + CD adicional por número de encantos.
 * Encantos adicionais empilham preço mas nunca dobram bônus mecânicos.
 */
export const ENCANTO_TIERS: Readonly<Record<EncantoCount, EncantoTierInfo>> =
  Object.freeze({
    1: { priceBonus: 18000, cdBonus: 10 },
    2: { priceBonus: 36000, cdBonus: 15 },
    3: { priceBonus: 72000, cdBonus: 20 },
  })

/** Preço adicional por número de encantos aplicados ao item. */
export function encantoPriceBonus(count: EncantoCount): number {
  return ENCANTO_TIERS[count].priceBonus
}

/** CD adicional na Ofício para fabricar item com N encantos. */
export function encantoCdBonus(count: EncantoCount): number {
  return ENCANTO_TIERS[count].cdBonus
}

/**
 * Mapeia número de encantos ao tier resultante (p334):
 *  1 encanto = menor, 2 = médio, 3 = maior.
 */
export function tierFromEncantoCount(
  count: EncantoCount,
): 'menor' | 'medio' | 'maior' {
  if (count === 1) return 'menor'
  if (count === 2) return 'medio'
  return 'maior'
}

// ─── Engenhoca (poder Inventor Engenhoqueiro) ───────────────────────
/**
 * Preço de fabricação de engenhoca baseada em magia.
 *  price = T$ 100 × PM da magia
 */
export function engenhocaPrice(pmCost: number): number {
  if (pmCost < 0) {
    throw new Error(
      `engenhocaPrice: pmCost must be ≥ 0, got ${pmCost}`,
    )
  }
  return 100 * pmCost
}

/**
 * CD do teste de Ofício (engenhoqueiro) para fabricar engenhoca.
 *  CD = 20 + PM da magia
 */
export function engenhocaCd(pmCost: number): number {
  if (pmCost < 0) {
    throw new Error(
      `engenhocaCd: pmCost must be ≥ 0, got ${pmCost}`,
    )
  }
  return 20 + pmCost
}

/** Tempo de fabricação de engenhoca — sempre 1 semana. */
export const ENGENHOCA_CRAFT_TIME_DAYS = 7
