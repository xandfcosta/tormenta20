import { caminhoSlotFor } from '@/shared/rules/abilities-caminhos'
import { Check, Lock, Search, X } from 'lucide-solid'
import { For, Index, Show, createMemo, createSignal } from 'solid-js'
import {
  type ClassEntry,
  type PowerOption,
  powerBlockedReason,
  powerChoiceOptions,
} from '@/entities/character/class-powers'
import { useForge } from '@/features/character-build/forge-context'
import { draftDeformidadeHeldPower } from '@/features/character-build/grant-helpers'
import {
  type PowerFacet,
  draftPowerPool,
  facetTally,
  filterPowers,
  powerLedger,
} from '@/features/character-build/power-pool'
import { devotoOptionsFor } from '@/shared/lib/abilities-cache'
import { cn } from '@/shared/lib/utils'
import { Input } from '@/shared/ui/input'
import { Select } from '@/shared/ui/select'
import { VirtualList } from '@/shared/ui/virtual-list'

const SOURCE_LABEL: Record<PowerOption['source'], string> = {
  class: 'classe',
  general: 'geral',
  tormenta: 'tormenta',
}

/**
 * Third step: poderes. The catalogue on the left, what the character has
 * actually taken on the right — including the empty slots still owed, so
 * "quantos faltam" is a thing you SEE rather than a number you read.
 *
 * The sub-choices (totem, escola, arma) live on the right too: they belong to
 * the power you own, not to the row you happened to click in the catalogue.
 */
export function PoderesStep() {
  const { draft } = useForge()
  const [query, setQuery] = createSignal('')
  const [facet, setFacet] = createSignal<PowerFacet>('all')

  const pool = createMemo(() => draftPowerPool(draft.values, draft.raceChoices))
  const byId = createMemo(() => new Map(pool().map((option) => [option.id, option])))
  const ledger = () =>
    powerLedger(draft.values.classes, draft.values.classPowers, draft.values.powerChoices, pool())

  /** Power already gained by the Deformidade swap: owned for prereqs (p23),
   *  but never re-pickable. */
  const held = () => draftDeformidadeHeldPower(draft.values.races, draft.raceChoices)
  const owned = () => {
    const ids = new Set(draft.values.classPowers)
    const swap = held()
    if (swap) ids.add(swap)
    return ids
  }

  const toggle = (id: string) => {
    const chosen = draft.values.classPowers
    draft.setValue(
      'classPowers',
      chosen.includes(id) ? chosen.filter((x) => x !== id) : [...chosen, id],
    )
  }

  const setChoice = (powerId: string, ids: string[]) =>
    draft.setValue('powerChoices', { ...draft.values.powerChoices, [powerId]: ids })

  return (
    <section class="flex min-h-0 flex-1 flex-col gap-3" aria-labelledby="forge-step-poderes">
      <h2 id="forge-step-poderes" class="sr-only">
        Poderes
      </h2>

      <ClassChoiceRows />

      <Show
        when={draft.values.classes[0]?.className}
        fallback={
          <p class="flex flex-1 items-center justify-center text-sm text-muted-foreground">
            Escolha uma classe primeiro — é ela que abre a lista de poderes.
          </p>
        }
      >
        <Show
          when={ledger().total > 0}
          fallback={
            // Not "no 1º nível": a Guerreiro 1 / Ladino 1 is a level-2
            // character and still earns nothing, because the slot belongs to
            // the SECOND level of a class, not to the character's total.
            <p class="flex flex-1 items-center justify-center text-center text-sm text-muted-foreground">
              Nenhuma vaga de poder ainda — a primeira chega no 2º nível de uma classe.
            </p>
          }
        >
          <div class="grid gap-4 lg:min-h-0 lg:flex-1 lg:grid-cols-[1.1fr_1fr]">
            <Catalogue
              pool={pool()}
              facet={facet()}
              query={query()}
              onFacet={setFacet}
              onQuery={setQuery}
              chosen={draft.values.classPowers}
              held={held()}
              canAdd={ledger().remaining > 0}
              level={draft.values.classes[0]?.level ?? 1}
              onToggle={toggle}
            />
            <TakenPowers
              chosenIds={draft.values.classPowers}
              byId={byId()}
              powerChoices={draft.values.powerChoices}
              ledger={ledger()}
              ownedCount={owned().size}
              onRemove={toggle}
              onChoice={setChoice}
            />
          </div>
        </Show>
      </Show>
    </section>
  )
}

/** Caminho and devoto — a class-level pick, not a slot spend, so it sits above
 *  the pool rather than competing with it. */
