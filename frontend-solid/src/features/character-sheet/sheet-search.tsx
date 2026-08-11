import { createMemo, createSignal, onCleanup, onMount } from 'solid-js'
import type { Character } from '@/shared/api/api'
import { CommandPalette, type PaletteGroup } from '@/shared/ui/command-palette'
import { resolveSheetTab } from './sheet-sections'
import { groupSheetEntries, rankSheetEntries } from './sheet-search-groups'
import { buildSheetSearchIndex } from './sheet-search-index'

/** True while the keystroke belongs to a field the player is typing in. */
function typingInAField(target: EventTarget | null): boolean {
  const element = target as HTMLElement | null
  return element?.tagName === 'INPUT' || element?.tagName === 'TEXTAREA'
}

/**
 * Sheet-global search palette: `/` or Ctrl/Cmd+K opens it, every row shows the
 * ANSWER inline (perícia total, item quantity, rule text) so most lookups end
 * without navigating — selecting a row still jumps to the block that owns the
 * fact, for the full context.
 *
 * @example <SheetSearch character={character} onNavigate={goToTab} />
 */
export function SheetSearch(props: {
  character: Character
  onNavigate: (tab: string) => void
}) {
  const [open, setOpen] = createSignal(false)
  const [query, setQuery] = createSignal('')

  onMount(() => {
    const onKeyDown = (event: KeyboardEvent) => {
      const palette =
        event.key === '/' || (event.key.toLowerCase() === 'k' && (event.ctrlKey || event.metaKey))
      // `/` is a legitimate character inside a field; Ctrl+K is not.
      if (!palette || (event.key === '/' && typingInAField(event.target))) return
      event.preventDefault()
      setOpen(!open())
    }
    window.addEventListener('keydown', onKeyDown)
    onCleanup(() => window.removeEventListener('keydown', onKeyDown))
  })

  // Built on demand: a few hundred entries, and only while the palette is open.
  const index = createMemo(() => (open() ? buildSheetSearchIndex(props.character) : []))

  const groups = createMemo<PaletteGroup[]>(() =>
    groupSheetEntries(rankSheetEntries(index(), query()), {
      ranked: query().trim() !== '',
    }).map((group) => ({
      heading: group.source,
      items: group.entries.map((entry) => ({
        id: `${entry.source}:${entry.name}`,
        label: entry.name,
        detail: entry.detail,
        onSelect: () => {
          // The index still carries the pre-merge tab names ('inventory',
          // 'equipment'); the URL should get the canonical one.
          props.onNavigate(resolveSheetTab(entry.tab))
          close()
        },
      })),
    })),
  )

  const close = () => {
    setOpen(false)
    setQuery('')
  }

  return (
    <CommandPalette
      open={open()}
      onOpenChange={(next) => (next ? setOpen(true) : close())}
      groups={groups()}
      query={query()}
      onQueryChange={setQuery}
      title="Buscar na ficha"
      description="Perícias, poderes, itens, magias e condições"
      placeholder="Buscar na ficha… (perícia, poder, item, magia)"
    />
  )
}
