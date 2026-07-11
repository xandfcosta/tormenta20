import * as React from 'react'
import { cn } from '@/shared/lib/utils'

/**
 * SectionHeading — h2-tier heading with an optional Tormenta glyph
 * inline. Two variants come with distinctive inline SVGs (not
 * lucide-generic): `aharadak` = tentacle sigil for corrupt / occult
 * sections (spells, active effects); `kallyadranoch` = dragon-claw
 * for combat / offensive sections. `default` renders the heading
 * without a glyph — used everywhere else so the marks stay meaningful.
 *
 * The glyphs are ~16px inline SVG so they follow currentColor. Text
 * uses the Redaction display face — deliberately degraded to sell
 * the diegetic "worn manuscript" tone without dropping into cliché
 * scrollwork.
 */
type SectionHeadingVariant = 'default' | 'aharadak' | 'kallyadranoch'

type SectionHeadingProps = React.ComponentProps<'h2'> & {
  variant?: SectionHeadingVariant
  as?: 'h1' | 'h2' | 'h3'
}

function AharadakGlyph({ className }: { className?: string }) {
  return (
    <svg
      viewBox="0 0 24 24"
      aria-hidden="true"
      className={cn('size-5 shrink-0', className)}
      fill="none"
      stroke="currentColor"
      strokeWidth="1.4"
      strokeLinecap="round"
      strokeLinejoin="round"
    >
      <circle cx="12" cy="12" r="2.4" />
      <path d="M12 9.6 C12 6.5 10 5 8 6.5 C6 8 6.5 10.5 8.5 11" />
      <path d="M14.4 12 C17 12 18.5 10 17.5 8 C16.5 6 14 6.5 13.5 8.5" />
      <path d="M12 14.4 C12 17.5 14 19 16 17.5 C18 16 17.5 13.5 15.5 13" />
      <path d="M9.6 12 C7 12 5.5 14 6.5 16 C7.5 18 10 17.5 10.5 15.5" />
    </svg>
  )
}

function KallyadranochGlyph({ className }: { className?: string }) {
  return (
    <svg
      viewBox="0 0 24 24"
      aria-hidden="true"
      className={cn('size-5 shrink-0', className)}
      fill="none"
      stroke="currentColor"
      strokeWidth="1.4"
      strokeLinecap="round"
      strokeLinejoin="round"
    >
      <path d="M4 6 C6 8 8 9 10 9" />
      <path d="M4 6 L7 5.5" />
      <path d="M8 12 C10 13 12 13.5 14 13.5" />
      <path d="M8 12 L11 11" />
      <path d="M12 18 C14 18 16 17 18 15" />
      <path d="M12 18 L15 18.5" />
    </svg>
  )
}

function SectionHeading({
  className,
  children,
  variant = 'default',
  as = 'h2',
  ...props
}: SectionHeadingProps) {
  const Tag = as
  return (
    <Tag
      data-slot="section-heading"
      data-variant={variant}
      className={cn(
        'font-display flex items-center gap-2 text-xl tracking-wide text-foreground',
        as === 'h1' && 'text-3xl',
        as === 'h3' && 'text-lg',
        className,
      )}
      {...props}
    >
      {variant === 'aharadak' && (
        <AharadakGlyph className="text-[color:var(--primary)]" />
      )}
      {variant === 'kallyadranoch' && (
        <KallyadranochGlyph className="text-[color:var(--primary)]" />
      )}
      <span>{children}</span>
    </Tag>
  )
}

export { SectionHeading }
export type { SectionHeadingVariant }