function ClassChoiceRows() {
  const { draft } = useForge()
  const rows = () => draft.values.classes.filter((entry) => entry.className)

  const set = (className: string, field: 'caminho' | 'devoto', value: string) =>
    draft.setValue('classChoices', {
      ...draft.values.classChoices,
      [className]: { ...draft.values.classChoices[className], [field]: value || undefined },
    })

  return (
    <For each={rows()}>
      {(entry) => (
        <ClassChoiceRow entry={entry} onChoice={(field, value) => set(entry.className, field, value)} />
      )}
    </For>
  )
}

function ClassChoiceRow(props: {
  entry: ClassEntry
  onChoice: (field: 'caminho' | 'devoto', value: string) => void
}) {
  const { draft } = useForge()
  const caminho = () => {
    const slot = caminhoSlotFor(props.entry.className)
    return slot && props.entry.level >= slot.minLevel ? slot : null
  }
  const devoto = () => devotoOptionsFor(props.entry.className)
  const picked = () => draft.values.classChoices[props.entry.className] ?? {}

  const option = (id: string | undefined, options: { id: string; name: string }[]) => {
    const match = id ? options.find((o) => o.id === id) : undefined
    return match ? { value: match.id, label: match.name } : null
  }

  return (
    <Show when={caminho() || devoto()}>
      <div class="flex flex-wrap items-end gap-3">
        <Show when={caminho()}>
          {(slot) => (
            <ChoiceField label={`Caminho de ${props.entry.className}`}>
              <Select
                options={slot().options.map((o) => ({ value: o.id, label: o.name }))}
                value={option(picked().caminho, [...slot().options])}
                onChange={(next) => props.onChoice('caminho', next?.value ?? '')}
                placeholder="Escolher caminho"
                size="sm"
                aria-label={`Caminho de ${props.entry.className}`}
              />
            </ChoiceField>
          )}
        </Show>
        <Show when={devoto()}>
          {(gods) => (
            <ChoiceField label={`Devoto (${props.entry.className})`}>
              <Select
                options={gods().map((d) => ({ value: d.id, label: d.name }))}
                value={option(picked().devoto, [...gods()])}
                onChange={(next) => props.onChoice('devoto', next?.value ?? '')}
                placeholder="Escolher deus"
                size="sm"
                aria-label={`Devoto de ${props.entry.className}`}
              />
            </ChoiceField>
          )}
        </Show>
      </div>
    </Show>
  )
}

function ChoiceField(props: { label: string; children: import('solid-js').JSX.Element }) {
  return (
    <div class="space-y-1">
      <p class="font-heading text-3xs uppercase tracking-[0.14em] text-muted-foreground">
        {props.label}
      </p>
      {props.children}
    </div>
  )
}

function Catalogue(props: {
  pool: PowerOption[]
  facet: PowerFacet
  query: string
  onFacet: (facet: PowerFacet) => void
  onQuery: (query: string) => void
  chosen: string[]
  held?: string
  canAdd: boolean
  level: number
  onToggle: (id: string) => void
}) {
  const tally = () => facetTally(props.pool)
  const shown = () => filterPowers(props.pool, props.facet, props.query)

  const facets = (): [PowerFacet, string][] => [
    ['all', 'Todos'],
    ['class', 'Da classe'],
    ['general', 'Gerais'],
    ...(tally().tormenta > 0 ? ([['tormenta', 'Tormenta']] as [PowerFacet, string][]) : []),
  ]

  return (
    <div class="flex flex-col gap-2 lg:min-h-0">
      <div class="relative shrink-0">
        <Search
          class="pointer-events-none absolute left-2 top-1/2 size-4 -translate-y-1/2 text-muted-foreground"
          aria-hidden="true"
        />
        <Input
          type="search"
          value={props.query}
          onInput={(event) => props.onQuery(event.currentTarget.value)}
          placeholder="Buscar poder"
          class="pl-8"
          aria-label="Buscar poder"
        />
      </div>

      <div class="flex flex-wrap gap-1.5">
        <For each={facets()}>
          {([value, label]) => (
            <button
              type="button"
              aria-pressed={props.facet === value}
              onClick={() => props.onFacet(value)}
              class={cn(
                'rounded-sm border px-2.5 py-1 text-xs transition-colors',
                props.facet === value
                  ? 'border-grimorio-gold bg-accent font-medium text-grimorio-gold'
                  : 'border-grimorio-iron text-muted-foreground hover:bg-accent',
              )}
            >
              {label} <span class="tabular-nums text-muted-foreground">{tally()[value]}</span>
            </button>
          )}
        </For>
      </div>

      <Show
        when={shown().length > 0}
        fallback={
          <p class="py-6 text-center text-xs text-muted-foreground">
            Nenhum poder corresponde à busca.
          </p>
        }
      >
        <VirtualList
          items={shown()}
          getKey={(option) => option.id}
          estimateSize={64}
          class="max-h-[45vh] rounded-sm border border-grimorio-iron p-1 lg:max-h-none lg:min-h-0 lg:flex-1"
          renderItem={(option) => (
            <PowerRow
              option={option}
              level={props.level}
              selected={props.chosen.includes(option.id)}
              heldViaDeformidade={option.id === props.held}
              canAdd={props.canAdd}
              onToggle={() => props.onToggle(option.id)}
            />
          )}
        />
      </Show>
    </div>
  )
}

