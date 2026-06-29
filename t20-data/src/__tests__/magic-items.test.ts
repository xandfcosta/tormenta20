import { describe, expect, it } from 'vitest'
import {
  MAGIC_ITEMS,
  MAGIC_ITEM_IDS,
  magicItemById,
  magicItemsByKind,
  magicItemsByTier,
  type MagicItemKind,
  type MagicItemTier,
} from '../magic-items'

/**
 * Magic items seed — PDF book Cap 8 (p329-349).
 *
 * Pins T20-specific rules:
 *  - No sintonização field exists; presence of magic items is wear/wield.
 *  - Tiers: menor / médio / maior / artefato (NOT D&D 5-tier ladder).
 *  - Artefatos lack price.
 *  - Pergaminhos have 1 charge; cajados have null charges.
 *  - Activation by PM, not daily charges.
 */

describe('MAGIC_ITEMS — shape & invariants', () => {
  it('has at least 13 entries', () => {
    expect(MAGIC_ITEM_IDS.length).toBeGreaterThanOrEqual(13)
  })

  it('all ids are unique', () => {
    expect(new Set(MAGIC_ITEM_IDS).size).toBe(MAGIC_ITEM_IDS.length)
  })

  it('every entry round-trips through magicItemById', () => {
    for (const id of MAGIC_ITEM_IDS) {
      expect(magicItemById(id).id).toBe(id)
    }
  })

  it('magicItemById throws on unknown id', () => {
    expect(() => magicItemById('not-real')).toThrow(/unknown magic item id/)
  })

  it('every entry has non-empty name + effect', () => {
    for (const id of MAGIC_ITEM_IDS) {
      const item = magicItemById(id)
      expect(item.name).toBeTruthy()
      expect(item.effect).toBeTruthy()
    }
  })

  it('every entry has a positive book page', () => {
    for (const id of MAGIC_ITEM_IDS) {
      expect(magicItemById(id).bookPage).toBeGreaterThan(0)
    }
  })

  it('pmActivationCost is non-negative', () => {
    for (const id of MAGIC_ITEM_IDS) {
      expect(magicItemById(id).pmActivationCost).toBeGreaterThanOrEqual(0)
    }
  })
})

describe('MAGIC_ITEMS — kind coverage', () => {
  const kinds: MagicItemKind[] = [
    'potion',
    'scroll',
    'staff',
    'weapon-enchant',
    'armor-enchant',
    'ring',
    'wondrous',
    'artifact',
  ]

  it.each(kinds.map((k) => [k]))(
    'kind %s is represented at least once',
    (kind) => {
      expect(magicItemsByKind(kind).length).toBeGreaterThan(0)
    },
  )
})

describe('MAGIC_ITEMS — tier coverage', () => {
  const tiers: MagicItemTier[] = ['menor', 'medio', 'maior', 'artefato']

  it.each(tiers.map((t) => [t]))(
    'tier %s is represented at least once',
    (tier) => {
      expect(magicItemsByTier(tier).length).toBeGreaterThan(0)
    },
  )
})

describe('MAGIC_ITEMS — T20-specific rules', () => {
  it('artefatos have priceTibar === null (not for sale)', () => {
    for (const item of magicItemsByTier('artefato')) {
      expect(item.priceTibar).toBeNull()
    }
  })

  it('non-artefato items always have a positive price', () => {
    for (const id of MAGIC_ITEM_IDS) {
      const item = magicItemById(id)
      if (item.tier !== 'artefato') {
        expect(item.priceTibar).not.toBeNull()
        expect(item.priceTibar as number).toBeGreaterThan(0)
      }
    }
  })

  it('scrolls have exactly 1 charge (single-use consumable)', () => {
    for (const item of magicItemsByKind('scroll')) {
      expect(item.charges).toBe(1)
    }
  })

  it('cajados have null charges (continuous, no charge system)', () => {
    for (const item of magicItemsByKind('staff')) {
      expect(item.charges).toBeNull()
    }
  })

  it('potions are tagged as consumivel + null charges (drink once)', () => {
    for (const item of magicItemsByKind('potion')) {
      expect(item.slot).toBe('consumivel')
      expect(item.charges).toBeNull()
    }
  })

  it('weapon-enchant entries occupy slot "arma"', () => {
    for (const item of magicItemsByKind('weapon-enchant')) {
      expect(item.slot).toBe('arma')
    }
  })

  it('armor-enchant entries occupy slot "armadura"', () => {
    for (const item of magicItemsByKind('armor-enchant')) {
      expect(item.slot).toBe('armadura')
    }
  })
})

describe('MAGIC_ITEMS — pinned entries (PDF integrity)', () => {
  it('Poção de Curar Ferimentos: 30 T$, círculo 1, p341', () => {
    const item = magicItemById('pocao-de-curar-ferimentos')
    expect(item.priceTibar).toBe(30)
    expect(item.castingCircle).toBe(1)
    expect(item.bookPage).toBe(341)
  })

  it('Cajado do Poder: 180000 T$, maior, p337', () => {
    const item = magicItemById('cajado-do-poder')
    expect(item.priceTibar).toBe(180000)
    expect(item.tier).toBe('maior')
    expect(item.bookPage).toBe(337)
  })

  it('Encanto Flamejante: 18000 T$ over base, weapon-enchant', () => {
    const item = magicItemById('encanto-flamejante')
    expect(item.priceTibar).toBe(18000)
    expect(item.kind).toBe('weapon-enchant')
  })

  it('Botas Aladas: pmActivationCost = 2', () => {
    expect(magicItemById('botas-aladas').pmActivationCost).toBe(2)
  })

  it('A Espada-Deus: artifact, no price, level 15 prereq, p346', () => {
    const item = magicItemById('espada-deus')
    expect(item.kind).toBe('artifact')
    expect(item.tier).toBe('artefato')
    expect(item.priceTibar).toBeNull()
    expect(item.requiresLevel).toBe(15)
    expect(item.bookPage).toBe(346)
  })

  it('Cota Élfica: maior tier, slot armadura, p340', () => {
    const item = magicItemById('cota-elfica')
    expect(item.tier).toBe('maior')
    expect(item.slot).toBe('armadura')
    expect(item.bookPage).toBe(340)
  })
})

describe('MAGIC_ITEMS immutability', () => {
  it('cannot mutate the MAGIC_ITEMS record at runtime', () => {
    expect(() => {
      // @ts-expect-error — guarded by Object.freeze, runtime should throw
      MAGIC_ITEMS['ghost'] = {} as never
    }).toThrow()
  })
})
