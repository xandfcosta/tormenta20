import {
  ATTRIBUTE_ABBR,
  ATTRIBUTE_KEYS,
  type AttributeKey,
  CLASS_VITALS,
  getOrigin,
  getRace,
  ownedClassPowers,
  type Raca,
  racasByTier,
} from '@tormenta20/t20-data'

/** No elective picks at creation — only auto-granted powers are previewed. */
const EMPTY_CHOICES: ReadonlySet<string> = new Set()

/** `+2` / `-1` — signed label for a delta chip. */
export function signed(n: number): string {
  return n >= 0 ? `+${n}` : `${n}`
}

// Race names (backend/RACES_CATALOG strings) → the rules-accurate racas model.
// Names are unified across sources, so a plain name lookup bridges cleanly.
const RACA_BY_NAME: ReadonlyMap<string, Raca> = new Map(
  [...racasByTier('comum'), ...racasByTier('extra')].map((r) => [r.name, r]),
)

/** The rules model for a race name, or undefined for an unknown string. */
export function raceModel(name: string): Raca | undefined {
  return RACA_BY_NAME.get(name)
}

/** A player's per-race attribute-choice state (floating picks / subrace).
 *  `applied` opts a SECONDARY race into the mechanics (GM-negotiated); the
 *  primary race always applies regardless. */
export type RaceChoice = {
  floatingPicks?: AttributeKey[]
  ascendencia?: string
  applied?: boolean
}
export type RaceChoiceState = Record<string, RaceChoice>

/** The races that apply mechanically: the primary (`raceIds[0]`) always, plus
 *  any secondary the player opted into via `applied`. */
export function appliedRaces(
  raceIds: string[],
  choices: RaceChoiceState = {},
): string[] {
  return raceIds.filter((name, i) => i === 0 || choices[name]?.applied === true)
}

/**
 * Resolve one race's attribute deltas given the player's choices. Unlike the
 * strict `resolveAtributoMod` (throws when incomplete), this is live-safe:
 * floating picks are applied partially as they're made, and a floating race's
 * guaranteed penalty (Lefou/Osteon −1) is always folded even before the +1s
 * are placed. Returns an empty map for unknown races or an unset subrace.
 */
export function resolveRaceDeltas(
  name: string,
  choice: RaceChoice = {},
): Partial<Record<AttributeKey, number>> {
  const raca = raceModel(name)
  if (!raca) return {}
  const mod = raca.atributoMod
  if (mod.kind === 'fixed') return { ...mod.mods }
  if (mod.kind === 'floating') {
    const out: Partial<Record<AttributeKey, number>> = {}
    if (mod.penalty) out[mod.penalty.attribute] = mod.penalty.value
    for (const a of choice.floatingPicks ?? []) {
      if (a === mod.exclude) continue
      out[a] = (out[a] ?? 0) + mod.value
    }
    return out
  }
  const variant = choice.ascendencia && mod.variants[choice.ascendencia]
  return variant ? { ...variant } : {}
}

/**
 * Sum resolved racial attribute deltas across every selected race, folding in
 * each race's floating-pick / subrace choices. This is what the wizard bakes
 * into the saved attributes on submit.
 */
export function raceAttributeDeltas(
  raceIds: string[],
  choices: RaceChoiceState = {},
): Partial<Record<AttributeKey, number>> {
  const out: Partial<Record<AttributeKey, number>> = {}
  for (const name of raceIds) {
    const deltas = resolveRaceDeltas(name, choices[name])
    for (const key of ATTRIBUTE_KEYS) {
      const d = deltas[key]
      if (d) out[key] = (out[key] ?? 0) + d
    }
  }
  return out
}

/**
 * Attribute deltas from every race that applies mechanically — the primary plus
 * any opted-in secondary (`appliedRaces`). Homebrew: secondary races are
 * GM-negotiated, applied only when the player toggles them. Matches the backend
 * sheet so the wizard preview equals the saved sheet.
 */
export function appliedRaceDeltas(
  raceIds: string[],
  choices: RaceChoiceState = {},
): Partial<Record<AttributeKey, number>> {
  const out: Partial<Record<AttributeKey, number>> = {}
  for (const name of appliedRaces(raceIds, choices)) {
    const deltas = resolveRaceDeltas(name, choices[name])
    for (const key of ATTRIBUTE_KEYS) {
      const d = deltas[key]
      if (d) out[key] = (out[key] ?? 0) + d
    }
  }
  return out
}

