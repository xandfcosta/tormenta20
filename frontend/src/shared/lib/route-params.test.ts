import { describe, expect, it } from 'vitest'
import { idParamSchema, idParams } from './route-params'

describe('idParamSchema', () => {
  it('coerces a numeric string to a positive integer', () => {
    expect(idParamSchema.parse('42')).toBe(42)
  })

  it('rejects non-numeric, zero, negative, and fractional ids', () => {
    for (const bad of ['abc', '0', '-1', '1.5', '']) {
      expect(idParamSchema.safeParse(bad).success).toBe(false)
    }
  })
})

describe('idParams', () => {
  it('parse coerces every named key to a number', () => {
    const { parse } = idParams('id', 'sid')
    expect(parse({ id: '3', sid: '7' })).toEqual({ id: 3, sid: 7 })
  })

  it('stringify turns the numbers back into strings for URL building', () => {
    const { stringify } = idParams('id', 'sid')
    expect(stringify({ id: 3, sid: 7 })).toEqual({ id: '3', sid: '7' })
  })

  it('parse throws when any key is malformed', () => {
    const { parse } = idParams('id')
    expect(() => parse({ id: 'nope' })).toThrow()
  })
})
