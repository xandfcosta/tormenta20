import { Show } from 'solid-js'
import { Button } from '@/shared/ui/button'
import { cn } from '@/shared/lib/utils'
import type { CharacterDraftStore } from '@/shared/stores/character-draft-store'
import { totalClassLevel } from './class-entries'
import { deriveDraftDefense } from './draft-defense'
import { deriveDraftVitals } from './draft-vitals'
import { type StepSlug, allStepsReady, stepIndex, stepReady, wizardSteps } from './wizard-steps'
import { FieldLabel, SectionTitle } from '@/shared/ui/section-label'

export type ForgeFooterProps = {
  draft: CharacterDraftStore
  current: StepSlug
  submitting: boolean
  /** Why the last forge attempt failed, shown in the band itself — a toast
   *  would be gone before the player looked away from the button. */
  error?: string | null
  /** Walk the wizard: −1 back, +1 forward. */
  onStep: (delta: -1 | 1) => void
  onCreate: () => void
}

/**
 * The Forja's bottom band: who is being forged (left), what the choices are
 * worth so far (middle), and the way onward (right).
 *
 * It is where the live preview lives in the full-stage layout — one band that
 * serves desktop and phone alike, so the "escolhi → meu PV subiu" loop survives
 * at every width instead of being a desktop-only rail.
 */
export function ForgeFooter(props: ForgeFooterProps) {
  const values = () => props.draft.values
  const raceChoices = () => props.draft.raceChoices

  const name = () => values().name.trim() || 'Novo personagem'
  /**
   * Multiclasse shows every class and the TOTAL level: PV/PM already count all
   * of them, and a line reading "Guerreiro Nv 1" beside PV 27 would be the
   * preview contradicting itself.
   */
  const lineage = () => {
    const classes = values().classes.filter((entry) => entry.className)
    const level = totalClassLevel(classes)
    const classLine =
      classes.length === 0
        ? ''
        : `${classes.map((entry) => entry.className).join(' · ')} Nv ${level}`
    return [values().races.join(' · '), classLine].filter(Boolean).join(' · ')
  }

  const defense = () => deriveDraftDefense(values(), raceChoices())
  const vitals = () => deriveDraftVitals(values(), raceChoices())

  const isLast = () => props.current === 'resumo'
  const hasPrevious = () => stepIndex(props.current, wizardSteps(values())) > 0
  const canAdvance = () => stepReady(props.current, values(), raceChoices())
  const canCreate = () => allStepsReady(values(), raceChoices())

  return (
    <div class="shrink-0 border-t border-grimorio-iron bg-card/80 backdrop-blur">
      <Show when={props.error}>
        {(message) => (
          <p
            role="alert"
            class="border-b border-grimorio-iron px-3 py-1.5 text-xs text-[color:var(--hp-hurt)] sm:px-5"
          >
            {message()}
          </p>
        )}
      </Show>
      <div class="flex flex-wrap items-center justify-between gap-x-4 gap-y-2 px-3 py-2 sm:px-5">
      <div class="flex min-w-0 flex-1 items-baseline gap-2">
        <SectionTitle as="p" tom="inherit" class="text-sm truncate text-foreground">
          {name()}
        </SectionTitle>
        <p class="hidden min-w-0 truncate text-xs text-muted-foreground sm:block">
          {lineage()}
        </p>
      </div>

      <div class="flex shrink-0 items-center gap-3 font-mono text-sm">
        <Stat label="Defesa" abbr="DEF" value={defense()} />
        <Stat label="Pontos de vida" abbr="PV" value={vitals().pvMax} />
        <Stat label="Pontos de mana" abbr="PM" value={vitals().pmMax} dim={vitals().pmMax === 0} />
      </div>

      <div class="flex shrink-0 items-center gap-2">
        <Show when={hasPrevious()}>
          <Button type="button" variant="outline" size="sm" onClick={() => props.onStep(-1)}>
            ‹ Voltar
          </Button>
        </Show>
        <Show
          when={isLast()}
          fallback={
            <Button
              type="button"
              size="sm"
              disabled={!canAdvance()}
              onClick={() => props.onStep(1)}
            >
              Próximo ›
            </Button>
          }
        >
          <Button
            type="button"
            size="sm"
            disabled={props.submitting || !canCreate()}
            onClick={() => props.onCreate()}
          >
            {props.submitting ? 'Forjando…' : 'Criar personagem'}
          </Button>
        </Show>
        </div>
      </div>
    </div>
  )
}

/**
 * One derived number. The abbreviation is decoration and the digits carry no
 * name of their own, so the whole thing is spelled out once in an `sr-only`
 * line: `aria-label` on a `<span>` is silently ignored (gotcha #20) and would
 * leave a screen reader announcing a bare "12".
 */
function Stat(props: { label: string; abbr: string; value: number; dim?: boolean }) {
  return (
    <span class="flex items-baseline gap-1">
      <FieldLabel
        aria-hidden="true">
        {props.abbr}
      </FieldLabel>
      <span
        aria-hidden="true"
        class={cn('tabular-nums', props.dim ? 'text-muted-foreground' : 'text-grimorio-gold')}
      >
        {props.value}
      </span>
      <span class="sr-only">
        {props.label} {props.value}
      </span>
    </span>
  )
}
