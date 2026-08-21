import type { DisplayFact } from '@/shared/api/display-facts'
import { For, Show } from 'solid-js'
import { cn } from '@/shared/lib/utils'

/**
 * Display-only mechanical facts (RD, immunities, senses, movement, action
 * economy) as reference chips. These are shown, never computed — the
 * companion-app affordance for effects the engine cannot model.
 *
 * @example <FactChips facts={catalog.displayFacts ?? []} />
 */
export function FactChips(props: { facts: readonly DisplayFact[]; class?: string }) {
  return (
    <Show when={props.facts.length > 0}>
      <ul class={cn('flex flex-wrap gap-1', props.class)}>
        <For each={props.facts}>
          {(fact) => (
            <li class="rounded-md border border-border bg-muted px-1.5 py-0.5 text-3xs leading-tight text-foreground">
              {fact.text}
            </li>
          )}
        </For>
      </ul>
    </Show>
  )
}
