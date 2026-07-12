import { useMutation, useQueryClient } from '@tanstack/react-query'
import { X } from 'lucide-react'
import type { ReactNode } from 'react'
import type { Character, CharacterItem } from '@/shared/api/api'
import { api } from '@/shared/api/api'
import { characterQueryOptions } from '@/entities/character/queries'
import { accentStrong, dimText, panelBg, surface } from '@/shared/lib/sheet-theme'
import { cn } from '@/shared/lib/utils'
import { equipBonuses } from './equip-bonuses'

/**
 * "Equipado" panel — the two equip capacity pools the rules actually track
 * (there are no body slots): Mãos (≤2 hand-slots; a `wielded2` weapon takes
 * both) and Vestidos (≤4 worn items). Each pool shows filled item slots plus
 * dashed placeholders up to its cap, and each equipped item lists what it
 * grants. Clicking ✕ unequips optimistically.
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

  const onUnequip = (id: number) => unequip.mutate(id)
  const vested = character.items.filter((i) => i.equipped === 'vested')
  const wielded = character.items.filter((i) => i.equipped === 'wielded')
  const twoHand = character.items.find((i) => i.equipped === 'wielded2')
  const handsUsed = twoHand ? 2 : wielded.length

  return (
    <section
      className={cn(
        'flex min-h-0 flex-1 flex-col overflow-hidden rounded-xl',
        surface,
        panelBg,
      )}
    >
      <div className="flex shrink-0 items-center border-b border-amber-700/30 px-3 py-2 dark:border-amber-500/20 sm:px-4">
        <h2 className={cn('font-serif text-lg font-bold tracking-wide', accentStrong)}>
          Equipado
        </h2>
      </div>

      <div className="min-h-0 flex-1 space-y-4 overflow-y-auto p-3 sm:p-4">
        <Pool title="Mãos" count={handsUsed} max={2}>
          {twoHand ? (
            <SlotCard label="Duas mãos" item={twoHand} onUnequip={onUnequip} wide />
          ) : (
            <>
              <SlotCard label="Mão principal" item={wielded[0]} onUnequip={onUnequip} />
              <SlotCard label="Mão secundária" item={wielded[1]} onUnequip={onUnequip} />
            </>
          )}
        </Pool>

        <Pool title="Vestidos" count={vested.length} max={4}>
          {Array.from({ length: 4 }, (_, i) => (
            <SlotCard
              key={vested[i]?.id ?? `empty-${i}`}
              label="Vestido"
              item={vested[i]}
              onUnequip={onUnequip}
            />
          ))}
        </Pool>
      </div>
    </section>
  )
}

/** A capacity pool: titled header with an x/max counter over a 2-col grid. */
function Pool({
  title,
  count,
  max,
  children,
}: {
  title: string
  count: number
  max: number
  children: ReactNode
}) {
  return (
    <div className="space-y-2">
      <div className="flex items-baseline justify-between">
        <h3 className={cn('text-xs font-bold uppercase tracking-widest', accentStrong)}>
          {title}
        </h3>
        <span
          className={cn(
            'font-mono text-xs',
            count >= max ? 'text-amber-700 dark:text-amber-400' : dimText,
          )}
        >
          {count}/{max}
        </span>
      </div>
      <div className="grid grid-cols-2 gap-2">{children}</div>
    </div>
  )
}

function SlotCard({
  label,
  item,
  onUnequip,
  wide,
}: {
  label: string
  item: CharacterItem | undefined
  onUnequip: (itemId: number) => void
  wide?: boolean
}) {
  if (!item) {
    return (
      <div
        className={cn(
          'flex min-h-[3.75rem] flex-col justify-center rounded-lg border border-dashed px-3 py-2',
          'border-amber-700/25 bg-amber-50/30 dark:border-amber-500/15 dark:bg-zinc-900/20',
          wide && 'col-span-2',
        )}
      >
        <span className={cn('text-[9px] uppercase tracking-widest', dimText)}>
          {label}
        </span>
        <span className={cn('text-xs', dimText)}>vazio</span>
      </div>
    )
  }

  const bonuses = equipBonuses(item)
  return (
    <div
      className={cn(
        'relative min-h-[3.75rem] rounded-lg border px-3 py-2',
        'border-amber-700/50 bg-amber-100/70 dark:border-amber-500/50 dark:bg-zinc-900/70',
        wide && 'col-span-2',
      )}
    >
      <div className="flex items-start justify-between gap-2">
        <div className="min-w-0">
          <span className={cn('block text-[9px] uppercase tracking-widest', dimText)}>
            {label}
          </span>
          <span
            className={cn('block truncate text-sm font-semibold', accentStrong)}
            title={item.name}
          >
            {item.name}
          </span>
        </div>
        <button
          type="button"
          className="inline-flex size-5 shrink-0 items-center justify-center rounded-full text-zinc-500 hover:bg-red-100 hover:text-red-700 dark:text-zinc-400 dark:hover:bg-red-950/40 dark:hover:text-red-400"
          onClick={() => onUnequip(item.id)}
          aria-label={`Desequipar ${item.name}`}
        >
          <X className="size-3" />
        </button>
      </div>
      {bonuses.length > 0 && (
        <div className="mt-1.5 flex flex-wrap gap-1">
          {bonuses.map((b) => (
            <span
              key={b}
              className="rounded-full border border-amber-700/20 bg-zinc-100/50 px-1.5 py-0.5 text-[10px] text-zinc-700 dark:border-amber-500/15 dark:bg-zinc-900/40 dark:text-zinc-300"
            >
              {b}
            </span>
          ))}
        </div>
      )}
    </div>
  )
}
