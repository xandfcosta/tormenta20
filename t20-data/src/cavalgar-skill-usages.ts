/**
 * Perícia Cavalgar (DES, aberta, sem penalidade de armadura) — 3 usos.
 *
 * PDF Cap 2 Perícias — Cavalgar (livro p116-117). Header verbatim:
 * "CAVALGAR — DES".
 * Intro verbatim: "Você sabe conduzir animais de montaria, como cavalos,
 * trobos e grifos. Ações simples não exigem testes — você pode encilhar,
 * montar, cavalgar em terreno plano e desmontar automaticamente. Ações
 * perigosas, entretanto, exigem testes na perícia."
 *
 * Usos:
 *  - Conduzir (p116) — CD 15 (terreno ruim / obstáculos pequenos) ou
 *    CD 25 (terreno perigoso / obstáculos grandes); parte do movimento;
 *    falha = cai da montaria e sofre 1d6 dano.
 *  - Galopar (p117) — ação completa; N quadrados de 1,5m = resultado
 *    do teste; ±2 por 1,5m de deslocamento acima/abaixo de 9m (mesma
 *    fórmula de Atletismo/Corrida).
 *  - Montar Rapidamente (p117) — CD 20; monta/desmonta como ação livre
 *    (normal é ação de movimento); falha por 5+ = cai no chão.
 *
 * Equipamento: exige sela; sem sela, -5 em todos os testes de Cavalgar.
 *
 * Nota Tabela 2-1 p115: NÃO é somente treinada; NÃO sofre penalidade de
 * armadura. Distinta de Adestramento/Manejar Animal (p115), que cobre
 * comandar animais como uma ação, e de Pilotagem (veículos).
 */

import { makeUsageByKind } from './skill-usage-resolver'

// ─── Types ────────────────────────────────────────────────────────────
export type CavalgarUsageKind = 'conduzir' | 'galopar' | 'montar-rapidamente'

/** Dificuldade do obstáculo em Conduzir (verbatim p116). */
export type ConduzirDifficulty =
  | 'ruim-obstaculo-pequeno'
  | 'perigoso-obstaculo-grande'

type UsageCommon = {
  id: CavalgarUsageKind
  name: string
  effect: string
  bookPage: 116 | 117
}

export type CavalgarConduzir = UsageCommon & {
  kind: 'conduzir'
  action: 'parte-do-movimento'
  cdRuim: 15
  cdPerigoso: 25
  fallDamageDice: '1d6'
}

export type CavalgarGalopar = UsageCommon & {
  kind: 'galopar'
  action: 'completa'
  squareMeters: 1.5
  baseSpeedMeters: 9
  modifierPerSpeedStep: 2
}

export type CavalgarMontarRapidamente = UsageCommon & {
  kind: 'montar-rapidamente'
  dc: 20
  /** Sucesso vira ação livre (normal é ação de movimento). */
  successPromotesToFreeAction: true
  /** Falha por 5+ derruba no chão. */
  fallMargin: 5
}

export type CavalgarUsage =
  | CavalgarConduzir
  | CavalgarGalopar
  | CavalgarMontarRapidamente

// ─── Constantes ──────────────────────────────────────────────────────
// Conduzir (p116 verbatim)
export const CONDUZIR_CD_RUIM = 15
export const CONDUZIR_CD_PERIGOSO = 25
export const CONDUZIR_FALL_DAMAGE_DICE = '1d6'

// Galopar (p117 verbatim)
export const GALOPAR_SQUARE_METERS = 1.5
export const GALOPAR_BASE_SPEED_METERS = 9
export const GALOPAR_MODIFIER_PER_SPEED_STEP = 2

// Montar Rapidamente (p117 verbatim)
export const MONTAR_RAPIDAMENTE_CD = 20
export const MONTAR_RAPIDAMENTE_FALL_MARGIN = 5

// Equipamento (p117 verbatim)
export const CAVALGAR_NO_SADDLE_PENALTY = -5

// Flags Tabela 2-1 p115
export const CAVALGAR_TRAINED_ONLY = false
export const CAVALGAR_ARMOR_PENALTY = false

