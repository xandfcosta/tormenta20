import {
  ATTRIBUTE_KEYS,
  type AttributeKey,
  CLASS_VITALS,
  collectVitalGrants,
} from '@tormenta20/t20-data'
import {
  type RaceChoiceState,
  appliedRaceDeltas,
  raceModel,
} from './grant-helpers'

type VitalsValues = {
  classes: { className: string; level: number }[]
  races: string[]
  origin: string
  strength: number
  dexterity: number
  constitution: number
  intelligence: number
  wisdom: number
  charisma: number
  classPowers: string[]
  originChoices: string[]
}

export type DraftVitals = { pvMax: number; pmMax: number }

/**
 * PV/PM máximos for the creation preview, mirroring the server sheet
 * (`computeVitals`) but from the wizard's BASE attributes + race deltas — the
 * same base+delta model the live DEF uses, so the panel stays self-consistent.
 * Multiclass collapses to primary class + total level, matching the backend
 * mapper (`toCharacterInput`). Race/attribute passive grants (Duro como Pedra,
 * Vontade de Ferro, caster attr→PM…) fold in via `collectVitalGrants`.
 */
export function deriveDraftVitals(
  v: VitalsValues,
  raceChoices: RaceChoiceState,
): DraftVitals {
  const primary = v.classes[0]?.className
  const vitals = primary ? CLASS_VITALS[primary] : undefined
  if (!vitals) return { pvMax: 0, pmMax: 0 }
  const level = v.classes.reduce((n, c) => n + (c.level || 0), 0) || 1
  const deltas = appliedRaceDeltas(v.races, raceChoices)
  const attrTotals = {} as Record<AttributeKey, number>
  for (const k of ATTRIBUTE_KEYS) attrTotals[k] = (v[k] ?? 0) + (deltas[k] ?? 0)
  const grants = collectVitalGrants({
    level,
    className: primary,
    raceId: raceModel(v.races[0] ?? '')?.id,
    powerIds: v.classPowers,
    origin: v.origin || undefined,
    originChoices: v.originChoices,
    attrTotals,
  })
  const pvBase =
    vitals.pvInicial +
    (level - 1) * vitals.pvPerLevel +
    attrTotals.constitution * level
  const pmBase = vitals.mpPerLevel * level
  return {
    pvMax: Math.max(0, pvBase + grants.pv),
    pmMax: Math.max(0, pmBase + grants.pm),
  }
}
