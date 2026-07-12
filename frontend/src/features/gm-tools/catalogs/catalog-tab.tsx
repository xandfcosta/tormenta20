import { type ReactNode, useMemo } from 'react'
import { VirtualList } from '@/shared/ui/virtual-list'
import { matchesQuery } from './catalog-model'

/**
 * One tab of the GM catalog browser: filters a static catalog array by the
 * shared search query and renders the matches as a virtualized list, so even
 * the ~560-entry power list stays cheap. `searchText` must be a stable
 * (module-level) function — it's assumed referentially constant so filtering
 * memoizes on `items` + `query` alone.
 */
export function CatalogTab<T>({
  items,
  query,
  searchText,
  estimateSize,
  gap = 8,
  getKey,
  renderRow,
  emptyLabel = 'Nada encontrado.',
}: {
  items: readonly T[]
  query: string
  searchText: (item: T) => string[]
  estimateSize: number
  gap?: number
  getKey: (item: T) => string
  renderRow: (item: T) => ReactNode
  emptyLabel?: string
}) {
  const filtered = useMemo(
    () => items.filter((it) => matchesQuery(searchText(it), query)),
    [items, query, searchText],
  )

  return (
    <div className="space-y-2">
      <p className="text-xs text-muted-foreground">
        {filtered.length} de {items.length}
      </p>
      {filtered.length === 0 ? (
        <p className="p-6 text-center text-sm text-muted-foreground">
          {emptyLabel}
        </p>
      ) : (
        <VirtualList
          className="max-h-[65vh] pr-1"
          items={filtered}
          estimateSize={estimateSize}
          gap={gap}
          getKey={getKey}
          renderItem={renderRow}
        />
      )}
    </div>
  )
}
