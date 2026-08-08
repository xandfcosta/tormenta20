import { describe, expect, it } from 'vitest'
import { allCatalogItems } from '@/shared/lib/catalog-cache'
import { conditionsList } from '@/shared/lib/rules-catalog-cache'
import { spellCatalog } from '@/shared/lib/spell-cache'
import { catalogPowers, catalogSearchRows, matchesQuery } from './catalog-model'

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

describe('catalogSearchRows (unified GM search — ALE-22)', () => {
  // Caches primed in test-setup.ts, so the accessors are warm.
  const catalogs = {
    conditions: conditionsList(),
    spells: Object.values(spellCatalog()),
    powers: catalogPowers(),
    items: allCatalogItems(),
  }

  it('finds a spell even though it lives in the Magias catalog, not Condições', () => {
    const rows = catalogSearchRows('bola de fogo', catalogs)
    const spell = rows.find(
      (r) => r.kind === 'spell' && /bola de fogo/i.test(r.value.name),
    )
    expect(spell).toBeDefined()
    // The matched group carries a header; unrelated catalogs are omitted.
    expect(rows.some((r) => r.kind === 'header' && r.label === 'Magias')).toBe(true)
    expect(rows.some((r) => r.kind === 'header' && r.label === 'Condições')).toBe(
      false,
    )
  })

  it('each header count equals the entries that follow it until the next header', () => {
    const rows = catalogSearchRows('luz', catalogs)
    for (let i = 0; i < rows.length; i++) {
      const head = rows[i]!
      if (head.kind !== 'header') continue
      let n = 0
      for (let j = i + 1; j < rows.length && rows[j]!.kind !== 'header'; j++) n++
      expect(n).toBe(head.count)
    }
  })

  it('returns nothing when no catalog matches', () => {
    expect(catalogSearchRows('zzzqqxnomatch', catalogs)).toEqual([])
  })

  it('keys are unique across the merged catalogs', () => {
    const rows = catalogSearchRows('a', catalogs)
    const keys = new Set(rows.map((r) => r.key))
    expect(keys.size).toBe(rows.length)
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
