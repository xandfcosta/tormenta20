import type { QueryClient } from '@tanstack/solid-query'
import type { ClassChoices } from '@tormenta20/t20-data'
import { invalidateCharacterDependents } from '@/entities/character/character-cache'
import { characterQueryOptions } from '@/entities/character/queries'
import { createCharacterWrite } from './character-write'
import {
  type Character,
  type UpdateAbilityChoicesInput,
  api,
} from '@/shared/api/api'
import { toast } from '@/shared/ui/sonner'

/** The character fields `PATCH :id/abilities` can write, all JSON blobs. */
type ChoiceField = keyof UpdateAbilityChoicesInput & keyof Character

export type ChoiceActions = {
  /** Elective class powers the character has taken. */
  setClassPowers: (ids: string[]) => Promise<void>
  setOriginChoices: (ids: string[]) => Promise<void>
  setRaceAbilityChoices: (ids: string[]) => Promise<void>
  /** Devoto/caminho per class name. */
  setClassChoices: (choices: ClassChoices) => Promise<void>
  /** Sub-choices of a power, keyed by power id. */
  setPowerChoices: (choices: Record<string, string[]>) => Promise<void>
}

/**
 * Writes to the five ability-choice blobs. In React each of the four panels
 * carried its own copy of this mutation (~25 identical lines each); here the
 * optimism lives once and the panels only say WHAT changed.
 *
 * Optimistic because a chosen power immediately changes the sheet's numbers —
 * waiting for the round-trip would show a stale Defesa next to a checked box.
 *
 * @example await choiceActions(queryClient, character.id).setClassPowers(ids)
 */
export function choiceActions(queryClient: QueryClient, characterId: number): ChoiceActions {
  const queryKey = characterQueryOptions(characterId).queryKey
  const characterWrite = createCharacterWrite(queryClient, characterId)

  /**
   * One optimistic blob write. `field` names both the request field and the
   * cached character's column — the endpoint patches a subset and echoes back
   * ONLY what it wrote, so the answer is merged field by field rather than
   * spread whole (which would blank the blobs it never touched).
   */
  const write = async (field: ChoiceField, input: UpdateAbilityChoicesInput) => {
    try {
      await characterWrite(
        (previous) => ({ ...previous, [field]: JSON.stringify(input[field]) }),
        async () => {
          const delta = await api.characters.updateAbilityChoices(characterId, input)
          const stored = delta[field]
          if (stored !== undefined) {
            queryClient.setQueryData<Character>(queryKey, (prev) =>
              prev ? { ...prev, [field]: stored } : prev,
            )
          }
          invalidateCharacterDependents(queryClient, characterId)
        },
      )
    } catch (failure) {
      toast.error('Falha ao salvar a escolha — a ficha voltou ao valor anterior')
      throw failure
    }
  }

  return {
    setClassPowers: (ids) => write('classPowers', { classPowers: ids }),
    setOriginChoices: (ids) => write('originChoices', { originChoices: ids }),
    setRaceAbilityChoices: (ids) => write('raceAbilityChoices', { raceAbilityChoices: ids }),
    setClassChoices: (choices) => write('classChoices', { classChoices: choices }),
    setPowerChoices: (choices) => write('powerChoices', { powerChoices: choices }),
  }
}
