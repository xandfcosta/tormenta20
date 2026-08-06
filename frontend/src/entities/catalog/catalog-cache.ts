import type { CatalogItem } from '@tormenta20/t20-data'

/**
 * Front-owned catalog cache with SYNC accessors mirroring t20-data's
 * (`getCatalogItem`…). Lets the core derive pipeline (derived.ts) and other
 * deep consumers read catalog data without a build-time `import` of the data —
 * so the ~44KB item catalog (and later races/origins/spells) tree-shakes out of
 * the bundle and is fetched + cached instead.
 *
 * Correctness rests on ONE invariant: the cache is primed (via the route
 * loader's `ensureCatalogs`) BEFORE any consumer renders. Catalogs are static +
 * cached-forever, so by render time the primed value is effectively a constant
 * and derived.ts needs no reactivity to it. Accessors return undefined before
 * priming — the loader gate guarantees that never happens on a real render.
 */
let itemsById: ReadonlyMap<string, CatalogItem> | null = null

/** Prime the item catalog from the fetched list. Idempotent (re-priming with
 *  the same static data is a no-op in effect). Called by `ensureCatalogs`. */
export function primeItemCatalog(items: readonly CatalogItem[]): void {
  itemsById = new Map(items.map((i) => [i.id, i]))
}

/** Cache-backed mirror of t20-data `getCatalogItem` — reads the primed item
 *  catalog. Undefined before prime (see the module invariant). */
export function getCatalogItem(id: string): CatalogItem | undefined {
  return itemsById?.get(id)
}

/** True once the item catalog has been primed — for a render-time gate/assert. */
export function isItemCatalogPrimed(): boolean {
  return itemsById !== null
}
