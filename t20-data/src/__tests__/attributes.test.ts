import { describe, expect, it } from 'vitest'
import { ATTRIBUTE_ABBR, ATTRIBUTE_KEYS } from '../attributes'

/**
 * PDF p10 — the six attributes of Tormenta 20 in canonical order:
 * Força, Destreza, Constituição, Inteligência, Sabedoria, Carisma.
 *
 * UI components and downstream catalogs key off both the english
 * `ATTRIBUTE_KEYS` (storage / type-narrowing) and the PT-BR
 * `ATTRIBUTE_ABBR` (visible labels). Drift between the two halves would
 * break the sheet labels silently — these specs pin both.
 */
describe('ATTRIBUTE_KEYS', () => {
  it('lists the six PDF attributes in canonical order', () => {
    expect(ATTRIBUTE_KEYS).toEqual([
      'strength',
      'dexterity',
      'constitution',
      'intelligence',
      'wisdom',
      'charisma',
    ])
  })

  it('has no duplicates', () => {
    expect(new Set(ATTRIBUTE_KEYS).size).toBe(ATTRIBUTE_KEYS.length)
  })
})

describe('ATTRIBUTE_ABBR', () => {
  it('covers every ATTRIBUTE_KEYS entry exactly once', () => {
    const abbrKeys = Object.keys(ATTRIBUTE_ABBR).sort()
    expect(abbrKeys).toEqual([...ATTRIBUTE_KEYS].sort())
  })

  it('uses PT-BR three-letter abbreviations', () => {
    // Sheet labels — checked in spec because the abbreviation table is
    // copy-pasted into the frontend (`expertise.test.ts`) and any drift
    // here would mismatch there.
    expect(ATTRIBUTE_ABBR).toEqual({
      strength: 'FOR',
      dexterity: 'DES',
      constitution: 'CON',
      intelligence: 'INT',
      wisdom: 'SAB',
      charisma: 'CAR',
    })
  })

  it('every abbreviation is unique', () => {
    const values = Object.values(ATTRIBUTE_ABBR)
    expect(new Set(values).size).toBe(values.length)
  })
})
