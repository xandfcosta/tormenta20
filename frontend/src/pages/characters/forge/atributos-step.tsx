import { POINT_BUY_BUDGET, POINT_BUY_MAX, POINT_BUY_MIN } from '@/shared/rules/point-buy'
import { ChevronDown, ChevronUp } from 'lucide-solid'
import { For, Index, Show } from 'solid-js'
import {
  type AttributeRow,
  attributeRows,
} from '@/features/character-build/attribute-rows'
import { useForge } from '@/features/character-build/forge-context'
import { anyRacePending, signed } from '@/features/character-build/grant-helpers'
import { pointBuyStatusFor } from '@/features/character-build/point-buy'
import { cn } from '@/shared/lib/utils'
import type { AttributeMode } from '@/shared/stores/character-draft-store'

const FREE_MIN = -5
const FREE_MAX = 10

/**
 * Fifth step: the six attributes, as six pillars. They are the character's
 * iconic numbers, so they get the stage rather than a form grid — each column
 * carries the editable base, what the race adds on top, the resulting total,
 * and (in point-buy) what that value costs.
 */
export function AtributosStep() {
  const { draft } = useForge()

  const mode = () => draft.attributeMode()
  const rows = () => attributeRows(draft.values, draft.raceChoices, mode())
  const bounds = () =>
    mode() === 'point-buy'
      ? { min: POINT_BUY_MIN, max: POINT_BUY_MAX }
      : { min: FREE_MIN, max: FREE_MAX }

  const status = () =>
    pointBuyStatusFor({
      strength: draft.values.strength,
      dexterity: draft.values.dexterity,
      constitution: draft.values.constitution,
      intelligence: draft.values.intelligence,
      wisdom: draft.values.wisdom,
      charisma: draft.values.charisma,
    })

  const adjust = (row: AttributeRow, delta: number) => {
    const next = row.base + delta
    if (next < bounds().min || next > bounds().max) return
    draft.setValue(row.key, next)
  }

  return (
    <section class="flex min-h-0 flex-1 flex-col gap-3" aria-labelledby="forge-step-atributos">
      <div class="flex flex-wrap items-center gap-x-4 gap-y-2">
        <h2
          id="forge-step-atributos"
          class="font-heading text-lg uppercase tracking-[0.16em] text-grimorio-gold"
        >
          Distribua os atributos
        </h2>
        <ModeToggle mode={mode()} onMode={draft.setAttributeMode} />
      </div>

      <Show when={mode() === 'point-buy'}>
        <PointMeter spent={status().spent} warnings={status().warnings} />
      </Show>

      <Show when={anyRacePending(draft.values.races, draft.raceChoices)}>
        <p class="text-2xs text-[color:var(--hp-hurt)]">
          Há escolhas de atributo de raça pendentes — os +1 não colocados não serão aplicados.
        </p>
      </Show>

      <p class="text-xs text-muted-foreground">
        {mode() === 'free'
          ? 'Edite a base livremente; o total já inclui os bônus de raça.'
          : 'Todos começam em 0; distribua os pontos da Tabela 1-1. Bônus de raça não gastam pontos.'}
      </p>

      {/* `Index`, not `For`: `rows()` is DERIVED, so every keystroke produces
          six brand-new objects and a reference-keyed For would rebuild all six
          pillars — the field being typed into would lose focus each digit. */}
      <div class="grid grid-cols-3 content-center gap-2 sm:gap-3 lg:min-h-0 lg:flex-1 lg:grid-cols-6">
        <Index each={rows()}>
          {(row) => (
            <AttributePillar
              row={row()}
              min={bounds().min}
              max={bounds().max}
              showCost={mode() === 'point-buy'}
              onAdjust={(delta) => adjust(row(), delta)}
              onSet={(value) => draft.setValue(row().key, value)}
            />
          )}
        </Index>
      </div>
    </section>
  )
}

