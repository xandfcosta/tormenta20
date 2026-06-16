import { describe, expect, it } from 'vitest'
import { CLASS_POWERS_CATALOG } from '../index'

/**
 * PDF Cap 1, Tabelas 1-5..1-18 — the progression table at the top of each
 * class entry. The PDF lists every level where a class gains an *automatic*
 * Habilidade de Classe (i.e., not "Poder de X" slots).
 *
 * Source verifications during initial review:
 *  - Guerreiro p64-66 (Tabela 1-13) — verified in PR #1
 *  - Ladino p72-74 (Tabela 1-15) — verified during this PR (spot check)
 *
 * The expected sets below are pulled from each class file's leading
 * comment, which itself cites the PDF table. This test is a regression
 * net: any silent rename, deletion, or level-shift on an auto power
 * breaks the build. Two-level guard — the test name shows what the book
 * says; the assertion checks the catalog still says the same thing.
 */
type LevelMultiset = number[]

function autoLevelsFor(className: string): LevelMultiset {
  return CLASS_POWERS_CATALOG.filter(
    (p) => p.className === className && p.grantedAtLevel !== undefined,
  )
    .map((p) => p.grantedAtLevel!)
    .sort((a, b) => a - b)
}

// Book auto-power *levels* per class — duplicates allowed when the table
// grants two habilidades at the same level (e.g., Paladino L1 has three).
const BOOK_AUTO_LEVELS: Record<string, LevelMultiset> = {
  Arcanista: [1, 1, 5, 9, 13, 17, 20],
  'Bárbaro': [1, 3, 5, 6, 8, 9, 11, 11, 14, 15, 16, 17, 20],
  Bardo: [1, 1, 2, 5, 6, 9, 10, 13, 14, 17, 20],
  Bucaneiro: [1, 1, 2, 3, 5, 7, 10, 11, 15, 19, 20],
  'Caçador': [1, 1, 3, 5, 5, 9, 13, 17, 20],
  Cavaleiro: [1, 1, 2, 5, 5, 7, 7, 9, 11, 12, 13, 15, 17, 17, 20],
  'Clérigo': [1, 1, 5, 9, 13, 17, 20],
  Druida: [1, 1, 1, 2, 6, 10, 14, 20],
  Guerreiro: [1, 3, 5, 6, 9, 13, 17, 20],
  Inventor: [1, 1, 2, 3, 5, 7, 8, 9, 10, 11, 13, 17, 20],
  Ladino: [1, 1, 2, 3, 4, 5, 7, 8, 9, 10, 11, 13, 15, 17, 19, 20],
  Lutador: [1, 1, 3, 5, 5, 7, 9, 9, 11, 13, 15, 17, 19, 20],
  Nobre: [1, 1, 1, 2, 3, 4, 5, 6, 10, 14, 18, 20],
  Paladino: [1, 1, 1, 2, 3, 5, 5, 6, 9, 10, 13, 14, 17, 18, 20],
}

describe('Auto-power level multisets pin to PDF Cap 1 tables', () => {
  for (const [className, expected] of Object.entries(BOOK_AUTO_LEVELS)) {
    it(`${className}: auto levels = [${expected.join(', ')}]`, () => {
      expect(autoLevelsFor(className)).toEqual(expected)
    })
  }
})

describe('Power id format', () => {
  it('every class power id starts with class.<slug>.', () => {
    for (const power of CLASS_POWERS_CATALOG) {
      expect(
        power.id.startsWith('class.'),
        `${power.id} should start with "class."`,
      ).toBe(true)
    }
  })

  it('every class power id is unique', () => {
    const ids = CLASS_POWERS_CATALOG.map((p) => p.id)
    expect(new Set(ids).size).toBe(ids.length)
  })

  it('catalog has 14 distinct class names', () => {
    const names = new Set(CLASS_POWERS_CATALOG.map((p) => p.className))
    expect(names.size).toBe(14)
  })
})
