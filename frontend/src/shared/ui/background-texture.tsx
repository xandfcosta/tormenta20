import { type ComponentProps, splitProps } from 'solid-js'
import { cn } from '@/shared/lib/utils'

export type BackgroundTextureVariant = 'stone' | 'parchment'

export type BackgroundTextureProps = ComponentProps<'div'> & {
  variant?: BackgroundTextureVariant
  vignette?: boolean
}

/**
 * Full-bleed decorative backdrop for a scene: `stone` (dark, default) or
 * `parchment`. Purely presentational (`aria-hidden`), sits behind content via
 * `-z-10`, so it needs a `position: relative` parent. Pass `vignette` to darken
 * the edges on stone scenes. Layered gradients — no image asset.
 *
 * @example <BackgroundTexture variant="stone" vignette />
 */
export function BackgroundTexture(props: BackgroundTextureProps) {
  const [local, rest] = splitProps(props, ['class', 'variant', 'vignette'])
  const variant = () => local.variant ?? 'stone'
  return (
    <div
      // Explicit "true": bare `aria-hidden` renders as `aria-hidden=""` in
      // Solid (React normalized it to "true"), and an empty value does NOT
      // hide the node from assistive tech.
      aria-hidden="true"
      data-slot="background-texture"
      data-variant={variant()}
      // Contrato observável do opt-in, como o `data-variant`: a classe é o
      // estilo dele, e testar a classe amarra o teste ao CSS.
      data-vignette={local.vignette ? 'true' : undefined}
      class={cn(
        'pointer-events-none absolute inset-0 -z-10',
        variant() === 'stone' ? 'grimorio-stone' : 'grimorio-parchment-bg',
        local.vignette && 'grimorio-vignette',
        local.class,
      )}
      {...rest}
    />
  )
}
