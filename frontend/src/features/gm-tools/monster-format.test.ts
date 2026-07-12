import { describe, expect, it } from 'vitest'
import { formatNd, normalizeMonsterName } from './monster-format'

describe('formatNd', () => {
  it('renders sub-1 NDs as book fractions', () => {
    expect(formatNd(0.25)).toBe('1/4')
    expect(formatNd(0.5)).toBe('1/2')
  })

  it('renders whole and higher NDs as the number', () => {
    expect(formatNd(0)).toBe('0')
    expect(formatNd(1)).toBe('1')
    expect(formatNd(12)).toBe('12')
  })
})

describe('normalizeMonsterName', () => {
  it('strips accents and lowercases for search', () => {
    expect(normalizeMonsterName('Espírito')).toBe('espirito')
    expect(normalizeMonsterName('  GOBLIN  ')).toBe('goblin')
  })

  it('makes accented queries match accented names', () => {
    expect(normalizeMonsterName('Aparição').includes(normalizeMonsterName('apari'))).toBe(true)
  })
})
