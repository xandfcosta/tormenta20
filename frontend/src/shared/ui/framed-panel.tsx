import type * as React from 'react'
import { cn } from '@/shared/lib/utils'

/**
 * FramedPanel — the default scene container: an iron border with a faint gold
 * inset filete. `stone` (dark, default) is the standard surface; `parchment`
 * is a light aged surface with dark ink for highlighted content (spell text,
 * descriptions, cards). Meant to be used inside a `.scene-grimorio` scope.
 */
type FramedPanelVariant = 'stone' | 'parchment'

type FramedPanelProps = React.ComponentProps<'div'> & {
  variant?: FramedPanelVariant
}

function FramedPanel({
  className,
  variant = 'stone',
  children,
  ...props
}: FramedPanelProps) {
  return (
    <div
      data-slot="framed-panel"
      data-variant={variant}
      className={cn(
        'grimorio-frame p-5',
        variant === 'stone'
          ? 'grimorio-frame--stone text-foreground'
          : 'grimorio-parchment-bg',
        className,
      )}
      {...props}
    >
      {children}
    </div>
  )
}

export { FramedPanel }
export type { FramedPanelVariant }
