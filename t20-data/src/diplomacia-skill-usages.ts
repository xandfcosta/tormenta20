/**
 * Perícia Diplomacia (CAR) — 3 usos canônicos.
 *
 * PDF Cap 2 Perícias — Diplomacia (livro p118). Header verbatim:
 * "DIPLOMACIA — CAR" (sem flag treinada, sem penalidade de armadura).
 * Intro verbatim: "Você convence pessoas com lábia e argumentação.
 * Usos de Diplomacia são efeitos mentais."
 *
 * Usos:
 *  - Barganha — teste oposto vs Vontade; -10% preço, -20% se margem ≥ 10
 *  - Mudar Atitude — teste oposto vs Vontade; 1 min (ou completa @ -10);
 *    1 categoria (ou 2 se margem ≥ 10); falha 5+ = 1 categoria oposta;
 *    limite 1x/alvo/dia; tabela em p259
 *  - Persuasão — CD 20; -5 custoso; -10 perigoso (ou fail auto);
 *    opposed opcional Vontade
 *
 * Cross-ref:
 *  - Tabela de atitudes (Hostil→Prestativo) vive em p259, não aqui.
 *  - Encoded flag `isMentalEffect: true` — efeitos mentais respeitam
 *    imunidade a mental (bestiário/condições).
 */

import { makeUsageByKind } from './skill-usage-resolver'

// ─── Types ────────────────────────────────────────────────────────────
export type DiplomaciaUsageKind = 'barganha' | 'mudar-atitude' | 'persuasao'

/** Categorias de atitude (verbatim p118 + tabela p259). */
export type Atitude =
  | 'hostil'
  | 'inamistoso'
  | 'indiferente'
  | 'amistoso'
  | 'prestativo'

/** Eixo numérico para deslocamento de categorias (0..4). */
export const ATITUDE_ORDER: readonly Atitude[] = Object.freeze([
  'hostil',
  'inamistoso',
  'indiferente',
  'amistoso',
  'prestativo',
])

/** Ação exigida para Mudar Atitude. */
export type MudarAtitudeAction = 'um-minuto' | 'completa'

type UsageCommon = {
  id: DiplomaciaUsageKind
  name: string
  effect: string
  isMentalEffect: true
  bookPage: 118
}

export type DiplomaciaBarganha = UsageCommon & {
  kind: 'barganha'
  opposedBy: 'vontade'
  /** Sucesso simples: -10% no preço. */
  baseDiscountPct: 10
  /** Sucesso por 10 ou mais: -20%. */
  bigDiscountMargin: 10
  bigDiscountPct: 20
  /** Falha por 5 ou mais: negociante recusa por 1 semana. */
  offenseMargin: 5
  offenseRefusalWeeks: 1
}

export type DiplomaciaMudarAtitude = UsageCommon & {
  kind: 'mudar-atitude'
  opposedBy: 'vontade'
  /** 1 categoria; margem 10+ = 2 categorias; falha 5+ = 1 categoria inversa. */
  singleShiftCategories: 1
  bigShiftMargin: 10
  bigShiftCategories: 2
  reverseShiftFailMargin: 5
  reverseShiftCategories: 1
  /** Ação padrão 1 minuto; ação completa impõe -10 no teste. */
  standardAction: 'um-minuto'
  rushedAction: 'completa'
  rushedActionPenalty: -10
  /** Verbatim: "só pode mudar a atitude de um mesmo alvo uma vez por dia". */
  maxAttemptsPerTargetPerDay: 1
  /** Tabela de atitudes referenciada em p259. */
  attitudeTablePage: 259
}

export type DiplomaciaPersuasao = UsageCommon & {
  kind: 'persuasao'
  dc: 20
  /** -5 se pedido custoso (verbatim). */
  costlyPenalty: -5
  /** -10 ou falha automática se pedido perigoso (verbatim). */
  dangerousPenalty: -10
  dangerousAutoFail: true
  /** Oposição opcional a critério do mestre. */
  optionalOpposedBy: 'vontade'
}

