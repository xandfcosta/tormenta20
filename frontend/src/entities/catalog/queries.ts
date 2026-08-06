import { queryOptions } from '@tanstack/react-query'
import { api } from '@/shared/api/api'

/**
 * Static rulebook catalogs, fetched once and cached forever (they only change
 * on a backend deploy). Replaces the old build-time `import { SPELL_CATALOG }`
 * from `@tormenta20/t20-data`, which bundled ~175KB of spell data into the JS.
 */
export const spellCatalogQueryOptions = queryOptions({
  queryKey: ['catalog', 'spells'] as const,
  queryFn: api.catalog.spells,
  staleTime: Number.POSITIVE_INFINITY,
})

export const bestiaryCatalogQueryOptions = queryOptions({
  queryKey: ['catalog', 'bestiary'] as const,
  queryFn: api.catalog.bestiary,
  staleTime: Number.POSITIVE_INFINITY,
})

export const itemCatalogQueryOptions = queryOptions({
  queryKey: ['catalog', 'items'] as const,
  queryFn: api.catalog.items,
  staleTime: Number.POSITIVE_INFINITY,
})
