/**
 * Perícia Religião (SAB, treinada, sem penalidade de armadura) — 4 usos.
 *
 * PDF Cap 2 Perícias — Religião (livro p122). Header verbatim:
 * "RELIGIÃO — SAB · TREINADA".
 * Intro verbatim: "Você possui conhecimento sobre os deuses e as
 * religiões de Arton."
 *
 * Usos:
 *  - Identificar Criatura (p122) — CD 15 + ND; delega mecânica à
 *    perícia [[misticismo-skill-usages]] (para criatura de origem
 *    divina como anjos, demônios, alguns mortos-vivos, construtos).
 *  - Identificar Item Mágico (p122) — item mágico de origem divina;
 *    CDs delegadas a Misticismo (20/25/30 por categoria; 1 hora ou
 *    -10 completa).
 *  - Informação (p122) — perguntas sobre deuses/profecias/planos;
 *    simples sem teste; complexa CD 20; mistério/enigma CD 30.
 *  - Rito (p122) — CD 20; cerimônia religiosa (batizado, casamento,
 *    funeral, penitência). Penitência exige sacrifício T$ 100 × nível
 *    do devoto OU missão sagrada.
 *
 * Nota Tabela 2-1 p115: APENAS TREINADA; NÃO sofre penalidade de
 * armadura. Cross-ref: [[misticismo-skill-usages]] (identificação
 * mágica); usos Identificar delegam a mecânica de tests para lá.
 */

import type { InformacaoDifficulty, ItemMagicoCategory } from './misticismo-skill-usages'
import {
  identificarItemMagicoCd as misticismoIdentificarItemMagicoCd,
  identificarItemMagicoRushedPenalty as misticismoIdentificarItemMagicoRushedPenalty,
} from './misticismo-skill-usages'
import { makeUsageByKind } from './skill-usage-resolver'

// ─── Types ────────────────────────────────────────────────────────────
export type ReligiaoUsageKind =
  | 'identificar-criatura'
  | 'identificar-item-magico'
  | 'informacao'
  | 'rito'

/** Tipo de cerimônia realizada em Rito (verbatim p122). */
export type RitoCerimonia =
  | 'batizado'
  | 'casamento'
  | 'funeral'
  | 'penitencia'
  | 'outro'

type UsageCommon = {
  id: ReligiaoUsageKind
  name: string
  effect: string
  bookPage: 122
}

export type ReligiaoIdentificarCriatura = UsageCommon & {
  kind: 'identificar-criatura'
  cdBase: 15
  cdIncludesNd: true
  /** Aplica-se apenas a criaturas de origem divina. */
  divineOriginOnly: true
  delegatesTo: 'misticismo'
}

export type ReligiaoIdentificarItemMagico = UsageCommon & {
  kind: 'identificar-item-magico'
  /** Aplica-se apenas a itens mágicos de origem divina. */
  divineOriginOnly: true
  delegatesTo: 'misticismo'
}

export type ReligiaoInformacao = UsageCommon & {
  kind: 'informacao'
  cdComplexa: 20
  cdMisterio: 30
  /** Perguntas simples não exigem teste. */
  simplesRequiresNoTest: true
}

export type ReligiaoRito = UsageCommon & {
  kind: 'rito'
  dc: 20
  /** Penitência exige T$ 100 × nível OU missão sagrada. */
  penitenciaSacrificeTibarPerLevel: 100
  penitenciaMissionAlternative: true
}

export type ReligiaoUsage =
  | ReligiaoIdentificarCriatura
  | ReligiaoIdentificarItemMagico
  | ReligiaoInformacao
  | ReligiaoRito

// ─── Constantes ──────────────────────────────────────────────────────
// Identificar Criatura (p122 verbatim; mecânica em Misticismo p121)
export const RELIGIAO_IDENTIFICAR_CRIATURA_CD_BASE = 15

// Informação (p122 verbatim)
export const RELIGIAO_INFORMACAO_CD_COMPLEXA = 20
export const RELIGIAO_INFORMACAO_CD_MISTERIO = 30

// Rito (p122 verbatim)
export const RITO_CD = 20
export const PENITENCIA_SACRIFICE_TIBAR_PER_LEVEL = 100

