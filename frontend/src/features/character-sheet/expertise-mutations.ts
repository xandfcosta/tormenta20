import type { QueryClient } from '@tanstack/solid-query'
import { invalidateCharacterDependents } from '@/entities/character/character-cache'
import { characterQueryOptions } from '@/entities/character/queries'
import {
  type AttributeKey,
  type Character,
  type CharacterExpertise,
  api,
} from '@/shared/api/api'

export type ExpertisePatch = { attribute?: AttributeKey; trained?: boolean }

/**
 * Applies an expertise patch to a cached character. Pure so the optimistic
 * update and the rollback are testable without a component or a network — the
 * React version inlined this inside `onMutate` closures, where the only way to
 * check it was to render the panel.
 */
export function patchExpertise(
  character: Character,
  name: string,
  patch: ExpertisePatch,
): Character {
  return {
    ...character,
    expertises: character.expertises.map((e) => (e.name === name ? { ...e, ...patch } : e)),
  }
}

/** Adds a custom "ofício", already trained (that is the point of inventing one). */
export function addCustomExpertise(
  character: Character,
  input: { name: string; attribute: AttributeKey },
): Character {
  return {
    ...character,
    expertises: [
      ...character.expertises,
      { name: input.name.trim(), attribute: input.attribute, trained: true, custom: true },
    ],
  }
}

export function removeExpertise(character: Character, name: string): Character {
  return {
    ...character,
    expertises: character.expertises.filter((e) => e.name !== name),
  }
}

/** Replaces one entry with the row the server confirmed. */
export function settleExpertise(
  character: Character,
  updated: CharacterExpertise,
): Character {
  return {
    ...character,
    expertises: character.expertises.map((e) => (e.name === updated.name ? updated : e)),
  }
}

export type ExpertiseActions = {
  /** Train/untrain, or rekey to another attribute. */
  update: (name: string, patch: ExpertisePatch) => Promise<void>
  addCustom: (input: { name: string; attribute: AttributeKey }) => Promise<void>
  remove: (name: string) => Promise<void>
}

/**
 * The three expertise writes, each optimistic and each rolling back to the
 * exact snapshot it took. Takes the query client as a parameter instead of
 * reading it from context, so a test drives it with a plain `QueryClient`.
 *
 * @example const actions = expertiseActions(queryClient, character.id)
 */
export function expertiseActions(
  queryClient: QueryClient,
  characterId: number,
): ExpertiseActions {
  const queryKey = characterQueryOptions(characterId).queryKey

  /** Snapshot → optimistic write → server call → rollback on failure. */
  async function optimistic(
    apply: (previous: Character) => Character,
    send: () => Promise<void>,
  ): Promise<void> {
    await queryClient.cancelQueries({ queryKey })
    const previous = queryClient.getQueryData<Character>(queryKey)
    if (previous) queryClient.setQueryData<Character>(queryKey, apply(previous))
    try {
      await send()
    } catch (failure) {
      if (previous) queryClient.setQueryData<Character>(queryKey, previous)
      throw failure
    }
  }

  return {
    update: (name, patch) =>
      optimistic(
        (previous) => patchExpertise(previous, name, patch),
        async () => {
          const updated = await api.characters.updateExpertise(characterId, { name, ...patch })
          queryClient.setQueryData<Character>(queryKey, (prev) =>
            prev ? settleExpertise(prev, updated) : prev,
          )
          // NOT the base key: it was just written with the server's own answer,
          // and invalidating would race a refetch against fresh data.
          invalidateCharacterDependents(queryClient, characterId)
        },
      ),

    addCustom: (input) =>
      optimistic(
        (previous) => addCustomExpertise(previous, input),
        async () => {
          await api.characters.addExpertise(characterId, input)
          await queryClient.invalidateQueries({ queryKey })
          invalidateCharacterDependents(queryClient, characterId)
        },
      ),

    remove: (name) =>
      optimistic(
        (previous) => removeExpertise(previous, name),
        async () => {
          await api.characters.deleteExpertise(characterId, name)
          invalidateCharacterDependents(queryClient, characterId)
        },
      ),
  }
}
