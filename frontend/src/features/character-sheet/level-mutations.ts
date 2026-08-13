import type { QueryClient } from '@tanstack/solid-query'
import { characterEffects } from '@/entities/character/derived'
import { optimisticLevelVitals } from '@/entities/character/level-vitals'
import { characterQueryOptions } from '@/entities/character/queries'
import { invalidateCharacterDependents } from '@/entities/character/character-cache'
import { type Character, type ClassLevelResult, api } from '@/shared/api/api'
import { toast } from '@/shared/ui/sonner'

/** T20 caps a character at 20th level; below 1 the class would not exist. */
export const MIN_LEVEL = 1
export const MAX_LEVEL = 20

/**
 * Which classes can still take the step. Levelling DOWN a 1st-level class would
 * delete it, which the stepper does not do — the class list is edited elsewhere.
 *
 * @example eligibleClasses(bardo3Guerreiro1, 'down') // só o Bardo
 */
export function eligibleClasses(character: Character, direction: 'up' | 'down') {
  return character.classes.filter((entry) =>
    direction === 'up' ? entry.level < MAX_LEVEL : entry.level > 1,
  )
}

/**
 * The character as it will look one class-level later — new class list, new
 * total, and the PV/PM the pools grow (or shrink) to. Pure so the optimistic
 * paint is testable without a component.
 *
 * PV/PM max are DERIVED from class levels, so they move WITH the step: the
 * server does the same, and not mirroring it left the bars on stale pools.
 */
export function bumpClassLevel(
  character: Character,
  className: string,
  delta: number,
): Character {
  const classes = character.classes.map((entry) =>
    entry.className === className ? { ...entry, level: entry.level + delta } : entry,
  )
  const level = classes.reduce((total, entry) => total + entry.level, 0)
  const vitals = optimisticLevelVitals(character, characterEffects(character), classes)
  return { ...character, classes, level, ...vitals }
}

/** Merges the server's authoritative level + pools over the optimistic guess. */
export function settleClassLevel(character: Character, result: ClassLevelResult): Character {
  return { ...character, level: result.level, classes: result.classes, ...result.vitals }
}

export type LevelActions = {
  bump: (className: string, delta: number) => Promise<void>
}

/**
 * The one level write, optimistic and rolling back to the snapshot it took.
 *
 * @example levelActions(queryClient, character.id).bump('Guerreiro', 1)
 */
export function levelActions(queryClient: QueryClient, characterId: number): LevelActions {
  const queryKey = characterQueryOptions(characterId).queryKey

  return {
    bump: async (className, delta) => {
      await queryClient.cancelQueries({ queryKey })
      const previous = queryClient.getQueryData<Character>(queryKey)
      if (!previous) return
      const entry = previous.classes.find((c) => c.className === className)
      if (!entry) throw new Error(`bump: classe ausente no personagem — ${className}`)

      queryClient.setQueryData<Character>(queryKey, bumpClassLevel(previous, className, delta))
      try {
        const result = await api.characters.updateClassLevel(characterId, {
          className,
          level: entry.level + delta,
        })
        queryClient.setQueryData<Character>(queryKey, (prev) =>
          prev ? settleClassLevel(prev, result) : prev,
        )
        invalidateCharacterDependents(queryClient, characterId)
      } catch (failure) {
        queryClient.setQueryData(queryKey, previous)
        toast.error('Falha ao mudar o nível — a ficha voltou ao valor anterior')
        throw failure
      }
    },
  }
}
