import type { QueryClient } from '@tanstack/solid-query'
import { type ProficiencyEntry, characterProficiencies } from '@tormenta20/t20-data'
import { invalidateCharacterDependents } from '@/entities/character/character-cache'
import { characterQueryOptions } from '@/entities/character/queries'
import { type Character, api } from '@/shared/api/api'
import { toast } from '@/shared/ui/sonner'

/**
 * The proficiencies the character owns. The blob is a bare `string[]` of
 * category ids; anything absent is not granted, and a corrupt blob means none
 * rather than an exception on a sheet the player is trying to read.
 *
 * @example ownedProficiencies({ proficiencies: '["armas-simples"]' })
 */
export function ownedProficiencies(character: Character): ReadonlySet<string> {
  try {
    const parsed = JSON.parse(character.proficiencies)
    if (!Array.isArray(parsed)) return new Set()
    return new Set(parsed.filter((x): x is string => typeof x === 'string'))
  } catch {
    return new Set()
  }
}

/** Every proficiency the character's classes have an opinion about. */
export function proficiencyCatalog(character: Character): ProficiencyEntry[] {
  return characterProficiencies(character.classes.map((c) => c.className))
}

/** Weapons in one group, armor + shields in the other — how the book reads. */
export function groupProficiencies(entries: ProficiencyEntry[]): {
  weapons: ProficiencyEntry[]
  armors: ProficiencyEntry[]
} {
  return {
    weapons: entries.filter((e) => e.category.startsWith('armas-')),
    armors: entries.filter(
      (e) => e.category.startsWith('armaduras-') || e.category === 'escudos',
    ),
  }
}

/** Toggling is a set operation, kept pure so the panel never rebuilds it. */
export function toggleProficiency(owned: ReadonlySet<string>, category: string): string[] {
  const next = new Set(owned)
  if (next.has(category)) next.delete(category)
  else next.add(category)
  return [...next]
}

/** What the classes grant on their own — the "restaurar padrão" target. */
export function classDefaults(character: Character): string[] {
  return proficiencyCatalog(character)
    .filter((entry) => entry.granted)
    .map((entry) => entry.category)
}

export type ProficiencyActions = {
  /** Replaces the whole set; the server answers the stored blob. */
  set: (categories: string[]) => Promise<void>
}

export function proficiencyActions(
  queryClient: QueryClient,
  characterId: number,
): ProficiencyActions {
  const queryKey = characterQueryOptions(characterId).queryKey

  return {
    set: async (categories) => {
      await queryClient.cancelQueries({ queryKey })
      const previous = queryClient.getQueryData<Character>(queryKey)
      queryClient.setQueryData<Character>(queryKey, (prev) =>
        prev ? { ...prev, proficiencies: JSON.stringify(categories) } : prev,
      )
      try {
        const delta = await api.characters.updateProficiencies(characterId, {
          proficiencies: categories,
        })
        queryClient.setQueryData<Character>(queryKey, (prev) =>
          prev ? { ...prev, proficiencies: delta.proficiencies } : prev,
        )
        invalidateCharacterDependents(queryClient, characterId)
      } catch (failure) {
        if (previous) queryClient.setQueryData(queryKey, previous)
        toast.error('Falha ao salvar proficiências — a ficha voltou ao valor anterior')
        throw failure
      }
    },
  }
}
