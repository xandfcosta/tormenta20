import { describe, expect, it } from 'vitest'
import { rollBounds, rollValueSchema } from './consume-roll'

describe('rollBounds', () => {
  it('computes [count, count*sides] for a well-formed die string', () => {
    expect(rollBounds('2d4')).toEqual({ min: 2, max: 8 })
    expect(rollBounds('1d8')).toEqual({ min: 1, max: 8 })
  })

  it('tolerates surrounding whitespace', () => {
    expect(rollBounds(' 3d6 ')).toEqual({ min: 3, max: 18 })
  })

  it('returns null for the fixed "0" and malformed strings', () => {
    expect(rollBounds('0')).toBeNull()
    expect(rollBounds('d6')).toBeNull()
    expect(rollBounds('2x4')).toBeNull()
    expect(rollBounds('0d6')).toBeNull()
  })
})

describe('rollValueSchema', () => {
  const schema = rollValueSchema({ dice: '2d4' })
  const firstError = (value: string) =>
    schema.safeParse(value).error?.issues[0]?.message

  it('requires a value', () => {
    expect(firstError('')).toBe('Informe o resultado do dado')
    expect(firstError('   ')).toBe('Informe o resultado do dado')
  })

  it('rejects non-digit input', () => {
    expect(firstError('1.5')).toBe('Valor inválido')
    expect(firstError('-3')).toBe('Valor inválido')
    expect(firstError('abc')).toBe('Valor inválido')
  })

  it('rejects a result the die cannot produce', () => {
    expect(firstError('1')).toBe('Fora do intervalo (2–8)')
    expect(firstError('9')).toBe('Fora do intervalo (2–8)')
  })

  it('accepts a valid in-range roll', () => {
    for (const v of ['2', '5', '8']) {
      expect(schema.safeParse(v).success).toBe(true)
    }
  })

  it('skips range validation when the die string is malformed', () => {
    expect(rollValueSchema({ dice: '0' }).safeParse('42').success).toBe(true)
  })
})
