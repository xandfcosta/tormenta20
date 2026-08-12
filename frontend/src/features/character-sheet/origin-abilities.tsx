import { useQueryClient } from '@tanstack/solid-query'
import type { OriginBenefit, OriginDefinition } from '@tormenta20/t20-data'
import { Check } from 'lucide-solid'
import { For, Show, createMemo, createSignal } from 'solid-js'
import type { Character } from '@/shared/api/api'
import { getOrigin } from '@/shared/lib/abilities-cache'
import { cn } from '@/shared/lib/utils'
import { toggleWithLimit } from './choice-lists'
import { choiceActions } from './choice-mutations'
import { type CardFocus, CollapsibleAbilityCard } from './collapsible-ability-card'
import { parseChoices } from './parse-choices'

const ORIGIN_BENEFIT_LIMIT = 2

/**
 * Origin section — the player picks `ORIGIN_BENEFIT_LIMIT` benefits out of the
 * origin's pool plus its unique power. An origin id missing from the catalog
 * degrades to a message instead of taking the sheet down.
 */
export function OriginAbilitySection(props: {
  character: Character
  focus: CardFocus
  pending: number
}) {
  const origin = () => getOrigin(props.character.origin)

  return (
    <CollapsibleAbilityCard
      id="origem"
      title={`Origem: ${origin()?.name ?? props.character.origin}`}
      pending={props.pending}
      focus={props.focus}
    >
      <Show
        when={origin()}
        fallback={<p class="text-xs italic text-muted-foreground">Origem não está no catálogo.</p>}
        keyed
      >
        {(definition) => <OriginPicker origin={definition} character={props.character} />}
      </Show>
    </CollapsibleAbilityCard>
  )
}

function OriginPicker(props: { origin: OriginDefinition; character: Character }) {
  const queryClient = useQueryClient()
  const [pending, setPending] = createSignal(false)

  const pool = createMemo<OriginBenefit[]>(() => [
    ...props.origin.benefits,
    props.origin.poderUnico,
  ])
  const selected = createMemo(() => {
    const ids = new Set(pool().map((b) => b.id))
    return parseChoices(props.character.originChoices).filter((id) => ids.has(id))
  })
  const remaining = () => ORIGIN_BENEFIT_LIMIT - selected().length

  const toggle = async (benefitId: string) => {
    const next = toggleWithLimit(selected(), benefitId, ORIGIN_BENEFIT_LIMIT)
    // Same length means the cap refused the pick (a real toggle always changes
    // it) — don't spend a request on a no-op.
    if (next.length === selected().length) return
    setPending(true)
    try {
      await choiceActions(queryClient, props.character.id).setOriginChoices(next)
    } catch {
      // choiceActions already rolled back and told the player.
    } finally {
      setPending(false)
    }
  }

  return (
    <>
      <p class="mb-2 text-[11px] text-muted-foreground">
        Escolha {ORIGIN_BENEFIT_LIMIT} benefícios (perícia, poder geral, ou o poder único da
        origem). Restantes: <span class="font-semibold">{Math.max(0, remaining())}</span>
      </p>
      <ul class="space-y-1.5">
        <For each={pool()}>
          {(benefit) => (
            <OriginBenefitRow
              benefit={benefit}
              isUnique={benefit.id === props.origin.poderUnico.id}
              selected={selected().includes(benefit.id)}
              atLimit={remaining() <= 0}
              disabled={pending()}
              onToggle={() => void toggle(benefit.id)}
            />
          )}
        </For>
      </ul>
    </>
  )
}

function OriginBenefitRow(props: {
  benefit: OriginBenefit
  isUnique: boolean
  selected: boolean
  atLimit: boolean
  disabled: boolean
  onToggle: () => void
}) {
  const blocked = () => !props.selected && props.atLimit
  return (
    <li
      class={cn(
        'flex gap-2 rounded-sm border border-border p-2',
        props.selected && 'bg-muted',
      )}
    >
      <button
        type="button"
        onClick={() => props.onToggle()}
        disabled={props.disabled || blocked()}
        aria-pressed={props.selected}
        aria-label={`${props.selected ? 'Remover' : 'Selecionar'} benefício: ${props.benefit.name}`}
        class={cn(
          'mt-0.5 flex size-4 shrink-0 items-center justify-center rounded-sm border border-border text-[10px]',
          props.selected ? 'bg-muted text-foreground' : 'hover:bg-muted',
          (props.disabled || blocked()) && 'cursor-not-allowed opacity-40',
        )}
      >
        <Show when={props.selected}>
          <Check aria-hidden="true" class="size-3" />
        </Show>
      </button>
      <div class="min-w-0 flex-1">
        <div class="flex flex-wrap items-center gap-1">
          <p class="text-xs font-semibold text-grimorio-gold">{props.benefit.name}</p>
          <span
            class={cn(
              'rounded-sm px-1 text-[9px] uppercase tracking-wide',
              props.benefit.kind === 'pericia'
                ? 'bg-emerald-500/20 text-emerald-100'
                : 'bg-violet-500/20 text-violet-100',
            )}
          >
            {props.benefit.kind === 'pericia' ? 'Perícia' : 'Poder'}
          </span>
          <Show when={props.isUnique}>
            <span class="rounded-sm bg-muted px-1 text-[9px] font-semibold uppercase tracking-wide text-foreground">
              Único
            </span>
          </Show>
        </div>
        <p class="mt-0.5 text-[11px] leading-snug text-muted-foreground">
          {props.benefit.description}
        </p>
      </div>
    </li>
  )
}
