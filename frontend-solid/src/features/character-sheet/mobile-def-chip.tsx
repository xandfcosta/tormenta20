import { Shield } from 'lucide-solid'
import { computedSheetFor } from '@/entities/character/computed-sheet'
import type { Character } from '@/shared/api/api'
import { cn } from '@/shared/lib/utils'

/**
 * Compact DEF chip for the viewports where the stats cluster is hidden (<md):
 * "does a 17 hit me?" has to be answerable without switching blocks. Read-only
 * — the full breakdown lives in the desktop Defesa box.
 */
export function MobileDefChip(props: {
  character: Character
  activeConditionals: ReadonlySet<string>
  class?: string
}) {
  const defense = () => computedSheetFor(props.character, props.activeConditionals).defense.total
  return (
    // role="img": a bare span is `generic`, which does not take an accessible
    // name — the shield glyph plus the number only read as "Defesa 17" here.
    <span
      role="img"
      class={cn(
        'flex items-center gap-1 rounded-md border border-destructive/50 px-1.5 py-0.5',
        'font-mono text-sm font-bold text-destructive',
        props.class,
      )}
      title="Defesa"
      aria-label={`Defesa ${defense()}`}
    >
      <Shield aria-hidden="true" class="size-3.5" />
      {defense()}
    </span>
  )
}
