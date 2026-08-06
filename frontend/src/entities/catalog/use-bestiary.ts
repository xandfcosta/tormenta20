import { useQuery } from '@tanstack/react-query'
import type { Monster } from '@tormenta20/t20-data'
import { bestiaryCatalogQueryOptions } from './queries'

/**
 * The bestiary (monster list) from the backend, replacing the build-time
 * `import { BESTIARY }`. `data` is undefined until the cached-forever fetch
 * resolves — GM/session screens fall back to an empty list meanwhile.
 *
 * @example const monsters = useBestiary().data ?? []
 */
export function useBestiary() {
  return useQuery(bestiaryCatalogQueryOptions)
}

/** Resolve a monster by id off a loaded bestiary (undefined until loaded). */
export function monsterById(
  bestiary: readonly Monster[] | undefined,
  id: string,
): Monster | undefined {
  return bestiary?.find((m) => m.id === id)
}
