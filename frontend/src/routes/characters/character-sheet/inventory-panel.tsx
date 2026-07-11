import { useMutation, useQueryClient } from '@tanstack/react-query'
import { Plus } from 'lucide-react'
import { CATALOG_ITEMS } from '@tormenta20/t20-data'
import { Button } from '@/shared/ui/button'
import type {
  Character,
  CharacterItem,
  CreateItemInput,
  UpdateItemInput,
} from '@/shared/api/api'
import { api } from '@/shared/api/api'
import { invalidateCharacterDependents } from '@/shared/lib/character-cache'
import {
  inventorySlotsTotal,
  isItemProficient,
  useCharacterEffects,
} from '@/shared/lib/derived'
import { characterQueryOptions } from '@/shared/lib/queries'
import {
  accentStrong,
  dimText,
  panelBg,
  surface,
} from '@/shared/lib/sheet-theme'
import { cn } from '@/shared/lib/utils'
import { AddCatalogItemDialog } from './catalog-picker-dialog'
import { formatLoad } from './item-describe'
import { ItemFormDialog } from './item-form-dialog'
import { InventoryRow } from './inventory-row'

function inventoryUsed(items: CharacterItem[]): number {
  return items.reduce((sum, it) => sum + it.quantity * it.slots, 0)
}

/**
 * "Inventário" panel — lists every CharacterItem row with an
 * inline equip picker, load bar, and access to catalog/custom-item
 * dialogs. All mutations use optimistic updates so the UI never
 * blocks on the round-trip; failures roll back via `onError`.
 */
export function InventoryPanel({ character }: { character: Character }) {
  const qc = useQueryClient()
  const queryKey = characterQueryOptions(character.id).queryKey
  const effects = useCharacterEffects(character)
  const max = inventorySlotsTotal(character, effects)
  const used = inventoryUsed(character.items)
  const pct = max > 0 ? Math.min(100, (used / max) * 100) : 0
  const over = used > max

  const addItem = useMutation<
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
        ? CATALOG_ITEMS.find((c) => c.id === input.catalogId)
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

  const updateItem = useMutation<
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
              items: prev.items.map((it) => (it.id === updated.id ? updated : it)),
            }
          : prev,
      )
      invalidateCharacterDependents(qc, character.id)
    },
  })

  const removeItem = useMutation<
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

  const consumeItem = useMutation<Character, Error, number>({
    mutationFn: (itemId) => api.characters.consumeItem(character.id, itemId),
    onSuccess: (next) => {
      qc.setQueryData<Character>(queryKey, next)
      invalidateCharacterDependents(qc, character.id)
    },
  })

  const items = character.items

  return (
    <section
      className={cn(
        'flex min-h-0 flex-1 flex-col overflow-hidden rounded-xl',
        surface,
        panelBg,
      )}
    >
      <div className="flex shrink-0 flex-wrap items-center justify-between gap-2 border-b border-amber-700/30 px-3 py-2 dark:border-amber-500/20 sm:px-4">
        <div className="min-w-0">
          <h2
            className={cn(
              'font-serif text-lg font-bold tracking-wide',
              accentStrong,
            )}
          >
            Inventário
          </h2>
          <p className={cn('text-[10px] sm:text-xs', dimText)}>
            carga{' '}
            <span
              className={cn(
                'font-mono',
                over
                  ? 'text-red-700 dark:text-red-400'
                  : 'text-zinc-700 dark:text-zinc-300',
              )}
            >
              {formatLoad(used)}
            </span>{' '}
            / {max}
            {over && (
              <span className="ml-2 text-[10px] uppercase tracking-widest text-red-700 dark:text-red-400">
                sobrecarga
              </span>
            )}
            <span className="ml-2">• limite 10 + 2×|FOR|</span>
          </p>
        </div>
        <div className="flex items-center gap-2">
          <AddCatalogItemDialog
            onAdd={(input, fail) => addItem.mutate(input, { onError: fail })}
          />
          <ItemFormDialog
            title="Novo item"
            submitLabel="Adicionar"
            trigger={
              <Button
                type="button"
                size="sm"
                variant="outline"
                className="h-7 gap-1 text-xs"
                aria-label="Adicionar item custom"
              >
                <Plus className="size-3.5" />
                Custom
              </Button>
            }
            onSubmit={(input, fail) => addItem.mutate(input, { onError: fail })}
          />
        </div>
      </div>

      <div className="shrink-0 px-3 pb-2 pt-2 sm:px-4">
        <div className="h-2 overflow-hidden rounded-full border border-amber-700/30 bg-amber-50/70 dark:border-zinc-800 dark:bg-zinc-950">
          <div
            className={cn(
              'h-full transition-all',
              over
                ? 'bg-gradient-to-r from-red-700 to-red-500'
                : 'bg-gradient-to-r from-amber-700 to-amber-500',
            )}
            style={{ width: `${pct}%` }}
          />
        </div>
      </div>

      <div className="min-h-0 flex-1 overflow-auto px-2 py-1">
        {items.length === 0 ? (
          <p className={cn('px-2 py-3 text-center text-xs', dimText)}>
            Nenhum item. Use "+ Item" para adicionar.
          </p>
        ) : (
          <div className="grid gap-y-0.5">
            <InventoryHeader className="hidden sm:flex" />
            {items.map((it) => (
              <InventoryRow
                key={it.id}
                item={it}
                proficient={isItemProficient(character, it)}
                onUpdate={(input, fail) =>
                  updateItem.mutate({ itemId: it.id, input }, { onError: fail })
                }
                onDelete={() => removeItem.mutate(it.id)}
                onConsume={() => consumeItem.mutate(it.id)}
              />
            ))}
          </div>
        )}
      </div>
    </section>
  )
}

function InventoryHeader({ className }: { className?: string }) {
  const cell = cn('text-[9px] uppercase tracking-widest', dimText)
  return (
    <div
      className={cn(
        'items-center gap-2 border-b border-amber-700/20 px-2 pb-1 pt-2 dark:border-amber-500/15',
        className,
      )}
    >
      <span className={cn(cell, 'flex-1')}>item</span>
      <span className={cn(cell, 'w-12 text-center')}>qtd</span>
      <span className={cn(cell, 'w-14 text-center')}>esp</span>
      <span className={cn(cell, 'w-14 text-right')}>total</span>
      <span className={cn(cell, 'w-20 text-center')}>equipar</span>
      <span className="size-7 shrink-0" aria-hidden />
      <span className="size-7 shrink-0" aria-hidden />
      <span className="size-7 shrink-0" aria-hidden />
    </div>
  )
}

