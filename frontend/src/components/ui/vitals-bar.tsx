import { cn } from '@/lib/utils'

/**
 * VitalsBar — shared bar component powering HpBar + MpBar.
 *
 * "Controlled decay" happens here: the fill color transitions across
 * three decay states derived from the current/max ratio. Vitals-bar
 * consumers pass explicit `--fill-*` CSS variable names so HP and MP
 * can reuse the same visual mechanism while resolving to different
 * semantic tokens (`--hp-full/hurt/critical` vs `--mp-arcane`).
 *
 * The number ticker is placed on top of the fill so damage flashes
 * animate just the digits — full-card shake was flagged in the
 * responsive audit as a vestibular risk.
 */
type VitalsBarProps = {
  label: string
  current: number
  max: number
  fillFull: string
  fillHurt?: string
  fillCritical?: string
  size?: 'sm' | 'md' | 'lg'
  className?: string
}

const trackHeights = {
  sm: 'h-3',
  md: 'h-5',
  lg: 'h-7',
} as const

function pickFill(
  ratio: number,
  full: string,
  hurt?: string,
  critical?: string,
): string {
  if (critical && ratio <= 0.25) return critical
  if (hurt && ratio <= 0.5) return hurt
  return full
}

function VitalsBar({
  label,
  current,
  max,
  fillFull,
  fillHurt,
  fillCritical,
  size = 'md',
  className,
}: VitalsBarProps) {
  const clamped = Math.max(0, Math.min(current, max))
  const ratio = max > 0 ? clamped / max : 0
  const percent = Math.round(ratio * 100)
  const state: 'full' | 'hurt' | 'critical' =
    fillCritical && ratio <= 0.25
      ? 'critical'
      : fillHurt && ratio <= 0.5
        ? 'hurt'
        : 'full'
  const fill = pickFill(ratio, fillFull, fillHurt, fillCritical)

  return (
    <div
      data-slot="vitals-bar"
      data-state={state}
      className={cn('flex w-full flex-col gap-1', className)}
    >
      <div className="flex items-baseline justify-between text-xs">
        <span className="font-display tracking-widest uppercase text-muted-foreground">
          {label}
        </span>
        <span className="font-hud tabular-nums text-foreground">
          {clamped}
          <span className="text-muted-foreground"> / {max}</span>
        </span>
      </div>
      <div
        role="progressbar"
        aria-label={label}
        aria-valuenow={clamped}
        aria-valuemin={0}
        aria-valuemax={max}
        className={cn(
          'relative w-full overflow-hidden rounded-sm border border-border/50 bg-muted/40',
          trackHeights[size],
        )}
      >
        <div
          data-slot="vitals-bar-fill"
          className="h-full transition-[width,background-color] duration-500 ease-out"
          style={{
            width: `${percent}%`,
            backgroundColor: `var(${fill})`,
          }}
        />
      </div>
    </div>
  )
}

export { VitalsBar }
