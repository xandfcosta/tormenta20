import { describe, expect, it } from 'vitest'
import type { AttributeKey } from '../attributes'
import { EXPERTISES, trainingBonusForLevel } from '../expertises'

/**
 * PDF Cap 2 (Perícias, p114-123). Each perícia has a canonical key
 * attribute. The PDF table also shows the count: 29 perícias total.
 *
 * Training bonus table (p123): +2 at L1-6, +4 at L7-14, +6 at L15+.
 */
const BOOK_PERICIAS: Record<string, AttributeKey> = {
  Acrobacia: 'dexterity',
  Adestramento: 'charisma',
  Atletismo: 'strength',
  'Atuação': 'charisma',
  Cavalgar: 'dexterity',
  Conhecimento: 'intelligence',
  Cura: 'wisdom',
  Diplomacia: 'charisma',
  'Enganação': 'charisma',
  Fortitude: 'constitution',
  Furtividade: 'dexterity',
  Guerra: 'intelligence',
  Iniciativa: 'dexterity',
  'Intimidação': 'charisma',
  'Intuição': 'wisdom',
  'Investigação': 'intelligence',
  Jogatina: 'charisma',
  Ladinagem: 'dexterity',
  Luta: 'strength',
  Misticismo: 'intelligence',
  Nobreza: 'intelligence',
  'Ofício': 'intelligence',
  'Percepção': 'wisdom',
  Pilotagem: 'dexterity',
  Pontaria: 'dexterity',
  Reflexos: 'dexterity',
  'Religião': 'wisdom',
  'Sobrevivência': 'wisdom',
  Vontade: 'wisdom',
}

describe('EXPERTISES vs PDF Cap 2', () => {
  it('catalog contains 29 perícias', () => {
    expect(EXPERTISES.length).toBe(29)
  })

  for (const [name, attribute] of Object.entries(BOOK_PERICIAS)) {
    it(`${name}: attribute key = ${attribute}`, () => {
      const entry = EXPERTISES.find((e) => e.name === name)
      expect(entry, `perícia "${name}" missing`).toBeDefined()
      expect(entry!.attribute).toBe(attribute)
    })
  }
})

describe('trainingBonusForLevel (PDF p123)', () => {
  it.each([
    [1, 2],
    [3, 2],
    [6, 2],
    [7, 4],
    [10, 4],
    [14, 4],
    [15, 6],
    [18, 6],
    [20, 6],
  ])('level %i → +%i', (level, expected) => {
    expect(trainingBonusForLevel(level)).toBe(expected)
  })
})
