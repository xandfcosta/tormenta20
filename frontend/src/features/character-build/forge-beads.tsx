import { For } from 'solid-js'
import { cn } from '@/shared/lib/utils'
import { romanNumeral } from '@/shared/lib/roman-numeral'
import { type StepSlug, type WizardStep, stepIndex } from './wizard-steps'
import { SectionLabel } from '@/shared/ui/section-label'

export type ForgeBeadsProps = {
  /** A caminhada DESTE personagem — pode ser mais curta que o catálogo. */
  steps: readonly WizardStep[]
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
 * @example <ForgeBeads steps={wizardSteps(v)} current="origem" reachable={5} onJump={goToStep} />
 */
export function ForgeBeads(props: ForgeBeadsProps) {
  const index = () => stepIndex(props.current, props.steps)
  const label = () => props.steps[index()]?.label ?? ''

  return (
    <div class="flex min-w-0 items-center gap-3">
      {/* Beads are the decorative half of the progress: below `sm` the header
          has no room for them and the roman numeral + label carry the position
          on their own. */}
      <ol class="hidden items-center gap-1 sm:flex" aria-label="Passos da criação">
        <For each={props.steps}>
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
                    'flex h-5 items-center justify-center gap-1 rounded-full px-1 text-4xs leading-none transition-colors',
                    // Enabled beads keep a pointer affordance; locked ones fade
                    // but stay in place, so the necklace never changes length.
                    clickable() && 'hover:scale-110 hover:text-grimorio-gold',
                    state() === 'current' && 'text-grimorio-gold',
                    state() === 'done' && 'text-grimorio-gold/70',
                    state() === 'ahead' && 'text-muted-foreground/40',
                  )}
                >
                  <span aria-hidden="true">{state() === 'ahead' ? '◇' : '❖'}</span>
                  {/* O nome do passo era só `sr-only`: quem enxerga via nove
                      losangos idênticos e não sabia o que cada um pedia. A
                      partir de `xl` ele aparece, e abaixo disso o numeral
                      romano ao lado continua nomeando o passo atual sozinho
                      (ALE-169). */}
                  <span aria-hidden="true" class="hidden xl:inline">
                    {step.label}
                  </span>
                  <span class="sr-only">{step.label}</span>
                </button>
              </li>
            )
          }}
        </For>
      </ol>

      <p class="flex min-w-0 items-baseline gap-2">
        {/* O numeral e o nome nomeiam o passo atual enquanto a trilha é só
            losango. A partir de `xl` ela nomeia TODOS, e repetir o atual ao
            lado dela vira ruído (ALE-169). O anúncio para leitor de tela
            continua, porque ele nunca dependeu de largura. */}
        <span
          aria-hidden="true"
          class="font-heading text-sm tracking-[0.2em] text-grimorio-gold xl:hidden"
        >
          {romanNumeral(index() + 1)}
        </span>
        <SectionLabel as="span" class="text-xs truncate xl:hidden">
          {label()}
        </SectionLabel>
        <span class="sr-only">
          Passo {index() + 1} de {props.steps.length} · {label()}
        </span>
      </p>
    </div>
  )
}
