/**
 * Perícia Intimidação (CAR) — 2 usos canônicos.
 *
 * PDF Cap 2 Perícias — Intimidação (livro p120). Header verbatim:
 * "INTIMIDAÇÃO — CAR" (sem flag treinada, sem penalidade de armadura).
 * Intro verbatim: "Você pode assustar ou coagir outras pessoas.
 * Usos de Intimidação são efeitos de medo."
 *
 * Importante: são "efeitos de medo" (fear), NÃO "efeitos mentais".
 * Imunidades a medo bloqueiam; imunidade a mental só bloqueia se a
 * criatura também tiver imunidade a medo (regra geral do bestiário).
 *
 * Usos (p120):
 *  - Assustar — ação padrão vs Vontade; alcance curto; sucesso abalado;
 *    margem 10+ = apavorado 1 rodada + abalado cena
 *  - Coagir — 1 min+ vs Vontade; adjacente; alvo Int ≥ -3;
 *    ordem perigosa/contra natureza dá +5 no Vontade (ou passa auto);
 *    deixa alvo hostil contra você
 *
 * Cross-ref:
 *  - `conditions.ts` — condições `abalado` e `apavorado`.
 *  - `diplomacia-skill-usages.ts` — Mudar Atitude (Coagir empurra alvo
 *    para atitude "hostil" no eixo `ATITUDE_ORDER`).
 */

import { makeUsageByKind } from './skill-usage-resolver'

// ─── Types ────────────────────────────────────────────────────────────
export type IntimidacaoUsageKind = 'assustar' | 'coagir'

type UsageCommon = {
  id: IntimidacaoUsageKind
  name: string
  opposedBy: 'vontade'
  effect: string
  isFearEffect: true
  bookPage: 120
}

export type IntimidacaoAssustar = UsageCommon & {
  kind: 'assustar'
  action: 'padrao'
  range: 'curto'
  /** Verbatim: "pelo resto da cena". */
  successCondition: 'abalado'
  successConditionDuration: 'resto-da-cena'
  /** Margem 10+ ativa apavorado 1 rodada + abalado cena. */
  bigSuccessMargin: 10
  bigSuccessBriefCondition: 'apavorado'
  bigSuccessBriefDurationRounds: 1
  bigSuccessThenCondition: 'abalado'
  bigSuccessThenDuration: 'resto-da-cena'
}

export type IntimidacaoCoagir = UsageCommon & {
  kind: 'coagir'
  action: 'um-minuto-ou-mais'
  positioning: 'adjacente'
  /** Verbatim: "criatura inteligente (Int -3 ou maior)". */
  targetMinIntelligence: -3
  /** Verbatim: ordem perigosa/contra natureza → +5 no Vontade do alvo. */
  dangerousOrderTargetVontadeBonus: 5
  /** Verbatim: "ou passa automaticamente". */
  dangerousOrderCanAutoPass: true
  /** Verbatim: "deixa o alvo hostil contra você". */
  leavesTargetHostile: true
}

export type IntimidacaoUsage = IntimidacaoAssustar | IntimidacaoCoagir

// ─── Constantes ──────────────────────────────────────────────────────
// Assustar (p120 verbatim)
export const ASSUSTAR_BIG_SUCCESS_MARGIN = 10
export const ASSUSTAR_APAVORADO_ROUNDS = 1

// Coagir (p120 verbatim)
export const COAGIR_TARGET_MIN_INT = -3
export const COAGIR_DANGEROUS_ORDER_VONTADE_BONUS = 5

// Flags Tabela 2-1
export const INTIMIDACAO_TRAINED_ONLY = false
export const INTIMIDACAO_ARMOR_PENALTY = false

