/**
 * Perícia Ladinagem (DES, treinada, penalidade de armadura) — 4 usos.
 *
 * PDF Cap 2 Perícias — Ladinagem (livro p120-121). Header verbatim:
 * "LADINAGEM — DES, treinada, penalidade de armadura".
 * Intro verbatim: "Você sabe exercer atividades ilícitas."
 *
 * Usos:
 *  - Abrir Fechadura (p120) — CD por qualidade; ação completa; gazua
 *  - Ocultar (p120) — ação padrão vs Percepção; ±5 por tamanho do objeto;
 *    revistador ganha +10 na Percepção
 *  - Punga (p120) — CD 20; ação padrão; Percepção da vítima CD = roll
 *  - Sabotar (p121) — CD 20 simples / 30 complexo; 1d4 rodadas ou
 *    ação completa @ -5; falha 5+ tem efeito colateral
 *
 * Nota: Ladinagem é a primeira perícia com `armorPenaltyApplies: true`
 * neste conjunto de módulos.
 */

// ─── Types ────────────────────────────────────────────────────────────
export type LadinagemUsageKind =
  | 'abrir-fechadura'
  | 'ocultar'
  | 'punga'
  | 'sabotar'

/** Qualidade da fechadura (verbatim p120). */
export type LockQuality = 'simples' | 'media' | 'superior'

/** Tamanho do objeto para Ocultar (verbatim p120). */
export type OcultarObjectSize = 'discreto-pequeno' | 'normal' | 'desajeitado-grande'

/** Complexidade da sabotagem (verbatim p121). */
export type SabotarComplexity = 'simples' | 'complexa'

/** Ritmo da sabotagem (padrão 1d4 rodadas OU ação completa @ -5). */
export type SabotarPace = 'padrao-1d4-rodadas' | 'ação-completa-com-penalidade'

type UsageCommon = {
  id: LadinagemUsageKind
  name: string
  effect: string
  bookPage: 120 | 121
}

export type LadinagemAbrirFechadura = UsageCommon & {
  kind: 'abrir-fechadura'
  action: 'completa'
  requiresGazua: true
  gazuaMissingPenalty: -5
}

export type LadinagemOcultar = UsageCommon & {
  kind: 'ocultar'
  action: 'padrao'
  opposedBy: 'percepcao'
  smallBonus: 5
  bigPenalty: -5
  searcherPercepcaoBonus: 10
}

export type LadinagemPunga = UsageCommon & {
  kind: 'punga'
  action: 'padrao'
  dc: 20
  /** Vítima faz Percepção com CD = resultado do seu teste de Ladinagem. */
  victimPercepcaoCdEqualsLadinagemRoll: true
  /** Serve para pegar OU plantar objeto. */
  canPlantObject: true
}

export type LadinagemSabotar = UsageCommon & {
  kind: 'sabotar'
  action: 'um-d4-rodadas'
  rushedAction: 'completa'
  rushedActionPenalty: -5
  dcSimples: 20
  dcComplexa: 30
  /** Falha por 5+ tem consequência (armadilha ativa, falso desabilitar…). */
  criticalFailureMargin: 5
}

export type LadinagemUsage =
  | LadinagemAbrirFechadura
  | LadinagemOcultar
  | LadinagemPunga
  | LadinagemSabotar

// ─── Constantes ──────────────────────────────────────────────────────
// Abrir Fechadura (p120 verbatim)
export const ABRIR_FECHADURA_CD: Readonly<Record<LockQuality, number>> = Object.freeze({
  simples: 20,
  media: 25,
  superior: 30,
})
export const ABRIR_FECHADURA_NO_GAZUA_PENALTY = -5

// Ocultar (p120 verbatim)
export const OCULTAR_SMALL_BONUS = 5
export const OCULTAR_BIG_PENALTY = -5
export const OCULTAR_SEARCHER_PERCEPCAO_BONUS = 10

// Punga (p120 verbatim)
export const PUNGA_CD = 20

// Sabotar (p121 verbatim)
export const SABOTAR_CD_SIMPLES = 20
export const SABOTAR_CD_COMPLEXA = 30
export const SABOTAR_RUSHED_PENALTY = -5
export const SABOTAR_CRITICAL_FAILURE_MARGIN = 5

// Flags Tabela 2-1
export const LADINAGEM_TRAINED_ONLY = true
export const LADINAGEM_ARMOR_PENALTY = true

