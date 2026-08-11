import { queryOptions } from '@tanstack/solid-query'
import { api } from '@/shared/api/api'

/**
 * Race definitions (innate abilities per race). Static rulebook reference, so
 * it never goes stale — fetched from /catalog instead of bundled, which is why
 * the front ships no catalog data (project_front_decouple_catalog).
 */
export const raceDefsQueryOptions = queryOptions({
  queryKey: ['catalog', 'race-defs'] as const,
  queryFn: api.catalog.raceDefs,
  staleTime: Number.POSITIVE_INFINITY,
})
