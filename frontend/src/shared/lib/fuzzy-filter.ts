import { rankItem } from '@tanstack/match-sorter-utils'

/**
 * Typo-tolerant, diacritic-insensitive match — the search rule behind the
 * roster and list filters. Matters for the accented Portuguese catalogs
 * ("necromante" finds "Necromante", "anao" finds "Anão"). Wrapped here so
 * features never import `match-sorter-utils` directly (third-party-boundary
 * rule).
 *
 * DEVIATION from the React app: there, this was a `FilterFn` plugged into a
 * headless react-table (v8). The Solid adapter is `@tanstack/solid-table`
 * **v9** — a restructured library (explicit feature opt-in, a feature-typed
 * filter-meta registry) whose wiring for a single client-side global filter
 * over a handful of rows costs far more than it returns. The typo tolerance,
 * which is the part that actually mattered, is preserved. A list that needs
 * real sorting/faceting/virtualization should bring solid-table in properly.
 *
 * @example roster.filter((c) => matchesQuery([c.name, c.origin], query()))
 */
export function fuzzyMatches(value: unknown, search: string): boolean {
  if (search.trim() === '') return true
  return rankItem(value, search).passed
}

/**
 * True when ANY of the indexed fields matches — the multi-column equivalent of
 * the table's global filter.
 *
 * @example matchesQuery([c.name, className, c.origin], 'necro')
 */
export function matchesQuery(fields: readonly unknown[], search: string): boolean {
  if (search.trim() === '') return true
  return fields.some((field) => fuzzyMatches(field, search))
}
