import type { DisplayFact } from '@tormenta20/t20-data'
import { cn } from '@/shared/lib/utils'

/**
 * Renders display-only mechanical facts (RD, immunities, senses, movement,
 * action economy) as reference chips. These are shown, never computed — the
 * companion-app affordance for effects the engine can't model.
 */
export function FactChips({
  facts,
  className,
}: {
  facts: DisplayFact[]
  className?: string
}) {
  if (facts.length === 0) return null
  return (
    <ul className={cn('flex flex-wrap gap-1', className)}>
      {facts.map((f) => (
        <li
          key={`${f.category}:${f.text}`}
          className="rounded border border-border bg-muted px-1.5 py-0.5 text-[10px] leading-tight text-foreground"
        >
          {f.text}
        </li>
      ))}
    </ul>
  )
}
