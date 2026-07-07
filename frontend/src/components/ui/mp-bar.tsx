import { VitalsBar } from './vitals-bar'

/**
 * MpBar — arcane counterpart to HpBar. Mana never triggers
 * "critical" the same way HP does mechanically, so the bar
 * degrades along a single arcane hue instead of shifting to red.
 */
type MpBarProps = {
  current: number
  max: number
  label?: string
  size?: 'sm' | 'md' | 'lg'
  className?: string
}

function MpBar({
  current,
  max,
  label = 'PM',
  size = 'md',
  className,
}: MpBarProps) {
  return (
    <VitalsBar
      label={label}
      current={current}
      max={max}
      fillFull="--mp-arcane"
      size={size}
      className={className}
    />
  )
}

export { MpBar }
