import { Check } from 'lucide-solid'
import { For, Show, createMemo } from 'solid-js'
import { useForge } from '@/features/character-build/forge-context'
import { appliedRaceDeltas } from '@/features/character-build/grant-helpers'
import {
  missingNotice,
  overflowNotice,
  seedFixedExpertises,
} from '@/features/character-build/pericia-bands'
import {
  type PericiaPlan,
  periciaBudget,
  periciaPlan,
} from '@/features/character-build/pericia-helpers'
import { cn } from '@/shared/lib/utils'

/**
 * Sixth step: perícias. Two budgets with disjoint pools — the class list and
 * the free picks bought by Inteligência and race — so they sit side by side
 * rather than stacked: what the player has to compare is how much of EACH is
 * left, and a stacked layout shows one at a time.
 */
export function PericiasStep() {
  const { draft } = useForge()

  const primary = () => draft.values.classes[0]?.className ?? ''
  const intTotal = () =>
    draft.values.intelligence +
    (appliedRaceDeltas(draft.values.races, draft.raceChoices).intelligence ?? 0)

  const plan = createMemo(() =>
    primary() ? periciaPlan(primary(), intTotal(), draft.values.races) : null,
  )
  const trained = () => draft.values.trainedExpertises

  // The class's fixed perícias are not a choice — seed them as soon as the plan
  // is known. Identity from `seedFixedExpertises` keeps this from writing (and
  // re-triggering) on every pass.
  const seeded = createMemo(() => {
    const current = plan()
    if (!current) return trained()
    const next = seedFixedExpertises(trained(), current)
    if (next !== trained()) draft.setValue('trainedExpertises', next)
    return next
  })

  const set = (next: string[]) => draft.setValue('trainedExpertises', next)
  const toggle = (name: string) =>
    set(
      seeded().includes(name)
        ? seeded().filter((entry) => entry !== name)
        : [...seeded(), name],
    )

  return (
    <section class="flex min-h-0 flex-1 flex-col gap-3" aria-labelledby="forge-step-pericias">
      <Show
        when={plan()}
        fallback={
          <div class="flex flex-1 items-center justify-center">
            <p class="text-sm text-muted-foreground">
              Escolha uma classe primeiro — é ela que abre a lista de perícias.
            </p>
          </div>
        }
      >
        {(current) => (
          <PericiaBands
            plan={current()}
            className={primary()}
            intTotal={intTotal()}
            trained={seeded()}
            onToggle={toggle}
            onSet={set}
          />
        )}
      </Show>
    </section>
  )
}

function PericiaBands(props: {
  plan: PericiaPlan
  className: string
  intTotal: number
  trained: string[]
  onToggle: (name: string) => void
  onSet: (next: string[]) => void
}) {
  const budget = () => periciaBudget(props.plan, props.trained)
  const missing = () => budget().classRemaining + budget().freeRemaining

  const pickEitherOr = (chosen: string, other: string) =>
    props.onSet([...props.trained.filter((n) => n !== other && n !== chosen), chosen])

  return (
    <>
      <div class="flex flex-wrap items-baseline gap-x-3 gap-y-1">
        <h2
          id="forge-step-pericias"
          class="font-heading text-lg uppercase tracking-[0.16em] text-grimorio-gold"
        >
          Treine as perícias
        </h2>
        <p class="text-xs text-muted-foreground">
          {props.className} · INT {props.intTotal >= 0 ? `+${props.intTotal}` : props.intTotal}
        </p>
      </div>

      <div class="flex flex-wrap items-center gap-x-4 gap-y-2">
        <Show when={props.plan.fixed.length > 0}>
          <p class="flex flex-wrap items-center gap-1.5 text-[11px] text-muted-foreground">
            <span class="uppercase tracking-wide">Fixas:</span>
            <For each={props.plan.fixed}>
              {(name) => (
                <span class="rounded-md border border-grimorio-gold/60 bg-accent px-2 py-0.5 text-xs text-foreground">
                  {name}
                </span>
              )}
            </For>
          </p>
        </Show>

        <Show when={props.plan.eitherOr}>
          {(pair) => (
            <div class="flex flex-wrap items-center gap-1.5">
              <span class="text-[11px] uppercase tracking-wide text-muted-foreground">
                Escolha uma:
              </span>
              <For each={pair()}>
                {(name, i) => (
                  <button
                    type="button"
                    aria-pressed={props.trained.includes(name)}
                    onClick={() => pickEitherOr(name, pair()[i() === 0 ? 1 : 0])}
                    class={cn(
                      'rounded-md border px-3 py-1 text-xs transition-colors',
                      props.trained.includes(name)
                        ? 'border-grimorio-gold bg-accent text-grimorio-gold'
                        : 'border-grimorio-iron hover:bg-accent',
                    )}
                  >
                    {name}
                  </button>
                )}
              </For>
            </div>
          )}
        </Show>
      </div>

      <Show when={overflowNotice(props.plan, props.trained)}>
        {(notice) => (
          <p class="rounded-md border border-grimorio-iron bg-muted/20 px-3 py-1.5 text-[11px] text-muted-foreground">
            {notice()} A cota da classe acabou, então o excedente sai do orçamento livre.
          </p>
        )}
      </Show>

      {/* Two columns only when there IS a second budget — a lone class band
          would otherwise sit in half the stage with the other half empty. */}
      <div
        class={cn(
          'grid gap-4 lg:min-h-0 lg:flex-1',
          props.plan.freeCount > 0 && 'lg:grid-cols-2',
        )}
      >
        <Band
          label="Da classe"
          spent={budget().classSpent}
          total={props.plan.classCount}
          pool={props.plan.classPool}
          trained={props.trained}
          // Class picks never hard-lock while free budget remains: the excess
          // spills into it, which is what the overflow notice explains.
          locked={budget().classRemaining === 0 && budget().freeRemaining === 0}
          onToggle={props.onToggle}
        />
        <Show when={props.plan.freeCount > 0}>
          <Band
            label="Livre"
            accent
            subtitle={freeBreakdown(props.plan)}
            spent={budget().freeSpent}
            total={props.plan.freeCount}
            pool={props.plan.freePool}
            trained={props.trained}
            locked={budget().freeRemaining === 0}
            onToggle={props.onToggle}
          />
        </Show>
      </div>

      <Show when={missingNotice(missing())}>
        {(notice) => <p class="text-[11px] text-[color:var(--hp-hurt)]">{notice()}</p>}
      </Show>
    </>
  )
}

