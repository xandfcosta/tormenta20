import { describe, expect, it } from 'vitest'
import { MONSTER_TIPOS, MONSTER_TIPO_LABEL, formatNd } from './monster-format'

describe('formatNd', () => {
  it('escreve o ND fracionário como o livro escreve', () => {
    expect(formatNd(0.25)).toBe('1/4')
    expect(formatNd(0.5)).toBe('1/2')
  })

  it('deixa o ND inteiro em paz', () => {
    expect(formatNd(1)).toBe('1')
    expect(formatNd(12)).toBe('12')
  })

  it('tolera a imprecisão de ponto flutuante do catálogo', () => {
    expect(formatNd(0.2500001)).toBe('1/4')
  })
})

describe('MONSTER_TIPO_LABEL', () => {
  it('rotula todos os tipos que os chips oferecem', () => {
    // Um tipo sem rótulo viraria um chip vazio na tela.
    for (const tipo of MONSTER_TIPOS) {
      expect(MONSTER_TIPO_LABEL[tipo]).toBeTruthy()
    }
  })
})
