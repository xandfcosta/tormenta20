import type { JSX, ParentProps } from 'solid-js'
import { Show } from 'solid-js'

/**
 * A journal entry on the tome page: an illuminated Cinzel heading in gold with
 * a small eyebrow and an optional primary action, over the section body.
 * Shared by the Visão geral panels, the Membros roster and the Sessões log so
 * every section reads as one hand-written book rather than stacked cards.
 */
export function TomeSection(
  props: ParentProps<{ eyebrow: string; title: string; action?: JSX.Element }>,
) {
  return (
    <section class="space-y-4">
      <header class="flex flex-wrap items-end justify-between gap-3">
        <div class="space-y-1">
          <p class="text-[11px] font-semibold uppercase tracking-[0.22em] text-muted-foreground">
            {props.eyebrow}
          </p>
          <h2 class="font-heading text-xl uppercase tracking-wide text-grimorio-gold sm:text-2xl">
            {props.title}
          </h2>
        </div>
        <Show when={props.action}>{(action) => action()}</Show>
      </header>
      {props.children}
    </section>
  )
}
