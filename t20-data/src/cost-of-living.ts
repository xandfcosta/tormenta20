/**
 * Custo de Vida — PDF Cap 6 p277 (Variante). Sistema opcional para
 * simplificar despesas entre aventuras: o jogador paga um valor mensal
 * fixo (moradia + comida + roupa + transporte) e recebe uma condição
 * de descanso padrão. Substitui pagar cada estadia/refeição isolada.
 *
 * A condição associada acopla direto com `rest.ts` — o motor de
 * recuperação PV/PM lê `RestCondition` e calcula pv/pm por dia.
 *
 * Fica de fora: o texto de "acumular 4 benefícios de treinamento = sobe
 * de nível" (Treinamento p276) — essa é regra de treinamento, não custo
 * de vida.
 */
import type { RestCondition } from './rest'

export const COST_OF_LIVING_TIERS = [
  'pobre',
  'medio',
  'rico',
  'luxuoso',
] as const

export type CostOfLivingTier = (typeof COST_OF_LIVING_TIERS)[number]

export type CostOfLivingRow = {
  tier: CostOfLivingTier
  label: string
  /** Custo mensal em T$ (Tibar). */
  monthlyTS: number
  /** Condição de descanso padrão associada — casa com `rest.ts`. */
  restCondition: RestCondition
  description: string
}

export const COST_OF_LIVING_TABLE: readonly CostOfLivingRow[] = Object.freeze([
  {
    tier: 'pobre',
    label: 'Pobre',
    monthlyTS: 10,
    restCondition: 'ruim',
    description:
      'Dorme na rua, em celeiros ou nas piores hospedarias. Come pão velho e veste trapos.',
  },
  {
    tier: 'medio',
    label: 'Médio',
    monthlyTS: 50,
    restCondition: 'normal',
    description:
      'Dorme em estalagens e come em tavernas. Estilo caro para pessoas comuns.',
  },
  {
    tier: 'rico',
    label: 'Rico',
    monthlyTS: 100,
    restCondition: 'confortavel',
    description:
      'Fica em quartos privativos, alimenta-se bem e veste-se com roupas feitas por alfaiates.',
  },
  {
    tier: 'luxuoso',
    label: 'Luxuoso',
    monthlyTS: 200,
    restCondition: 'luxuosa',
    description:
      'Dorme nas melhores estalagens ou em castelos de nobres. Banquetes e carruagens.',
  },
])

export function costOfLivingRow(tier: CostOfLivingTier): CostOfLivingRow {
  const row = COST_OF_LIVING_TABLE.find((r) => r.tier === tier)
  if (!row) throw new Error(`Unknown CostOfLivingTier: ${tier}`)
  return row
}

/** Menor tier cujo `monthlyTS` cabe no orçamento mensal informado. */
export function affordableCostOfLiving(
  monthlyBudgetTS: number,
): CostOfLivingTier | null {
  if (monthlyBudgetTS < 0) {
    throw new Error(`monthlyBudgetTS must be >= 0, got ${monthlyBudgetTS}`)
  }
  let best: CostOfLivingTier | null = null
  for (const row of COST_OF_LIVING_TABLE) {
    if (row.monthlyTS <= monthlyBudgetTS) best = row.tier
    else break
  }
  return best
}

/** Condição de descanso derivada do tier de custo. Atalho comum. */
export function restConditionForCost(tier: CostOfLivingTier): RestCondition {
  return costOfLivingRow(tier).restCondition
}