export type DiplomaciaUsage =
  | DiplomaciaBarganha
  | DiplomaciaMudarAtitude
  | DiplomaciaPersuasao

// ─── Constantes ──────────────────────────────────────────────────────
// Barganha (p118 verbatim)
export const BARGANHA_BASE_DISCOUNT_PCT = 10
export const BARGANHA_BIG_DISCOUNT_MARGIN = 10
export const BARGANHA_BIG_DISCOUNT_PCT = 20
export const BARGANHA_OFFENSE_MARGIN = 5
export const BARGANHA_OFFENSE_REFUSAL_WEEKS = 1

// Mudar Atitude (p118 verbatim)
export const MUDAR_ATITUDE_BIG_SHIFT_MARGIN = 10
export const MUDAR_ATITUDE_REVERSE_FAIL_MARGIN = 5
export const MUDAR_ATITUDE_RUSHED_PENALTY = -10
export const MUDAR_ATITUDE_MAX_PER_TARGET_PER_DAY = 1

// Persuasão (p118 verbatim)
export const PERSUASAO_CD = 20
export const PERSUASAO_COSTLY_PENALTY = -5
export const PERSUASAO_DANGEROUS_PENALTY = -10

// Flags Tabela 2-1
export const DIPLOMACIA_TRAINED_ONLY = false
export const DIPLOMACIA_ARMOR_PENALTY = false

// ─── Catálogo ─────────────────────────────────────────────────────────
export const DIPLOMACIA_USAGES: readonly DiplomaciaUsage[] = Object.freeze([
  {
    id: 'barganha',
    kind: 'barganha',
    name: 'Barganha',
    isMentalEffect: true,
    opposedBy: 'vontade',
    baseDiscountPct: 10,
    bigDiscountMargin: 10,
    bigDiscountPct: 20,
    offenseMargin: 5,
    offenseRefusalWeeks: 1,
    effect:
      'Teste oposto vs Vontade; sucesso muda preço em 10%, margem 10+ = 20%; falha por 5+ ofende (1 semana).',
    bookPage: 118,
  },
  {
    id: 'mudar-atitude',
    kind: 'mudar-atitude',
    name: 'Mudar Atitude',
    isMentalEffect: true,
    opposedBy: 'vontade',
    singleShiftCategories: 1,
    bigShiftMargin: 10,
    bigShiftCategories: 2,
    reverseShiftFailMargin: 5,
    reverseShiftCategories: 1,
    standardAction: 'um-minuto',
    rushedAction: 'completa',
    rushedActionPenalty: -10,
    maxAttemptsPerTargetPerDay: 1,
    attitudeTablePage: 259,
    effect:
      'Move atitude 1 categoria (2 se margem 10+); falha 5+ move 1 na direção oposta; 1 min ou completa @ -10; 1x/alvo/dia.',
    bookPage: 118,
  },
  {
    id: 'persuasao',
    kind: 'persuasao',
    name: 'Persuasão',
    isMentalEffect: true,
    dc: 20,
    costlyPenalty: -5,
    dangerousPenalty: -10,
    dangerousAutoFail: true,
    optionalOpposedBy: 'vontade',
    effect:
      'CD 20; -5 se custoso; -10 (ou fail auto) se perigoso; mestre pode tornar oposto vs Vontade.',
    bookPage: 118,
  },
])

export const diplomaciaUsageByKind = makeUsageByKind<DiplomaciaUsageKind, DiplomaciaUsage>(
  DIPLOMACIA_USAGES,
  'diplomaciaUsageByKind',
)

// ─── Helpers — Barganha ──────────────────────────────────────────────
export type BarganhaOutcome = 'refused' | 'no-change' | 'discount' | 'big-discount'