/** True while a race still owes an attribute choice (floating or subrace). */
export function racePending(name: string, choice: RaceChoice = {}): boolean {
  const mod = raceModel(name)?.atributoMod
  if (!mod) return false
  if (mod.kind === 'floating') {
    const placed = (choice.floatingPicks ?? []).filter(
      (a) => a !== mod.exclude,
    ).length
    return placed !== mod.count
  }
  if (mod.kind === 'subraca-gated') {
    return !(choice.ascendencia && mod.variants[choice.ascendencia])
  }
  return false
}

/** Only races that apply mechanically (primary + opted-in secondaries) can have
 *  a pending attribute choice. */
export function anyRacePending(
  raceIds: string[],
  choices: RaceChoiceState = {},
): boolean {
  return appliedRaces(raceIds, choices).some((name) =>
    racePending(name, choices[name]),
  )
}

/** Compact signature for a race tile: `+2 CON`, `+1×3`, or `2 ascend.`. */
export function raceSignature(name: string): string {
  const mod = raceModel(name)?.atributoMod
  if (!mod) return ''
  if (mod.kind === 'floating') return `+${mod.value}×${mod.count}`
  if (mod.kind === 'subraca-gated') return '2 ascend.'
  const top = ATTRIBUTE_KEYS.map(
    (k): [AttributeKey, number] => [k, mod.mods[k] ?? 0],
  )
    .filter(([, v]) => v > 0)
    .sort((a, b) => b[1] - a[1])[0]
  return top ? `+${top[1]} ${ATTRIBUTE_ABBR[top[0]]}` : ''
}

/** Shape the choice-capture UI needs to render controls for a race. */
export type RaceChoiceMeta =
  | { kind: 'none' }
  | {
      kind: 'floating'
      count: number
      value: number
      exclude?: AttributeKey
      penalty?: { attribute: AttributeKey; value: number }
    }
  | { kind: 'subrace'; options: string[] }

export function raceChoiceMeta(name: string): RaceChoiceMeta {
  const mod = raceModel(name)?.atributoMod
  if (!mod) return { kind: 'none' }
  if (mod.kind === 'floating') {
    return {
      kind: 'floating',
      count: mod.count,
      value: mod.value,
      exclude: mod.exclude,
      penalty: mod.penalty,
    }
  }
  if (mod.kind === 'subraca-gated') {
    return { kind: 'subrace', options: Object.keys(mod.variants) }
  }
  return { kind: 'none' }
}

/** Split the available race names into Comuns / Outras (tier), preserving
 *  only names the backend actually offers. */
export function racesByTier(available: string[]): {
  comuns: string[]
  extras: string[]
} {
  const offered = new Set(available)
  const names = (tier: 'comum' | 'extra') =>
    racasByTier(tier)
      .map((r) => r.name)
      .filter((n) => offered.has(n))
  return { comuns: names('comum'), extras: names('extra') }
}

export type ClassTile = {
  className: string
  pvInicial: number
  mpPerLevel: number
}

/** Class name + its level-1 vitals, for the primary-class tile grid. */
export function classTiles(available: string[]): ClassTile[] {
  return available.map((className) => {
    const v = CLASS_VITALS[className]
    return {
      className,
      pvInicial: v?.pvInicial ?? 0,
      mpPerLevel: v?.mpPerLevel ?? 0,
    }
  })
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
 * What a class contributes through a given level: its vitals (PV/PM) and every
 * power auto-granted at level ≤ `level` (each labelled with the level it's
 * gained). Elective powers the player chooses are not included here.
 */
export function classGrant(
  className: string,
  level = 1,
): {
  vitals: { pvInicial: number; pvPerLevel: number; mpPerLevel: number } | null
  powers: GrantLine[]
} {
  const vitals = CLASS_VITALS[className] ?? null
  const powers = ownedClassPowers(className, level, EMPTY_CHOICES)
    .filter((p) => p.grantedAtLevel !== undefined)
    .sort((a, b) => (a.grantedAtLevel ?? 0) - (b.grantedAtLevel ?? 0))
    .map((p) => ({
      id: p.id,
      name: `Nv ${p.grantedAtLevel} · ${p.name}`,
      description: p.description,
    }))
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
