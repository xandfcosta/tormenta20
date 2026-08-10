import { Link } from '@tanstack/react-router'
import { Plus } from 'lucide-react'
import { useEffect, useRef } from 'react'
import type { Character } from '@/shared/api/api'
import { hueFromName } from '@/shared/lib/hue-from-name'
import { cn } from '@/shared/lib/utils'

function initials(name: string): string {
  const parts = name.trim().split(/\s+/).filter(Boolean)
  if (parts.length === 0) return '?'
  return parts
    .slice(0, 2)
    .map((w) => w[0])
    .join('')
    .toUpperCase()
}

/**
 * Filmstrip index under the stage: one hue chip per character for O(1) jumps
 * across long rosters. Auto-centers the selected chip; the terminal "+" chip
 * routes to creation (the fighting-game "random slot", repurposed).
 */
export function CharacterFilmstrip({
  roster,
  selectedId,
  onSelect,
  onHover,
}: {
  roster: Character[]
  selectedId: number
  onSelect: (id: number) => void
  /** Fired when the pointer enters a chip — e.g. a subtle hover cue. */
  onHover?: () => void
}) {
  const stripRef = useRef<HTMLDivElement>(null)
  // biome-ignore lint/correctness/useExhaustiveDependencies: recenter on selection change only.
  useEffect(() => {
    stripRef.current
      ?.querySelector('[aria-current="true"]')
      ?.scrollIntoView({ behavior: 'smooth', inline: 'center', block: 'nearest' })
  }, [selectedId])
  return (
    <div
      ref={stripRef}
      role="listbox"
      aria-label="Personagens"
      // overflow-y-hidden: a lone overflow-x-auto lets the browser add a
      // spurious vertical scrollbar on this single-row strip (sub-pixel height).
      className="flex items-center gap-1.5 overflow-x-auto overflow-y-hidden border-t border-border px-1 py-2"
    >
      {roster.map((c) => {
        const hue = hueFromName(c.name)
        const active = c.id === selectedId
        return (
          <button
            key={c.id}
            type="button"
            role="option"
            aria-selected={active}
            aria-current={active}
            title={c.name}
            onClick={() => onSelect(c.id)}
            onMouseEnter={onHover}
            className={cn(
              'flex size-10 shrink-0 items-center justify-center rounded-md border font-display text-xs text-white/80 transition-all',
              active
                ? 'scale-110 border-grimorio-gold ring-2 ring-grimorio-gold ring-offset-2 ring-offset-background'
                : 'border-grimorio-iron/60 opacity-60 hover:opacity-100',
            )}
            style={{
              background: `linear-gradient(155deg, oklch(0.5 0.14 ${hue}), oklch(0.3 0.08 ${hue}))`,
            }}
          >
            {initials(c.name)}
          </button>
        )
      })}
      <Link
        to="/characters/new"
        title="Novo personagem"
        className="flex size-10 shrink-0 items-center justify-center rounded-md border border-dashed border-border text-muted-foreground transition-colors hover:bg-accent hover:text-foreground"
      >
        <Plus className="size-4" />
      </Link>
    </div>
  )
}
