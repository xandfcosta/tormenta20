import { Search, X } from 'lucide-react'
import { useState } from 'react'
import { type CatalogItem, getCatalogItem } from '@tormenta20/t20-data'
import { Input } from '@/shared/ui/input'
import { VirtualList } from '@/shared/ui/virtual-list'
import { cn } from '@/shared/lib/utils'
import {
  type PurchaseMap,
  purchasesTotal,
  SHOP_CATEGORIES,
  type ShopCategoryKey,
  shopCatalog,
} from './starting-equipment'

const tibarFmt = (v: number) =>
  `T$ ${v.toLocaleString('pt-BR', { maximumFractionDigits: 2 })}`

/** One comparable stat token per item (dano, Defesa, or espaço). */
function itemStat(item: CatalogItem): string {
  if (item.weapon) return item.weapon.damage
  const defense =
    item.armor?.defense ?? item.shield?.defense ?? undefined
  if (defense !== undefined) return `Defesa +${defense}`
  return `${item.slots} espaço${item.slots === 1 ? '' : 's'}`
}

/**
 * Loja do Equipamento inicial (p140: o dinheiro inicial "pode ser usado para
 * comprar itens"). Search + category chips over the buyable catalog; each row
 * has a −/qty/+ stepper. Buying beyond the remaining T$ is blocked — raising
 * the T$ inicial field (GM) reopens the budget. `remaining` may be NEGATIVE
 * (money lowered after buying): shown truthfully so the player removes items.
 */
export function StartingShop({
  purchases,
  remaining,
  onChange,
}: {
  purchases: PurchaseMap
  remaining: number
  onChange: (next: PurchaseMap) => void
}) {
  const [query, setQuery] = useState('')
  const [category, setCategory] = useState<ShopCategoryKey>('all')
  const q = query.trim().toLowerCase()
  const items = shopCatalog(category).filter(
    (i) => !q || i.name.toLowerCase().includes(q),
  )
  const hasPurchases = Object.values(purchases).some((qty) => qty > 0)
  const setQty = (id: string, qty: number) => {
    const next = { ...purchases }
    if (qty <= 0) delete next[id]
    else next[id] = qty
    onChange(next)
  }
  return (
    <div className="space-y-2">
      <p className="text-[11px] font-semibold uppercase tracking-wide text-muted-foreground">
        Loja{' '}
        <span
          className={cn(
            'normal-case tracking-normal',
            remaining < 0 && 'font-semibold text-[color:var(--hp-hurt)]',
          )}
        >
          · restante {tibarFmt(remaining)}
        </span>
      </p>
      {remaining <= 0 && !hasPurchases ? (
        <p className="rounded-md border border-dashed border-border p-2 text-xs text-muted-foreground">
          Role ou defina seu dinheiro inicial acima para comprar itens.
        </p>
      ) : (
        <>
          <PurchasedSummary purchases={purchases} onRemove={(id) => setQty(id, 0)} />
          <div className="relative">
            <Search className="pointer-events-none absolute left-2 top-1/2 size-3.5 -translate-y-1/2 text-muted-foreground" />
            <Input
              value={query}
              onChange={(e) => setQuery(e.target.value)}
              placeholder="Buscar item"
              aria-label="Buscar item"
              className="h-8 pl-7 text-xs"
            />
          </div>
          <div className="flex flex-wrap gap-1.5">
            {SHOP_CATEGORIES.map((c) => (
              <button
                key={c.key}
                type="button"
                aria-pressed={category === c.key}
                onClick={() => setCategory(c.key)}
                className={cn(
                  'rounded-md border px-2.5 py-1.5 text-xs transition-colors sm:py-1',
                  category === c.key
                    ? 'border-primary bg-accent font-medium'
                    : 'border-border text-muted-foreground hover:bg-accent',
                )}
              >
                {c.label}
              </button>
            ))}
          </div>
          {items.length === 0 ? (
            <p className="py-4 text-center text-xs text-muted-foreground">
              Nenhum item encontrado{q ? ` para “${query.trim()}”` : ''}.
            </p>
          ) : (
            <VirtualList
              items={items}
              getKey={(i) => i.id}
              estimateSize={48}
              gap={4}
              className="max-h-72 p-0.5"
              renderItem={(item) => (
                <ShopRow
                  item={item}
                  qty={purchases[item.id] ?? 0}
                  canBuy={remaining >= item.price}
                  onQty={(qty) => setQty(item.id, qty)}
                />
              )}
            />
          )}
        </>
      )}
    </div>
  )
}

/** Compact review of everything bought: name ×qty · subtotal, with remove. */
function PurchasedSummary({
  purchases,
  onRemove,
}: {
  purchases: PurchaseMap
  onRemove: (id: string) => void
}) {
  const rows = Object.entries(purchases).filter(([, qty]) => qty > 0)
  if (rows.length === 0) return null
  return (
    <div className="space-y-1 rounded-md border border-border p-2">
      <p className="text-[10px] font-semibold uppercase tracking-wide text-muted-foreground">
        Comprado · {tibarFmt(purchasesTotal(purchases))}
      </p>
      <ul className="flex flex-wrap gap-1.5">
        {rows.map(([id, qty]) => {
          const item = getCatalogItem(id)
          if (!item) return null
          return (
            <li
              key={id}
              className="flex items-center gap-1 rounded-md border border-border px-1.5 py-0.5 text-xs"
            >
              {item.name}
              {qty > 1 ? ` ×${qty}` : ''}
              <span className="text-[10px] text-muted-foreground">
                · {tibarFmt(item.price * qty)}
              </span>
              <button
                type="button"
                aria-label={`Remover ${item.name}`}
                onClick={() => onRemove(id)}
                className="ml-0.5 rounded p-0.5 text-muted-foreground hover:bg-accent hover:text-foreground"
              >
                <X className="size-3" />
              </button>
            </li>
          )
        })}
      </ul>
    </div>
  )
}

function ShopRow({
  item,
  qty,
  canBuy,
  onQty,
}: {
  item: CatalogItem
  qty: number
  canBuy: boolean
  onQty: (qty: number) => void
}) {
  return (
    <div className="flex items-center gap-2 rounded-md border border-border p-2">
      <div className="min-w-0 flex-1">
        <p className="truncate text-xs font-semibold">{item.name}</p>
        <p className="text-[10px] text-muted-foreground">
          {tibarFmt(item.price)} · {itemStat(item)}
        </p>
      </div>
      <div className="flex items-center gap-1">
        <QtyButton
          label={`Remover ${item.name}`}
          disabled={qty === 0}
          onClick={() => onQty(qty - 1)}
        >
          −
        </QtyButton>
        <span className="w-6 text-center font-mono text-xs">{qty}</span>
        <QtyButton
          label={`Comprar ${item.name}`}
          disabled={!canBuy}
          title={canBuy ? undefined : `Saldo insuficiente (${tibarFmt(item.price)})`}
          onClick={() => onQty(qty + 1)}
        >
          +
        </QtyButton>
      </div>
    </div>
  )
}

function QtyButton({
  label,
  disabled,
  title,
  onClick,
  children,
}: {
  label: string
  disabled: boolean
  title?: string
  onClick: () => void
  children: React.ReactNode
}) {
  return (
    <button
      type="button"
      aria-label={label}
      title={title}
      disabled={disabled}
      onClick={onClick}
      className={cn(
        'flex size-8 items-center justify-center rounded-md border border-border text-sm transition-colors sm:size-7',
        disabled ? 'opacity-40' : 'hover:bg-accent',
      )}
    >
      {children}
    </button>
  )
}
