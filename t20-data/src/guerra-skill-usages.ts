/**
 * Perícia Guerra (INT, treinada, sem penalidade de armadura) — 2 usos.
 *
 * PDF Cap 2 Perícias — Guerra (livro p119). Header verbatim:
 * "GUERRA — INT · TREINADA".
 * Intro verbatim: "Você foi educado em tática, estratégia e logística."
 *
 * Usos:
 *  - Analisar Terreno (p119) — CD 20; ação de movimento; sucesso
 *    revela uma vantagem do campo (cobertura, camuflagem ou terreno
 *    elevado, se houver).
 *  - Plano de Ação (p119) — CD 20; ação padrão; orienta um aliado em
 *    alcance médio; sucesso concede +5 na Iniciativa dele; se a nova
 *    Iniciativa passar a do usuário E o aliado ainda não tiver agido
 *    nesta rodada, ele age imediatamente após o turno do usuário; nas
 *    próximas rodadas segue a nova ordem.
 *
 * Nota Tabela 2-1 p115: APENAS TREINADA; NÃO sofre penalidade de
 * armadura. Nenhuma cross-ref explícita no bloco.
 */

// ─── Types ────────────────────────────────────────────────────────────
export type GuerraUsageKind = 'analisar-terreno' | 'plano-de-acao'

/** Vantagem revelada por Analisar Terreno (verbatim p119). */
export type TerrainAdvantage = 'cobertura' | 'camuflagem' | 'terreno-elevado'

type UsageCommon = {
  id: GuerraUsageKind
  name: string
  effect: string
  bookPage: 119
}

export type GuerraAnalisarTerreno = UsageCommon & {
  kind: 'analisar-terreno'
  action: 'movimento'
  dc: 20
  /** Sucesso apenas revela vantagem existente; não confere bônus por si. */
  revealsAdvantage: true
}

export type GuerraPlanoDeAcao = UsageCommon & {
  kind: 'plano-de-acao'
  action: 'padrao'
  dc: 20
  targetRange: 'medio'
  iniciativaBonus: 5
}

export type GuerraUsage = GuerraAnalisarTerreno | GuerraPlanoDeAcao

// ─── Constantes ──────────────────────────────────────────────────────
// Analisar Terreno (p119 verbatim)
export const ANALISAR_TERRENO_CD = 20

// Plano de Ação (p119 verbatim)
export const PLANO_DE_ACAO_CD = 20
export const PLANO_DE_ACAO_INICIATIVA_BONUS = 5

// Flags Tabela 2-1 p115
export const GUERRA_TRAINED_ONLY = true
export const GUERRA_ARMOR_PENALTY = false

// ─── Catálogo ─────────────────────────────────────────────────────────
export const GUERRA_USAGES: readonly GuerraUsage[] = Object.freeze([
  {
    id: 'analisar-terreno',
    kind: 'analisar-terreno',
    name: 'Analisar Terreno',
    action: 'movimento',
    dc: 20,
    revealsAdvantage: true,
    effect:
      'Ação de movimento; CD 20; sucesso revela vantagem existente (cobertura, camuflagem ou terreno elevado).',
    bookPage: 119,
  },
  {
    id: 'plano-de-acao',
    kind: 'plano-de-acao',
    name: 'Plano de Ação',
    action: 'padrao',
    dc: 20,
    targetRange: 'medio',
    iniciativaBonus: 5,
    effect:
      'Ação padrão; CD 20; aliado em alcance médio; sucesso concede +5 Iniciativa; se nova Iniciativa > sua E aliado não agiu, age logo após seu turno; próximas rodadas seguem nova ordem.',
    bookPage: 119,
  },
])

const usagesByKind = new Map<GuerraUsageKind, GuerraUsage>(
  GUERRA_USAGES.map((u) => [u.kind, u]),
)

export function guerraUsageByKind(kind: GuerraUsageKind): GuerraUsage {
  const usage = usagesByKind.get(kind)
  if (!usage) {
    throw new Error(`guerraUsageByKind: unknown kind ${kind}`)
  }
  return usage
}

// ─── Helpers — Analisar Terreno ─────────────────────────────────────
/** CD 20 (verbatim). */
export function analisarTerrenoCd(): number {
  return ANALISAR_TERRENO_CD
}

// ─── Helpers — Plano de Ação ────────────────────────────────────────
/** CD 20 (verbatim). */
export function planoDeAcaoCd(): number {
  return PLANO_DE_ACAO_CD
}

/** +5 na Iniciativa do aliado alvo. */
export function planoDeAcaoIniciativaBonus(): number {
  return PLANO_DE_ACAO_INICIATIVA_BONUS
}

export type PlanoDeAcaoOrderOutcome =
  | 'acts-immediately-after-caster'
  | 'new-order-next-round'

/**
 * Resolve o efeito de reordenação na rodada atual:
 *  - Se nova Iniciativa do aliado > Iniciativa do usuário E aliado
 *    ainda não agiu nesta rodada → age imediatamente após o turno
 *    do usuário.
 *  - Caso contrário (nova ≤ usuário OU já agiu) → aliado segue nova
 *    ordem apenas nas próximas rodadas.
 */
export function planoDeAcaoRoundOrder(
  callerIniciativa: number,
  allyNewIniciativa: number,
  allyAlreadyActedThisRound: boolean,
): PlanoDeAcaoOrderOutcome {
  if (allyNewIniciativa > callerIniciativa && !allyAlreadyActedThisRound) {
    return 'acts-immediately-after-caster'
  }
  return 'new-order-next-round'
}
