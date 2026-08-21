import { anyClassChoicePending } from '@/entities/character/class-powers'
import { originGrant } from './grant-helpers'
import { type RaceChoiceState, anyRacePending, appliedRaceDeltas } from './grant-helpers'
import { periciaBudget, periciaPlan } from './pericia-helpers'
import { draftPowerPool, powerLedger } from './power-pool'
import { type CharacterFormValues, type StepSlug, stepReady, wizardSteps } from './wizard-steps'

/** How many benefits an origin grants (p56). */
const ORIGIN_BENEFIT_CAP = 2

export type Pendencia = {
  /** The step that can fix it — the Resumo links back there. */
  step: StepSlug
  label: string
}

/**
 * Everything the build still owes, in the order the wizard asks for it. None of
 * it blocks creation: the character can be forged with holes and finished on
 * the sheet — but it must be SAID here, because the Resumo is the last chance
 * to notice before the draft is gone.
 *
 * @example creationPendencias(draft.values, draft.raceChoices) // [{ step: 'pericias', label: 'Falta 1 perícia' }]
 */
export function creationPendencias(
  values: CharacterFormValues,
  raceChoices: RaceChoiceState,
): Pendencia[] {
  return [
    ...racePendencias(values, raceChoices),
    ...classPendencias(values),
    ...powerPendencias(values, raceChoices),
    ...originPendencias(values),
    ...periciaPendencias(values, raceChoices),
  ]
}

/** What a step is missing when its own gate is not satisfied — the wording the
 *  Resumo uses to explain a dead "Criar personagem" button. */
const BLOCKER_LABEL: Record<StepSlug, string> = {
  raca: 'Escolha uma raça',
  classe: 'Escolha uma classe',
  poderes: 'Escolha os poderes',
  origem: 'Escolha uma origem',
  atributos: 'Distribua os atributos',
  pericias: 'Treine as perícias',
  equipamento: 'Monte o equipamento',
  identidade: 'Dê um nome ao personagem (e confira PV/PM atuais)',
  resumo: 'Confira o resumo',
}

/**
 * The gates that actually BLOCK the forge, as opposed to the soft pendências
 * above. They are the same predicates the footer's button reads, so a disabled
 * "Criar personagem" always has a reason on screen — a dead button with no
 * explanation is the worst thing a last screen can do.
 *
 * @example creationBlockers(values, {}) // [{ step: 'raca', label: 'Escolha uma raça' }]
 */
export function creationBlockers(
  values: CharacterFormValues,
  raceChoices: RaceChoiceState,
): Pendencia[] {
  return wizardSteps(values)
    .filter((step) => !stepReady(step.slug, values, raceChoices))
    .map((step) => ({
      step: step.slug,
      label: BLOCKER_LABEL[step.slug],
    }))
}

function racePendencias(
  values: CharacterFormValues,
  raceChoices: RaceChoiceState,
): Pendencia[] {
  if (!anyRacePending(values.races, raceChoices)) return []
  return [
    {
      step: 'raca',
      label: 'Bônus de atributo de raça não colocado — o +1 não será aplicado',
    },
  ]
}

function classPendencias(values: CharacterFormValues): Pendencia[] {
  if (!anyClassChoicePending(values.classes, values.classChoices)) return []
  return [{ step: 'poderes', label: 'Falta escolher caminho ou devoto de uma classe' }]
}

function powerPendencias(
  values: CharacterFormValues,
  raceChoices: RaceChoiceState,
): Pendencia[] {
  const pool = draftPowerPool(values, raceChoices)
  const ledger = powerLedger(values.classes, values.classPowers, values.powerChoices, pool)
  const out: Pendencia[] = []
  if (ledger.remaining > 0) {
    out.push({
      step: 'poderes',
      label: plural(ledger.remaining, 'poder para escolher', 'poderes para escolher'),
    })
  }
  // A power taken but left without its sub-choice (totem, escola, arma) is a
  // half-power: it saves, and then the sheet cannot resolve what it does.
  const byId = new Map(pool.map((option) => [option.id, option]))
  for (const id of values.classPowers) {
    const option = byId.get(id)
    if (option?.choice && (values.powerChoices[id]?.length ?? 0) === 0) {
      out.push({ step: 'poderes', label: `${option.name} sem ${option.choice.label.toLowerCase()}` })
    }
  }
  return out
}

function originPendencias(values: CharacterFormValues): Pendencia[] {
  if (!values.origin) return []
  const missing = ORIGIN_BENEFIT_CAP - values.originChoices.length
  const out: Pendencia[] = []
  if (missing > 0) {
    out.push({
      step: 'origem',
      label: plural(missing, 'benefício de origem', 'benefícios de origem'),
    })
  }
  const grant = originGrant(values.origin)
  for (const benefitId of values.originChoices) {
    const benefit = grant?.benefits.find((b) => b.id === benefitId)
    if (benefit?.powerPick && (values.powerChoices[benefitId]?.length ?? 0) === 0) {
      out.push({ step: 'origem', label: `${benefit.name} sem o poder escolhido` })
    }
  }
  return out
}

function periciaPendencias(
  values: CharacterFormValues,
  raceChoices: RaceChoiceState,
): Pendencia[] {
  const primary = values.classes[0]?.className
  if (!primary) return []
  const intTotal =
    values.intelligence + (appliedRaceDeltas(values.races, raceChoices).intelligence ?? 0)
  const plan = periciaPlan(primary, intTotal, values.races)
  if (!plan) return []
  const budget = periciaBudget(plan, values.trainedExpertises)
  const missing = budget.classRemaining + budget.freeRemaining
  return missing > 0
    ? [{ step: 'pericias', label: plural(missing, 'perícia para treinar', 'perícias para treinar') }]
    : []
}

/** "Falta 1 perícia" / "Faltam 2 perícias" — the verb agrees, which is the
 *  kind of thing only a real screen catches. */
function plural(count: number, one: string, many: string): string {
  return count === 1 ? `Falta 1 ${one}` : `Faltam ${count} ${many}`
}
