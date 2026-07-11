import * as React from 'react'
import { cva, type VariantProps } from 'class-variance-authority'

import { cn } from '@/shared/lib/utils'

/**
 * PageChrome — the outermost wrapper for every top-level route.
 * Applies the design-system container width + padding + optional
 * ornamental border. Noise texture is already global (index.css
 * `body::before`), so PageChrome only handles layout.
 *
 * `bordered` opts into the ornamental frame — reserved for hero-tier
 * routes (landing, character-sheet, session tracker). Lists/tables
 * skip it to avoid nested-frame clutter.
 */
const pageChromeVariants = cva(
  'relative mx-auto w-full',
  {
    variants: {
      width: {
        compact: 'max-w-3xl',
        default: 'max-w-6xl',
        wide: 'max-w-[90rem]',
        full: 'max-w-none',
      },
      padded: {
        true: 'px-4 py-6 sm:px-6 md:px-8 md:py-8',
        false: '',
      },
      bordered: {
        true: 'rounded-lg border border-border/60 bg-card/50 shadow-[0_1px_0_rgba(0,0,0,0.04),0_20px_60px_-30px_rgba(0,0,0,0.35)]',
        false: '',
      },
    },
    defaultVariants: {
      width: 'default',
      padded: true,
      bordered: false,
    },
  },
)

type PageChromeProps = React.ComponentProps<'section'> &
  VariantProps<typeof pageChromeVariants>

function PageChrome({
  className,
  width,
  padded,
  bordered,
  ...props
}: PageChromeProps) {
  return (
    <section
      data-slot="page-chrome"
      data-width={width ?? 'default'}
      data-bordered={bordered ? 'true' : 'false'}
      className={cn(
        pageChromeVariants({ width, padded, bordered }),
        className,
      )}
      {...props}
    />
  )
}

export { PageChrome, pageChromeVariants }
