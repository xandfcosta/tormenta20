import { Popover as KPopover } from '@kobalte/core/popover'
import { type ComponentProps, splitProps } from 'solid-js'
import { useSceneContainer } from '@/shared/lib/scene-container'
import { cn } from '@/shared/lib/utils'

/**
 * Popover on Kobalte. Radix's `data-[state=open]` becomes `data-[expanded]`;
 * placement is `placement="bottom"` rather than `side`/`align` pairs.
 *
 * @example
 * <Popover>
 *   <PopoverTrigger>Filtros</PopoverTrigger>
 *   <PopoverContent>…</PopoverContent>
 * </Popover>
 */
export const Popover = KPopover
export const PopoverTrigger = KPopover.Trigger
export const PopoverAnchor = KPopover.Anchor

export type PopoverContentProps = ComponentProps<typeof KPopover.Content> & {
  /** Portal target. Defaults to the enclosing grimório scene, else body. */
  mount?: Node
}

export function PopoverContent(props: PopoverContentProps) {
  const [local, rest] = splitProps(props, ['class', 'mount'])
  const scene = useSceneContainer()
  return (
    <KPopover.Portal mount={local.mount ?? scene() ?? undefined}>
      <KPopover.Content
        data-slot="popover-content"
        class={cn(
          'z-50 w-72 rounded-sm border bg-popover p-4 text-popover-foreground shadow-md outline-hidden data-[closed]:animate-out data-[closed]:fade-out-0 data-[closed]:zoom-out-95 data-[expanded]:animate-in data-[expanded]:fade-in-0 data-[expanded]:zoom-in-95',
          local.class,
        )}
        {...rest}
      />
    </KPopover.Portal>
  )
}

export function PopoverHeader(props: ComponentProps<'div'>) {
  const [local, rest] = splitProps(props, ['class'])
  return <div data-slot="popover-header" class={cn('flex flex-col gap-1 text-sm', local.class)} {...rest} />
}

export function PopoverTitle(props: ComponentProps<typeof KPopover.Title>) {
  const [local, rest] = splitProps(props, ['class'])
  return <KPopover.Title data-slot="popover-title" class={cn('font-medium', local.class)} {...rest} />
}

export function PopoverDescription(props: ComponentProps<typeof KPopover.Description>) {
  const [local, rest] = splitProps(props, ['class'])
  return (
    <KPopover.Description
      data-slot="popover-description"
      class={cn('text-muted-foreground', local.class)}
      {...rest}
    />
  )
}
