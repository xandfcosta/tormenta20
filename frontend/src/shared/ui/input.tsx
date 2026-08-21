import { type ComponentProps, splitProps } from 'solid-js'
import { cn } from '@/shared/lib/utils'

/**
 * Text input. Same classes as the React kit; `aria-invalid` drives the error
 * ring, so callers mark validity rather than swapping styles.
 *
 * @example <Input id="email" type="email" aria-invalid={invalid()} />
 */
export function Input(props: ComponentProps<'input'>) {
  const [local, rest] = splitProps(props, ['class'])
  return (
    <input
      data-slot="input"
      class={cn(
        'h-9 w-full min-w-0 rounded-sm border border-input bg-transparent px-3 py-1 text-base shadow-xs transition-[color,box-shadow] outline-none selection:bg-primary selection:text-primary-foreground file:inline-flex file:h-7 file:border-0 file:bg-transparent file:text-sm file:font-medium file:text-foreground placeholder:text-muted-foreground disabled:pointer-events-none disabled:cursor-not-allowed disabled:opacity-50 md:text-sm dark:bg-input/30',
        '',
        'aria-invalid:border-destructive aria-invalid:ring-destructive/20 dark:aria-invalid:ring-destructive/40',
        local.class,
      )}
      {...rest}
    />
  )
}
