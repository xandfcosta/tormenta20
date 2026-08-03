import { FLAG_ACTIVATIONS } from '@tormenta20/t20-data'
import type { Character } from '@/shared/api/api'
import { useAllConditionals } from '@/entities/character/derived'
import { parseActiveConditions } from './conditions-section'
import { cn } from '@/shared/lib/utils'

/**
 * Numeric pill next to the "Efeitos" tab. Counts what the tab SHOWS as
 * units, not raw engine entries: active book conditions + running
 * scene/day effects + active stances (one per flag — Fúria's 8 tier
 * modifiers are one thing) + switched-on situational groups. The old
 * raw-entry count read "11" for a lone Fúria + two toggles.
 */
export function EffectsCountBadge({ character }: { character: Character }) {
  const all = useAllConditionals(character)
  const conditions = parseActiveConditions(character.activeConditions).length
  const consumableActive = (character.activeEffects ?? []).length

  const stanceFlags = new Set<string>()
  const situationalGroups = new Set<string>()
  for (const e of all) {
    if (!e.active) continue
    const flag = e.effect.flag
    if (flag && FLAG_ACTIVATIONS[flag]) stanceFlags.add(flag)
    else situationalGroups.add(flag ?? e.id)
  }
  const total =
    conditions + consumableActive + stanceFlags.size + situationalGroups.size
  if (total === 0 && all.length === 0) return null
  return (
    <span
      className={cn(
        'ml-0.5 inline-flex h-4 min-w-[1rem] items-center justify-center rounded-full px-1 text-[10px] font-bold',
        total > 0
          ? 'bg-primary text-primary-foreground'
          : 'bg-muted text-muted-foreground',
      )}
      aria-label={`${total} efeitos ativos`}
    >
      {total}
    </span>
  )
}
