import { type ComponentProps, splitProps } from 'solid-js'
import { cn } from '@/shared/lib/utils'

/**
 * A keyboard-shortcut badge — the little key hint beside an action, e.g.
 * `Abrir ficha <Kbd>⏎</Kbd>`. Shown only on laptop+desktop (≥xl): the keys
 * don't apply on touch, so the badge hides on tablet/phone. Renders inline so
 * it flows within the button/label text.
 *
 * @example <Button>Abrir ficha <Kbd>⏎</Kbd></Button>
 */
export function Kbd(props: ComponentProps<'kbd'>) {
  const [local, rest] = splitProps(props, ['class'])
  return <kbd class={cn('ml-1 hidden text-3xs opacity-70 xl:inline', local.class)} {...rest} />
}
