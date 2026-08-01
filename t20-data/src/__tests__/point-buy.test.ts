import { describe, expect, it } from 'vitest'
import {
  POINT_BUY_BUDGET,
  pointBuyCost,
  pointBuySpent,
  pointBuyWarnings,
} from '../point-buy'

const spread = (over: Partial<Record<string, number>>) => ({
  strength: 0,
  dexterity: 0,
  constitution: 0,
  intelligence: 0,
  wisdom: 0,
  charisma: 0,
  ...over,
})

describe('pointBuyCost — Tabela 1-1 (p17)', () => {
  it('custos pinados do livro', () => {
    expect(pointBuyCost(-1)).toBe(-1)
    expect(pointBuyCost(0)).toBe(0)
    expect(pointBuyCost(1)).toBe(1)
    expect(pointBuyCost(2)).toBe(2)
    expect(pointBuyCost(3)).toBe(4)
    expect(pointBuyCost(4)).toBe(7)
  })

  it('fora do intervalo lança com o valor ofensor', () => {
    expect(() => pointBuyCost(5)).toThrow(/got 5/)
    expect(() => pointBuyCost(-2)).toThrow(/got -2/)
  })
})

describe('pointBuySpent + pointBuyWarnings', () => {
  it('spread do exemplo: 4+2+2+1+1+0 = 10 pontos, legal', () => {
    const attrs = spread({
      strength: 3, // 4
      dexterity: 2, // 2
      constitution: 2, // 2
      intelligence: 1, // 1
      wisdom: 1, // 1
    })
    expect(pointBuySpent(attrs)).toBe(POINT_BUY_BUDGET)
    expect(pointBuyWarnings(attrs)).toEqual([])
  })

  it('−1 devolve 1 ponto', () => {
    expect(pointBuySpent(spread({ charisma: -1, strength: 4, dexterity: 2, constitution: 2 }))).toBe(10)
  })

  it('acima do limite avisa', () => {
    const w = pointBuyWarnings(spread({ strength: 4, dexterity: 4 }))
    expect(w.some((x) => x.includes('excedem'))).toBe(true)
  })

  it('dois atributos em −1 avisa (só UM pode, p17)', () => {
    const w = pointBuyWarnings(spread({ charisma: -1, wisdom: -1 }))
    expect(w.some((x) => x.includes('UM atributo'))).toBe(true)
  })

  it('valor fora do intervalo avisa sem lançar', () => {
    const w = pointBuyWarnings(spread({ strength: 5 }))
    expect(w.some((x) => x.includes('fora do intervalo'))).toBe(true)
  })
})
