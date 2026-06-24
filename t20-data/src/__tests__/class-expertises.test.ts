import { describe, expect, it } from 'vitest'
import {
  CLASS_EXPERTISES_TRAINED,
  classTrainedExpertises,
} from '../class-expertises'
import { EXPERTISE_NAMES } from '../expertises'
import type { ExpertiseName } from '../expertises'

/**
 * PDF Cap 1 (per-class entries p36-83) — "Perícias treinadas" line. Each
 * base class auto-trains a fixed list plus an "à escolha" pool. The four
 * classes Bucaneiro / Caçador / Guerreiro / Nobre also have a "X ou Y"
 * either-or slot that resolves to a single trained perícia.
 *
 * These specs pin every class's training data against the PDF so a
 * future catalog edit can't silently change a grant. The resolver
 * helper is tested for multiclass merging, either/or filtering, and the
 * "perícias only from the pool" gate.
 */
const PDF_FIXED_TRAINED: Record<string, ExpertiseName[]> = {
  Arcanista: ['Misticismo', 'Vontade'],
  'Bárbaro': ['Fortitude', 'Luta'],
  Bardo: ['Atuação', 'Reflexos'],
  Bucaneiro: ['Reflexos'],
  'Caçador': ['Sobrevivência'],
  Cavaleiro: ['Fortitude', 'Luta'],
  'Clérigo': ['Religião', 'Vontade'],
  Druida: ['Sobrevivência', 'Vontade'],
  Guerreiro: ['Fortitude'],
  Inventor: ['Ofício', 'Vontade'],
  Ladino: ['Ladinagem', 'Reflexos'],
  Lutador: ['Fortitude', 'Luta'],
  Nobre: ['Vontade'],
  Paladino: ['Luta', 'Vontade'],
}

const PDF_CHOOSE_COUNT: Record<string, number> = {
  Arcanista: 2,
  'Bárbaro': 4,
  Bardo: 6,
  Bucaneiro: 4,
  'Caçador': 6,
  Cavaleiro: 2,
  'Clérigo': 2,
  Druida: 4,
  Guerreiro: 2,
  Inventor: 4,
  Ladino: 8,
  Lutador: 4,
  Nobre: 4,
  Paladino: 2,
}

const PDF_EITHER_OR: Record<string, [ExpertiseName, ExpertiseName]> = {
  Bucaneiro: ['Luta', 'Pontaria'],
  'Caçador': ['Luta', 'Pontaria'],
  Guerreiro: ['Luta', 'Pontaria'],
  Nobre: ['Diplomacia', 'Intimidação'],
}

describe('CLASS_EXPERTISES_TRAINED — catalog completeness', () => {
  it('covers all 14 base classes from Cap 1', () => {
    expect(Object.keys(CLASS_EXPERTISES_TRAINED).sort()).toEqual(
      Object.keys(PDF_FIXED_TRAINED).sort(),
    )
  })

  it('every fixed perícia is a known EXPERTISE_NAMES entry', () => {
    const names = new Set<string>(EXPERTISE_NAMES)
    const bad: string[] = []
    for (const [className, entry] of Object.entries(CLASS_EXPERTISES_TRAINED)) {
      for (const p of entry.fixed) {
        if (!names.has(p)) bad.push(`${className}: fixed=${p}`)
      }
    }
    expect(bad).toEqual([])
  })

  it('every choosePool entry is a known EXPERTISE_NAMES entry', () => {
    const names = new Set<string>(EXPERTISE_NAMES)
    const bad: string[] = []
    for (const [className, entry] of Object.entries(CLASS_EXPERTISES_TRAINED)) {
      for (const p of entry.choosePool) {
        if (!names.has(p)) bad.push(`${className}: pool=${p}`)
      }
    }
    expect(bad).toEqual([])
  })

  it('every eitherOr option (when present) is a known EXPERTISE_NAMES entry', () => {
    const names = new Set<string>(EXPERTISE_NAMES)
    const bad: string[] = []
    for (const [className, entry] of Object.entries(CLASS_EXPERTISES_TRAINED)) {
      if (!entry.eitherOr) continue
      for (const p of entry.eitherOr.options) {
        if (!names.has(p)) bad.push(`${className}: eitherOr=${p}`)
      }
    }
    expect(bad).toEqual([])
  })

  it('choosePool has no duplicate entries', () => {
    for (const [className, entry] of Object.entries(CLASS_EXPERTISES_TRAINED)) {
      expect(
        new Set(entry.choosePool).size,
        `${className} has duplicate pool entries`,
      ).toBe(entry.choosePool.length)
    }
  })

  it('fixed and choosePool do not overlap (a perícia is either auto or pick)', () => {
    for (const [className, entry] of Object.entries(CLASS_EXPERTISES_TRAINED)) {
      const fixed = new Set(entry.fixed)
      for (const p of entry.choosePool) {
        expect(
          fixed.has(p),
          `${className}: ${p} appears in BOTH fixed and choosePool`,
        ).toBe(false)
      }
    }
  })
})

