import { useState } from 'react'
import { Gem, Plus } from 'lucide-react'
import {
  CATALOG_ITEMS,
  IMPROVEMENTS,
  MATERIALS,
  familyFor,
  getCatalogItem,
} from '@tormenta20/t20-data'
import { Button } from '@/shared/ui/button'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from '@/shared/ui/dialog'
import { Input } from '@/shared/ui/input'
import { NumberInput } from '@/shared/ui/number-input'
import type {
  CharacterItem,
  CreateItemInput,
  UpdateItemInput,
} from '@/shared/api/api'
import { parseImprovementIds } from '@/entities/character/derived'
import {
  accentStrong,
  dimText,
  selectClass,
  subtleText,
} from '@/shared/lib/sheet-theme'
import { cn } from '@/shared/lib/utils'
import { formatLoad } from './item-describe'
import { normalize } from './normalize'

const EQUIP_PICKER_OPTIONS: { value: '' | 'vested' | 'wielded' | 'wielded2'; label: string }[] = [
  { value: '', label: '—' },
  { value: 'vested', label: 'Vestido' },
  { value: 'wielded', label: '1 mão' },
  { value: 'wielded2', label: '2 mãos' },
]

/**
 * Overlay dialog for applying improvements + a special material to an
 * already-owned item. Returns null when the item's catalog category
 * doesn't accept improvements (consumables, meals, catalysts, etc.).
 */
export function OverlayPickerDialog({
  item,
  onUpdate,
}: {
  item: CharacterItem
  onUpdate: (input: UpdateItemInput, onError: (e: Error) => void) => void
}) {
  const catalog = item.catalogId ? getCatalogItem(item.catalogId) : undefined
  const [open, setOpen] = useState(false)
  const [improvements, setImprovements] = useState<string[]>(
    parseImprovementIds(item.improvements),
  )
  const [material, setMaterial] = useState<string | null>(item.material)
  const [error, setError] = useState<string | null>(null)

  if (!catalog) return null
  if (
    catalog.category === 'consumable' ||
    catalog.category === 'meal' ||
    catalog.category === 'catalyst' ||
    catalog.category === 'improvement' ||
    catalog.category === 'material' ||
    catalog.category === 'animal' ||
    catalog.category === 'vehicle'
  ) {
    return null
  }

  const baseFamily = familyFor(catalog)
  const availableImprovements = IMPROVEMENTS.filter((imp) =>
    imp.appliesTo?.includes(baseFamily),
  )
  const availableMaterials = MATERIALS.filter((mat) =>
    mat.appliesTo?.includes(baseFamily),
  )

  const reset = () => {
    setImprovements(parseImprovementIds(item.improvements))
    setMaterial(item.material)
    setError(null)
  }

  const toggleImprovement = (id: string) => {
    setImprovements((prev) =>
      prev.includes(id) ? prev.filter((x) => x !== id) : [...prev, id],
    )
  }

  const apply = () => {
    setError(null)
    onUpdate({ improvements, material }, (e) => setError(e.message))
    setOpen(false)
  }

  return (
    <Dialog
      open={open}
      onOpenChange={(next) => {
        setOpen(next)
        if (next) reset()
      }}
    >
      <DialogTrigger asChild>
        <Button
          type="button"
          variant="ghost"
          size="icon"
          className={cn(
            'size-7',
            subtleText,
            'hover:bg-amber-200/60 hover:text-amber-900 dark:hover:bg-zinc-800/60 dark:hover:text-amber-200',
          )}
          aria-label={`Melhorias e material de ${item.name}`}
        >
          <Gem className="size-3.5" />
        </Button>
      </DialogTrigger>
      <DialogContent
        className={cn(
          'w-[calc(100vw-1.5rem)] max-w-[calc(100vw-1.5rem)] p-4 sm:w-full sm:max-w-md sm:p-6',
          'border-amber-700/40 bg-amber-50 text-zinc-900 dark:border-amber-500/40 dark:bg-zinc-950 dark:text-zinc-100',
        )}
      >
        <DialogHeader>
          <DialogTitle className={cn('font-serif', accentStrong)}>
            Melhorias & Material — {item.name}
          </DialogTitle>
        </DialogHeader>
        <div className="space-y-4">
          <section>
            <h3 className={cn('text-xs uppercase tracking-widest', dimText)}>
              Melhorias
            </h3>
            {availableImprovements.length === 0 ? (
              <p className={cn('mt-1 text-xs italic', dimText)}>
                Nenhuma melhoria compatível.
              </p>
            ) : (
              <ul className="mt-2 space-y-1">
                {availableImprovements.map((imp) => {
                  const checked = improvements.includes(imp.id)
                  return (
                    <li key={imp.id}>
                      <label className="flex cursor-pointer items-start gap-2 rounded-md px-2 py-1 text-xs hover:bg-amber-100 dark:hover:bg-zinc-900/60">
                        <input
                          type="checkbox"
                          checked={checked}
                          onChange={() => toggleImprovement(imp.id)}
                          className="mt-0.5"
                        />
                        <span className="flex-1">
                          <span className="font-semibold">{imp.name}</span>
                          <span className={cn('ml-2', dimText)}>
                            {imp.modifiers
                              .map((m) => m.note ?? '')
                              .filter(Boolean)
                              .join(', ')}
                          </span>
                        </span>
                      </label>
                    </li>
                  )
                })}
              </ul>
            )}
          </section>
          <section>
            <h3 className={cn('text-xs uppercase tracking-widest', dimText)}>
              Material
            </h3>
            {availableMaterials.length === 0 ? (
              <p className={cn('mt-1 text-xs italic', dimText)}>
                Nenhum material compatível.
              </p>
            ) : (
              <ul className="mt-2 space-y-1">
                <li>
                  <label className="flex cursor-pointer items-start gap-2 rounded-md px-2 py-1 text-xs hover:bg-amber-100 dark:hover:bg-zinc-900/60">
                    <input
                      type="radio"
                      name={`material-${item.id}`}
                      checked={material === null}
                      onChange={() => setMaterial(null)}
                      className="mt-0.5"
                    />
                    <span className={cn('flex-1 italic', dimText)}>nenhum</span>
                  </label>
                </li>
                {availableMaterials.map((mat) => (
                  <li key={mat.id}>
                    <label className="flex cursor-pointer items-start gap-2 rounded-md px-2 py-1 text-xs hover:bg-amber-100 dark:hover:bg-zinc-900/60">
                      <input
                        type="radio"
                        name={`material-${item.id}`}
                        checked={material === mat.id}
                        onChange={() => setMaterial(mat.id)}
                        className="mt-0.5"
                      />
                      <span className="flex-1">
                        <span className="font-semibold">{mat.name}</span>
                        <span className={cn('ml-2', dimText)}>
                          {mat.modifiers
                            .map((m) => m.note ?? '')
                            .filter(Boolean)
                            .join(', ')}
                        </span>
                      </span>
                    </label>
                  </li>
                ))}
              </ul>
            )}
          </section>
          {error ? (
            <p className="text-xs text-red-700 dark:text-red-400">{error}</p>
          ) : null}
          <div className="flex justify-end gap-2">
            <Button
              type="button"
              variant="ghost"
              onClick={() => setOpen(false)}
            >
              Cancelar
            </Button>
            <Button type="button" onClick={apply}>
              Aplicar
            </Button>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  )
}

