import { For } from 'solid-js'
import { cn } from '@/shared/lib/utils'
import { romanNumeral } from '@/shared/lib/roman-numeral'
import { WIZARD_STEPS, type StepSlug, stepIndex } from './wizard-steps'

export type ForgeBeadsProps = {
  current: StepSlug
  /** Index of the furthest step the player may jump to; beyond it is locked. */
  reachable: number
  onJump: (slug: StepSlug) => void
}

/**
 * Progress of the Forja as a necklace of beads: one per step, filled behind the
 * player, hollow ahead. Compact enough to sit in the scene header at every form
 * factor, which is why the full-stage layout can afford to show progress at all.
 *
 * The beads are decoration WITH a handle — each is a real button that jumps to
 * an already-reached step. The position is announced in words, not numerals.
 *
 * @example <ForgeBeads current="origem" reachable={5} onJump={goToStep} />
 */
export function ForgeBeads(props: ForgeBeadsProps) {
  const index = () => stepIndex(props.current)
  const label = () => WIZARD_STEPS[index()].label

  return (
    <div class="flex min-w-0 items-center gap-3">
      {/* Beads are the decorative half of the progress: below `sm` the header
          has no room for them and the roman numeral + label carry the position
          on their own. */}
      <ol class="hidden items-center gap-1 sm:flex" aria-label="Passos da criação">
        <For each={WIZARD_STEPS}>
          {(step, i) => {
            const state = () =>
              i() === index() ? 'current' : i() < index() ? 'done' : 'ahead'
            const clickable = () => i() <= props.reachable && i() !== index()
            return (
              <li>
                <button
                  type="button"
                  disabled={!clickable()}
                  aria-current={state() === 'current' ? 'step' : undefined}
                  onClick={() => clickable() && props.onJump(step.slug)}
                  class={cn(
                    'flex size-5 items-center justify-center rounded-full text-[9px] leading-none transition-colors',
                    // Enabled beads keep a pointer affordance; locked ones fade
                    // but stay in place, so the necklace never changes length.
                    clickable() && 'hover:scale-110 hover:text-grimorio-gold',
                    state() === 'current' && 'text-grimorio-gold',
                    state() === 'done' && 'text-grimorio-gold/70',
                    state() === 'ahead' && 'text-muted-foreground/40',
                  )}
                >
                  <span aria-hidden="true">{state() === 'ahead' ? '◇' : '❖'}</span>
                  <span class="sr-only">{step.label}</span>
                </button>
              </li>
            )
          }}
        </For>
      </ol>

      <p class="flex min-w-0 items-baseline gap-2">
        <span
          aria-hidden="true"
          class="font-heading text-sm tracking-[0.2em] text-grimorio-gold"
        >
          {romanNumeral(index() + 1)}
        </span>
        <span class="truncate font-heading text-xs uppercase tracking-[0.16em] text-muted-foreground">
          {label()}
        </span>
        <span class="sr-only">
          Passo {index() + 1} de {WIZARD_STEPS.length} · {label()}
        </span>
      </p>
    </div>
  )
}
