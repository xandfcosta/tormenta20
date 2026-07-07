import { cn } from '@/lib/utils'

/**
 * DicePill — visual chip for a die roll. Renders as `Nd<sides>` in
 * the mono face inside a hexagonal-ish pill. Optional `modifier`
 * appends `+N` / `-N`. `variant='crit'` marks a natural-20 result
 * with ember accent for post-roll history views.
 */
type DiceKind = 4 | 6 | 8 | 10 | 12 | 20 | 100

type DicePillProps = {
  count?: number
  sides: DiceKind
  modifier?: number
  variant?: 'default' | 'crit' | 'fail'
  className?: string
}

function DicePill({
  count = 1,
  sides,
  modifier,
  variant = 'default',
  className,
}: DicePillProps) {
  const modText =
    modifier === undefined || modifier === 0
      ? ''
      : modifier > 0
        ? ` +${modifier}`
        : ` ${modifier}`
  return (
    <span
      data-slot="dice-pill"
      data-variant={variant}
      className={cn(
        'inline-flex items-center gap-1 rounded-md border px-2 py-0.5 font-hud text-xs tabular-nums',
        variant === 'default' &&
          'border-border/60 bg-card text-foreground',
        variant === 'crit' &&
          'border-[color:var(--primary)] bg-[color-mix(in_oklch,var(--primary)_18%,transparent)] text-[color:var(--primary)]',
        variant === 'fail' &&
          'border-[color:var(--hp-critical)]/70 bg-[color-mix(in_oklch,var(--hp-critical)_15%,transparent)] text-[color:var(--hp-critical)]',
        className,
      )}
      aria-label={`${count}d${sides}${modText}`}
    >
      <span>
        {count}d{sides}
      </span>
      {modText && <span>{modText}</span>}
    </span>
  )
}

export { DicePill }
export type { DiceKind }
