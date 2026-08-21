import { Search } from 'lucide-solid'
import { For, Show, createEffect, createSignal, onCleanup } from 'solid-js'
import { cn } from '@/shared/lib/utils'
import { Dialog, DialogContent, DialogDescription, DialogTitle } from './dialog'

export type PaletteItem = {
  /** Stable identity for the row. */
  id: string
  label: string
  /** The inline ANSWER, so most lookups end without navigating. */
  detail?: string
  onSelect: () => void
}

export type PaletteGroup = { heading: string; items: PaletteItem[] }

/**
 * Keyboard-first command palette.
 *
 * Hand-rolled rather than a dependency: the React app used `cmdk`, which has no
 * Solid build, and the ranking this needs already lives in the caller (it hands
 * over groups already filtered and sorted). What is left is a dialog, an input,
 * and a cursor — worth owning.
 *
 * Focus: it opens from a global hotkey, so there is no trigger element for the
 * dialog to return focus to. It remembers whatever was focused when it opened
 * and restores it on close, or the scene-nav cursor is left orphaned.
 *
 * @example
 * <CommandPalette open={open()} onOpenChange={setOpen} groups={groups()}
 *   query={query()} onQueryChange={setQuery} title="Buscar na ficha" />
 */
export function CommandPalette(props: {
  open: boolean
  onOpenChange: (open: boolean) => void
  groups: PaletteGroup[]
  query: string
  onQueryChange: (query: string) => void
  title: string
  description?: string
  placeholder?: string
  emptyMessage?: string
}) {
  const [active, setActive] = createSignal(0)
  let previouslyFocused: HTMLElement | null = null

  const flat = () => props.groups.flatMap((group) => group.items)

  // Typing re-ranks the list, so the cursor must go back to the top or it
  // points at whatever happened to sit at that index before.
  createEffect(() => {
    props.query
    setActive(0)
  })

  createEffect(() => {
    if (props.open) {
      previouslyFocused = document.activeElement as HTMLElement | null
      return
    }
    // Restore on close — including Esc, which never runs a select handler.
    previouslyFocused?.focus?.()
    previouslyFocused = null
  })

  onCleanup(() => {
    previouslyFocused = null
  })

  const move = (delta: number) => {
    const total = flat().length
    if (total === 0) return
    setActive((current) => (current + delta + total) % total)
  }

  const onKeyDown = (event: KeyboardEvent) => {
    if (event.key === 'ArrowDown') {
      event.preventDefault()
      move(1)
      return
    }
    if (event.key === 'ArrowUp') {
      event.preventDefault()
      move(-1)
      return
    }
    if (event.key === 'Enter') {
      event.preventDefault()
      flat()[active()]?.onSelect()
    }
  }

  // Running index across groups, so the cursor is continuous while the headings
  // still break the list up visually.
  const indexOf = (groupIndex: number, itemIndex: number) =>
    props.groups.slice(0, groupIndex).reduce((sum, group) => sum + group.items.length, 0) +
    itemIndex

  return (
    <Dialog open={props.open} onOpenChange={props.onOpenChange}>
      <DialogContent class="w-[calc(100vw-1.5rem)] max-w-xl gap-0 overflow-hidden p-0 sm:w-full">
        {/* The palette's own input is the visible label, so the dialog's title
            and description exist for assistive tech only. */}
        <DialogTitle class="sr-only">{props.title}</DialogTitle>
        <Show when={props.description}>
          {(description) => <DialogDescription class="sr-only">{description()}</DialogDescription>}
        </Show>

        <div class="flex items-center gap-2 border-b border-border px-3">
          <Search aria-hidden="true" class="size-4 shrink-0 text-muted-foreground" />
          <input
            type="text"
            value={props.query}
            onInput={(event) => props.onQueryChange(event.currentTarget.value)}
            onKeyDown={onKeyDown}
            placeholder={props.placeholder}
            aria-label={props.title}
            role="combobox"
            aria-expanded={flat().length > 0}
            aria-controls="command-palette-list"
            aria-activedescendant={flat()[active()] ? rowId(flat()[active()].id) : undefined}
            autocomplete="off"
            autofocus
            class="h-12 w-full bg-transparent text-sm outline-none placeholder:text-muted-foreground"
          />
        </div>

        {/* Divs, not ul/li: the listbox/option roles are interactive and a list
            element may not carry them. Same shape cmdk renders. */}
        <div
          id="command-palette-list"
          role="listbox"
          aria-label={props.title}
          class="max-h-80 overflow-y-auto p-1"
        >
          <Show
            when={flat().length > 0}
            fallback={
              <p class="px-3 py-6 text-center text-sm text-muted-foreground">
                {props.emptyMessage ?? 'Nada encontrado.'}
              </p>
            }
          >
            <For each={props.groups}>
              {(group, groupIndex) => (
                // Flat listbox: the headings group VISUALLY only. ARIA's
                // `role="group"` would be the formal way, but each option is
                // self-describing and the input announces the active one.
                <div>
                  <p class="px-2 pb-1 pt-2 text-[10px] font-bold uppercase tracking-widest text-muted-foreground">
                    {group.heading}
                  </p>
                  <For each={group.items}>
                    {(item, itemIndex) => (
                      <PaletteRow
                        item={item}
                        active={indexOf(groupIndex(), itemIndex()) === active()}
                        onHover={() => setActive(indexOf(groupIndex(), itemIndex()))}
                      />
                    )}
                  </For>
                </div>
              )}
            </For>
          </Show>
        </div>
      </DialogContent>
    </Dialog>
  )
}

/** DOM id of a row, so the input can point at it with aria-activedescendant. */
function rowId(itemId: string): string {
  return `command-palette-${itemId.replace(/[^a-zA-Z0-9_-]/g, '-')}`
}

function PaletteRow(props: { item: PaletteItem; active: boolean; onHover: () => void }) {
  return (
    // The row IS the option: the keyboard drives it from the input through
    // aria-activedescendant, and a pointer clicks it directly — so it needs no
    // focus of its own, and wrapping a button inside an option confuses AT.
    // biome-ignore lint/a11y/useKeyWithClickEvents: keyboard lives on the input.
    <div
      id={rowId(props.item.id)}
      role="option"
      // -1: reachable programmatically (the aria-activedescendant pattern),
      // never in the tab order — the input keeps the focus.
      tabIndex={-1}
      aria-selected={props.active}
      onClick={() => props.item.onSelect()}
      onMouseMove={() => props.onHover()}
      class={cn(
        'flex w-full cursor-pointer items-center gap-2 rounded-none px-2 py-1.5 text-left text-sm',
        props.active && 'bg-accent text-accent-foreground',
      )}
    >
      <span class="min-w-0 flex-1 truncate font-medium">{props.item.label}</span>
      <Show when={props.item.detail}>
        {(detail) => (
          <span class="ml-2 max-w-[55%] truncate text-xs text-muted-foreground">{detail()}</span>
        )}
      </Show>
    </div>
  )
}
