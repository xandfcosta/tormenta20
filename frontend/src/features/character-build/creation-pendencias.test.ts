import { describe, expect, it } from 'vitest'
import { creationBlockers, creationPendencias } from './creation-pendencias'
import { wizardDefaults } from './wizard-steps'

const complete = {
  ...wizardDefaults,
  name: 'Thal',
  races: ['Anão'],
  origin: 'Acólito',
  originChoices: ['origin-acolito-pericia-Cura', 'origin-acolito-pericia-Religião'],
  classes: [{ className: 'Guerreiro', level: 1 }],
  size: 'Médio',
}

const labels = (values: typeof complete, raceChoices = {}) =>
  creationPendencias(values, raceChoices).map((p) => p.label)

describe('creationPendencias', () => {
  it('says nothing when the build owes nothing', () => {
    const done = {
      ...complete,
      trainedExpertises: ['Fortitude', 'Luta', 'Pontaria', 'Percepção', 'Cavalgar'],
    }

    expect(creationPendencias(done, {}).every((p) => p.step !== 'pericias')).toBe(true)
  })

  it('counts the perícias still owed', () => {
    expect(labels(complete).join(' ')).toMatch(/perícia/i)
  })

  it('reports an origin still short of its two benefits', () => {
    const found = labels({ ...complete, originChoices: [] })

    expect(found.join(' ')).toMatch(/benefício/i)
  })

  it('reports an unplaced floating race bonus', () => {
    const found = labels({ ...complete, races: ['Humano'] }, {})

    expect(found.join(' ')).toMatch(/atributo/i)
  })

  it('reports an elective power slot left empty', () => {
    const found = labels({ ...complete, classes: [{ className: 'Guerreiro', level: 3 }] })

    expect(found.join(' ')).toMatch(/poder/i)
  })

  it('reports a caminho the class owes', () => {
    const found = labels({ ...complete, classes: [{ className: 'Arcanista', level: 1 }] })

    expect(found.join(' ')).toMatch(/caminho|devoto/i)
  })

  it('does not repeat the hard gates — those are blockers, not pendencies', () => {
    const found = labels({ ...complete, races: [] })

    expect(found.join(' ')).not.toMatch(/escolha uma raça/i)
  })

  it('carries the step each pendency belongs to, so the player can go fix it', () => {
    const pendencias = creationPendencias({ ...complete, originChoices: [] }, {})

    expect(pendencias.every((p) => p.step.length > 0)).toBe(true)
    expect(pendencias.some((p) => p.step === 'origem')).toBe(true)
  })
})

describe('creationBlockers', () => {
  it('says nothing when every gate is satisfied', () => {
    expect(creationBlockers(complete, {})).toEqual([])
  })

  it('names the missing race — a dead Criar button must have a reason', () => {
    const blockers = creationBlockers({ ...complete, races: [] }, {})

    expect(blockers.map((b) => b.step)).toContain('raca')
    expect(blockers[0].label).toMatch(/raça/i)
  })

  it('catches a PV atual above the maximum, which lives in Identidade', () => {
    const blockers = creationBlockers({ ...complete, hpCurrent: 99, hpMax: 10 }, {})

    expect(blockers.map((b) => b.step)).toContain('identidade')
  })

  it('lists every unmet gate, not just the first', () => {
    const blockers = creationBlockers({ ...complete, races: [], origin: '' }, {})

    expect(blockers.map((b) => b.step)).toEqual(['raca', 'origem'])
  })
})
