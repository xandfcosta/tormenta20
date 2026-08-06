import { useMutation, useQueryClient } from '@tanstack/react-query'
import { toast } from 'sonner'
import {
  firstErrorMessage,
  validateConsumeQuantity,
  validateEquipChange,
} from '@tormenta20/t20-data'
import { allCatalogItems, getCatalogItem } from '@/shared/lib/catalog-cache'
import type {
  Character,
  CharacterItem,
  ConsumeItemInput,
  ConsumeItemResult,
  CreateItemInput,
  UpdateItemInput,
} from '@/shared/api/api'
import { api } from '@/shared/api/api'
import { invalidateCharacterDependents } from '@/entities/character/character-cache'
import { characterQueryOptions } from '@/entities/character/queries'

export type ItemMutations = {
  addItem: (input: CreateItemInput, fail: (e: Error) => void) => void
  /** Equip-cap pre-validated update (≤4 vested / ≤2 hands). */
  changeItem: (
    itemId: number,
    input: UpdateItemInput,
    fail: (e: Error) => void,
  ) => void
  removeItem: (itemId: number) => void
  /** Quantity pre-validated consume. */
  consumeItem: (item: CharacterItem, input?: ConsumeItemInput) => void
}

/**
 * The four inventory mutations with their optimistic-cache choreography,
 * extracted from the old InventoryPanel so any items view (the Mochila bag)
 * can reuse them. All updates patch the cached Character first and roll back
 * via `onError`; domain rules (equip caps, consume quantity) are pre-checked
 * with the shared t20-data validators so a mutation only fires when the
 * server will accept it.
 *
 * @example const { changeItem } = useItemMutations(character)
 */