function PowerRow(props: {
  option: PowerOption
  level: number
  selected: boolean
  heldViaDeformidade: boolean
  canAdd: boolean
  onToggle: () => void
}) {
  const { draft } = useForge()
  const blocked = () =>
    props.heldViaDeformidade
      ? 'já obtido pela Deformidade'
      : powerBlockedReason(props.option, props.level, {
          chosenIds: new Set(draft.values.classPowers),
          classChoices: draft.values.classChoices,
        })
  const locked = () => !props.selected && (!!blocked() || !props.canAdd)

  return (
    <button
      type="button"
      aria-pressed={props.selected}
      disabled={locked()}
      onClick={() => props.onToggle()}
      class={cn(
        'flex w-full items-start gap-2 rounded-sm border p-2 text-left transition-colors',
        props.selected
          ? 'border-grimorio-gold bg-accent'
          : locked()
            ? 'border-grimorio-iron opacity-50'
            : 'border-grimorio-iron hover:bg-accent',
      )}
    >
      <span class="mt-0.5 flex size-4 shrink-0 items-center justify-center rounded-none border border-grimorio-iron">
        <Show when={props.selected}>
          <Check aria-hidden="true" class="size-3 text-grimorio-gold" />
        </Show>
        <Show when={!props.selected && blocked()}>
          <Lock aria-hidden="true" class="size-2.5 text-muted-foreground" />
        </Show>
      </span>
      <span class="min-w-0 flex-1">
        <span class="flex flex-wrap items-baseline gap-1.5">
          <span class="text-xs font-semibold">{props.option.name}</span>
          <span
            class={cn(
              'shrink-0 text-3xs',
              props.option.source === 'tormenta'
                ? 'text-[color:var(--hp-hurt)]'
                : 'text-muted-foreground',
            )}
          >
            · {SOURCE_LABEL[props.option.source]}
          </span>
          <Show when={blocked()}>
            {(reason) => (
              <span class="rounded-none border border-grimorio-iron px-1 text-4xs text-muted-foreground">
                {reason()}
              </span>
            )}
          </Show>
        </span>
        <span class="line-clamp-2 block text-2xs leading-snug text-muted-foreground">
          {props.option.description}
        </span>
      </span>
    </button>
  )
}

/** The right column: what the character took, and the slots still owed. */
function TakenPowers(props: {
  chosenIds: string[]
  byId: Map<string, PowerOption>
  powerChoices: Record<string, string[]>
  ledger: { total: number; used: number; remaining: number }
  ownedCount: number
  onRemove: (id: string) => void
  onChoice: (powerId: string, ids: string[]) => void
}) {
  const empties = () => Array.from({ length: props.ledger.remaining })

  return (
    <section
      aria-label="Seus poderes"
      class="flex flex-col gap-2 lg:min-h-0 lg:overflow-y-auto lg:pl-4 lg:border-l lg:border-grimorio-iron"
    >
      <div class="flex flex-wrap items-baseline gap-x-2">
        <p class="font-heading text-2xs uppercase tracking-[0.16em] text-grimorio-gold">
          Seus poderes
        </p>
        <p aria-hidden="true" class="flex gap-0.5 text-3xs text-grimorio-gold">
          <Index each={Array.from({ length: props.ledger.total })}>
            {(_, i) => <span>{i < props.ledger.used ? '●' : '○'}</span>}
          </Index>
        </p>
        <p class="text-2xs text-muted-foreground">
          {props.ledger.used} de {props.ledger.total}
        </p>
      </div>

      <For each={props.chosenIds}>
        {(id) => (
          <Show
            when={props.byId.get(id)}
            keyed
            // A pick whose id left the pool — the player changed class after
            // choosing — still EATS a slot, so hiding it would leave "2 de 2"
            // over an empty column with no way out.
            fallback={<OrphanPowerCard id={id} onRemove={() => props.onRemove(id)} />}
          >
            {(option) => (
              <TakenPowerCard
                option={option}
                choiceIds={props.powerChoices[id] ?? []}
                canAddMore={props.ledger.remaining > 0}
                onRemove={() => props.onRemove(id)}
                onChoice={(ids) => props.onChoice(id, ids)}
              />
            )}
          </Show>
        )}
      </For>

      <For each={empties()}>
        {() => (
          <p class="rounded-sm border border-dashed border-grimorio-iron p-3 text-center text-2xs text-muted-foreground">
            Vaga livre — escolha um poder ao lado.
          </p>
        )}
      </For>

      <Show when={props.chosenIds.length === 0 && props.ledger.total === 0}>
        <p class="text-2xs text-muted-foreground">Nenhuma vaga de poder neste nível.</p>
      </Show>
    </section>
  )
}

