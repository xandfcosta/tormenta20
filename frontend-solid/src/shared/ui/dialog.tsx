import { Dialog as KDialog } from '@kobalte/core/dialog'
import { XIcon } from 'lucide-solid'
import { type ComponentProps, Show, splitProps } from 'solid-js'
import { cn } from '@/shared/lib/utils'

/**
 * Modal dialog on Kobalte. Radix's `data-[state=open|closed]` becomes
 * `data-[expanded]` / `data-[closed]`; everything else keeps the shadcn look.
 *
 * The React version portalled into the current grimório scene so the dialog
 * inherited the scene tokens. That seam (`useSceneContainer`) is shared/lib
 * and lands with ALE-66 — until then `DialogContent` takes an explicit
 * `mount` element, which is what Kobalte's Portal accepts.
 *
 * @example
 * <Dialog>
 *   <DialogTrigger>Abrir</DialogTrigger>
 *   <DialogContent><DialogTitle>Confirmar</DialogTitle></DialogContent>
 * </Dialog>
 */
export const Dialog = KDialog
export const DialogTrigger = KDialog.Trigger
export const DialogClose = KDialog.CloseButton

export function DialogOverlay(props: ComponentProps<typeof KDialog.Overlay>) {
  const [local, rest] = splitProps(props, ['class'])
  return (
    <KDialog.Overlay
      data-slot="dialog-overlay"
      class={cn(
        'fixed inset-0 z-50 bg-black/50 data-[closed]:animate-out data-[closed]:fade-out-0 data-[expanded]:animate-in data-[expanded]:fade-in-0',
        local.class,
      )}
      {...rest}
    />
  )
}

export type DialogContentProps = ComponentProps<typeof KDialog.Content> & {
  showCloseButton?: boolean
  /** Portal target; defaults to document.body. Scene-aware in ALE-66. */
  mount?: Node
}

export function DialogContent(props: DialogContentProps) {
  const [local, rest] = splitProps(props, ['class', 'children', 'showCloseButton', 'mount'])
  return (
    <KDialog.Portal mount={local.mount}>
      <DialogOverlay />
      <KDialog.Content
        data-slot="dialog-content"
        class={cn(
          'fixed top-[50%] left-[50%] z-50 grid w-full max-w-[calc(100%-2rem)] translate-x-[-50%] translate-y-[-50%] gap-4 rounded-lg border bg-background p-6 shadow-lg duration-200 outline-none data-[closed]:animate-out data-[closed]:fade-out-0 data-[closed]:zoom-out-95 data-[expanded]:animate-in data-[expanded]:fade-in-0 data-[expanded]:zoom-in-95 sm:max-w-lg',
          local.class,
        )}
        {...rest}
      >
        {local.children}
        <Show when={local.showCloseButton ?? true}>
          <KDialog.CloseButton
            data-slot="dialog-close"
            // Kobalte defaults this to aria-label="Dismiss", which OVERRIDES any
            // sr-only text inside — the app is pt-BR, so name it explicitly.
            aria-label="Fechar"
            class="absolute top-4 right-4 rounded-xs opacity-70 ring-offset-background transition-opacity hover:opacity-100 focus:ring-2 focus:ring-ring focus:ring-offset-2 focus:outline-hidden disabled:pointer-events-none [&_svg]:pointer-events-none [&_svg]:shrink-0"
          >
            <XIcon class="size-4" />
          </KDialog.CloseButton>
        </Show>
      </KDialog.Content>
    </KDialog.Portal>
  )
}

export function DialogHeader(props: ComponentProps<'div'>) {
  const [local, rest] = splitProps(props, ['class'])
  return (
    <div
      data-slot="dialog-header"
      class={cn('flex flex-col gap-2 text-center sm:text-left', local.class)}
      {...rest}
    />
  )
}

export function DialogFooter(props: ComponentProps<'div'>) {
  const [local, rest] = splitProps(props, ['class'])
  return (
    <div
      data-slot="dialog-footer"
      class={cn('flex flex-col-reverse gap-2 sm:flex-row sm:justify-end', local.class)}
      {...rest}
    />
  )
}

export function DialogTitle(props: ComponentProps<typeof KDialog.Title>) {
  const [local, rest] = splitProps(props, ['class'])
  return (
    <KDialog.Title
      data-slot="dialog-title"
      class={cn('text-lg leading-none font-semibold', local.class)}
      {...rest}
    />
  )
}

export function DialogDescription(props: ComponentProps<typeof KDialog.Description>) {
  const [local, rest] = splitProps(props, ['class'])
  return (
    <KDialog.Description
      data-slot="dialog-description"
      class={cn('text-sm text-muted-foreground', local.class)}
      {...rest}
    />
  )
}
