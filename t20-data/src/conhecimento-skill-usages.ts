/**
 * Perícia Conhecimento (INT, treinada, sem penalidade de armadura) — 2 usos.
 *
 * PDF Cap 2 Perícias — Conhecimento (livro p117). Header verbatim:
 * "CONHECIMENTO — INT · TREINADA".
 * Intro verbatim: "Você estudou diversos campos do saber, como
 * aritmética, astronomia, dialética, geografia, história..."
 *
 * Usos:
 *  - Idiomas (p117) — CD 20 (idiomas desconhecidos) ou CD 30 (exóticos
 *    ou muito antigos); falha por 5+ = conclusão falsa.
 *  - Informação (p117) — perguntas gerais; simples sem teste; complexa
 *    CD 20; mistério/enigma CD 30 (mesmo padrão de Misticismo/Religião/
 *    Nobreza).
 *
 * Nota Tabela 2-1 p115: APENAS TREINADA; NÃO sofre penalidade de
 * armadura. Não delega para Misticismo/Nobreza/Religião/Guerra — cada
 * uma cobre seu próprio domínio especializado, e Conhecimento cobre
 * saberes gerais (aritmética, astronomia, dialética, geografia,
 * história).
 */

import type { InformacaoDifficulty } from './misticismo-skill-usages'

// ─── Types ────────────────────────────────────────────────────────────
export type ConhecimentoUsageKind = 'idiomas' | 'informacao'

/** Categoria do idioma para Idiomas (p117). */
export type IdiomaKind = 'padrao' | 'exotico-ou-antigo'

type UsageCommon = {
  id: ConhecimentoUsageKind
  name: string
  effect: string
  bookPage: 117
}

export type ConhecimentoIdiomas = UsageCommon & {
  kind: 'idiomas'
  cdPadrao: 20
  cdExoticoOuAntigo: 30
  /** Falha por 5+ = conclusão falsa. */
  wrongConclusionMargin: 5
}

export type ConhecimentoInformacao = UsageCommon & {
  kind: 'informacao'
  cdComplexa: 20
  cdMisterio: 30
  simplesRequiresNoTest: true
}

export type ConhecimentoUsage = ConhecimentoIdiomas | ConhecimentoInformacao

// ─── Constantes ──────────────────────────────────────────────────────
// Idiomas (p117 verbatim)
export const IDIOMAS_CD_PADRAO = 20
export const IDIOMAS_CD_EXOTICO_OU_ANTIGO = 30
export const IDIOMAS_WRONG_CONCLUSION_MARGIN = 5

// Informação (p117 verbatim)
export const CONHECIMENTO_INFORMACAO_CD_COMPLEXA = 20
export const CONHECIMENTO_INFORMACAO_CD_MISTERIO = 30

// Flags Tabela 2-1 p115
export const CONHECIMENTO_TRAINED_ONLY = true
export const CONHECIMENTO_ARMOR_PENALTY = false

// ─── Catálogo ─────────────────────────────────────────────────────────
export const CONHECIMENTO_USAGES: readonly ConhecimentoUsage[] = Object.freeze([
  {
    id: 'idiomas',
    kind: 'idiomas',
    name: 'Idiomas',
    cdPadrao: 20,
    cdExoticoOuAntigo: 30,
    wrongConclusionMargin: 5,
    effect:
      'Entende idiomas desconhecidos; CD 20 padrão, CD 30 exótico/antigo; falha por 5+ = conclusão falsa.',
    bookPage: 117,
  },
  {
    id: 'informacao',
    kind: 'informacao',
    name: 'Informação',
    cdComplexa: 20,
    cdMisterio: 30,
    simplesRequiresNoTest: true,
    effect:
      'Perguntas gerais: simples sem teste; complexa CD 20; mistério/enigma CD 30.',
    bookPage: 117,
  },
])

const usagesByKind = new Map<ConhecimentoUsageKind, ConhecimentoUsage>(
  CONHECIMENTO_USAGES.map((u) => [u.kind, u]),
)

export function conhecimentoUsageByKind(
  kind: ConhecimentoUsageKind,
): ConhecimentoUsage {
  const usage = usagesByKind.get(kind)
  if (!usage) {
    throw new Error(`conhecimentoUsageByKind: unknown kind ${kind}`)
  }
  return usage
}

// ─── Helpers — Idiomas ──────────────────────────────────────────────
/** CD por categoria de idioma. */
export function idiomasCd(kind: IdiomaKind): number {
  return kind === 'padrao' ? IDIOMAS_CD_PADRAO : IDIOMAS_CD_EXOTICO_OU_ANTIGO
}

export type IdiomasOutcome = 'success' | 'failed' | 'wrong-conclusion'

/**
 * Resolve Idiomas:
 *  - roll ≥ CD → success.
 *  - falha por < 5 → failed.
 *  - falha por ≥ 5 → wrong-conclusion (tira conclusão falsa).
 */
export function idiomasOutcome(rollResult: number, cd: number): IdiomasOutcome {
  const delta = rollResult - cd
  if (delta >= 0) return 'success'
  if (Math.abs(delta) >= IDIOMAS_WRONG_CONCLUSION_MARGIN) return 'wrong-conclusion'
  return 'failed'
}

// ─── Helpers — Informação ───────────────────────────────────────────
/**
 * CD por dificuldade:
 *  - simples → null (não exige teste)
 *  - complexa → 20
 *  - mistério/enigma → 30
 */
export function conhecimentoInformacaoCd(
  difficulty: InformacaoDifficulty,
): number | null {
  switch (difficulty) {
    case 'simples':
      return null
    case 'complexa':
      return CONHECIMENTO_INFORMACAO_CD_COMPLEXA
    case 'misterio-ou-enigma':
      return CONHECIMENTO_INFORMACAO_CD_MISTERIO
  }
}
