import { describe, expect, it } from 'vitest'
import {
  CONSUMABLE_MAGIC_ITEMS,
  consumableById,
  consumablesByCircle,
  consumablesByKind,
  consumablesBySpell,
  tierForCircle,
} from '../consumable-magic-items'

/**
 * PDF Cap 6 (Tesouro), Tabela 8-12, p341. Pinned:
 *  - Tier mapping: menor=1-2°, médio=3-4°, maior=5°.
 *  - Prices fixed per row of Tabela 8-12 — NOT a formula:
 *      menor T$ 30; médio aprimorado T$ 120-270; maior T$ 750-3000.
 *  - Poções e pergaminhos compartilham a mesma tabela.
 *  - Activation: poção/óleo/granada = ação padrão; pergaminho = ação
 *    padrão OU ação da magia (o maior). Exige conhecer a magia OU
 *    Misticismo CD 20 + custo PM.
 *
 * Subset: 37 entries (21 poções + 16 pergaminhos).
 */

describe('CONSUMABLE_MAGIC_ITEMS — shape & invariants', () => {
  it('catalog has exactly 37 entries', () => {
    expect(CONSUMABLE_MAGIC_ITEMS.length).toBe(37)
  })

  it('all ids unique', () => {
    const ids = CONSUMABLE_MAGIC_ITEMS.map((i) => i.id)
    expect(new Set(ids).size).toBe(ids.length)
  })

  it('all names unique', () => {
    const names = CONSUMABLE_MAGIC_ITEMS.map((i) => i.name)
    expect(new Set(names).size).toBe(names.length)
  })

  it('every entry has kind pocao|pergaminho', () => {
    for (const i of CONSUMABLE_MAGIC_ITEMS) {
      expect(['pocao', 'pergaminho']).toContain(i.kind)
    }
  })

  it('every spellCircle is 1..5', () => {
    for (const i of CONSUMABLE_MAGIC_ITEMS) {
      expect(i.spellCircle).toBeGreaterThanOrEqual(1)
      expect(i.spellCircle).toBeLessThanOrEqual(5)
    }
  })

  it('every bookPage === 341', () => {
    for (const i of CONSUMABLE_MAGIC_ITEMS) {
      expect(i.bookPage).toBe(341)
    }
  })

  it('catalog is frozen', () => {
    expect(Object.isFrozen(CONSUMABLE_MAGIC_ITEMS)).toBe(true)
  })
})

describe('CONSUMABLE_MAGIC_ITEMS — kind distribution', () => {
  it('21 poções + 16 pergaminhos = 37', () => {
    expect(consumablesByKind('pocao').length).toBe(21)
    expect(consumablesByKind('pergaminho').length).toBe(16)
  })

  it('consumablesByKind filtra corretamente', () => {
    for (const p of consumablesByKind('pocao')) {
      expect(p.kind).toBe('pocao')
    }
    for (const p of consumablesByKind('pergaminho')) {
      expect(p.kind).toBe('pergaminho')
    }
  })
})

describe('CONSUMABLE_MAGIC_ITEMS — pricing per Tabela 8-12', () => {
  it('preço mínimo é T$ 30 (entrada base menor)', () => {
    const cheapest = CONSUMABLE_MAGIC_ITEMS.reduce(
      (m, i) => Math.min(m, i.priceTibar),
      Infinity,
    )
    expect(cheapest).toBe(30)
  })

  it('preço máximo é T$ 3000 (Curar Ferimentos suprema)', () => {
    const max = CONSUMABLE_MAGIC_ITEMS.reduce(
      (m, i) => Math.max(m, i.priceTibar),
      0,
    )
    expect(max).toBe(3000)
  })

  it('preços válidos: 30 / 120 / 270 / 750 / 1080 / 3000', () => {
    const valid = new Set([30, 120, 270, 750, 1080, 3000])
    for (const i of CONSUMABLE_MAGIC_ITEMS) {
      expect(valid.has(i.priceTibar)).toBe(true)
    }
  })

  it('toda entrada de círculo 1 em T$ 30 é tier menor', () => {
    for (const i of CONSUMABLE_MAGIC_ITEMS) {
      if (i.spellCircle === 1 && i.priceTibar === 30) {
        expect(i.tier).toBe('menor')
      }
    }
  })
})

