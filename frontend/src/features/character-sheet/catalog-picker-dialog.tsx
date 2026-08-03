import { useState } from 'react'
import { useForm } from '@tanstack/react-form'
import { z } from 'zod'
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
import { Field, FieldError, FieldLabel } from '@/shared/ui/field'
import { Input } from '@/shared/ui/input'
import { NumberInput } from '@/shared/ui/number-input'
import { VirtualList } from '@/shared/ui/virtual-list'
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
import { equipOptionsForCatalog } from './equip-options'
import { formatLoad, overlayNotesSummary } from './item-describe'
import {
  ITEM_DIALOG_CONTENT,
  ItemDialogFooter,
  ItemDialogSection,
} from './item-dialog-kit'
import { normalize } from './normalize'

/** How many of a catalog item to add — whole units only. */
const catalogAddSchema = z.object({
  quantity: z
    .number()
    .int('Quantidade deve ser inteiro ≥ 1.')
    .min(1, 'Quantidade deve ser inteiro ≥ 1.')
    .max(9999, 'Máximo 9999.'),
})


// Distinct catalog categories, sorted, for the add-dialog category filter.
const CATALOG_CATEGORIES = [
  ...new Set(CATALOG_ITEMS.map((c) => c.category)),
].sort()

// pt-BR labels for the English catalog category ids.
const CATEGORY_LABEL: Record<string, string> = {
  animal: 'Animal',
  apparel: 'Vestuário',
  'armor-heavy': 'Armadura pesada',
  'armor-light': 'Armadura leve',
  catalyst: 'Catalisador',
  consumable: 'Consumível',
  improvement: 'Melhoria',
  material: 'Material',
  meal: 'Alimentação',
  shield: 'Escudo',
  vehicle: 'Veículo',
  'weapon-exotic': 'Arma exótica',
  'weapon-firearm': 'Arma de fogo',
  'weapon-martial': 'Arma marcial',
  'weapon-simple': 'Arma simples',
}

