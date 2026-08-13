import { CLASS_VITALS } from '@/shared/rules/class-vitals'
import { ATTRIBUTE_KEYS, type AttributeKey } from '@/shared/api/attribute-keys'
import { computeVitalPools, type VitalContext } from '@/entities/character/vital-pools'
import {
  type RaceChoiceState,
  appliedRaceDeltas,
  draftTormentaCarismaExtra,
} from './grant-helpers'

type VitalsValues = {
  classes: { className: string; level: number }[]
  races: string[]
  origin: string
  godPower?: string
  strength: number
  dexterity: number
  constitution: number
  intelligence: number
  wisdom: number
  charisma: number
  classPowers: string[]
  originChoices: string[]
  powerChoices?: Record<string, string[]>
  /** Per-class caminho/devoto picks — Caminho do Arcanista adds the key
   *  attribute to PM (p37), so the preview must see it. */
  classChoices?: Record<string, { devoto?: string; caminho?: string }>
}

export type DraftVitals = { pvMax: number; pmMax: number }

/**
 * PV/PM máximos for the creation preview, mirroring the server sheet
 * (`computeVitals`) but from the wizard's BASE attributes + race deltas — the
 * same base+delta model the live DEF uses, so the panel stays self-consistent.
 * Multiclass follows p34-35 (PV seed = 1ª classe, PM por classe) via the same
 * shared pools the engine uses. Race/attribute passive grants (Duro como
 * Pedra, Vontade de Ferro, caster attr→PM…) fold in via `collectVitalGrants`.
 */
export function deriveDraftVitals(
  v: VitalsValues,
  raceChoices: RaceChoiceState,
): DraftVitals {
  const primary = v.classes[0]?.className
  // Guard: no valid primary class yet (early wizard) → no vitals preview.
  if (!primary || !CLASS_VITALS[primary]) return { pvMax: 0, pmMax: 0 }
  const level = v.classes.reduce((n, c) => n + (c.level || 0), 0) || 1
  const classEntries = v.classes.filter((c) => c.className && c.level > 0)
  const deltas = appliedRaceDeltas(v.races, raceChoices)
  const attrTotals = {} as Record<AttributeKey, number>
  for (const k of ATTRIBUTE_KEYS) attrTotals[k] = (v[k] ?? 0) + (deltas[k] ?? 0)
  // CAR loss from pool/origem tormenta picks (CAR-scaled grants stay exact).
  attrTotals.charisma += draftTormentaCarismaExtra(
    v.races,
    raceChoices,
    v.classPowers ?? [],
    v.powerChoices ?? {},
    v.originChoices ?? [],
  )
  // `raceId` is the race NAME (the Go engine's getRace resolves by it) — v.races
  // holds names, so pass directly instead of the raceModel slug.
  const ctx: VitalContext = {
    level,
    classes: classEntries,
    raceId: v.races[0] ?? '',
    raceAbilityChoices: [],
    powerIds: v.classPowers ?? [],
    classChoices: v.classChoices ?? {},
    godPower: v.godPower ?? '',
    origin: v.origin ?? '',
    originChoices: v.originChoices ?? [],
    attrTotals,
  }
  return computeVitalPools(ctx)
}
