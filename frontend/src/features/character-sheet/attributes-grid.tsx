import { For, createMemo } from 'solid-js'
import { computedSheetFor } from '@/entities/character/computed-sheet'
import { ATTRIBUTE_ABBR, ATTRIBUTE_KEYS } from '@/entities/character/expertise'
import type { Character } from '@/shared/api/api'
import { cn } from '@/shared/lib/utils'
import { signed } from './signed'

/**
 * The six attribute boxes. `class` sets the column template, so the caller (HUD
 * strip vs. aside column) decides how they wrap.
 *
 * Stored attributes are BASE (pre-race): the racial modifier is folded in by
 * the engine, so what shows here is the derived total — the number the player
 * actually rolls with.
 */
export function AttributesGrid(props: {
  character: Character
  activeConditionals: ReadonlySet<string>
  class?: string
}) {
  const attributes = createMemo(
    () => computedSheetFor(props.character, props.activeConditionals).attributes,
  )
  return (
    <div class={cn('grid gap-2', props.class)}>
      <For each={ATTRIBUTE_KEYS}>
        {(key) => (
          <AttributeBox label={ATTRIBUTE_ABBR[key]} value={attributes()[key].total} />
        )}
      </For>
    </div>
  )
}

function AttributeBox(props: { label: string; value: number }) {
  return (
    <div class="rounded-none border-2 border-grimorio-iron bg-grimorio-panel p-2 text-center">
      <p class="text-[9px] font-bold uppercase tracking-widest text-muted-foreground">
        {props.label}
      </p>
      <p class="mt-0.5 text-2xl font-bold leading-none text-foreground">{signed(props.value)}</p>
    </div>
  )
}
