import { Check, Search } from 'lucide-solid'
import { For, type JSX, Show, createSignal } from 'solid-js'
import { useForge } from '@/features/character-build/forge-context'
import { originGrant } from '@/features/character-build/grant-helpers'
import { GrantBox } from '@/features/character-build/grant-panels'
import { originSwitchPatch } from '@/features/character-build/origin-switch'
import { powerPickOptions } from '@/entities/character/class-powers'
import { toggleWithLimit } from '@/features/character-sheet/choice-lists'
import { matchesQuery } from '@/shared/lib/fuzzy-filter'
import { cn } from '@/shared/lib/utils'
import { Input } from '@/shared/ui/input'
import { Select } from '@/shared/ui/select'
import { FieldLabel, SectionLabel, SectionTitle } from '@/shared/ui/section-label'

/** Every origin offers a pool and the character takes two of it (p56). */
const ORIGIN_BENEFIT_CAP = 2

/**
 * Fourth step: origem — where the character comes from, and the two benefits
 * that past buys.
 *
 * A narrow name list rather than the tile grid Raça and Classe use: there are
 * 35 origins and the data carries no description for the origin itself, so a
 * grid would be 35 tiles of bare name. The stage goes to the detail instead,
 * because the real decision here is not WHICH origin but WHICH TWO benefits.
 */
export function OrigemStep() {
  const { draft, options } = useForge()
  const [query, setQuery] = createSignal('')

  const filtered = () => options.origins.filter((name) => matchesQuery([name], query()))
  const chosen = () => draft.values.origin

  // One decision, one patch: switching origin also drops the old benefits, the
  // item picks and the coin they had already rolled (see `originSwitchPatch`).
  const pick = (name: string) => draft.patchValues(originSwitchPatch(draft.values, name))

  const toggleBenefit = (benefitId: string) =>
    draft.setValue(
      'originChoices',
      toggleWithLimit(draft.values.originChoices, benefitId, ORIGIN_BENEFIT_CAP),
    )

  const setPowerPick = (benefitId: string, powerId: string) =>
    draft.setValue('powerChoices', {
      ...draft.values.powerChoices,
      [benefitId]: powerId ? [powerId] : [],
    })

  return (
    <section class="flex min-h-0 flex-1 flex-col gap-3" aria-labelledby="forge-step-origem">
      <SectionTitle
        id="forge-step-origem"
       
      >
        De onde você veio
      </SectionTitle>

      {/* The name column is CAPPED, not proportional: a fraction of a 1920
          stage gives 600px to a column of short words while the detail — the
          part being read — waits. Extra width goes to the benefits. */}
      <div class="grid gap-4 lg:min-h-0 lg:flex-1 lg:grid-cols-[minmax(11rem,18rem)_1fr]">
        <OriginList
          names={filtered()}
          query={query()}
          onQuery={setQuery}
          value={chosen()}
          onPick={pick}
        />

        {/* The detail is the stage here, so it fills the column instead of
            floating as a card with dead space under it. */}
        <section
          aria-label="Origem escolhida"
          class="flex flex-col lg:min-h-0 lg:overflow-y-auto lg:pr-1"
        >
          <Show
            when={chosen()}
            fallback={
              <p class="flex flex-1 items-center justify-center rounded-md border border-dashed border-grimorio-iron p-4 text-center text-xs text-muted-foreground">
                Escolha uma origem para ver os benefícios que ela oferece.
              </p>
            }
          >
            {(name) => (
              <BenefitPicker
                originName={name()}
                picks={draft.values.originChoices}
                powerChoices={draft.values.powerChoices}
                onToggle={toggleBenefit}
                onPowerPick={setPowerPick}
              />
            )}
          </Show>
        </section>
      </div>
    </section>
  )
}

function OriginList(props: {
  names: string[]
  query: string
  onQuery: (next: string) => void
  value: string
  onPick: (name: string) => void
}) {
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
          onInput={(e) => props.onQuery(e.currentTarget.value)}
          placeholder="Buscar origem"
          class="pl-8"
          aria-label="Buscar origem"
        />
      </div>

      {/* Capped below lg: 35 names stacked full-height buried the detail under a
          wall of origins on a landscape phone — the benefits were a 35-item
          scroll away. Beside the detail (lg+) the list simply fills its column. */}
      <div
        role="listbox"
        aria-label="Origens"
        class="max-h-[40vh] overflow-y-auto rounded-sm border border-grimorio-iron p-1 lg:max-h-none lg:min-h-0 lg:flex-1"
      >
        <For each={props.names}>
          {(name) => (
            <button
              type="button"
              role="option"
              aria-selected={props.value === name}
              onClick={() => props.onPick(name)}
              class={cn(
                'w-full truncate rounded-none px-2 py-1.5 text-left text-xs transition-colors',
                props.value === name
                  ? 'bg-accent font-medium text-grimorio-gold'
                  : 'text-muted-foreground hover:bg-accent hover:text-foreground',
              )}
            >
              {name}
            </button>
          )}
        </For>
        <Show when={props.names.length === 0}>
          <p class="p-3 text-center text-xs text-muted-foreground">
            Nenhuma origem para “{props.query}”.
          </p>
        </Show>
      </div>
    </div>
  )
}