describe('tierForCircle — mapping helper', () => {
  it('círculo 1 e 2 → menor', () => {
    expect(tierForCircle(1)).toBe('menor')
    expect(tierForCircle(2)).toBe('menor')
  })

  it('círculo 3 e 4 → médio', () => {
    expect(tierForCircle(3)).toBe('medio')
    expect(tierForCircle(4)).toBe('medio')
  })

  it('círculo 5 → maior', () => {
    expect(tierForCircle(5)).toBe('maior')
  })
})

describe('CONSUMABLE_MAGIC_ITEMS — pinned canonical entries', () => {
  it('Poção de Curar Ferimentos menor: 2d8+2 PV, T$ 30', () => {
    const i = consumableById('pocao-de-curar-ferimentos-menor')!
    expect(i.kind).toBe('pocao')
    expect(i.spell).toBe('Curar Ferimentos')
    expect(i.priceTibar).toBe(30)
    expect(i.effect).toMatch(/2d8\+2/)
  })

  it('Poção de Curar Ferimentos suprema: 11d8+11 PV, T$ 3000, círculo 5', () => {
    const i = consumableById('pocao-de-curar-ferimentos-suprema')!
    expect(i.spellCircle).toBe(5)
    expect(i.tier).toBe('maior')
    expect(i.priceTibar).toBe(3000)
    expect(i.effect).toMatch(/11d8\+11/)
  })

  it('Poção de Bola de Fogo é granada (arremesso)', () => {
    const i = consumableById('pocao-de-bola-de-fogo')!
    expect(i.name).toMatch(/granada/)
    expect(i.spell).toBe('Bola de Fogo')
  })

  it('Poção de Velocidade: círculo 3, médio, T$ 270', () => {
    const i = consumableById('pocao-de-velocidade')!
    expect(i.spellCircle).toBe(3)
    expect(i.tier).toBe('medio')
    expect(i.priceTibar).toBe(270)
  })

  it('Pergaminho de Curar Ferimentos: kind pergaminho, círculo 1', () => {
    const i = consumableById('pergaminho-de-curar-ferimentos-menor')!
    expect(i.kind).toBe('pergaminho')
    expect(i.spellCircle).toBe(1)
    expect(i.priceTibar).toBe(30)
  })

  it('Pergaminho de Arma Mágica aprimorada: T$ 750', () => {
    const i = consumableById('pergaminho-de-arma-magica-aprimorada')!
    expect(i.priceTibar).toBe(750)
    expect(i.effect).toMatch(/bônus \+3/)
  })

  it('Pergaminho de Compreensão: lê qualquer idioma', () => {
    const i = consumableById('pergaminho-de-compreensao')!
    expect(i.effect).toMatch(/idioma/)
  })
})

describe('CONSUMABLE_MAGIC_ITEMS — lookups', () => {
  it('consumableById hit', () => {
    expect(consumableById('pocao-de-velocidade')?.name).toMatch(/Velocidade/)
  })

  it('consumableById miss', () => {
    expect(consumableById('pocao-de-marshmallow')).toBeUndefined()
  })

  it('consumablesByCircle filtra por círculo', () => {
    for (const i of consumablesByCircle(5)) {
      expect(i.spellCircle).toBe(5)
    }
  })

  it('consumablesBySpell agrupa por magia (Curar Ferimentos: 4 poções + 1 pergaminho)', () => {
    const curas = consumablesBySpell('Curar Ferimentos')
    expect(curas.length).toBe(5)
    expect(curas.filter((c) => c.kind === 'pocao').length).toBe(4)
    expect(curas.filter((c) => c.kind === 'pergaminho').length).toBe(1)
  })

  it('consumablesBySpell retorna [] para magia desconhecida', () => {
    expect(consumablesBySpell('Telepatia Universal')).toEqual([])
  })
})
