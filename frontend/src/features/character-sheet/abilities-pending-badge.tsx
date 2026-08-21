import { Show, createMemo } from 'solid-js'
import type { Character } from '@/shared/api/api'
import { computePendencias } from './pendencias'

/**
 * Numeric pill on the "Poderes" tab trigger — counts unmade required choices
 * (race variants, origin benefits, class powers/paths). Renders nothing when
 * there's nothing outstanding, like the Efeitos badge.
 */
export function AbilitiesPendingBadge(props: { character: Character }) {
  const count = createMemo(() => computePendencias(props.character).length)

  return (
    <Show when={count() > 0}>
      {/* Hidden from assistive tech with the meaning spelled out beside it: a
          bare `aria-label` on a <span> has no role to carry it and announces
          only the number. */}
      <span
        aria-hidden="true"
        class="ml-0.5 inline-flex h-4 min-w-4 items-center justify-center rounded-full bg-destructive px-1 text-3xs font-bold text-white"
      >
        {count()}
      </span>
      <span class="sr-only">{count()} escolhas pendentes</span>
    </Show>
  )
}
