import {
  ATTRIBUTE_KEYS,
  type AttributeKey,
  CLASS_VITALS,
  getOrigin,
  getRace,
  ownedClassPowers,
} from '@tormenta20/t20-data'

/** No elective picks at creation — only auto-granted powers are previewed. */
const EMPTY_CHOICES: ReadonlySet<string> = new Set()

/** `+2` / `-1` — signed label for a delta chip. */
export function signed(n: number): string {
  return n >= 0 ? `+${n}` : `${n}`
}

/**
 * Sum the fixed racial attribute bonuses across every selected race. Only the
 * static `attributeBonuses` are folded — floating picks (Humano's +1×3) and
 * subrace-gated bonuses (Suraggel) live in race abilities and are resolved
 * later on the sheet, so they are intentionally excluded here.
 */
export function raceAttributeDeltas(
  raceIds: string[],
): Partial<Record<AttributeKey, number>> {
  const out: Partial<Record<AttributeKey, number>> = {}
  for (const id of raceIds) {
    const bonuses = getRace(id)?.attributeBonuses
    if (!bonuses) continue
    for (const key of ATTRIBUTE_KEYS) {
      const delta = bonuses[key]
      if (delta) out[key] = (out[key] ?? 0) + delta
    }
  }
  return out
}

export type GrantLine = { id: string; name: string; description: string }

/** Attribute deltas + innate abilities a race contributes, for the preview. */
export function raceGrant(raceId: string): {
  name: string
  deltas: [AttributeKey, number][]
  abilities: GrantLine[]
} | null {
  const race = getRace(raceId)
  if (!race) return null
  const deltas = ATTRIBUTE_KEYS.flatMap((k): [AttributeKey, number][] => {
    const v = race.attributeBonuses[k]
    return v ? [[k, v]] : []
  })
  const abilities = race.abilities.map((a) => ({
    id: a.id,
    name: a.name,
    description: a.description,
  }))
  return { name: race.name, deltas, abilities }
}

/**
 * What a class contributes at level 1: the attribute preset it fills, its
 * vitals (PV/PM), and the powers auto-granted at level 1.
 */
export function classGrant(className: string): {
  vitals: { pvInicial: number; pvPerLevel: number; mpPerLevel: number } | null
  powers: GrantLine[]
} {
  const vitals = CLASS_VITALS[className] ?? null
  const powers = ownedClassPowers(className, 1, EMPTY_CHOICES)
    .filter((p) => p.grantedAtLevel === 1)
    .map((p) => ({ id: p.id, name: p.name, description: p.description }))
  return {
    vitals: vitals
      ? {
          pvInicial: vitals.pvInicial,
          pvPerLevel: vitals.pvPerLevel,
          mpPerLevel: vitals.mpPerLevel,
        }
      : null,
    powers,
  }
}

/** Benefit pool + exclusive poder único an origin offers (player picks 2). */
export function originGrant(originId: string): {
  name: string
  benefits: GrantLine[]
  poderUnico: GrantLine | null
} | null {
  const origin = getOrigin(originId)
  if (!origin) return null
  const benefits = origin.benefits.map((b) => ({
    id: b.id,
    name: b.name,
    description: b.description,
  }))
  const poderUnico = origin.poderUnico
    ? {
        id: origin.poderUnico.id,
        name: origin.poderUnico.name,
        description: origin.poderUnico.description,
      }
    : null
  return { name: origin.name, benefits, poderUnico }
}
