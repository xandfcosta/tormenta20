import { describe, expect, it } from 'vitest'
import {
  allStepsReady,
  type CharacterFormValues,
  furthestReachableIndex,
  isStepSlug,
  stepAt,
  stepReady,
  WIZARD_STEPS,
  wizardDefaults,
} from './wizard-steps'

const complete: CharacterFormValues = {
  ...wizardDefaults,
  name: 'Thorin',
  races: ['Anão'],
  origin: 'Batedor',
  classes: [{ className: 'Guerreiro', level: 1 }],
  size: 'Médio',
}

describe('stepReady', () => {
  it('raca needs at least one race', () => {
    expect(stepReady('raca', wizardDefaults, {})).toBe(false)
    expect(stepReady('raca', { ...wizardDefaults, races: ['Elfo'] }, {})).toBe(
      true,
    )
  })

  it('classe needs a primary class', () => {
    expect(stepReady('classe', wizardDefaults, {})).toBe(false)
    expect(stepReady('classe', complete, {})).toBe(true)
  })

  it('origem needs an origin', () => {
    expect(stepReady('origem', wizardDefaults, {})).toBe(false)
    expect(stepReady('origem', complete, {})).toBe(true)
  })

  it('atributos is always ready (preset-seeded)', () => {
    expect(stepReady('atributos', wizardDefaults, {})).toBe(true)
  })

  it('vitalidade enforces current ≤ max', () => {
    expect(
      stepReady('vitalidade', { ...complete, hpCurrent: 99, hpMax: 10 }, {}),
    ).toBe(false)
    expect(stepReady('vitalidade', complete, {})).toBe(true)
  })

  it('identidade needs a name and size', () => {
    expect(stepReady('identidade', { ...complete, name: '  ' }, {})).toBe(false)
    expect(stepReady('identidade', complete, {})).toBe(true)
  })
})

describe('furthestReachableIndex', () => {
  it('is the first (raça) step when nothing is chosen', () => {
    expect(furthestReachableIndex(wizardDefaults, {})).toBe(0)
  })

  it('advances one step once its predecessor is ready', () => {
    expect(
      furthestReachableIndex({ ...wizardDefaults, races: ['Anão'] }, {}),
    ).toBe(1)
  })

  it('reaches the last step when every step is ready', () => {
    expect(furthestReachableIndex(complete, {})).toBe(WIZARD_STEPS.length - 1)
  })
})

describe('allStepsReady', () => {
  it('false until every required choice is made', () => {
    expect(allStepsReady(wizardDefaults, {})).toBe(false)
  })
  it('true for a complete build', () => {
    expect(allStepsReady(complete, {})).toBe(true)
  })
})

describe('stepAt — andar um passo', () => {
  it('avança e recua na ordem declarada', () => {
    expect(stepAt('raca', 1)).toBe('classe')
    expect(stepAt('classe', -1)).toBe('raca')
  })

  it('devolve null nas pontas (não circula)', () => {
    expect(stepAt('raca', -1)).toBeNull()
    expect(stepAt('resumo', 1)).toBeNull()
  })
})

describe('isStepSlug — slug vindo da URL', () => {
  it('aceita um passo real', () => {
    expect(isStepSlug('pericias')).toBe(true)
  })

  it('recusa qualquer outra coisa', () => {
    expect(isStepSlug('inventario')).toBe(false)
    expect(isStepSlug('')).toBe(false)
  })
})
