import type { Character } from '@/shared/api/api'
import { computePendencias } from './pendencias'

/**
 * Numeric pill on the "Habilidades" tab trigger — counts unmade required
 * choices (race variants, origin benefits, class powers/paths). Renders
 * nothing when there's nothing outstanding, matching the Efeitos badge.
 */
export function AbilitiesPendingBadge({ character }: { character: Character }) {
  const count = computePendencias(character).length
  if (count === 0) return null
  return (
    <span
      className="ml-0.5 inline-flex h-4 min-w-[1rem] items-center justify-center rounded-full bg-destructive px-1 text-[10px] font-bold text-white"
      aria-label={`${count} escolhas pendentes`}
    >
      {count}
    </span>
  )
}
