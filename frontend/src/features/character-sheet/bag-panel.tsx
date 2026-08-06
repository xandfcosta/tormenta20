import { useState } from 'react'
import { Plus } from 'lucide-react'
import { getCatalogItem } from '@/shared/lib/catalog-cache'
import { Button } from '@/shared/ui/button'
import { Input } from '@/shared/ui/input'
import type { Character, CharacterItem } from '@/shared/api/api'
import {
  attributeTotal,
  inventorySlotsTotal,
  isItemProficient,
  useCharacterEffects,
} from '@/entities/character/derived'
import {
  accentStrong,
  dimText,
  panelBg,
  surface,
} from '@/shared/lib/sheet-theme'
import { cn } from '@/shared/lib/utils'
import { BagEquippedStrip } from './bag-equipped-strip'
import { BagItemSheet } from './bag-item-sheet'
import { partitionBag } from './bag-slots'
import { BagTile } from './bag-tile'
import { AddCatalogItemDialog } from './catalog-picker-dialog'
import { formatLoad, loadLimitLabel } from './item-describe'
import { ItemFormDialog } from './item-form-dialog'
import { useItemMutations } from './use-item-mutations'

const CATEGORY_FILTERS = [
  { key: 'all', label: 'tudo', match: () => true },
  {
    key: 'weapons',
    label: 'armas',
    match: (c: string) => c.startsWith('weapon-'),
  },
  {
    key: 'defense',
    label: 'defesa',
    match: (c: string) => c.startsWith('armor-') || c === 'shield',
  },
  {
    key: 'consumables',
    label: 'consumo',
    match: (c: string) => c === 'consumable' || c === 'meal',
  },
  {
    key: 'other',
    label: 'outros',
    match: (c: string) =>
      !c.startsWith('weapon-') &&
      !c.startsWith('armor-') &&
      !['shield', 'consumable', 'meal'].includes(c),
  },
] as const

type FilterKey = (typeof CATEGORY_FILTERS)[number]['key']

function matchesFilter(item: CharacterItem, key: FilterKey): boolean {
  if (key === 'all') return true
  const category = item.catalogId
    ? getCatalogItem(item.catalogId)?.category
    : undefined
  const filter = CATEGORY_FILTERS.find((f) => f.key === key)!
  // Custom items have no category — they only show under "tudo"/"outros".
  return filter.match(category ?? 'gear')
}

/**
 * "Mochila" — the game-bag view that replaced the Inventário table and the
 * Equipado slots tab. Top: paper-doll strip (Mãos 2 / Vestidos 4) showing
 * what is empunhado/vestido at a glance. Below: carga meter + search +
 * category filter over a tile grid of stowed items. Every tile/card opens
 * the same action sheet (equip states, Usar, detalhes, melhorias, editar,
 * remover). Proficiências live in their own tab.
 */
export function BagPanel({ character }: { character: Character }) {
  const effects = useCharacterEffects(character)
  const mutations = useItemMutations(character)
  const max = inventorySlotsTotal(character, effects)
  const used = character.items.reduce((s, it) => s + it.quantity * it.slots, 0)
  const pct = max > 0 ? Math.min(100, (used / max) * 100) : 0
  const over = used > max

  const [query, setQuery] = useState('')
  const [filter, setFilter] = useState<FilterKey>('all')
  const [openItemId, setOpenItemId] = useState<number | null>(null)

  const partition = partitionBag(character.items)
  const q = query.trim().toLowerCase()
  const stowed = partition.stowed
    .filter((it) => (q ? it.name.toLowerCase().includes(q) : true))
    .filter((it) => matchesFilter(it, filter))

  const openItem = character.items.find((it) => it.id === openItemId)

  return (
    <section
      className={cn(
        'flex min-h-0 flex-1 flex-col overflow-hidden rounded-xl',
        surface,
        panelBg,
      )}
    >
      <header className="flex shrink-0 flex-wrap items-center justify-between gap-2 border-b border-border px-3 py-2 sm:px-4">
        <div className="min-w-0">
          <h2 className={cn('text-lg font-bold tracking-wide', accentStrong)}>
            Mochila
          </h2>
          <p className={cn('text-[10px] sm:text-xs', dimText)}>
            carga{' '}
            <span
              className={cn(
                'font-mono',
                over ? 'text-red-700 dark:text-red-400' : 'text-foreground',
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
            <span className="ml-2">
              • {loadLimitLabel(max, attributeTotal(character, 'strength', effects))}
            </span>
          </p>
        </div>
        <div className="flex items-center gap-2">
          <AddCatalogItemDialog onAdd={mutations.addItem} />
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
            onSubmit={mutations.addItem}
          />
        </div>
      </header>

      <div className="min-h-0 flex-1 space-y-3 overflow-y-auto p-3 sm:p-4">
        <BagEquippedStrip
          partition={partition}
          onOpen={(item) => setOpenItemId(item.id)}
          onUnequip={(item) =>
            mutations.changeItem(item.id, { equipped: null }, () => {})
          }
        />

        <div className="space-y-2">
          <div className="flex items-baseline justify-between">
            <h3 className={cn('text-[10px] font-bold uppercase tracking-widest', accentStrong)}>
              Mochila (guardado)
            </h3>
            <span className={cn('font-mono text-xs', dimText)}>
              {partition.stowed.length} ite{partition.stowed.length === 1 ? 'm' : 'ns'}
            </span>
          </div>
          <div className="h-2 overflow-hidden rounded-full border border-border bg-muted">
            <div
              className={cn('h-full transition-all', over ? 'bg-destructive' : 'bg-primary')}
              style={{ width: `${pct}%` }}
            />
          </div>
          <div className="flex flex-wrap items-center gap-2">
            <Input
              value={query}
              onChange={(e) => setQuery(e.target.value)}
              placeholder="Buscar item…"
              aria-label="Buscar item na mochila"
              className="h-8 min-w-32 flex-1 text-xs"
            />
            <div className="flex gap-1">
              {CATEGORY_FILTERS.map((f) => (
                <button
                  key={f.key}
                  type="button"
                  onClick={() => setFilter(f.key)}
                  className={cn(
                    'rounded-full border px-2 py-0.5 text-[10px] uppercase tracking-wider transition-colors',
                    filter === f.key
                      ? 'border-primary/60 bg-primary/10 text-primary'
                      : cn('border-border', dimText, 'hover:text-foreground'),
                  )}
                >
                  {f.label}
                </button>
              ))}
            </div>
          </div>
          {stowed.length === 0 ? (
            <p className={cn('py-4 text-center text-xs', dimText)}>
              {q || filter !== 'all'
                ? 'Nenhum item para esse filtro.'
                : 'Mochila vazia. Use "+ Catálogo" para adicionar.'}
            </p>
          ) : (
            <div className="grid grid-cols-3 gap-2 sm:grid-cols-4 lg:grid-cols-6">
              {stowed.map((it) => (
                <BagTile
                  key={it.id}
                  item={it}
                  proficient={isItemProficient(character, it)}
                  onOpen={() => setOpenItemId(it.id)}
                />
              ))}
            </div>
          )}
        </div>

      </div>

      {openItem && (
        <BagItemSheet
          item={openItem}
          proficient={isItemProficient(character, openItem)}
          open
          onOpenChange={(next) => {
            if (!next) setOpenItemId(null)
          }}
          mutations={mutations}
        />
      )}
    </section>
  )
}
