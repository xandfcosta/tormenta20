import { ATTRIBUTE_KEYS, type AttributeKey, type ItemEffects } from '@tormenta20/t20-data'
import type { Character } from '@/shared/api/api'
import { attributeTotal, parseChoiceSet, parseClassChoices } from './derived'
import { computeVitalPools, type VitalContext } from './vital-pools'

type ClassEntry = { className: string; level: number }
type LevelVitals = {
  hpMax: number
  hpCurrent: number
  mpMax: number
  mpCurrent: number
}

/**
 * Build the normalized `VitalContext` for a character + hypothetical class list —
 * the input the Go vitals engine (and the TS test fallback) consume. Exported so
 * the parity harness can feed the same context into `computeVitalPools` (the Inc.3
 * `vitals` oracle). `raceId` is the race NAME (getRace resolves by it).
 */
export function buildVitalContext(
  character: Character,
  effects: ItemEffects,
  classes: readonly ClassEntry[],
): VitalContext {
  const attrTotals = {} as Record<AttributeKey, number>
  for (const k of ATTRIBUTE_KEYS) {
    attrTotals[k] = attributeTotal(character, k, effects)
  }
  return {
    level: classes.reduce((n, c) => n + c.level, 0),
    classes: [...classes],
    raceId: character.races[0]?.race ?? '',
    raceAbilityChoices: [...parseChoiceSet(character.raceAbilityChoices)],
    powerIds: [...parseChoiceSet(character.classPowers)],
    classChoices: parseClassChoices(character.classChoices),
    godPower: character.godPower || '',
    origin: character.origin || '',
    originChoices: [...parseChoiceSet(character.originChoices)],
    attrTotals,
  }
}

/**
 * Optimistic mirror of the server's level vitals sync (`levelVitalsPatch`):
 * recomputes the pools for the NEW class levels and shifts the currents by
 * the max delta — level up heals the gained PV/PM, level down takes them
 * back, clamped into [0, newMax]. The authoritative server delta still
 * reconciles in onSuccess; this only removes the visual lag on the bars.
 *
 * @example optimisticLevelVitals(c, effects, [{ className: 'Guerreiro', level: 9 }])
 */
export function optimisticLevelVitals(
  character: Character,
  effects: ItemEffects,
  nextClasses: readonly ClassEntry[],
): LevelVitals {
  const prev = computeVitalPools(buildVitalContext(character, effects, character.classes))
  const next = computeVitalPools(buildVitalContext(character, effects, nextClasses))
  return {
    hpMax: next.pvMax,
    hpCurrent: shiftCurrent(character.hpCurrent, next.pvMax - prev.pvMax, next.pvMax),
    mpMax: next.pmMax,
    mpCurrent: shiftCurrent(character.mpCurrent, next.pmMax - prev.pmMax, next.pmMax),
  }
}

/** current + delta, kept inside [0, max] — same rule as the server patch. */
function shiftCurrent(current: number, delta: number, max: number): number {
  return Math.min(Math.max(0, current + delta), max)
}
