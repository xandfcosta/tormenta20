import type { QueryClient } from '@tanstack/react-query'
import { primeItemCatalog } from './catalog-cache'
import { itemCatalogQueryOptions } from './queries'

/**
 * Route-loader gate: fetch (cached-forever) the catalogs the sheet/wizard need
 * and prime the sync cache BEFORE the route renders, so `catalog-cache`'s sync
 * accessors are warm on the first derive. Call from the character route loaders
 * (`characters/$id`, `characters/new`) alongside the character/options fetch.
 *
 * Grows one line per catalog as items → races → origins → spells migrate off
 * the build-time t20-data import.
 *
 * @example loader: async ({ context, params }) => {
 *   await ensureCatalogs(context.queryClient)
 *   return context.queryClient.ensureQueryData(characterQueryOptions(params.id))
 * }
 */
export async function ensureCatalogs(qc: QueryClient): Promise<void> {
  const items = await qc.ensureQueryData(itemCatalogQueryOptions)
  primeItemCatalog(items)
}
