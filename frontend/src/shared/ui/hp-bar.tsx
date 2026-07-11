import { VitalsBar } from './vitals-bar'

/**
 * HpBar — wraps VitalsBar with the HP semantic decay palette. Splits
 * ratio into full → hurt (≤50%) → critical (≤25%). Bar color, not
 * width, carries the "how bad is it" signal — width alone is easy to
 * miss on mobile at speed.
 */
type HpBarProps = {
  current: number
  max: number
  label?: string
  size?: 'sm' | 'md' | 'lg'
  className?: string
}

function HpBar({
  current,
  max,
  label = 'PV',
  size = 'md',
  className,
}: HpBarProps) {
  return (
    <VitalsBar
      label={label}
      current={current}
      max={max}
      fillFull="--hp-full"
      fillHurt="--hp-hurt"
      fillCritical="--hp-critical"
      size={size}
      className={className}
    />
  )
}

export { HpBar }
