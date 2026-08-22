import { type VariantProps, cva } from 'class-variance-authority'
import { type ComponentProps, splitProps } from 'solid-js'
import { cn } from '@/shared/lib/utils'

const buttonVariants = cva(
  "inline-flex shrink-0 items-center justify-center gap-2 rounded-sm text-sm font-medium whitespace-nowrap transition-all outline-none disabled:pointer-events-none disabled:opacity-50 aria-invalid:border-destructive aria-invalid:ring-destructive/20 dark:aria-invalid:ring-destructive/40 [&_svg]:pointer-events-none [&_svg]:shrink-0 [&_svg:not([class*='size-'])]:size-4",
  {
    variants: {
      variant: {
        default: 'bg-primary text-primary-foreground hover:bg-primary/90',
        // Sem os `dark:` que vieram do shadcn: o app NÃO põe a classe `dark`
        // em lugar nenhum — a paleta escura mora no `:root` —, então eles
        // nunca aplicaram. Um deles era literalmente um `dark:` sem utilitário
        // atrás, herdado com o kit.
        destructive: 'bg-destructive text-white hover:bg-destructive/90',
        outline:
          'border bg-background shadow-xs hover:bg-accent hover:text-accent-foreground dark:border-input dark:bg-input/30 dark:hover:bg-input/50',
        secondary: 'bg-secondary text-secondary-foreground hover:bg-secondary/80',
        ghost: 'hover:bg-accent hover:text-accent-foreground dark:hover:bg-accent/50',
        link: 'text-primary underline-offset-4 hover:underline',
      },
      size: {
        default: 'h-9 px-4 py-2 has-[>svg]:px-3',
        xs: "h-6 gap-1 rounded-sm px-2 text-xs has-[>svg]:px-1.5 [&_svg:not([class*='size-'])]:size-3",
        sm: 'h-8 gap-1.5 rounded-sm px-3 has-[>svg]:px-2.5',
        lg: 'h-10 rounded-sm px-6 has-[>svg]:px-4',
        icon: 'size-9',
        'icon-xs': "size-6 rounded-sm [&_svg:not([class*='size-'])]:size-3",
        'icon-sm': 'size-8',
        'icon-lg': 'size-10',
      },
    },
    defaultVariants: { variant: 'default', size: 'default' },
  },
)

export type ButtonProps = ComponentProps<'button'> & VariantProps<typeof buttonVariants>

/**
 * The app's button. Same variants and classes as the React kit — only the
 * plumbing changed: `splitProps` instead of rest-destructuring (Solid props
 * are reactive; destructuring them freezes their values).
 *
 * The React version had an `asChild` escape hatch (Radix Slot). Solid has no
 * equivalent, and the app's uses were all "render a link that looks like a
 * button" — style the `<A>` with `buttonVariants()` instead.
 *
 * @example <Button variant="outline" size="sm" onClick={roll}>Rolar</Button>
 */
export function Button(props: ButtonProps) {
  const [local, rest] = splitProps(props, ['class', 'variant', 'size', 'type'])
  return (
    <button
      type={local.type ?? 'button'}
      data-slot="button"
      data-variant={local.variant ?? 'default'}
      data-size={local.size ?? 'default'}
      class={cn(buttonVariants({ variant: local.variant, size: local.size }), local.class)}
      {...rest}
    />
  )
}

export { buttonVariants }
