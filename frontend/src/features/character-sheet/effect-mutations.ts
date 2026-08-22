import type { QueryClient } from '@tanstack/solid-query'
import type { ConditionId } from '@/shared/api/catalog-types'
import { invalidateCharacterDependents } from '@/entities/character/character-cache'
import { characterQueryOptions } from '@/entities/character/queries'
import { createCharacterWrite } from './character-write'
import { type Character, api } from '@/shared/api/api'
import { toast } from '@/shared/ui/sonner'

/**
 * Delta merge for the end-scene/end-day answers: drop the cached effects whose
 * scope the server just expired, so the sheet updates without a refetch.
 *
 * Which scopes went is the SERVER's word, not the button's — ending the day
 * ends the running scene too, and reading only "day" would leave cleared scene
 * buffs painted on the sheet.
 *
 * @example dropClearedScopes(character, ['scene'])
 */
export function dropClearedScopes(
  character: Character,
  clearedScopes: readonly string[],
): Character {
  return {
    ...character,
    activeEffects: character.activeEffects.filter((e) => !clearedScopes.includes(e.scope)),
  }
}

export type EffectActions = {
  /** Removes one running effect. Confirmation-first: see the comment inside. */
  remove: (effectId: number) => Promise<void>
  endScene: () => Promise<void>
  endDay: () => Promise<void>
}

/**
 * Mutations of the "Efeitos ativos" list. Stateless between calls, so it may be
 * built per event (`effectActions(queryClient, id)`) — unlike
 * `createVitalActions`, which owns a debounce timer.
 *
 * @example await effectActions(queryClient, character.id).endScene()
 */
export function effectActions(queryClient: QueryClient, characterId: number): EffectActions {
  const queryKey = characterQueryOptions(characterId).queryKey

  const merge = (transform: (prev: Character) => Character) => {
    queryClient.setQueryData<Character>(queryKey, (prev) => (prev ? transform(prev) : prev))
    invalidateCharacterDependents(queryClient, characterId)
  }

  const clearScope = async (
    expire: (id: number) => Promise<{ clearedScopes: ('scene' | 'day')[] }>,
    failureMessage: string,
  ) => {
    try {
      const { clearedScopes } = await expire(characterId)
      merge((prev) => dropClearedScopes(prev, clearedScopes))
    } catch (failure) {
      toast.error(failureMessage)
      throw failure
    }
  }

  return {
    // NOT optimistic: an effect the server no longer has answers 404, and a
    // row that vanished and came back reads as a bug at the table. It only
    // leaves the list once the server confirms which id went.
    remove: async (effectId) => {
      try {
        const { id } = await api.characters.removeActiveEffect(characterId, effectId)
        merge((prev) => ({
          ...prev,
          activeEffects: prev.activeEffects.filter((e) => e.id !== id),
        }))
      } catch (failure) {
        toast.error('Falha ao remover o efeito — ele continua ativo na ficha')
        throw failure
      }
    },

    endScene: () =>
      clearScope(api.characters.endScene, 'Falha ao encerrar a cena — os efeitos continuam ativos'),

    endDay: () =>
      clearScope(api.characters.endDay, 'Falha ao encerrar o dia — os efeitos continuam ativos'),
  }
}

export type ConditionActions = {
  /** Replaces the whole set; the server answers the stored blob. */
  set: (conditions: ConditionId[]) => Promise<void>
}

/**
 * Book conditions (p394-395). Optimistic: a condition applied mid-fight has to
 * land on the numbers at once — the sheet recomputes off `activeConditions`
 * (ALE-28), so waiting for the round-trip would show stale defense.
 *
 * Rolls back and RETHROWS without saying anything: telling the player is the
 * caller's job, because where the failure has to be said depends on where the
 * action was taken (ALE-216). Applying now happens inside a dialog, and a toast
 * fired from an open modal is never announced — `createConditionEditing` toasts
 * for the pickers outside one, the dialog shows `DialogInlineError`.
 *
 * @example await conditionActions(queryClient, character.id).set(['caido'])
 */
export function conditionActions(
  queryClient: QueryClient,
  characterId: number,
): ConditionActions {
  const queryKey = characterQueryOptions(characterId).queryKey
  const characterWrite = createCharacterWrite(queryClient, characterId)

  return {
    set: (conditions) =>
      characterWrite(
        (previous) => ({ ...previous, activeConditions: JSON.stringify(conditions) }),
        async () => {
          const delta = await api.characters.updateConditions(characterId, conditions)
          queryClient.setQueryData<Character>(queryKey, (prev) =>
            prev ? { ...prev, activeConditions: delta.activeConditions } : prev,
          )
          invalidateCharacterDependents(queryClient, characterId)
        },
      ),
  }
}
