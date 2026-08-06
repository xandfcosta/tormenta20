import type { QueryClient } from '@tanstack/react-query'
import { primeAbilities } from '@/shared/lib/abilities-cache'
import { primeActivations } from '@/shared/lib/activation-cache'
import { primeItemCatalog } from '@/shared/lib/catalog-cache'
import { primeDivinePowers } from '@/shared/lib/divine-powers-cache'
import { ensureEngine, primeEngineCatalogs } from '@/shared/lib/engine-wasm'
import { primeRacas } from '@/shared/lib/racas-cache'
import { primeRulesCatalogs } from '@/shared/lib/rules-catalog-cache'
import { primeSpellCatalog } from '@/shared/lib/spell-cache'
import {
  activationsCatalogQueryOptions,
  classPowersCatalogQueryOptions,
  conditionsCatalogQueryOptions,
  deusesCatalogQueryOptions,
  divinePowersCatalogQueryOptions,
  generalPowersCatalogQueryOptions,
  grantedPowersCatalogQueryOptions,
  itemCatalogQueryOptions,
  origensCatalogQueryOptions,
  originsCatalogQueryOptions,
  racasCatalogQueryOptions,
  raceDefsCatalogQueryOptions,
  spellCatalogQueryOptions,
  tormentaPowersCatalogQueryOptions,
} from './queries'

/**
 * Route-loader gate: fetch (cached-forever) the catalogs the sheet/wizard need
 * and prime the sync caches BEFORE the route renders, so the `catalog-cache` /
 * `abilities-cache` sync accessors are warm on the first derive. Called from
 * `__root.tsx` beforeLoad (app-wide, once, after login).
 *
 * All catalogs fetch in parallel — they're independent, static, and cached
 * forever, so the gate cost is one round-trip's latency, not the sum.
 *
 * Grows one line per catalog as more data migrates off the build-time t20-data
 * import (items — B.2; races/origins/class-powers/general-powers/deuses/
 * granted-powers — B.3).
 */
export async function ensureCatalogs(qc: QueryClient): Promise<void> {
  const [
    items,
    races,
    origins,
    classPowers,
    generalPowers,
    deuses,
    granted,
    spells,
    racas,
    origens,
    conditions,
    tormentaPowers,
    divinePowers,
    activations,
  ] = await Promise.all([
    qc.ensureQueryData(itemCatalogQueryOptions),
    qc.ensureQueryData(raceDefsCatalogQueryOptions),
    qc.ensureQueryData(originsCatalogQueryOptions),
    qc.ensureQueryData(classPowersCatalogQueryOptions),
    qc.ensureQueryData(generalPowersCatalogQueryOptions),
    qc.ensureQueryData(deusesCatalogQueryOptions),
    qc.ensureQueryData(grantedPowersCatalogQueryOptions),
    qc.ensureQueryData(spellCatalogQueryOptions),
    qc.ensureQueryData(racasCatalogQueryOptions),
    qc.ensureQueryData(origensCatalogQueryOptions),
    qc.ensureQueryData(conditionsCatalogQueryOptions),
    qc.ensureQueryData(tormentaPowersCatalogQueryOptions),
    qc.ensureQueryData(divinePowersCatalogQueryOptions),
    qc.ensureQueryData(activationsCatalogQueryOptions),
  ])
  primeItemCatalog(items)
  primeAbilities({
    races,
    origins,
    classPowers,
    generalPowers,
    deuses,
    grantedPowers: granted,
  })
  primeSpellCatalog(spells)
  primeRacas(racas, origens)
  primeRulesCatalogs(conditions, tormentaPowers)
  primeDivinePowers(divinePowers)
  primeActivations(activations)
}

/**
 * Warm the Go/WASM engine and prime its catalogs with the SAME data
 * `ensureCatalogs` uses — so the sheet derive is sync-ready on the first render.
 * Loads the wasm in parallel with the (cached) catalog fetches, then primes.
 * Runs alongside `ensureCatalogs` from the root beforeLoad.
 *
 * The engine is now LOAD-BEARING (task #8): the front bundle ships no TS derive,
 * so `resolveEffects` throws if the engine isn't primed. We still don't reject the
 * whole app gate on a wasm failure — non-sheet routes stay usable and the failure
 * surfaces (loudly) as a sheet-scoped error boundary rather than a blank app.
 */
export async function ensureEngineCatalogs(qc: QueryClient): Promise<void> {
  try {
    const [payload] = await Promise.all([buildEnginePayload(qc), ensureEngine()])
    primeEngineCatalogs(payload)
  } catch (err) {
    console.error('engine-wasm warm failed — sheet pages will error until it loads:', err)
  }
}

/** Assemble the engine-catalog JSON from the (cached) fetched catalogs — the
 *  exact shape `engine.PrimeEngineCatalogs` / the parity dump expect. */
async function buildEnginePayload(qc: QueryClient): Promise<string> {
  const [items, races, origins, classPowers, generalPowers, racas, tormentaPowers] =
    await Promise.all([
      qc.ensureQueryData(itemCatalogQueryOptions),
      qc.ensureQueryData(raceDefsCatalogQueryOptions),
      qc.ensureQueryData(originsCatalogQueryOptions),
      qc.ensureQueryData(classPowersCatalogQueryOptions),
      qc.ensureQueryData(generalPowersCatalogQueryOptions),
      qc.ensureQueryData(racasCatalogQueryOptions),
      qc.ensureQueryData(tormentaPowersCatalogQueryOptions),
    ])
  return JSON.stringify({
    items,
    races,
    origins,
    classPowers,
    generalPowers,
    racas,
    tormentaPowerIds: Object.keys(tormentaPowers),
  })
}
