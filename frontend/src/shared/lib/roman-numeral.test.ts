import { describe, expect, it } from 'vitest'
import { romanNumeral } from './roman-numeral'

describe('romanNumeral', () => {
  it('escreve os algarismos simples', () => {
    expect(romanNumeral(1)).toBe('I')
    expect(romanNumeral(3)).toBe('III')
    expect(romanNumeral(5)).toBe('V')
    expect(romanNumeral(10)).toBe('X')
  })

  it('usa a forma subtrativa (IV, IX, XL, CM)', () => {
    expect(romanNumeral(4)).toBe('IV')
    expect(romanNumeral(9)).toBe('IX')
    expect(romanNumeral(40)).toBe('XL')
    expect(romanNumeral(900)).toBe('CM')
  })

  it('compõe números grandes', () => {
    expect(romanNumeral(1987)).toBe('MCMLXXXVII')
  })

  it('devolve o número em arábico quando não há forma romana', () => {
    // Zero e negativos não existem no sistema; a Forja pediria isso só por bug,
    // e um algarismo faltando seria pior que um número comum na tela.
    expect(romanNumeral(0)).toBe('0')
    expect(romanNumeral(-3)).toBe('-3')
    expect(romanNumeral(4000)).toBe('4000')
    expect(romanNumeral(2.5)).toBe('2.5')
  })
})
