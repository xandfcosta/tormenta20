import { ArrowDown, ArrowUp } from 'lucide-react'
import { cn } from '@/lib/utils'

/**
 * StatBlock — a large numeric readout for character attributes,
 * modifiers, or aggregated derived values. `label` sits above the
 * number in a small caps serif; the number itself uses the HUD mono
 * face so digits align across cards.
 *
 * `trend` renders an inline arrow next to the value — useful for
 * conditional-effect deltas (e.g. +2 STR from a buff). Omit for
 * static readings.
 */
type StatBlockProps = {
  label: string
  value: number | string
  sublabel?: string
  trend?: 'up' | 'down'
  size?: 'sm' | 'md' | 'lg'
  className?: string
}

const valueSizes = {
  sm: 'text-2xl',
  md: 'text-4xl',
  lg: 'text-6xl',
} as const

function StatBlock({
  label,
  value,
  sublabel,
  trend,
  size = 'md',
  className,
}: StatBlockProps) {
  return (
    <div
      data-slot="stat-block"
      data-trend={trend ?? 'none'}
      className={cn(
        'flex flex-col items-center gap-1 rounded-md border border-border/60 bg-card/70 px-3 py-4 text-center',
        className,
      )}
    >
      <span className="font-display text-[0.7rem] tracking-[0.18em] uppercase text-muted-foreground">
        {label}
      </span>
      <span
        className={cn(
          'font-hud font-semibold leading-none text-foreground',
          valueSizes[size],
        )}
      >
        <span className="inline-flex items-baseline gap-1">
          {value}
          {trend === 'up' && (
            <ArrowUp
              className="size-4 text-[color:var(--hp-full)]"
              aria-label="tendência positiva"
            />
          )}
          {trend === 'down' && (
            <ArrowDown
              className="size-4 text-[color:var(--hp-critical)]"
              aria-label="tendência negativa"
            />
          )}
        </span>
      </span>
      {sublabel && (
        <span className="text-xs text-muted-foreground">{sublabel}</span>
      )}
    </div>
  )
}

export { StatBlock }
