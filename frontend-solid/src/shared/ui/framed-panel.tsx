import { type ComponentProps, splitProps } from 'solid-js'
import { cn } from '@/shared/lib/utils'

export type FramedPanelVariant = 'stone' | 'parchment'

export type FramedPanelProps = ComponentProps<'div'> & {
  variant?: FramedPanelVariant
}

/**
 * The default scene container: an iron border with a faint gold inset filete.
 * `stone` (dark, default) is the standard surface; `parchment` is a light aged
 * surface with dark ink for highlighted content (spell text, descriptions).
 * Meant to be used inside a `.scene-grimorio` scope.
 *
 * @example <FramedPanel variant="parchment" class="p-3">…</FramedPanel>
 */
export function FramedPanel(props: FramedPanelProps) {
  const [local, rest] = splitProps(props, ['class', 'variant', 'children'])
  const variant = () => local.variant ?? 'stone'
  return (
    <div
      data-slot="framed-panel"
      data-variant={variant()}
      class={cn(
        'grimorio-frame p-5',
        variant() === 'stone' ? 'grimorio-frame--stone text-foreground' : 'grimorio-parchment-bg',
        local.class,
      )}
      {...rest}
    >
      {local.children}
    </div>
  )
}
