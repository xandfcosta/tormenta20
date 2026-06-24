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
