import { createVirtualizer } from '@tanstack/solid-virtual'
import { For, type JSX, Show } from 'solid-js'
import { cn } from '@/shared/lib/utils'

export type VirtualListProps<T> = {
  items: readonly T[]
  /** First-paint height guess per row in px; the real height is measured. */
  estimateSize?: number
  /** Rows rendered beyond the viewport on each side. */
  overscan?: number
  getKey: (item: T, index: number) => string | number
  renderItem: (item: T) => JSX.Element
  /** Sets the scroll container's bounded height. */
  class?: string
}

/**
 * Project-owned wrapper over `@tanstack/solid-virtual`, so features never
 * import the library directly. Renders only the rows near the viewport, which
 * is what keeps the item catalog (~400 entries) cheap to scroll inside a
 * dialog. Rows self-measure, so wrapping names need no fixed height.
 *
 * Only the bounded-container variant is ported: nothing in the Solid app
 * virtualizes against the page scroll yet, and an unused second variant is a
 * second thing to keep correct.
 *
 * @example
 * <VirtualList class="max-h-56" items={filtered()} estimateSize={34}
 *   getKey={(c) => c.id} renderItem={(c) => <CatalogRow catalog={c} />} />
 */
export function VirtualList<T>(props: VirtualListProps<T>) {
  let scrollRef: HTMLDivElement | undefined
  const virtualizer = createVirtualizer({
    get count() {
      return props.items.length
    },
    getScrollElement: () => scrollRef ?? null,
    estimateSize: () => props.estimateSize ?? 56,
    // Measurements keyed by item identity, not index: filtering shuffles items
    // across indices, and index-keyed heights would leave stale gaps. The
    // out-of-range guard is not paranoia — the virtualizer asks about index -1
    // for an element it could not identify, and reading `items[-1].id` there
    // took down the whole scene.
    getItemKey: (index) => {
      const item = props.items[index]
      return item === undefined ? index : props.getKey(item, index)
    },
    get overscan() {
      return props.overscan ?? 8
    },
  })

  return (
    <div ref={scrollRef} class={cn('overflow-y-auto', props.class)}>
      <div
        data-slot="virtual-spacer"
        class="relative w-full"
        style={{ height: `${virtualizer.getTotalSize()}px` }}
      >
        <For each={virtualizer.getVirtualItems()}>
          {(row) => (
            <div
              data-index={row.index}
              // Measured in a microtask on purpose: Solid runs `ref` BEFORE it
              // sets the attributes, and the virtualizer identifies what it
              // measured by reading `data-index` off the node. Measuring in the
              // same tick reads an empty attribute, resolves to index -1 and
              // poisons the size cache.
              ref={(element) => queueMicrotask(() => virtualizer.measureElement(element))}
              class="absolute top-0 left-0 w-full"
              style={{ transform: `translateY(${row.start}px)` }}
            >
              {/* `keyed`: a row div survives a filter change (the virtualizer
                  reconciles virtual items by index), so a plain Show would keep
                  showing the item this row painted first — search for "escudo"
                  and read back "Adaga". */}
              <Show keyed when={props.items[row.index]}>
                {(item) => props.renderItem(item)}
              </Show>
            </div>
          )}
        </For>
      </div>
    </div>
  )
}