function AttributePillar(props: {
  row: AttributeRow
  min: number
  max: number
  showCost: boolean
  onAdjust: (delta: number) => void
  onSet: (value: number) => void
}) {
  const fieldId = () => `attr-${props.row.key}`
  const atMin = () => props.row.base <= props.min
  const atMax = () => props.row.base >= props.max

  return (
    <div class="flex flex-col items-center justify-center gap-1 rounded-sm border border-grimorio-iron bg-muted/10 p-2 lg:py-4">
      <label
        for={fieldId()}
        class="text-center font-heading text-3xs uppercase tracking-[0.14em] text-muted-foreground"
      >
        {props.row.label}
      </label>

      <Step
        direction="up"
        label={`Aumentar ${props.row.label}`}
        disabled={atMax()}
        onClick={() => props.onAdjust(1)}
      />

      {/* A real number field, not just the arrows: typing −1 must not cost six
          taps, and the keyboard has to reach the value like any other input. */}
      <input
        id={fieldId()}
        type="number"
        inputMode="numeric"
        min={props.min}
        max={props.max}
        value={props.row.base}
        onInput={(event) => props.onSet(Number(event.currentTarget.value) || 0)}
        class="w-full rounded-sm border border-grimorio-iron bg-transparent py-1 text-center font-heading text-4xl tabular-nums text-foreground outline-none focus-visible:border-grimorio-gold focus-visible:ring-2 focus-visible:ring-grimorio-gold/40"
      />

      <Step
        direction="down"
        label={`Diminuir ${props.row.label}`}
        disabled={atMin()}
        onClick={() => props.onAdjust(-1)}
      />

      {/* A dash, not "sem bônus de raça": the line exists to hold the column
          height steady, and six copies of a negative sentence is noise. */}
      <p class="text-center text-3xs leading-tight text-muted-foreground">
        <Show
          when={props.row.raceDelta !== 0}
          fallback={
            <>
              <span aria-hidden="true">—</span>
              <span class="sr-only">sem bônus de raça</span>
            </>
          }
        >
          <span
            class="tabular-nums"
            style={{
              color: props.row.raceDelta > 0 ? 'var(--hp-full)' : 'var(--hp-hurt)',
            }}
          >
            {signed(props.row.raceDelta)} raça
          </span>
        </Show>
      </p>

      <p class="font-mono text-sm text-grimorio-gold">
        <span aria-hidden="true">= {props.row.total}</span>
        <span class="sr-only">
          {props.row.label} total {props.row.total}
        </span>
      </p>

      <Show when={props.showCost}>
        <p class="text-3xs tabular-nums text-muted-foreground">
          {props.row.cost === null ? 'fora da tabela' : `${props.row.cost} pts`}
        </p>
      </Show>
    </div>
  )
}

function Step(props: {
  direction: 'up' | 'down'
  label: string
  disabled: boolean
  onClick: () => void
}) {
  return (
    <button
      type="button"
      aria-label={props.label}
      disabled={props.disabled}
      onClick={() => props.onClick()}
      class="flex h-5 w-full items-center justify-center rounded-none text-muted-foreground transition-colors hover:text-grimorio-gold disabled:opacity-30 disabled:hover:text-muted-foreground"
    >
      {props.direction === 'up' ? (
        <ChevronUp aria-hidden="true" class="size-4" />
      ) : (
        <ChevronDown aria-hidden="true" class="size-4" />
      )}
    </button>
  )
}

/** Free edit (default) vs. the book's point buy (p17). Draft-only UI state. */
function ModeToggle(props: { mode: AttributeMode; onMode: (mode: AttributeMode) => void }) {
  const options: [AttributeMode, string][] = [
    ['free', 'Livre'],
    ['point-buy', `Compra de pontos (${POINT_BUY_BUDGET} pts, p17)`],
  ]
  return (
    <div class="flex flex-wrap items-center gap-1.5">
      <For each={options}>
        {([value, label]) => (
          <button
            type="button"
            aria-pressed={props.mode === value}
            onClick={() => props.onMode(value)}
            class={cn(
              'rounded-sm border px-2.5 py-1 text-xs transition-colors',
              props.mode === value
                ? 'border-grimorio-gold bg-accent font-medium text-grimorio-gold'
                : 'border-grimorio-iron text-muted-foreground hover:bg-accent',
            )}
          >
            {label}
          </button>
        )}
      </For>
    </div>
  )
}

function PointMeter(props: { spent: number | null; warnings: string[] }) {
  const spent = () => props.spent ?? 0
  const over = () => spent() > POINT_BUY_BUDGET
  const filled = () => Math.min(100, Math.max(0, (spent() / POINT_BUY_BUDGET) * 100))

  return (
    <div class="space-y-1">
      <div
        role="progressbar"
        aria-valuenow={spent()}
        aria-valuemin={0}
        aria-valuemax={POINT_BUY_BUDGET}
        aria-label={`Pontos gastos: ${props.spent ?? 0} de ${POINT_BUY_BUDGET}`}
        class="flex items-center gap-3"
      >
        <div class="h-1.5 max-w-xs flex-1 overflow-hidden rounded-full bg-grimorio-iron">
          <div
            // `--hp-critical` (red), not `--hp-hurt`: hurt is amber at hue 70,
            // a hair from the gold at 85 — an over-budget bar painted with it
            // reads as "full", which is the opposite of the warning.
            class={cn(
              'h-full transition-[width]',
              over() ? 'bg-[var(--hp-critical)]' : 'bg-grimorio-gold',
            )}
            style={{ width: `${filled()}%` }}
          />
        </div>
        <p
          class={cn(
            'font-mono text-xs',
            over() ? 'text-[color:var(--hp-hurt)]' : 'text-muted-foreground',
          )}
        >
          {props.spent ?? '—'} de {POINT_BUY_BUDGET} pts
        </p>
      </div>
      <For each={props.warnings}>
        {(warning) => <p class="text-2xs text-[color:var(--hp-hurt)]">{warning}</p>}
      </For>
    </div>
  )
}