describe('CLASS_EXPERTISES_TRAINED — per-class vs PDF', () => {
  for (const [className, expected] of Object.entries(PDF_FIXED_TRAINED)) {
    it(`${className} fixed perícias match PDF (${expected.join(', ')})`, () => {
      const entry = CLASS_EXPERTISES_TRAINED[className]
      expect([...entry.fixed].sort()).toEqual([...expected].sort())
    })
  }

  for (const [className, count] of Object.entries(PDF_CHOOSE_COUNT)) {
    it(`${className} chooseCount = ${count}`, () => {
      expect(CLASS_EXPERTISES_TRAINED[className].chooseCount).toBe(count)
    })
  }

  for (const [className, options] of Object.entries(PDF_EITHER_OR)) {
    it(`${className} eitherOr options = ${options.join(' | ')}`, () => {
      const entry = CLASS_EXPERTISES_TRAINED[className]
      expect(entry.eitherOr).toBeDefined()
      expect([...entry.eitherOr!.options].sort()).toEqual([...options].sort())
    })
  }

  it('classes WITHOUT an either/or slot do not declare one', () => {
    const withEitherOr = new Set(Object.keys(PDF_EITHER_OR))
    for (const className of Object.keys(CLASS_EXPERTISES_TRAINED)) {
      if (withEitherOr.has(className)) continue
      expect(
        CLASS_EXPERTISES_TRAINED[className].eitherOr,
        `${className} has unexpected eitherOr`,
      ).toBeUndefined()
    }
  })
})

describe('classTrainedExpertises — resolver', () => {
  it('returns the fixed list with no picks supplied', () => {
    const result = classTrainedExpertises(['Arcanista'])
    expect([...result].sort()).toEqual(['Misticismo', 'Vontade'])
  })

  it('folds in chosen perícias from the pool', () => {
    const result = classTrainedExpertises(['Arcanista'], {
      Arcanista: { chosen: ['Conhecimento', 'Percepção'] },
    })
    expect([...result].sort()).toEqual(
      ['Conhecimento', 'Misticismo', 'Percepção', 'Vontade'].sort(),
    )
  })

  it('ignores chosen perícias that are not in the pool (defensive)', () => {
    const result = classTrainedExpertises(['Arcanista'], {
      Arcanista: { chosen: ['Atletismo'] }, // not in Arcanista's pool
    })
    expect(result.has('Atletismo')).toBe(false)
  })

  it('resolves eitherOr to the chosen option', () => {
    const result = classTrainedExpertises(['Guerreiro'], {
      Guerreiro: { eitherOr: 'Pontaria' },
    })
    expect(result.has('Pontaria')).toBe(true)
    expect(result.has('Luta')).toBe(false)
  })

  it('drops an eitherOr choice that is not in the option list', () => {
    const result = classTrainedExpertises(['Guerreiro'], {
      Guerreiro: { eitherOr: 'Misticismo' as never }, // not in options
    })
    expect(result.has('Misticismo')).toBe(false)
    expect(result.has('Luta')).toBe(false)
    expect(result.has('Pontaria')).toBe(false)
  })

  it('multiclass union: a perícia is trained if ANY class trains it', () => {
    // Guerreiro fixed = Fortitude; Arcanista fixed = Misticismo, Vontade.
    const result = classTrainedExpertises(['Guerreiro', 'Arcanista'])
    expect(result.has('Fortitude')).toBe(true)
    expect(result.has('Misticismo')).toBe(true)
    expect(result.has('Vontade')).toBe(true)
  })

  it('handles per-class picks independently in multiclass', () => {
    const result = classTrainedExpertises(['Guerreiro', 'Bardo'], {
      Guerreiro: { eitherOr: 'Luta', chosen: ['Atletismo'] },
      Bardo: { chosen: ['Acrobacia', 'Furtividade'] },
    })
    expect(result.has('Luta')).toBe(true)
    expect(result.has('Atletismo')).toBe(true)
    expect(result.has('Acrobacia')).toBe(true)
    expect(result.has('Furtividade')).toBe(true)
  })

  it('ignores unknown class names without crashing', () => {
    const result = classTrainedExpertises(['Hexer', 'Guerreiro'])
    expect(result.has('Fortitude')).toBe(true)
  })

  it('returns an empty Set for an empty classes list', () => {
    expect(classTrainedExpertises([]).size).toBe(0)
  })
})

describe('Resistências (Fortitude / Reflexos / Vontade) — PDF p36-83 derivation', () => {
  // Per PDF: trained-in-save status is read off the perícias line, NOT
  // a separate field. Multiclass: union of all saves trained across classes.
  it('Bárbaro is auto-trained in Fortitude', () => {
    expect(classTrainedExpertises(['Bárbaro']).has('Fortitude')).toBe(true)
  })

  it('Ladino is auto-trained in Reflexos', () => {
    expect(classTrainedExpertises(['Ladino']).has('Reflexos')).toBe(true)
  })

  it('Clérigo is auto-trained in Vontade', () => {
    expect(classTrainedExpertises(['Clérigo']).has('Vontade')).toBe(true)
  })

  it('Caçador is NOT auto-trained in any save (both in choose pool)', () => {
    const result = classTrainedExpertises(['Caçador'])
    expect(result.has('Fortitude')).toBe(false)
    expect(result.has('Reflexos')).toBe(false)
    // Caçador *can* pick Fortitude or Reflexos from the pool — but without
    // an explicit pick neither is granted.
  })
})
