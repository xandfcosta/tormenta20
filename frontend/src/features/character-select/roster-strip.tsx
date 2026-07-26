import { Link } from '@tanstack/react-router'
import { Plus } from 'lucide-react'
import { type KeyboardEvent, useEffect, useRef } from 'react'
import type { Character } from '@/shared/api/api'
import { CharacterPortrait } from '@/shared/ui/character-portrait'
import { hueFromName } from '@/shared/lib/hue-from-name'
import { cn } from '@/shared/lib/utils'

/**
 * The roster. Two layouts share one thumb: a 2-row horizontal snap-**strip**
 * (the Valorant-style bottom rail — denser than a single row) and an
 * **expanded** full grid for scanning big rosters. Selection is roving-tabindex
 * with arrow-key nav; Enter opens the selected sheet. The selected thumb is
 * scrolled into view whenever selection changes.
 */
export function Roster({
  roster,
  selectedId,
  onSelect,
  onOpen,
  expanded,
}: {
  roster: Character[]
  selectedId: number
  onSelect: (id: number) => void
  onOpen: (id: number) => void
  expanded: boolean
}) {
  const containerRef = useRef<HTMLDivElement>(null)

  // Keep the selected thumb visible as selection moves (strip can overflow).
  useEffect(() => {
    const el = containerRef.current?.querySelector<HTMLElement>(
      `[data-char-id="${selectedId}"]`,
    )
    el?.scrollIntoView({ block: 'nearest', inline: 'nearest' })
  }, [selectedId])

  const move = (delta: number) => {
    const idx = roster.findIndex((c) => c.id === selectedId)
    const next = roster[Math.min(roster.length - 1, Math.max(0, idx + delta))]
    if (next) onSelect(next.id)
  }

  const onKeyDown = (e: KeyboardEvent<HTMLDivElement>) => {
    if (e.key === 'ArrowRight' || e.key === 'ArrowDown') {
      e.preventDefault()
      move(1)
    } else if (e.key === 'ArrowLeft' || e.key === 'ArrowUp') {
      e.preventDefault()
      move(-1)
    } else if (e.key === 'Enter' || e.key === ' ') {
      e.preventDefault()
      onOpen(selectedId)
    }
  }

  if (roster.length === 0) {
    return (
      <p className="py-6 text-center text-sm text-muted-foreground">
        Nenhum personagem corresponde à busca.
      </p>
    )
  }

  return (
    <div
      ref={containerRef}
      role="listbox"
      aria-label="Personagens"
      onKeyDown={onKeyDown}
      className={cn(
        // p-1.5 keeps the selected thumb's offset outline from being clipped
        // by the scroll container's overflow edge.
        'gap-2 p-1.5',
        expanded
          ? 'grid auto-rows-min grid-cols-3 overflow-y-auto sm:grid-cols-4 lg:grid-cols-6'
          : 'grid grid-flow-col grid-rows-2 auto-cols-max snap-x scroll-pl-2 overflow-x-auto pb-2',
      )}
    >
      {roster.map((c) => (
        <Thumb
          key={c.id}
          character={c}
          selected={c.id === selectedId}
          expanded={expanded}
          onSelect={() => onSelect(c.id)}
          onOpen={() => onOpen(c.id)}
        />
      ))}
    </div>
  )
}

/** One-line affordance hint for the otherwise-invisible interaction model. */
export function RosterHint() {
  return (
    <p className="text-[11px] text-muted-foreground">
      Clique para ver · clique de novo (ou Enter) para abrir · ↔ navega
    </p>
  )
}

function Thumb({
  character,
  selected,
  expanded,
  onSelect,
  onOpen,
}: {
  character: Character
  selected: boolean
  expanded: boolean
  onSelect: () => void
  onOpen: () => void
}) {
  const hue = hueFromName(character.name)
  return (
    <button
      type="button"
      role="option"
      aria-selected={selected}
      aria-label={character.name}
      tabIndex={selected ? 0 : -1}
      data-char-id={character.id}
      title={character.name}
      // Selecting and opening share the thumb: first click selects, a second
      // click on the already-selected thumb opens the sheet (matches the Enter
      // key + keeps the panel button as the explicit path).
      onClick={() => (selected ? onOpen() : onSelect())}
      style={selected ? { outlineColor: `oklch(0.6 0.16 ${hue})` } : undefined}
      className={cn(
        'flex shrink-0 snap-start flex-col items-center gap-1 rounded-lg border border-border p-1.5 transition-colors',
        expanded ? 'w-full' : 'w-20',
        selected
          ? 'bg-accent outline outline-2 outline-offset-2'
          : 'hover:bg-accent',
      )}
    >
      <CharacterPortrait
        name={character.name}
        size="lg"
        hue={hue}
        className="aspect-square text-2xl"
      />
      <span className="w-full truncate text-center text-[11px]">
        {character.name}
      </span>
    </button>
  )
}

/** Dashed "new character" tile — pinned in the header, not trailing the strip. */
export function NewCharacterTile() {
  return (
    <Link
      to="/characters/new"
      aria-label="Novo personagem"
      className="flex shrink-0 items-center gap-1 rounded-lg border border-dashed px-3 py-2 text-sm text-muted-foreground hover:bg-accent hover:text-foreground"
    >
      <Plus className="size-4" />
      <span>Novo</span>
    </Link>
  )
}
