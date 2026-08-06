import { CLASS_POWERS_CATALOG } from './classes/index'
import { ORIGINS_CATALOG } from './origins'
import { RACES_CATALOG } from './races'
import type {
  ClassPower,
  OriginBenefit,
  OriginDefinition,
  RaceAbility,
  RaceDefinition,
} from './types'

/**
 * Catalog assembly + lookup helpers. Race entries live in `./races`, origin
 * entries in `./origins`, class powers under `./classes/`. The pure modifier
 * builders `raceModifiers`/`originModifiers` live in `./race-logic` /
 * `./origin-logic` (data-free) so the front can call them off its cache; they
 * are re-exported here to keep this hub's public surface unchanged.
 */
export { RACES_CATALOG } from './races'
export { raceModifiers } from './race-logic'
export { ORIGINS_CATALOG, originModifiers } from './origins'
export {
  CLASS_POWERS_CATALOG,
  CLASS_POWER_SLOTS,
  classPowerModifiers,
  classPowerModifiersIn,
  ownedClassPowers,
  ownedClassPowersIn,
  slotsForClassLevel,
  unlockedKinds,
} from './classes/index'
export type { ClassChoiceSelections, ClassPowerSlot } from './classes/index'

const racesById = new Map<string, RaceDefinition>(
  RACES_CATALOG.map((r) => [r.id, r]),
)
const originsById = new Map<string, OriginDefinition>(
  ORIGINS_CATALOG.map((o) => [o.id, o]),
)
const classPowersById = new Map<string, ClassPower>(
  CLASS_POWERS_CATALOG.map((p) => [p.id, p]),
)

export function getRace(id: string): RaceDefinition | undefined {
  return racesById.get(id)
}

/** True when any of the given races (by id/name) is Tormenta-touched (Lefou),
 *  unlocking the poderes da Tormenta pool. */
export function racesGrantTormenta(raceNames: readonly string[]): boolean {
  return raceNames.some((n) => getRace(n)?.grantsTormentaPowers === true)
}

export function getOrigin(id: string): OriginDefinition | undefined {
  return originsById.get(id)
}

export function getClassPower(id: string): ClassPower | undefined {
  return classPowersById.get(id)
}

export function classPowersFor(className: string): ClassPower[] {
  return CLASS_POWERS_CATALOG.filter((p) => p.className === className)
}

export function getRaceAbility(abilityId: string): RaceAbility | undefined {
  for (const race of RACES_CATALOG) {
    const found = race.abilities.find((a) => a.id === abilityId)
    if (found) return found
  }
  return undefined
}

/**
 * Find an origin benefit by id across all origens. Includes poderes únicos
 * since they're stored alongside the regular benefits list.
 */
export function getOriginBenefit(benefitId: string): OriginBenefit | undefined {
  for (const origin of ORIGINS_CATALOG) {
    const found = origin.benefits.find((b) => b.id === benefitId)
    if (found) return found
    if (origin.poderUnico.id === benefitId) return origin.poderUnico
  }
  return undefined
}
