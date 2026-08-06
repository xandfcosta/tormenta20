import { describe, expect, it } from 'vitest'
import { sheetSearchFilter } from './sheet-search-filter'

describe('sheetSearchFilter', () => {
  it('ignores accents both ways', () => {
    expect(sheetSearchFilter('Fúria Perícia', 'furia')).toBeGreaterThan(0)
    expect(sheetSearchFilter('Concentração de Combate', 'concentraçao')).toBeGreaterThan(0)
  })

  it('is case-insensitive', () => {
    expect(sheetSearchFilter('Totem Espiritual', 'TOTEM')).toBeGreaterThan(0)
  })

  it('empty search matches everything', () => {
    expect(sheetSearchFilter('Adaga', '')).toBe(1)
    expect(sheetSearchFilter('Adaga', '   ')).toBe(1)
  })

  it('fuzzy subsequence matches in-order scattered chars', () => {
    expect(sheetSearchFilter('Concentração de Combate', 'ccombate')).toBeGreaterThan(0)
    expect(sheetSearchFilter('Furtividade', 'ftvd')).toBeGreaterThan(0)
  })

  it('rejects out-of-order or missing chars', () => {
    expect(sheetSearchFilter('Adaga', 'z')).toBe(0)
    expect(sheetSearchFilter('Fúria', 'airuf')).toBe(0)
  })

  it('multi-term: every term must match', () => {
    expect(sheetSearchFilter('Concentração de Combate', 'conc comb')).toBeGreaterThan(0)
    expect(sheetSearchFilter('Concentração de Combate', 'conc xyz')).toBe(0)
  })

  it('ranks prefix > word-start > infix > subsequence', () => {
    const prefix = sheetSearchFilter('Fúria', 'fu')
    const wordStart = sheetSearchFilter('Pele de Ferro', 'fe')
    const infix = sheetSearchFilter('Enfeitiçar', 'fe')
    const subsequence = sheetSearchFilter('Fortitude Elevada', 'fde')
    expect(prefix).toBeGreaterThan(wordStart)
    expect(wordStart).toBeGreaterThan(infix)
    expect(infix).toBeGreaterThan(subsequence)
  })

  it('tighter subsequence spread scores higher', () => {
    const tight = sheetSearchFilter('Instinto', 'inst')
    const loose = sheetSearchFilter('Investigação constante', 'inst')
    expect(tight).toBeGreaterThan(loose)
  })
})
