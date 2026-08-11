import type { JSX, ParentProps } from 'solid-js'
import { Show } from 'solid-js'
import { cn } from '@/shared/lib/utils'

/**
 * Shared skeleton for every item dialog (bag sheet, melhorias picker,
 * catálogo, item custom): one content width, titled sections wearing the same
 * uppercase label, and a bordered right-aligned footer. Keeping it here is what
 * makes the dialogs read as one family instead of four cousins.
 */

/** DialogContent class shared by all item dialogs. */
export const ITEM_DIALOG_CONTENT =
  'w-[calc(100vw-1.5rem)] max-w-[calc(100vw-1.5rem)] p-4 sm:w-full sm:max-w-md sm:p-5'

/** Title row of an item dialog — name plus whatever chips ride along. */
export const ITEM_DIALOG_TITLE =
  'flex flex-wrap items-center gap-2 font-heading uppercase tracking-wide text-grimorio-gold'

/** One titled block inside an item dialog. */
export function ItemDialogSection(props: ParentProps<{ title: string; class?: string }>) {
  return (
    <section class={cn('space-y-1.5', props.class)}>
      <h3 class="text-[10px] font-bold uppercase tracking-widest text-muted-foreground">
        {props.title}
      </h3>
      {props.children}
    </section>
  )
}

/** Item identity line: quantity × slots = total, plus any warning. */
export function ItemDialogMeta(props: ParentProps) {
  return <p class="text-xs text-muted-foreground">{props.children}</p>
}

/** Bottom action row — bordered, actions pinned right, label pinned left. */
export function ItemDialogFooter(props: ParentProps<{ label?: string }>) {
  return (
    <div class="flex items-center justify-end gap-2 border-t border-border pt-3">
      <Show when={props.label}>
        {(label) => (
          <span class="mr-auto text-[10px] uppercase tracking-widest text-muted-foreground">
            {label()}
          </span>
        )}
      </Show>
      {props.children}
    </div>
  )
}

/** Error line shared by the item forms, announced as it appears. */
export function ItemDialogError(props: { message: string | null }): JSX.Element {
  return (
    <Show when={props.message}>
      {(message) => (
        <p class="text-xs text-destructive" role="alert">
          {message()}
        </p>
      )}
    </Show>
  )
}