function BenefitPicker(props: {
  originName: string
  picks: string[]
  powerChoices: Record<string, string[]>
  onToggle: (benefitId: string) => void
  onPowerPick: (benefitId: string, powerId: string) => void
}) {
  const grant = () => originGrant(props.originName)
  /** The poder único competes for the same two slots as the plain benefits. */
  const pool = () => {
    const g = grant()
    if (!g) return []
    return g.poderUnico ? [...g.benefits, g.poderUnico] : g.benefits
  }
  const full = () => props.picks.length >= ORIGIN_BENEFIT_CAP
  const remaining = () => ORIGIN_BENEFIT_CAP - props.picks.length

  return (
    <Show when={grant()}>
      {(g) => (
        <GrantBox title={g().name} class="flex flex-1 flex-col">
          <div class="flex flex-wrap items-baseline gap-x-2">
            <FieldLabel as="p" class="text-2xs font-semibold">
              Escolha {ORIGIN_BENEFIT_CAP} benefícios
            </FieldLabel>
            <p aria-hidden="true" class="flex gap-0.5 text-3xs text-grimorio-gold">
              <For each={[0, 1]}>{(i) => <span>{i < props.picks.length ? '●' : '○'}</span>}</For>
            </p>
            <p class="text-2xs text-muted-foreground">
              {props.picks.length} de {ORIGIN_BENEFIT_CAP}
            </p>
          </div>

          <div class="grid content-start gap-1.5 sm:grid-cols-2">
            <For each={pool()}>
              {(benefit) => (
                <BenefitCard
                  name={benefit.name}
                  description={benefit.description}
                  poderUnico={benefit.id === g().poderUnico?.id}
                  selected={props.picks.includes(benefit.id)}
                  locked={!props.picks.includes(benefit.id) && full()}
                  onToggle={() => props.onToggle(benefit.id)}
                >
                  <Show when={props.picks.includes(benefit.id) && benefit.powerPick}>
                    {(pick) => (
                      <FreePowerPicker
                        pool={pick()}
                        value={props.powerChoices[benefit.id]?.[0] ?? ''}
                        onPick={(powerId) => props.onPowerPick(benefit.id, powerId)}
                      />
                    )}
                  </Show>
                </BenefitCard>
              )}
            </For>
          </div>

          <Show when={remaining() > 0}>
            <p class="text-2xs text-[color:var(--hp-hurt)]">
              {remaining() === 1
                ? 'Falta 1 benefício'
                : `Faltam ${remaining()} benefícios`}{' '}
              — ou termine depois na ficha.
            </p>
          </Show>
        </GrantBox>
      )}
    </Show>
  )
}

function BenefitCard(props: {
  name: string
  description: string
  poderUnico: boolean
  selected: boolean
  locked: boolean
  onToggle: () => void
  children?: JSX.Element
}) {
  return (
    <div
      class={cn(
        'rounded-sm border p-2 transition-colors',
        props.selected
          ? 'border-grimorio-gold bg-accent'
          : props.locked
            ? 'border-grimorio-iron opacity-40'
            : 'border-grimorio-iron',
        props.poderUnico && 'sm:col-span-2',
      )}
    >
      <button
        type="button"
        aria-pressed={props.selected}
        disabled={props.locked}
        onClick={() => props.onToggle()}
        class="flex w-full items-start gap-2 text-left"
      >
        <span class="mt-0.5 flex size-3.5 shrink-0 items-center justify-center rounded-none border border-grimorio-iron">
          <Show when={props.selected}>
            <Check aria-hidden="true" class="size-2.5 text-grimorio-gold" />
          </Show>
        </span>
        <span class="min-w-0 flex-1">
          <span class="flex flex-wrap items-baseline gap-1.5">
            <span class="text-xs font-semibold">{props.name}</span>
            <Show when={props.poderUnico}>
              <SectionLabel as="span" tom="gold" class="text-4xs">
                ✦ poder único
              </SectionLabel>
            </Show>
          </span>
          <span class="block text-2xs leading-snug text-muted-foreground">
            {props.description}
          </span>
        </span>
      </button>
      {props.children}
    </div>
  )
}

/**
 * Concrete pick for a benefit that reads "um poder de combate/da Tormenta à sua
 * escolha". Prereqs are advisory — the benefit text says they apply and the GM
 * arbitrates, so the list is never filtered down to legal picks only.
 */
function FreePowerPicker(props: {
  pool: 'combate' | 'tormenta'
  value: string
  onPick: (powerId: string) => void
}) {
  const options = () =>
    powerPickOptions(props.pool).map((o) => ({ value: o.value, label: o.label }))
  const selected = () => options().find((o) => o.value === props.value) ?? null

  return (
    <div class="mt-1.5 space-y-1 border-t border-dashed border-grimorio-iron pt-1.5">
      {/* A Select, not the picker combobox: this field holds a value that must
          stay visible in the trigger (gotcha #19). */}
      <Select
        options={options()}
        value={selected()}
        onChange={(option) => props.onPick(option?.value ?? '')}
        placeholder="Escolher poder"
        size="sm"
        aria-label={`Poder ${props.pool === 'combate' ? 'de combate' : 'da Tormenta'} concedido`}
      />
      <Show when={!selected()}>
        <p class="text-2xs text-[color:var(--hp-hurt)]">
          Escolha o poder que este benefício concede.
        </p>
      </Show>
    </div>
  )
}
