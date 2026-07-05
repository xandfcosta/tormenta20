/**
 * Perícia Enganação (CAR) — 6 usos canônicos.
 *
 * PDF Cap 2 Perícias — Enganação (livro p118-119). Header verbatim:
 * "ENGANAÇÃO — CAR" (sem flag treinada, sem penalidade de armadura).
 * Intro verbatim: "Você manipula pessoas com blefes e trapaças."
 *
 * Usos (todos p118-119):
 *  - Disfarce — teste oposto vs Percepção; 10 min + estojo
 *  - Falsificação — teste oposto vs Percepção; -10 complexo/assinatura
 *  - Fintar — teste oposto vs Reflexos; ação padrão; alcance curto
 *  - Insinuação — CD 20; observadores fazem Intuição oposta
 *  - Intriga — CD 20 (30 se improvável); 1+ dia; rastreável via Investigação
 *  - Mentir — teste oposto vs Intuição; -10 implausível
 *
 * Cross-ref:
 *  - `intuicao-skill-usages.ts` — Perceber Mentira delega para Mentir
 *    (opposed roll) e reflete `IMPLAUSIBLE_LIE_ENGANACAO_PENALTY = -10`.
 *  - Falsificação combina com Ofício (`oficio-craft-rules.ts`) para
 *    falsificar objetos físicos além de documentos.
 */

import { makeUsageByKind } from './skill-usage-resolver'

// ─── Types ────────────────────────────────────────────────────────────
export type EnganacaoUsageKind =
  | 'disfarce'
  | 'falsificacao'
  | 'fintar'
  | 'insinuacao'
  | 'intriga'
  | 'mentir'

/** Ação/duração exigida por um uso. */
export type EnganacaoAction =
  | 'padrao'
  | '10-minutos'
  | 'dia-ou-mais'
  | 'nao-especificado'

/** Perícia com que o teste é oposto (se opposed roll). */
export type EnganacaoOpposedBy = 'percepcao' | 'reflexos' | 'intuicao' | null

type UsageCommon = {
  id: EnganacaoUsageKind
  name: string
  action: EnganacaoAction
  opposedBy: EnganacaoOpposedBy
  effect: string
  bookPage: 118 | 119
}

export type EnganacaoDisfarce = UsageCommon & {
  kind: 'disfarce'
  action: '10-minutos'
  opposedBy: 'percepcao'
  requiresDisguiseKit: true
}

export type EnganacaoFalsificacao = UsageCommon & {
  kind: 'falsificacao'
  action: 'nao-especificado'
  opposedBy: 'percepcao'
  combinesWithOficio: true
}

export type EnganacaoFintar = UsageCommon & {
  kind: 'fintar'
  action: 'padrao'
  opposedBy: 'reflexos'
  range: 'curto'
  /** Alvo fica desprevenido contra próximo ataque até fim do próximo turno. */
  makesTargetDesprevenido: true
}

export type EnganacaoInsinuacao = UsageCommon & {
  kind: 'insinuacao'
  dc: 20
  opposedBy: null
  /** Falha por 5+ deturpa mensagem. */
  garbleByMargin: 5
  /** Observadores externos fazem Intuição oposta ao teste para compreender. */
  observersRollIntuicaoOpposed: true
}

export type EnganacaoIntriga = UsageCommon & {
  kind: 'intriga'
  action: 'dia-ou-mais'
  opposedBy: null
  /** CD 20 normal, 30 se muito improvável. */
  dc: 20
  dcImprovavel: 30
  /** Falha por 5+ expõe fofoqueiro ao alvo. */
  exposeByMargin: 5
  /** CD de Investigação para rastrear = teste de Enganação usado. */
  investigationCdEqualsEnganacaoRoll: true
}

export type EnganacaoMentir = UsageCommon & {
  kind: 'mentir'
  action: 'nao-especificado'
  opposedBy: 'intuicao'
  /** Verbatim: "-10 em seu teste" se mentira muito implausível. */
  implausiblePenalty: -10
}

export type EnganacaoUsage =
  | EnganacaoDisfarce
  | EnganacaoFalsificacao
  | EnganacaoFintar
  | EnganacaoInsinuacao
  | EnganacaoIntriga
  | EnganacaoMentir

// ─── Constantes ──────────────────────────────────────────────────────
// Disfarce (p118 verbatim)
export const DISFARCE_COMPLEX_PENALTY = -5
export const DISFARCE_WITHOUT_KIT_PENALTY = -5
export const DISFARCE_KNOWS_SPECIFIC_PERSON_BONUS_PERCEPCAO = 10
export const DISFARCE_DURATION_MINUTES = 10

// Falsificação (p119 verbatim)
export const FALSIFICACAO_COMPLEX_PENALTY = -10

// Insinuação (p119 verbatim)
export const INSINUACAO_CD = 20
export const INSINUACAO_GARBLE_MARGIN = 5

// Intriga (p119 verbatim)
export const INTRIGA_CD = 20
export const INTRIGA_IMPROVAVEL_CD = 30
export const INTRIGA_EXPOSE_MARGIN = 5

// Mentir (p119 verbatim)
export const MENTIR_IMPLAUSIBLE_PENALTY = -10

// Perícia flags (Tabela 2-1)
export const ENGANACAO_TRAINED_ONLY = false
export const ENGANACAO_ARMOR_PENALTY = false

