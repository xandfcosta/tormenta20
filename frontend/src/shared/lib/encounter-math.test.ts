import { describe, expect, it } from 'vitest'
import { computeGroupNd, xpForNd } from './encounter-math'

describe('computeGroupNd', () => {
  it('multiplica quando o ND é fracionário (p282)', () => {
    // Quatro de ND 1/4 valem ND 1; dois de ND 1/2 também.
    expect(computeGroupNd(0.25, 4)).toBe(1)
    expect(computeGroupNd(0.5, 2)).toBe(1)
  })

  it('soma 2 a cada dobra quando o ND é inteiro (p282)', () => {
    expect(computeGroupNd(1, 2)).toBe(3)
    expect(computeGroupNd(5, 4)).toBe(9)
  })

  it('um monstro sozinho vale o próprio ND', () => {
    expect(computeGroupNd(7, 1)).toBe(7)
    expect(computeGroupNd(0.25, 1)).toBe(0.25)
  })

  it('cai entre as dobras num grupo de 3', () => {
    const three = computeGroupNd(1, 3)

    expect(three).toBeGreaterThan(computeGroupNd(1, 2))
    expect(three).toBeLessThan(computeGroupNd(1, 4))
  })

  it('grupo vazio não vale ND nenhum', () => {
    expect(computeGroupNd(5, 0)).toBe(0)
    expect(computeGroupNd(5, -1)).toBe(0)
  })
})

describe('xpForNd', () => {
  it('mil por ponto de ND (p326)', () => {
    expect(xpForNd(1)).toBe(1000)
    expect(xpForNd(0.25)).toBe(250)
  })
})
