import { Link } from '@tanstack/solid-router'
import { Plus } from 'lucide-solid'
import { For, createEffect } from 'solid-js'
import type { Character } from '@/shared/api/api'
import { hueFromName } from '@/shared/lib/hue-from-name'
import { cn } from '@/shared/lib/utils'

function initials(name: string): string {
  const parts = name.trim().split(/\s+/).filter(Boolean)
  if (parts.length === 0) return '?'
  return parts
    .slice(0, 2)
    .map((word) => word[0])
    .join('')
    .toUpperCase()
}

export type CharacterFilmstripProps = {
  roster: Character[]
  selectedId: number
  onSelect: (id: number) => void
  /** Fired when the pointer enters a chip — e.g. a subtle hover cue. */
  onHover?: () => void
}

/**
 * Filmstrip index under the stage: one hue chip per character for O(1) jumps
 * across long rosters. Auto-centers the selected chip; the terminal "+" chip
 * routes to creation (the fighting-game "random slot", repurposed).
 */
export function CharacterFilmstrip(props: CharacterFilmstripProps) {
  let strip: HTMLDivElement | undefined

  // Recenter whenever the selection moves. In React this needed an explicit
  // dep array (and a biome suppression); here the effect tracks `selectedId`
  // by reading it, and nothing else.
  createEffect(() => {
    const id = props.selectedId
    strip
      ?.querySelector(`[data-chip-id="${id}"]`)
      ?.scrollIntoView({ behavior: 'smooth', inline: 'center', block: 'nearest' })
  })

  return (
    <div
      ref={strip}
      role="listbox"
      aria-label="Personagens"
      // overflow-y-hidden: a lone overflow-x-auto lets the browser add a
      // spurious vertical scrollbar on this single-row strip (sub-pixel height).
      class="flex items-center gap-1.5 overflow-x-auto overflow-y-hidden border-t border-border px-1 py-2"
    >
      <For each={props.roster}>
        {(character) => {
          const hue = hueFromName(character.name)
          const active = () => character.id === props.selectedId
          return (
            <button
              type="button"
              role="option"
              data-chip-id={character.id}
              aria-selected={active()}
              aria-current={active()}
              title={character.name}
              onClick={() => props.onSelect(character.id)}
              onMouseEnter={() => props.onHover?.()}
              class={cn(
                'flex size-10 shrink-0 items-center justify-center rounded-md border font-display text-xs text-white/80 transition-all',
                active()
                  ? 'scale-110 border-grimorio-gold ring-2 ring-grimorio-gold ring-offset-2 ring-offset-background'
                  : 'border-grimorio-iron/60 opacity-60 hover:opacity-100',
              )}
              style={{
                background: `linear-gradient(155deg, oklch(0.5 0.14 ${hue}), oklch(0.3 0.08 ${hue}))`,
              }}
            >
              {initials(character.name)}
            </button>
          )
        }}
      </For>
      <Link
        to="/characters/new"
        title="Novo personagem"
        class="flex size-10 shrink-0 items-center justify-center rounded-md border border-dashed border-border text-muted-foreground transition-colors hover:bg-accent hover:text-foreground"
      >
        <Plus class="size-4" />
      </Link>
    </div>
  )
}
