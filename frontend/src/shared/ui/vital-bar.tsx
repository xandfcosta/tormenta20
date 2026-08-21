import { cn } from '@/shared/lib/utils'

/**
 * HP fill token by ratio — the COLOR, not just the width, says "how bad".
 * Lives in shared because both the sheet's HUD and the session tracker paint
 * the same bar, and a feature may not import another feature.
 */
export function hpFillVar(percent: number): string {
  if (percent <= 25) return '--hp-critical'
  if (percent <= 50) return '--hp-hurt'
  return '--hp-full'
}

/**
 * Read-only PV/PM bar. A real `progressbar`, so the number is readable by
 * assistive tech and by the E2E suite — the tracker's rows are the one place
 * a player watches someone else's health, and "some green" is not an answer.
 *
 * @example <VitalBar kind="hp" label="PV" current={12} max={30} />
 */
export function VitalBar(props: {
  kind: 'hp' | 'mp'
  label: string
  current: number
  max: number
  class?: string
}) {
  const percent = () =>
    props.max > 0 ? Math.max(0, Math.min(100, (props.current / props.max) * 100)) : 0
  const fillVar = () => (props.kind === 'hp' ? hpFillVar(percent()) : '--mp-arcane')

  return (
    <div class={cn('flex items-center gap-1.5', props.class)}>
      <span
        class="w-7 shrink-0 text-3xs font-bold uppercase tracking-wider"
        style={{ color: `var(${fillVar()})` }}
      >
        {props.label}
      </span>
      <div
        role="progressbar"
        aria-valuenow={props.current}
        aria-valuemin={0}
        aria-valuemax={props.max}
        aria-label={`${props.label} ${props.current} de ${props.max}`}
        class="h-1.5 min-w-0 flex-1 overflow-hidden rounded-full bg-muted"
      >
        <div
          class="h-full rounded-full transition-[width]"
          style={{ width: `${percent()}%`, background: `var(${fillVar()})` }}
        />
      </div>
      {/* 13px e cor de texto normal, não 10px em muted: PV e PM eram o MENOR
          texto da cena sendo os números mais lidos do combate (ALE-163). */}
      <span class="shrink-0 font-mono text-[13px] tabular-nums text-foreground">
        {props.current}/{props.max}
      </span>
    </div>
  )
}
