import {
  SPELL_CATALOG,
  spellByName as srcSpellByName,
  spellEffectByName as srcSpellEffectByName,
} from '@tormenta20/t20-data'
import { beforeAll, describe, expect, it } from 'vitest'
import {
  buffSpells,
  hasSpell,
  isSpellCatalogPrimed,
  primeSpellCatalog,
  spellById,
  spellByName,
  spellCatalog,
  spellEffectByName,
} from './spell-cache'

/**
 * The cache must faithfully mirror the t20-data spell lookups — the front now
 * runs these against fetched-and-cached data instead of the bundled catalog.
 * Prime with the real catalog (Node test → no bundle concern) and compare.
 */
describe('spell-cache', () => {
  const anyId = Object.keys(SPELL_CATALOG)[0]

  beforeAll(() => {
    primeSpellCatalog(SPELL_CATALOG)
  })

  it('reports primed after prime', () => {
    expect(isSpellCatalogPrimed()).toBe(true)
  })

  it('spellCatalog returns the primed record', () => {
    expect(spellCatalog()).toBe(SPELL_CATALOG)
  })

  it('spellById returns the entry and throws on unknown (matches source)', () => {
    expect(spellById(anyId)).toBe(SPELL_CATALOG[anyId])
    expect(() => spellById('not-a-spell')).toThrow(/unknown spell id/)
  })

  it('hasSpell is a non-throwing existence check', () => {
    expect(hasSpell(anyId)).toBe(true)
    expect(hasSpell('not-a-spell')).toBe(false)
  })

  it('spellByName mirrors the source (accent-insensitive)', () => {
    expect(spellByName('Visão Mística')).toBe(srcSpellByName('Visão Mística'))
    expect(spellByName('visao mistica')).toBe(srcSpellByName('visao mistica'))
    expect(spellByName('not a spell')).toBeNull()
  })

  it('spellEffectByName mirrors the source', () => {
    const name = SPELL_CATALOG[anyId].name
    expect(spellEffectByName(name)).toBe(srcSpellEffectByName(name))
    expect(spellEffectByName('not a spell')).toBeNull()
  })

  it('buffSpells returns every spell with a buff block', () => {
    const expected = Object.values(SPELL_CATALOG).filter((s) => s.buff)
    expect(buffSpells()).toHaveLength(expected.length)
    expect(buffSpells().every((s) => s.buff)).toBe(true)
  })
})
