import { Show } from 'solid-js'
import { Button } from '@/shared/ui/button'
import { cn } from '@/shared/lib/utils'
import type { CharacterDraftStore } from '@/shared/stores/character-draft-store'
import { totalClassLevel } from './class-entries'
import { deriveDraftDefense } from './draft-defense'
import { deriveDraftVitals } from './draft-vitals'
import { type StepSlug, allStepsReady, stepIndex, stepReady } from './wizard-steps'

export type ForgeFooterProps = {
  draft: CharacterDraftStore
  current: StepSlug
  submitting: boolean
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
  const hasPrevious = () => stepIndex(props.current) > 0
  const canAdvance = () => stepReady(props.current, values(), raceChoices())
  const canCreate = () => allStepsReady(values(), raceChoices())

  return (
    <div class="flex shrink-0 flex-wrap items-center justify-between gap-x-4 gap-y-2 border-t border-grimorio-iron bg-card/80 px-3 py-2 backdrop-blur sm:px-5">
      <div class="flex min-w-0 flex-1 items-baseline gap-2">
        <p class="truncate font-heading text-sm uppercase tracking-[0.12em] text-foreground">
          {name()}
        </p>
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
      <span
        aria-hidden="true"
        class="text-[10px] uppercase tracking-widest text-muted-foreground"
      >
        {props.abbr}
      </span>
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
