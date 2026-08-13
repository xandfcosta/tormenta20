import { initials } from '@/shared/lib/initials'
import { Link } from '@tanstack/solid-router'
import { Plus } from 'lucide-solid'
import { For, createEffect } from 'solid-js'
import type { Character } from '@/shared/api/api'
import { hueFromName } from '@/shared/lib/hue-from-name'
import { cn } from '@/shared/lib/utils'

export type CharacterFilmstripProps = {
  roster: Character[]
  /** `'novo'` when the cursor is on the trailing create slot (ALE-98). */
  selectedId: number | 'novo'
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
      {/* Reachable by keyboard as a cursor position (the stage shows the "?"
          frame), and still a one-click Link for the mouse — the arrows are
          additive, they never take a click away (ALE-98). `data-chip-id` keeps
          it inside the auto-centering, so arrowing to the end scrolls it in. */}
      <Link
        to="/characters/new"
        title="Novo personagem"
        data-chip-id="novo"
        aria-current={props.selectedId === 'novo'}
        class={cn(
          'flex size-10 shrink-0 items-center justify-center rounded-md border border-dashed transition-all',
          props.selectedId === 'novo'
            ? 'scale-110 border-grimorio-gold text-grimorio-gold ring-2 ring-grimorio-gold ring-offset-2 ring-offset-background'
            : 'border-border text-muted-foreground hover:bg-accent hover:text-foreground',
        )}
      >
        <Plus class="size-4" />
      </Link>
    </div>
  )
}
