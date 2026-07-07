import { useMutation, useQueryClient } from '@tanstack/react-query'
import { X } from 'lucide-react'
import type {
  Character,
  CharacterItem,
} from '@/lib/api'
import { api } from '@/lib/api'
import { characterQueryOptions } from '@/lib/queries'
import {
  accentStrong,
  dimText,
  panelBg,
  surface,
} from '@/lib/sheet-theme'
import { cn } from '@/lib/utils'

/**
 * "Equipado" panel — visual summary of which items are currently
 * equipped by slot. Each slot is derived from `character.items`
 * filtered by the `equipped` field. Clicking a filled slot unequips
 * the item optimistically.
 *
 * Slot geometry mirrors the paper-doll layout on the character sheet
 * (head / hands / torso / feet / accessory) — a two-hand weapon takes
 * over both hand slots and forces the torso row below.
 */
export function EquipmentPanel({ character }: { character: Character }) {
  const qc = useQueryClient()
  const queryKey = characterQueryOptions(character.id).queryKey

  const unequip = useMutation<
    CharacterItem,
    Error,
    number,
    { previous: Character | undefined }
  >({
    mutationFn: (itemId) =>
      api.characters.updateItem(character.id, itemId, { equipped: null }),
    onMutate: async (itemId) => {
      await qc.cancelQueries({ queryKey })
      const previous = qc.getQueryData<Character>(queryKey)
      qc.setQueryData<Character>(queryKey, (prev) =>
        prev
          ? {
              ...prev,
              items: prev.items.map((it) =>
                it.id === itemId ? { ...it, equipped: null } : it,
              ),
            }
          : prev,
      )
      return { previous }
    },
    onError: (_e, _v, ctx) => {
      if (ctx?.previous) qc.setQueryData(queryKey, ctx.previous)
    },
  })

  const vested = character.items.filter((i) => i.equipped === 'vested')
  const wielded = character.items.filter((i) => i.equipped === 'wielded')
  const twoHand = character.items.find((i) => i.equipped === 'wielded2')

  return (
    <section
      className={cn(
        'flex min-h-0 flex-1 flex-col overflow-hidden rounded-xl',
        surface,
        panelBg,
      )}
    >
      <div className="flex shrink-0 items-center justify-between gap-2 border-b border-amber-700/30 px-3 py-2 dark:border-amber-500/20 sm:px-4">
        <h2
          className={cn(
            'font-serif text-lg font-bold tracking-wide',
            accentStrong,
          )}
        >
          Equipado
        </h2>
        <p className={cn('text-[10px] sm:text-xs', dimText)}>
          {vested.length}/4 vestidos •{' '}
          {twoHand ? '2/2' : `${wielded.length}/2`} mãos
        </p>
      </div>

      <div className="flex min-h-0 flex-1 items-center justify-center overflow-y-auto p-4">
        <div className="grid w-full max-w-xs gap-3">
          <div className="flex justify-center">
            <EquipmentSlot
              label="Cabeça"
              item={vested[0]}
              onUnequip={(id) => unequip.mutate(id)}
              size="md"
            />
          </div>

          <div className="grid grid-cols-3 items-stretch gap-3">
            {twoHand ? (
              <EquipmentSlot
                label="Duas mãos"
                item={twoHand}
                onUnequip={(id) => unequip.mutate(id)}
                size="md"
                className="col-span-3"
              />
            ) : (
              <>
                <EquipmentSlot
                  label="Mão principal"
                  item={wielded[0]}
                  onUnequip={(id) => unequip.mutate(id)}
                  size="md"
                />
                <EquipmentSlot
                  label="Tronco"
                  item={vested[1]}
                  onUnequip={(id) => unequip.mutate(id)}
                  size="md"
                />
                <EquipmentSlot
                  label="Mão secundária"
                  item={wielded[1]}
                  onUnequip={(id) => unequip.mutate(id)}
                  size="md"
                />
              </>
            )}
          </div>

          {twoHand && (
            <div className="flex justify-center">
              <EquipmentSlot
                label="Tronco"
                item={vested[1]}
                onUnequip={(id) => unequip.mutate(id)}
                size="md"
              />
            </div>
          )}

          <div className="grid grid-cols-2 gap-3">
            <EquipmentSlot
              label="Pés"
              item={vested[2]}
              onUnequip={(id) => unequip.mutate(id)}
              size="md"
            />
            <EquipmentSlot
              label="Acessório"
              item={vested[3]}
              onUnequip={(id) => unequip.mutate(id)}
              size="md"
            />
          </div>
        </div>
      </div>
    </section>
  )
}

function EquipmentSlot({
  label,
  item,
  onUnequip,
  size,
  className,
}: {
  label: string
  item: CharacterItem | undefined
  onUnequip: (itemId: number) => void
  size: 'sm' | 'md'
  className?: string
}) {
  const filled = !!item
  return (
    <div
      className={cn(
        'relative flex flex-col items-center justify-center rounded-lg border-2 px-2 py-2 text-center transition-colors',
        size === 'md' ? 'min-h-[64px]' : 'min-h-[44px]',
        filled
          ? 'border-amber-700/60 bg-amber-100/80 dark:border-amber-500/60 dark:bg-zinc-900/70'
          : 'border-dashed border-amber-700/30 bg-amber-50/40 dark:border-amber-500/20 dark:bg-zinc-900/30',
        className,
      )}
    >
      <span
        className={cn(
          'text-[9px] uppercase tracking-widest',
          dimText,
        )}
      >
        {label}
      </span>
      {filled ? (
        <span
          className={cn(
            'mt-0.5 line-clamp-2 text-xs font-semibold',
            accentStrong,
          )}
          title={item.name}
        >
          {item.name}
        </span>
      ) : (
        <span className={cn('mt-0.5 text-xs', dimText)}>—</span>
      )}
      {filled && (
        <button
          type="button"
          className={cn(
            'absolute right-1 top-1 inline-flex size-5 items-center justify-center rounded-full text-zinc-500 hover:bg-red-100 hover:text-red-700 dark:text-zinc-400 dark:hover:bg-red-950/40 dark:hover:text-red-400',
          )}
          onClick={() => onUnequip(item.id)}
          aria-label={`Desequipar ${item.name}`}
        >
          <X className="size-3" />
        </button>
      )}
    </div>
  )
}
