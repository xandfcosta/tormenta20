import { Search } from 'lucide-react'
import { useState } from 'react'
import { Input } from '@/shared/ui/input'
import { VirtualList } from '@/shared/ui/virtual-list'
import { cn } from '@/shared/lib/utils'
import {
  type PurchaseMap,
  SHOP_CATEGORIES,
  type ShopCategoryKey,
  shopCatalog,
} from './starting-equipment'

const tibarFmt = (v: number) =>
  `T$ ${v.toLocaleString('pt-BR', { maximumFractionDigits: 2 })}`

/**
 * Loja do Equipamento inicial (p140: o dinheiro inicial "pode ser usado para
 * comprar itens"). Search + category chips over the buyable catalog; each row
 * has a −/qty/+ stepper. Buying beyond the remaining T$ is blocked — raising
 * the T$ inicial field (GM) reopens the budget.
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
  const setQty = (id: string, qty: number) => {
    const next = { ...purchases }
    if (qty <= 0) delete next[id]
    else next[id] = qty
    onChange(next)
  }
  return (
    <div className="space-y-2">
      <p className="text-[11px] font-semibold uppercase tracking-wide text-muted-foreground">
        Loja · comprar com T$ iniciais (p140)
      </p>
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
              'rounded-md border px-2 py-0.5 text-xs transition-colors',
              category === c.key
                ? 'border-primary bg-accent font-medium'
                : 'border-border text-muted-foreground hover:bg-accent',
            )}
          >
            {c.label}
          </button>
        ))}
      </div>
      <VirtualList
        items={items}
        getKey={(i) => i.id}
        estimateSize={44}
        gap={4}
        className="max-h-72 p-0.5"
        renderItem={(item) => {
          const qty = purchases[item.id] ?? 0
          const canBuy = remaining >= item.price
          return (
            <div className="flex items-center gap-2 rounded-md border border-border p-2">
              <div className="min-w-0 flex-1">
                <p className="truncate text-xs font-semibold">{item.name}</p>
                <p className="text-[10px] text-muted-foreground">
                  {tibarFmt(item.price)}
                </p>
              </div>
              <div className="flex items-center gap-1">
                <QtyButton
                  label={`Remover ${item.name}`}
                  disabled={qty === 0}
                  onClick={() => setQty(item.id, qty - 1)}
                >
                  −
                </QtyButton>
                <span className="w-6 text-center font-mono text-xs">{qty}</span>
                <QtyButton
                  label={`Comprar ${item.name}`}
                  disabled={!canBuy}
                  onClick={() => setQty(item.id, qty + 1)}
                >
                  +
                </QtyButton>
              </div>
            </div>
          )
        }}
      />
    </div>
  )
}

function QtyButton({
  label,
  disabled,
  onClick,
  children,
}: {
  label: string
  disabled: boolean
  onClick: () => void
  children: React.ReactNode
}) {
  return (
    <button
      type="button"
      aria-label={label}
      disabled={disabled}
      onClick={onClick}
      className={cn(
        'flex size-6 items-center justify-center rounded-md border border-border text-sm transition-colors',
        disabled ? 'opacity-40' : 'hover:bg-accent',
      )}
    >
      {children}
    </button>
  )
}
