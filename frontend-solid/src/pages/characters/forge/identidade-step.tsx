import { Show } from 'solid-js'
import { totalClassLevel } from '@/features/character-build/class-entries'
import { DevocaoPanel } from '@/features/character-build/devocao-panel'
import { deriveDraftVitals } from '@/features/character-build/draft-vitals'
import { useForge } from '@/features/character-build/forge-context'
import { cn } from '@/shared/lib/utils'
import { Input } from '@/shared/ui/input'
import { NumberInput } from '@/shared/ui/number-input'
import { Select } from '@/shared/ui/select'

/**
 * Last step before the Resumo: who this person is. The name gets the top of the
 * stage as a frontispiece, because it is the one thing still missing for the
 * character to exist — everything above it was a build.
 *
 * Vitalidade closes the step rather than owning one of its own: PV/PM are four
 * numbers and two of them are derived, which floated on a full stage (ALE-94).
 */
export function IdentidadeStep() {
  const { draft, options } = useForge()

  const lineage = () => {
    const classes = draft.values.classes.filter((entry) => entry.className)
    const classLine =
      classes.length === 0
        ? ''
        : `${classes.map((entry) => entry.className).join(' · ')} Nv ${totalClassLevel(classes)}`
    return [draft.values.races.join(' · '), classLine, draft.values.origin]
      .filter(Boolean)
      .join(' · ')
  }

  const vitals = () => deriveDraftVitals(draft.values, draft.raceChoices)

  const pickGod = (god: string) => {
    // A new god invalidates the granted power chosen under the previous one —
    // the per-class devoto slot re-syncs from the forge (see ForgeProvider).
    draft.patchValues({ god, godPower: '' })
  }

  return (
    <section
      class="flex min-h-0 flex-1 flex-col justify-center gap-4"
      aria-labelledby="forge-step-identidade"
    >
      <h2 id="forge-step-identidade" class="sr-only">
        Identidade
      </h2>

      <Frontispiece
        name={draft.values.name}
        lineage={lineage()}
        onName={(value) => draft.setValue('name', value)}
      />

      <div class="grid gap-4 lg:grid-cols-2">
        {/* Capped: three short controls stretched across half a 1440 stage read
            as a broken form, not a generous one. */}
        <div class="max-w-xs space-y-3">
          <Field label="Deus" name="identidade-god">
            <Select
              options={[{ value: '', label: 'Nenhum' }, ...toOptions(options.gods)]}
              value={
                draft.values.god
                  ? { value: draft.values.god, label: draft.values.god }
                  : { value: '', label: 'Nenhum' }
              }
              onChange={(option) => pickGod(option?.value ?? '')}
              placeholder="Nenhum"
              aria-label="Deus"
            />
          </Field>

          <Field label="Tamanho" name="identidade-size" hint="Padrão: Médio.">
            <Select
              options={toOptions(options.sizes)}
              value={{ value: draft.values.size, label: draft.values.size }}
              onChange={(option) => draft.setValue('size', option?.value ?? '')}
              placeholder="Selecionar tamanho"
              aria-label="Tamanho"
            />
          </Field>

          <Field
            label="Deslocamento (m)"
            name="identidade-displacement"
            hint="Padrão: 9 metros por ação de movimento."
          >
            <NumberInput
              id="identidade-displacement"
              min={0}
              max={120}
              value={draft.values.displacement}
              onChange={(value) => draft.setValue('displacement', value)}
              aria-label="Deslocamento em metros"
              spinnerLabel="deslocamento"
            />
          </Field>
        </div>

        <Show when={draft.values.god}>
          {(god) => (
            <DevocaoPanel
              godName={god()}
              value={draft.values.godPower ?? ''}
              onChange={(powerName) => draft.setValue('godPower', powerName)}
              raceNames={draft.values.races}
              classNames={draft.values.classes.map((entry) => entry.className)}
            />
          )}
        </Show>
      </div>

      <div class="grid gap-3 border-t border-grimorio-iron pt-4 sm:grid-cols-2">
        <VitalMeter
          kind="hp"
          label="Pontos de vida"
          abbr="PV"
          max={vitals().pvMax}
          current={draft.values.hpCurrent}
          onCurrent={(value) => draft.setValue('hpCurrent', value)}
        />
        <VitalMeter
          kind="mp"
          label="Pontos de mana"
          abbr="PM"
          max={vitals().pmMax}
          current={draft.values.mpCurrent}
          onCurrent={(value) => draft.setValue('mpCurrent', value)}
        />
      </div>

      <p class="text-[11px] text-muted-foreground">
        Os máximos derivam da classe, Constituição, nível e poderes — só o atual é editável,
        para quem começa a história já ferido.
      </p>
    </section>
  )
}

