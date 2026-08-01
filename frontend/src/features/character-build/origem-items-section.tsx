import {
  type OrigemItemGrant,
  origemItemGrantsByName,
} from '@tormenta20/t20-data'
import { Combobox } from '@/shared/ui/combobox'
import { cn } from '@/shared/lib/utils'
import { shopCatalog, weaponOptions } from './starting-equipment'

/** Picked value per grant, keyed by the grant's verbatim label. */
export type OrigemItemPicks = Record<string, string>

/**
 * Itens da origem (book p85-95) — fixed grants render as text; choice grants
 * ("Arma marcial", "até T$ N", "X OU Y", "(escolha)") render pickers, and the
 * T$-dice grant rolls once into the money field. Under-filling is soft.
 */
export function OrigemItemsSection({
  originName,
  picks,
  onPick,
  onMoneyRoll,
}: {
  originName: string
  picks: OrigemItemPicks
  onPick: (label: string, value: string) => void
  onMoneyRoll: (label: string, amount: number) => void
}) {
  if (!originName) return null
  const grants = origemItemGrantsByName(originName)
  return (
    <div className="space-y-1.5">
      <p className="text-[11px] font-semibold uppercase tracking-wide text-muted-foreground">
        Origem · {originName}
      </p>
      {grants.length === 0 ? (
        <p className="text-sm text-muted-foreground">Sem itens de origem.</p>
      ) : (
        grants.map((g) => (
          <OrigemGrantRow
            key={g.kind === 'fixed' ? g.name : g.label}
            grant={g}
            value={picks[grantKey(g)] ?? ''}
            onPick={(v) => onPick(grantKey(g), v)}
            onMoneyRoll={onMoneyRoll}
          />
        ))
      )}
    </div>
  )
}

export function grantKey(g: OrigemItemGrant): string {
  return g.kind === 'fixed' ? g.name : g.label
}

function rollDice(count: number, sides: number): number {
  let total = 0
  for (let i = 0; i < count; i++) total += 1 + Math.floor(Math.random() * sides)
  return total
}

function OrigemGrantRow({
  grant,
  value,
  onPick,
  onMoneyRoll,
}: {
  grant: OrigemItemGrant
  value: string
  onPick: (value: string) => void
  onMoneyRoll: (label: string, amount: number) => void
}) {
  switch (grant.kind) {
    case 'fixed':
      return <p className="text-sm">✓ {grant.name}</p>
    case 'weapon': {
      const options = grant.categories.flatMap((c) => weaponOptions(c))
      return (
        <PickRow label={grant.label} pending={!value}>
          <Combobox
            options={options.map((w) => ({ value: w.id, label: w.name }))}
            value={value}
            onChange={onPick}
            placeholder="Escolher arma"
            searchPlaceholder="Buscar arma…"
            emptyMessage="Nenhuma."
            allowClear
            clearLabel="Nenhuma"
          />
        </PickRow>
      )
    }
    case 'anyItem': {
      const options = shopCatalog('all').filter((i) => i.price <= grant.maxPrice)
      return (
        <PickRow label={grant.label} pending={!value}>
          <Combobox
            options={options.map((i) => ({
              value: i.id,
              label: `${i.name} (T$ ${i.price.toLocaleString('pt-BR')})`,
            }))}
            value={value}
            onChange={onPick}
            placeholder="Escolher item"
            searchPlaceholder="Buscar item…"
            emptyMessage="Nenhum."
            allowClear
            clearLabel="Nenhum"
          />
        </PickRow>
      )
    }
    case 'oneOf':
      return (
        <PickRow label={grant.label} pending={!value}>
          <div className="flex flex-wrap gap-1.5">
            {grant.options.map((opt) => (
              <button
                key={opt}
                type="button"
                aria-pressed={value === opt}
                onClick={() => onPick(value === opt ? '' : opt)}
                className={cn(
                  'rounded-md border px-2 py-0.5 text-xs transition-colors',
                  value === opt
                    ? 'border-primary bg-accent font-medium'
                    : 'border-border text-muted-foreground hover:bg-accent',
                )}
              >
                {opt}
              </button>
            ))}
          </div>
        </PickRow>
      )
    case 'money': {
      const rolled = value !== ''
      const [count, sides] = grant.dice.split('d').map(Number)
      return (
        <PickRow label={grant.label} pending={false}>
          <button
            type="button"
            disabled={rolled}
            onClick={() => {
              const amount = rollDice(count, sides)
              onPick(String(amount))
              onMoneyRoll(grant.label, amount)
            }}
            className={cn(
              'rounded-md border border-border px-2.5 py-1 text-xs',
              rolled ? 'opacity-60' : 'hover:bg-accent',
            )}
          >
            {rolled ? `Rolado: T$ ${value} (somado)` : `🎲 Rolar ${grant.dice}`}
          </button>
        </PickRow>
      )
    }
  }
}

function PickRow({
  label,
  pending,
  children,
}: {
  label: string
  pending: boolean
  children: React.ReactNode
}) {
  return (
    <div className="space-y-1">
      <p className="text-sm">
        {label}
        {pending && (
          <span className="ml-1.5 text-[11px] text-[color:var(--hp-hurt)]">
            · escolha pendente
          </span>
        )}
      </p>
      {children}
    </div>
  )
}
