/**
 * Perícia Atletismo (FOR, aberta) — 4 usos.
 *
 * PDF Cap 2 Perícias — Atletismo (livro p116). Header verbatim:
 * "ATLETISMO — FOR".
 * Intro verbatim: "Você pode realizar façanhas atléticas."
 *
 * Usos:
 *  - Corrida (p116) — ação completa; avança N quadrados de 1,5m = roll;
 *    ±2 por 1,5m de deslocamento acima/abaixo de 9m; linha reta, sem
 *    terreno difícil; 1 + CON rodadas antes de Fortitude CD 15 (+1/
 *    teste anterior); falha = fatigado (cumulativo).
 *  - Escalar (p116) — ação de movimento; sucesso = ½ deslocamento;
 *    falha 5+ = cai; CDs 10/15/20/25 por superfície; -5 opcional para
 *    deslocamento total; desprevenido; dano força novo teste; pegar
 *    aliado caindo: CD superfície +10, falha 5+ = ambos caem.
 *  - Natação (p116) — ação de movimento no início do turno na água;
 *    CDs 10/15/20 (calma/agitada/tempestuosa); ½ deslocamento; falha
 *    = boia; falha 5+ = afunda; pode gastar 2ª ação de movimento
 *    para outro teste; prender respiração 1 + CON rodadas, depois
 *    Fortitude CD 15 (+1/anterior); falha = afoga (0 PV); ainda
 *    submerso perde 3d6 PV/rodada. SOFRE penalidade de armadura
 *    (situacional a este uso).
 *  - Saltar (p116) — parte do movimento (sem ação); salto longo CD 5
 *    por 1,5m; salto em altura CD 15 por 1,5m; sem 6m de impulso,
 *    +10 na CD.
 *
 * Nota Tabela 2-1 p115: NÃO é somente treinada; NÃO tem penalidade de
 * armadura GLOBAL (header p116 só lista "FOR"). Nadar aplica penalidade
 * de armadura por menção explícita dentro do uso Natação.
 */

import { makeUsageByKind } from './skill-usage-resolver'

// ─── Types ────────────────────────────────────────────────────────────
export type AtletismoUsageKind = 'corrida' | 'escalar' | 'natacao' | 'saltar'

/** Superfícies para Escalar (p116). */
export type EscalarSurface =
  | 'apoios-pes-e-maos'
  | 'arvore'
  | 'muro-com-reentrancias'
  | 'muro-liso'

/** Condições de água para Natação (p116). */
export type NatacaoWater = 'calma' | 'agitada' | 'tempestuosa'

/** Tipo de salto (p116). */
export type SaltarKind = 'longo' | 'altura'

type UsageCommon = {
  id: AtletismoUsageKind
  name: string
  effect: string
  bookPage: 116
}

export type AtletismoCorrida = UsageCommon & {
  kind: 'corrida'
  action: 'completa'
  squareMeters: 1.5
  baseSpeedMeters: 9
  modifierPerSpeedStep: 2
  fadigaCdBase: 15
  fadigaCdIncrement: 1
  straightLineOnly: true
  cannotEnterDifficultTerrain: true
}

export type AtletismoEscalar = UsageCommon & {
  kind: 'escalar'
  action: 'movimento'
  cdApoios: 10
  cdArvore: 15
  cdMuroReentrancias: 20
  cdMuroLiso: 25
  fullSpeedPenalty: -5
  fallMargin: 5
  isFlatFooted: true
  damageForcesNewTest: true
  catchAllyCdBonus: 10
}

export type AtletismoNatacao = UsageCommon & {
  kind: 'natacao'
  action: 'movimento'
  cdCalma: 10
  cdAgitada: 15
  cdTempestuosa: 20
  sinkMargin: 5
  /** Penalidade de armadura aplica dentro deste uso, apesar de a perícia não ter flag global. */
  armorPenaltyApplies: true
  holdBreathCdBase: 15
  holdBreathCdIncrement: 1
  drownStillSubmergedHpLossD6: 3
}

export type AtletismoSaltar = UsageCommon & {
  kind: 'saltar'
  action: 'parte-do-movimento'
  longJumpCdPer1p5m: 5
  highJumpCdPer1p5m: 15
  minRunUpMeters: 6
  noRunUpCdIncrease: 10
}

export type AtletismoUsage =
  | AtletismoCorrida
  | AtletismoEscalar
  | AtletismoNatacao
  | AtletismoSaltar

// ─── Constantes ──────────────────────────────────────────────────────
// Corrida (p116 verbatim)
export const CORRIDA_SQUARE_METERS = 1.5
export const CORRIDA_BASE_SPEED_METERS = 9
export const CORRIDA_MODIFIER_PER_SPEED_STEP = 2
export const CORRIDA_FADIGA_CD_BASE = 15
export const CORRIDA_FADIGA_CD_INCREMENT = 1

