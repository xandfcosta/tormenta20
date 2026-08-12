import { z } from 'zod'
import type { RaceChoiceState } from './grant-helpers'

/**
 * Single source of truth for the stepped creation flow: the ordered steps and
 * the per-step readiness predicates that drive the stepper lock states, the
 * footer "Próximo" gate, and the index-route redirect. Order follows the T20
 * build order with the biggest identity picks first (raça, classe) so the
 * running aggregate visibly grows, and derived numbers (atributos/vitalidade)
 * after their inputs.
 */
export const WIZARD_STEPS = [
  { slug: 'raca', label: 'Raça' },
  { slug: 'classe', label: 'Classe' },
  { slug: 'poderes', label: 'Poderes' },
  { slug: 'origem', label: 'Origem' },
  { slug: 'atributos', label: 'Atributos' },
  { slug: 'pericias', label: 'Perícias' },
  { slug: 'equipamento', label: 'Equipamento' },
  { slug: 'vitalidade', label: 'Vitalidade' },
  { slug: 'identidade', label: 'Identidade' },
  { slug: 'resumo', label: 'Resumo' },
] as const

export type StepSlug = (typeof WIZARD_STEPS)[number]['slug']

export function stepIndex(slug: StepSlug): number {
  return WIZARD_STEPS.findIndex((s) => s.slug === slug)
}

/**
 * The step one walk away, or null at either end of the flow. Keeps "next" and
 * "previous" defined by the order above and nowhere else.
 *
 * @example stepAt('classe', -1) // 'raca'
 */
export function stepAt(current: StepSlug, delta: -1 | 1): StepSlug | null {
  return WIZARD_STEPS[stepIndex(current) + delta]?.slug ?? null
}

/** Whether a string names a step — for validating a slug arriving from the URL. */
export function isStepSlug(value: string): value is StepSlug {
  return WIZARD_STEPS.some((s) => s.slug === value)
}

const classEntrySchema = z.object({
  className: z.string().min(1, 'Choose a class'),
  level: z.number().int().min(1).max(20),
})

export const characterSchema = z
  .object({
    name: z.string().min(1, 'Name is required'),
    races: z.array(z.string()).min(1, 'Select at least one race'),
    origin: z.string().min(1, 'Origin is required'),
    classes: z.array(classEntrySchema).min(1, 'Add at least one class'),
    god: z.string().optional(),
    godPower: z.string().optional(),
    tibar: z.number().int().min(0).optional(),
    startingWeaponSimple: z.string().optional(),
    startingWeaponMartial: z.string().optional(),
    startingArmor: z.string().optional(),
    startingShield: z.boolean().optional(),
    startingPurchases: z.record(z.string(), z.number().int().min(0)).optional(),
    originItemPicks: z.record(z.string(), z.string()).optional(),
    startingMoneyRolled: z.boolean().optional(),
    hpMax: z.number().int().min(1),
    hpCurrent: z.number().int().min(0),
    mpMax: z.number().int().min(0),
    mpCurrent: z.number().int().min(0),
    strength: z.number().int().min(-5).max(10),
    dexterity: z.number().int().min(-5).max(10),
    constitution: z.number().int().min(-5).max(10),
    intelligence: z.number().int().min(-5).max(10),
    wisdom: z.number().int().min(-5).max(10),
    charisma: z.number().int().min(-5).max(10),
    size: z.string().min(1, 'Size is required'),
    displacement: z.number().int().min(0).max(120),
    // Creation-time ability choices (all optional / soft-gated). classChoices
    // is keyed by className → { devoto?, caminho? }.
    classPowers: z.array(z.string()),
    originChoices: z.array(z.string()),
    trainedExpertises: z.array(z.string()),
    classChoices: z.record(
      z.string(),
      z.object({
        devoto: z.string().optional(),
        caminho: z.string().optional(),
      }),
    ),
    // Power sub-choices — powerId -> option id[] (totem/school/companion/weapon).
    powerChoices: z.record(z.string(), z.array(z.string())),
  })
  .superRefine((v, ctx) => {
    if (v.hpCurrent > v.hpMax) {
      ctx.addIssue({
        code: 'custom',
        path: ['hpCurrent'],
        message: 'HP current cannot exceed HP max',
      })
    }
    if (v.mpCurrent > v.mpMax) {
      ctx.addIssue({
        code: 'custom',
        path: ['mpCurrent'],
        message: 'MP current cannot exceed MP max',
      })
    }
    const seen = new Set<string>()
    v.classes.forEach((entry, index) => {
      if (!entry.className) return
      if (seen.has(entry.className)) {
        ctx.addIssue({
          code: 'custom',
          path: ['classes', index, 'className'],
          message: `Class "${entry.className}" already added — combine levels in one entry instead`,
        })
      } else {
        seen.add(entry.className)
      }
    })
  })

