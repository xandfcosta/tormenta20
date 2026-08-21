import { CLASS_VITALS } from '@/shared/rules/class-vitals'
import { carismaLossFromPowers } from '@/shared/rules/tormenta-carisma'
import type { Raca, TormentaPowerId } from '@/shared/api/catalog-types'
import { ATTRIBUTE_ABBR, ATTRIBUTE_KEYS, type AttributeKey } from '@/shared/api/attribute-keys'
import {
  getOrigin,
  getOriginBenefit,
  getRace,
  ownedClassPowers,
  raceWithDeformidade,
} from '@/shared/lib/abilities-cache'
import { racasByTier, racasList } from '@/shared/lib/racas-cache'
import { raceAttributeMeta } from '@/shared/lib/race-attribute-meta'
import { tormentaPowersRecord } from '@/shared/lib/rules-catalog-cache'

/** No elective picks at creation — only auto-granted powers are previewed. */
const EMPTY_CHOICES: ReadonlySet<string> = new Set()

/** `+2` / `-1` — signed label for a delta chip. */
export function signed(n: number): string {
  return n >= 0 ? `+${n}` : `${n}`
}

/** The rules model for a race name, or undefined for an unknown string. Reads
 *  the primed racas cache (loader-gate warm; a module-level Map would evaluate
 *  before priming). Names are unified across sources so a plain name bridges. */
export function raceModel(name: string): Raca | undefined {
  return racasList().find((r) => r.name === name)
}

/** A player's per-race attribute-choice state (floating picks / subrace).
 *  `applied` opts a SECONDARY race into the mechanics (GM-negotiated); the
 *  primary race always applies regardless. `deformidade` captures the Lefou
 *  p23 choice: ≤2 perícias com +2, uma trocável por um poder da Tormenta. */
export type RaceChoice = {
  floatingPicks?: AttributeKey[]
  ascendencia?: string
  applied?: boolean
  deformidade?: DeformidadeDraft
}

export type DeformidadeDraft = {
  pericias: string[]
  tormentaPower?: string
}

/**
 * Deformidade draft → submit payload: drops empty slots ('' placeholders from
 * the swap toggle) and returns undefined when nothing was chosen or the race
 * doesn't own the ability, so stale drafts never persist.
 */
export function deformidadePayload(
  raceName: string,
  choice: RaceChoice | undefined,
): DeformidadeDraft | undefined {
  const draft = choice?.deformidade
  if (!draft || !raceWithDeformidade([raceName])) return undefined
  const pericias = draft.pericias.filter(Boolean)
  const tormentaPower = draft.tormentaPower || undefined
  if (pericias.length === 0 && !tormentaPower) return undefined
  return { pericias, tormentaPower }
}

/**
 * The poder da Tormenta held via a Deformidade swap in the wizard draft —
 * from any APPLIED race that owns the ability. Used by the Poderes step to
 * block re-taking it and to satisfy `requiresPower` prereqs (it's owned).
 */
export function draftDeformidadeHeldPower(
  raceIds: string[],
  choices: RaceChoiceState = {},
): string | undefined {
  for (const name of appliedRaces(raceIds, choices)) {
    const payload = deformidadePayload(name, choices[name])
    if (payload?.tormentaPower) return payload.tormentaPower
  }
  return undefined
}

/**
 * EXTRA Carisma delta from poderes da Tormenta beyond the Deformidade swap —
 * pool picks (classPowers) and free-pick origem benefits. The race deltas
 * already carry the swap's −1 (`resolveRaceDeltas`), and the p136 loss
 * escalates over the TOTAL count, so this returns only the difference. Keeps
 * the wizard preview equal to the saved sheet. Always ≤ 0.
 */
export function draftTormentaCarismaExtra(
  raceIds: string[],
  choices: RaceChoiceState,
  classPowers: string[],
  powerChoices: Record<string, string[]>,
  originChoices: string[],
): number {
  const held = draftDeformidadeHeldPower(raceIds, choices)
  const originPicked = originChoices.flatMap((benefitId) =>
    getOriginBenefit(benefitId)?.powerPick ? (powerChoices[benefitId] ?? []) : [],
  )
  const picked = [...new Set([...classPowers, ...originPicked])].filter(
    (id) => id in tormentaPowersRecord(),
  )
  const count = picked.length + (held && !picked.includes(held) ? 1 : 0)
  if (count === 0) return 0
  const swapLoss = held ? carismaLossFromPowers(1) : 0
  return swapLoss - carismaLossFromPowers(count)
}

/**
 * The Deformidade picks as one display line for the Resumo
 * ("Deformidade: +2 Furtividade · poder da Tormenta: Dentes Afiados (−1 CAR)"),
 * or null when the race lacks the ability / nothing was chosen.
 */
export function deformidadeSummary(
  raceName: string,
  choice: RaceChoice | undefined,
): string | null {
  const payload = deformidadePayload(raceName, choice)
  if (!payload) return null
  const parts = payload.pericias.map((p) => `+2 ${p}`)
  if (payload.tormentaPower) {
    const power =
      tormentaPowersRecord()[payload.tormentaPower as TormentaPowerId]
    parts.push(
      `poder da Tormenta: ${power?.name ?? payload.tormentaPower} (−1 CAR)`,
    )
  }
  return `Deformidade: ${parts.join(' · ')}`
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
  const out = attributeModDeltas(name, choice)
  // Deformidade swap: um poder da Tormenta real perde Carisma (p136); os
  // bônus de perícia não (p23). Mostrado no preview junto aos mods raciais.
  if (choice.deformidade?.tormentaPower && raceWithDeformidade([name])) {
    out.charisma = (out.charisma ?? 0) - carismaLossFromPowers(1)
  }
  return out
}

function attributeModDeltas(
  name: string,
  choice: RaceChoice,
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
  // Delegado desde a ALE-169: a FICHA precisa da mesma resposta e não pode
  // importar desta feature, então a regra mudou para `shared/lib`. Este nome
  // fica porque a forja inteira o usa.
  return raceAttributeMeta(name)
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

export type GrantLine = {
  id: string
  name: string
  description: string
  /** Free-pick origem benefit — pool the player picks the concrete power from. */
  powerPick?: 'combate' | 'tormenta'
}

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
    powerPick: b.powerPick,
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
