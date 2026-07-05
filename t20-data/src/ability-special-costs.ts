/**
 * Custos Especiais de habilidade.
 *
 * PDF Cap 5 Jogando p224 — "USANDO HABILIDADES" > "CUSTOS ESPECIAIS".
 *
 * O livro define exatamente três tipos de custos especiais:
 *  1. Componente Material — ingredientes na mão, consumidos mesmo em falha.
 *  2. Penalidade de PM — reduz PM máximo enquanto habilidade está ativa;
 *     recupera ao fim da duração.
 *  3. Sacrifício de PM — reduz PM máximo permanentemente.
 *
 * Timing (p224): custo é pago na ativação, ANTES de conhecer o resultado
 * do teste; se a habilidade falha, o custo é gasto do mesmo jeito.
 *
 * NÃO cobertos por esta seção: Sacrifício de PV, Corrupção, Descarregar
 * (esses vivem em outros módulos ou não existem no livro base).
 */

// ─── Types ────────────────────────────────────────────────────────────
export type SpecialCostKind =
  | 'componente-material'
  | 'penalidade-pm'
  | 'sacrificio-pm'

type CostCommon = {
  kind: SpecialCostKind
  /** Descrição em texto do custo. */
  description: string
  /** True se o custo é pago mesmo se a habilidade falhar. */
  consumedOnFailure: true
  bookPage: 224
}

export type ComponenteMaterial = CostCommon & {
  kind: 'componente-material'
  /** Descrição do ingrediente (ex.: 'pó de esmeralda', 'sangue de virgem'). */
  ingredient: string
  /** Valor em T$ (tibar) do ingrediente, se especificado pela habilidade. */
  tibarValue?: number
  /** Ingrediente deve estar na mão para ativar. */
  mustBeInHand: true
}

export type PenalidadePm = CostCommon & {
  kind: 'penalidade-pm'
  /** PM máximo reduzido enquanto habilidade está ativa. */
  amount: number
  /** Reversível no fim da duração da habilidade. */
  temporary: true
}

export type SacrificioPm = CostCommon & {
  kind: 'sacrificio-pm'
  /** PM máximo permanentemente reduzido. */
  amount: number
  /** Irreversível — não recupera com descanso. */
  permanent: true
}

export type SpecialCost = ComponenteMaterial | PenalidadePm | SacrificioPm

// ─── Constantes ──────────────────────────────────────────────────────
/** Custo é pago mesmo se a habilidade falhar (p224). */
export const COST_PAID_ON_ACTIVATION = true

/** Componente material precisa estar na mão para ativar. */
export const COMPONENTE_MATERIAL_MUST_BE_IN_HAND = true

// ─── Factories ──────────────────────────────────────────────────────
/**
 * Constrói um Componente Material.
 * `tibarValue` é opcional — habilidades variam.
 */
export function componenteMaterial(
  ingredient: string,
  tibarValue?: number,
): ComponenteMaterial {
  if (tibarValue !== undefined && tibarValue < 0) {
    throw new Error(
      `componenteMaterial: tibarValue must be ≥ 0, got ${tibarValue}`,
    )
  }
  return {
    kind: 'componente-material',
    ingredient,
    tibarValue,
    mustBeInHand: true,
    consumedOnFailure: true,
    description: tibarValue !== undefined
      ? `Componente Material: ${ingredient} (T$ ${tibarValue}).`
      : `Componente Material: ${ingredient}.`,
    bookPage: 224,
  }
}

/** Constrói uma Penalidade de PM temporária. */
export function penalidadePm(amount: number): PenalidadePm {
  if (amount <= 0) {
    throw new Error(`penalidadePm: amount must be > 0, got ${amount}`)
  }
  return {
    kind: 'penalidade-pm',
    amount,
    temporary: true,
    consumedOnFailure: true,
    description: `Penalidade de PM: -${amount} PM máximo enquanto ativa.`,
    bookPage: 224,
  }
}

/** Constrói um Sacrifício de PM permanente. */
export function sacrificioPm(amount: number): SacrificioPm {
  if (amount <= 0) {
    throw new Error(`sacrificioPm: amount must be > 0, got ${amount}`)
  }
  return {
    kind: 'sacrificio-pm',
    amount,
    permanent: true,
    consumedOnFailure: true,
    description: `Sacrifício de PM: -${amount} PM máximo permanente.`,
    bookPage: 224,
  }
}

// ─── Helpers — aplicação ─────────────────────────────────────────────
/**
 * Reduz o PM máximo pela Penalidade de PM (temporária, reversível).
 * Retorna novo PM máximo enquanto o efeito estiver ativo.
 */
export function applyPenalidadePm(
  currentMaxPm: number,
  penalty: PenalidadePm,
): number {
  if (currentMaxPm < 0) {
    throw new Error(
      `applyPenalidadePm: currentMaxPm must be ≥ 0, got ${currentMaxPm}`,
    )
  }
  return Math.max(0, currentMaxPm - penalty.amount)
}

/**
 * Restaura o PM máximo ao término da habilidade (fim da Penalidade de PM).
 */
export function removePenalidadePm(
  reducedMaxPm: number,
  penalty: PenalidadePm,
): number {
  if (reducedMaxPm < 0) {
    throw new Error(
      `removePenalidadePm: reducedMaxPm must be ≥ 0, got ${reducedMaxPm}`,
    )
  }
  return reducedMaxPm + penalty.amount
}

/**
 * Reduz permanentemente o PM máximo (Sacrifício de PM).
 * Livro não fixa piso; a habilidade individual pode restringir.
 */
export function applySacrificioPm(
  currentMaxPm: number,
  sacrifice: SacrificioPm,
): number {
  if (currentMaxPm < 0) {
    throw new Error(
      `applySacrificioPm: currentMaxPm must be ≥ 0, got ${currentMaxPm}`,
    )
  }
  return Math.max(0, currentMaxPm - sacrifice.amount)
}

/** True se um custo especial deste tipo é pago mesmo em caso de falha. */
export function isPaidOnFailure(cost: SpecialCost): boolean {
  return cost.consumedOnFailure
}

/** Resolve narrativa de custo especial pelo `kind`. */
export function describeSpecialCost(cost: SpecialCost): string {
  return cost.description
}