// ─── Catálogo ─────────────────────────────────────────────────────────
export const INTIMIDACAO_USAGES: readonly IntimidacaoUsage[] = Object.freeze([
  {
    id: 'assustar',
    kind: 'assustar',
    name: 'Assustar',
    action: 'padrao',
    range: 'curto',
    opposedBy: 'vontade',
    isFearEffect: true,
    successCondition: 'abalado',
    successConditionDuration: 'resto-da-cena',
    bigSuccessMargin: 10,
    bigSuccessBriefCondition: 'apavorado',
    bigSuccessBriefDurationRounds: 1,
    bigSuccessThenCondition: 'abalado',
    bigSuccessThenDuration: 'resto-da-cena',
    effect:
      'Ação padrão vs Vontade em alcance curto; sucesso deixa alvo abalado pela cena; margem 10+ = apavorado 1 rodada, então abalado.',
    bookPage: 120,
  },
  {
    id: 'coagir',
    kind: 'coagir',
    name: 'Coagir',
    action: 'um-minuto-ou-mais',
    positioning: 'adjacente',
    opposedBy: 'vontade',
    isFearEffect: true,
    targetMinIntelligence: -3,
    dangerousOrderTargetVontadeBonus: 5,
    dangerousOrderCanAutoPass: true,
    leavesTargetHostile: true,
    effect:
      'Vs Vontade em 1+ min, alvo adjacente Int ≥ -3; obedece ordem; ordem perigosa/contra natureza dá +5 (ou passa auto); alvo fica hostil.',
    bookPage: 120,
  },
])

export const intimidacaoUsageByKind = makeUsageByKind<IntimidacaoUsageKind, IntimidacaoUsage>(
  INTIMIDACAO_USAGES,
  'intimidacaoUsageByKind',
)

// ─── Helpers — Assustar ──────────────────────────────────────────────
export type AssustarOutcome = 'no-effect' | 'abalado' | 'apavorado-then-abalado'

/**
 * Resolve Assustar vs Vontade:
 *  - Vitória por 10+ → apavorado 1 rodada + abalado resto da cena.
 *  - Vitória por 0-9 → abalado resto da cena.
 *  - Derrota qualquer → sem efeito.
 */
export function assustarOutcome(
  intimidacaoTotal: number,
  vontadeTotal: number,
): AssustarOutcome {
  const delta = intimidacaoTotal - vontadeTotal
  if (delta >= ASSUSTAR_BIG_SUCCESS_MARGIN) return 'apavorado-then-abalado'
  if (delta >= 0) return 'abalado'
  return 'no-effect'
}

// ─── Helpers — Coagir ────────────────────────────────────────────────
/**
 * Verbatim: "Int -3 ou maior". Retorna se alvo é elegível para Coagir.
 */
export function coagirTargetIsEligible(targetIntelligence: number): boolean {
  return targetIntelligence >= COAGIR_TARGET_MIN_INT
}

/**
 * Modificador no Vontade do alvo:
 *  - Ordem normal: 0.
 *  - Ordem perigosa/contra natureza: +5.
 * Retorna também flag se o alvo pode escolher passar automaticamente.
 */
export function coagirTargetVontadeModifier(dangerousOrContraryOrder: boolean): {
  bonus: number
  canAutoPass: boolean
} {
  if (dangerousOrContraryOrder) {
    return { bonus: COAGIR_DANGEROUS_ORDER_VONTADE_BONUS, canAutoPass: true }
  }
  return { bonus: 0, canAutoPass: false }
}

export type CoagirOutcome = 'refused' | 'obeys-and-hostile'

/**
 * Resolve Coagir. `dangerousOrContraryOrder` aplica +5 (ou passa auto —
 * assumido não escolhido; use `coagirTargetVontadeModifier` para expor
 * o `canAutoPass` ao chamador).
 * Sucesso deixa alvo hostil independente do resultado do subteste.
 */
export function coagirOutcome(
  intimidacaoTotal: number,
  vontadeTotal: number,
  opts: { dangerousOrContraryOrder?: boolean } = {},
): CoagirOutcome {
  const { bonus } = coagirTargetVontadeModifier(
    opts.dangerousOrContraryOrder === true,
  )
  const adjustedVontade = vontadeTotal + bonus
  return intimidacaoTotal - adjustedVontade >= 0
    ? 'obeys-and-hostile'
    : 'refused'
}
