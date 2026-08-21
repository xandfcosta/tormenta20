import { Match, Switch } from 'solid-js'
import {
  type RaceChoice,
  type RaceChoiceMeta,
  raceChoiceMeta,
  resolveRaceDeltas,
} from './grant-helpers'
import { DeltaBadges } from './grant-panels'
import { RaceFloatingPicker, RaceSubracePicker } from '@/shared/ui/race-attribute-picker'

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
 *
 * As pastilhas em si moram em `shared/ui` desde a ALE-169: a FICHA precisa do
 * mesmo controle, e ela é uma feature irmã — a FSD não deixa ela importar
 * daqui. O que fica neste arquivo é o que é da forja: ler a meta da raça e
 * mostrar a prévia dos deltas na ascendência.
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
  return (
    <RaceFloatingPicker
      count={props.meta.count}
      value={props.meta.value}
      exclude={props.meta.exclude}
      penalty={props.meta.penalty}
      picks={props.choice.floatingPicks ?? []}
      onChange={(floatingPicks) => props.onChange({ ...props.choice, floatingPicks })}
    />
  )
}

function SubracePicker(props: {
  raceName: string
  options: string[]
  choice: RaceChoice
  onChange: (next: RaceChoice) => void
}) {
  return (
    <RaceSubracePicker
      options={props.options}
      value={props.choice.ascendencia}
      onChange={(ascendencia) => props.onChange({ ...props.choice, ascendencia })}
      preview={(option) => (
        <DeltaBadges deltas={resolveRaceDeltas(props.raceName, { ascendencia: option })} />
      )}
    />
  )
}