/**
 * Resolve Barganha vs Vontade:
 *  - Vitória por 10+ → -20% (`big-discount`).
 *  - Vitória por 0-9 → -10% (`discount`).
 *  - Derrota por < 5 → sem mudança (`no-change`).
 *  - Derrota por 5+ → ofensa; 1 semana sem tratar (`refused`).
 */
export function barganhaOutcome(
  diplomaciaTotal: number,
  vontadeTotal: number,
): BarganhaOutcome {
  const delta = diplomaciaTotal - vontadeTotal
  if (delta >= BARGANHA_BIG_DISCOUNT_MARGIN) return 'big-discount'
  if (delta >= 0) return 'discount'
  if (Math.abs(delta) >= BARGANHA_OFFENSE_MARGIN) return 'refused'
  return 'no-change'
}

/** Desconto percentual efetivo dado um outcome de Barganha. */
export function barganhaDiscountPct(outcome: BarganhaOutcome): number {
  if (outcome === 'big-discount') return BARGANHA_BIG_DISCOUNT_PCT
  if (outcome === 'discount') return BARGANHA_BASE_DISCOUNT_PCT
  return 0
}

// ─── Helpers — Mudar Atitude ─────────────────────────────────────────
/** Modificador no teste conforme ação escolhida. */
export function mudarAtitudeActionPenalty(action: MudarAtitudeAction): number {
  return action === 'completa' ? MUDAR_ATITUDE_RUSHED_PENALTY : 0
}

/**
 * Deslocamento em categorias (positivo = melhora, negativo = piora).
 * Direção positiva assumida a favor do agente; use `.direction` no
 * chamador para escolher subir ou descer categoria em vitória.
 */
export function mudarAtitudeShift(
  diplomaciaTotal: number,
  vontadeTotal: number,
): number {
  const delta = diplomaciaTotal - vontadeTotal
  if (delta >= MUDAR_ATITUDE_BIG_SHIFT_MARGIN) return 2
  if (delta >= 0) return 1
  if (Math.abs(delta) >= MUDAR_ATITUDE_REVERSE_FAIL_MARGIN) return -1
  return 0
}

/** Índice numérico de uma atitude (0=hostil ... 4=prestativo). */
export function atitudeIndex(a: Atitude): number {
  const i = ATITUDE_ORDER.indexOf(a)
  if (i < 0) throw new Error(`atitudeIndex: unknown atitude ${a}`)
  return i
}

/**
 * Aplica shift à atitude, clamp em [hostil, prestativo].
 * `direction`: 'up' melhora (para agente), 'down' piora.
 */
export function applyAtitudeShift(
  from: Atitude,
  shift: number,
  direction: 'up' | 'down',
): Atitude {
  const i = atitudeIndex(from)
  const signed = direction === 'up' ? shift : -shift
  const next = Math.max(0, Math.min(ATITUDE_ORDER.length - 1, i + signed))
  return ATITUDE_ORDER[next]!
}

// ─── Helpers — Persuasão ─────────────────────────────────────────────
export type PersuasaoRequestKind = 'trivial' | 'costly' | 'dangerous'

export type PersuasaoModifier = {
  penalty: number
  autoFail: boolean
}

/**
 * Modificador do pedido:
 *  - trivial: 0
 *  - custoso: -5
 *  - perigoso: -10 (ou fail automático — mestre decide via `autoFailIfDangerous`)
 */
export function persuasaoModifier(
  kind: PersuasaoRequestKind,
  opts: { autoFailIfDangerous?: boolean } = {},
): PersuasaoModifier {
  if (kind === 'costly') return { penalty: PERSUASAO_COSTLY_PENALTY, autoFail: false }
  if (kind === 'dangerous') {
    return {
      penalty: PERSUASAO_DANGEROUS_PENALTY,
      autoFail: opts.autoFailIfDangerous === true,
    }
  }
  return { penalty: 0, autoFail: false }
}
