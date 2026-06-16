import { describe, expect, it } from 'vitest'
import { APPAREL } from '../catalog/apparel'

/**
 * PDF Cap 3 — Vestuário, Esotéricos, Ferramentas (p152-158). Catalog
 * groups them all under category=apparel since none gate weapon/armor
 * proficiency. 37 entries covering:
 *
 *  - 21 vestuários (Andrajos, Bandana, Botas reforçadas, …, Veste de seda)
 *  - 10 esotéricos (Bolsa de pó, Cajado arcano, …, Varinha arcana)
 *  - 6  ferramentas / outros (Mochila, Símbolo sagrado, Bandoleira,
 *                              Maleta de medicamentos, Luneta, Coleção)
 *
 * Spot checks pin the canonical price + modifier for items the engine
 * surfaces in UI (perícia bonuses, deslocamento, fear resistance, etc.).
 */
describe('APPAREL catalog scope', () => {
  it('catalog contains 37 entries', () => {
    expect(APPAREL).toHaveLength(37)
  })

  it('every entry has category=apparel', () => {
    for (const a of APPAREL) expect(a.category).toBe('apparel')
  })

  it('every entry uses a valid equip slot', () => {
    // Esotéricos (cajado, varinha, etc.) ship as wielded foci; vestuário
    // is vested; mochila/símbolo as 'either'. wielded2 doesn't apply to
    // apparel (no off-hand-only catalog entry).
    for (const a of APPAREL) {
      expect(['vested', 'wielded', 'either']).toContain(a.equip)
    }
  })

  it('every id is unique', () => {
    const ids = APPAREL.map((a) => a.id)
    expect(new Set(ids).size).toBe(ids.length)
  })
})

describe('APPAREL — vestuário spot checks vs PDF p152-154', () => {
  it('Bandana: T$5, +1 Intimidação while vested', () => {
    const b = APPAREL.find((a) => a.id === 'bandana')!
    expect(b.price).toBe(5)
    const mod = b.modifiers[0]
    expect(mod.target).toEqual({ k: 'expertise', name: 'Intimidação' })
    expect(mod.amount).toBe(1)
    expect(mod.bonusType).toBe('item')
  })

  it('Botas reforçadas: T$20, +1.5m displacement while vested', () => {
    const b = APPAREL.find((a) => a.id === 'botas-reforcadas')!
    expect(b.price).toBe(20)
    const mod = b.modifiers[0]
    expect(mod.target).toEqual({ k: 'displacement' })
    expect(mod.amount).toBe(1.5)
  })

  it('Andrajos de aldeão: T$1, context-conditional Investigação bonus', () => {
    const a = APPAREL.find((x) => x.id === 'andrajos-aldeao')!
    expect(a.price).toBe(1)
    const mod = a.modifiers[0]
    expect(mod.condition?.c).toBe('context')
  })

  it('Traje de viajante: T$10 — starter gear (PDF p140)', () => {
    const t = APPAREL.find((a) => a.id === 'traje-viajante')!
    expect(t.price).toBe(10)
  })

  it('Traje da corte: T$100 — premium nobility wear', () => {
    const t = APPAREL.find((a) => a.id === 'traje-corte')!
    expect(t.price).toBe(100)
  })
})

describe('APPAREL — esotéricos spot checks vs PDF p155', () => {
  it('Cajado arcano: T$1000 — premium spell focus', () => {
    const c = APPAREL.find((a) => a.id === 'cajado-arcano')!
    expect(c.price).toBe(1000)
  })

  it('Varinha arcana exists', () => {
    expect(APPAREL.find((a) => a.id === 'varinha-arcana')).toBeDefined()
  })

  it('Símbolo sagrado exists (divine focus)', () => {
    expect(APPAREL.find((a) => a.id === 'simbolo-sagrado')).toBeDefined()
  })
})

describe('APPAREL — ferramentas spot checks vs PDF p156-158', () => {
  it('Mochila do aventureiro: starter gear', () => {
    expect(APPAREL.find((a) => a.id === 'mochila-aventureiro')).toBeDefined()
  })

  it('Bandoleira de poções: extends potion belt slots', () => {
    expect(APPAREL.find((a) => a.id === 'bandoleira-pocoes')).toBeDefined()
  })

  it('Maleta de medicamentos: Cura aid', () => {
    expect(APPAREL.find((a) => a.id === 'maleta-medicamentos')).toBeDefined()
  })

  it('Luneta: Percepção at range', () => {
    expect(APPAREL.find((a) => a.id === 'luneta')).toBeDefined()
  })
})

describe('APPAREL modifier hygiene', () => {
  it('no apparel entry carries armor-typed defense bonus', () => {
    // Armor-typed defense is exclusive to armor + shield categories;
    // an apparel entry with k=defense+bonusType=armor would silently
    // displace those (non-stacking suppression).
    for (const a of APPAREL) {
      for (const m of a.modifiers) {
        const isArmorDefense =
          m.target.k === 'defense' && m.bonusType === 'armor'
        expect(
          isArmorDefense,
          `${a.id} apparel must not carry armor-typed defense`,
        ).toBe(false)
      }
    }
  })

  it('no apparel entry occupies more than 4 slots (per p141 vested limit)', () => {
    for (const a of APPAREL) {
      expect(a.slots).toBeLessThanOrEqual(4)
    }
  })
})
