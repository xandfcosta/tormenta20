/**
 * Ferimentos & Morte + Dano Não Letal + Golpe de Misericórdia.
 *
 * PDF Cap 5 Jogando p235-236. Este módulo cobre o que falta em
 * [[conditions]] (que já tem sangrando/inconsciente/indefeso):
 *  - Fórmula do limiar de morte (-10 ou -metade do PV total, o mais
 *    negativo).
 *  - Dano não letal como track paralela ao dano letal.
 *  - Golpe de Misericórdia: ação completa, crítico automático,
 *    chance de morte instantânea por tipo de alvo.
 *  - Resolver de estado de saúde (saudável / inconsciente-sangrando
 *    / estabilizado / morto) baseado em PV atual + PV máximo.
 *
 * Regras de sangrando (Const CD 15, perde 1d6/rodada) e condição
 * inconsciente vivem em [[conditions]]. Cura para estabilizar (CD 15)
 * vive em [[cura-skill-usages]].
 */

// ─── Types ────────────────────────────────────────────────────────────
export type DamageKind = 'letal' | 'nao-letal'

export type HealthState =
  | 'saudavel'
  | 'inconsciente-sangrando'
  | 'estabilizado'
  | 'morto'

/** Categoria do alvo para chance de morte instantânea em Golpe de Misericórdia (p235). */
export type CoupDeGraceTargetKind = 'pc-ou-npc-importante' | 'npc-secundario'

// ─── Constantes ──────────────────────────────────────────────────────
/** Limiar absoluto mínimo de morte (p236). */
export const DEATH_ABSOLUTE_THRESHOLD = -10

/** CD do teste de Constituição para auto-estabilizar por conta própria (p236). */
export const AUTO_STABILIZATION_CONSTITUICAO_CD = 15

/** CD do teste de Cura para estabilizar aliado sangrando (p236 / Cura skill). */
export const EXTERNAL_STABILIZATION_CURA_CD = 15

/** Penalidade -5 no ataque para converter dano letal ↔ não-letal (p236). */
export const NONLETHAL_CONVERSION_ATTACK_PENALTY = -5

// Golpe de Misericórdia (p235)
export const COUP_DE_GRACE_ACTION = 'completa' as const
/** Automatic critical hit — multiplier pela arma. */
export const COUP_DE_GRACE_AUTO_CRITICAL = true

/** Chance de morte instantânea por categoria de alvo (p235). */
export const COUP_DE_GRACE_INSTANT_DEATH_CHANCE: Readonly<
  Record<CoupDeGraceTargetKind, number>
> = Object.freeze({
  'pc-ou-npc-importante': 0.25,
  'npc-secundario': 0.75,
})

// ─── Helpers — Limiar de morte ──────────────────────────────────────
/**
 * Limiar de morte em PV: o menor (mais negativo) entre -10 e -metade
 * do PV total. Exemplos verbatim livro p236: 12 PV → morre a -10;
 * 30 PV → morre a -15.
 */
export function deathThreshold(maxPv: number): number {
  if (maxPv <= 0) {
    throw new Error(`deathThreshold: maxPv must be > 0, got ${maxPv}`)
  }
  const halfNegative = -Math.floor(maxPv / 2)
  return Math.min(DEATH_ABSOLUTE_THRESHOLD, halfNegative)
}

/** True se o PV atual está no ou abaixo do limiar de morte. */
export function isDead(currentPv: number, maxPv: number): boolean {
  return currentPv <= deathThreshold(maxPv)
}

// ─── Helpers — Estado de saúde ──────────────────────────────────────
/**
 * Resolve estado de saúde:
 *  - currentPv ≥ 1 → saudável (pode agir normalmente).
 *  - 1 > currentPv > deathThreshold → inconsciente + sangrando OU
 *    estabilizado (depende de flag externa).
 *  - currentPv ≤ deathThreshold → morto.
 *
 * A caller passes `isStabilized` para distinguir entre alguém ainda
 * sangrando e alguém que passou no teste de Constituição CD 15.
 */
export function healthState(
  currentPv: number,
  maxPv: number,
  isStabilized: boolean = false,
): HealthState {
  if (isDead(currentPv, maxPv)) return 'morto'
  if (currentPv >= 1) return 'saudavel'
  return isStabilized ? 'estabilizado' : 'inconsciente-sangrando'
}

/**
 * True se um efeito de cura que traga o PV para ≥ 1 tira o personagem
 * da inconsciência (p236: qualquer efeito que restaure ≥ 1 PV recupera
 * a consciência).
 */
export function healingRestoresConsciousness(
  newPv: number,
): boolean {
  return newPv >= 1
}

// ─── Helpers — Estabilização ────────────────────────────────────────
/** True se o teste de Constituição CD 15 estabiliza o sangrando. */
export function autoStabilizationSucceeds(constituicaoRoll: number): boolean {
  return constituicaoRoll >= AUTO_STABILIZATION_CONSTITUICAO_CD
}

/** True se o teste de Cura CD 15 estabiliza aliado sangrando. */
export function externalStabilizationSucceeds(curaRoll: number): boolean {
  return curaRoll >= EXTERNAL_STABILIZATION_CURA_CD
}

// ─── Helpers — Dano Não Letal ───────────────────────────────────────
/**
 * Ordem de aplicação de cura: cura letal primeiro, depois não-letal (p236).
 * Retorna `{lethalHealed, nonlethalHealed}` dado dano letal e não-letal
 * acumulados e uma quantidade `heal` aplicada.
 */
export function applyHealingPriority(
  lethalDamageTaken: number,
  nonlethalDamageTaken: number,
  heal: number,
): { lethalHealed: number; nonlethalHealed: number } {
  if (
    lethalDamageTaken < 0 ||
    nonlethalDamageTaken < 0 ||
    heal < 0
  ) {
    throw new Error(
      `applyHealingPriority: values must be ≥ 0 (got lethal=${lethalDamageTaken}, nonlethal=${nonlethalDamageTaken}, heal=${heal})`,
    )
  }
  const lethalHealed = Math.min(heal, lethalDamageTaken)
  const remaining = heal - lethalHealed
  const nonlethalHealed = Math.min(remaining, nonlethalDamageTaken)
  return { lethalHealed, nonlethalHealed }
}

/**
 * True se dano não-letal acumulado ≥ PV atual, derrubando o personagem
 * sem matar (p236). Diferente de dano letal: não gera sangrando/morte.
 */
export function nonlethalKnocksOut(
  nonlethalDamageTaken: number,
  currentPv: number,
): boolean {
  return nonlethalDamageTaken >= currentPv
}

// ─── Helpers — Golpe de Misericórdia ────────────────────────────────
/**
 * Chance (0-1) de morte instantânea no Golpe de Misericórdia por
 * categoria de alvo (p235).
 */
export function coupDeGraceInstantDeathChance(
  target: CoupDeGraceTargetKind,
): number {
  return COUP_DE_GRACE_INSTANT_DEATH_CHANCE[target]
}

/**
 * Resolve o rolo 1d4 do Golpe de Misericórdia:
 *  - PC/importante: morre com 1 em 1d4.
 *  - NPC secundário: morre com 1-3 em 1d4.
 */
export function coupDeGraceKills(
  d4Roll: 1 | 2 | 3 | 4,
  target: CoupDeGraceTargetKind,
): boolean {
  if (target === 'pc-ou-npc-importante') return d4Roll === 1
  return d4Roll <= 3
}
