import type { CatalogItem } from '@/shared/api/item-types'
import { Plus } from 'lucide-solid'
import { For, Show, createMemo, createSignal } from 'solid-js'
import type { CreateItemInput, EquippedSlot } from '@/shared/api/api'
import { allCatalogItems } from '@/shared/lib/catalog-cache'
import { Button } from '@/shared/ui/button'
import { Dialog, DialogContent, DialogHeader, DialogTitle } from '@/shared/ui/dialog'
import { DialogInlineError } from '@/shared/ui/dialog-inline-error'
import { FieldFrame } from '@/shared/ui/field-frame'
import { Input } from '@/shared/ui/input'
import { NumberInput } from '@/shared/ui/number-input'
import { VirtualList } from '@/shared/ui/virtual-list'
import { catalogCategories, categoryLabel } from './catalog-categories'
import { equipOptionsForCatalog } from './equip-options'
import {
  ITEM_DIALOG_CONTENT,
  ITEM_DIALOG_TITLE,
  ItemDialogFooter,
} from './item-dialog-kit'
import { formatLoad } from './item-describe'
import { itemWriteMessage } from './item-mutations'
import { normalize } from './normalize'
import { FieldLabel } from '@/shared/ui/section-label'

/**
 * Catalog entries matching the picker's two filters. Accent-insensitive on
 * both name and category, so "escudo" finds the shields without the player
 * knowing the English ids.
 *
 * @example filterCatalog(allCatalogItems(), 'espada', '') // as espadas
 */
export function filterCatalog(
  items: readonly CatalogItem[],
  search: string,
  category: string,
): CatalogItem[] {
  const query = normalize(search.trim())
  return items.filter((c) => {
    if (category && c.category !== category) return false
    if (query === '') return true
    return normalize(c.name).includes(query) || normalize(c.category).includes(query)
  })
}

export type CatalogAddDialogProps = {
  onAdd: (input: CreateItemInput) => Promise<void>
}

/**
 * "Adicionar do catálogo": search the catalog, pick an entry, choose how many
 * and (when the entry allows it) which slot it lands equipped in.
 *
 * @example <CatalogAddDialog onAdd={(input) => actions().add(input)} />
 */
