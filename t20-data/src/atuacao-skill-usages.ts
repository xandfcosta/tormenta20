/**
 * Perícia Atuação (CAR, treinada, sem penalidade de armadura) — 2 usos.
 *
 * PDF Cap 2 Perícias — Atuação (livro p116). Header verbatim:
 * "ATUAÇÃO — CAR · TREINADA".
 * Intro verbatim: "Você pode fazer apresentações artísticas, incluindo
 * música, dança e dramaturgia."
 *
 * Usos:
 *  - Apresentação (p116) — CD 20; 1 dia (ou noite); sucesso paga T$ 1d6
 *    + 1d6 por 5 pontos acima da CD; local inadequado = metade; local
 *    propício = padrão; local especialmente propício = dobro.
 *  - Impressionar (p116) — oposto Atuação vs Vontade do alvo; sucesso
 *    = +2 em testes de perícias Carisma-based contra ele no mesmo dia;
 *    falha = -2 e não pode tentar de novo no mesmo dia; plateia usa
 *    um único teste com o melhor valor. Duração: minutos (canto/dança)
 *    a horas (apresentação teatral).
 *
 * Nota Tabela 2-1 p115: APENAS TREINADA; NÃO sofre penalidade de
 * armadura. Cross-ref: efeito de Impressionar afeta perícias Carisma-
 * based (Diplomacia, Enganação, Intimidação, Jogatina) contra o alvo.
 */

import { makeUsageByKind } from './skill-usage-resolver'

// ─── Types ────────────────────────────────────────────────────────────
export type AtuacaoUsageKind = 'apresentacao' | 'impressionar'

/** Qualidade do local da apresentação (p116). */
export type LocationQuality =
  | 'inadequado'
  | 'propicio'
  | 'especialmente-propicio'

type UsageCommon = {
  id: AtuacaoUsageKind
  name: string
  effect: string
  bookPage: 116
}

export type AtuacaoApresentacao = UsageCommon & {
  kind: 'apresentacao'
  dc: 20
  duration: 'um-dia-ou-noite'
  baseTibarDice: '1d6'
  extraDicePerMargin: 5
}

export type AtuacaoImpressionar = UsageCommon & {
  kind: 'impressionar'
  opposedBy: 'vontade'
  successCharismaSkillBonus: 2
  failureCharismaSkillPenalty: -2
  /** Falha impede nova tentativa contra o mesmo alvo no mesmo dia. */
  failureLocksOutSameDay: true
  /** Múltiplos alvos: mestre faz um teste pela plateia usando o melhor valor. */
  audienceUsesBestValue: true
}

export type AtuacaoUsage = AtuacaoApresentacao | AtuacaoImpressionar

// ─── Constantes ──────────────────────────────────────────────────────
// Apresentação (p116 verbatim)
export const APRESENTACAO_CD = 20
export const APRESENTACAO_BASE_TIBAR_DICE = '1d6'
export const APRESENTACAO_EXTRA_DIE_PER_MARGIN = 5

/** Multiplicador de payout por qualidade do local (p116 verbatim). */
export const LOCATION_QUALITY_MULTIPLIER: Readonly<
  Record<LocationQuality, number>
> = Object.freeze({
  inadequado: 0.5,
  propicio: 1,
  'especialmente-propicio': 2,
})

// Impressionar (p116 verbatim)
export const IMPRESSIONAR_SUCCESS_CHARISMA_BONUS = 2
export const IMPRESSIONAR_FAILURE_CHARISMA_PENALTY = -2

// Flags Tabela 2-1 p115
export const ATUACAO_TRAINED_ONLY = true
export const ATUACAO_ARMOR_PENALTY = false

// ─── Catálogo ─────────────────────────────────────────────────────────
export const ATUACAO_USAGES: readonly AtuacaoUsage[] = Object.freeze([
  {
    id: 'apresentacao',
    kind: 'apresentacao',
    name: 'Apresentação',
    dc: 20,
    duration: 'um-dia-ou-noite',
    baseTibarDice: '1d6',
    extraDicePerMargin: 5,
    effect:
      'CD 20; 1 dia/noite; T$ 1d6 + 1d6 por 5 pts acima da CD; local ajusta payout (inadequado ½, propício 1×, especialmente propício 2×).',
    bookPage: 116,
  },
  {
    id: 'impressionar',
    kind: 'impressionar',
    name: 'Impressionar',
    opposedBy: 'vontade',
    successCharismaSkillBonus: 2,
    failureCharismaSkillPenalty: -2,
    failureLocksOutSameDay: true,
    audienceUsesBestValue: true,
    effect:
      'Oposto vs Vontade; sucesso +2 em perícias CAR contra alvo no dia; falha -2 e trava tentativas hoje; plateia usa melhor valor em um teste.',
    bookPage: 116,
  },
])

export const atuacaoUsageByKind = makeUsageByKind<AtuacaoUsageKind, AtuacaoUsage>(
  ATUACAO_USAGES,
  'atuacaoUsageByKind',
)

// ─── Helpers — Apresentação ─────────────────────────────────────────
/**
 * Quantidade de d6s pagos ao passar:
 *  - roll < 20 → 0 (falha).
 *  - roll ≥ 20 → 1 + floor((roll - 20) / 5).
 */
export function apresentacaoD6Count(rollResult: number): number {
  if (rollResult < APRESENTACAO_CD) return 0
  return (
    1 +
    Math.floor(
      (rollResult - APRESENTACAO_CD) / APRESENTACAO_EXTRA_DIE_PER_MARGIN,
    )
  )
}

/** Multiplicador do payout por qualidade do local. */
export function apresentacaoLocationMultiplier(
  location: LocationQuality,
): number {
  return LOCATION_QUALITY_MULTIPLIER[location]
}

/**
 * Payout esperado em T$ dado o valor bruto do d6 total e o local.
 * (Não rola dados — recebe soma dos d6 e aplica multiplicador.)
 */
export function apresentacaoPayout(
  d6TotalTibar: number,
  location: LocationQuality,
): number {
  if (d6TotalTibar < 0) {
    throw new Error(
      `apresentacaoPayout: d6TotalTibar must be ≥ 0, got ${d6TotalTibar}`,
    )
  }
  return d6TotalTibar * apresentacaoLocationMultiplier(location)
}

// ─── Helpers — Impressionar ─────────────────────────────────────────
export type ImpressionarOutcome = 'success' | 'failed'

/**
 * Resolve Impressionar contra Vontade do alvo (empate favorece
 * atacante — convenção T20).
 */
export function impressionarOutcome(
  atuacaoRoll: number,
  targetVontadeRoll: number,
): ImpressionarOutcome {
  return atuacaoRoll >= targetVontadeRoll ? 'success' : 'failed'
}

/**
 * Modificador aplicado a testes de perícias baseadas em Carisma
 * contra o alvo no mesmo dia após o resultado de Impressionar.
 */
export function impressionarCharismaModifier(
  outcome: ImpressionarOutcome,
): number {
  return outcome === 'success'
    ? IMPRESSIONAR_SUCCESS_CHARISMA_BONUS
    : IMPRESSIONAR_FAILURE_CHARISMA_PENALTY
}

/**
 * CD do Impressionar contra plateia: melhor Vontade entre os alvos
 * (mestre faz um único teste pela plateia com o melhor valor).
 */
export function impressionarAudienceCd(
  vontadeRolls: readonly number[],
): number {
  if (vontadeRolls.length === 0) {
    throw new Error('impressionarAudienceCd: vontadeRolls must be non-empty')
  }
  return Math.max(...vontadeRolls)
}
