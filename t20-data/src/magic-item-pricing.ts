/**
 * Magic item pricing composition — combina base mundane + melhorias + encantos.
 *
 * PDF refs:
 *  - Cap 8 p333-334 (Itens Mágicos / Itens Encantados / Itens Específicos)
 *  - Cap 3 p164 (Item Superior / melhorias)
 *  - Tabela 8-7 (encantos, já em `crafting-rules.ts`)
 *  - Preço pergaminho/poção `scrollPotionPrice` já em `crafting-rules.ts`
 *
 * Três branches distintos:
 *  - **encantado**: arma/armadura mundana + até 4 melhorias + até 3 encantos
 *  - **específico**: preço fixo em Tabela 8-9/8-11/8-13-15 (lookup em catalog)
 *  - **consumível**: T$ 30 × PM² (scrollPotionPrice em crafting-rules.ts)
 *
 * Máximos verbatim p334: "quatro melhorias e três encantos (o máximo possível)".
 * Regra "Bônus por encantos não se acumulam" (p334) é responsabilidade do caller.
 *
 * Exemplo worked p334:
 *  - Espada longa T$ 15 + 1 encanto  = T$ 18.015 (CD 30)
 *  - Espada longa T$ 15 + 4 melhorias (T$ 18.000) + 3 encantos = T$ 90.015
 */

import {
  OFICIO_CD_COMPLEX,
  encantoCdBonus,
  encantoPriceBonus,
  scrollPotionCd,
  scrollPotionPrice,
} from './crafting-rules'

/** Máximo de melhorias que um item superior pode receber (p164). */
export const MAX_MELHORIAS_PER_ITEM = 4

/** Máximo de encantos por item mágico permanente (p334). */
export const MAX_ENCANTOS_PER_ITEM = 3

/**
 * Obra-prima: item com 4 melhorias exigido por Tabela 5-3 (CD 40 quase-impossível).
 * Também referenciado como capstone do Inventor.
 */
export const OBRA_PRIMA_MELHORIAS = 4

/** Círculos de magia disponíveis para poção/pergaminho fabricados por PJ. */
export type PocaoCirculo = 1 | 2 | 3 | 4 | 5

/**
 * Custo em PM por círculo (padrão de conversor Tormenta 20).
 * 1º = 1 PM, 2º = 3 PM, 3º = 6 PM, 4º = 10 PM, 5º = 15 PM.
 */
export const POCAO_PM_BY_CIRCULO: Readonly<Record<PocaoCirculo, number>> =
  Object.freeze({
    1: 1,
    2: 3,
    3: 6,
    4: 10,
    5: 15,
  })

/** Contagem de encantos aplicados (0 = puramente superior, sem encantos). */
export type EncantoCountOptional = 0 | 1 | 2 | 3

export type EncantadoPriceInput = {
  /** Preço da versão mundana do item (ex.: espada longa T$ 15). */
  mundaneBasePrice: number
  /** Preço de cada melhoria aplicada. Vazio = sem melhorias. */
  melhoriaPrices: readonly number[]
  /** Número de encantos aplicados (0-3). */
  encantoCount: EncantoCountOptional
}

/**
 * Compõe preço de item encantado: base mundane + soma de melhorias + encantos.
 * Encantos entram como bloco fixo por Tabela 8-7, não por-encanto.
 */
export function computeEncantadoPrice(input: EncantadoPriceInput): number {
  const { mundaneBasePrice, melhoriaPrices, encantoCount } = input
  if (mundaneBasePrice < 0) {
    throw new Error(
      `computeEncantadoPrice: mundaneBasePrice must be ≥ 0, got ${mundaneBasePrice}`,
    )
  }
  if (melhoriaPrices.length > MAX_MELHORIAS_PER_ITEM) {
    throw new Error(
      `computeEncantadoPrice: max ${MAX_MELHORIAS_PER_ITEM} melhorias, got ${melhoriaPrices.length}`,
    )
  }
  for (const p of melhoriaPrices) {
    if (p < 0) {
      throw new Error(
        `computeEncantadoPrice: melhoria price must be ≥ 0, got ${p}`,
      )
    }
  }
  const melhoriaTotal = melhoriaPrices.reduce((a, b) => a + b, 0)
  const encantoTotal =
    encantoCount === 0 ? 0 : encantoPriceBonus(encantoCount)
  return mundaneBasePrice + melhoriaTotal + encantoTotal
}

/**
 * CD de Ofício para fabricar item encantado: base complexa (20) + Tabela 8-7.
 * Se `encantoCount = 0`, retorna apenas CD base — item superior puro.
 * NOTA: itens específicos (não-encantados) usam `MAGIC_ITEM_CRAFT_CD` em
 * `reward-tables.ts` (30/40/50 por tier).
 */
export function computeEncantadoCd(encantoCount: EncantoCountOptional): number {
  if (encantoCount === 0) return OFICIO_CD_COMPLEX
  return OFICIO_CD_COMPLEX + encantoCdBonus(encantoCount)
}

/** Preço de poção/pergaminho por círculo — atalho para scrollPotionPrice(PM). */
export function pocaoPriceByCirculo(circulo: PocaoCirculo): number {
  return scrollPotionPrice(POCAO_PM_BY_CIRCULO[circulo])
}

/** CD de Ofício para fabricar poção/pergaminho por círculo. */
export function pocaoCdByCirculo(circulo: PocaoCirculo): number {
  return scrollPotionCd(POCAO_PM_BY_CIRCULO[circulo])
}
