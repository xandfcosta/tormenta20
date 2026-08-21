import { type VariantProps, cva } from 'class-variance-authority'
import { type ComponentProps, splitProps } from 'solid-js'
import { cn } from '@/shared/lib/utils'

const badgeVariants = cva(
  "inline-flex w-fit shrink-0 items-center justify-center gap-1 overflow-hidden rounded-full border border-transparent px-2 py-0.5 text-xs font-medium whitespace-nowrap transition-[color,box-shadow] aria-invalid:border-destructive aria-invalid:ring-destructive/20 dark:aria-invalid:ring-destructive/40 [&>svg]:pointer-events-none [&>svg]:size-3",
  {
    variants: {
      variant: {
        default: 'bg-primary text-primary-foreground [a&]:hover:bg-primary/90',
        secondary: 'bg-secondary text-secondary-foreground [a&]:hover:bg-secondary/90',
        destructive:
          'bg-destructive text-white dark:bg-destructive/60 dark: [a&]:hover:bg-destructive/90',
        outline: 'border-border text-foreground [a&]:hover:bg-accent [a&]:hover:text-accent-foreground',
        ghost: '[a&]:hover:bg-accent [a&]:hover:text-accent-foreground',
        link: 'text-primary underline-offset-4 [a&]:hover:underline',
      },
    },
    defaultVariants: { variant: 'default' },
  },
)

export type BadgeProps = ComponentProps<'span'> & VariantProps<typeof badgeVariants>

/**
 * Small status pill. Renders a `<span>`; the `[a&]:` variants still apply when
 * a caller nests it inside a link.
 *
 * @example <Badge variant="outline">Nv 10</Badge>
 */
export function Badge(props: BadgeProps) {
  const [local, rest] = splitProps(props, ['class', 'variant'])
  return (
    <span
      data-slot="badge"
      data-variant={local.variant ?? 'default'}
      class={cn(badgeVariants({ variant: local.variant }), local.class)}
      {...rest}
    />
  )
}

export { badgeVariants }
