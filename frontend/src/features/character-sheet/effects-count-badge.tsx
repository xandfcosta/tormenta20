import type { Character } from '@/shared/api/api'
import { useAllConditionals } from '@/entities/character/derived'
import { cn } from '@/shared/lib/utils'

/**
 * Small numeric pill shown next to the "Efeitos" tab trigger. Counts
 * both consumable-driven `activeEffects` and conditional entries that
 * are currently toggled on. Renders nothing when the character has no
 * effects at all (avoids a zero-badge next to a mostly-empty tab).
 */
export function EffectsCountBadge({ character }: { character: Character }) {
  const all = useAllConditionals(character)
  const condActive = all.filter((e) => e.active).length
  const consumableActive = (character.activeEffects ?? []).length
  const total = condActive + consumableActive
  if (all.length === 0 && consumableActive === 0) return null
  return (
    <span
      className={cn(
        'ml-0.5 inline-flex h-4 min-w-[1rem] items-center justify-center rounded-full px-1 text-[10px] font-bold',
        total > 0
          ? 'bg-amber-700 text-amber-50 dark:bg-amber-500 dark:text-zinc-900'
          : 'bg-zinc-300 text-zinc-700 dark:bg-zinc-700 dark:text-zinc-300',
      )}
      aria-label={`${total} efeitos ativos`}
    >
      {total}
    </span>
  )
}
