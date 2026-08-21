import type { CatalogItem } from '@/shared/api/item-types'
import { Search } from 'lucide-solid'
import { For, Show, createSignal } from 'solid-js'
import { matchesQuery } from '@/shared/lib/fuzzy-filter'
import { cn } from '@/shared/lib/utils'
import { Input } from '@/shared/ui/input'
import { VirtualList } from '@/shared/ui/virtual-list'
import {
  type PurchaseMap,
  SHOP_CATEGORIES,
  type ShopCategoryKey,
  shopCatalog,
} from './starting-equipment'
import { SectionLabel } from '@/shared/ui/section-label'

const tibarFmt = (value: number) =>
  `T$ ${value.toLocaleString('pt-BR', { maximumFractionDigits: 2 })}`

/** One comparable stat per row: dano, Defesa, or the space it eats. */
function itemStat(item: CatalogItem): string {
  if (item.weapon) return item.weapon.damage
  const defense = item.armor?.defense ?? item.shield?.defense
  if (defense !== undefined) return `Defesa +${defense}`
  return `${item.slots} espaço${item.slots === 1 ? '' : 's'}`
}

export type StartingShopProps = {
  purchases: PurchaseMap
  /** May be NEGATIVE when the money was lowered after buying — shown as-is so
   *  the player removes items instead of wondering. */
  remaining: number
  onChange: (next: PurchaseMap) => void
}

/**
 * The starting shop (p140 — o dinheiro inicial pode comprar itens). Search and
 * category chips over the buyable catalog, each row with a −/qty/+ stepper.
 * Buying past the remaining T$ is blocked; raising the T$ field reopens it.
 */
export function StartingShop(props: StartingShopProps) {
  const [query, setQuery] = createSignal('')
  const [category, setCategory] = createSignal<ShopCategoryKey>('all')

  const items = () =>
    shopCatalog(category()).filter((item) => matchesQuery([item.name], query()))
  const hasPurchases = () => Object.values(props.purchases).some((qty) => qty > 0)

  const setQty = (id: string, qty: number) => {
    const next = { ...props.purchases }
    if (qty <= 0) delete next[id]
    else next[id] = qty
    props.onChange(next)
  }

  return (
    <div class="space-y-2 rounded-sm border border-grimorio-iron p-3">
      <SectionLabel>
        Loja{' '}
        <span
          class={cn(
            'normal-case tracking-normal',
            props.remaining < 0 && 'font-semibold text-[color:var(--hp-hurt)]',
          )}
        >
          · restante {tibarFmt(props.remaining)}
        </span>
      </SectionLabel>

      <Show
        when={props.remaining > 0 || hasPurchases()}
        fallback={
          <p class="rounded-sm border border-dashed border-grimorio-iron p-2 text-xs text-muted-foreground">
            Role ou defina seu dinheiro inicial acima para comprar itens.
          </p>
        }
      >
        <div class="relative">
          <Search
            class="pointer-events-none absolute left-2 top-1/2 size-4 -translate-y-1/2 text-muted-foreground"
            aria-hidden="true"
          />
          <Input
            type="search"
            value={query()}
            onInput={(event) => setQuery(event.currentTarget.value)}
            placeholder="Buscar item"
            aria-label="Buscar item"
            class="pl-8"
          />
        </div>

        <div class="flex flex-wrap gap-1.5">
          <For each={SHOP_CATEGORIES}>
            {(shopCategory) => (
              <button
                type="button"
                aria-pressed={category() === shopCategory.key}
                onClick={() => setCategory(shopCategory.key)}
                class={cn(
                  'rounded-sm border px-2.5 py-1 text-xs transition-colors',
                  category() === shopCategory.key
                    ? 'border-grimorio-gold bg-accent font-medium text-grimorio-gold'
                    : 'border-grimorio-iron text-muted-foreground hover:bg-accent',
                )}
              >
                {shopCategory.label}
              </button>
            )}
          </For>
        </div>

        <Show
          when={items().length > 0}
          fallback={
            <p class="py-4 text-center text-xs text-muted-foreground">
              Nenhum item encontrado{query().trim() ? ` para “${query().trim()}”` : ''}.
            </p>
          }
        >
          <VirtualList
            items={items()}
            getKey={(item) => item.id}
            estimateSize={52}
            class="max-h-72 p-0.5"
            renderItem={(item) => (
              <ShopRow
                item={item}
                qty={props.purchases[item.id] ?? 0}
                canBuy={props.remaining >= item.price}
                onQty={(qty) => setQty(item.id, qty)}
              />
            )}
          />
        </Show>
      </Show>
    </div>
  )
}

function ShopRow(props: {
  item: CatalogItem
  qty: number
  canBuy: boolean
  onQty: (qty: number) => void
}) {
  return (
    <div class="flex items-center gap-2 rounded-sm border border-grimorio-iron p-2">
      <div class="min-w-0 flex-1">
        <p class="truncate text-xs font-semibold">{props.item.name}</p>
        <p class="text-3xs text-muted-foreground">
          {tibarFmt(props.item.price)} · {itemStat(props.item)}
        </p>
      </div>
      <div class="flex items-center gap-1">
        <QtyButton
          label={`Remover ${props.item.name}`}
          disabled={props.qty === 0}
          onClick={() => props.onQty(props.qty - 1)}
        >
          −
        </QtyButton>
        <span class="w-6 text-center font-mono text-xs">{props.qty}</span>
        <QtyButton
          label={`Comprar ${props.item.name}`}
          disabled={!props.canBuy}
          onClick={() => props.onQty(props.qty + 1)}
        >
          +
        </QtyButton>
      </div>
    </div>
  )
}

function QtyButton(props: {
  label: string
  disabled: boolean
  onClick: () => void
  children: string
}) {
  return (
    <button
      type="button"
      aria-label={props.label}
      disabled={props.disabled}
      onClick={() => props.onClick()}
      class={cn(
        'flex size-8 items-center justify-center rounded-sm border border-grimorio-iron text-sm transition-colors sm:size-7',
        props.disabled ? 'opacity-40' : 'hover:bg-accent',
      )}
    >
      {props.children}
    </button>
  )
}
