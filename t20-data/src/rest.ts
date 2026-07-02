/**
 * Descanso (rest) recovery — PDF p106 "Recuperando PV e PM".
 *
 * T20 has a *single* rest mechanic: 8h of sleep. Recovery is gated by
 * lodging quality (4 tiers). Pontos de vida and pontos de mana recover
 * by the same amount, never beyond their max.
 *
 *   Ruim         — ½ nível      (dormir ao relento, sem saco / acampamento)
 *   Normal       —   nível      (estalagem comum)
 *   Confortável  — 2× nível     (mid-tier inn / camp confortável)
 *   Luxuosa      — 3× nível     (luxury inn)
 *
 * Verbatim (book p106):
 *   "Com uma noite de descanso (pelo menos oito horas de sono), você
 *    recupera PV e PM de acordo com seu nível e condições de descanso."
 *
 * "Você nunca pode recuperar mais pontos de vida ou mana do que perdeu" —
 * caller clamps to max via `Math.min(curMax, currentValue + amount)`.
 *
 * Cuidados Prolongados (book p117 — Cura) layers +1/level/day per
 * patient on top of the lodging tier. Modeled here so the resolver
 * shows up in tests; UI/Caller decides whether the patient received the
 * care.
 */
export const REST_CONDITIONS = [
  'ruim',
  'normal',
  'confortavel',
  'luxuosa',
] as const

export type RestCondition = (typeof REST_CONDITIONS)[number]

export const REST_CONDITION_LABELS: Record<RestCondition, string> = {
  ruim: 'Ruim',
  normal: 'Normal',
  confortavel: 'Confortável',
  luxuosa: 'Luxuosa',
}

/**
 * PV/PM recovered from one 8h sleep at the given lodging tier. Same
 * value applies to PV and PM.
 *
 *   ruim        → Math.floor(level / 2)
 *   normal      → level
 *   confortavel → 2 * level
 *   luxuosa     → 3 * level
 *
 * Returns 0 for level < 1 (defensive — a level-0 character is not in
 * play).
 */
export function restRecoveryAmount(
  level: number,
  condition: RestCondition,
): number {
  if (level < 1) return 0
  switch (condition) {
    case 'ruim':
      return Math.floor(level / 2)
    case 'normal':
      return level
    case 'confortavel':
      return level * 2
    case 'luxuosa':
      return level * 3
  }
}

/**
 * Apply `Cuidados Prolongados` (PDF p117) on top of the base rest
 * amount. A trained Cura caregiver spends 1h per patient; each patient
 * gets `+1 per caregiver level` over the day. Returns the final amount
 * the patient recovers from the night's rest given the care.
 */
export function restRecoveryWithCare(
  level: number,
  condition: RestCondition,
  caregiverLevel: number,
): number {
  const base = restRecoveryAmount(level, condition)
  if (caregiverLevel < 1) return base
  return base + caregiverLevel
}

/** Duração mínima de um descanso para conferir recuperação (PDF p106). */
export const MIN_SLEEP_HOURS = 8

/**
 * Dormir de armadura pesada aplica a condição Fatigado durante o dia
 * seguinte (PDF p152). Constante exportada como marcador — a UI/resolver
 * decide se aplica a condição via `conditions.ts`.
 */
export const HEAVY_ARMOR_SLEEP_APPLIES_FATIGADO = true

/**
 * Descanso Natural — poder de Devoto de Allihanna (PDF p133).
 *
 *   "Para você, dormir ao relento conta como condição de descanso
 *    confortável."
 *
 * Substitui `ruim` (relento) por `confortavel`. Demais condições
 * (estalagem normal, quarto luxuoso) passam intactas.
 */
export function descansoNaturalOverride(
  condition: RestCondition,
): RestCondition {
  if (condition === 'ruim') return 'confortavel'
  return condition
}

/**
 * Sono forçado (PDF p318). Personagem que fica sem dormir uma noite:
 *  1. Não recupera PV nem PM.
 *  2. A partir da segunda noite sem dormir, Fortitude CD 15 +1 por
 *     teste anterior. Falha → fatigado; nova falha → exausto; nova
 *     falha → inconsciente até dormir 8h.
 */
export const SONO_FORT_CD_BASE = 15

/**
 * CD do teste de Fortitude por noite sem dormir.
 * `previousChecks = 0` (primeira noite após o limite) → 15.
 */
export function sonoForcadoFortCd(previousChecks: number): number {
  if (previousChecks < 0) {
    throw new Error(
      `sonoForcadoFortCd: previousChecks must be ≥ 0, got ${previousChecks}`,
    )
  }
  return SONO_FORT_CD_BASE + previousChecks
}

/** Recuperação de PV/PM ao pular uma noite de sono (PDF p318): sempre 0. */
export function noSleepRecovery(): number {
  return 0
}
