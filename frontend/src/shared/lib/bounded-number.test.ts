import { describe, expect, it } from 'vitest'
import { clampToRange, rangeSchema } from './bounded-number'

const LEVEL = { min: 1, max: 20 }
const ND = { min: 0, max: 20, step: 0.25 }

describe('clampToRange', () => {
  it('passes through an in-range, on-step value untouched', () => {
    expect(clampToRange(7, LEVEL)).toBe(7)
    expect(clampToRange(0.5, ND)).toBe(0.5)
  })

  it('clamps below min and above max', () => {
    expect(clampToRange(0, LEVEL)).toBe(1)
    expect(clampToRange(99, LEVEL)).toBe(20)
    expect(clampToRange(-4, ND)).toBe(0)
    expect(clampToRange(50, ND)).toBe(20)
  })

  it('snaps an off-step value to the nearest increment', () => {
    expect(clampToRange(0.3, ND)).toBe(0.25)
    expect(clampToRange(0.4, ND)).toBe(0.5)
    expect(clampToRange(3.5, LEVEL)).toBe(4)
  })

  it('falls back to min for NaN / non-finite input', () => {
    expect(clampToRange(Number.NaN, LEVEL)).toBe(1)
    expect(clampToRange(Number.POSITIVE_INFINITY, ND)).toBe(20)
  })

  it('always yields a value the range schema accepts', () => {
    const schema = rangeSchema(ND)
    for (const v of [-100, -0.1, 0.13, 5.5, 19.99, 1000, Number.NaN]) {
      expect(schema.safeParse(clampToRange(v, ND)).success).toBe(true)
    }
  })
})
