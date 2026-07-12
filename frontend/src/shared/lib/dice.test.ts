import { describe, expect, it } from 'vitest'
import { rollD20 } from './dice'

describe('rollD20', () => {
  it('always returns an integer in 1–20', () => {
    for (let i = 0; i < 500; i++) {
      const r = rollD20()
      expect(Number.isInteger(r)).toBe(true)
      expect(r).toBeGreaterThanOrEqual(1)
      expect(r).toBeLessThanOrEqual(20)
    }
  })
})
