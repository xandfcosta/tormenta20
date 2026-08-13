import { classPowerModifiersIn, ownedClassPowersIn } from '@/shared/rules/abilities-classes-ownership'
import { devotoOptionsIn } from '@/shared/rules/abilities-devoto-options'
import type { ClassPower, Deus, GeneralPower, GrantedPower, OriginBenefit, OriginDefinition, RaceDefinition } from '@/shared/api/catalog-types'
import type { ClassChoiceSelections } from '@/shared/api/catalog-types'
import type { Modifier } from '@/shared/api/item-types'

/**
 * Front-owned cache for the abilities cluster (races, origins, class powers,
 * general powers, deuses, granted powers) with SYNC accessors mirroring the
 * t20-data lookups (`getRace`, `getOrigin`, `getClassPower`…). Lets the core
 * derive pipeline (derived.ts) and the sheet/build/wizard consumers read
 * abilities data WITHOUT a build-time `import` of the ~149KB t20-abilities
 * chunk — it's fetched from `GET /catalog/*` and cached instead
 * (project_front_decouple_catalog B.3).
 *
 * Same contract as `catalog-cache.ts`: correctness rests on the cache being
 * primed (via the root loader's `ensureCatalogs`) BEFORE any consumer renders.
 * Catalogs are static + cached-forever, so by render time the primed value is
 * effectively constant and derived.ts needs no reactivity. Only the two
 * catalog-parametrized RULES (`ownedClassPowersIn`, `devotoOptionsIn`) are
 * imported from t20-data — both data-free, so they don't re-anchor the chunk.
 * The pure entity-modifier builders (`raceModifiers`/`originModifiers`) and the
 * slot/caminho tables stay imported straight from `@tormenta20/t20-data` by
 * consumers — they already tree-shake (data-free modules).
 */
let raceList: readonly RaceDefinition[] = []
let racesById: ReadonlyMap<string, RaceDefinition> = new Map()
let originList: readonly OriginDefinition[] = []
let originsById: ReadonlyMap<string, OriginDefinition> = new Map()
let classPowerList: readonly ClassPower[] = []
let classPowersById: ReadonlyMap<string, ClassPower> = new Map()
let generalPowerList: readonly GeneralPower[] = []
let generalPowersById: ReadonlyMap<string, GeneralPower> = new Map()
let deusList: readonly Deus[] = []
let grantedPowerList: readonly GrantedPower[] = []
let primed = false

/** Everything the abilities cache needs, as fetched from `/catalog/*`. */
export type AbilitiesCatalogs = {
  races: readonly RaceDefinition[]
  origins: readonly OriginDefinition[]
  classPowers: readonly ClassPower[]
  generalPowers: readonly GeneralPower[]
  deuses: readonly Deus[]
  grantedPowers: readonly GrantedPower[]
}

/** Prime the abilities cache from the fetched catalogs. Idempotent (re-priming
 *  with the same static data is a no-op in effect). Called by `ensureCatalogs`
 *  and by the test setup. */
export function primeAbilities(catalogs: AbilitiesCatalogs): void {
  raceList = catalogs.races
  racesById = new Map(catalogs.races.map((r) => [r.id, r]))
  originList = catalogs.origins
  originsById = new Map(catalogs.origins.map((o) => [o.id, o]))
  classPowerList = catalogs.classPowers
  classPowersById = new Map(catalogs.classPowers.map((p) => [p.id, p]))
  generalPowerList = catalogs.generalPowers
  generalPowersById = new Map(catalogs.generalPowers.map((p) => [p.id, p]))
  deusList = catalogs.deuses
  grantedPowerList = catalogs.grantedPowers
  primed = true
}

/** True once the abilities cache has been primed — for a render-time gate. */
export function isAbilitiesPrimed(): boolean {
  return primed
}

// --- Lookups (cache-backed mirrors of the t20-data catalog hub) ---

export function getRace(id: string): RaceDefinition | undefined {
  return racesById.get(id)
}

export function getOrigin(id: string): OriginDefinition | undefined {
  return originsById.get(id)
}

