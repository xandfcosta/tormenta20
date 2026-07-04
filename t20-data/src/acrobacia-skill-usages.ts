/**
 * Perícia Acrobacia (DES, aberta, penalidade de armadura) — 6 usos.
 *
 * PDF Cap 2 Perícias — Acrobacia (livro p115). Header verbatim:
 * "ACROBACIA — DES · ARMADURA".
 * Intro verbatim: "Você consegue fazer proezas acrobáticas."
 *
 * Usos:
 *  - Amortecer Queda (p115) — CD 15, APENAS TREINADO, reação; -1d6 base
 *    + 1d6 por 5 pts acima da CD; se reduz dano a 0, cai de pé.
 *  - Equilíbrio (p115) — CDs 10/15/20 por superfície; ação de movimento;
 *    metade do deslocamento; -5 para full speed; desprevenido; novo
 *    teste se sofrer dano (falha = cai).
 *  - Escapar (p115) — cordas CD = Des roll amarrador +10; redes CD 20;
 *    algemas CD 30; ação completa.
 *  - Levantar-se Rapidamente (p115) — CD 20, APENAS TREINADO; consome
 *    ação de movimento; sucesso = fica em pé como ação livre; falha
 *    consome ação e continua caído.
 *  - Passar por Espaço Apertado (p115) — CD 25, APENAS TREINADO; ação
 *    completa; metade do deslocamento; espaços de criatura 1 categoria
 *    menor.
 *  - Passar por Inimigo (p115) — teste oposto vs Acrobacia/Iniciativa/
 *    Luta do oponente (o melhor); parte do movimento; espaço = terreno
 *    difícil.
 */

// ─── Types ────────────────────────────────────────────────────────────
export type AcrobaciaUsageKind =
  | 'amortecer-queda'
  | 'equilibrio'
  | 'escapar'
  | 'levantar-se-rapidamente'
  | 'passar-por-espaco-apertado'
  | 'passar-por-inimigo'

/** Tipo de superfície para Equilíbrio (p115). */
export type EquilibrioSurface = 'piso-escorregadio' | 'estreita' | 'muito-estreita'

/** Amarra da qual se está escapando (p115). */
export type EscaparBinding =
  | { kind: 'cordas'; binderDestrezaRoll: number }
  | { kind: 'redes' }
  | { kind: 'algemas' }

/** Perícia/atributo do oponente contra a qual Passar por Inimigo é oposta. */
export type PassarPorInimigoOpposedBy = 'acrobacia' | 'iniciativa' | 'luta'

type UsageCommon = {
  id: AcrobaciaUsageKind
  name: string
  effect: string
  bookPage: 115
}

export type AcrobaciaAmortecerQueda = UsageCommon & {
  kind: 'amortecer-queda'
  action: 'reacao'
  dc: 15
  trainedOnly: true
  baseReductionDice: '1d6'
  extraDicePerMargin: 5
  /** Reduzir dano a zero faz cair de pé. */
  zeroDamageLandsOnFeet: true
}

export type AcrobaciaEquilibrio = UsageCommon & {
  kind: 'equilibrio'
  action: 'movimento'
  cdEscorregadio: 10
  cdEstreita: 15
  cdMuitoEstreita: 20
  fullSpeedPenalty: -5
  fallMargin: 5
  /** Quem se equilibra fica desprevenido. */
  isFlatFooted: true
  /** Sofrer dano exige novo teste; falha = cai. */
  damageForcesNewTest: true
}

export type AcrobaciaEscapar = UsageCommon & {
  kind: 'escapar'
  action: 'completa'
  cdRedes: 20
  cdAlgemas: 30
  cordasBinderBonus: 10
}

export type AcrobaciaLevantarSeRapidamente = UsageCommon & {
  kind: 'levantar-se-rapidamente'
  action: 'movimento'
  dc: 20
  trainedOnly: true
  /** Sucesso deixa em pé como ação livre; falha consome a ação de movimento. */
  successPromotesToFreeAction: true
}

export type AcrobaciaPassarPorEspacoApertado = UsageCommon & {
  kind: 'passar-por-espaco-apertado'
  action: 'completa'
  dc: 25
  trainedOnly: true
  /** Espaços para criatura N categorias menores. */
  sizeCategoriesSmaller: 1
  /** Avança metade do deslocamento. */
  halfSpeedMovement: true
}

export type AcrobaciaPassarPorInimigo = UsageCommon & {
  kind: 'passar-por-inimigo'
  action: 'parte-do-movimento'
  /** Oposto contra o MELHOR de Acrobacia/Iniciativa/Luta do inimigo. */
  opposedByBestOf: readonly PassarPorInimigoOpposedBy[]
  /** Espaço ocupado conta como terreno difícil. */
  countsAsDifficultTerrain: true
}

