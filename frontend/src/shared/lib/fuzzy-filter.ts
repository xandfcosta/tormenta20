import { rankItem } from '@tanstack/match-sorter-utils'
import type { FilterFn } from '@tanstack/react-table'

/**
 * Project-owned fuzzy filter for TanStack Table (global or column filter).
 * Ranks the cell value against the query with `match-sorter`, which is
 * typo-tolerant and (by default) diacritic-insensitive — a good fit for the
 * accented Portuguese catalogs. Wrap the lib here so features never import
 * `match-sorter-utils` directly (frontend third-party-boundary rule).
 *
 * Usage: `columnHelper.accessor('name', { filterFn: fuzzyFilter<Row>() })` or
 * the table option `globalFilterFn: fuzzyFilter<Row>()`. A factory so the
 * returned `FilterFn` is typed to the table's row shape (no `any` cast).
 */
export function fuzzyFilter<T>(): FilterFn<T> {
  return (row, columnId, value, addMeta) => {
    const search = String(value ?? '')
    if (search.trim() === '') return true
    const itemRank = rankItem(row.getValue(columnId), search)
    addMeta({ itemRank })
    return itemRank.passed
  }
}
