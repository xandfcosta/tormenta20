import { describe, expect, it } from 'vitest'
import {
  characterProficiencies,
  CLASS_PROFICIENCIES,
  PROFICIENCY_CATEGORIES,
  PROFICIENCY_LABELS,
} from '../proficiencies'

/**
 * PDF Cap 1 — each class entry's "Proficiências" line (pages 36-83). Pinned:
 *
 *   Arcanista (p36):    nenhuma
 *   Bárbaro (p42):      armas marciais, escudos
 *   Bardo (p46):        armas marciais
 *   Bucaneiro (p50):    armas marciais
 *   Caçador (p54):      armas marciais, escudos
 *   Cavaleiro (p58):    armas marciais, armaduras pesadas, escudos
 *   Clérigo (p62):      armaduras pesadas, escudos
 *   Druida (p66):       escudos
 *   Guerreiro (p64):    armas marciais, armaduras pesadas, escudos
 *   Inventor (p70):     nenhuma
 *   Ladino (p73):       nenhuma
 *   Lutador (p76):      nenhuma
 *   Nobre (p79):        armas marciais, armaduras pesadas, escudos
 *   Paladino (p82):     armas marciais, armaduras pesadas, escudos
 *
 * Per PDF p142: all characters are proficient with armas simples (baseline).
 * Per T20 stacking convention: armaduras-pesadas implies armaduras-leves.
 */
import type { ProficiencyCategory } from '../proficiencies'

const BOOK_CLASS_PROFICIENCIES: Record<string, ProficiencyCategory[]> = {
  Arcanista: [],
  'Bárbaro': ['armas-marciais', 'escudos'],
  Bardo: ['armas-marciais'],
  Bucaneiro: ['armas-marciais'],
  'Caçador': ['armas-marciais', 'escudos'],
  Cavaleiro: ['armas-marciais', 'armaduras-pesadas', 'escudos'],
  'Clérigo': ['armaduras-pesadas', 'escudos'],
  Druida: ['escudos'],
  Guerreiro: ['armas-marciais', 'armaduras-pesadas', 'escudos'],
  Inventor: [],
  Ladino: [],
  Lutador: [],
  Nobre: ['armas-marciais', 'armaduras-pesadas', 'escudos'],
  Paladino: ['armas-marciais', 'armaduras-pesadas', 'escudos'],
}

describe('CLASS_PROFICIENCIES vs PDF Cap 1 class entries', () => {
  it('covers all 14 classes', () => {
    expect(Object.keys(CLASS_PROFICIENCIES).sort()).toEqual(
      Object.keys(BOOK_CLASS_PROFICIENCIES).sort(),
    )
  })

  for (const [className, expected] of Object.entries(
    BOOK_CLASS_PROFICIENCIES,
  )) {
    it(`${className}: proficiencies = [${expected.join(', ') || 'none'}]`, () => {
      const actual = [...CLASS_PROFICIENCIES[className]].sort()
      expect(actual).toEqual([...expected].sort())
    })
  }
})

describe('PROFICIENCY_CATEGORIES + labels', () => {
  it('lists all 7 categories from PDF Cap 3 p142', () => {
    expect([...PROFICIENCY_CATEGORIES].sort()).toEqual(
      [
        'armas-simples',
        'armas-marciais',
        'armas-exoticas',
        'armas-de-fogo',
        'armaduras-leves',
        'armaduras-pesadas',
        'escudos',
      ].sort(),
    )
  })

  it('every category has a UI label', () => {
    for (const cat of PROFICIENCY_CATEGORIES) {
      expect(PROFICIENCY_LABELS[cat]).toBeTruthy()
    }
  })
})

describe('characterProficiencies — baseline + class merge', () => {
  it('every character is proficient with armas simples (PDF p142 baseline)', () => {
    const result = characterProficiencies(['Arcanista'])
    const simples = result.find((p) => p.category === 'armas-simples')
    expect(simples?.granted).toBe(true)
    expect(simples?.sources).toContain('Todas as classes')
  })

  it('Guerreiro grants armas-marciais, armaduras-pesadas, escudos + implicit armaduras-leves', () => {
    const result = characterProficiencies(['Guerreiro'])
    const granted = result.filter((p) => p.granted).map((p) => p.category)
    expect(granted.sort()).toEqual(
      [
        'armas-simples',
        'armas-marciais',
        'armaduras-leves',
        'armaduras-pesadas',
        'escudos',
      ].sort(),
    )
  })

  it('Arcanista grants only the baseline armas-simples', () => {
    const result = characterProficiencies(['Arcanista'])
    const granted = result.filter((p) => p.granted).map((p) => p.category)
    expect(granted).toEqual(['armas-simples'])
  })

  it('armaduras-pesadas implies armaduras-leves with the granting class as source', () => {
    const result = characterProficiencies(['Cavaleiro'])
    const leves = result.find((p) => p.category === 'armaduras-leves')
    expect(leves?.granted).toBe(true)
    expect(leves?.sources).toContain('Cavaleiro')
  })

  it('multiclass merges sources across classes', () => {
    const result = characterProficiencies(['Bárbaro', 'Cavaleiro'])
    const escudos = result.find((p) => p.category === 'escudos')
    expect(escudos?.sources.sort()).toEqual(['Bárbaro', 'Cavaleiro'])
  })

  it('returns one entry per category regardless of granted-ness', () => {
    const result = characterProficiencies(['Arcanista'])
    expect(result.map((p) => p.category).sort()).toEqual(
      [...PROFICIENCY_CATEGORIES].sort(),
    )
  })

  it('ignores unknown class names without crashing', () => {
    const result = characterProficiencies(['Hexer', 'Guerreiro'])
    const marciais = result.find((p) => p.category === 'armas-marciais')
    expect(marciais?.granted).toBe(true)
    expect(marciais?.sources).toEqual(['Guerreiro'])
  })

  it('classes without armaduras-pesadas don’t get implicit armaduras-leves', () => {
    const result = characterProficiencies(['Druida'])
    const leves = result.find((p) => p.category === 'armaduras-leves')
    expect(leves?.granted).toBe(false)
  })
})
