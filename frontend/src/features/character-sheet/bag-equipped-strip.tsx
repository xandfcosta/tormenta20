import { X } from 'lucide-react'
import type { CharacterItem } from '@/shared/api/api'
import { accentStrong, dimText } from '@/shared/lib/sheet-theme'
import { cn } from '@/shared/lib/utils'
import type { BagPartition } from './bag-slots'
import { equipBonuses } from './equip-bonuses'
import { itemOverlayNames } from './item-describe'

/**
 * The bag's paper-doll strip — the two capacity pools the rules track (there
 * are no body slots in T20): Mãos (≤2 hand-slots; a `wielded2` weapon takes
 * both) and Vestidos (≤4). Filled cards open the item's action sheet;
 * the ✕ is a quick unequip that skips the sheet.
 */
export function BagEquippedStrip({
  partition,
  onOpen,
  onUnequip,
}: {
  partition: BagPartition
  onOpen: (item: CharacterItem) => void
  onUnequip: (item: CharacterItem) => void
}) {
  const { twoHand, wielded, vested, handsUsed } = partition
  return (
    <div className="grid gap-3 lg:grid-cols-[1fr_2fr]">
      <PoolBox title="Mãos" count={handsUsed} max={2}>
        {twoHand ? (
          <EquippedCard label="Duas mãos" item={twoHand} onOpen={onOpen} onUnequip={onUnequip} wide />
        ) : (
          <>
            <EquippedCard label="Mão principal" item={wielded[0]} onOpen={onOpen} onUnequip={onUnequip} />
            <EquippedCard label="Mão secundária" item={wielded[1]} onOpen={onOpen} onUnequip={onUnequip} />
          </>
        )}
      </PoolBox>
      <PoolBox title="Vestidos" count={vested.length} max={4} columns={4}>
        {Array.from({ length: 4 }, (_, i) => (
          <EquippedCard
            key={vested[i]?.id ?? `empty-${i}`}
            label="Vestido"
            item={vested[i]}
            onOpen={onOpen}
            onUnequip={onUnequip}
          />
        ))}
      </PoolBox>
    </div>
  )
}

/** Titled capacity pool: x/max counter over a slot grid (2 cols on phone). */
function PoolBox({
  title,
  count,
  max,
  columns = 2,
  children,
}: {
  title: string
  count: number
  max: number
  columns?: 2 | 4
  children: React.ReactNode
}) {
  return (
    <div className="space-y-1.5">
      <div className="flex items-baseline justify-between">
        <h3 className={cn('text-[10px] font-bold uppercase tracking-widest', accentStrong)}>
          {title}
        </h3>
        <span
          className={cn(
            'font-mono text-xs',
            count >= max ? 'text-foreground' : dimText,
          )}
        >
          {count}/{max}
        </span>
      </div>
      <div className={cn('grid grid-cols-2 gap-2', columns === 4 && 'sm:grid-cols-4')}>
        {children}
      </div>
    </div>
  )
}

function EquippedCard({
  label,
  item,
  onOpen,
  onUnequip,
  wide,
}: {
  label: string
  item: CharacterItem | undefined
  onOpen: (item: CharacterItem) => void
  onUnequip: (item: CharacterItem) => void
  wide?: boolean
}) {
  if (!item) {
    return (
      <div
        className={cn(
          'flex min-h-[3.75rem] flex-col justify-center rounded-lg border border-dashed border-border bg-muted/30 px-2.5 py-2',
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
  const overlays = itemOverlayNames(item)
  return (
    <div
      className={cn(
        'relative min-h-[3.75rem] rounded-lg border border-primary/40 bg-muted px-2.5 py-2 text-left',
        wide && 'col-span-2',
      )}
    >
      <button
        type="button"
        onClick={() => onOpen(item)}
        aria-label={`Abrir ${item.name}`}
        className="block w-[calc(100%-1.5rem)] text-left"
      >
        <span className={cn('block text-[9px] uppercase tracking-widest', dimText)}>
          {label}
        </span>
        <span
          className={cn('block truncate text-sm font-semibold', accentStrong)}
          title={item.name}
        >
          {item.name}
        </span>
        {(bonuses.length > 0 || overlays.length > 0) && (
          <span className="mt-1 flex flex-wrap gap-1">
            {[...overlays, ...bonuses].map((b) => (
              <span
                key={b}
                className="rounded-full border border-border bg-muted px-1.5 py-0.5 text-[10px] text-foreground"
              >
                {b}
              </span>
            ))}
          </span>
        )}
      </button>
      <button
        type="button"
        className="absolute right-1.5 top-1.5 inline-flex size-5 items-center justify-center rounded-full text-foreground hover:bg-red-100 hover:text-red-700 dark:hover:bg-red-950/40 dark:hover:text-red-400"
        onClick={() => onUnequip(item)}
        aria-label={`Desequipar ${item.name}`}
      >
        <X className="size-3" />
      </button>
    </div>
  )
}