// ─── Catálogo ─────────────────────────────────────────────────────────
export const ENGANACAO_USAGES: readonly EnganacaoUsage[] = Object.freeze([
  {
    id: 'disfarce',
    kind: 'disfarce',
    name: 'Disfarce',
    action: '10-minutos',
    opposedBy: 'percepcao',
    requiresDisguiseKit: true,
    effect:
      'Muda aparência sua ou de outrem; oposto por Percepção; complexos e falta de estojo penalizam.',
    bookPage: 118,
  },
  {
    id: 'falsificacao',
    kind: 'falsificacao',
    name: 'Falsificação',
    action: 'nao-especificado',
    opposedBy: 'percepcao',
    combinesWithOficio: true,
    effect:
      'Falsifica documento (ou objeto físico com Ofício); -10 se complexo/assinatura/carimbo.',
    bookPage: 119,
  },
  {
    id: 'fintar',
    kind: 'fintar',
    name: 'Fintar',
    action: 'padrao',
    opposedBy: 'reflexos',
    range: 'curto',
    makesTargetDesprevenido: true,
    effect:
      'Ação padrão vs Reflexos em alcance curto; alvo desprevenido contra próximo ataque até fim do próximo turno.',
    bookPage: 119,
  },
  {
    id: 'insinuacao',
    kind: 'insinuacao',
    name: 'Insinuação',
    action: 'nao-especificado',
    opposedBy: null,
    dc: 20,
    garbleByMargin: 5,
    observersRollIntuicaoOpposed: true,
    effect:
      'CD 20; mensagem cifrada para receptor; falha por 5+ deturpa; observadores fazem Intuição oposta para compreender.',
    bookPage: 119,
  },
  {
    id: 'intriga',
    kind: 'intriga',
    name: 'Intriga',
    action: 'dia-ou-mais',
    opposedBy: null,
    dc: 20,
    dcImprovavel: 30,
    exposeByMargin: 5,
    investigationCdEqualsEnganacaoRoll: true,
    effect:
      'Espalha fofoca; CD 20 (30 se improvável); falha por 5+ expõe; Investigação rastreia com CD igual ao teste.',
    bookPage: 119,
  },
  {
    id: 'mentir',
    kind: 'mentir',
    name: 'Mentir',
    action: 'nao-especificado',
    opposedBy: 'intuicao',
    implausiblePenalty: -10,
    effect:
      'Teste oposto vs Intuição; -10 se mentira muito implausível.',
    bookPage: 119,
  },
])

export const enganacaoUsageByKind = makeUsageByKind<EnganacaoUsageKind, EnganacaoUsage>(
  ENGANACAO_USAGES,
  'enganacaoUsageByKind',
)

// ─── Helpers — Disfarce ──────────────────────────────────────────────
/**
 * Penalidade agregada no teste de Disfarce:
 *  - `complex`: -5 se disfarce complexo (ex. raça diferente).
 *  - `withoutKit`: -5 se sem estojo de disfarces.
 */
export function disfarcePenalty(opts: {
  complex?: boolean
  withoutKit?: boolean
}): number {
  let p = 0
  if (opts.complex) p += DISFARCE_COMPLEX_PENALTY
  if (opts.withoutKit) p += DISFARCE_WITHOUT_KIT_PENALTY
  return p
}

/**
 * Bônus no teste de Percepção do observador quando conhece a pessoa
 * específica imitada. +10 verbatim.
 */
export function disfarcePercepcaoBonus(knowsSpecificPerson: boolean): number {
  return knowsSpecificPerson ? DISFARCE_KNOWS_SPECIFIC_PERSON_BONUS_PERCEPCAO : 0
}

// ─── Helpers — Falsificação ──────────────────────────────────────────
/** -10 se documento complexo ou com assinatura/carimbo específico. */
export function falsificacaoPenalty(complexOrSignedOrStamped: boolean): number {
  return complexOrSignedOrStamped ? FALSIFICACAO_COMPLEX_PENALTY : 0
}

// ─── Helpers — Insinuação ────────────────────────────────────────────
export type InsinuacaoOutcome = 'received' | 'garbled' | 'failed'

/**
 * Resolve Insinuação vs CD 20:
 *  - Passa → receptor entende.
 *  - Falha por < 5 → mensagem falha, receptor não entende.
 *  - Falha por ≥ 5 → receptor entende algo diferente.
 */
export function insinuacaoOutcome(checkResult: number): InsinuacaoOutcome {
  const delta = checkResult - INSINUACAO_CD
  if (delta >= 0) return 'received'
  if (Math.abs(delta) >= INSINUACAO_GARBLE_MARGIN) return 'garbled'
  return 'failed'
}

// ─── Helpers — Intriga ───────────────────────────────────────────────
/** CD 30 se muito improvável, 20 caso contrário. */
export function intrigaCd(improvavel: boolean): number {
  return improvavel ? INTRIGA_IMPROVAVEL_CD : INTRIGA_CD
}

export type IntrigaOutcome = 'spread' | 'failed' | 'exposed'

/**
 * Resolve Intriga contra sua CD:
 *  - Passa → fofoca se espalha (rastreável via Investigação com CD =
 *    seu teste).
 *  - Falha por < 5 → não se espalha.
 *  - Falha por ≥ 5 → alvo descobre que é você fofocando.
 */
export function intrigaOutcome(
  checkResult: number,
  cd: number,
): IntrigaOutcome {
  const delta = checkResult - cd
  if (delta >= 0) return 'spread'
  if (Math.abs(delta) >= INTRIGA_EXPOSE_MARGIN) return 'exposed'
  return 'failed'
}

/** CD para investigar a fonte da fofoca (verbatim: "igual ao seu teste"). */
export function intrigaInvestigationCd(enganacaoRoll: number): number {
  return enganacaoRoll
}

// ─── Helpers — Mentir ────────────────────────────────────────────────
/** -10 aplicado ao teste do mentiroso se mentira implausível. */
export function mentirPenalty(implausible: boolean): number {
  return implausible ? MENTIR_IMPLAUSIBLE_PENALTY : 0
}
