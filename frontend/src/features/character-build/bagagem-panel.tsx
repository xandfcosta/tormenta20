import { cn } from '@/shared/lib/utils'
import {
  type BagGroup,
  type BagLine,
  type PurchaseMap,
  purchasesTotal,
} from './starting-equipment'

const tibarFmt = (v: number) =>
  v.toLocaleString('pt-BR', { maximumFractionDigits: 2 })

/**
 * "Sua bagagem" — live preview of the inventory this step will save, grouped
 * by source. Ghost lines (◇ pendente) mark unmade choices and scroll-focus
 * their chooser. Fully derived from form state; the slots bar and wallet chip
 * are the step's only gauges.
 */
export function BagagemPanel({
  groups,
  slotsUsed,
  slotsCapacity,
  tibar,
  purchases,
  onPurchaseQty,
}: {
  groups: BagGroup[]
  slotsUsed: number
  slotsCapacity: number
  tibar: number
  purchases: PurchaseMap
  onPurchaseQty: (id: string, qty: number) => void
}) {
  const spent = purchasesTotal(purchases)
  const remaining = tibar - spent
  const over = slotsUsed > slotsCapacity
  return (
    <aside className="space-y-2 rounded-lg border border-border bg-card p-3 lg:sticky lg:top-4">
      <p className="font-display text-sm tracking-wide">Sua bagagem</p>
      <SlotsBar used={slotsUsed} capacity={slotsCapacity} over={over} />
      <WalletChip tibar={tibar} spent={spent} remaining={remaining} />
      <div className="space-y-2 border-t border-border pt-2">
        {groups.map((g) => (
          <BagGroupBlock
            key={g.title}
            group={g}
            onPurchaseQty={onPurchaseQty}
            purchases={purchases}
          />
        ))}
      </div>
    </aside>
  )
}

function SlotsBar({
  used,
  capacity,
  over,
}: {
  used: number
  capacity: number
  over: boolean
}) {
  const pct = capacity > 0 ? Math.min(100, (used / capacity) * 100) : 0
  return (
    <div className="space-y-1">
      <div className="h-2 overflow-hidden rounded-full bg-muted">
        <div
          className={cn(
            'h-full rounded-full transition-all',
            over ? 'bg-[color:var(--hp-hurt)]' : 'bg-primary',
          )}
          style={{ width: `${pct}%` }}
        />
      </div>
      <p
        className={cn(
          'text-[11px]',
          over
            ? 'font-semibold text-[color:var(--hp-hurt)]'
            : 'text-muted-foreground',
        )}
      >
        Espaços {used}/{capacity} (10 + 2×FOR)
        {over ? ' — sobrecarregado (p141)' : ''}
      </p>
    </div>
  )
}

function WalletChip({
  tibar,
  spent,
  remaining,
}: {
  tibar: number
  spent: number
  remaining: number
}) {
  return (
    <p
      className={cn(
        'text-xs',
        remaining < 0
          ? 'font-semibold text-[color:var(--hp-hurt)]'
          : 'text-foreground',
      )}
    >
      ⛃ T$ {tibarFmt(tibar)}
      {spent > 0 && (
        <>
          {' '}· gasto {tibarFmt(spent)} →{' '}
          <span className="font-semibold">{tibarFmt(remaining)}</span>
          {remaining < 0 ? ' — remova itens' : ''}
        </>
      )}
    </p>
  )
}

function BagGroupBlock({
  group,
  purchases,
  onPurchaseQty,
}: {
  group: BagGroup
  purchases: PurchaseMap
  onPurchaseQty: (id: string, qty: number) => void
}) {
  return (
    <div className="space-y-0.5">
      <p className="text-[10px] font-semibold uppercase tracking-wide text-muted-foreground">
        {group.title}
        {group.title === 'Kit' ? ' · automático' : ''}
      </p>
      <ul className="space-y-0.5">
        {group.lines.map((line, i) => (
          <BagLineRow
            key={line.kind === 'item' ? `${line.name}-${i}` : line.label}
            line={line}
            isPurchase={group.title === 'Comprado'}
            purchases={purchases}
            onPurchaseQty={onPurchaseQty}
          />
        ))}
      </ul>
    </div>
  )
}

function BagLineRow({
  line,
  isPurchase,
  purchases,
  onPurchaseQty,
}: {
  line: BagLine
  isPurchase: boolean
  purchases: PurchaseMap
  onPurchaseQty: (id: string, qty: number) => void
}) {
  if (line.kind === 'ghost') {
    return (
      <li>
        <button
          type="button"
          onClick={() =>
            document
              .getElementById(line.anchor)
              ?.scrollIntoView({ behavior: 'smooth', block: 'center' })
          }
          className="flex w-full items-center gap-1 rounded border border-dashed border-[color:var(--hp-hurt)]/60 px-1.5 py-0.5 text-left text-[11px] text-[color:var(--hp-hurt)] hover:bg-accent"
        >
          ◇ {line.label} · pendente
        </button>
      </li>
    )
  }
  const qty = line.catalogId ? (purchases[line.catalogId] ?? 0) : 0
  return (
    <li className="flex items-center gap-1 text-[11px]">
      <span className="min-w-0 flex-1 truncate">
        · {line.name}
        {line.qty > 1 ? ` ×${line.qty}` : ''}
        {line.price !== undefined && (
          <span className="text-muted-foreground">
            {' '}· T$ {tibarFmt(line.price * line.qty)}
          </span>
        )}
      </span>
      <span className="shrink-0 text-[10px] text-muted-foreground">
        {line.slots * line.qty > 0 ? `${line.slots * line.qty}e` : '—'}
      </span>
      {isPurchase && line.catalogId && (
        <span className="flex shrink-0 items-center gap-0.5">
          <BagQty label={`Remover ${line.name}`} onClick={() => onPurchaseQty(line.catalogId as string, qty - 1)}>
            −
          </BagQty>
          <BagQty label={`Comprar ${line.name}`} onClick={() => onPurchaseQty(line.catalogId as string, qty + 1)}>
            +
          </BagQty>
        </span>
      )}
    </li>
  )
}

function BagQty({
  label,
  onClick,
  children,
}: {
  label: string
  onClick: () => void
  children: React.ReactNode
}) {
  return (
    <button
      type="button"
      aria-label={label}
      onClick={onClick}
      className="flex size-5 items-center justify-center rounded border border-border text-xs hover:bg-accent"
    >
      {children}
    </button>
  )
}