export type AcrobaciaUsage =
  | AcrobaciaAmortecerQueda
  | AcrobaciaEquilibrio
  | AcrobaciaEscapar
  | AcrobaciaLevantarSeRapidamente
  | AcrobaciaPassarPorEspacoApertado
  | AcrobaciaPassarPorInimigo

// ─── Constantes ──────────────────────────────────────────────────────
// Amortecer Queda (p115 verbatim)
export const AMORTECER_QUEDA_CD = 15
export const AMORTECER_QUEDA_TRAINED_ONLY = true
export const AMORTECER_QUEDA_EXTRA_DICE_PER_MARGIN = 5

// Equilíbrio (p115 verbatim)
export const EQUILIBRIO_CD_ESCORREGADIO = 10
export const EQUILIBRIO_CD_ESTREITA = 15
export const EQUILIBRIO_CD_MUITO_ESTREITA = 20
export const EQUILIBRIO_FULL_SPEED_PENALTY = -5
export const EQUILIBRIO_FALL_MARGIN = 5

// Escapar (p115 verbatim)
export const ESCAPAR_CD_REDES = 20
export const ESCAPAR_CD_ALGEMAS = 30
export const ESCAPAR_CORDAS_BINDER_BONUS = 10

// Levantar-se Rapidamente (p115 verbatim)
export const LEVANTAR_SE_RAPIDAMENTE_CD = 20
export const LEVANTAR_SE_RAPIDAMENTE_TRAINED_ONLY = true

// Passar por Espaço Apertado (p115 verbatim)
export const ESPACO_APERTADO_CD = 25
export const ESPACO_APERTADO_TRAINED_ONLY = true
export const ESPACO_APERTADO_SIZE_STEPS_SMALLER = 1

// Passar por Inimigo (p115 verbatim)
export const PASSAR_POR_INIMIGO_OPPOSED_BY: readonly PassarPorInimigoOpposedBy[] =
  Object.freeze(['acrobacia', 'iniciativa', 'luta'] as const)

// Flags Tabela 2-1 p115
export const ACROBACIA_TRAINED_ONLY = false
export const ACROBACIA_ARMOR_PENALTY = true

// ─── Catálogo ─────────────────────────────────────────────────────────
export const ACROBACIA_USAGES: readonly AcrobaciaUsage[] = Object.freeze([
  {
    id: 'amortecer-queda',
    kind: 'amortecer-queda',
    name: 'Amortecer Queda',
    action: 'reacao',
    dc: 15,
    trainedOnly: true,
    baseReductionDice: '1d6',
    extraDicePerMargin: 5,
    zeroDamageLandsOnFeet: true,
    effect:
      'Reação; CD 15; apenas treinado; -1d6 dano de queda + 1d6 por 5 acima da CD; dano zerado = cai de pé.',
    bookPage: 115,
  },
  {
    id: 'equilibrio',
    kind: 'equilibrio',
    name: 'Equilíbrio',
    action: 'movimento',
    cdEscorregadio: 10,
    cdEstreita: 15,
    cdMuitoEstreita: 20,
    fullSpeedPenalty: -5,
    fallMargin: 5,
    isFlatFooted: true,
    damageForcesNewTest: true,
    effect:
      'Movimento; CDs 10/15/20 por superfície; sucesso = metade do deslocamento; falha 5+ = cai; -5 para deslocamento total; desprevenido; dano força novo teste.',
    bookPage: 115,
  },
  {
    id: 'escapar',
    kind: 'escapar',
    name: 'Escapar',
    action: 'completa',
    cdRedes: 20,
    cdAlgemas: 30,
    cordasBinderBonus: 10,
    effect:
      'Ação completa; cordas CD = Des roll do amarrador +10; redes CD 20; algemas CD 30.',
    bookPage: 115,
  },
  {
    id: 'levantar-se-rapidamente',
    kind: 'levantar-se-rapidamente',
    name: 'Levantar-se Rapidamente',
    action: 'movimento',
    dc: 20,
    trainedOnly: true,
    successPromotesToFreeAction: true,
    effect:
      'Movimento; CD 20; apenas treinado; sucesso = fica em pé como ação livre; falha consome ação e continua caído.',
    bookPage: 115,
  },
  {
    id: 'passar-por-espaco-apertado',
    kind: 'passar-por-espaco-apertado',
    name: 'Passar por Espaço Apertado',
    action: 'completa',
    dc: 25,
    trainedOnly: true,
    sizeCategoriesSmaller: 1,
    halfSpeedMovement: true,
    effect:
      'Ação completa; CD 25; apenas treinado; espaços para criatura 1 categoria menor; metade do deslocamento.',
    bookPage: 115,
  },
  {
    id: 'passar-por-inimigo',
    kind: 'passar-por-inimigo',
    name: 'Passar por Inimigo',
    action: 'parte-do-movimento',
    opposedByBestOf: PASSAR_POR_INIMIGO_OPPOSED_BY,
    countsAsDifficultTerrain: true,
    effect:
      'Parte do movimento; teste oposto vs melhor de Acrobacia/Iniciativa/Luta do inimigo; falha encerra ação de movimento; espaço = terreno difícil.',
    bookPage: 115,
  },
])

