import type { ParentProps } from 'solid-js'
import { Show } from 'solid-js'
import { cn } from '@/shared/lib/utils'
import { FieldLabel } from '@/shared/ui/section-label'

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
      <FieldLabel as="h3" class="font-bold">
        {props.title}
      </FieldLabel>
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
          <FieldLabel class="mr-auto">
            {label()}
          </FieldLabel>
        )}
      </Show>
      {props.children}
    </div>
  )
}
