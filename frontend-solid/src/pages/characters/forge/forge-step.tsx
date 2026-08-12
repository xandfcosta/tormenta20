import type { Component } from 'solid-js'
import { Dynamic } from 'solid-js/web'
import { createCurrentStep } from '@/features/character-build/current-step'
import type { StepSlug } from '@/features/character-build/wizard-steps'
import { AtributosStep } from './atributos-step'
import { ClasseStep } from './classe-step'
import { EquipamentoStep } from './equipamento-step'
import { IdentidadeStep } from './identidade-step'
import { OrigemStep } from './origem-step'
import { PericiasStep } from './pericias-step'
import { PoderesStep } from './poderes-step'
import { RacaStep } from './raca-step'
import { ResumoStep } from './resumo-step'

/**
 * Registry of the Forja's steps. It holds the COMPONENT, never a
 * `render(draft)` call: a function invoked with a value captures that value,
 * and the step would never see the next edit (gotcha #14 of the port).
 *
 * Total, not partial: the type makes a new step in `WIZARD_STEPS` a compile
 * error until it has a screen, which is cheaper than an empty stage.
 */
const STEP_COMPONENTS: Record<StepSlug, Component> = {
  raca: RacaStep,
  classe: ClasseStep,
  poderes: PoderesStep,
  origem: OrigemStep,
  atributos: AtributosStep,
  pericias: PericiasStep,
  equipamento: EquipamentoStep,
  identidade: IdentidadeStep,
  resumo: ResumoStep,
}

export function ForgeStep() {
  const current = createCurrentStep()

  return <Dynamic component={STEP_COMPONENTS[current()]} />
}
