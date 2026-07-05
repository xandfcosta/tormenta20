import { describe, expect, it } from 'vitest'
import {
  COMMUNITY_TABLE,
  COMMUNITY_TIERS,
  communityRow,
  minCommunityForItemPrice,
} from '../community-types'

/**
 * PDF Cap 6 p271-273 — "Tipos de Comunidades". 4 patamares em ordem
 * crescente de tamanho + poder.
 */

describe('COMMUNITY_TIERS (p271-273)', () => {
  it('lists all 4 tiers in book order (small → large)', () => {
    expect(COMMUNITY_TIERS).toEqual(['aldeia', 'vila', 'cidade', 'metropole'])
  })

  it('table matches enum 1:1', () => {
    expect(COMMUNITY_TABLE.map((r) => r.tier)).toEqual([...COMMUNITY_TIERS])
  })
})

describe('COMMUNITY_TABLE population caps (p271-272)', () => {
  it('aldeia: até 1.000 habitantes', () => {
    expect(communityRow('aldeia').maxPopulation).toBe(1000)
  })

  it('vila: até 5.000 habitantes', () => {
    expect(communityRow('vila').maxPopulation).toBe(5000)
  })

  it('cidade: até 25.000 habitantes', () => {
    expect(communityRow('cidade').maxPopulation).toBe(25000)
  })

  it('metropole: sem teto (~100k baseline, algumas > 1M)', () => {
    expect(communityRow('metropole').maxPopulation).toBeNull()
  })

  it('populations are strictly increasing (ignoring metropole=null)', () => {
    const caps = COMMUNITY_TABLE.filter((r) => r.maxPopulation !== null).map(
      (r) => r.maxPopulation as number,
    )
    for (let i = 1; i < caps.length; i++) {
      expect(caps[i]).toBeGreaterThan(caps[i - 1]!)
    }
  })
})

describe('COMMUNITY_TABLE guard force (p271-272)', () => {
  it('aldeia: sem guarda formal (camponeses 2d10 improvisam)', () => {
    const g = communityRow('aldeia').guardForce
    expect(g.kind).toBe('nenhuma')
    expect(g.dice).toBe('2d10')
  })

  it('aldeia: magistrado tem 1d4+1 guardas', () => {
    expect(communityRow('aldeia').magistrateGuards).toBe('1d4+1')
  })

  it('vila: milícia 10d10 comandada por sargento', () => {
    const g = communityRow('vila').guardForce
    expect(g.kind).toBe('milicia')
    expect(g.dice).toBe('10d10')
    expect(g.leaderLabel).toBe('sargento')
  })

  it('cidade: força formal liderada por capitão L8+', () => {
    const g = communityRow('cidade').guardForce
    expect(g.kind).toBe('formal')
    expect(g.leaderLabel).toContain('8')
  })

  it('metropole: exército', () => {
    expect(communityRow('metropole').guardForce.kind).toBe('exercito')
  })
})

describe('COMMUNITY_TABLE economia (p271-272)', () => {
  it('aldeia: itemPriceCapTS = 0 (só armazém — nada além do essencial)', () => {
    expect(communityRow('aldeia').itemPriceCapTS).toBe(0)
    expect(communityRow('aldeia').cashOnHand).toBeNull()
  })

  it('vila: itens até T$ 1.000, cofre 1d6 × T$ 1.000', () => {
    const c = communityRow('vila')
    expect(c.itemPriceCapTS).toBe(1000)
    expect(c.cashOnHand).toEqual({ dice: '1d6', multiplierTS: 1000 })
  })

  it('cidade: itens até T$ 10.000, cofre 2d4 × T$ 10.000', () => {
    const c = communityRow('cidade')
    expect(c.itemPriceCapTS).toBe(10000)
    expect(c.cashOnHand).toEqual({ dice: '2d4', multiplierTS: 10000 })
  })

  it('metropole: teto e cofre nulos (virtualmente ilimitado)', () => {
    const c = communityRow('metropole')
    expect(c.itemPriceCapTS).toBeNull()
    expect(c.cashOnHand).toBeNull()
  })
})

describe('minCommunityForItemPrice (p271-272)', () => {
  it('item T$ 500 → vila (primeira com teto ≥ 500)', () => {
    expect(minCommunityForItemPrice(500)).toBe('vila')
  })

  it('item T$ 1.000 → vila (exatamente no teto)', () => {
    expect(minCommunityForItemPrice(1000)).toBe('vila')
  })

  it('item T$ 1.001 → cidade (excede teto da vila)', () => {
    expect(minCommunityForItemPrice(1001)).toBe('cidade')
  })

  it('item T$ 10.000 → cidade (exatamente no teto)', () => {
    expect(minCommunityForItemPrice(10000)).toBe('cidade')
  })

  it('item T$ 50.000 → metropole (excede teto da cidade)', () => {
    expect(minCommunityForItemPrice(50000)).toBe('metropole')
  })

  it('rejects negative price', () => {
    expect(() => minCommunityForItemPrice(-1)).toThrow(/priceTS/)
  })
})

describe('COMMUNITY_TABLE law formality (p271-273)', () => {
  it('aldeia: nenhuma (senso comum / dogmas)', () => {
    expect(communityRow('aldeia').lawFormality).toBe('nenhuma')
  })

  it('vila: simples (milícia, sargento, multa/pelourinho)', () => {
    expect(communityRow('vila').lawFormality).toBe('simples')
  })

  it('cidade e metropole: complexa (juízes, promotores, guildas)', () => {
    expect(communityRow('cidade').lawFormality).toBe('complexa')
    expect(communityRow('metropole').lawFormality).toBe('complexa')
  })
})
