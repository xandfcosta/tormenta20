import { For, Show } from 'solid-js'
import { cn } from '@/shared/lib/utils'
import {
  type BagGroup,
  type BagLine,
  type PurchaseMap,
  purchasesTotal,
} from './starting-equipment'

const tibarFmt = (value: number) =>
  value.toLocaleString('pt-BR', { maximumFractionDigits: 2 })

export type BagagemPanelProps = {
  groups: BagGroup[]
  slotsUsed: number
  slotsCapacity: number
  tibar: number
  purchases: PurchaseMap
  onPurchaseQty: (id: string, qty: number) => void
}

/**
 * "Sua bagagem" — live preview of the inventory this step will save, grouped by
 * source. Ghost lines (◇ pendente) mark choices not yet made and scroll-focus
 * their chooser. Wholly derived from the draft; the slots bar and the wallet
 * are the step's only gauges.
 */
export function BagagemPanel(props: BagagemPanelProps) {
  const spent = () => purchasesTotal(props.purchases)
  const remaining = () => props.tibar - spent()
  const over = () => props.slotsUsed > props.slotsCapacity

  return (
    <aside class="space-y-2 rounded-sm border border-grimorio-iron bg-muted/20 p-3 lg:sticky lg:top-0">
      <p class="font-heading text-[11px] uppercase tracking-[0.16em] text-grimorio-gold">
        Sua bagagem
      </p>
      <SlotsBar used={props.slotsUsed} capacity={props.slotsCapacity} over={over()} />
      <Wallet tibar={props.tibar} spent={spent()} remaining={remaining()} />
      <div class="space-y-2 border-t border-grimorio-iron pt-2">
        <For each={props.groups}>
          {(group) => (
            <BagGroupBlock
              group={group}
              purchases={props.purchases}
              onPurchaseQty={props.onPurchaseQty}
            />
          )}
        </For>
      </div>
    </aside>
  )
}

function SlotsBar(props: { used: number; capacity: number; over: boolean }) {
  const percent = () =>
    props.capacity > 0 ? Math.min(100, (props.used / props.capacity) * 100) : 0

  return (
    <div class="space-y-1">
      <div
        role="progressbar"
        aria-label="Espaços de inventário"
        aria-valuenow={props.used}
        aria-valuemin={0}
        aria-valuemax={props.capacity}
        class="h-2 overflow-hidden rounded-full bg-grimorio-iron"
      >
        <div
          // `--hp-critical` for overweight, not `--hp-hurt`: amber sits a hair
          // from the gold fill and would read as "full" instead of "too much".
          class={cn(
            'h-full rounded-full transition-all',
            props.over ? 'bg-[var(--hp-critical)]' : 'bg-grimorio-gold',
          )}
          style={{ width: `${percent()}%` }}
        />
      </div>
      <p
        class={cn(
          'text-[11px]',
          props.over ? 'font-semibold text-[color:var(--hp-hurt)]' : 'text-muted-foreground',
        )}
      >
        Espaços {props.used}/{props.capacity} (10 + 2×FOR)
        {props.over ? ' — sobrecarregado (p141)' : ''}
      </p>
    </div>
  )
}

function Wallet(props: { tibar: number; spent: number; remaining: number }) {
  return (
    <p
      class={cn(
        'text-xs',
        props.remaining < 0 ? 'font-semibold text-[color:var(--hp-hurt)]' : 'text-foreground',
      )}
    >
      <span aria-hidden="true">⛃ </span>T$ {tibarFmt(props.tibar)}
      <Show when={props.spent > 0}>
        <> · gasto {tibarFmt(props.spent)} → </>
        <span class="font-semibold">{tibarFmt(props.remaining)}</span>
        <Show when={props.remaining < 0}> — remova itens</Show>
      </Show>
    </p>
  )
}

function BagGroupBlock(props: {
  group: BagGroup
  purchases: PurchaseMap
  onPurchaseQty: (id: string, qty: number) => void
}) {
  return (
    <div class="space-y-0.5">
      <p class="font-heading text-[10px] uppercase tracking-[0.14em] text-muted-foreground">
        {props.group.title}
        {props.group.title === 'Kit' ? ' · automático' : ''}
      </p>
      <ul class="space-y-0.5">
        <For each={props.group.lines}>
          {(line) => (
            <BagLineRow
              line={line}
              isPurchase={props.group.title === 'Comprado'}
              purchases={props.purchases}
              onPurchaseQty={props.onPurchaseQty}
            />
          )}
        </For>
      </ul>
    </div>
  )
}

function BagLineRow(props: {
  line: BagLine
  isPurchase: boolean
  purchases: PurchaseMap
  onPurchaseQty: (id: string, qty: number) => void
}) {
  const item = () => (props.line.kind === 'item' ? props.line : null)

  return (
    <Show when={item()} fallback={<GhostRow line={props.line} />}>
      {(item) => (
        <li class="flex items-center gap-1 text-[11px]">
          <span class="min-w-0 flex-1 truncate">
            · {item().name}
            {item().qty > 1 ? ` ×${item().qty}` : ''}
            <Show when={item().price !== undefined}>
              <span class="text-muted-foreground">
                {' '}
                · T$ {tibarFmt((item().price ?? 0) * item().qty)}
              </span>
            </Show>
          </span>
          <span class="shrink-0 text-[10px] text-muted-foreground">
            {item().slots * item().qty > 0 ? `${item().slots * item().qty} esp.` : '—'}
          </span>
          <Show when={props.isPurchase && item().catalogId}>
            {(catalogId) => (
              <span class="flex shrink-0 items-center gap-0.5">
                <QtyStep
                  label={`Remover ${item().name}`}
                  onClick={() =>
                    props.onPurchaseQty(catalogId(), (props.purchases[catalogId()] ?? 0) - 1)
                  }
                >
                  −
                </QtyStep>
                <QtyStep
                  label={`Comprar ${item().name}`}
                  onClick={() =>
                    props.onPurchaseQty(catalogId(), (props.purchases[catalogId()] ?? 0) + 1)
                  }
                >
                  +
                </QtyStep>
              </span>
            )}
          </Show>
        </li>
      )}
    </Show>
  )
}

/** A choice not yet made. Clicking it walks the player to the chooser that
 *  fills it, which is the whole point of showing the hole. */
function GhostRow(props: { line: BagLine }) {
  const ghost = () => (props.line.kind === 'ghost' ? props.line : null)
  return (
    <Show when={ghost()}>
      {(line) => (
        <li>
          <button
            type="button"
            onClick={() =>
              document
                .getElementById(line().anchor)
                ?.scrollIntoView({ behavior: 'smooth', block: 'center' })
            }
            class="flex w-full items-center gap-1 rounded-md border border-dashed border-[color:var(--hp-hurt)]/60 px-1.5 py-0.5 text-left text-[11px] text-[color:var(--hp-hurt)] hover:bg-accent"
          >
            ◇ {line().label} · pendente
          </button>
        </li>
      )}
    </Show>
  )
}

function QtyStep(props: { label: string; onClick: () => void; children: string }) {
  return (
    <button
      type="button"
      aria-label={props.label}
      onClick={() => props.onClick()}
      class="flex size-5 items-center justify-center rounded-md border border-grimorio-iron text-xs hover:bg-accent"
    >
      {props.children}
    </button>
  )
}