export function getClassPower(id: string): ClassPower | undefined {
  return classPowersById.get(id)
}

export function getGeneralPower(id: string): GeneralPower | undefined {
  return generalPowersById.get(id)
}

/** Find an origin benefit by id across all origens, including poderes únicos
 *  (stored alongside the regular benefits list). */
export function getOriginBenefit(benefitId: string): OriginBenefit | undefined {
  for (const origin of originList) {
    const found = origin.benefits.find((b) => b.id === benefitId)
    if (found) return found
    if (origin.poderUnico.id === benefitId) return origin.poderUnico
  }
  return undefined
}

/** True when any of the given races (by id/name) is Tormenta-touched (Lefou),
 *  unlocking the poderes da Tormenta pool. */
export function racesGrantTormenta(raceNames: readonly string[]): boolean {
  return raceNames.some((n) => getRace(n)?.grantsTormentaPowers === true)
}

export function classPowersFor(className: string): ClassPower[] {
  return classPowerList.filter((p) => p.className === className)
}

/** Every general power a character may take by substituting a class-power slot
 *  (excludes `tormenta` powers, which require Tormenta exposure). Mirrors the
 *  t20-data `allGeneralPowers`. */
export function allGeneralPowers(): GeneralPower[] {
  return generalPowerList.filter((p) => p.kind !== 'tormenta')
}

/** Class powers the character owns for a class + level + chosen ids (+ caminho
 *  /devoto picks). Runs the shared ownership rule against the cached catalog. */
export function ownedClassPowers(
  className: string,
  classLevel: number,
  chosenIds: ReadonlySet<string>,
  choices?: ClassChoiceSelections,
): ClassPower[] {
  return ownedClassPowersIn(
    classPowerList,
    className,
    classLevel,
    chosenIds,
    choices,
  )
}

/** Deus options for a class's devoto picker (or null when it has none). Runs
 *  the shared per-class whitelist against the cached DEUSES. */
export function devotoOptionsFor(className: string): Deus[] | null {
  return devotoOptionsIn(deusList, className)
}

/** Modifiers from the class powers owned at a level. Runs the shared ownership
 *  rule against the cached catalog — the cache-backed twin of t20-data's
 *  `classPowerModifiers`, used by the front vital-grants resolver. */
export function classPowerModifiers(
  className: string,
  classLevel: number,
  chosenIds: ReadonlySet<string>,
  choices?: ClassChoiceSelections,
): Modifier[] {
  return classPowerModifiersIn(
    classPowerList,
    className,
    classLevel,
    chosenIds,
    choices,
  )
}

/** Granted (god) power by exact NAME — mirrors t20-data `grantedPowerByName`
 *  (the catalog keys god powers by name to match the book's prose). */
export function grantedPowerByName(name: string): GrantedPower | undefined {
  return grantedPowerList.find((p) => p.name === name)
}

/** First race (by abilities name) in `raceNames` that owns Deformidade (Lefou,
 *  p23). Cache-backed twin of t20-data `raceWithDeformidade`. */
export function raceWithDeformidade(
  raceNames: readonly string[],
): string | undefined {
  const owners = new Set(
    raceList.filter((r) => r.hasDeformidade === true).map((r) => r.name),
  )
  return raceNames.find((n) => owners.has(n))
}

// --- Raw list accessors (were the t20-data data-const exports). Read them
// inside components/functions that run AFTER the gate, never at module
// top-level (which evaluates before priming). ---

export function classPowerCatalog(): readonly ClassPower[] {
  return classPowerList
}

export function generalPowerCatalog(): readonly GeneralPower[] {
  return generalPowerList
}

export function grantedPowers(): readonly GrantedPower[] {
  return grantedPowerList
}

export function deuses(): readonly Deus[] {
  return deusList
}

/** The full race/origin lists — for callers that iterate the whole catalog. */
export function raceCatalog(): readonly RaceDefinition[] {
  return raceList
}

export function originCatalog(): readonly OriginDefinition[] {
  return originList
}
