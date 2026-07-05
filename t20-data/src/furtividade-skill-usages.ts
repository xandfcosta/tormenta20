/**
 * Perícia Furtividade (DES, penalidade de armadura) — 2 usos.
 *
 * PDF Cap 2 Perícias — Furtividade (livro p119). Header verbatim:
 * "FURTIVIDADE — Des · Armadura".
 * Intro verbatim: "Você sabe ser discreto e sorrateiro."
 *
 * Usos:
 *  - Esconder-se (p119) — ação livre no final do turno, oposto Percepção;
 *    -5 se se moveu (metade do deslocamento evita); -20 se atacou/ação
 *    chamativa; sucesso vs cada observador = camuflagem total contra ele.
 *  - Seguir (p119) — teste oposto Percepção da vítima; -5 em local sem
 *    esconderijo ou sem movimento; vítima +5 em Percepção se tomando
 *    precauções; falha = vítima percebe na metade do caminho.
 *
 * Nota Tabela 2-1 p115: NÃO é somente treinada; SOFRE penalidade de
 * armadura (header p119 lista apenas "Des · Armadura", sem "Treinada").
 */

import { makeUsageByKind } from './skill-usage-resolver'

// ─── Types ────────────────────────────────────────────────────────────
export type FurtividadeUsageKind = 'esconder-se' | 'seguir'

/** Ritmo de movimento durante o turno para Esconder-se (p119). */
export type EsconderSeMovement = 'parado' | 'metade-deslocamento' | 'deslocamento-completo'

/** Ação chamativa realizada no turno (ataque, magia, etc.). */
export type EsconderSeAction = 'nenhuma' | 'atacou-ou-chamativa'

/** Terreno onde a perseguição ocorre (p119). */
export type SeguirTerrain = 'com-esconderijos' | 'sem-esconderijos-ou-movimento'

type UsageCommon = {
  id: FurtividadeUsageKind
  name: string
  effect: string
  bookPage: 119
}

export type FurtividadeEsconderSe = UsageCommon & {
  kind: 'esconder-se'
  action: 'livre-fim-de-turno'
  opposedBy: 'percepcao'
  movedPenalty: -5
  /** Metade do deslocamento evita a penalidade de -5. */
  halfSpeedAvoidsMovementPenalty: true
  attackOrFlashyPenalty: -20
  /** Sucesso vs um observador = camuflagem total contra ele. */
  successGrantsCamuflagemTotal: true
}

export type FurtividadeSeguir = UsageCommon & {
  kind: 'seguir'
  action: 'estendida'
  opposedBy: 'percepcao'
  noCoverPenalty: -5
  targetPrecautionPercepcaoBonus: 5
  /** Falha faz a vítima perceber na metade do caminho. */
  failureNoticeFraction: 0.5
}

export type FurtividadeUsage = FurtividadeEsconderSe | FurtividadeSeguir

// ─── Constantes ──────────────────────────────────────────────────────
// Esconder-se (p119 verbatim)
export const ESCONDER_SE_MOVED_PENALTY = -5
export const ESCONDER_SE_ATTACK_OR_FLASHY_PENALTY = -20

// Seguir (p119 verbatim)
export const SEGUIR_NO_COVER_PENALTY = -5
export const SEGUIR_TARGET_PRECAUTION_PERCEPCAO_BONUS = 5
/** Vítima percebe o perseguidor na metade do caminho ao falhar. */
export const SEGUIR_FAILURE_NOTICE_FRACTION = 0.5

// Flags Tabela 2-1 p115
export const FURTIVIDADE_TRAINED_ONLY = false
export const FURTIVIDADE_ARMOR_PENALTY = true

