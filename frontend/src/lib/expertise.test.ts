import { describe, expect, it } from 'vitest'
import type { Character, CharacterExpertise } from './api'
import {
  EXPERTISES,
  expertiseStateFor,
  expertiseTotal,
  trainingBonusForLevel,
} from './expertise'

/**
 * PDF Cap 2 (Perícias) — total = ½ nível + atributo + (treinamento se
 * treinado). Trained bonus: +2 at L1-6, +4 at L7-14, +6 at L15+.
 * Trained-only perícias (Adestramento, Conhecimento, Guerra, …) can't be
 * used at all when not trained; this layer only computes the *total* —
 * the UI hides the row when trained=false.
 */
function character(over: Partial<Character> = {}): Character {
  return {
    id: 1,
    name: 'X',
    level: 1,
    hpMax: 12,
    hpCurrent: 12,
    mpMax: 4,
    mpCurrent: 4,
    strength: 0,
    dexterity: 0,
    constitution: 0,
    intelligence: 0,
    wisdom: 0,
    charisma: 0,
    size: 'M',
    displacement: 9,
    expertises: [],
    races: [],
    classes: [],
    items: [],
    activeEffects: [],
    ...over,
  } as Character
}

describe('expertiseTotal — PDF p123 formula', () => {
  it('L1 untrained Acrobacia with DEX 2 → 0+2+0 = 2', () => {
    const c = character({ level: 1, dexterity: 2 })
    const state: CharacterExpertise = {
      name: 'Acrobacia',
      attribute: 'dexterity',
      trained: false,
      custom: false,
    }
    expect(expertiseTotal(c, state)).toBe(2)
  })

  it('L1 trained Acrobacia with DEX 2 → 0+2+2 = 4 (½ level rounds down)', () => {
    const c = character({ level: 1, dexterity: 2 })
    const state: CharacterExpertise = {
      name: 'Acrobacia',
      attribute: 'dexterity',
      trained: true,
      custom: false,
    }
    expect(expertiseTotal(c, state)).toBe(4)
  })

  it('L7 trained Luta with STR 3 → 3+3+4 = 10', () => {
    const c = character({ level: 7, strength: 3 })
    const state: CharacterExpertise = {
      name: 'Luta',
      attribute: 'strength',
      trained: true,
      custom: false,
    }
    expect(expertiseTotal(c, state)).toBe(10)
  })

  it('L20 trained Misticismo with INT 5 → 10+5+6 = 21', () => {
    const c = character({ level: 20, intelligence: 5 })
    const state: CharacterExpertise = {
      name: 'Misticismo',
      attribute: 'intelligence',
      trained: true,
      custom: false,
    }
    expect(expertiseTotal(c, state)).toBe(21)
  })

  it('½ level rounds down (L3 → 1, L4 → 2)', () => {
    const state: CharacterExpertise = {
      name: 'Atletismo',
      attribute: 'strength',
      trained: false,
      custom: false,
    }
    expect(expertiseTotal(character({ level: 3, strength: 0 }), state)).toBe(1)
    expect(expertiseTotal(character({ level: 4, strength: 0 }), state)).toBe(2)
  })
})

describe('expertiseStateFor — fallback to default', () => {
  it('returns the matching row when present', () => {
    const row: CharacterExpertise = {
      name: 'Acrobacia',
      attribute: 'dexterity',
      trained: true,
      custom: false,
    }
    const c = character({ expertises: [row] })
    expect(expertiseStateFor(c, EXPERTISES[0])).toBe(row)
  })

  it('returns a stub row when the row is missing (trained=false, custom=false)', () => {
    const c = character({ expertises: [] })
    const def = EXPERTISES.find((e) => e.name === 'Acrobacia')!
    const state = expertiseStateFor(c, def)
    expect(state.trained).toBe(false)
    expect(state.custom).toBe(false)
    expect(state.attribute).toBe(def.attribute)
  })
})

describe('EXPERTISES — frontend def list', () => {
  it('mirrors t20-data count + flags trained-only perícias', () => {
    expect(EXPERTISES).toHaveLength(29)
    const trainedOnly = EXPERTISES.filter((e) => e.trainedOnly).map((e) => e.name)
    expect(trainedOnly.sort()).toEqual(
      [
        'Adestramento',
        'Conhecimento',
        'Guerra',
        'Jogatina',
        'Ladinagem',
        'Misticismo',
        'Nobreza',
        'Ofício',
        'Pilotagem',
        'Religião',
      ].sort(),
    )
  })

  it('every def carries an abbr matching its attribute', () => {
    for (const e of EXPERTISES) {
      expect(e.abbr).toBeTruthy()
      expect(['FOR', 'DES', 'CON', 'INT', 'SAB', 'CAR']).toContain(e.abbr)
    }
  })
})

describe('trainingBonusForLevel re-export', () => {
  it('matches the L1/L7/L15 thresholds', () => {
    expect(trainingBonusForLevel(1)).toBe(2)
    expect(trainingBonusForLevel(7)).toBe(4)
    expect(trainingBonusForLevel(15)).toBe(6)
  })
})
