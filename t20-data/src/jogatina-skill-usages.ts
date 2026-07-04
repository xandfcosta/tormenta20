/**
 * Perícia Jogatina (CAR, treinada, sem penalidade de armadura) — 1 uso.
 *
 * PDF Cap 2 Perícias — Jogatina (livro p120). Header verbatim:
 * "JOGATINA — CAR · TREINADA".
 * Intro verbatim: "Você sabe jogar jogos de azar."
 *
 * Único uso:
 *  - Apostar (p120) — pague T$ 1d10 e faça teste; tabela de faixas
 *    de resultado dá o ganho (múltiplo da aposta). Mestre pode variar
 *    aposta básica de T$ 1d3 (taverna do porto) até 1d10 × T$ 1.000
 *    (cassino de luxo em Valkaria).
 *
 * Faixas do teste (verbatim p120):
 *  - ≤9   → nenhum ganho (perde aposta)
 *  - 10-14 → metade da aposta
 *  - 15-19 → valor da aposta (empata)
 *  - 20-29 → dobro
 *  - 30-39 → triplo
 *  - ≥40   → quíntuplo
 *
 * Nota Tabela 2-1 p115: APENAS TREINADA; NÃO sofre penalidade de
 * armadura. Nenhuma cross-ref explícita no bloco (apenas menção à
 * moeda T$ e à cidade de Valkaria como sabor).
 */

// ─── Types ────────────────────────────────────────────────────────────
export type JogatinaUsageKind = 'apostar'

/** Contexto do jogo (afeta a aposta básica sugerida pelo mestre). */
export type BetSetting = 'porto-taverna' | 'padrao' | 'cassino-luxo'

/** Faixa de payout do teste de Apostar (verbatim p120). */
export type ApostarPayoutTier =
  | 'nenhum'
  | 'metade'
  | 'empate'
  | 'dobro'
  | 'triplo'
  | 'quintuplo'

type UsageCommon = {
  id: JogatinaUsageKind
  name: string
  effect: string
  bookPage: 120
}

export type JogatinaApostar = UsageCommon & {
  kind: 'apostar'
  defaultBetRoll: '1d10'
  /** Duração narrativa: uma noite de jogatina. */
  duration: 'uma-noite'
}

export type JogatinaUsage = JogatinaApostar

// ─── Constantes ──────────────────────────────────────────────────────
// Apostar (p120 verbatim)
export const APOSTA_BASE_ROLL_EXPRESSION = '1d10'
export const APOSTA_PORTO_TAVERNA_ROLL_EXPRESSION = '1d3'
/** 1d10 × T$ 1000 (cassino de luxo em Valkaria). */
export const APOSTA_CASSINO_LUXO_ROLL_EXPRESSION = '1d10*1000'

/** Multiplicador do payout por faixa (verbatim p120). */
export const APOSTAR_TIER_MULTIPLIER: Readonly<
  Record<ApostarPayoutTier, number>
> = Object.freeze({
  nenhum: 0,
  metade: 0.5,
  empate: 1,
  dobro: 2,
  triplo: 3,
  quintuplo: 5,
})

// Flags Tabela 2-1 p115
export const JOGATINA_TRAINED_ONLY = true
export const JOGATINA_ARMOR_PENALTY = false

// ─── Catálogo ─────────────────────────────────────────────────────────
export const JOGATINA_USAGES: readonly JogatinaUsage[] = Object.freeze([
  {
    id: 'apostar',
    kind: 'apostar',
    name: 'Apostar',
    defaultBetRoll: '1d10',
    duration: 'uma-noite',
    effect:
      'Pague T$ 1d10 (ou aposta variável) e teste; ≤9 nada; 10-14 metade; 15-19 empata; 20-29 dobro; 30-39 triplo; ≥40 quíntuplo. Duração narrativa (uma noite).',
    bookPage: 120,
  },
])

const usagesByKind = new Map<JogatinaUsageKind, JogatinaUsage>(
  JOGATINA_USAGES.map((u) => [u.kind, u]),
)

export function jogatinaUsageByKind(kind: JogatinaUsageKind): JogatinaUsage {
  const usage = usagesByKind.get(kind)
  if (!usage) {
    throw new Error(`jogatinaUsageByKind: unknown kind ${kind}`)
  }
  return usage
}

// ─── Helpers — Apostar ──────────────────────────────────────────────
/** Faixa de payout dado o resultado do teste de Jogatina. */
export function apostarTierByRoll(rollTotal: number): ApostarPayoutTier {
  if (rollTotal <= 9) return 'nenhum'
  if (rollTotal <= 14) return 'metade'
  if (rollTotal <= 19) return 'empate'
  if (rollTotal <= 29) return 'dobro'
  if (rollTotal <= 39) return 'triplo'
  return 'quintuplo'
}

/** Multiplicador do payout dado o resultado do teste. */
export function apostarPayoutMultiplier(rollTotal: number): number {
  return APOSTAR_TIER_MULTIPLIER[apostarTierByRoll(rollTotal)]
}

/**
 * Recebido bruto em T$ dada a aposta paga e o resultado do teste.
 * Não desconta a aposta original — retorna o "ganho" da tabela.
 */
export function apostarPayoutGross(
  rollTotal: number,
  apostaTibar: number,
): number {
  if (apostaTibar < 0) {
    throw new Error(
      `apostarPayoutGross: apostaTibar must be ≥ 0, got ${apostaTibar}`,
    )
  }
  return apostaTibar * apostarPayoutMultiplier(rollTotal)
}

/**
 * Resultado líquido em T$ (payout - aposta paga).
 * Negativo = perda; zero = empate; positivo = lucro.
 */
export function apostarNetResult(
  rollTotal: number,
  apostaTibar: number,
): number {
  return apostarPayoutGross(rollTotal, apostaTibar) - apostaTibar
}

/** Expressão de rolagem sugerida pelo mestre para a aposta básica. */
export function apostaBaseRollExpression(setting: BetSetting): string {
  switch (setting) {
    case 'porto-taverna':
      return APOSTA_PORTO_TAVERNA_ROLL_EXPRESSION
    case 'padrao':
      return APOSTA_BASE_ROLL_EXPRESSION
    case 'cassino-luxo':
      return APOSTA_CASSINO_LUXO_ROLL_EXPRESSION
  }
}
