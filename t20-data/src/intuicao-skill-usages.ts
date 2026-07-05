/**
 * Perícia Intuição (SAB) — 2 usos canônicos.
 *
 * PDF Cap 2 Perícias — Intuição (livro p120).
 *
 * Header verbatim: "INTUIÇÃO — SAB". Sem flag de treinada no header, sem
 * penalidade de armadura. Um USO específico (Pressentimento) é
 * "Apenas Treinado"; o outro (Perceber Mentira) é aberto.
 *
 * Intro verbatim: "Esta perícia mede sua empatia e 'sexto sentido'."
 *
 * Cross-ref (Enganação p119):
 *  - Mentir → oposto por Intuição; mentiras implausíveis dão -10 no
 *    teste de Enganação (efetivo +10 relativo para Intuição).
 *  - Insinuação → CD fixa 20; outros fazem Intuição oposta ao teste de
 *    Enganação para entender mensagem cifrada.
 *
 * Perceber Mentira delega CD/regra para Enganação (verbatim: "veja a
 * perícia Enganação").
 */

import { makeUsageByKind } from './skill-usage-resolver'

// ─── Types ────────────────────────────────────────────────────────────
export type IntuicaoUsageKind = 'perceber-mentira' | 'pressentimento'

type UsageCommon = {
  id: IntuicaoUsageKind
  name: string
  trainedOnly: boolean
  effect: string
  bookPage: 120
}

export type IntuicaoPerceberMentira = UsageCommon & {
  kind: 'perceber-mentira'
  /** Teste oposto à Enganação (Mentir) do alvo. */
  contest: 'opposed-vs-enganacao-mentir'
  /** Verbatim Enganação p119: implausível → -10 no teste do mentiroso. */
  liarImplausiblePenalty: -10
  trainedOnly: false
}

export type IntuicaoPressentimento = UsageCommon & {
  kind: 'pressentimento'
  dc: 20
  trainedOnly: true
  /**
   * Verbatim: "apenas indica se há algo anormal, mas não revela a causa".
   */
  revealsCause: false
}

export type IntuicaoUsage = IntuicaoPerceberMentira | IntuicaoPressentimento

// ─── Constantes ──────────────────────────────────────────────────────
/** CD verbatim p120. */
export const PRESSENTIMENTO_CD = 20

/**
 * Penalidade aplicada no teste de Enganação do mentiroso quando a
 * mentira é muito implausível (verbatim Enganação p119).
 * Do ponto de vista da Intuição, isso é uma vantagem relativa.
 */
export const IMPLAUSIBLE_LIE_ENGANACAO_PENALTY = -10

/** Perícia é aberta (sem flag "Apenas Treinado" no header). Tabela 2-1. */
export const INTUICAO_TRAINED_ONLY = false

/** Sem penalidade de armadura (Tabela 2-1). */
export const INTUICAO_ARMOR_PENALTY = false

// ─── Catálogo ─────────────────────────────────────────────────────────
export const INTUICAO_USAGES: readonly IntuicaoUsage[] = Object.freeze([
  {
    id: 'perceber-mentira',
    kind: 'perceber-mentira',
    name: 'Perceber Mentira',
    trainedOnly: false,
    contest: 'opposed-vs-enganacao-mentir',
    liarImplausiblePenalty: -10,
    effect:
      'Descobre se alguém está mentindo (teste oposto vs Enganação-Mentir do alvo).',
    bookPage: 120,
  },
  {
    id: 'pressentimento',
    kind: 'pressentimento',
    name: 'Pressentimento',
    trainedOnly: true,
    dc: 20,
    revealsCause: false,
    effect:
      'Analisa pessoa/situação para detectar anomalia (índole, caráter, ambiente estranho); indica apenas presença da anomalia, não a causa.',
    bookPage: 120,
  },
])

export const intuicaoUsageByKind = makeUsageByKind<IntuicaoUsageKind, IntuicaoUsage>(
  INTUICAO_USAGES,
  'intuicaoUsageByKind',
)

// ─── Helpers ─────────────────────────────────────────────────────────
/** CD de Pressentimento — fixa 20 (verbatim). */
export function pressentimentoCd(): number {
  return PRESSENTIMENTO_CD
}

/**
 * Ajuste no teste de Enganação do mentiroso quando o teste de Perceber
 * Mentira o desafia. Verbatim Enganação p119: -10 se implausível.
 */
export function enganacaoAdjustmentForLie(implausible: boolean): number {
  return implausible ? IMPLAUSIBLE_LIE_ENGANACAO_PENALTY : 0
}

export type PerceberMentiraOutcome = 'detected-lie' | 'believes-lie' | 'tied'

/**
 * Resolve teste oposto Intuição-Perceber Mentira vs Enganação-Mentir.
 * Verbatim: teste oposto simples. Aplique `enganacaoAdjustmentForLie`
 * ao total do mentiroso ANTES de comparar, se aplicável.
 */
export function resolvePerceberMentira(
  intuicaoTotal: number,
  enganacaoTotal: number,
): PerceberMentiraOutcome {
  if (intuicaoTotal > enganacaoTotal) return 'detected-lie'
  if (intuicaoTotal < enganacaoTotal) return 'believes-lie'
  return 'tied'
}
