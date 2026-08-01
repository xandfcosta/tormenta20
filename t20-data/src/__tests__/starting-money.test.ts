import { describe, expect, it } from 'vitest'
import { startingMoneyForLevel } from '../starting-money'

describe('Tabela 3-1: Dinheiro Inicial (p140)', () => {
  it('amostras pinadas do livro', () => {
    expect(startingMoneyForLevel(2)).toBe(300)
    expect(startingMoneyForLevel(6)).toBe(3_000)
    expect(startingMoneyForLevel(10)).toBe(13_000)
    expect(startingMoneyForLevel(20)).toBe(260_000)
  })

  it('nível 1 → null (rola 4d6)', () => {
    expect(startingMoneyForLevel(1)).toBeNull()
  })

  it('nível fora do range lança com o valor', () => {
    expect(() => startingMoneyForLevel(21)).toThrow(/got 21/)
    expect(() => startingMoneyForLevel(0)).toThrow(/got 0/)
  })
})