// Escalar (p116 verbatim)
export const ESCALAR_CD_APOIOS = 10
export const ESCALAR_CD_ARVORE = 15
export const ESCALAR_CD_MURO_REENTRANCIAS = 20
export const ESCALAR_CD_MURO_LISO = 25
export const ESCALAR_FULL_SPEED_PENALTY = -5
export const ESCALAR_FALL_MARGIN = 5
export const ESCALAR_CATCH_ALLY_CD_BONUS = 10

// Natação (p116 verbatim)
export const NATACAO_CD_CALMA = 10
export const NATACAO_CD_AGITADA = 15
export const NATACAO_CD_TEMPESTUOSA = 20
export const NATACAO_SINK_MARGIN = 5
export const NATACAO_HOLD_BREATH_CD_BASE = 15
export const NATACAO_HOLD_BREATH_CD_INCREMENT = 1
export const NATACAO_DROWN_STILL_SUBMERGED_HP_LOSS_D6 = 3

// Saltar (p116 verbatim)
export const SALTAR_LONG_JUMP_CD_PER_1_5M = 5
export const SALTAR_HIGH_JUMP_CD_PER_1_5M = 15
export const SALTAR_MIN_RUN_UP_METERS = 6
export const SALTAR_NO_RUN_UP_CD_INCREASE = 10

// Flags Tabela 2-1 p115
export const ATLETISMO_TRAINED_ONLY = false
/** Header p116 lista somente "FOR" — sem flag global de armadura. */
export const ATLETISMO_ARMOR_PENALTY = false
/** Uso Natação aplica penalidade de armadura por texto verbatim p116. */
export const ATLETISMO_NATACAO_ARMOR_PENALTY = true

// ─── Catálogo ─────────────────────────────────────────────────────────
export const ATLETISMO_USAGES: readonly AtletismoUsage[] = Object.freeze([
  {
    id: 'corrida',
    kind: 'corrida',
    name: 'Corrida',
    action: 'completa',
    squareMeters: 1.5,
    baseSpeedMeters: 9,
    modifierPerSpeedStep: 2,
    fadigaCdBase: 15,
    fadigaCdIncrement: 1,
    straightLineOnly: true,
    cannotEnterDifficultTerrain: true,
    effect:
      'Ação completa; avança N quadrados 1,5m = roll; ±2 por 1,5m de deslocamento acima/abaixo de 9m; 1+CON rodadas, depois Fortitude CD 15 (+1/anterior) ou fatigado.',
    bookPage: 116,
  },
  {
    id: 'escalar',
    kind: 'escalar',
    name: 'Escalar',
    action: 'movimento',
    cdApoios: 10,
    cdArvore: 15,
    cdMuroReentrancias: 20,
    cdMuroLiso: 25,
    fullSpeedPenalty: -5,
    fallMargin: 5,
    isFlatFooted: true,
    damageForcesNewTest: true,
    catchAllyCdBonus: 10,
    effect:
      'Movimento; CDs 10/15/20/25 por superfície; sucesso = ½ deslocamento; falha 5+ = cai; -5 para full; desprevenido; dano força novo teste; pegar aliado CD superfície+10.',
    bookPage: 116,
  },
  {
    id: 'natacao',
    kind: 'natacao',
    name: 'Natação',
    action: 'movimento',
    cdCalma: 10,
    cdAgitada: 15,
    cdTempestuosa: 20,
    sinkMargin: 5,
    armorPenaltyApplies: true,
    holdBreathCdBase: 15,
    holdBreathCdIncrement: 1,
    drownStillSubmergedHpLossD6: 3,
    effect:
      'Movimento na água; CDs 10/15/20; ½ deslocamento; falha = boia; falha 5+ = afunda; prender 1+CON rodadas, Fortitude CD 15 (+1/anterior) ou afoga (0 PV); segue submerso perde 3d6 PV/rodada. Sofre penalidade de armadura.',
    bookPage: 116,
  },
  {
    id: 'saltar',
    kind: 'saltar',
    name: 'Saltar',
    action: 'parte-do-movimento',
    longJumpCdPer1p5m: 5,
    highJumpCdPer1p5m: 15,
    minRunUpMeters: 6,
    noRunUpCdIncrease: 10,
    effect:
      'Parte do movimento; salto longo CD 5 por 1,5m; salto em altura CD 15 por 1,5m; sem 6m de impulso, +10 na CD.',
    bookPage: 116,
  },
])

export const atletismoUsageByKind = makeUsageByKind<AtletismoUsageKind, AtletismoUsage>(
  ATLETISMO_USAGES,
  'atletismoUsageByKind',
)

// ─── Helpers — Corrida ───────────────────────────────────────────────
/**
 * Modificador em testes de Corrida por deslocamento base:
 * ±2 por 1,5m acima/abaixo de 9m (elfo 12m = +4; anão 6m = -4).
 * Trunca para passos inteiros de 1,5m.
 */
