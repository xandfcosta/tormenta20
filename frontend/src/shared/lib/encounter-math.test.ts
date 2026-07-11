import { describe, expect, it } from 'vitest'
import { computeGroupNd } from './encounter-math'

/**
 * Book Cap 7 p282 examples — these are the canonical illustrations
 * the rules text gives; pinning them here catches drift.
 */

describe('computeGroupNd', () => {
  it('zero quantity → 0', () => {
    expect(computeGroupNd(3, 0)).toBe(0)
  })

  it('quantity 1 = the monster ND unchanged', () => {
    expect(computeGroupNd(3, 1)).toBe(3)
    expect(computeGroupNd(0.5, 1)).toBe(0.5)
  })

  it('ND < 1: four ND 1/4 = ND 1 (book example)', () => {
    expect(computeGroupNd(0.25, 4)).toBe(1)
  })

  it('ND < 1: two ND 1/2 = ND 1 (book example)', () => {
    expect(computeGroupNd(0.5, 2)).toBe(1)
  })

  it('ND ≥ 1: two ND 1 = ND 3 (book example)', () => {
    expect(computeGroupNd(1, 2)).toBe(3)
  })

  it('ND ≥ 1: four ND 5 = ND 9 (book example)', () => {
    expect(computeGroupNd(5, 4)).toBe(9)
  })

  it('doubling adds +2 (book rule)', () => {
    expect(computeGroupNd(2, 2)).toBe(4)
    expect(computeGroupNd(2, 4)).toBe(6)
    expect(computeGroupNd(2, 8)).toBe(8)
  })

  it('non-doubling quantity interpolates between doublings', () => {
    /* 3 = 2^1.585; ND 2 + 2×1.585 ≈ 5.17 */
    const v = computeGroupNd(2, 3)
    expect(v).toBeGreaterThan(4)
    expect(v).toBeLessThan(6)
  })
})