export function useItemMutations(character: Character): ItemMutations {
  const qc = useQueryClient()
  const queryKey = characterQueryOptions(character.id).queryKey

  const add = useMutation<
    CharacterItem,
    Error,
    CreateItemInput,
    { previous: Character | undefined; tempId: number }
  >({
    mutationFn: (input) => api.characters.addItem(character.id, input),
    onMutate: async (input) => {
      await qc.cancelQueries({ queryKey })
      const previous = qc.getQueryData<Character>(queryKey)
      const tempId = -Date.now()
      const catalog = input.catalogId
        ? allCatalogItems().find((c) => c.id === input.catalogId)
        : undefined
      const optimistic: CharacterItem = {
        id: tempId,
        catalogId: input.catalogId ?? null,
        name: input.name ?? catalog?.name ?? '...',
        quantity: input.quantity,
        slots: input.slots ?? catalog?.slots ?? 1,
        equipped: input.equipped ?? null,
        improvements: JSON.stringify(input.improvements ?? []),
        material: input.material ?? null,
      }
      qc.setQueryData<Character>(queryKey, (prev) =>
        prev ? { ...prev, items: [...prev.items, optimistic] } : prev,
      )
      return { previous, tempId }
    },
    onError: (_e, _v, ctx) => {
      if (ctx?.previous) qc.setQueryData(queryKey, ctx.previous)
    },
    onSuccess: (created, _v, ctx) => {
      qc.setQueryData<Character>(queryKey, (prev) =>
        prev
          ? {
              ...prev,
              items: prev.items.map((it) =>
                ctx && it.id === ctx.tempId ? created : it,
              ),
            }
          : prev,
      )
      invalidateCharacterDependents(qc, character.id)
    },
  })

  const update = useMutation<
    CharacterItem,
    Error,
    { itemId: number; input: UpdateItemInput },
    { previous: Character | undefined }
  >({
    mutationFn: ({ itemId, input }) =>
      api.characters.updateItem(character.id, itemId, input),
    onMutate: async ({ itemId, input }) => {
      await qc.cancelQueries({ queryKey })
      const previous = qc.getQueryData<Character>(queryKey)
      qc.setQueryData<Character>(queryKey, (prev) =>
        prev
          ? {
              ...prev,
              items: prev.items.map((it) => {
                if (it.id !== itemId) return it
                const { improvements, ...rest } = input
                const merged: CharacterItem = { ...it, ...rest }
                if (improvements !== undefined) {
                  merged.improvements = JSON.stringify(improvements)
                }
                return merged
              }),
            }
          : prev,
      )
      return { previous }
    },
    onError: (_e, _v, ctx) => {
      if (ctx?.previous) qc.setQueryData(queryKey, ctx.previous)
    },
    onSuccess: (updated) => {
      qc.setQueryData<Character>(queryKey, (prev) =>
        prev
          ? {
              ...prev,
              items: prev.items.map((it) =>
                it.id === updated.id ? updated : it,
              ),
            }
          : prev,
      )
      invalidateCharacterDependents(qc, character.id)
    },
  })

  const remove = useMutation<
    { id: number },
    Error,
    number,
    { previous: Character | undefined }
  >({
    mutationFn: (itemId) => api.characters.deleteItem(character.id, itemId),
    onMutate: async (itemId) => {
      await qc.cancelQueries({ queryKey })
      const previous = qc.getQueryData<Character>(queryKey)
      qc.setQueryData<Character>(queryKey, (prev) =>
        prev
          ? { ...prev, items: prev.items.filter((it) => it.id !== itemId) }
          : prev,
      )
      return { previous }
    },
    onError: (_e, _v, ctx) => {
      if (ctx?.previous) qc.setQueryData(queryKey, ctx.previous)
    },
    onSuccess: () => {
      invalidateCharacterDependents(qc, character.id)
    },
  })

  const consume = useMutation<
    ConsumeItemResult,
    Error,
    { itemId: number; input?: ConsumeItemInput },
    { previous: Character | undefined }
  >({
    mutationFn: ({ itemId, input }) =>
      api.characters.consumeItem(character.id, itemId, input),
    onMutate: async ({ itemId, input }) => {
      // Optimistic: decrement (delete at 1→0) + apply the instant PV/PM gain
      // the same way the server clamps it. Scene/day effect rows stay
      // server-authoritative (reconciled in onSuccess).
      await qc.cancelQueries({ queryKey })
      const previous = qc.getQueryData<Character>(queryKey)
      qc.setQueryData<Character>(queryKey, (prev) => {
        if (!prev) return prev
        const item = prev.items.find((i) => i.id === itemId)
        if (!item) return prev
        const items =
          item.quantity > 1
            ? prev.items.map((i) =>
                i.id === itemId ? { ...i, quantity: i.quantity - 1 } : i,
              )
            : prev.items.filter((i) => i.id !== itemId)
        const instant = item.catalogId
          ? getCatalogItem(item.catalogId)?.consumable?.instant
          : undefined
        let { hpCurrent, mpCurrent } = prev
        if (instant?.hp) {
          const gain = input?.hpRolled ?? instant.hp.bonus ?? 0
          hpCurrent = Math.min(prev.hpMax, hpCurrent + gain)
        }
        if (instant?.mp) {
          const gain = input?.mpRolled ?? instant.mp.bonus ?? 0
          mpCurrent = Math.min(prev.mpMax, mpCurrent + gain)
        }
        return { ...prev, items, hpCurrent, mpCurrent }
      })
      return { previous }
    },
    onError: (_e, _v, ctx) => {
      if (ctx?.previous) qc.setQueryData(queryKey, ctx.previous)
    },
    // Merge the delta into the cached character: reconcile item count, append
    // a created scene/day effect, set vitals.
    onSuccess: (delta) => {
      qc.setQueryData<Character>(queryKey, (prev) => {
        if (!prev) return prev
        const items = delta.item.removed
          ? prev.items.filter((i) => i.id !== delta.item.id)
          : prev.items.map((i) =>
              i.id === delta.item.id
                ? { ...i, quantity: delta.item.quantity }
                : i,
            )
        const activeEffects = delta.effect
          ? [
              ...prev.activeEffects.filter((e) => e.id !== delta.effect!.id),
              delta.effect,
            ]
          : prev.activeEffects
        return {
          ...prev,
          items,
          activeEffects,
          hpCurrent: delta.hpCurrent,
          mpCurrent: delta.mpCurrent,
        }
      })
      invalidateCharacterDependents(qc, character.id)
    },
  })

  const changeItem: ItemMutations['changeItem'] = (itemId, input, fail) => {
    if (input.equipped) {
      const others = character.items
        .filter((it) => it.id !== itemId)
        .map((it) => it.equipped)
      const err = firstErrorMessage(validateEquipChange(others, input.equipped))
      if (err) {
        toast.error(err)
        return
      }
    }
    update.mutate({ itemId, input }, { onError: fail })
  }

  const consumeItem: ItemMutations['consumeItem'] = (item, input) => {
    const err = firstErrorMessage(validateConsumeQuantity(item.quantity))
    if (err) {
      toast.error(err)
      return
    }
    consume.mutate({ itemId: item.id, input })
  }

  return {
    addItem: (input, fail) => add.mutate(input, { onError: fail }),
    changeItem,
    removeItem: (itemId) => remove.mutate(itemId),
    consumeItem,
  }
}
