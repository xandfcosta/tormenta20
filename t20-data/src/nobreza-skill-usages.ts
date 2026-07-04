/**
 * Perícia Nobreza (INT, treinada, sem penalidade de armadura) — 2 usos.
 *
 * PDF Cap 2 Perícias — Nobreza (livro p121). Header verbatim:
 * "NOBREZA — INT · TREINADA".
 * Intro verbatim: "Você recebeu a educação de um nobre."
 *
 * Usos:
 *  - Etiqueta (p121) — CD 15; portar-se em ambientes aristocráticos
 *    (bailes, audiências).
 *  - Informação (p121) — perguntas sobre leis, tradições, linhagens,
 *    heráldica; simples sem teste; complexa CD 20; mistério/enigma
 *    CD 30 (mesmo padrão de Misticismo p121 e Religião p122).
 *
 * Nota Tabela 2-1 p115: APENAS TREINADA; NÃO sofre penalidade de
 * armadura. Nenhuma cross-ref explícita a outra perícia.
 */

import type { InformacaoDifficulty } from './misticismo-skill-usages'

// ─── Types ────────────────────────────────────────────────────────────
export type NobrezaUsageKind = 'etiqueta' | 'informacao'

type UsageCommon = {
  id: NobrezaUsageKind
  name: string
  effect: string
  bookPage: 121
}

export type NobrezaEtiqueta = UsageCommon & {
  kind: 'etiqueta'
  dc: 15
}

export type NobrezaInformacao = UsageCommon & {
  kind: 'informacao'
  cdComplexa: 20
  cdMisterio: 30
  simplesRequiresNoTest: true
}

export type NobrezaUsage = NobrezaEtiqueta | NobrezaInformacao

// ─── Constantes ──────────────────────────────────────────────────────
// Etiqueta (p121 verbatim)
export const ETIQUETA_CD = 15

// Informação (p121 verbatim)
export const NOBREZA_INFORMACAO_CD_COMPLEXA = 20
export const NOBREZA_INFORMACAO_CD_MISTERIO = 30

// Flags Tabela 2-1 p115
export const NOBREZA_TRAINED_ONLY = true
export const NOBREZA_ARMOR_PENALTY = false

// ─── Catálogo ─────────────────────────────────────────────────────────
export const NOBREZA_USAGES: readonly NobrezaUsage[] = Object.freeze([
  {
    id: 'etiqueta',
    kind: 'etiqueta',
    name: 'Etiqueta',
    dc: 15,
    effect:
      'CD 15; portar-se em ambientes aristocráticos (bailes, audiências).',
    bookPage: 121,
  },
  {
    id: 'informacao',
    kind: 'informacao',
    name: 'Informação',
    cdComplexa: 20,
    cdMisterio: 30,
    simplesRequiresNoTest: true,
    effect:
      'Leis/tradições/linhagens/heráldica: simples sem teste; complexa CD 20; mistério/enigma CD 30.',
    bookPage: 121,
  },
])

const usagesByKind = new Map<NobrezaUsageKind, NobrezaUsage>(
  NOBREZA_USAGES.map((u) => [u.kind, u]),
)

export function nobrezaUsageByKind(kind: NobrezaUsageKind): NobrezaUsage {
  const usage = usagesByKind.get(kind)
  if (!usage) {
    throw new Error(`nobrezaUsageByKind: unknown kind ${kind}`)
  }
  return usage
}

// ─── Helpers — Etiqueta ──────────────────────────────────────────────
/** CD sempre 15 (verbatim). */
export function etiquetaCd(): number {
  return ETIQUETA_CD
}

// ─── Helpers — Informação ───────────────────────────────────────────
/**
 * CD por dificuldade:
 *  - simples → null (não exige teste)
 *  - complexa → 20
 *  - mistério/enigma → 30
 */
export function nobrezaInformacaoCd(
  difficulty: InformacaoDifficulty,
): number | null {
  switch (difficulty) {
    case 'simples':
      return null
    case 'complexa':
      return NOBREZA_INFORMACAO_CD_COMPLEXA
    case 'misterio-ou-enigma':
      return NOBREZA_INFORMACAO_CD_MISTERIO
  }
}