export type CharacterFormValues = z.infer<typeof characterSchema>

export const wizardDefaults: CharacterFormValues = {
  name: '',
  races: [],
  origin: '',
  classes: [],
  god: '',
  godPower: '',
  tibar: 0,
  startingWeaponSimple: '',
  startingWeaponMartial: '',
  startingArmor: '',
  startingShield: true,
  startingPurchases: {},
  originItemPicks: {},
  startingMoneyRolled: false,
  hpMax: 10,
  hpCurrent: 10,
  mpMax: 0,
  mpCurrent: 0,
  strength: 0,
  dexterity: 0,
  constitution: 0,
  intelligence: 0,
  wisdom: 0,
  charisma: 0,
  size: 'Médio',
  displacement: 9,
  classPowers: [],
  originChoices: [],
  trainedExpertises: [],
  classChoices: {},
  powerChoices: {},
}

export const toOptions = (values: string[]) =>
  values.map((v) => ({ value: v, label: v }))

/**
 * Whether a step's required choices are satisfied. Race pending floating/
 * subrace picks are intentionally NOT blocking (unplaced +1s simply don't
 * apply, matching the single-page behavior); a soft warning surfaces instead.
 */
export function stepReady(
  slug: StepSlug,
  v: CharacterFormValues,
  _raceChoices: RaceChoiceState,
): boolean {
  switch (slug) {
    case 'raca':
      return v.races.length > 0
    case 'classe':
      return !!v.classes[0]?.className
    case 'poderes':
      return true // elective powers + caminho/devoto are soft (sheet catches)
    case 'origem':
      return v.origin.length > 0
    case 'atributos':
      return true // preset-seeded, always within range
    case 'pericias':
      return true // trained perícias are soft (sheet catches)
    case 'equipamento':
      return true // kit picks are soft — finish on the sheet
    case 'vitalidade':
      return v.hpMax >= 1 && v.hpCurrent <= v.hpMax && v.mpCurrent <= v.mpMax
    case 'identidade':
      return v.name.trim().length > 0 && v.size.length > 0
    case 'resumo':
      return true
  }
}

/**
 * Index of the first step whose predecessors are all ready — i.e. the furthest
 * step the player may reach/jump to. Steps beyond this are locked.
 */
export function furthestReachableIndex(
  v: CharacterFormValues,
  raceChoices: RaceChoiceState,
): number {
  for (let i = 0; i < WIZARD_STEPS.length; i++) {
    if (!stepReady(WIZARD_STEPS[i].slug, v, raceChoices)) return i
  }
  return WIZARD_STEPS.length - 1
}

/** True once every step is ready — gates the final "Criar" action. Pending
 *  race choices are a soft warning, not a block (surfaced on the Resumo). */
export function allStepsReady(
  v: CharacterFormValues,
  raceChoices: RaceChoiceState,
): boolean {
  return WIZARD_STEPS.every((s) => stepReady(s.slug, v, raceChoices))
}
