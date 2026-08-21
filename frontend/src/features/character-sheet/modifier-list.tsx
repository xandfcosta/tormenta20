import type { Modifier } from '@/shared/api/item-types'
import { For, Show } from 'solid-js'
import { describeConditionalTarget } from './conditional-target-label'
import { signed } from './signed'

/** The numeric modifiers of an effect, as "Defesa +2" style pairs. Shared by
 *  the buff picker and the running-effect rows. */
export function ModifierList(props: { modifiers: readonly Modifier[]; class?: string }) {
  return (
    <Show when={props.modifiers.length > 0}>
      <ul class={props.class ?? 'flex flex-wrap gap-x-3 gap-y-0.5 text-2xs'}>
        <For each={props.modifiers}>
          {(modifier) => (
            <li class="flex items-center gap-1">
              <span class="text-muted-foreground">
                {describeConditionalTarget(modifier.target)}
              </span>
              <span
                class={
                  modifier.amount >= 0
                    ? 'font-mono font-semibold text-emerald-300'
                    : 'font-mono font-semibold text-red-300'
                }
              >
                {signed(modifier.amount)}
              </span>
            </li>
          )}
        </For>
      </ul>
    </Show>
  )
}
