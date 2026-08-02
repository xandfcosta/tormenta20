import { Shield } from 'lucide-react'
import { useMemo } from 'react'
import type { Character } from '@/shared/api/api'
import { cn } from '@/shared/lib/utils'
import {
  characterEffects,
  defenseTotal,
} from '@/entities/character/derived'

/**
 * Compact DEF chip for viewports where the stats cluster is hidden (<lg):
 * "does 17 hit?" must be answerable without a tab switch (audit P1). Read-only
 * — the full breakdown lives in the desktop Defesa box / Vitais tab.
 */
export function MobileDefChip({
  character,
  className,
}: {
  character: Character
  className?: string
}) {
  const defense = useMemo(
    () => defenseTotal(character, characterEffects(character)).total,
    [character],
  )
  return (
    <span
      className={cn(
        'flex items-center gap-1 rounded-md border border-red-800/50 px-1.5 py-0.5',
        'font-mono text-sm font-bold text-red-800 dark:border-red-500/40 dark:text-red-200',
        className,
      )}
      title="Defesa"
      aria-label={`Defesa ${defense}`}
    >
      <Shield className="size-3.5" />
      {defense}
    </span>
  )
}
