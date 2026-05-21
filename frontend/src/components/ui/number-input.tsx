import * as React from 'react'
import { ChevronDown, ChevronUp } from 'lucide-react'

import { cn } from '@/lib/utils'
import { Input } from '@/components/ui/input'

type NumberInputProps = Omit<
  React.ComponentProps<'input'>,
  'value' | 'onChange' | 'type'
> & {
  value: number | string
  onChange: (value: number) => void
  /**
   * Fired only when the value changes via the +/- spinner buttons.
   * Typing in the input does NOT fire this. Useful for triggering
   * a commit (mutation, autosave) without firing on every keystroke.
   */
  onCommit?: (value: number) => void
  min?: number
  max?: number
  step?: number
}

function clamp(n: number, min?: number, max?: number): number {
  let out = n
  if (typeof min === 'number' && out < min) out = min
  if (typeof max === 'number' && out > max) out = max
  return out
}

function NumberInput({
  className,
  value,
  onChange,
  onCommit,
  min,
  max,
  step = 1,
  disabled,
  ...rest
}: NumberInputProps) {
  const numericValue = typeof value === 'number' ? value : Number(value) || 0

  const adjust = (delta: number) => {
    if (disabled) return
    const next = clamp(numericValue + delta, min, max)
    onChange(next)
    onCommit?.(next)
  }

  const atMin = typeof min === 'number' && numericValue <= min
  const atMax = typeof max === 'number' && numericValue >= max

  return (
    <div className={cn('relative', className)}>
      <Input
        {...rest}
        type="number"
        inputMode="numeric"
        value={value}
        onChange={(e) => onChange(Number(e.target.value))}
        min={min}
        max={max}
        step={step}
        disabled={disabled}
        className="pr-7"
      />
      <div
        aria-hidden
        className="pointer-events-none absolute inset-y-0 right-0 flex w-6 flex-col"
      >
        <SpinnerButton
          direction="up"
          disabled={disabled || atMax}
          onClick={() => adjust(step)}
        />
        <SpinnerButton
          direction="down"
          disabled={disabled || atMin}
          onClick={() => adjust(-step)}
        />
      </div>
    </div>
  )
}

function SpinnerButton({
  direction,
  disabled,
  onClick,
}: {
  direction: 'up' | 'down'
  disabled?: boolean
  onClick: () => void
}) {
  const Icon = direction === 'up' ? ChevronUp : ChevronDown
  return (
    <button
      type="button"
      tabIndex={-1}
      disabled={disabled}
      onClick={onClick}
      aria-label={direction === 'up' ? 'Increase' : 'Decrease'}
      className={cn(
        'pointer-events-auto flex flex-1 items-center justify-center text-muted-foreground transition-colors',
        'hover:text-foreground hover:bg-accent/50',
        'disabled:cursor-not-allowed disabled:opacity-30 disabled:hover:bg-transparent',
        direction === 'up' ? 'rounded-tr-md' : 'rounded-br-md',
      )}
    >
      <Icon className="size-3" />
    </button>
  )
}

export { NumberInput }
