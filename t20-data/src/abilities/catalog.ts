import type { Modifier } from '../items/types'
import { CLASS_POWERS_CATALOG } from './classes/index'
import { ORIGINS_CATALOG, originModifiers } from './origins'
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
 * entries in `./origins`, class powers under `./classes/`.
 */
export { RACES_CATALOG } from './races'
export { ORIGINS_CATALOG, originModifiers } from './origins'
export {
  CLASS_POWERS_CATALOG,
  CLASS_POWER_SLOTS,
  classPowerModifiers,
  ownedClassPowers,
  slotsForClassLevel,
  unlockedKinds,
} from './classes/index'
export type { ClassPowerSlot } from './classes/index'

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

/**
 * Build the list of Modifier[] contributed by a race definition + variant
 * choices for the player. Race attribute bonuses are emitted as `attribute`
 * target modifiers so they flow through the standard engine. Abilities with
 * variants only contribute modifiers from the chosen variant id.
 */
export function raceModifiers(
  race: RaceDefinition,
  variantChoices: ReadonlySet<string>,
): Modifier[] {
  const out: Modifier[] = []
  for (const [attr, amount] of Object.entries(race.attributeBonuses)) {
    if (typeof amount !== 'number' || amount === 0) continue
    out.push({
      target: { k: 'attribute', name: attr as never },
      amount,
      bonusType: 'untyped',
      note: race.name,
    })
  }
  for (const ability of race.abilities) {
    if (ability.modifiers) out.push(...ability.modifiers)
    if (ability.variants) {
      const chosen = ability.variants.find((v) => variantChoices.has(v.id))
      if (chosen?.modifiers) out.push(...chosen.modifiers)
    }
  }
  return out
}
