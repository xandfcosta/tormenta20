import { describe, expect, it } from 'vitest'
import {
  CATALOG_ITEMS,
  familyFor,
  getCatalogItem,
  isCatalogId,
  requiredProficiency,
} from '../catalog'
import { ARMORS } from '../catalog/armors'
import { SHIELDS } from '../catalog/shields'
import { WEAPONS } from '../catalog/weapons'

/**
 * Cross-catalog plumbing: ids are globally unique, lookup helpers behave,
 * proficiency mapping covers every wielded/vested combat item type.
 * Proficiency category names mirror PDF Cap 3 (p142): armas simples,
 * marciais, exóticas, de fogo, armaduras leves, pesadas, escudos.
 */
describe('CATALOG_ITEMS — global plumbing', () => {
  it('every catalog id is globally unique across all item files', () => {
    const ids = CATALOG_ITEMS.map((it) => it.id)
    expect(new Set(ids).size).toBe(ids.length)
  })

  it('getCatalogItem returns the same instance as the source array', () => {
    const adaga = getCatalogItem('adaga')
    expect(adaga).toBeDefined()
    expect(adaga).toBe(WEAPONS.find((w) => w.id === 'adaga'))
  })

  it('getCatalogItem returns undefined for unknown id', () => {
    expect(getCatalogItem('not-a-real-id')).toBeUndefined()
  })

  it('isCatalogId distinguishes catalog ids from unknowns', () => {
    expect(isCatalogId('adaga')).toBe(true)
    expect(isCatalogId('ghost')).toBe(false)
  })
})

describe('familyFor — overlay attach gating', () => {
  it('classifies weapons under family=weapon', () => {
    for (const w of WEAPONS) expect(familyFor(w)).toBe('weapon')
  })

  it('classifies armors under family=armor', () => {
    for (const a of ARMORS) expect(familyFor(a)).toBe('armor')
  })

  it('classifies shields under family=shield', () => {
    for (const s of SHIELDS) expect(familyFor(s)).toBe('shield')
  })
})

describe('requiredProficiency — Cap 3 proficiency categories', () => {
  it.each([
    ['adaga', 'armas-simples'],
    ['espada-longa', 'armas-marciais'],
    ['katana', 'armas-exoticas'],
    ['pistola', 'armas-de-fogo'],
    ['armadura-couro', 'armaduras-leves'],
    ['cota-malha', 'armaduras-pesadas'],
    ['escudo-leve', 'escudos'],
  ])('%s requires %s', (id, expected) => {
    const item = getCatalogItem(id)!
    expect(requiredProficiency(item)).toBe(expected)
  })

  it('returns null for non-combat items', () => {
    // Apparel, consumables, materials etc don't gate on proficiency.
    const nonCombat = CATALOG_ITEMS.filter(
      (it) =>
        !it.category.startsWith('weapon-') &&
        !it.category.startsWith('armor-') &&
        it.category !== 'shield',
    )
    for (const it of nonCombat) expect(requiredProficiency(it)).toBeNull()
  })
})
