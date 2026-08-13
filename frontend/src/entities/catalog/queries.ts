import { queryOptions } from '@tanstack/solid-query'
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


export const itemCatalogQueryOptions = queryOptions({
  queryKey: ['catalog', 'items'] as const,
  queryFn: api.catalog.items,
  staleTime: Number.POSITIVE_INFINITY,
})

/** The bestiary. Unlike the abilities cluster it is NOT primed into a module
 *  cache: only the Mesa do Mestre and the in-session monster add read it, and
 *  they can wait for a query like any other screen. */
export const bestiaryCatalogQueryOptions = queryOptions({
  queryKey: ['catalog', 'bestiary'] as const,
  queryFn: api.catalog.bestiary,
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

export const racasCatalogQueryOptions = queryOptions({
  queryKey: ['catalog', 'races'] as const,
  queryFn: api.catalog.races,
  staleTime: Number.POSITIVE_INFINITY,
})

export const origensCatalogQueryOptions = queryOptions({
  queryKey: ['catalog', 'origens'] as const,
  queryFn: api.catalog.origens,
  staleTime: Number.POSITIVE_INFINITY,
})

export const conditionsCatalogQueryOptions = queryOptions({
  queryKey: ['catalog', 'conditions'] as const,
  queryFn: api.catalog.conditions,
  staleTime: Number.POSITIVE_INFINITY,
})

export const tormentaPowersCatalogQueryOptions = queryOptions({
  queryKey: ['catalog', 'tormenta-powers'] as const,
  queryFn: api.catalog.tormentaPowers,
  staleTime: Number.POSITIVE_INFINITY,
})

export const divinePowersCatalogQueryOptions = queryOptions({
  queryKey: ['catalog', 'divine-powers'] as const,
  queryFn: api.catalog.divinePowers,
  staleTime: Number.POSITIVE_INFINITY,
})

export const activationsCatalogQueryOptions = queryOptions({
  queryKey: ['catalog', 'activations'] as const,
  queryFn: api.catalog.activations,
  staleTime: Number.POSITIVE_INFINITY,
})

// Tabelas de livro autoradas no servidor (ALE-102) — eram as últimas que o front
// importava em tempo de build. Buscadas uma vez e primadas como as outras.
export const classExpertisesCatalogQueryOptions = queryOptions({
  queryKey: ['catalog', 'class-expertises'] as const,
  queryFn: api.catalog.classExpertises,
  staleTime: Number.POSITIVE_INFINITY,
})

export const devotoTermsCatalogQueryOptions = queryOptions({
  queryKey: ['catalog', 'devoto-terms'] as const,
  queryFn: api.catalog.devotoTerms,
  staleTime: Number.POSITIVE_INFINITY,
})

export const gmTablesCatalogQueryOptions = queryOptions({
  queryKey: ['catalog', 'gm-tables'] as const,
  queryFn: api.catalog.gmTables,
  staleTime: Number.POSITIVE_INFINITY,
})

export const dungeonDesignCatalogQueryOptions = queryOptions({
  queryKey: ['catalog', 'dungeon-design'] as const,
  queryFn: api.catalog.dungeonDesign,
  staleTime: Number.POSITIVE_INFINITY,
})
