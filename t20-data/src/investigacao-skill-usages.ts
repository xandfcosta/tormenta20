/**
 * Perícia Investigação (INT) — 2 usos canônicos.
 *
 * PDF Cap 2 Perícias — Investigação (livro p120). Header verbatim:
 * "INVESTIGAÇÃO — INT" (sem flag treinada, sem penalidade de armadura).
 * Intro verbatim: "Você sabe encontrar pistas e informações."
 *
 * Usos (p120):
 *  - Interrogar — 1 hora + T$ 3d6; CD 20 restrita / 30 confidencial;
 *    informação geral não exige teste
 *  - Procurar — ação completa até 1 dia; CD 15 bagunça / 20 escondido /
 *    30 muito escondido; armadilhas usam CD própria; rastros achável
 *    mas seguir requer Sobrevivência
 *
 * Cross-ref:
 *  - `enganacao-skill-usages.ts` Intriga: rastrear a fonte usa Investigação
 *    com CD igual ao teste de Enganação (regra vive no módulo Enganação).
 *  - `sobrevivencia-skill-usages.ts` Rastrear: seguir rastros achados
 *    aqui exige Sobrevivência (verbatim redirecionamento).
 */

import { makeUsageByKind } from './skill-usage-resolver'

// ─── Types ────────────────────────────────────────────────────────────
export type InvestigacaoUsageKind = 'interrogar' | 'procurar'

/** Categoria de informação para Interrogar. */
export type InfoCategory = 'geral' | 'restrita' | 'confidencial'

/** Nível de ocultação para Procurar (fora armadilhas/rastros). */
export type SearchTier = 'bagunca' | 'escondido' | 'muito-escondido'

type UsageCommon = {
  id: InvestigacaoUsageKind
  name: string
  effect: string
  bookPage: 120
}

export type InvestigacaoInterrogar = UsageCommon & {
  kind: 'interrogar'
  action: 'uma-hora'
  /** CDs verbatim: geral (sem teste), restrita 20, confidencial 30. */
  dcRestrita: 20
  dcConfidencial: 30
  /** T$ 3d6 custo verbatim (bebidas, subornos etc.). */
  costDice: '3d6'
  costCurrency: 'tibar'
}

export type InvestigacaoProcurar = UsageCommon & {
  kind: 'procurar'
  /** Duração verbatim: "desde uma ação completa até um dia". */
  actionMin: 'completa'
  actionMax: 'um-dia'
  /** CDs por tier de ocultação. */
  dcBaguncaOrPlainSight: 15
  dcEscondido: 20
  dcMuitoEscondido: 30
  /** Armadilhas usam CD específica da armadilha (não vem daqui). */
  trapsUseTheirOwnCd: true
  /** Rastros: achável aqui, seguir requer Sobrevivência. */
  followingTracksRequiresSobrevivencia: true
}

export type InvestigacaoUsage = InvestigacaoInterrogar | InvestigacaoProcurar

// ─── Constantes ──────────────────────────────────────────────────────
// Interrogar (p120 verbatim)
export const INTERROGAR_CD_RESTRITA = 20
export const INTERROGAR_CD_CONFIDENCIAL = 30
export const INTERROGAR_COST_DICE = '3d6' as const
export const INTERROGAR_DURATION_HOURS = 1

// Procurar (p120 verbatim)
export const PROCURAR_CD_BAGUNCA = 15
export const PROCURAR_CD_ESCONDIDO = 20
export const PROCURAR_CD_MUITO_ESCONDIDO = 30

// Flags Tabela 2-1
export const INVESTIGACAO_TRAINED_ONLY = false
export const INVESTIGACAO_ARMOR_PENALTY = false

// ─── Catálogo ─────────────────────────────────────────────────────────
export const INVESTIGACAO_USAGES: readonly InvestigacaoUsage[] = Object.freeze([
  {
    id: 'interrogar',
    kind: 'interrogar',
    name: 'Interrogar',
    action: 'uma-hora',
    dcRestrita: 20,
    dcConfidencial: 30,
    costDice: '3d6',
    costCurrency: 'tibar',
    effect:
      'Descobre informação: geral (sem teste), restrita CD 20, confidencial CD 30; 1 hora + T$ 3d6.',
    bookPage: 120,
  },
  {
    id: 'procurar',
    kind: 'procurar',
    name: 'Procurar',
    actionMin: 'completa',
    actionMax: 'um-dia',
    dcBaguncaOrPlainSight: 15,
    dcEscondido: 20,
    dcMuitoEscondido: 30,
    trapsUseTheirOwnCd: true,
    followingTracksRequiresSobrevivencia: true,
    effect:
      'Examina local; CDs 15/20/30 por ocultação; armadilhas usam CD própria; rastros achável, seguir usa Sobrevivência.',
    bookPage: 120,
  },
])

export const investigacaoUsageByKind = makeUsageByKind<InvestigacaoUsageKind, InvestigacaoUsage>(
  INVESTIGACAO_USAGES,
  'investigacaoUsageByKind',
)

// ─── Helpers — Interrogar ────────────────────────────────────────────
/**
 * CD por categoria de informação.
 *  - geral: sem teste (retorna 0 — chamador não deve rolar).
 *  - restrita: CD 20.
 *  - confidencial: CD 30.
 */
export function interrogarCd(category: InfoCategory): number {
  if (category === 'geral') return 0
  if (category === 'restrita') return INTERROGAR_CD_RESTRITA
  return INTERROGAR_CD_CONFIDENCIAL
}

/** Verbatim: "gasta uma hora e T$ 3d6". */
export function interrogarRequiresRoll(category: InfoCategory): boolean {
  return category !== 'geral'
}

// ─── Helpers — Procurar ──────────────────────────────────────────────
/** CD por nível de ocultação (fora armadilhas/rastros). */
export function procurarCd(tier: SearchTier): number {
  if (tier === 'bagunca') return PROCURAR_CD_BAGUNCA
  if (tier === 'escondido') return PROCURAR_CD_ESCONDIDO
  return PROCURAR_CD_MUITO_ESCONDIDO
}

/**
 * Verbatim: "para segui-los deve usar Sobrevivência". Achar rastro passa
 * por Investigação, seguir passa por Sobrevivência (`rastrearCd`).
 */
export function followingTracksSkill(): 'sobrevivencia' {
  return 'sobrevivencia'
}
