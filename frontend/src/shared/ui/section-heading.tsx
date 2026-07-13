import * as React from 'react'
import { cn } from '@/shared/lib/utils'

/**
 * SectionHeading — a plain heading. The decorative Tormenta glyphs + Cinzel
 * display face were stripped (functional focus); the `variant` prop is kept
 * so existing callers compile but no longer changes the look.
 */
type SectionHeadingVariant = 'default' | 'aharadak' | 'kallyadranoch'

type SectionHeadingProps = React.ComponentProps<'h2'> & {
  variant?: SectionHeadingVariant
  as?: 'h1' | 'h2' | 'h3'
}

function SectionHeading({
  className,
  children,
  variant: _variant = 'default',
  as = 'h2',
  ...props
}: SectionHeadingProps) {
  const Tag = as
  return (
    <Tag
      data-slot="section-heading"
      className={cn(
        'font-semibold tracking-tight text-foreground',
        as === 'h1' && 'text-2xl',
        as === 'h2' && 'text-xl',
        as === 'h3' && 'text-lg',
        className,
      )}
      {...props}
    >
      {children}
    </Tag>
  )
}

export { SectionHeading }
export type { SectionHeadingVariant }