/**
 * "Adicionar do catálogo" dialog. Searchable list backed by the
 * static `CATALOG_ITEMS` from `t20-data`; selection triggers `onAdd`
 * with the create input. Passes catalog id + quantity + optional
 * `equipped` slot to the caller mutation.
 */
export function AddCatalogItemDialog({
  onAdd,
}: {
  onAdd: (input: CreateItemInput, onError: (e: Error) => void) => void
}) {
  const [open, setOpen] = useState(false)
  const [catalogId, setCatalogId] = useState('')
  const [search, setSearch] = useState('')
  const [quantity, setQuantity] = useState<string>('1')
  const [equipped, setEquipped] = useState<'' | 'vested' | 'wielded' | 'wielded2'>('')
  const [error, setError] = useState<string | null>(null)

  const reset = () => {
    setCatalogId('')
    setSearch('')
    setQuantity('1')
    setEquipped('')
    setError(null)
  }

  const selected = catalogId
    ? CATALOG_ITEMS.find((c) => c.id === catalogId)
    : undefined

  const filtered =
    search.trim() === ''
      ? CATALOG_ITEMS
      : CATALOG_ITEMS.filter(
          (c) =>
            normalize(c.name).includes(normalize(search)) ||
            normalize(c.category).includes(normalize(search)),
        )

  const apply = () => {
    if (!selected) {
      setError('Selecione um item do catálogo.')
      return
    }
    const qty = Number(quantity)
    if (!Number.isInteger(qty) || qty < 1) {
      setError('Quantidade deve ser inteiro ≥ 1.')
      return
    }
    onAdd(
      {
        catalogId: selected.id,
        quantity: qty,
        equipped: equipped || undefined,
      },
      (e) => setError(e.message),
    )
    setOpen(false)
    reset()
  }

  return (
    <Dialog
      open={open}
      onOpenChange={(o) => {
        setOpen(o)
        if (!o) reset()
      }}
    >
      <DialogTrigger asChild>
        <Button
          type="button"
          size="sm"
          variant="outline"
          className="h-7 gap-1 text-xs"
          aria-label="Adicionar do catálogo"
        >
          <Plus className="size-3.5" />
          Catálogo
        </Button>
      </DialogTrigger>
      <DialogContent
        className={cn(
          'w-[calc(100vw-1.5rem)] max-w-[calc(100vw-1.5rem)] p-4 sm:w-full sm:max-w-md sm:p-6',
          'border-amber-700/40 bg-amber-50 text-zinc-900 dark:border-amber-500/40 dark:bg-zinc-950 dark:text-zinc-100',
        )}
      >
        <DialogHeader>
          <DialogTitle className={cn('font-serif', accentStrong)}>
            Adicionar do catálogo
          </DialogTitle>
        </DialogHeader>
        <div className="space-y-4">
          <div className="space-y-1">
            <span className={cn('text-[10px] uppercase tracking-widest', dimText)}>
              item
            </span>
            <Input
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              placeholder="Buscar pelo nome ou categoria..."
              autoFocus
            />
            <div
              className={cn(
                'mt-1 max-h-56 overflow-y-auto rounded-md border',
                'border-amber-700/30 bg-amber-50/80 dark:border-amber-500/20 dark:bg-zinc-900/60',
              )}
            >
              {filtered.length === 0 ? (
                <p className={cn('px-3 py-4 text-center text-xs', dimText)}>
                  Nenhum item.
                </p>
              ) : (
                filtered.map((opt) => {
                  const active = catalogId === opt.id
                  return (
                    <button
                      key={opt.id}
                      type="button"
                      onClick={() => {
                        setCatalogId(opt.id)
                        if (error) setError(null)
                      }}
                      className={cn(
                        'flex w-full items-center justify-between gap-2 px-3 py-1.5 text-left text-sm transition-colors',
                        active
                          ? 'bg-amber-200/70 text-amber-900 dark:bg-amber-500/20 dark:text-amber-200'
                          : 'hover:bg-amber-100/60 dark:hover:bg-zinc-800/60',
                      )}
                    >
                      <span className="truncate">{opt.name}</span>
                      <span className={cn('shrink-0 text-[10px]', dimText)}>
                        {opt.category}
                      </span>
                    </button>
                  )
                })
              )}
            </div>
          </div>
          {selected && (
            <div
              className={cn(
                'rounded-md border px-3 py-2 text-[11px]',
                'border-amber-700/30 bg-amber-100/70 dark:border-amber-500/30 dark:bg-zinc-900/60',
              )}
            >
              <p className={cn('font-semibold', accentStrong)}>{selected.name}</p>
              <p className={dimText}>
                {selected.category} • esp {formatLoad(selected.slots)} • T${' '}
                {selected.price}
              </p>
              {selected.weapon && (
                <p className={dimText}>
                  dano {selected.weapon.damage} • crit {selected.weapon.critRange}
                  /×{selected.weapon.critMult}
                </p>
              )}
              {selected.armor && (
                <p className={dimText}>
                  Def +{selected.armor.defense} • penalidade{' '}
                  {selected.armor.penalty} •{' '}
                  {selected.armor.heavy ? 'pesada' : 'leve'}
                </p>
              )}
              {selected.shield && (
                <p className={dimText}>
                  Def +{selected.shield.defense} • penalidade{' '}
                  {selected.shield.penalty}
                </p>
              )}
            </div>
          )}
          <div className="grid grid-cols-2 gap-3">
            <div className="space-y-1">
              <span
                className={cn('text-[10px] uppercase tracking-widest', dimText)}
              >
                quantidade
              </span>
              <NumberInput
                value={quantity}
                onChange={(v) => setQuantity(String(v))}
                min={1}
                max={9999}
                step={1}
                aria-label="Quantidade"
              />
            </div>
            <div className="space-y-1">
              <span
                className={cn('text-[10px] uppercase tracking-widest', dimText)}
              >
                equipar
              </span>
              <select
                value={equipped}
                onChange={(e) =>
                  setEquipped(e.target.value as typeof equipped)
                }
                className={cn(selectClass, 'h-9 w-full px-2 text-sm')}
                aria-label="Equipar"
              >
                {EQUIP_PICKER_OPTIONS.map((opt) => (
                  <option key={opt.value} value={opt.value}>
                    {opt.label}
                  </option>
                ))}
              </select>
            </div>
          </div>
          {error && (
            <p className="text-xs text-destructive" role="alert">
              {error}
            </p>
          )}
          <div className="flex justify-end gap-2">
            <Button type="button" variant="outline" onClick={() => setOpen(false)}>
              Cancelar
            </Button>
            <Button type="button" onClick={apply} disabled={!selected}>
              Adicionar
            </Button>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  )
}
