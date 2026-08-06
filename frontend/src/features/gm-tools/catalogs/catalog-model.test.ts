import { describe, expect, it } from 'vitest'
import { catalogPowers, matchesQuery } from './catalog-model'

describe('catalogPowers', () => {
  // Cache is primed once in test-setup.ts, so the accessor is warm here.
  const powers = catalogPowers()

  it('merges the three power sources, sorted by name, each with text', () => {
    expect(powers.length).toBeGreaterThan(500)
    for (let i = 1; i < powers.length; i++) {
      expect(
        powers[i - 1]!.name.localeCompare(powers[i]!.name, 'pt-BR'),
      ).toBeLessThanOrEqual(0)
    }
    // Every entry carries a description (divine powers, which lack one, are
    // excluded by design).
    expect(powers.every((p) => p.description.length > 0)).toBe(true)
  })

  it('keeps ids unique across the merged sources', () => {
    const ids = new Set(powers.map((p) => p.id))
    expect(ids.size).toBe(powers.length)
  })
})

describe('matchesQuery', () => {
  it('empty query matches everything', () => {
    expect(matchesQuery(['whatever'], '')).toBe(true)
  })

  it('is accent- and case-insensitive', () => {
    expect(matchesQuery(['Ilusão'], 'ilusao')).toBe(true)
  })

  it('ANDs all terms across the searchable fields', () => {
    expect(matchesQuery(['Luz', 'ilumina a área'], 'luz area')).toBe(true)
    expect(matchesQuery(['Luz', 'ilumina a área'], 'luz trevas')).toBe(false)
  })
})