export function corridaMovementModifier(deslocamentoMeters: number): number {
  const stepsFromBase = Math.trunc(
    (deslocamentoMeters - CORRIDA_BASE_SPEED_METERS) / CORRIDA_SQUARE_METERS,
  )
  return stepsFromBase * CORRIDA_MODIFIER_PER_SPEED_STEP
}

/** Quadrados de 1,5m percorridos em uma Corrida = resultado do teste. */
export function corridaSquaresCovered(rollResult: number): number {
  return rollResult
}

/** Metros percorridos em uma Corrida. */
export function corridaMetersCovered(rollResult: number): number {
  return rollResult * CORRIDA_SQUARE_METERS
}

/** Máximo de rodadas de corrida antes do primeiro teste de Fortitude. */
export function corridaMaxRoundsBeforeFadiga(constituicao: number): number {
  return 1 + constituicao
}

/** CD da Fortitude escalando por testes anteriores. */
export function corridaFadigaCd(previousTests: number): number {
  return CORRIDA_FADIGA_CD_BASE + previousTests * CORRIDA_FADIGA_CD_INCREMENT
}

// ─── Helpers — Escalar ──────────────────────────────────────────────
/** CD por superfície (p116). */
export function escalarCd(surface: EscalarSurface): number {
  switch (surface) {
    case 'apoios-pes-e-maos':
      return ESCALAR_CD_APOIOS
    case 'arvore':
      return ESCALAR_CD_ARVORE
    case 'muro-com-reentrancias':
      return ESCALAR_CD_MURO_REENTRANCIAS
    case 'muro-liso':
      return ESCALAR_CD_MURO_LISO
  }
}

/** -5 opcional para avançar deslocamento total em vez de metade. */
export function escalarFullSpeedPenalty(fullSpeed: boolean): number {
  return fullSpeed ? ESCALAR_FULL_SPEED_PENALTY : 0
}

/** CD para pegar aliado caindo = CD da superfície + 10. */
export function escalarCatchAllyCd(surface: EscalarSurface): number {
  return escalarCd(surface) + ESCALAR_CATCH_ALLY_CD_BONUS
}

export type EscalarOutcome = 'success' | 'no-progress' | 'falls'

/** Resolve Escalar (falha por 5+ = cai). */
export function escalarOutcome(rollResult: number, cd: number): EscalarOutcome {
  const delta = rollResult - cd
  if (delta >= 0) return 'success'
  if (Math.abs(delta) >= ESCALAR_FALL_MARGIN) return 'falls'
  return 'no-progress'
}

export type EscalarCatchOutcome = 'caught' | 'missed' | 'both-fall'

/** Resolve tentativa de pegar aliado caindo (falha por 5+ = cai junto). */
export function escalarCatchAllyOutcome(
  catcherRoll: number,
  catchCd: number,
): EscalarCatchOutcome {
  const delta = catcherRoll - catchCd
  if (delta >= 0) return 'caught'
  if (Math.abs(delta) >= ESCALAR_FALL_MARGIN) return 'both-fall'
  return 'missed'
}

// ─── Helpers — Natação ──────────────────────────────────────────────
/** CD por condição de água (p116). */
export function natacaoCd(water: NatacaoWater): number {
  switch (water) {
    case 'calma':
      return NATACAO_CD_CALMA
    case 'agitada':
      return NATACAO_CD_AGITADA
    case 'tempestuosa':
      return NATACAO_CD_TEMPESTUOSA
  }
}

export type NatacaoOutcome = 'advances' | 'floats' | 'sinks'

/** Resolve Natação (falha por 5+ = afunda). */
export function natacaoOutcome(rollResult: number, cd: number): NatacaoOutcome {
  const delta = rollResult - cd
  if (delta >= 0) return 'advances'
  if (Math.abs(delta) >= NATACAO_SINK_MARGIN) return 'sinks'
  return 'floats'
}

/** Rodadas de fôlego antes do primeiro teste de Fortitude. */
export function prenderRespiracaoMaxRounds(constituicao: number): number {
  return 1 + constituicao
}

/** CD Fortitude para não afogar escalando por testes anteriores. */
export function afogamentoCd(previousTests: number): number {
  return (
    NATACAO_HOLD_BREATH_CD_BASE +
    previousTests * NATACAO_HOLD_BREATH_CD_INCREMENT
  )
}

// ─── Helpers — Saltar ───────────────────────────────────────────────
/**
 * CD para salto (longo ou altura) com ceil para o próximo passo de 1,5m,
 * mais +10 se não teve 6m de impulso.
 */
export function saltarCd(
  kind: SaltarKind,
  distanceMeters: number,
  hasRunUp: boolean = true,
): number {
  const step = CORRIDA_SQUARE_METERS
  const cdPerStep =
    kind === 'longo' ? SALTAR_LONG_JUMP_CD_PER_1_5M : SALTAR_HIGH_JUMP_CD_PER_1_5M
  const steps = Math.max(1, Math.ceil(distanceMeters / step))
  const base = steps * cdPerStep
  return hasRunUp ? base : base + SALTAR_NO_RUN_UP_CD_INCREASE
}
