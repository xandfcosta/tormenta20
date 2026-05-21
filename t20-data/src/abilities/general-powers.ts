import type { Modifier } from '../items/types'
import type { Prerequisite } from './types'

/**
 * Power "kinds" mirror the PDF's four general-power pools — Combate, Destino,
 * Magia, Tormenta — plus per-class pools (`barbaro`, `bardo`, etc.). When a
 * class grants a power slot at a given level, the slot's `kinds` list tells
 * the picker which pools the player may draw from.
 */
export type PowerKind =
  | 'combate'
  | 'destino'
  | 'magia'
  | 'tormenta'
  | 'arcanista'
  | 'barbaro'
  | 'bardo'
  | 'bucaneiro'
  | 'cacador'
  | 'cavaleiro'
  | 'clerigo'
  | 'druida'
  | 'guerreiro'
  | 'inventor'
  | 'ladino'
  | 'lutador'
  | 'nobre'
  | 'paladino'

/**
 * General powers (PDF Cap 4 — power pools shared across classes). Stored
 * separately from class-specific electives because multiple classes draw from
 * the same pool. Same shape as ClassPower minus the `className` field.
 */
export type GeneralPower = {
  id: string
  kind: PowerKind
  name: string
  description: string
  minLevel?: number
  prerequisites?: Prerequisite[]
  modifiers?: Modifier[]
}

/**
 * General Powers (PDF Cap 2 — "Poderes Gerais"). Empty until Cap 2 pages are
 * audited. Per PDF p33: players may substitute a class power slot for a
 * general power. Pools: Combate, Destino, Magia, Tormenta — plus a handful
 * of misc. Class-specific lists live under `./classes/`; this catalog is
 * separate so the same general power isn't duplicated 14 times.
 */
export const GENERAL_POWERS_CATALOG: GeneralPower[] = []

const generalPowersById = new Map<string, GeneralPower>(
  GENERAL_POWERS_CATALOG.map((p) => [p.id, p]),
)

export function getGeneralPower(id: string): GeneralPower | undefined {
  return generalPowersById.get(id)
}

export function generalPowersByKinds(
  allowedKinds: ReadonlyArray<PowerKind>,
): GeneralPower[] {
  const set = new Set(allowedKinds)
  return GENERAL_POWERS_CATALOG.filter((p) => set.has(p.kind))
}
