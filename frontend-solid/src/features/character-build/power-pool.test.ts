import { describe, expect, it } from 'vitest'
import {
  draftPowerPool,
  facetTally,
  filterPowers,
  powerLedger,
} from './power-pool'

const guerreiro = [{ className: 'Guerreiro', level: 3 }]

describe('draftPowerPool', () => {
  it('offers the class electives and every general power', () => {
    const pool = draftPowerPool({ classes: guerreiro, races: ['Humano'] }, {})

    expect(pool.some((p) => p.source === 'class')).toBe(true)
    expect(pool.some((p) => p.source === 'general')).toBe(true)
  })

  it('is empty without a primary class — there is no pool to draw from yet', () => {
    expect(draftPowerPool({ classes: [], races: ['Humano'] }, {})).toEqual([])
  })

  it('withholds poderes da Tormenta from a race that does not grant them', () => {
    const pool = draftPowerPool({ classes: guerreiro, races: ['Humano'] }, {})

    expect(pool.some((p) => p.source === 'tormenta')).toBe(false)
  })

  it('opens the Tormenta pool for Lefou', () => {
    const pool = draftPowerPool({ classes: guerreiro, races: ['Lefou'] }, {})

    expect(pool.some((p) => p.source === 'tormenta')).toBe(true)
  })

  it('opens it for a secondary race only once the player opts in', () => {
    const races = ['Humano', 'Lefou']
    const off = draftPowerPool({ classes: guerreiro, races }, {})
    const on = draftPowerPool({ classes: guerreiro, races }, { Lefou: { applied: true } })

    // A secondary race is flavor until the table agrees it is mechanical.
    expect(off.some((p) => p.source === 'tormenta')).toBe(false)
    expect(on.some((p) => p.source === 'tormenta')).toBe(true)
  })
})

describe('facetTally', () => {
  it('counts each source and the whole pool', () => {
    const pool = draftPowerPool({ classes: guerreiro, races: ['Lefou'] }, {})
    const tally = facetTally(pool)

    expect(tally.all).toBe(pool.length)
    expect(tally.class + tally.general + tally.tormenta).toBe(tally.all)
  })
})

describe('filterPowers', () => {
  const pool = draftPowerPool({ classes: guerreiro, races: ['Humano'] }, {})

  it('narrows to one source', () => {
    expect(filterPowers(pool, 'class', '').every((p) => p.source === 'class')).toBe(true)
  })

  it('matches the search against the name', () => {
    const hit = filterPowers(pool, 'all', 'ataque poderoso')

    expect(hit.length).toBeGreaterThan(0)
    expect(hit.every((p) => /ataque/i.test(p.name))).toBe(true)
  })

  it('combines facet and search', () => {
    expect(filterPowers(pool, 'class', 'zzzznada')).toEqual([])
  })
})

describe('powerLedger', () => {
  const pool = draftPowerPool({ classes: guerreiro, races: ['Humano'] }, {})
  const anyClassPower = () => {
    const found = pool.find((p) => p.source === 'class' && !p.choice?.repeatable)
    if (!found) throw new Error('Guerreiro sem poder de classe não-repetível no catálogo')
    return found.id
  }

  it('counts the slots the classes earn', () => {
    // Guerreiro Nv 3 → um slot por nível a partir do 2º.
    expect(powerLedger(guerreiro, [], {}, pool).total).toBe(2)
  })

  it('spends one slot per plain pick', () => {
    const ledger = powerLedger(guerreiro, [anyClassPower()], {}, pool)

    expect(ledger.used).toBe(1)
    expect(ledger.remaining).toBe(1)
  })

  it('never reports negative room left', () => {
    const repeated = [anyClassPower(), 'a', 'b', 'c']

    expect(powerLedger(guerreiro, repeated, {}, pool).remaining).toBe(0)
  })
})
