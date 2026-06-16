import { describe, expect, it } from 'vitest'
import {
  attributePresetForClass,
  CLASS_ATTRIBUTE_PRIORITY,
  CLASS_PRESET_VALUES,
  computeClassAttributePreset,
} from '../classes'

/**
 * PDF p17 — point-buy spread costs (1pt/+1, 2pt/+2, 4pt/+3). The class
 * preset uses 3/2/2/1/1/0 which sums to 4+2+2+1+1=10 points, the standard
 * starting budget. These tests pin:
 *  - all 14 classes have a priority list
 *  - each priority list is a permutation of the 6 attributes
 *  - computed preset matches 3/2/2/1/1/0 in priority order
 *  - the spread totals exactly 10 point-buy points
 */
const BOOK_CLASSES = [
  'Arcanista',
  'Bárbaro',
  'Bardo',
  'Bucaneiro',
  'Caçador',
  'Cavaleiro',
  'Clérigo',
  'Druida',
  'Guerreiro',
  'Inventor',
  'Ladino',
  'Lutador',
  'Nobre',
  'Paladino',
] as const

const POINT_BUY_COST: Record<number, number> = { 0: 0, 1: 1, 2: 2, 3: 4 }

describe('CLASS_ATTRIBUTE_PRIORITY shape', () => {
  it('covers all 14 base classes', () => {
    expect(Object.keys(CLASS_ATTRIBUTE_PRIORITY).sort()).toEqual(
      [...BOOK_CLASSES].sort(),
    )
  })

  for (const className of BOOK_CLASSES) {
    it(`${className}: priority is a permutation of the 6 attributes`, () => {
      const priority = CLASS_ATTRIBUTE_PRIORITY[className]
      expect(priority).toHaveLength(6)
      expect(new Set(priority).size).toBe(6)
    })
  }
})

describe('CLASS_PRESET_VALUES spread', () => {
  it('is the canonical 3/2/2/1/1/0', () => {
    expect([...CLASS_PRESET_VALUES]).toEqual([3, 2, 2, 1, 1, 0])
  })

  it('costs exactly 10 point-buy points (PDF p17 table)', () => {
    const total = CLASS_PRESET_VALUES.reduce(
      (sum, v) => sum + (POINT_BUY_COST[v] ?? 0),
      0,
    )
    expect(total).toBe(10)
  })
})

describe('computeClassAttributePreset', () => {
  it('maps priority entries to 3/2/2/1/1/0 in order', () => {
    const out = computeClassAttributePreset([
      'strength',
      'dexterity',
      'constitution',
      'intelligence',
      'wisdom',
      'charisma',
    ])
    expect(out).toEqual({
      strength: 3,
      dexterity: 2,
      constitution: 2,
      intelligence: 1,
      wisdom: 1,
      charisma: 0,
    })
  })

  it('returns 0 for attributes not in the priority list', () => {
    // Shouldn't happen for well-formed entries, but the function must
    // handle a truncated list without crashing.
    const out = computeClassAttributePreset(['strength', 'dexterity'])
    expect(out.constitution).toBe(0)
    expect(out.charisma).toBe(0)
  })
})

describe('attributePresetForClass', () => {
  it('Arcanista preset: INT 3, CON 2, DEX 2, CHA 1, WIS 1, STR 0', () => {
    expect(attributePresetForClass('Arcanista')).toEqual({
      intelligence: 3,
      constitution: 2,
      dexterity: 2,
      charisma: 1,
      wisdom: 1,
      strength: 0,
    })
  })

  it('Guerreiro preset: STR 3, CON 2, DEX 2, WIS 1, CHA 1, INT 0', () => {
    expect(attributePresetForClass('Guerreiro')).toEqual({
      strength: 3,
      constitution: 2,
      dexterity: 2,
      wisdom: 1,
      charisma: 1,
      intelligence: 0,
    })
  })

  it('returns null for unknown class', () => {
    expect(attributePresetForClass('Hexer')).toBeNull()
  })

  it('every class preset totals 10 point-buy points', () => {
    for (const className of BOOK_CLASSES) {
      const preset = attributePresetForClass(className)!
      const total = Object.values(preset).reduce(
        (sum, v) => sum + (POINT_BUY_COST[v] ?? 0),
        0,
      )
      expect(total, `${className} preset cost`).toBe(10)
    }
  })
})
