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

// Abilities cluster (B.3) — each fetched once, cached forever, primed into the
// abilities-cache by `ensureCatalogs` before the sheet/build/wizard renders.
export const raceDefsCatalogQueryOptions = queryOptions({
  queryKey: ['catalog', 'race-defs'] as const,
  queryFn: api.catalog.raceDefs,
  staleTime: Number.POSITIVE_INFINITY,
})

export const originsCatalogQueryOptions = queryOptions({
  queryKey: ['catalog', 'origins'] as const,
  queryFn: api.catalog.origins,
  staleTime: Number.POSITIVE_INFINITY,
})

export const classPowersCatalogQueryOptions = queryOptions({
  queryKey: ['catalog', 'class-powers'] as const,
  queryFn: api.catalog.classPowers,
  staleTime: Number.POSITIVE_INFINITY,
})

export const generalPowersCatalogQueryOptions = queryOptions({
  queryKey: ['catalog', 'general-powers'] as const,
  queryFn: api.catalog.generalPowers,
  staleTime: Number.POSITIVE_INFINITY,
})

export const deusesCatalogQueryOptions = queryOptions({
  queryKey: ['catalog', 'deuses'] as const,
  queryFn: api.catalog.deuses,
  staleTime: Number.POSITIVE_INFINITY,
})

export const grantedPowersCatalogQueryOptions = queryOptions({
  queryKey: ['catalog', 'granted-powers'] as const,
  queryFn: api.catalog.grantedPowers,
  staleTime: Number.POSITIVE_INFINITY,
})
