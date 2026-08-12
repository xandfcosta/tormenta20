import type { Component } from 'solid-js'
import { Dynamic } from 'solid-js/web'
import { createCurrentStep } from '@/features/character-build/current-step'
import { WIZARD_STEPS, type StepSlug, stepIndex } from '@/features/character-build/wizard-steps'
import { RacaStep } from './raca-step'

/**
 * Registry of the Forja's steps. It holds the COMPONENT, never a
 * `render(draft)` call: a function invoked with a value captures that value,
 * and the step would never see the next edit (gotcha #14 of the port).
 *
 * Partial while the scene lands slice by slice (ALE-94) — an unregistered step
 * says so on stage instead of rendering nothing.
 */
const STEP_COMPONENTS: Partial<Record<StepSlug, Component>> = {
  raca: RacaStep,
}

export function ForgeStep() {
  const current = createCurrentStep()
  const component = () => STEP_COMPONENTS[current()]

  return (
    <Dynamic component={component() ?? (() => <StepPending slug={current()} />)} />
  )
}

/** Scaffolding: deleted as each remaining step of ALE-94 lands. */
function StepPending(props: { slug: StepSlug }) {
  const label = () => WIZARD_STEPS[stepIndex(props.slug)].label
  return (
    <div class="flex flex-1 flex-col items-center justify-center gap-2 text-center">
      <p class="font-heading text-lg uppercase tracking-[0.16em] text-muted-foreground">
        {label()}
      </p>
      <p class="text-sm text-muted-foreground">Este passo ainda está sendo forjado (ALE-94).</p>
    </div>
  )
}
