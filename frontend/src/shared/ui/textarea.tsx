import { type ComponentProps, splitProps } from 'solid-js'
import { cn } from '@/shared/lib/utils'

/**
 * Multi-line text input. Mirrors `Input`'s ring/invalid grammar so a form reads
 * as one control family; `aria-invalid` drives the error style.
 *
 * @example <Textarea id="descricao" rows={6} aria-invalid={invalid()} />
 */
export function Textarea(props: ComponentProps<'textarea'>) {
  const [local, rest] = splitProps(props, ['class'])
  return (
    <textarea
      data-slot="textarea"
      class={cn(
        'field-sizing-content min-h-16 w-full rounded-sm border border-input bg-transparent px-3 py-2 text-base shadow-xs transition-[color,box-shadow] outline-none placeholder:text-muted-foreground disabled:cursor-not-allowed disabled:opacity-50 md:text-sm dark:bg-input/30',
        'focus-visible:border-ring focus-visible:ring-[3px] focus-visible:ring-ring/50',
        'aria-invalid:border-destructive aria-invalid:ring-destructive/20 dark:aria-invalid:ring-destructive/40',
        local.class,
      )}
      {...rest}
    />
  )
}
