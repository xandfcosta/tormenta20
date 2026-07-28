import { EXPERTISE_NAMES } from '@tormenta20/t20-data'
import { describe, expect, it } from 'vitest'
import { bandPicksRemaining, periciaPlan } from './pericia-helpers'

describe('periciaPlan — class band vs +INT band', () => {
  it('keeps the base class count free of the INT bonus', () => {
    const plan = periciaPlan('Bárbaro', 3)
    // Bárbaro base chooseCount is 4; INT must NOT inflate it.
    expect(plan?.classCount).toBe(4)
    expect(plan?.intCount).toBe(3)
  })

  it('folds a non-positive INT mod to a zero-count band', () => {
    const plan = periciaPlan('Bárbaro', -1)
    expect(plan?.intCount).toBe(0)
  })

  it('draws the INT pool from perícias OUTSIDE the class list (book rule)', () => {
    const plan = periciaPlan('Arcanista', 2)
    if (!plan) throw new Error('expected a plan for Arcanista')
    // Partition: no perícia appears in both bands, and fixed/either-or are excluded.
    for (const name of plan.intPool) {
      expect(plan.classPool).not.toContain(name)
      expect(plan.fixed).not.toContain(name)
    }
    // The two bands + fixed + either-or cover the whole perícia space.
    const union = new Set([
      ...plan.fixed,
      ...(plan.eitherOr ?? []),
      ...plan.classPool,
      ...plan.intPool,
    ])
    expect(union.size).toBe(EXPERTISE_NAMES.length)
  })

  it('returns null for an unknown class', () => {
    expect(periciaPlan('Hexer', 2)).toBeNull()
  })
})

describe('bandPicksRemaining', () => {
  it('counts only picks that belong to the given pool', () => {
    const pool = ['Guerra', 'Ofício', 'Percepção']
    // 'Acrobacia' is off-pool and must not consume the class cap.
    expect(bandPicksRemaining(pool, 2, ['Guerra', 'Acrobacia'])).toBe(1)
  })

  it('never goes below zero', () => {
    const pool = ['Guerra', 'Ofício']
    expect(bandPicksRemaining(pool, 1, ['Guerra', 'Ofício'])).toBe(0)
  })
})
