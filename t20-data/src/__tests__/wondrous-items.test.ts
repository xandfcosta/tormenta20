import { describe, expect, it } from 'vitest'
import {
  WONDROUS_ITEMS,
  wondrousItemById,
  wondrousItemsBySlot,
  wondrousItemsByTier,
  type WondrousSlot,
} from '../wondrous-items'

/**
 * PDF Cap 6 (Tesouro):
 *  - Tabela 8-13 (menores, p342): T$ 3.000-9.000
 *  - Tabela 8-14 (médios, p343):  T$ 10.500-25.500
 *  - Tabela 8-15 (maiores, p343): T$ 30.000-150.000
 *
 * Subset: 38 entries (16 menores / 14 médios / 8 maiores).
 *
 * T20 has NO formal slot system — `slot` is a UI hint. Stacking rule
 * (p333) is "identical bonuses do not stack; use the highest" — not
 * encoded here, owned by the engine.
 */

const ALL_SLOTS: readonly WondrousSlot[] = [
  'anel',
  'cabeca',
  'pescoco',
  'corpo',
  'cintura',
  'maos',
  'pes',
  'outro',
]

describe('WONDROUS_ITEMS — shape & invariants', () => {
  it('catalog has exactly 38 entries', () => {
    expect(WONDROUS_ITEMS.length).toBe(38)
  })

  it('all ids unique', () => {
    const ids = WONDROUS_ITEMS.map((i) => i.id)
    expect(new Set(ids).size).toBe(ids.length)
  })

  it('all names unique', () => {
    const names = WONDROUS_ITEMS.map((i) => i.name)
    expect(new Set(names).size).toBe(names.length)
  })

  it('every entry has requiresAttunement === false (sem attunement T20)', () => {
    for (const i of WONDROUS_ITEMS) {
      expect(i.requiresAttunement).toBe(false)
    }
  })

  it('every slot is in the WondrousSlot union', () => {
    for (const i of WONDROUS_ITEMS) {
      expect(ALL_SLOTS).toContain(i.slot)
    }
  })

  it('every bookPage in 342-345', () => {
    for (const i of WONDROUS_ITEMS) {
      expect(i.bookPage).toBeGreaterThanOrEqual(342)
      expect(i.bookPage).toBeLessThanOrEqual(345)
    }
  })

  it('catalog is frozen', () => {
    expect(Object.isFrozen(WONDROUS_ITEMS)).toBe(true)
  })
})

describe('WONDROUS_ITEMS — tier distribution + pricing bands', () => {
  it('16 menores / 14 médios / 8 maiores', () => {
    expect(wondrousItemsByTier('menor').length).toBe(16)
    expect(wondrousItemsByTier('medio').length).toBe(14)
    expect(wondrousItemsByTier('maior').length).toBe(8)
  })

  it('menores price band T$ 3000-9000', () => {
    for (const i of wondrousItemsByTier('menor')) {
      expect(i.priceTibar).toBeGreaterThanOrEqual(3000)
      expect(i.priceTibar).toBeLessThanOrEqual(9000)
    }
  })

  it('médios price band T$ 10500-25500', () => {
    for (const i of wondrousItemsByTier('medio')) {
      expect(i.priceTibar).toBeGreaterThanOrEqual(10500)
      expect(i.priceTibar).toBeLessThanOrEqual(25500)
    }
  })

  it('maiores price band T$ 30000-150000', () => {
    for (const i of wondrousItemsByTier('maior')) {
      expect(i.priceTibar).toBeGreaterThanOrEqual(30000)
      expect(i.priceTibar).toBeLessThanOrEqual(150000)
    }
  })
})

