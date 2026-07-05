import { describe, expect, it } from 'vitest'
import {
  COST_OF_LIVING_TABLE,
  COST_OF_LIVING_TIERS,
  affordableCostOfLiving,
  costOfLivingRow,
  restConditionForCost,
} from '../cost-of-living'
import { restRecoveryAmount } from '../rest'

/**
 * PDF Cap 6 p277 — "Variante: Custo de Vida". 4 tiers × T$/mês × condição
 * de descanso resultante.
 */

describe('COST_OF_LIVING_TIERS (p277)', () => {
  it('lists 4 tiers in ascending cost', () => {
    expect(COST_OF_LIVING_TIERS).toEqual([
      'pobre',
      'medio',
      'rico',
      'luxuoso',
    ])
  })

  it('table matches enum 1:1', () => {
    expect(COST_OF_LIVING_TABLE.map((r) => r.tier)).toEqual([
      ...COST_OF_LIVING_TIERS,
    ])
  })
})

describe('costOfLivingRow (p277)', () => {
  it('pobre = T$ 10 / condição ruim', () => {
    const r = costOfLivingRow('pobre')
    expect(r.monthlyTS).toBe(10)
    expect(r.restCondition).toBe('ruim')
  })

  it('medio = T$ 50 / normal', () => {
    const r = costOfLivingRow('medio')
    expect(r.monthlyTS).toBe(50)
    expect(r.restCondition).toBe('normal')
  })

  it('rico = T$ 100 / confortável', () => {
    const r = costOfLivingRow('rico')
    expect(r.monthlyTS).toBe(100)
    expect(r.restCondition).toBe('confortavel')
  })

  it('luxuoso = T$ 200 / luxuosa', () => {
    const r = costOfLivingRow('luxuoso')
    expect(r.monthlyTS).toBe(200)
    expect(r.restCondition).toBe('luxuosa')
  })

  it('costs monotonically increase', () => {
    const costs = COST_OF_LIVING_TABLE.map((r) => r.monthlyTS)
    for (let i = 1; i < costs.length; i++) {
      expect(costs[i]).toBeGreaterThan(costs[i - 1]!)
    }
  })
})

describe('affordableCostOfLiving (p277)', () => {
  it('T$ 5 → null (below pobre)', () => {
    expect(affordableCostOfLiving(5)).toBeNull()
  })

  it('T$ 10 → pobre (exactly)', () => {
    expect(affordableCostOfLiving(10)).toBe('pobre')
  })

  it('T$ 49 → pobre (below medio)', () => {
    expect(affordableCostOfLiving(49)).toBe('pobre')
  })

  it('T$ 50 → medio', () => {
    expect(affordableCostOfLiving(50)).toBe('medio')
  })

  it('T$ 199 → rico', () => {
    expect(affordableCostOfLiving(199)).toBe('rico')
  })

  it('T$ 200 → luxuoso', () => {
    expect(affordableCostOfLiving(200)).toBe('luxuoso')
  })

  it('T$ 10.000 → luxuoso (highest tier, no upper cap)', () => {
    expect(affordableCostOfLiving(10000)).toBe('luxuoso')
  })

  it('rejects negative budget', () => {
    expect(() => affordableCostOfLiving(-1)).toThrow()
  })
})

describe('restConditionForCost — integração com rest.ts (p277 + p106)', () => {
  it('pobre → condição ruim → ½ nível recuperado', () => {
    const cond = restConditionForCost('pobre')
    expect(cond).toBe('ruim')
    expect(restRecoveryAmount(10, cond)).toBe(5)
  })

  it('medio → normal → nível cheio', () => {
    const cond = restConditionForCost('medio')
    expect(cond).toBe('normal')
    expect(restRecoveryAmount(10, cond)).toBe(10)
  })

  it('luxuoso → luxuosa → 3× nível', () => {
    const cond = restConditionForCost('luxuoso')
    expect(cond).toBe('luxuosa')
    expect(restRecoveryAmount(10, cond)).toBe(30)
  })
})
