/**
 * Cavaleiro — poderes relacionados a parceiros (PDF p54-55).
 *
 * Catálogo de 5 poderes: Autoridade Feudal, Escudeiro, Pajem, Título
 * e Caminho da Montaria. Encoda level gates + tempos de reinvocação +
 * exceções ao limite de parceiros.
 *
 * Exceções ao limite de parceiros (p54, verbatim):
 *  - Escudeiro: "não conta em seu limite de parceiros"
 *  - Pajem:     "não conta em seu limite de parceiros"
 *
 * Demais poderes (Autoridade Feudal, Título, Caminho Montaria) contam
 * contra o limite padrão de `parceiro-rules.ts:parceiroLimit`.
 *
 * Cross-refs:
 *  - `parceiro-rules.ts:cavaleiroMontariaTier(level)` — veterano <11, mestre 11+
 *  - `parceiro-rules.ts:parceiroLimit(pcLevel)` — limite geral
 */

import type { ParceiroTier } from './parceiro-rules'

export type CavaleiroParceiroPowerId =
  | 'autoridade-feudal'
  | 'escudeiro'
  | 'pajem'
  | 'titulo'
  | 'caminho-montaria'

export type RetrainTimeUnit = 'semana' | 'mes'

export type RetrainTime = {
  unit: RetrainTimeUnit
  count: number
}

export type CavaleiroParceiroPower = {
  id: CavaleiroParceiroPowerId
  name: string
  bookPage: 54 | 55
  /** Nível mínimo de cavaleiro para adquirir. */
  minLevel: number
  /** True se conta contra `parceiroLimit`. */
  countsAgainstLimit: boolean
  /** Patamar do parceiro concedido, ou null se o poder não gera parceiro típico (Escudeiro/Pajem). */
  grantedTier: ParceiroTier | null
  /** Tempo de reinvocação/treinamento após morte. null = sem regra. */
  retrain: RetrainTime | null
  /** True se requer compra prévia (Caminho Montaria — comprar outra). */
  requiresPurchase: boolean
  /** Requisitos adicionais além do minLevel. */
  additionalPrereqs: readonly string[]
}

// ─── Level gates (p54-55) ────────────────────────────────────────────
export const AUTORIDADE_FEUDAL_MIN_LEVEL = 6
export const TITULO_MIN_LEVEL = 10
export const MONTARIA_UNLOCK_LEVEL = 5
export const MONTARIA_MESTRE_LEVEL = 11

// ─── Powers catalog ──────────────────────────────────────────────────
const RAW: readonly CavaleiroParceiroPower[] = [
  {
    id: 'autoridade-feudal',
    name: 'Autoridade Feudal',
    bookPage: 54,
    minLevel: AUTORIDADE_FEUDAL_MIN_LEVEL,
    countsAgainstLimit: true,
    grantedTier: 'iniciante',
    retrain: null,
    requiresPurchase: false,
    additionalPrereqs: [],
  },
  {
    id: 'escudeiro',
    name: 'Escudeiro',
    bookPage: 54,
    minLevel: 1,
    countsAgainstLimit: false,
    grantedTier: null,
    retrain: { unit: 'mes', count: 1 },
    requiresPurchase: false,
    additionalPrereqs: [],
  },
  {
    id: 'pajem',
    name: 'Pajem',
    bookPage: 54,
    minLevel: 1,
    countsAgainstLimit: false,
    grantedTier: null,
    retrain: { unit: 'semana', count: 1 },
    requiresPurchase: false,
    additionalPrereqs: [],
  },
  {
    id: 'titulo',
    name: 'Título',
    bookPage: 55,
    minLevel: TITULO_MIN_LEVEL,
    countsAgainstLimit: true,
    grantedTier: 'veterano',
    retrain: null,
    requiresPurchase: false,
    additionalPrereqs: [
      'Autoridade Feudal',
      'ter conquistado terras ou realizado serviço para um nobre-suserano',
    ],
  },
  {
    id: 'caminho-montaria',
    name: 'Caminho: Montaria',
    bookPage: 55,
    minLevel: MONTARIA_UNLOCK_LEVEL,
    countsAgainstLimit: true,
    grantedTier: 'veterano',
    retrain: { unit: 'semana', count: 1 },
    requiresPurchase: true,
    additionalPrereqs: [],
  },
]

export const CAVALEIRO_PARCEIRO_POWERS: readonly CavaleiroParceiroPower[] =
  Object.freeze(RAW)

export function cavaleiroParceiroPowerById(
  id: CavaleiroParceiroPowerId,
): CavaleiroParceiroPower | undefined {
  return CAVALEIRO_PARCEIRO_POWERS.find((p) => p.id === id)
}

/** Filtra poderes disponíveis ao nível informado. */
export function unlockedCavaleiroParceiroPowers(
  cavaleiroLevel: number,
): readonly CavaleiroParceiroPower[] {
  if (cavaleiroLevel < 1) {
    throw new Error(
      `unlockedCavaleiroParceiroPowers: cavaleiroLevel must be ≥ 1, got ${cavaleiroLevel}`,
    )
  }
  return CAVALEIRO_PARCEIRO_POWERS.filter((p) => p.minLevel <= cavaleiroLevel)
}

/** True se o poder concede parceiro que NÃO conta contra o limite p260. */
export function hasParceiroLimitException(
  id: CavaleiroParceiroPowerId,
): boolean {
  const power = cavaleiroParceiroPowerById(id)
  if (!power) return false
  return !power.countsAgainstLimit
}
