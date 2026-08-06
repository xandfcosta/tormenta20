import type { QueryClient } from '@tanstack/react-query'
import { primeAbilities } from '@/shared/lib/abilities-cache'
import { primeItemCatalog } from '@/shared/lib/catalog-cache'
import {
  classPowersCatalogQueryOptions,
  deusesCatalogQueryOptions,
  generalPowersCatalogQueryOptions,
  grantedPowersCatalogQueryOptions,
  itemCatalogQueryOptions,
  originsCatalogQueryOptions,
  raceDefsCatalogQueryOptions,
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
  const [items, races, origins, classPowers, generalPowers, deuses, granted] =
    await Promise.all([
      qc.ensureQueryData(itemCatalogQueryOptions),
      qc.ensureQueryData(raceDefsCatalogQueryOptions),
      qc.ensureQueryData(originsCatalogQueryOptions),
      qc.ensureQueryData(classPowersCatalogQueryOptions),
      qc.ensureQueryData(generalPowersCatalogQueryOptions),
      qc.ensureQueryData(deusesCatalogQueryOptions),
      qc.ensureQueryData(grantedPowersCatalogQueryOptions),
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
}
