import { describe, expect, it } from 'vitest'
import { CATALOG_POWERS, matchesQuery } from './catalog-model'

describe('CATALOG_POWERS', () => {
  it('merges the three power sources, sorted by name, each with text', () => {
    expect(CATALOG_POWERS.length).toBeGreaterThan(500)
    for (let i = 1; i < CATALOG_POWERS.length; i++) {
      expect(
        CATALOG_POWERS[i - 1]!.name.localeCompare(
          CATALOG_POWERS[i]!.name,
          'pt-BR',
        ),
      ).toBeLessThanOrEqual(0)
    }
    // Every entry carries a description (divine powers, which lack one, are
    // excluded by design).
    expect(CATALOG_POWERS.every((p) => p.description.length > 0)).toBe(true)
  })

  it('keeps ids unique across the merged sources', () => {
    const ids = new Set(CATALOG_POWERS.map((p) => p.id))
    expect(ids.size).toBe(CATALOG_POWERS.length)
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