export function CatalogAddDialog(props: CatalogAddDialogProps) {
  const [open, setOpen] = createSignal(false)
  const [search, setSearch] = createSignal('')
  const [category, setCategory] = createSignal('')
  const [catalogId, setCatalogId] = createSignal('')
  const [equipped, setEquipped] = createSignal<'' | EquippedSlot>('')
  const [quantity, setQuantity] = createSignal(1)
  const [formError, setFormError] = createSignal<string | null>(null)
  const [pending, setPending] = createSignal(false)

  const selected = createMemo(() =>
    catalogId() ? allCatalogItems().find((c) => c.id === catalogId()) : undefined,
  )
  // A single "—" (consumables, meals) means the entry cannot be equipped at
  // all, so the control disappears instead of offering one dead choice.
  const equipChoices = createMemo(() => {
    const catalog = selected()
    return catalog ? equipOptionsForCatalog(catalog) : []
  })
  const filtered = createMemo(() => filterCatalog(allCatalogItems(), search(), category()))

  const reset = () => {
    setSearch('')
    setCategory('')
    setCatalogId('')
    setEquipped('')
    setQuantity(1)
    setFormError(null)
  }

  const pick = (id: string) => {
    setCatalogId(id)
    setEquipped('')
    setFormError(null)
  }

  const submit = async (event: SubmitEvent) => {
    event.preventDefault()
    const catalog = selected()
    if (!catalog) return
    setFormError(null)
    setPending(true)
    try {
      await props.onAdd({
        catalogId: catalog.id,
        quantity: quantity(),
        equipped: equipped() || undefined,
      })
      setOpen(false)
      reset()
    } catch (failure) {
      setFormError(itemWriteMessage(failure, 'Não foi possível adicionar o item.'))
    } finally {
      setPending(false)
    }
  }

  return (
    <>
      <Button
        type="button"
        size="sm"
        variant="outline"
        class="h-7 gap-1 text-xs"
        aria-label="Adicionar do catálogo"
        onClick={() => {
          reset()
          setOpen(true)
        }}
      >
        <Plus aria-hidden="true" class="size-3.5" />
        Catálogo
      </Button>

      <Dialog open={open()} onOpenChange={setOpen}>
        <DialogContent class={ITEM_DIALOG_CONTENT}>
          <DialogHeader>
            <DialogTitle class={ITEM_DIALOG_TITLE}>Adicionar do catálogo</DialogTitle>
          </DialogHeader>
          <form class="space-y-4" onSubmit={submit} noValidate>
            <div class="space-y-1">
              <FieldLabel>item</FieldLabel>
              <div class="flex gap-2">
                <Input
                  value={search()}
                  onInput={(event) => setSearch(event.currentTarget.value)}
                  placeholder="Buscar pelo nome…"
                  aria-label="Buscar no catálogo"
                  class="flex-1"
                />
                <select
                  value={category()}
                  onChange={(event) => setCategory(event.currentTarget.value)}
                  aria-label="Categoria"
                  class="h-9 max-w-[45%] cursor-pointer rounded-sm border border-input bg-transparent px-2 text-sm outline-none focus:ring-2 focus:ring-ring"
                >
                  <option value="">Todas categorias</option>
                  <For each={catalogCategories()}>
                    {(id) => <option value={id}>{categoryLabel(id)}</option>}
                  </For>
                </select>
              </div>
              <Show
                when={filtered().length > 0}
                fallback={
                  <p class="mt-1 rounded-sm border border-border bg-muted px-3 py-4 text-center text-xs text-muted-foreground">
                    Nenhum item.
                  </p>
                }
              >
                <VirtualList
                  class="mt-1 max-h-56 rounded-sm border border-border bg-muted"
                  items={filtered()}
                  estimateSize={34}
                  getKey={(entry) => entry.id}
                  renderItem={(entry) => (
                    <CatalogRow
                      catalog={entry}
                      selected={catalogId() === entry.id}
                      onPick={() => pick(entry.id)}
                    />
                  )}
                />
              </Show>
            </div>

            <Show when={selected()}>{(catalog) => <CatalogPreview catalog={catalog()} />}</Show>

            <div class="grid grid-cols-2 gap-3">
              <FieldFrame name="catalog-quantity" label="Quantidade">
                <NumberInput
                  id="catalog-quantity"
                  value={quantity()}
                  onChange={setQuantity}
                  min={1}
                  max={9999}
                  step={1}
                />
              </FieldFrame>
              <Show when={equipChoices().length > 1}>
                <div class="space-y-2">
                  <label
                    for="catalog-equipped"
                    class="text-sm font-medium leading-none text-foreground"
                  >
                    Equipar
                  </label>
                  <select
                    id="catalog-equipped"
                    value={equipped()}
                    onChange={(event) =>
                      setEquipped(event.currentTarget.value as '' | EquippedSlot)
                    }
                    class="h-9 w-full cursor-pointer rounded-sm border border-input bg-transparent px-2 text-sm outline-none focus:ring-2 focus:ring-ring"
                  >
                    <For each={equipChoices()}>
                      {(option) => <option value={option.value}>{option.label}</option>}
                    </For>
                  </select>
                </div>
              </Show>
            </div>

            <DialogInlineError message={formError()} />
            <ItemDialogFooter>
              <Button type="button" variant="outline" onClick={() => setOpen(false)}>
                Cancelar
              </Button>
              <Button type="submit" disabled={!selected() || pending()}>
                {pending() ? 'Adicionando…' : 'Adicionar'}
              </Button>
            </ItemDialogFooter>
          </form>
        </DialogContent>
      </Dialog>
    </>
  )
}

function CatalogRow(props: { catalog: CatalogItem; selected: boolean; onPick: () => void }) {
  return (
    <button
      type="button"
      onClick={() => props.onPick()}
      aria-pressed={props.selected}
      class="flex w-full items-center justify-between gap-2 px-3 py-1.5 text-left text-sm transition-colors hover:bg-accent aria-pressed:bg-accent aria-pressed:text-grimorio-gold"
    >
      <span class="truncate">{props.catalog.name}</span>
      <span class="shrink-0 text-3xs text-muted-foreground">
        {categoryLabel(props.catalog.category)}
      </span>
    </button>
  )
}

/** The picked entry's headline numbers, so the choice is confirmable without
 *  opening the full item sheet. */
function CatalogPreview(props: { catalog: CatalogItem }) {
  return (
    <div class="rounded-sm border border-border bg-muted px-3 py-2 text-2xs">
      <p class="font-semibold text-grimorio-gold">{props.catalog.name}</p>
      <p class="text-muted-foreground">
        {categoryLabel(props.catalog.category)} • esp {formatLoad(props.catalog.slots)} • T${' '}
        {props.catalog.price}
      </p>
      <Show when={props.catalog.weapon}>
        {(weapon) => (
          <p class="text-muted-foreground">
            dano {weapon().damage} • crit {weapon().critRange}/×{weapon().critMult}
          </p>
        )}
      </Show>
      <Show when={props.catalog.armor}>
        {(armor) => (
          <p class="text-muted-foreground">
            Def +{armor().defense} • penalidade {armor().penalty} •{' '}
            {armor().heavy ? 'pesada' : 'leve'}
          </p>
        )}
      </Show>
      <Show when={props.catalog.shield}>
        {(shield) => (
          <p class="text-muted-foreground">
            Def +{shield().defense} • penalidade {shield().penalty}
          </p>
        )}
      </Show>
    </div>
  )
}