const categoryLabel = (c: string): string => CATEGORY_LABEL[c] ?? c

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
            'hover:bg-muted hover:text-foreground dark:hover:bg-muted dark:hover:text-foreground',
          )}
          aria-label={`Melhorias e material de ${item.name}`}
        >
          <Gem className="size-3.5" />
        </Button>
      </DialogTrigger>
      <DialogContent
        className={cn(ITEM_DIALOG_CONTENT, 'border-border bg-muted text-foreground')}
      >
        <DialogHeader>
          <DialogTitle className={cn(accentStrong)}>
            Melhorias & Material — {item.name}
          </DialogTitle>
        </DialogHeader>
        <div className="space-y-4">
          <ItemDialogSection title="Melhorias">
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
                      <label className="flex cursor-pointer items-start gap-2 rounded-md px-2 py-1 text-xs hover:bg-muted dark:hover:bg-muted">
                        <input
                          type="checkbox"
                          checked={checked}
                          onChange={() => toggleImprovement(imp.id)}
                          className="mt-0.5"
                        />
                        <span className="flex-1">
                          <span className="font-semibold">{imp.name}</span>
                          <span className={cn('ml-2', dimText)}>
                            {overlayNotesSummary(imp.modifiers)}
                          </span>
                        </span>
                      </label>
                    </li>
                  )
                })}
              </ul>
            )}
          </ItemDialogSection>
          <ItemDialogSection title="Material">
            {availableMaterials.length === 0 ? (
              <p className={cn('mt-1 text-xs italic', dimText)}>
                Nenhum material compatível.
              </p>
            ) : (
              <ul className="mt-2 space-y-1">
                <li>
                  <label className="flex cursor-pointer items-start gap-2 rounded-md px-2 py-1 text-xs hover:bg-muted dark:hover:bg-muted">
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
                    <label className="flex cursor-pointer items-start gap-2 rounded-md px-2 py-1 text-xs hover:bg-muted dark:hover:bg-muted">
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
                          {overlayNotesSummary(mat.modifiers)}
                        </span>
                      </span>
                    </label>
                  </li>
                ))}
              </ul>
            )}
          </ItemDialogSection>
          {error ? (
            <p className="text-xs text-red-700 dark:text-red-400">{error}</p>
          ) : null}
          <ItemDialogFooter>
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
          </ItemDialogFooter>
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
  const [category, setCategory] = useState('')
  const [equipped, setEquipped] = useState<'' | 'vested' | 'wielded' | 'wielded2'>('')
  const [formError, setFormError] = useState<string | null>(null)

  const selected = catalogId
    ? CATALOG_ITEMS.find((c) => c.id === catalogId)
    : undefined
  // Slot choices for the picked entry; a single "—" (consumables, meals)
  // means the item cannot be equipped, so the control disappears.
  const equipChoices = selected ? equipOptionsForCatalog(selected) : []

  const form = useForm({
    defaultValues: { quantity: 1 },
    validators: { onSubmit: catalogAddSchema },
    onSubmit: ({ value }) => {
      if (!selected) return
      setFormError(null)
      onAdd(
        {
          catalogId: selected.id,
          quantity: value.quantity,
          equipped: equipped || undefined,
        },
        (e) => setFormError(e.message),
      )
      setOpen(false)
      resetLocal()
      form.reset()
    },
  })

  const resetLocal = () => {
    setCatalogId('')
    setSearch('')
    setCategory('')
    setEquipped('')
    setFormError(null)
  }

  const close = (next: boolean) => {
    setOpen(next)
    if (!next) {
      resetLocal()
      form.reset()
    }
  }

  const filtered = CATALOG_ITEMS.filter((c) => {
    if (category && c.category !== category) return false
    if (search.trim() === '') return true
    return (
      normalize(c.name).includes(normalize(search)) ||
      normalize(c.category).includes(normalize(search))
    )
  })

  return (
    <Dialog open={open} onOpenChange={close}>
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
        className={cn(ITEM_DIALOG_CONTENT, 'border-border bg-muted text-foreground')}
      >
        <DialogHeader>
          <DialogTitle className={cn(accentStrong)}>
            Adicionar do catálogo
          </DialogTitle>
        </DialogHeader>
        <form
          className="space-y-4"
          onSubmit={(e) => {
            e.preventDefault()
            e.stopPropagation()
            form.handleSubmit()
          }}
        >
          <div className="space-y-1">
            <span className={cn('text-[10px] uppercase tracking-widest', dimText)}>
              item
            </span>
            <div className="flex gap-2">
              <Input
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                placeholder="Buscar pelo nome..."
                autoFocus
                className="flex-1"
              />
              <select
                value={category}
                onChange={(e) => setCategory(e.target.value)}
                className={cn(selectClass, 'h-9 max-w-[45%] px-2 text-sm')}
                aria-label="Categoria"
              >
                <option value="">Todas categorias</option>
                {CATALOG_CATEGORIES.map((c) => (
                  <option key={c} value={c}>
                    {categoryLabel(c)}
                  </option>
                ))}
              </select>
            </div>
            {filtered.length === 0 ? (
              <p
                className={cn(
                  'mt-1 rounded-md border px-3 py-4 text-center text-xs',
                  'border-border bg-muted  ',
                  dimText,
                )}
              >
                Nenhum item.
              </p>
            ) : (
              <VirtualList
                className={cn(
                  'mt-1 max-h-56 rounded-md border',
                  'border-border bg-muted  ',
                )}
                items={filtered}
                estimateSize={34}
                getKey={(opt) => opt.id}
                renderItem={(opt) => (
                  <button
                    type="button"
                    onClick={() => {
                      setCatalogId(opt.id)
                      setEquipped('')
                      setFormError(null)
                    }}
                    className={cn(
                      'flex w-full items-center justify-between gap-2 px-3 py-1.5 text-left text-sm transition-colors',
                      catalogId === opt.id
                        ? 'bg-muted text-foreground  '
                        : 'hover:bg-muted dark:hover:bg-muted',
                    )}
                  >
                    <span className="truncate">{opt.name}</span>
                    <span className={cn('shrink-0 text-[10px]', dimText)}>
                      {categoryLabel(opt.category)}
                    </span>
                  </button>
                )}
              />
            )}
          </div>
          {selected && (
            <div
              className={cn(
                'rounded-md border px-3 py-2 text-[11px]',
                'border-border bg-muted  ',
              )}
            >
              <p className={cn('font-semibold', accentStrong)}>{selected.name}</p>
              <p className={dimText}>
                {categoryLabel(selected.category)} • esp{' '}
                {formatLoad(selected.slots)} • T${' '}
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
            <form.Field
              name="quantity"
              validators={{ onChange: catalogAddSchema.shape.quantity }}
            >
              {(f) => {
                const invalid = f.state.meta.isTouched && !f.state.meta.isValid
                return (
                  <Field data-invalid={invalid}>
                    <FieldLabel htmlFor={f.name}>Quantidade</FieldLabel>
                    <NumberInput
                      id={f.name}
                      value={f.state.value}
                      onChange={(v) => f.handleChange(v)}
                      onBlur={f.handleBlur}
                      min={1}
                      max={9999}
                      step={1}
                      aria-invalid={invalid}
                    />
                    {invalid && <FieldError errors={f.state.meta.errors} />}
                  </Field>
                )
              }}
            </form.Field>
            {equipChoices.length > 1 && (
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
                  {equipChoices.map((opt) => (
                    <option key={opt.value} value={opt.value}>
                      {opt.label}
                    </option>
                  ))}
                </select>
              </div>
            )}
          </div>
          {formError && (
            <p className="text-xs text-destructive" role="alert">
              {formError}
            </p>
          )}
          <ItemDialogFooter>
            <Button type="button" variant="outline" onClick={() => close(false)}>
              Cancelar
            </Button>
            <form.Subscribe
              selector={(s) => s.canSubmit}
              children={(canSubmit) => (
                <Button type="submit" disabled={!selected || !canSubmit}>
                  Adicionar
                </Button>
              )}
            />
          </ItemDialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}