// ─── Catálogo ─────────────────────────────────────────────────────────
export const CAVALGAR_USAGES: readonly CavalgarUsage[] = Object.freeze([
  {
    id: 'conduzir',
    kind: 'conduzir',
    name: 'Conduzir',
    action: 'parte-do-movimento',
    cdRuim: 15,
    cdPerigoso: 25,
    fallDamageDice: '1d6',
    effect:
      'Parte do movimento; CD 15 ruim/pequeno, CD 25 perigoso/grande; falha = cai da montaria + 1d6 dano.',
    bookPage: 116,
  },
  {
    id: 'galopar',
    kind: 'galopar',
    name: 'Galopar',
    action: 'completa',
    squareMeters: 1.5,
    baseSpeedMeters: 9,
    modifierPerSpeedStep: 2,
    effect:
      'Ação completa; avança quadrados 1,5m = roll; ±2 por 1,5m de deslocamento acima/abaixo de 9m.',
    bookPage: 117,
  },
  {
    id: 'montar-rapidamente',
    kind: 'montar-rapidamente',
    name: 'Montar Rapidamente',
    dc: 20,
    successPromotesToFreeAction: true,
    fallMargin: 5,
    effect:
      'CD 20; sucesso = monta/desmonta como ação livre (normal é ação de movimento); falha por 5+ = cai no chão.',
    bookPage: 117,
  },
])

export const cavalgarUsageByKind = makeUsageByKind<CavalgarUsageKind, CavalgarUsage>(
  CAVALGAR_USAGES,
  'cavalgarUsageByKind',
)

// ─── Helpers — Conduzir ─────────────────────────────────────────────
/** CD por dificuldade do obstáculo. */
export function conduzirCd(difficulty: ConduzirDifficulty): number {
  return difficulty === 'ruim-obstaculo-pequeno'
    ? CONDUZIR_CD_RUIM
    : CONDUZIR_CD_PERIGOSO
}

export type ConduzirOutcome = 'success' | 'falls-and-takes-1d6'

/** Resolve Conduzir (falha = cai + 1d6 dano). */
export function conduzirOutcome(rollResult: number, cd: number): ConduzirOutcome {
  return rollResult >= cd ? 'success' : 'falls-and-takes-1d6'
}

// ─── Helpers — Galopar ──────────────────────────────────────────────
/**
 * Modificador em Galopar por deslocamento base da montaria:
 * ±2 por 1,5m acima/abaixo de 9m (trunca para passos inteiros).
 */
export function galoparMovementModifier(deslocamentoMeters: number): number {
  const stepsFromBase = Math.trunc(
    (deslocamentoMeters - GALOPAR_BASE_SPEED_METERS) / GALOPAR_SQUARE_METERS,
  )
  return stepsFromBase * GALOPAR_MODIFIER_PER_SPEED_STEP
}

/** Quadrados de 1,5m percorridos ao galopar = resultado do teste. */
export function galoparSquaresCovered(rollResult: number): number {
  return rollResult
}

/** Metros percorridos ao galopar. */
export function galoparMetersCovered(rollResult: number): number {
  return rollResult * GALOPAR_SQUARE_METERS
}

// ─── Helpers — Montar Rapidamente ───────────────────────────────────
/** CD 20 (verbatim). */
export function montarRapidamenteCd(): number {
  return MONTAR_RAPIDAMENTE_CD
}

export type MontarRapidamenteOutcome =
  | 'free-action'
  | 'normal-movement-action'
  | 'falls-prone'

/**
 * Resolve Montar Rapidamente:
 *  - roll ≥ 20 → vira ação livre.
 *  - CD > roll > CD-5 → não vira livre (monta como ação de movimento normal).
 *  - roll ≤ CD-5 → cai no chão (falha por 5+).
 */
export function montarRapidamenteOutcome(
  rollResult: number,
  cd: number,
): MontarRapidamenteOutcome {
  const delta = rollResult - cd
  if (delta >= 0) return 'free-action'
  if (Math.abs(delta) >= MONTAR_RAPIDAMENTE_FALL_MARGIN) return 'falls-prone'
  return 'normal-movement-action'
}

// ─── Helpers — Equipamento ──────────────────────────────────────────
/** Penalidade global de -5 em todos os testes de Cavalgar sem sela. */
export function cavalgarSaddlePenalty(hasSaddle: boolean): number {
  return hasSaddle ? 0 : CAVALGAR_NO_SADDLE_PENALTY
}
