import { ChevronDown, ChevronUp } from 'lucide-solid'
import { type ComponentProps, splitProps } from 'solid-js'
import { cn } from '@/shared/lib/utils'
import { Input } from './input'

export type NumberInputProps = Omit<
  ComponentProps<'input'>,
  'value' | 'onChange' | 'type'
> & {
  value: number | string
  onChange: (value: number) => void
  min?: number
  max?: number
  step?: number
  /** Names the spinner buttons ("Aumentar deslocamento"). Required in practice
   *  wherever a screen holds more than one — three bare "Aumentar" buttons are
   *  three identical announcements. */
  spinnerLabel?: string
}

/** Keeps a spinner step inside the field's own bounds. */
export function clampToRange(n: number, min?: number, max?: number): number {
  if (typeof min === 'number' && n < min) return min
  if (typeof max === 'number' && n > max) return max
  return n
}

/**
 * Numeric field with its own spinner column — a `0,5` step is a tap, not a
 * keyboard exercise (T20 tracks encumbrance in half-espaço steps). The native
 * spinner is unstyleable and invisible on touch, hence the pair of buttons.
 *
 * The React kit also carried an `onCommit` for "changed by spinner, not by
 * typing"; nothing in this app used it on a bag field, so it is not ported.
 *
 * @example <NumberInput value={slots()} onChange={setSlots} min={0.5} step={0.5} />
 */
export function NumberInput(props: NumberInputProps) {
  const [local, rest] = splitProps(props, [
    'class',
    'value',
    'onChange',
    'min',
    'max',
    'step',
    'disabled',
    'spinnerLabel',
  ])
  const numeric = () => (typeof local.value === 'number' ? local.value : Number(local.value) || 0)
  const step = () => local.step ?? 1
  const atMin = () => typeof local.min === 'number' && numeric() <= local.min
  const atMax = () => typeof local.max === 'number' && numeric() >= local.max

  const adjust = (delta: number) => {
    if (local.disabled) return
    local.onChange(clampToRange(numeric() + delta, local.min, local.max))
  }

  return (
    <div class={cn('relative', local.class)}>
      <Input
        {...rest}
        type="number"
        inputMode="numeric"
        value={local.value}
        onInput={(event) => local.onChange(Number(event.currentTarget.value))}
        min={local.min}
        max={local.max}
        step={step()}
        disabled={local.disabled}
        class="pr-7"
      />
      <div class="pointer-events-none absolute inset-y-0 right-0 flex w-6 flex-col">
        <SpinnerButton
          direction="up"
          name={local.spinnerLabel}
          disabled={local.disabled || atMax()}
          onClick={() => adjust(step())}
        />
        <SpinnerButton
          direction="down"
          name={local.spinnerLabel}
          disabled={local.disabled || atMin()}
          onClick={() => adjust(-step())}
        />
      </div>
    </div>
  )
}

/** Pointer affordance only — `tabIndex={-1}` leaves the keyboard to the field's
 *  own arrow keys, and the labels are pt-BR like the rest of the app. */
function SpinnerButton(props: {
  direction: 'up' | 'down'
  name?: string
  disabled?: boolean
  onClick: () => void
}) {
  const label = () =>
    [props.direction === 'up' ? 'Aumentar' : 'Diminuir', props.name].filter(Boolean).join(' ')

  return (
    <button
      type="button"
      tabIndex={-1}
      disabled={props.disabled}
      onClick={() => props.onClick()}
      aria-label={label()}
      class={cn(
        'pointer-events-auto flex flex-1 items-center justify-center text-muted-foreground transition-colors',
        'hover:bg-accent/50 hover:text-foreground',
        'disabled:cursor-not-allowed disabled:opacity-30 disabled:hover:bg-transparent',
        props.direction === 'up' ? 'rounded-tr-sm' : 'rounded-br-sm',
      )}
    >
      {props.direction === 'up' ? (
        <ChevronUp aria-hidden="true" class="size-3" />
      ) : (
        <ChevronDown aria-hidden="true" class="size-3" />
      )}
    </button>
  )
}