/** One capped band: a bead counter over its own pool. */
function Band(props: {
  label: string
  subtitle?: string
  accent?: boolean
  spent: number
  total: number
  pool: string[]
  trained: string[]
  locked: boolean
  onToggle: (name: string) => void
}) {
  const beads = () => Array.from({ length: props.total }, (_, i) => i < props.spent)

  return (
    <section
      aria-label={`Perícias · ${props.label}`}
      class={cn('flex min-h-0 flex-col gap-1.5', props.accent && 'lg:border-l lg:border-grimorio-iron lg:pl-4')}
    >
      <div class="flex flex-wrap items-baseline gap-x-2">
        <p class="font-heading text-[11px] uppercase tracking-[0.16em] text-muted-foreground">
          {props.accent ? '✦ ' : ''}
          {props.label}
        </p>
        <p aria-hidden="true" class="flex gap-0.5 text-[10px] text-grimorio-gold">
          <For each={beads()}>{(filled) => <span>{filled ? '●' : '○'}</span>}</For>
        </p>
        <p class="text-[11px] text-muted-foreground">
          {props.spent} de {props.total}
        </p>
      </div>
      <Show when={props.subtitle}>
        {(subtitle) => <p class="text-[11px] text-muted-foreground/80">{subtitle()}</p>}
      </Show>

      {/* auto-fill so the pool reflows whether the band owns half the stage or
          all of it — the column count is not a property of the viewport here. */}
      <div class="grid grid-cols-[repeat(auto-fill,minmax(9.5rem,1fr))] gap-1.5 p-0.5 lg:min-h-0 lg:overflow-y-auto">
        <For each={props.pool}>
          {(name) => {
            const selected = () => props.trained.includes(name)
            const itemLocked = () => !selected() && props.locked
            return (
              <button
                type="button"
                aria-pressed={selected()}
                disabled={itemLocked()}
                onClick={() => props.onToggle(name)}
                class={cn(
                  'flex items-center gap-1.5 rounded-md border px-2 py-1 text-left text-xs transition-colors',
                  selected()
                    ? 'border-grimorio-gold bg-accent'
                    : itemLocked()
                      ? 'border-grimorio-iron opacity-40'
                      : 'border-grimorio-iron hover:bg-accent',
                )}
              >
                <span class="flex size-3.5 shrink-0 items-center justify-center rounded-sm border border-grimorio-iron">
                  <Show when={selected()}>
                    <Check aria-hidden="true" class="size-2.5 text-grimorio-gold" />
                  </Show>
                </span>
                <span class="truncate">{name}</span>
              </button>
            )
          }}
        </For>
      </div>
    </section>
  )
}

/** Where the free slots come from, spelled out under the band's title. */
function freeBreakdown(plan: PericiaPlan): string {
  const parts: string[] = []
  if (plan.intCount > 0) parts.push(`${plan.intCount} de Inteligência`)
  if (plan.raceCount > 0) parts.push(`${plan.raceCount} de raça`)
  return `${parts.join(' · ')} · qualquer perícia`
}
