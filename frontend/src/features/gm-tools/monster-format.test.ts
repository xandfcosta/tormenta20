import { describe, expect, it } from 'vitest'
import { formatNd } from './monster-format'

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

// O bloco `MONSTER_TIPO_LABEL` saiu inteiro na ALE-187: ele varria os tipos
// afirmando que cada um tem rótulo, e o `Record<MonsterTipo, string>` é TOTAL —
// o typechecker já recusa um tipo sem rótulo, que era o defeito descrito. Só a
// string vazia escapava dele, e ninguém a escreve.