// ─── Catálogo ─────────────────────────────────────────────────────────
export const FURTIVIDADE_USAGES: readonly FurtividadeUsage[] = Object.freeze([
  {
    id: 'esconder-se',
    kind: 'esconder-se',
    name: 'Esconder-se',
    action: 'livre-fim-de-turno',
    opposedBy: 'percepcao',
    movedPenalty: -5,
    halfSpeedAvoidsMovementPenalty: true,
    attackOrFlashyPenalty: -20,
    successGrantsCamuflagemTotal: true,
    effect:
      'Ação livre no fim do turno; oposto Percepção; camuflagem total vs quem falhar; -5 se moveu (metade do deslocamento evita); -20 se atacou/chamativa.',
    bookPage: 119,
  },
  {
    id: 'seguir',
    kind: 'seguir',
    name: 'Seguir',
    action: 'estendida',
    opposedBy: 'percepcao',
    noCoverPenalty: -5,
    targetPrecautionPercepcaoBonus: 5,
    failureNoticeFraction: 0.5,
    effect:
      'Teste oposto Percepção; -5 sem esconderijo/movimento; vítima +5 se precavida; falha = vítima percebe na metade do caminho.',
    bookPage: 119,
  },
])

export const furtividadeUsageByKind = makeUsageByKind<FurtividadeUsageKind, FurtividadeUsage>(
  FURTIVIDADE_USAGES,
  'furtividadeUsageByKind',
)

// ─── Helpers — Esconder-se ───────────────────────────────────────────
/**
 * Penalidade de movimento no teste de Furtividade:
 *  - parado → 0
 *  - metade-deslocamento → 0 (evita a penalidade)
 *  - deslocamento-completo → -5
 */
export function esconderSeMovementPenalty(movement: EsconderSeMovement): number {
  if (movement === 'deslocamento-completo') return ESCONDER_SE_MOVED_PENALTY
  return 0
}

/** Penalidade se atacou ou fez ação muito chamativa no turno. */
export function esconderSeActionPenalty(action: EsconderSeAction): number {
  return action === 'atacou-ou-chamativa' ? ESCONDER_SE_ATTACK_OR_FLASHY_PENALTY : 0
}

/** Modificador total no teste de Furtividade para Esconder-se. */
export function esconderSeTotalModifier(
  movement: EsconderSeMovement,
  action: EsconderSeAction,
): number {
  return esconderSeMovementPenalty(movement) + esconderSeActionPenalty(action)
}

export type EsconderSeOutcome = 'camuflagem-total' | 'percebido'

/**
 * Resolve Esconder-se contra um observador:
 *  - Furtividade > Percepção do observador → camuflagem total contra ele.
 *  - Empate ou menor → observador percebe (padrão T20: atacante vence
 *    empate em oposto, mas Esconder-se favorece o observador porque
 *    "todos que falharem" implica que ele precisa falhar estritamente).
 *
 * Nota: T20 core usa "quem falhar" (< resultado do atacante); empate
 * conta como falha do observador → escondido.
 */
export function esconderSeOutcome(
  furtividadeRoll: number,
  observerPercepcaoRoll: number,
): EsconderSeOutcome {
  return observerPercepcaoRoll < furtividadeRoll ? 'camuflagem-total' : 'percebido'
}

// ─── Helpers — Seguir ────────────────────────────────────────────────
/** Penalidade no teste de Furtividade conforme terreno. */
export function seguirTerrainPenalty(terrain: SeguirTerrain): number {
  return terrain === 'sem-esconderijos-ou-movimento' ? SEGUIR_NO_COVER_PENALTY : 0
}

/** Bônus no Percepção da vítima se ela toma precauções contra perseguição. */
export function seguirTargetPercepcaoBonus(isTakingPrecautions: boolean): number {
  return isTakingPrecautions ? SEGUIR_TARGET_PRECAUTION_PERCEPCAO_BONUS : 0
}

export type SeguirOutcome = 'segue-ate-destino' | 'percebido-na-metade'

/**
 * Resolve Seguir:
 *  - Furtividade >= Percepção da vítima → segue até o destino.
 *  - Menor → vítima percebe na metade do caminho.
 */
export function seguirOutcome(
  furtividadeRoll: number,
  targetPercepcaoRoll: number,
): SeguirOutcome {
  return furtividadeRoll >= targetPercepcaoRoll ? 'segue-ate-destino' : 'percebido-na-metade'
}
