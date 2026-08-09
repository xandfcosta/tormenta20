import type * as React from 'react'
import { cn } from '@/shared/lib/utils'

/**
 * BackgroundTexture — a full-bleed decorative backdrop for a scene. `stone`
 * (dark, default) or `parchment`. Purely presentational (`aria-hidden`), sits
 * behind content via `-z-10`, so place it inside a `position: relative`
 * parent. Pass `vignette` to darken the edges on stone scenes. Lightweight —
 * layered gradients, no image asset.
 */
type BackgroundTextureVariant = 'stone' | 'parchment'

type BackgroundTextureProps = React.ComponentProps<'div'> & {
  variant?: BackgroundTextureVariant
  vignette?: boolean
}

function BackgroundTexture({
  className,
  variant = 'stone',
  vignette = false,
  ...props
}: BackgroundTextureProps) {
  return (
    <div
      aria-hidden
      data-slot="background-texture"
      data-variant={variant}
      className={cn(
        'pointer-events-none absolute inset-0 -z-10',
        variant === 'stone' ? 'grimorio-stone' : 'grimorio-parchment-bg',
        vignette && 'grimorio-vignette',
        className,
      )}
      {...props}
    />
  )
}

export { BackgroundTexture }
export type { BackgroundTextureVariant }