// Flags Tabela 2-1 p115
export const RELIGIAO_TRAINED_ONLY = true
export const RELIGIAO_ARMOR_PENALTY = false

// ─── Catálogo ─────────────────────────────────────────────────────────
export const RELIGIAO_USAGES: readonly ReligiaoUsage[] = Object.freeze([
  {
    id: 'identificar-criatura',
    kind: 'identificar-criatura',
    name: 'Identificar Criatura',
    cdBase: 15,
    cdIncludesNd: true,
    divineOriginOnly: true,
    delegatesTo: 'misticismo',
    effect:
      'CD 15 + ND para criaturas de origem divina (anjos, demônios, alguns mortos-vivos/construtos); mecânica delega a Misticismo.',
    bookPage: 122,
  },
  {
    id: 'identificar-item-magico',
    kind: 'identificar-item-magico',
    name: 'Identificar Item Mágico',
    divineOriginOnly: true,
    delegatesTo: 'misticismo',
    effect:
      'Identifica item mágico de origem divina; CDs/duração delegadas a Misticismo (20/25/30 por categoria; 1h ou completa @ -10).',
    bookPage: 122,
  },
  {
    id: 'informacao',
    kind: 'informacao',
    name: 'Informação',
    cdComplexa: 20,
    cdMisterio: 30,
    simplesRequiresNoTest: true,
    effect:
      'Deuses/profecias/planos: simples sem teste; complexa CD 20; mistério/enigma CD 30.',
    bookPage: 122,
  },
  {
    id: 'rito',
    kind: 'rito',
    name: 'Rito',
    dc: 20,
    penitenciaSacrificeTibarPerLevel: 100,
    penitenciaMissionAlternative: true,
    effect:
      'CD 20; cerimônia religiosa (batizado, casamento, funeral, penitência). Penitência exige T$ 100 × nível do devoto OU missão sagrada.',
    bookPage: 122,
  },
])

export const religiaoUsageByKind = makeUsageByKind<ReligiaoUsageKind, ReligiaoUsage>(
  RELIGIAO_USAGES,
  'religiaoUsageByKind',
)

// ─── Helpers — Identificar Criatura ─────────────────────────────────
/** CD = 15 + ND da criatura divina. */
export function religiaoIdentificarCriaturaCd(nd: number): number {
  return RELIGIAO_IDENTIFICAR_CRIATURA_CD_BASE + nd
}

// ─── Helpers — Identificar Item Mágico (delegado a Misticismo) ──────
/** Delega a `misticismo-skill-usages` — retorna CD 20/25/30 por categoria. */
export function religiaoIdentificarItemMagicoCd(
  category: ItemMagicoCategory,
): number {
  return misticismoIdentificarItemMagicoCd(category)
}

/** Delega a `misticismo-skill-usages` — -10 opcional para reduzir a ação completa. */
export function religiaoIdentificarItemMagicoRushedPenalty(
  rushed: boolean,
): number {
  return misticismoIdentificarItemMagicoRushedPenalty(rushed)
}

// ─── Helpers — Informação ───────────────────────────────────────────
/**
 * CD por dificuldade:
 *  - simples → null (não exige teste)
 *  - complexa → 20
 *  - mistério/enigma → 30
 */
export function religiaoInformacaoCd(
  difficulty: InformacaoDifficulty,
): number | null {
  switch (difficulty) {
    case 'simples':
      return null
    case 'complexa':
      return RELIGIAO_INFORMACAO_CD_COMPLEXA
    case 'misterio-ou-enigma':
      return RELIGIAO_INFORMACAO_CD_MISTERIO
  }
}

// ─── Helpers — Rito ─────────────────────────────────────────────────
/** CD 20 para qualquer cerimônia. */
export function ritoCd(): number {
  return RITO_CD
}

/** Sacrifício em T$ para penitência = 100 × nível do devoto. */
export function penitenciaSacrificeTibar(devotoLevel: number): number {
  if (devotoLevel < 1) {
    throw new Error(
      `penitenciaSacrificeTibar: devotoLevel must be ≥ 1, got ${devotoLevel}`,
    )
  }
  return devotoLevel * PENITENCIA_SACRIFICE_TIBAR_PER_LEVEL
}
