import {
  ATTRIBUTE_KEYS,
  type AttributeKey,
  type ItemEffects,
  collectVitalGrants,
  multiclassMpPool,
  multiclassPvPool,
} from '@tormenta20/t20-data'
import type { Character } from '@/shared/api/api'
import { attributeTotal, parseChoiceSet, parseClassChoices } from './derived'

type ClassEntry = { className: string; level: number }
type LevelVitals = {
  hpMax: number
  hpCurrent: number
  mpMax: number
  mpCurrent: number
}

/** Engine PV/PM pools for an arbitrary class list — same shared helpers the
 *  server sheet uses (multiclass p34-35 + p34 floor + vital grants). */
function enginePools(
  character: Character,
  effects: ItemEffects,
  classes: readonly ClassEntry[],
): { pv: number; pm: number } {
  const attrTotals = {} as Record<AttributeKey, number>
  for (const k of ATTRIBUTE_KEYS) {
    attrTotals[k] = attributeTotal(character, k, effects)
  }
  const grants = collectVitalGrants({
    level: classes.reduce((n, c) => n + c.level, 0),
    className: classes[0]?.className ?? '',
    classes,
    raceId: character.races[0]?.race,
    raceAbilityChoices: [...parseChoiceSet(character.raceAbilityChoices)],
    powerIds: [...parseChoiceSet(character.classPowers)],
    classChoices: parseClassChoices(character.classChoices),
    godPower: character.godPower || undefined,
    origin: character.origin || undefined,
    originChoices: [...parseChoiceSet(character.originChoices)],
    attrTotals,
  })
  return {
    pv: Math.max(0, multiclassPvPool(classes, attrTotals.constitution) + grants.pv),
    pm: Math.max(0, multiclassMpPool(classes) + grants.pm),
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
  const prev = enginePools(character, effects, character.classes)
  const next = enginePools(character, effects, nextClasses)
  return {
    hpMax: next.pv,
    hpCurrent: shiftCurrent(character.hpCurrent, next.pv - prev.pv, next.pv),
    mpMax: next.pm,
    mpCurrent: shiftCurrent(character.mpCurrent, next.pm - prev.pm, next.pm),
  }
}

/** current + delta, kept inside [0, max] — same rule as the server patch. */
function shiftCurrent(current: number, delta: number, max: number): number {
  return Math.min(Math.max(0, current + delta), max)
}