describe('WONDROUS_ITEMS — pinned canonical entries', () => {
  it('Anel da Proteção: anel, menor, +2 Defesa, T$ 9000', () => {
    const i = wondrousItemById('anel-da-protecao')!
    expect(i.slot).toBe('anel')
    expect(i.tier).toBe('menor')
    expect(i.priceTibar).toBe(9000)
    expect(i.effect).toMatch(/\+2 em Defesa/)
  })

  it('Manto da Resistência: corpo, menor, +2 testes resistência', () => {
    const i = wondrousItemById('manto-da-resistencia')!
    expect(i.slot).toBe('corpo')
    expect(i.tier).toBe('menor')
    expect(i.effect).toMatch(/\+2 em testes de resistência/)
  })

  it('Botas Velozes: pés, médio, +3m + Velocidade', () => {
    const i = wondrousItemById('botas-velozes')!
    expect(i.slot).toBe('pes')
    expect(i.effect).toMatch(/\+3m/)
    expect(i.effect).toMatch(/Velocidade/)
  })

  it('Cinto da Força do Gigante: cintura, médio, +2 Força', () => {
    const i = wondrousItemById('cinto-da-forca-do-gigante')!
    expect(i.slot).toBe('cintura')
    expect(i.tier).toBe('medio')
    expect(i.effect).toMatch(/\+2 em Força/)
  })

  it('Colar Guardião: pescoço, maior, +5 Defesa, T$ 51000', () => {
    const i = wondrousItemById('colar-guardiao')!
    expect(i.slot).toBe('pescoco')
    expect(i.tier).toBe('maior')
    expect(i.priceTibar).toBe(51000)
    expect(i.effect).toMatch(/\+5 na Defesa/)
  })

  it('Anel da Regeneração: anel, maior, Cura Acelerada 5, T$ 150000', () => {
    const i = wondrousItemById('anel-da-regeneracao')!
    expect(i.slot).toBe('anel')
    expect(i.tier).toBe('maior')
    expect(i.priceTibar).toBe(150000)
    expect(i.effect).toMatch(/Cura Acelerada 5/)
  })

  it('Robe do Arquimago: requer Conjurador arcano (única class-gated)', () => {
    const i = wondrousItemById('robe-do-arquimago')!
    expect(i.requiresClass).toBe('Conjurador arcano')
    expect(i.effect).toMatch(/arcano/)
  })

  it('Medalhão de Lena: 100 PV ao ser reduzido a 0', () => {
    const i = wondrousItemById('medalhao-de-lena')!
    expect(i.effect).toMatch(/100 PV/)
    expect(i.effect).toMatch(/uma vez por dia/)
  })

  it('Tapete Voador: voo 12m, 3m x 3m, 6 criaturas Médias', () => {
    const i = wondrousItemById('tapete-voador')!
    expect(i.effect).toMatch(/voo 12m/)
    expect(i.effect).toMatch(/3m x 3m/)
  })
})

describe('WONDROUS_ITEMS — class restriction', () => {
  it('apenas Robe do Arquimago tem requiresClass não-null', () => {
    const restricted = WONDROUS_ITEMS.filter((i) => i.requiresClass !== null)
    expect(restricted.length).toBe(1)
    expect(restricted[0]!.id).toBe('robe-do-arquimago')
  })
})

describe('WONDROUS_ITEMS — slot distribution', () => {
  it('cada slot da union aparece em ao menos uma entrada', () => {
    for (const slot of ALL_SLOTS) {
      const count = wondrousItemsBySlot(slot).length
      expect(count, `slot ${slot} sem entradas`).toBeGreaterThan(0)
    }
  })

  it('wondrousItemsBySlot("anel") retorna apenas anéis', () => {
    for (const i of wondrousItemsBySlot('anel')) {
      expect(i.slot).toBe('anel')
    }
  })

  it('slot "anel" tem ≥ 8 entradas (anéis dominam o catálogo)', () => {
    expect(wondrousItemsBySlot('anel').length).toBeGreaterThanOrEqual(8)
  })
})

describe('WONDROUS_ITEMS — lookup helpers', () => {
  it('wondrousItemById hit', () => {
    expect(wondrousItemById('botas-velozes')?.name).toBe('Botas Velozes')
  })

  it('wondrousItemById miss', () => {
    expect(wondrousItemById('chapeu-do-bobo')).toBeUndefined()
  })

  it('wondrousItemsByTier soma a 38 entradas', () => {
    let total = 0
    for (const tier of ['menor', 'medio', 'maior', 'artefato'] as const) {
      total += wondrousItemsByTier(tier).length
    }
    expect(total).toBe(38)
  })
})
