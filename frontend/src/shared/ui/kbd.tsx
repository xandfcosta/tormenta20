import type { ComponentProps } from 'react'
import { cn } from '@/shared/lib/utils'

/**
 * A keyboard-shortcut badge — the little key hint beside an action, e.g.
 * `Abrir ficha <Kbd>⏎</Kbd>`. Shown only on laptop+desktop (≥xl): the keys
 * don't apply on touch, so the badge hides on tablet/phone. Renders inline so
 * it flows within the button/label text.
 *
 * @example
 * <Button>Abrir ficha <Kbd>⏎</Kbd></Button>
 */
export function Kbd({ className, ...props }: ComponentProps<'kbd'>) {
  return (
    <kbd
      className={cn('ml-1 hidden text-[10px] opacity-70 xl:inline', className)}
      {...props}
    />
  )
}
