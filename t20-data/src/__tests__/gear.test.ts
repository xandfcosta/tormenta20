import { describe, expect, it } from 'vitest'
import { GEAR } from '../items/catalog/gear'
import { CATALOG_ITEMS, getCatalogItem } from '../items/catalog'

/**
 * Mundane gear catalog — PDF book Cap 3 (p155-162). Only the items
 * that were MISSING from existing catalogs; pre-existing entries
 * (luneta, simbolo-sagrado, mochila-aventureiro, maleta-medicamentos,
 * bandoleira-pocoes, coleccao-de-livros, ácido, bálsamo restaurador,
 * fogo alquímico, etc.) are NOT redefined here.
 *
 * Pinned:
 *  - Adventuring gear / ferramentas → apparel
 *  - Água benta + óleo → consumable
 *  - Rações / refeição comum → meal
 *  - Slots 0.5 for itens leves; 0 for mochila básica (não ocupa espaço)
 *  - Specific items: Corda T$ 1, Tocha T$ 1, Lampião T$ 7, Saco de
 *    Dormir T$ 1, Gazua T$ 5, Pé de Cabra T$ 2, Sela T$ 20,
 *    Instrumentos de Ofício T$ 30
 */

describe('GEAR — shape & invariants', () => {
  it('has at least 20 new entries', () => {
    expect(GEAR.length).toBeGreaterThanOrEqual(20)
  })

  it('all ids unique within GEAR', () => {
    const ids = GEAR.map((g) => g.id)
    expect(new Set(ids).size).toBe(ids.length)
  })

  it('all ids unique across the full catalog (no collisions)', () => {
    const allIds = CATALOG_ITEMS.map((it) => it.id)
    expect(new Set(allIds).size).toBe(allIds.length)
  })

  it('every entry has non-negative price and slots', () => {
    for (const g of GEAR) {
      expect(g.price).toBeGreaterThanOrEqual(0)
      expect(g.slots).toBeGreaterThanOrEqual(0)
    }
  })

  it('every entry has empty `modifiers` (contextual effects in description)', () => {
    for (const g of GEAR) {
      expect(g.modifiers).toEqual([])
    }
  })
})

describe('GEAR — category constraints', () => {
  it('GEAR uses only apparel / consumable / meal categories', () => {
    for (const g of GEAR) {
      expect(['apparel', 'consumable', 'meal']).toContain(g.category)
    }
  })

  it('every consumable + meal entry carries a consumable spec', () => {
    for (const g of GEAR) {
      if (g.category === 'consumable' || g.category === 'meal') {
        expect(g.consumable).toBeDefined()
      }
    }
  })

  it('apparel entries have no consumable spec', () => {
    for (const g of GEAR) {
      if (g.category === 'apparel') {
        expect(g.consumable).toBeUndefined()
      }
    }
  })
})

describe('GEAR — pinned canonical entries (book p155-157)', () => {
  it('Mochila básica: T$ 2, 0 slots (não ocupa espaço, p141)', () => {
    const m = getCatalogItem('mochila')!
    expect(m.price).toBe(2)
    expect(m.slots).toBe(0)
  })

  it('Tocha: T$ 1, 1 slot', () => {
    const t = getCatalogItem('tocha')!
    expect(t.price).toBe(1)
    expect(t.slots).toBe(1)
  })

  it('Corda: T$ 1, 1 slot', () => {
    expect(getCatalogItem('corda')?.price).toBe(1)
  })

  it('Lampião: T$ 7, 1 slot', () => {
    expect(getCatalogItem('lampiao')?.price).toBe(7)
  })

  it('Saco de Dormir: T$ 1, 1 slot', () => {
    expect(getCatalogItem('saco-de-dormir')?.price).toBe(1)
  })

  it('Gazua: T$ 5, 1 slot', () => {
    expect(getCatalogItem('gazua')?.price).toBe(5)
  })

  it('Pé de Cabra: T$ 2, 1 slot', () => {
    expect(getCatalogItem('pe-de-cabra')?.price).toBe(2)
  })

  it('Sela: T$ 20, 1 slot', () => {
    expect(getCatalogItem('sela')?.price).toBe(20)
  })

  it('Instrumentos de Ofício: T$ 30, 1 slot', () => {
    expect(getCatalogItem('instrumentos-de-oficio')?.price).toBe(30)
  })

  it('Estojo de Disfarces: T$ 50, 1 slot', () => {
    expect(getCatalogItem('estojo-de-disfarces')?.price).toBe(50)
  })

  it('Algemas: T$ 15, 1 slot', () => {
    expect(getCatalogItem('algemas')?.price).toBe(15)
  })

  it('Arpéu: T$ 5, 1 slot', () => {
    expect(getCatalogItem('arpeu')?.price).toBe(5)
  })
})

describe('GEAR — consumáveis novos (água benta + óleo)', () => {
  it('Água benta: T$ 10, 0.5 slot, consumable instant', () => {
    const a = getCatalogItem('agua-benta')!
    expect(a.category).toBe('consumable')
    expect(a.price).toBe(10)
    expect(a.slots).toBe(0.5)
    expect(a.consumable?.scope).toBe('instant')
  })

  it('Óleo: T$ 1, 0.5 slot, consumable instant', () => {
    const o = getCatalogItem('oleo')!
    expect(o.price).toBe(1)
    expect(o.slots).toBe(0.5)
    expect(o.consumable?.scope).toBe('instant')
  })
})

describe('GEAR — rações (meal category)', () => {
  it('Ração de Viagem: T$ 1, 0.5 slot, meal', () => {
    const r = getCatalogItem('racao-de-viagem')!
    expect(r.category).toBe('meal')
    expect(r.price).toBe(1)
    expect(r.slots).toBe(0.5)
  })

  it('Refeição Comum: T$ 1, 0.5 slot, meal', () => {
    const r = getCatalogItem('refeicao-comum')!
    expect(r.category).toBe('meal')
    expect(r.slots).toBe(0.5)
  })
})

describe('GEAR — slot distribution', () => {
  it('consumable + meal entries all 0.5 slots (regra item leve p141)', () => {
    for (const g of GEAR) {
      if (g.category === 'consumable' || g.category === 'meal') {
        expect(g.slots).toBe(0.5)
      }
    }
  })

  it('mochila básica is the only entry with slots: 0', () => {
    const zero = GEAR.filter((g) => g.slots === 0).map((g) => g.id)
    expect(zero).toEqual(['mochila'])
  })
})
