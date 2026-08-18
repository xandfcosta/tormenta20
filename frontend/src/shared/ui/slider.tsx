import { createUniqueId } from 'solid-js'
import { cn } from '@/shared/lib/utils'

export type SliderProps = {
  /** Visible name, associated by `for`+`id` — it is also the a11y name. */
  label: string
  value: number
  onChange: (value: number) => void
  min?: number
  max?: number
  step?: number
  /** How the current value reads on screen. */
  format?: (value: number) => string
  class?: string
}

/**
 * A single-value slider on the platform's own `input[type=range]`.
 *
 * NOT on Kobalte, and the reason is worth keeping: its slider positions the
 * thumb from an index that is only registered after the first render, so the
 * first pass emits `left: calc(NaN%)` — a browser drops the invalid
 * declaration, but jsdom's CSS parser THROWS and takes the whole tree with it,
 * which made the Hub's quick menu unrenderable in tests. The native control
 * already gives role, keyboard and value announcement for free (ALE-180).
 *
 * @example <Slider label="Volume" value={vol()} onChange={setVol} format={(v) => `${v}%`} />
 */
export function Slider(props: SliderProps) {
  const id = createUniqueId()
  return (
    <div class={cn('flex flex-col gap-1', props.class)}>
      <div class="flex items-center justify-between text-xs text-muted-foreground">
        <label for={id}>{props.label}</label>
        <span class="tabular-nums">{(props.format ?? String)(props.value)}</span>
      </div>
      <input
        id={id}
        type="range"
        min={props.min ?? 0}
        max={props.max ?? 100}
        step={props.step ?? 5}
        value={props.value}
        onInput={(event) => props.onChange(event.currentTarget.valueAsNumber)}
        class="h-1.5 w-full cursor-pointer accent-grimorio-gold"
      />
    </div>
  )
}
