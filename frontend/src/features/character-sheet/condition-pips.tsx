import { conditionEffectSummary } from '@/shared/rules/condition-modifiers'
import type { ConditionId } from '@/shared/api/catalog-types'
import { For, Show, createMemo } from 'solid-js'
import type { Character } from '@/shared/api/api'
import { conditionsRecord } from '@/shared/lib/rules-catalog-cache'
import { cn } from '@/shared/lib/utils'
import { Popover, PopoverContent, PopoverTrigger } from '@/shared/ui/popover'
import { parseActiveConditions } from './active-conditions'

/**
 * The character's active conditions at a glance. Lives here, not inside the
 * Efeitos block (ALE-86): the HUD shows it on every viewport, and a chip should
 * not have to import the whole panel that manages conditions.
 *
 * `mini` is the HUD variant — it shares ONE row with the class badges, and
 * beyond three conditions folds into a ⚠+N popover, because a dedicated
 * conditions row doubled the nameplate height.
 */
export function ConditionPips(props: { character: Character; class?: string; mini?: boolean }) {
  const active = createMemo(() => parseActiveConditions(props.character.activeConditions))
  const shown = createMemo(() => (props.mini ? active().slice(0, 3) : active()))
  const overflow = () => active().length - shown().length

  const chipClass = () =>
    props.mini
      ? 'max-w-20 truncate rounded border border-[color:var(--hp-hurt)]/60 bg-[color:var(--hp-hurt)]/15 px-1 text-[9px] font-semibold uppercase tracking-wide text-[color:var(--hp-hurt)]'
      : 'rounded border border-[color:var(--hp-hurt)]/60 bg-[color:var(--hp-hurt)]/15 px-1.5 py-px text-[10px] font-semibold uppercase tracking-wide text-[color:var(--hp-hurt)]'

  return (
    <Show when={active().length > 0}>
      <ul class={cn('flex flex-wrap items-center gap-1', props.class)}>
        <For each={shown()}>
          {(id) => (
            <li title={conditionsRecord()[id].description} class={chipClass()}>
              {conditionsRecord()[id].name}
            </li>
          )}
        </For>
        <Show when={overflow() > 0}>
          <li>
            <Popover>
              <PopoverTrigger
                as="button"
                type="button"
                class={chipClass()}
                aria-label={`Mais ${overflow()} condições`}
              >
                ⚠+{overflow()}
              </PopoverTrigger>
              <PopoverContent class="w-72 space-y-2 text-xs">
                <For each={active()}>{(id) => <ConditionDetail id={id} />}</For>
              </PopoverContent>
            </Popover>
          </li>
        </Show>
      </ul>
    </Show>
  )
}

function ConditionDetail(props: { id: ConditionId }) {
  return (
    <div>
      <p class="font-semibold uppercase">
        {conditionsRecord()[props.id].name}{' '}
        <span class="font-normal normal-case text-[color:var(--hp-hurt)]">
          {conditionEffectSummary(props.id)}
        </span>
      </p>
      <p class="text-muted-foreground">{conditionsRecord()[props.id].description}</p>
    </div>
  )
}