/** The name, given the top of the page like the title of a grimório entry. */
function Frontispiece(props: {
  name: string
  lineage: string
  onName: (value: string) => void
}) {
  return (
    <div class="flex flex-col items-center gap-1.5 pt-2">
      <label for="identidade-name" class="sr-only">
        Nome
      </label>
      <Input
        id="identidade-name"
        value={props.name}
        onInput={(event) => props.onName(event.currentTarget.value)}
        placeholder="Nome do personagem"
        required
        // `dark:bg-transparent` as well: the kit's Input paints `dark:bg-input/30`,
        // and a filled bar here reads as a form field instead of a title line.
        class="h-auto max-w-2xl rounded-none border-x-0 border-t-0 border-b border-grimorio-gold/50 bg-transparent px-2 py-1 text-center font-heading text-2xl tracking-[0.06em] text-foreground shadow-none focus-visible:border-grimorio-gold focus-visible:ring-0 sm:text-3xl dark:bg-transparent"
      />
      <Show when={props.lineage}>
        <p class="text-xs text-muted-foreground">{props.lineage}</p>
      </Show>
    </div>
  )
}

/** Label + control + hint, associated by `for`/`id` — biome rejects nesting. */
function Field(props: {
  label: string
  name: string
  hint?: string
  children: import('solid-js').JSX.Element
}) {
  return (
    <div class="space-y-1">
      <label
        for={props.name}
        class="font-heading text-[11px] uppercase tracking-[0.14em] text-muted-foreground"
      >
        {props.label}
      </label>
      {props.children}
      <Show when={props.hint}>
        {(hint) => <p class="text-[11px] text-muted-foreground/80">{hint()}</p>}
      </Show>
    </div>
  )
}

/**
 * One pool: the derived maximum in full size, the bar it fills, and the only
 * editable half — the current value.
 */
function VitalMeter(props: {
  kind: 'hp' | 'mp'
  label: string
  abbr: string
  max: number
  current: number
  onCurrent: (value: number) => void
}) {
  const percent = () =>
    props.max > 0 ? Math.max(0, Math.min(100, (props.current / props.max) * 100)) : 0
  const fill = () => (props.kind === 'hp' ? 'var(--hp-full)' : 'var(--mp-arcane)')

  return (
    <div class="space-y-2 rounded-md border border-grimorio-iron bg-muted/10 p-3">
      <div class="flex items-baseline justify-between gap-2">
        <p class="font-heading text-[11px] uppercase tracking-[0.16em] text-muted-foreground">
          {props.label}
        </p>
        <p class="font-heading text-3xl tabular-nums text-grimorio-gold">
          <span aria-hidden="true">{props.max}</span>
          <span class="sr-only">
            {props.abbr} máximo {props.max}
          </span>
        </p>
      </div>

      <div
        role="progressbar"
        aria-label={props.label}
        aria-valuenow={props.current}
        aria-valuemin={0}
        aria-valuemax={props.max}
        class="h-1.5 overflow-hidden rounded-full bg-grimorio-iron"
      >
        <div
          class={cn('h-full transition-[width]')}
          style={{ width: `${percent()}%`, background: fill() }}
        />
      </div>

      <div class="flex items-center gap-2">
        <label
          for={`identidade-${props.kind}-current`}
          class="text-[11px] uppercase tracking-wide text-muted-foreground"
        >
          {props.abbr} atual
        </label>
        <NumberInput
          id={`identidade-${props.kind}-current`}
          min={0}
          max={props.max}
          value={props.current}
          onChange={props.onCurrent}
          class="w-24"
          aria-label={`${props.abbr} atual`}
          spinnerLabel={`${props.abbr} atual`}
        />
      </div>
    </div>
  )
}

const toOptions = (values: string[]) => values.map((value) => ({ value, label: value }))
