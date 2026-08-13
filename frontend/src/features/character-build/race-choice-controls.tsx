import { ATTRIBUTE_ABBR, ATTRIBUTE_KEYS, type AttributeKey } from '@/shared/api/attribute-keys'
import { For, Match, Show, Switch } from 'solid-js'
import { cn } from '@/shared/lib/utils'
import {
  type RaceChoice,
  type RaceChoiceMeta,
  raceChoiceMeta,
  resolveRaceDeltas,
} from './grant-helpers'
import { DeltaBadges } from './grant-panels'

export type RaceChoiceControlsProps = {
  raceName: string
  choice: RaceChoice
  onChange: (next: RaceChoice) => void
}

/**
 * Attribute-choice capture for a chosen race. Floating races (Humano, Lefou,
 * Osteon, Sereia) get +1 pills with the excluded attribute locked out and the
 * guaranteed penalty spelled out; subrace races (Suraggel) get ascendência
 * cards showing what each one is worth. Fixed races render nothing.
 */
export function RaceChoiceControls(props: RaceChoiceControlsProps) {
  const meta = () => raceChoiceMeta(props.raceName)
  // Narrowing accessors instead of casts inside the JSX: `Match` can't refine a
  // discriminated union on its own, and a cast here would outlive the day
  // someone adds a third kind of racial choice.
  const floating = () => {
    const m = meta()
    return m.kind === 'floating' ? m : null
  }
  const subrace = () => {
    const m = meta()
    return m.kind === 'subrace' ? m : null
  }

  return (
    <Switch>
      {/* keyed: swapping race must rebuild the control, not re-point it — a
          reused pill grid would keep the previous race's placed picks. */}
      <Match when={floating()} keyed>
        {(m) => <FloatingPicker meta={m} choice={props.choice} onChange={props.onChange} />}
      </Match>
      <Match when={subrace()} keyed>
        {(m) => (
          <SubracePicker
            raceName={props.raceName}
            options={m.options}
            choice={props.choice}
            onChange={props.onChange}
          />
        )}
      </Match>
    </Switch>
  )
}

function FloatingPicker(props: {
  meta: Extract<RaceChoiceMeta, { kind: 'floating' }>
  choice: RaceChoice
  onChange: (next: RaceChoice) => void
}) {
  const picks = () => props.choice.floatingPicks ?? []
  // The excluded attribute never counts toward the quota, even if a stale draft
  // carries it — otherwise the counter would claim a pick that grants nothing.
  const placed = () => picks().filter((a) => a !== props.meta.exclude).length

  const toggle = (attr: AttributeKey) => {
    if (attr === props.meta.exclude) return
    if (picks().includes(attr)) {
      props.onChange({ ...props.choice, floatingPicks: picks().filter((a) => a !== attr) })
      return
    }
    if (placed() < props.meta.count) {
      props.onChange({ ...props.choice, floatingPicks: [...picks(), attr] })
    }
  }

  return (
    <div class="space-y-1.5">
      <p class="text-[11px] font-semibold uppercase tracking-wide text-muted-foreground">
        Distribua +{props.meta.value} em {props.meta.count} atributos · {placed()}/
        {props.meta.count}
      </p>
      <div class="flex flex-wrap gap-1.5">
        <For each={ATTRIBUTE_KEYS}>
          {(attr) => {
            const excluded = () => attr === props.meta.exclude
            const selected = () => picks().includes(attr)
            const full = () => placed() >= props.meta.count && !selected()
            return (
              <button
                type="button"
                aria-pressed={selected()}
                disabled={excluded() || full()}
                onClick={() => toggle(attr)}
                title={excluded() ? `Não pode aumentar ${ATTRIBUTE_ABBR[attr]}` : undefined}
                class={cn(
                  'rounded-md border px-2 py-1 font-mono text-xs transition-colors',
                  selected()
                    ? 'border-grimorio-gold bg-accent text-grimorio-gold'
                    : 'border-grimorio-iron',
                  (excluded() || full()) && 'opacity-40',
                )}
              >
                {ATTRIBUTE_ABBR[attr]}
              </button>
            )
          }}
        </For>
      </div>
      <Show when={props.meta.penalty}>
        {(penalty) => (
          <p class="text-[11px] text-muted-foreground">
            Penalidade fixa:{' '}
            <span class="font-mono">
              −{Math.abs(penalty().value)} {ATTRIBUTE_ABBR[penalty().attribute]}
            </span>
          </p>
        )}
      </Show>
    </div>
  )
}

function SubracePicker(props: {
  raceName: string
  options: string[]
  choice: RaceChoice
  onChange: (next: RaceChoice) => void
}) {
  return (
    <div class="space-y-1.5">
      <p class="text-[11px] font-semibold uppercase tracking-wide text-muted-foreground">
        Escolha a ascendência
      </p>
      <div class="grid gap-1.5 sm:grid-cols-2">
        <For each={props.options}>
          {(option) => (
            <button
              type="button"
              aria-pressed={props.choice.ascendencia === option}
              onClick={() => props.onChange({ ...props.choice, ascendencia: option })}
              class={cn(
                'space-y-1 rounded-md border p-2 text-left transition-colors',
                props.choice.ascendencia === option
                  ? 'border-grimorio-gold bg-accent'
                  : 'border-grimorio-iron hover:bg-accent',
              )}
            >
              <p class="text-xs font-semibold capitalize">{option}</p>
              <DeltaBadges deltas={resolveRaceDeltas(props.raceName, { ascendencia: option })} />
            </button>
          )}
        </For>
      </div>
    </div>
  )
}
