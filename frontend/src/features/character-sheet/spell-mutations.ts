import type { QueryClient } from '@tanstack/solid-query'
import { invalidateCharacterDependents } from '@/entities/character/character-cache'
import { characterQueryOptions } from '@/entities/character/queries'
import { createCharacterWrite } from './character-write'
import {
  type Character,
  type CharacterSpell,
  type SpellAugmentPick,
  api,
} from '@/shared/api/api'
import { toast } from '@/shared/ui/sonner'

export type SpellActions = {
  /** Adds the spell to the grimoire, unprepared. */
  learn: (catalogSpellId: string) => Promise<void>
  unlearn: (catalogSpellId: string) => Promise<void>
  setPrepared: (catalogSpellId: string, prepared: boolean) => Promise<void>
  /** Spends the PM. NOT optimistic — see the comment on the method. */
  cast: (catalogSpellId: string, augments: SpellAugmentPick[]) => Promise<void>
}

/**
 * The grimoire's four writes. Stateless between calls, so it may be built per
 * event.
 *
 * Learn/unlearn/prepare are optimistic: the caller pre-validates with the
 * shared t20-data rules, so the server's answer is predictable and the row can
 * flip instantly. Casting is not — the server owns limits the client cannot
 * fully know.
 *
 * @example await spellActions(queryClient, character.id).setPrepared(id, true)
 */
export function spellActions(queryClient: QueryClient, characterId: number): SpellActions {
  const queryKey = characterQueryOptions(characterId).queryKey
  const characterWrite = createCharacterWrite(queryClient, characterId)

  const patchSpells = (transform: (spells: CharacterSpell[]) => CharacterSpell[]) =>
    queryClient.setQueryData<Character>(queryKey, (prev) =>
      prev ? { ...prev, spells: transform(prev.spells) } : prev,
    )

  /** Runs an optimistic spellbook edit, rolling the whole character back if the
   *  server disagrees with the prediction. The cancel/snapshot/rollback core is
   *  shared (`createCharacterWrite`); what stays local is the spellbook slice
   *  and this screen's failure toast. */
  const optimistic = async (
    paint: (spells: CharacterSpell[]) => CharacterSpell[],
    write: () => Promise<CharacterSpell[] | null>,
    failureMessage: string,
  ) => {
    try {
      await characterWrite(
        (previous) => ({ ...previous, spells: paint(previous.spells) }),
        async () => {
          const settled = await write()
          if (settled) patchSpells(() => settled)
          invalidateCharacterDependents(queryClient, characterId)
        },
      )
    } catch (failure) {
      toast.error(failureMessage)
      throw failure
    }
  }

  return {
    learn: (catalogSpellId) => {
      // Temporary id until the server assigns the real row — swapped on settle,
      // because a later prepare/unlearn would otherwise target a row that does
      // not exist server-side.
      const optimisticRow: CharacterSpell = {
        id: -1,
        catalogSpellId,
        prepared: false,
        learnedAt: '',
      }
      return optimistic(
        (spells) => [...spells, optimisticRow],
        async () => {
          const row = await api.characters.learnSpell(characterId, catalogSpellId)
          const current = queryClient.getQueryData<Character>(queryKey)?.spells ?? []
          return current.map((spell) =>
            spell.catalogSpellId === catalogSpellId ? row : spell,
          )
        },
        'Falha ao aprender a magia — o grimório voltou ao que estava',
      )
    },

    unlearn: (catalogSpellId) =>
      optimistic(
        (spells) => spells.filter((spell) => spell.catalogSpellId !== catalogSpellId),
        async () => {
          await api.characters.unlearnSpell(characterId, catalogSpellId)
          return null
        },
        'Falha ao esquecer a magia — ela continua no grimório',
      ),

    setPrepared: (catalogSpellId, prepared) =>
      optimistic(
        (spells) =>
          spells.map((spell) =>
            spell.catalogSpellId === catalogSpellId ? { ...spell, prepared } : spell,
          ),
        async () => {
          await api.characters.setSpellPrepared(characterId, catalogSpellId, prepared)
          return null
        },
        'Falha ao mudar a preparação — a magia voltou ao estado anterior',
      ),

    /**
     * NOT optimistic: the server re-validates prepared, augments and the
     * per-spell PM limit and answers 400 when any of them fails. Painting the
     * PM first would show a cost the character never paid.
     */
    cast: async (catalogSpellId, augments) => {
      const result = await api.characters.castSpell(characterId, catalogSpellId, augments)
      queryClient.setQueryData<Character>(queryKey, (prev) =>
        prev
          ? {
              ...prev,
              mpCurrent: result.mpCurrent,
              // Catalysts the cast consumed (a scene effect the server deleted).
              activeEffects: prev.activeEffects.filter(
                (effect) => !result.removedEffectIds.includes(effect.id),
              ),
            }
          : prev,
      )
      invalidateCharacterDependents(queryClient, characterId)
    },
  }
}
