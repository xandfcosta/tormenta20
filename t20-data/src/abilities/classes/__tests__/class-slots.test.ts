import { describe, expect, it } from 'vitest'
import {
  CLASS_POWER_SLOTS,
  CLASS_POWERS_CATALOG,
  classPowerModifiers,
  ownedClassPowers,
  slotsForClassLevel,
  unlockedKinds,
} from '../index'

/**
 * PDF Cap 1 (p33): "Todas as classes possuem uma habilidade 'Poder' (Poder
 * de Arcanista, Poder de Bárbaro...) que permite escolher um poder de uma
 * lista." Per-class progression tables (Tabela 1-5..1-18) show a free slot
 * at every level from 2 to 20.
 *
 * These tests guard:
 *  - all 14 classes are present
 *  - each opens 19 power slots at L2..L20
 *  - slotsForClassLevel filters by classLevel correctly
 *  - ownedClassPowers + classPowerModifiers fold auto+chosen correctly
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

describe('CLASS_POWER_SLOTS vs PDF Cap 1', () => {
  it('contains all 14 base classes', () => {
    expect(Object.keys(CLASS_POWER_SLOTS).sort()).toEqual(
      [...BOOK_CLASSES].sort(),
    )
  })

  for (const className of BOOK_CLASSES) {
    it(`${className}: opens 19 power slots at levels 2..20`, () => {
      const slots = CLASS_POWER_SLOTS[className]
      expect(slots).toHaveLength(19)
      expect(slots.map((s) => s.level)).toEqual([
        2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20,
      ])
    })
  }
})

describe('slotsForClassLevel — earned vs locked', () => {
  it('returns 0 slots at L1 (no class power until L2)', () => {
    expect(slotsForClassLevel('Guerreiro', 1)).toEqual([])
  })

  it('returns 1 slot at L2', () => {
    const slots = slotsForClassLevel('Guerreiro', 2)
    expect(slots).toHaveLength(1)
    expect(slots[0]?.level).toBe(2)
  })

  it('returns 4 slots at L5 (L2 through L5)', () => {
    expect(slotsForClassLevel('Guerreiro', 5)).toHaveLength(4)
  })

  it('returns 19 slots at L20', () => {
    expect(slotsForClassLevel('Guerreiro', 20)).toHaveLength(19)
  })

  it('returns empty array for unknown class', () => {
    expect(slotsForClassLevel('Não Existe', 10)).toEqual([])
  })
})

describe('unlockedKinds', () => {
  it('returns the class kind once L2 reached', () => {
    expect(unlockedKinds('Guerreiro', 2)).toEqual(['guerreiro'])
  })

  it('returns empty at L1', () => {
    expect(unlockedKinds('Guerreiro', 1)).toEqual([])
  })
})

describe('ownedClassPowers', () => {
  it('returns auto powers granted at-or-below classLevel', () => {
    // Guerreiro auto powers: L1, L3, L5, L6, L9, L13, L17, L20.
    const owned = ownedClassPowers('Guerreiro', 5, new Set())
    const levels = owned.map((p) => p.grantedAtLevel).sort((a, b) => a! - b!)
    expect(levels).toEqual([1, 3, 5])
  })

  it('excludes auto powers above classLevel', () => {
    const owned = ownedClassPowers('Guerreiro', 1, new Set())
    expect(owned.map((p) => p.grantedAtLevel)).toEqual([1])
  })

  it('includes elective powers present in chosenIds set', () => {
    const ambidestria = CLASS_POWERS_CATALOG.find(
      (p) => p.id === 'class.guerreiro.ambidestria',
    )
    expect(ambidestria).toBeDefined()
    const owned = ownedClassPowers(
      'Guerreiro',
      2,
      new Set([ambidestria!.id]),
    )
    expect(owned.some((p) => p.id === ambidestria!.id)).toBe(true)
  })

  it('does not leak powers from other classes', () => {
    const owned = ownedClassPowers('Guerreiro', 20, new Set())
    for (const p of owned) expect(p.className).toBe('Guerreiro')
  })
})

describe('classPowerModifiers — auto + chosen fold', () => {
  it('returns empty array when no powers carry modifiers', () => {
    // Guerreiro auto powers don't have any modifiers in the catalog.
    const mods = classPowerModifiers('Guerreiro', 20, new Set())
    expect(mods).toEqual([])
  })

  it('ignores chosen ids that aren’t in the catalog', () => {
    const mods = classPowerModifiers('Guerreiro', 5, new Set(['ghost.id']))
    expect(mods).toEqual([])
  })
})
