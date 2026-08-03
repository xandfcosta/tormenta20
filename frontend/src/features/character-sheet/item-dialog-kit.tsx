import { accentStrong, dimText } from '@/shared/lib/sheet-theme'
import { cn } from '@/shared/lib/utils'

/**
 * Shared structure for every item dialog (bag sheet, melhorias picker,
 * catálogo, item custom): the same content width, titled sections with the
 * same uppercase label, and a bordered right-aligned footer. Keeping the
 * skeleton here is what makes the dialogs read as one family.
 */

/** DialogContent className shared by all item dialogs. */
export const ITEM_DIALOG_CONTENT =
  'w-[calc(100vw-1.5rem)] max-w-[calc(100vw-1.5rem)] p-4 sm:w-full sm:max-w-md sm:p-5'

/** One titled block inside an item dialog. */
export function ItemDialogSection({
  title,
  children,
  className,
}: {
  title: string
  children: React.ReactNode
  className?: string
}) {
  return (
    <section className={cn('space-y-1.5', className)}>
      <h3 className={cn('text-[10px] font-bold uppercase tracking-widest', dimText)}>
        {title}
      </h3>
      {children}
    </section>
  )
}

/** Item identity line: quantity × slots = total, plus optional warnings. */
export function ItemDialogMeta({ children }: { children: React.ReactNode }) {
  return <p className={cn('text-xs', dimText)}>{children}</p>
}

/** Bottom action row — bordered, actions pinned right, label pinned left. */
export function ItemDialogFooter({
  label,
  children,
}: {
  label?: string
  children: React.ReactNode
}) {
  return (
    <div className="flex items-center justify-end gap-2 border-t border-border pt-3">
      {label && (
        <span className={cn('mr-auto text-[10px] uppercase tracking-widest', dimText)}>
          {label}
        </span>
      )}
      {children}
    </div>
  )
}

/** Emphasized inline name used in dialog titles. */
export const itemDialogTitleClass = cn('flex flex-wrap items-center gap-2', accentStrong)
