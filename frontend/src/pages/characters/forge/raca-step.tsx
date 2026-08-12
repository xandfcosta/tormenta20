import { Show } from 'solid-js'
import { RacePicker } from '@/features/character-build/race-picker'
import { useForge } from '@/features/character-build/forge-context'

/**
 * First step of the Forja: the lineage. Everything downstream (attribute
 * deltas, free perícias, which powers are even reachable) hangs off it, which
 * is why it comes first and why the grants are shown before committing.
 */
export function RacaStep() {
  const { draft, options } = useForge()

  return (
    <section class="flex min-h-0 flex-1 flex-col gap-3" aria-labelledby="forge-step-raca">
      <div class="flex flex-wrap items-baseline gap-x-3 gap-y-1">
        <h2
          id="forge-step-raca"
          class="font-heading text-lg uppercase tracking-[0.16em] text-grimorio-gold"
        >
          Escolha a linhagem
        </h2>
        <Show
          when={draft.values.races.length > 1}
          fallback={
            <p class="text-xs text-muted-foreground">
              A primeira raça escolhida é a que vale mecanicamente.
            </p>
          }
        >
          <p class="text-xs text-muted-foreground">
            {draft.values.races.length} raças · só a primeira aplica sozinha; as outras
            dependem do mestre.
          </p>
        </Show>
      </div>

      <RacePicker
        options={options.races}
        value={draft.values.races}
        choices={draft.raceChoices}
        onChange={(next) => draft.setValue('races', next)}
        onChoice={(name, choice) => draft.setRaceChoice(name, choice)}
      />
    </section>
  )
}
