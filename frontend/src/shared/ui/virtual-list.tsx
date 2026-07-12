import { type ReactNode, useRef } from 'react'
import {
  type VirtualItem,
  type Virtualizer,
  useVirtualizer,
  useWindowVirtualizer,
} from '@tanstack/react-virtual'
import { cn } from '@/shared/lib/utils'

/**
 * Thin project-owned wrapper over `@tanstack/react-virtual` so features never
 * import the library directly. Renders only the rows near the viewport, which
 * keeps big catalogs (spells ~200, class powers ~460, bestiary) cheap to
 * scroll. Rows are measured dynamically (`measureElement`), so variable-height
 * content (wrapping names, expandable rows) needs no fixed height — pass
 * `estimateSize` only as the first-paint guess.
 *
 * Two variants:
 *   - `VirtualList` — owns a bounded scroll container (`className` sets its
 *     height); use inside dialogs, drawers, panels.
 *   - `WindowVirtualList` — virtualizes against the page scroll; use for
 *     full-page lists that shouldn't introduce a nested scrollbar.
 */
type SharedProps<T> = {
  items: readonly T[]
  /** First-paint height guess per row in px; real height is measured. */
  estimateSize?: number
  /** Rows rendered beyond the viewport on each side. */
  overscan?: number
  /** Vertical gap between rows in px. */
  gap?: number
  getKey: (item: T, index: number) => string | number
  renderItem: (item: T) => ReactNode
}

export function VirtualList<T>({
  items,
  estimateSize = 56,
  overscan = 8,
  gap = 0,
  getKey,
  renderItem,
  className,
}: SharedProps<T> & { className?: string }) {
  const scrollRef = useRef<HTMLDivElement>(null)
  const virtualizer = useVirtualizer({
    count: items.length,
    getScrollElement: () => scrollRef.current,
    estimateSize: () => estimateSize,
    overscan,
    gap,
  })

  return (
    <div ref={scrollRef} className={cn('overflow-y-auto', className)}>
      <VirtualRows
        virtualizer={virtualizer}
        items={items}
        getKey={getKey}
        renderItem={renderItem}
      />
    </div>
  )
}

export function WindowVirtualList<T>({
  items,
  estimateSize = 56,
  overscan = 8,
  gap = 0,
  getKey,
  renderItem,
}: SharedProps<T>) {
  const listRef = useRef<HTMLDivElement>(null)
  const virtualizer = useWindowVirtualizer({
    count: items.length,
    estimateSize: () => estimateSize,
    overscan,
    gap,
    // Offset of this list from the top of the page scroll, so row positions
    // account for whatever chrome sits above it.
    scrollMargin: listRef.current?.offsetTop ?? 0,
  })

  return (
    <div ref={listRef}>
      <VirtualRows
        virtualizer={virtualizer}
        items={items}
        getKey={getKey}
        renderItem={renderItem}
        scrollMargin={virtualizer.options.scrollMargin}
      />
    </div>
  )
}

/**
 * Shared absolute-positioned row layer for both variants. The total-size
 * spacer reserves scroll height; each visible row is translated into place and
 * self-measures via `measureElement` (keyed by `data-index`).
 */
function VirtualRows<T>({
  virtualizer,
  items,
  getKey,
  renderItem,
  scrollMargin = 0,
}: {
  virtualizer:
    | Virtualizer<HTMLDivElement, Element>
    | Virtualizer<Window, Element>
  items: readonly T[]
  getKey: (item: T, index: number) => string | number
  renderItem: (item: T) => ReactNode
  scrollMargin?: number
}) {
  return (
    <div
      style={{
        height: virtualizer.getTotalSize(),
        position: 'relative',
        width: '100%',
      }}
    >
      {virtualizer.getVirtualItems().map((row: VirtualItem) => {
        const item = items[row.index]
        if (item === undefined) return null
        return (
          <div
            key={getKey(item, row.index)}
            data-index={row.index}
            ref={virtualizer.measureElement}
            style={{
              position: 'absolute',
              top: 0,
              left: 0,
              width: '100%',
              transform: `translateY(${row.start - scrollMargin}px)`,
            }}
          >
            {renderItem(item)}
          </div>
        )
      })}
    </div>
  )
}
