import type { ReactNode } from 'react'
import { cn } from '@/shared/lib/utils'

/**
 * A bordered stat tile: a small uppercase label over a large mono value.
 * Shared by the character-select stat triple and the creation wizard's
 * class/attribute previews so both read as the same "the numbers are the
 * character" language. `dim` fades a not-applicable cell (e.g. 0-max PM).
 */
export function StatCell({
  label,
  dim,
  children,
}: {
  label: string
  dim?: boolean
  children: ReactNode
}) {
  return (
    <div
      className={cn(
        'flex flex-col items-center rounded-md border border-border py-2',
        dim && 'opacity-50',
      )}
    >
      <span className="text-[10px] uppercase tracking-widest text-muted-foreground">
        {label}
      </span>
      <span className="font-mono text-3xl font-semibold tabular-nums">
        {children}
      </span>
    </div>
  )
}
