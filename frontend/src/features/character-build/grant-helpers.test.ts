import { describe, expect, it } from 'vitest'
import {
  classGrant,
  originGrant,
  raceAttributeDeltas,
  raceGrant,
  signed,
} from './grant-helpers'

describe('signed', () => {
  it('prefixes non-negative with +', () => {
    expect(signed(2)).toBe('+2')
    expect(signed(0)).toBe('+0')
    expect(signed(-1)).toBe('-1')
  })
})

describe('raceAttributeDeltas', () => {
  it('returns the fixed racial bonuses of a single race', () => {
    expect(raceAttributeDeltas(['Anão'])).toEqual({
      constitution: 2,
      wisdom: 1,
      dexterity: -1,
    })
  })

  it('sums bonuses across multiple races', () => {
    // Anão CON+2/SAB+1/DES-1 + Elfo INT+2/DES+1/CON-1
    expect(raceAttributeDeltas(['Anão', 'Elfo'])).toEqual({
      constitution: 1,
      wisdom: 1,
      dexterity: 0,
      intelligence: 2,
    })
  })

  it('excludes floating-choice races (Humano gets no fixed bonus)', () => {
    expect(raceAttributeDeltas(['Humano'])).toEqual({})
  })

  it('ignores unknown race ids', () => {
    expect(raceAttributeDeltas(['NotARace'])).toEqual({})
  })
})

describe('raceGrant', () => {
  it('lists attribute deltas and abilities for a known race', () => {
    const grant = raceGrant('Anão')
    expect(grant?.name).toBe('Anão')
    expect(grant?.deltas).toContainEqual(['constitution', 2])
    expect(grant?.abilities.length).toBeGreaterThan(0)
  })

  it('returns null for an unknown race', () => {
    expect(raceGrant('NotARace')).toBeNull()
  })
})

describe('classGrant', () => {
  it('exposes class vitals for a known class', () => {
    const grant = classGrant('Guerreiro')
    expect(grant.vitals?.pvInicial).toBeGreaterThan(0)
  })

  it('returns null vitals for an unknown class', () => {
    expect(classGrant('NotAClass').vitals).toBeNull()
  })
})

describe('originGrant', () => {
  it('lists benefits and the unique power for a known origin', () => {
    const grant = originGrant('Acólito')
    expect(grant?.name).toBe('Acólito')
    expect(grant?.benefits.length).toBeGreaterThan(0)
    expect(grant?.poderUnico?.name).toBeTruthy()
  })

  it('returns null for an unknown origin', () => {
    expect(originGrant('NotAnOrigin')).toBeNull()
  })
})