const usagesByKind = new Map<AcrobaciaUsageKind, AcrobaciaUsage>(
  ACROBACIA_USAGES.map((u) => [u.kind, u]),
)

export function acrobaciaUsageByKind(kind: AcrobaciaUsageKind): AcrobaciaUsage {
  const usage = usagesByKind.get(kind)
  if (!usage) {
    throw new Error(`acrobaciaUsageByKind: unknown kind ${kind}`)
  }
  return usage
}

// ─── Helpers — Amortecer Queda ──────────────────────────────────────
/**
 * Número total de d6s reduzidos do dano da queda:
 *  - roll < 15 → 0 (falha).
 *  - roll ≥ 15 → 1 + floor((roll - 15) / 5).
 */
export function amortecerQuedaD6ReductionCount(rollResult: number): number {
  if (rollResult < AMORTECER_QUEDA_CD) return 0
  const extra = Math.floor(
    (rollResult - AMORTECER_QUEDA_CD) / AMORTECER_QUEDA_EXTRA_DICE_PER_MARGIN,
  )
  return 1 + extra
}

export type AmortecerQuedaOutcome = 'failed' | 'reduced' | 'landed-on-feet'

/**
 * Resolve Amortecer Queda dado o roll e o dano final após redução:
 *  - roll < 15 → failed.
 *  - dano final ≤ 0 → landed-on-feet.
 *  - resto → reduced.
 */
export function amortecerQuedaOutcome(
  rollResult: number,
  fallDamageAfterReduction: number,
): AmortecerQuedaOutcome {
  if (rollResult < AMORTECER_QUEDA_CD) return 'failed'
  return fallDamageAfterReduction <= 0 ? 'landed-on-feet' : 'reduced'
}

// ─── Helpers — Equilíbrio ────────────────────────────────────────────
/** CD por tipo de superfície. */
export function equilibrioCd(surface: EquilibrioSurface): number {
  switch (surface) {
    case 'piso-escorregadio':
      return EQUILIBRIO_CD_ESCORREGADIO
    case 'estreita':
      return EQUILIBRIO_CD_ESTREITA
    case 'muito-estreita':
      return EQUILIBRIO_CD_MUITO_ESTREITA
  }
}

/** Penalidade se optar por deslocamento total em vez de metade. */
export function equilibrioFullSpeedPenalty(fullSpeed: boolean): number {
  return fullSpeed ? EQUILIBRIO_FULL_SPEED_PENALTY : 0
}

export type EquilibrioOutcome = 'success' | 'no-progress' | 'falls'

/**
 * Resolve Equilíbrio:
 *  - roll ≥ CD → success (avança metade, ou total com -5).
 *  - CD > roll > CD-5 → no-progress (não avança, não cai).
 *  - roll ≤ CD-5 → falls.
 */
export function equilibrioOutcome(
  rollResult: number,
  cd: number,
): EquilibrioOutcome {
  const delta = rollResult - cd
  if (delta >= 0) return 'success'
  if (Math.abs(delta) >= EQUILIBRIO_FALL_MARGIN) return 'falls'
  return 'no-progress'
}

// ─── Helpers — Escapar ───────────────────────────────────────────────
/** CD por tipo de amarra. */
export function escaparCd(binding: EscaparBinding): number {
  switch (binding.kind) {
    case 'cordas':
      return binding.binderDestrezaRoll + ESCAPAR_CORDAS_BINDER_BONUS
    case 'redes':
      return ESCAPAR_CD_REDES
    case 'algemas':
      return ESCAPAR_CD_ALGEMAS
  }
}

// ─── Helpers — Passar por Inimigo ────────────────────────────────────
/** Melhor de Acrobacia/Iniciativa/Luta do oponente (verbatim p115). */
export function passarPorInimigoOpposedRoll(
  opponentAcrobacia: number,
  opponentIniciativa: number,
  opponentLuta: number,
): number {
  return Math.max(opponentAcrobacia, opponentIniciativa, opponentLuta)
}

export type PassarPorInimigoOutcome = 'crosses' | 'blocked'

/**
 * Resolve Passar por Inimigo (empate favorece atacante — convenção T20):
 *  - Sua Acrobacia ≥ melhor do oponente → crosses.
 *  - Menor → blocked (ação de movimento encerra).
 */
export function passarPorInimigoOutcome(
  ownAcrobaciaRoll: number,
  opponentBestRoll: number,
): PassarPorInimigoOutcome {
  return ownAcrobaciaRoll >= opponentBestRoll ? 'crosses' : 'blocked'
}