/** A pick that no longer belongs to the current class list. It keeps its slot
 *  until the player says otherwise — nothing is thrown away behind their back. */
function OrphanPowerCard(props: { id: string; onRemove: () => void }) {
  return (
    <div class="space-y-1 rounded-sm border border-dashed border-[color:var(--hp-hurt)]/60 p-2">
      <div class="flex items-start gap-2">
        <p class="min-w-0 flex-1 text-xs font-semibold text-muted-foreground">{props.id}</p>
        <button
          type="button"
          aria-label={`Remover ${props.id}`}
          onClick={() => props.onRemove()}
          class="shrink-0 text-muted-foreground transition-colors hover:text-penalty-ink"
        >
          <X aria-hidden="true" class="size-3.5" />
        </button>
      </div>
      <p class="text-2xs text-[color:var(--hp-hurt)]">
        Não está na lista da classe atual — ainda ocupa uma vaga.
      </p>
    </div>
  )
}

function TakenPowerCard(props: {
  option: PowerOption
  choiceIds: string[]
  canAddMore: boolean
  onRemove: () => void
  onChoice: (ids: string[]) => void
}) {
  return (
    <div class="space-y-1.5 rounded-sm border border-grimorio-gold/60 bg-accent/40 p-2">
      <div class="flex items-start gap-2">
        <p class="min-w-0 flex-1 text-xs font-semibold">{props.option.name}</p>
        <button
          type="button"
          aria-label={`Remover ${props.option.name}`}
          onClick={() => props.onRemove()}
          class="shrink-0 text-muted-foreground transition-colors hover:text-penalty-ink"
        >
          <X aria-hidden="true" class="size-3.5" />
        </button>
      </div>
      <Show when={props.option.choice}>
        {(choice) => (
          <PowerChoicePicker
            choice={choice()}
            value={props.choiceIds}
            canAddMore={props.canAddMore}
            onChange={props.onChoice}
          />
        )}
      </Show>
    </div>
  )
}

/**
 * The sub-choice a power demands when taken. A repeatable power (Aumento de
 * Atributo, Especialização) can hold several, and each extra pick eats another
 * slot — which is why adding one is gated on there being room.
 */
function PowerChoicePicker(props: {
  choice: NonNullable<PowerOption['choice']>
  value: string[]
  canAddMore: boolean
  onChange: (ids: string[]) => void
}) {
  const options = () => powerChoiceOptions(props.choice)
  const toggle = (id: string) => {
    if (!props.choice.repeatable) {
      props.onChange(props.value.includes(id) ? [] : [id])
      return
    }
    if (props.value.includes(id)) props.onChange(props.value.filter((x) => x !== id))
    else if (props.canAddMore) props.onChange([...props.value, id])
  }

  return (
    <div class="space-y-1 border-t border-dashed border-grimorio-iron pt-1.5">
      <p class="font-heading text-3xs uppercase tracking-[0.14em] text-muted-foreground">
        {props.choice.label}
        {props.choice.repeatable ? ` (${props.value.length})` : ''}
      </p>
      <div class="flex flex-wrap gap-1">
        <For each={options()}>
          {(option) => {
            const on = () => props.value.includes(option.id)
            const locked = () => props.choice.repeatable && !on() && !props.canAddMore
            return (
              <button
                type="button"
                aria-pressed={on()}
                disabled={locked()}
                onClick={() => toggle(option.id)}
                class={cn(
                  'rounded-sm border px-2 py-0.5 text-2xs transition-colors',
                  on()
                    ? 'border-grimorio-gold bg-accent'
                    : locked()
                      ? 'border-grimorio-iron opacity-40'
                      : 'border-grimorio-iron hover:bg-accent',
                )}
              >
                {option.name}
                <Show when={option.note}>
                  {(note) => <span class="ml-1 text-muted-foreground">· {note()}</span>}
                </Show>
              </button>
            )
          }}
        </For>
      </div>
      <Show when={props.value.length === 0}>
        <p class="text-3xs text-[color:var(--hp-hurt)]">
          Escolha {props.choice.label.toLowerCase()}.
        </p>
      </Show>
    </div>
  )
}