// ─── Catálogo ─────────────────────────────────────────────────────────
export const LADINAGEM_USAGES: readonly LadinagemUsage[] = Object.freeze([
  {
    id: 'abrir-fechadura',
    kind: 'abrir-fechadura',
    name: 'Abrir Fechadura',
    action: 'completa',
    requiresGazua: true,
    gazuaMissingPenalty: -5,
    effect:
      'Abre fechadura trancada; CD 20/25/30 por qualidade; ação completa; sem gazua sofre -5.',
    bookPage: 120,
  },
  {
    id: 'ocultar',
    kind: 'ocultar',
    name: 'Ocultar',
    action: 'padrao',
    opposedBy: 'percepcao',
    smallBonus: 5,
    bigPenalty: -5,
    searcherPercepcaoBonus: 10,
    effect:
      'Esconde objeto em si; ação padrão vs Percepção; +5 discreto/pequeno, -5 desajeitado/grande; revistador +10.',
    bookPage: 120,
  },
  {
    id: 'punga',
    kind: 'punga',
    name: 'Punga',
    action: 'padrao',
    dc: 20,
    victimPercepcaoCdEqualsLadinagemRoll: true,
    canPlantObject: true,
    effect:
      'Rouba (ou planta) objeto; CD 20; ação padrão; vítima faz Percepção com CD = seu resultado.',
    bookPage: 120,
  },
  {
    id: 'sabotar',
    kind: 'sabotar',
    name: 'Sabotar',
    action: 'um-d4-rodadas',
    rushedAction: 'completa',
    rushedActionPenalty: -5,
    dcSimples: 20,
    dcComplexa: 30,
    criticalFailureMargin: 5,
    effect:
      'Desabilita dispositivo; CD 20 simples / 30 complexo; 1d4 rodadas (ou completa @ -5); falha 5+ dá efeito colateral.',
    bookPage: 121,
  },
])

const usagesByKind = new Map<LadinagemUsageKind, LadinagemUsage>(
  LADINAGEM_USAGES.map((u) => [u.kind, u]),
)

export function ladinagemUsageByKind(kind: LadinagemUsageKind): LadinagemUsage {
  const usage = usagesByKind.get(kind)
  if (!usage) {
    throw new Error(`ladinagemUsageByKind: unknown kind ${kind}`)
  }
  return usage
}

// ─── Helpers — Abrir Fechadura ───────────────────────────────────────
/** CD por qualidade da fechadura. */
export function abrirFechaduraCd(quality: LockQuality): number {
  return ABRIR_FECHADURA_CD[quality]
}

/** Penalidade agregada se sem gazua. */
export function abrirFechaduraPenalty(hasGazua: boolean): number {
  return hasGazua ? 0 : ABRIR_FECHADURA_NO_GAZUA_PENALTY
}

// ─── Helpers — Ocultar ───────────────────────────────────────────────
/** Modificador no teste de Ladinagem por tamanho do objeto. */
export function ocultarSizeModifier(size: OcultarObjectSize): number {
  if (size === 'discreto-pequeno') return OCULTAR_SMALL_BONUS
  if (size === 'desajeitado-grande') return OCULTAR_BIG_PENALTY
  return 0
}

/** Bônus no Percepção do observador se estiver revistando ativamente. */
export function ocultarObserverPercepcaoBonus(isSearching: boolean): number {
  return isSearching ? OCULTAR_SEARCHER_PERCEPCAO_BONUS : 0
}

// ─── Helpers — Punga ─────────────────────────────────────────────────
/** CD que a vítima faz na Percepção — igual ao resultado do seu roll. */
export function pungaVictimPercepcaoCd(ladinagemRoll: number): number {
  return ladinagemRoll
}

export type PungaOutcome =
  | 'success-unseen'
  | 'success-seen'
  | 'failed-unseen'
  | 'failed-seen'

/**
 * Resolve Punga:
 *  - Sucesso vs CD 20: você pega/planta.
 *  - Vítima faz Percepção com CD = seu roll de Ladinagem; sucesso dela
 *    = ela percebe (mesmo se você tiver pegado).
 */
export function pungaOutcome(
  ladinagemRoll: number,
  victimPercepcaoRoll: number,
): PungaOutcome {
  const stole = ladinagemRoll >= PUNGA_CD
  const noticed = victimPercepcaoRoll >= ladinagemRoll
  if (stole && noticed) return 'success-seen'
  if (stole) return 'success-unseen'
  if (noticed) return 'failed-seen'
  return 'failed-unseen'
}

// ─── Helpers — Sabotar ───────────────────────────────────────────────
/** CD por complexidade da sabotagem. */
export function sabotarCd(complexity: SabotarComplexity): number {
  return complexity === 'simples' ? SABOTAR_CD_SIMPLES : SABOTAR_CD_COMPLEXA
}

/** Modificador no teste conforme ritmo escolhido. */
export function sabotarPaceModifier(pace: SabotarPace): number {
  return pace === 'ação-completa-com-penalidade' ? SABOTAR_RUSHED_PENALTY : 0
}

export type SabotarOutcome = 'success' | 'failed' | 'critical-failure'

/**
 * Resolve Sabotar contra CD:
 *  - Passa → success.
 *  - Falha por < 5 → failed (sem efeito colateral).
 *  - Falha por ≥ 5 → critical-failure (armadilha ativa, falso
 *    desabilitar etc.).
 */
export function sabotarOutcome(
  checkResult: number,
  cd: number,
): SabotarOutcome {
  const delta = checkResult - cd
  if (delta >= 0) return 'success'
  if (Math.abs(delta) >= SABOTAR_CRITICAL_FAILURE_MARGIN) return 'critical-failure'
  return 'failed'
}
