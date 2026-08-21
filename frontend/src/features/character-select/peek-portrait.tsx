import { initials } from '@/shared/lib/initials'
import { portraitGradient } from './select-helpers'
import { Show } from 'solid-js'
import type { Character } from '@/shared/api/api'

export type PeekPortraitProps = {
  character: Character | null
  side: 'left' | 'right'
  onClick: () => void
}

/**
 * The dimmed ±1 neighbour flanking the stage — restrained coverflow, no 3D.
 * Clicking it steps the selection that way. Renders a fixed-width spacer when
 * there is no neighbour, so the stage never shifts sideways at the ends.
 *
 * Its own file because BOTH stages show it (a hero's, and the trailing create
 * slot's) — the first version of the create slot copied the markup instead and
 * shipped a peek with no gradient and no monogram: a black box with a caption.
 *
 * The name is always on, never hover-gated: hover doesn't exist on touch or
 * under keyboard navigation, and two initials don't say who's next.
 */
export function PeekPortrait(props: PeekPortraitProps) {
  return (
    <Show
      when={props.character}
      fallback={<div class="w-20 sm:w-28 lg:w-32" aria-hidden="true" />}
    >
      {(character) => (
        <button
          type="button"
          onClick={() => props.onClick()}
          title={character().name}
          aria-label={`${props.side === 'left' ? 'Anterior' : 'Próximo'}: ${character().name}`}
          class="group relative aspect-[3/4] w-20 overflow-hidden rounded-sm border border-grimorio-iron opacity-50 transition-all hover:opacity-80 sm:w-28 lg:w-32"
          style={{ background: portraitGradient(character().name) }}
        >
          <span class="absolute inset-0 flex select-none items-center justify-center font-display text-4xl text-white/20 sm:text-5xl">
            {initials(character().name)}
          </span>
          <span class="absolute inset-x-0 bottom-0 line-clamp-2 bg-black/60 px-1 py-0.5 text-center text-[10px] leading-tight text-white">
            {character().name}
          </span>
        </button>
      )}
    </Show>
  )
}
