import { Tooltip as KTooltip } from '@kobalte/core/tooltip'
import { type ComponentProps, splitProps } from 'solid-js'
import { cn } from '@/shared/lib/utils'

/**
 * Tooltip on Kobalte.
 *
 * Note there is no `TooltipProvider` to mount at the app root: Radix kept the
 * shared open-delay there, Kobalte puts it on each tooltip (`openDelay`). The
 * root's `<TooltipProvider delayDuration={150}>` therefore has no counterpart —
 * pass `openDelay={150}` where a tooltip needs it.
 *
 * @example
 * <Tooltip openDelay={150}>
 *   <TooltipTrigger>?</TooltipTrigger>
 *   <TooltipContent>Defesa = 10 + DES + armadura</TooltipContent>
 * </Tooltip>
 */
export const Tooltip = KTooltip
export const TooltipTrigger = KTooltip.Trigger

export type TooltipContentProps = ComponentProps<typeof KTooltip.Content> & {
  /** Portal target; defaults to document.body. Scene-aware in ALE-66. */
  mount?: Node
}

export function TooltipContent(props: TooltipContentProps) {
  const [local, rest] = splitProps(props, ['class', 'children', 'mount'])
  return (
    <KTooltip.Portal mount={local.mount}>
      <KTooltip.Content
        data-slot="tooltip-content"
        class={cn(
          'z-50 w-fit animate-in rounded-md bg-foreground px-3 py-1.5 text-xs text-balance text-background fade-in-0 zoom-in-95 data-[closed]:animate-out data-[closed]:fade-out-0 data-[closed]:zoom-out-95',
          local.class,
        )}
        {...rest}
      >
        {local.children}
        <KTooltip.Arrow />
      </KTooltip.Content>
    </KTooltip.Portal>
  )
}
