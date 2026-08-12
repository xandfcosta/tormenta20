import type { QueryClient } from '@tanstack/solid-query'
import {
  firstErrorMessage,
  validateConsumeQuantity,
  validateEquipChange,
} from '@tormenta20/t20-data'
import { invalidateCharacterDependents } from '@/entities/character/character-cache'
import { characterQueryOptions } from '@/entities/character/queries'
import {
  type Character,
  type CharacterItem,
  type ConsumeItemInput,
  type ConsumeItemResult,
  type CreateItemInput,
  type UpdateItemInput,
  api,
} from '@/shared/api/api'
import { allCatalogItems, getCatalogItem } from '@/shared/lib/catalog-cache'
import { toSubmitFailure } from '@/shared/lib/form-errors'

/**
 * The optimistic transforms behind the four inventory writes, as pure
 * functions. The React version buried these inside `onMutate` closures, so the
 * only way to test "does a failed equip roll back correctly?" was to render the
 * whole panel and stub the network.
 */

/** A not-yet-persisted row. Negative id so it cannot collide with a real one. */
export function optimisticItem(input: CreateItemInput, tempId: number): CharacterItem {
  const catalog = input.catalogId
    ? allCatalogItems().find((c) => c.id === input.catalogId)
    : undefined
  return {
    id: tempId,
    catalogId: input.catalogId ?? null,
    name: input.name ?? catalog?.name ?? '...',
    quantity: input.quantity,
    slots: input.slots ?? catalog?.slots ?? 1,
    equipped: input.equipped ?? null,
    improvements: JSON.stringify(input.improvements ?? []),
    material: input.material ?? null,
  }
}

export function addItem(character: Character, item: CharacterItem): Character {
  return { ...character, items: [...character.items, item] }
}

/** Swaps the temporary row for the one the server persisted. */
export function settleAddedItem(
  character: Character,
  tempId: number,
  created: CharacterItem,
): Character {
  return {
    ...character,
    items: character.items.map((it) => (it.id === tempId ? created : it)),
  }
}

export function updateItem(
  character: Character,
  itemId: number,
  input: UpdateItemInput,
): Character {
  return {
    ...character,
    items: character.items.map((it) => {
      if (it.id !== itemId) return it
      const { improvements, ...rest } = input
      const merged: CharacterItem = { ...it, ...rest }
      // `improvements` is a string[] on the wire and a JSON string on the row.
      if (improvements !== undefined) merged.improvements = JSON.stringify(improvements)
      return merged
    }),
  }
}

/**
 * Replaces a row with the one the server persisted. Not `updateItem`: that
 * takes the WIRE shape (`improvements` as string[]), while this takes the
 * stored row (JSON string) — merging one as the other corrupts the field.
 */
export function settleUpdatedItem(character: Character, updated: CharacterItem): Character {
  return {
    ...character,
    items: character.items.map((it) => (it.id === updated.id ? updated : it)),
  }
}

export function removeItem(character: Character, itemId: number): Character {
  return { ...character, items: character.items.filter((it) => it.id !== itemId) }
}

/**
 * Spending one unit: decrement (or drop the row at 1→0) and apply the instant
 * PV/PM gain the same way the server clamps it. Scene/day effect rows stay
 * server-authoritative — they are reconciled from the delta.
 */
export function consumeItem(
  character: Character,
  itemId: number,
  input?: ConsumeItemInput,
): Character {
  const item = character.items.find((i) => i.id === itemId)
  if (!item) return character
  const items =
    item.quantity > 1
      ? character.items.map((i) => (i.id === itemId ? { ...i, quantity: i.quantity - 1 } : i))
      : character.items.filter((i) => i.id !== itemId)

  const instant = item.catalogId ? getCatalogItem(item.catalogId)?.consumable?.instant : undefined
  let { hpCurrent, mpCurrent } = character
  if (instant?.hp) {
    hpCurrent = Math.min(character.hpMax, hpCurrent + (input?.hpRolled ?? instant.hp.bonus ?? 0))
  }
  if (instant?.mp) {
    mpCurrent = Math.min(character.mpMax, mpCurrent + (input?.mpRolled ?? instant.mp.bonus ?? 0))
  }
  return { ...character, items, hpCurrent, mpCurrent }
}

/** Merges the server's consume delta: item count, any effect row, vitals. */
export function settleConsume(character: Character, delta: ConsumeItemResult): Character {
  const items = delta.item.removed
    ? character.items.filter((i) => i.id !== delta.item.id)
    : character.items.map((i) =>
        i.id === delta.item.id ? { ...i, quantity: delta.item.quantity } : i,
      )
  const activeEffects = delta.effect
    ? [...character.activeEffects.filter((e) => e.id !== delta.effect?.id), delta.effect]
    : character.activeEffects
  return { ...character, items, activeEffects, hpCurrent: delta.hpCurrent, mpCurrent: delta.mpCurrent }
}

