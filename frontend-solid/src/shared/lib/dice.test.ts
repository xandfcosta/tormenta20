import { describe, expect, it } from 'vitest'
import { parseDiceNotation, rollDice } from './dice'

describe('rollDice', () => {
  it('stays inside the possible range of the roll', () => {
    for (let i = 0; i < 200; i++) {
      const total = rollDice(4, 6)
      expect(total).toBeGreaterThanOrEqual(4)
      expect(total).toBeLessThanOrEqual(24)
    }
  })

  it('rolls nothing for zero dice', () => {
    expect(rollDice(0, 6)).toBe(0)
  })
})

describe('parseDiceNotation', () => {
  it('reads the notation the origin grants carry', () => {
    expect(parseDiceNotation('2d6')).toEqual({ count: 2, sides: 6 })
  })

  it('refuses junk instead of rolling NaN into the wallet', () => {
    expect(parseDiceNotation('d6')).toBeNull()
    expect(parseDiceNotation('2x6')).toBeNull()
    expect(parseDiceNotation('0d6')).toBeNull()
  })
})
