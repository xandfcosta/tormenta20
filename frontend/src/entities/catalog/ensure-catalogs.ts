import type { QueryClient } from '@tanstack/react-query'
import { primeAbilities } from '@/shared/lib/abilities-cache'
import { primeActivations } from '@/shared/lib/activation-cache'
import { primeItemCatalog } from '@/shared/lib/catalog-cache'
import { primeDivinePowers } from '@/shared/lib/divine-powers-cache'
import { primeRacas } from '@/shared/lib/racas-cache'
import { primeRulesCatalogs } from '@/shared/lib/rules-catalog-cache'
import { primeSpellCatalog } from '@/shared/lib/spell-cache'
import {
  classPowersCatalogQueryOptions,
  conditionsCatalogQueryOptions,
  deusesCatalogQueryOptions,
  divinePowersCatalogQueryOptions,
  generalPowersCatalogQueryOptions,
  grantedPowersCatalogQueryOptions,
  activationsCatalogQueryOptions,
  itemCatalogQueryOptions,
  origensCatalogQueryOptions,
  originsCatalogQueryOptions,
  raceDefsCatalogQueryOptions,
  racasCatalogQueryOptions,
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
