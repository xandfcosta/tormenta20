import type { OrigemItemGrant } from '@tormenta20/t20-data'
import { For, type JSX, Show, Switch, Match } from 'solid-js'
import { origemItemGrantsByName } from '@/shared/lib/racas-cache'
import { cn } from '@/shared/lib/utils'
import { Select } from '@/shared/ui/select'
import { parseDiceNotation, rollDice } from './roll-dice'
import { shopCatalog, weaponOptions } from './starting-equipment'

/** Picked value per grant, keyed by the grant's verbatim label. */
export type OrigemItemPicks = Record<string, string>

export type OrigemItemsSectionProps = {
  originName: string
  picks: OrigemItemPicks
  onPick: (label: string, value: string) => void
  onMoneyRoll: (amount: number) => void
}

/** The key a grant's pick is stored under — its name when fixed, else its label. */
export function grantKey(grant: OrigemItemGrant): string {
  return grant.kind === 'fixed' ? grant.name : grant.label
}

/**
 * Itens da origem (p85-95): fixed grants are text, choice grants get a picker,
 * and the T$ grant rolls ONCE into the money field. Leaving a pick unmade is
 * soft — it shows as a ghost line in the bag and can be finished on the sheet.
 */
export function OrigemItemsSection(props: OrigemItemsSectionProps) {
  const grants = () => (props.originName ? origemItemGrantsByName(props.originName) : [])

  return (
    <Show when={props.originName}>
      <div class="space-y-1.5 rounded-md border border-grimorio-iron p-3" id="chooser-origem">
        <p class="font-heading text-[11px] uppercase tracking-[0.16em] text-muted-foreground">
          Origem · {props.originName}
        </p>
        <Show
          when={grants().length > 0}
          fallback={<p class="text-xs text-muted-foreground">Sem itens de origem.</p>}
        >
          <For each={grants()}>
            {(grant) => (
              <OrigemGrantRow
                grant={grant}
                value={props.picks[grantKey(grant)] ?? ''}
                onPick={(value) => props.onPick(grantKey(grant), value)}
                onMoneyRoll={props.onMoneyRoll}
              />
            )}
          </For>
        </Show>
      </div>
    </Show>
  )
}

function OrigemGrantRow(props: {
  grant: OrigemItemGrant
  value: string
  onPick: (value: string) => void
  onMoneyRoll: (amount: number) => void
}) {
  return (
    <Switch>
      <Match when={props.grant.kind === 'fixed' && props.grant}>
        {(grant) => <p class="text-xs">✓ {grant().name}</p>}
      </Match>

      <Match when={props.grant.kind === 'weapon' && props.grant}>
        {(grant) => (
          <PickRow label={grant().label} pending={!props.value}>
            <ItemSelect
              options={grant()
                .categories.flatMap((category) => weaponOptions(category))
                .map((weapon) => ({ value: weapon.id, label: weapon.name }))}
              value={props.value}
              onPick={props.onPick}
              placeholder="Escolher arma"
              label={grant().label}
            />
          </PickRow>
        )}
      </Match>

      <Match when={props.grant.kind === 'anyItem' && props.grant}>
        {(grant) => (
          <PickRow label={grant().label} pending={!props.value}>
            <ItemSelect
              options={shopCatalog('all')
                .filter((item) => item.price <= grant().maxPrice)
                .map((item) => ({
                  value: item.id,
                  label: `${item.name} (T$ ${item.price.toLocaleString('pt-BR')})`,
                }))}
              value={props.value}
              onPick={props.onPick}
              placeholder="Escolher item"
              label={grant().label}
            />
          </PickRow>
        )}
      </Match>

      <Match when={props.grant.kind === 'oneOf' && props.grant}>
        {(grant) => (
          <PickRow label={grant().label} pending={!props.value}>
            <div class="flex flex-wrap gap-1.5">
              <For each={grant().options}>
                {(option) => (
                  <button
                    type="button"
                    aria-pressed={props.value === option}
                    onClick={() => props.onPick(props.value === option ? '' : option)}
                    class={cn(
                      'rounded-md border px-2 py-0.5 text-xs transition-colors',
                      props.value === option
                        ? 'border-grimorio-gold bg-accent font-medium text-grimorio-gold'
                        : 'border-grimorio-iron text-muted-foreground hover:bg-accent',
                    )}
                  >
                    {option}
                  </button>
                )}
              </For>
            </div>
          </PickRow>
        )}
      </Match>

      <Match when={props.grant.kind === 'money' && props.grant}>
        {(grant) => (
          <PickRow label={grant().label} pending={false}>
            <MoneyGrantButton
              dice={grant().dice}
              rolledValue={props.value}
              onRoll={(amount) => {
                props.onPick(String(amount))
                props.onMoneyRoll(amount)
              }}
            />
          </PickRow>
        )}
      </Match>
    </Switch>
  )
}

/** Rolls once. The rolled amount is remembered so re-entering the step (or
 *  walking back to it) cannot re-roll a better result. */
function MoneyGrantButton(props: {
  dice: string
  rolledValue: string
  onRoll: (amount: number) => void
}) {
  const notation = () => parseDiceNotation(props.dice)
  const rolled = () => props.rolledValue !== ''

  return (
    <Show when={notation()}>
      {(dice) => (
        <button
          type="button"
          disabled={rolled()}
          onClick={() => props.onRoll(rollDice(dice().count, dice().sides))}
          class={cn(
            'rounded-md border border-grimorio-iron px-2.5 py-1 text-xs',
            rolled() ? 'opacity-60' : 'hover:bg-accent',
          )}
        >
          {rolled() ? `Rolado: T$ ${props.rolledValue} (somado)` : `🎲 Rolar ${props.dice}`}
        </button>
      )}
    </Show>
  )
}

function ItemSelect(props: {
  options: { value: string; label: string }[]
  value: string
  onPick: (value: string) => void
  placeholder: string
  label: string
}) {
  const selected = () => props.options.find((option) => option.value === props.value) ?? null
  return (
    <Select
      options={props.options}
      value={selected()}
      onChange={(option) => props.onPick(option?.value ?? '')}
      placeholder={props.placeholder}
      size="sm"
      aria-label={props.label}
    />
  )
}

function PickRow(props: { label: string; pending: boolean; children: JSX.Element }) {
  return (
    <div class="space-y-1">
      <p class="text-xs">
        {props.label}
        <Show when={props.pending}>
          <span class="ml-1.5 text-[11px] text-[color:var(--hp-hurt)]">· escolha pendente</span>
        </Show>
      </p>
      {props.children}
    </div>
  )
}