/**
 * Why an inventory write was refused before it left the client, or null when it
 * is allowed. The rules live in `t20-data` and the server enforces the same
 * ones, so pre-checking is what makes the optimistic update safe: we only paint
 * a change the backend will accept.
 */
export function equipRefusal(
  character: Character,
  itemId: number,
  equipped: UpdateItemInput['equipped'],
): string | null {
  if (!equipped) return null
  const others = character.items.filter((it) => it.id !== itemId).map((it) => it.equipped)
  return firstErrorMessage(validateEquipChange(others, equipped)) ?? null
}

export function consumeRefusal(item: CharacterItem): string | null {
  return firstErrorMessage(validateConsumeQuantity(item.quantity)) ?? null
}

export type ItemActions = {
  add: (input: CreateItemInput) => Promise<void>
  /** Equip caps (≤4 vested / ≤2 hands) are checked before the request. */
  change: (itemId: number, input: UpdateItemInput) => Promise<void>
  remove: (itemId: number) => Promise<void>
  consume: (item: CharacterItem, input?: ConsumeItemInput) => Promise<void>
}

/** Raised instead of calling the backend when a domain rule already says no. */
export class ItemRefused extends Error {
  constructor(reason: string) {
    super(reason)
    this.name = 'ItemRefused'
  }
}

/**
 * What to show the player when an inventory write fails. An `ItemRefused`
 * already carries the rule's own sentence ("Máximo de 2 mãos ocupadas") and
 * must reach the screen verbatim — routing it through `toSubmitFailure` would
 * flatten the one useful message into "Erro inesperado".
 *
 * @example itemWriteMessage(failure, 'Não foi possível salvar o item.')
 */
export function itemWriteMessage(failure: unknown, fallback: string): string {
  if (failure instanceof ItemRefused) return failure.message
  return toSubmitFailure(failure).formError ?? fallback
}

/**
 * The four inventory writes, each optimistic and each rolling back to the exact
 * snapshot it took. Takes the query client as a parameter so a test drives it
 * with a plain `QueryClient`.
 *
 * @example const items = itemActions(queryClient, character.id)
 */
export function itemActions(queryClient: QueryClient, characterId: number): ItemActions {
  const queryKey = characterQueryOptions(characterId).queryKey
  const cached = () => queryClient.getQueryData<Character>(queryKey)

  async function optimistic(
    apply: (previous: Character) => Character,
    send: () => Promise<void>,
  ): Promise<void> {
    await queryClient.cancelQueries({ queryKey })
    const previous = cached()
    if (previous) queryClient.setQueryData<Character>(queryKey, apply(previous))
    try {
      await send()
    } catch (failure) {
      if (previous) queryClient.setQueryData<Character>(queryKey, previous)
      throw failure
    }
  }

  return {
    add: (input) => {
      const tempId = -Date.now()
      return optimistic(
        (previous) => addItem(previous, optimisticItem(input, tempId)),
        async () => {
          const created = await api.characters.addItem(characterId, input)
          queryClient.setQueryData<Character>(queryKey, (prev) =>
            prev ? settleAddedItem(prev, tempId, created) : prev,
          )
          invalidateCharacterDependents(queryClient, characterId)
        },
      )
    },

    change: async (itemId, input) => {
      const character = cached()
      const refusal = character ? equipRefusal(character, itemId, input.equipped) : null
      if (refusal) throw new ItemRefused(refusal)
      await optimistic(
        (previous) => updateItem(previous, itemId, input),
        async () => {
          const updated = await api.characters.updateItem(characterId, itemId, input)
          queryClient.setQueryData<Character>(queryKey, (prev) =>
            prev ? settleUpdatedItem(prev, updated) : prev,
          )
          invalidateCharacterDependents(queryClient, characterId)
        },
      )
    },

    remove: (itemId) =>
      optimistic(
        (previous) => removeItem(previous, itemId),
        async () => {
          await api.characters.deleteItem(characterId, itemId)
          invalidateCharacterDependents(queryClient, characterId)
        },
      ),

    consume: async (item, input) => {
      const refusal = consumeRefusal(item)
      if (refusal) throw new ItemRefused(refusal)
      await optimistic(
        (previous) => consumeItem(previous, item.id, input),
        async () => {
          const delta = await api.characters.consumeItem(characterId, item.id, input)
          queryClient.setQueryData<Character>(queryKey, (prev) =>
            prev ? settleConsume(prev, delta) : prev,
          )
          invalidateCharacterDependents(queryClient, characterId)
        },
      )
    },
  }
}
