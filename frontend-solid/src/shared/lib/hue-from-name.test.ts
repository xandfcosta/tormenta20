import { describe, expect, it } from 'vitest'
import { hueFromName } from './hue-from-name'

describe('hueFromName', () => {
  it('is deterministic for the same name', () => {
    expect(hueFromName('Thorvald')).toBe(hueFromName('Thorvald'))
  })

  it('always returns a hue within [0, 359]', () => {
    for (const name of ['A', 'Thorvald', 'Míriel Ávila', '', '日本語']) {
      const hue = hueFromName(name)
      expect(hue).toBeGreaterThanOrEqual(0)
      expect(hue).toBeLessThan(360)
    }
  })

  it('spreads different names across different hues', () => {
    const names = ['Thorvald', 'Akira', 'Míriel', 'Bardo', 'Zenith']
    const hues = new Set(names.map(hueFromName))
    // Not a strict guarantee, but a collision across all 5 would signal a
    // degenerate hash. Expect at least 4 distinct buckets.
    expect(hues.size).toBeGreaterThanOrEqual(4)
  })

  it('empty string is stable (hue 0)', () => {
    expect(hueFromName('')).toBe(0)
  })
})
