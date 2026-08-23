import { describe, expect, it } from 'vitest'
import { erroDoTibar, saldoResultante } from './tibar-rules'

const TETO = 1_000_000

/**
 * A conta do dinheiro (ALE-224). Unitário e não de tela porque o que está aqui
 * é REGRA e tem armadilha: o binário, o piso e o teto. A composição — qual
 * gesto o botão dispara, o que a prévia mostra — se prova montando a Mochila.
 */
describe('saldoResultante', () => {
  it('receber soma, gastar subtrai, corrigir escreve', () => {
    expect(saldoResultante(1200, 'receber', 350)).toBe(1550)
    expect(saldoResultante(1200, 'gastar', 80)).toBe(1120)
    expect(saldoResultante(1200, 'corrigir', 350)).toBe(350)
  })

  /**
   * O dinheiro do livro é fracionário — uma vela custa T$ 0,1 (Tabela 3-2,
   * p143) — e décimos não fecham em binário: `1200.3 - 80.1` dá
   * 1120.1999999999998, e esse número iria para o banco e para a tela. Duas
   * casas é a mesma precisão que o `formatTibar` mostra, então o que se lê e o
   * que se grava passam a ser o mesmo número.
   */
  it('fecha em duas casas, senão o décimo do livro vira dízima', () => {
    expect(saldoResultante(1200.3, 'gastar', 80.1)).toBe(1120.2)
    expect(saldoResultante(0.1, 'receber', 0.2)).toBe(0.3)
  })
})

describe('erroDoTibar', () => {
  it('a operação que cabe passa', () => {
    expect(erroDoTibar(1200, 'gastar', 80, TETO)).toBeNull()
    expect(erroDoTibar(0, 'receber', 350, TETO)).toBeNull()
  })

  /**
   * O piso é ZERO e não um aviso: dívida na ficha viraria carga de moeda
   * NEGATIVA, que COMPRARIA espaço na mochila em vez de ocupar (p141, ALE-215).
   * A mensagem traz o saldo porque "valor inválido" manda o jogador conferir a
   * ficha no meio da sessão.
   */
  it('gastar mais do que se tem é recusado dizendo quanto se tem', () => {
    expect(erroDoTibar(50, 'gastar', 80, TETO)).toBe('Você tem T$ 50.')
    expect(erroDoTibar(1234.5, 'gastar', 2000, TETO)).toBe('Você tem T$ 1.234,5.')
  })

  it('gastar tudo é permitido — o piso é zero, não um', () => {
    expect(erroDoTibar(50, 'gastar', 50, TETO)).toBeNull()
  })

  // O teto não é regra do livro: é o que impede um número absurdo de virar um
  // bilhão de espaços de carga. Somar precisa respeitá-lo, e é o caso que só
  // aparece pelo modo novo — antes só se escrevia o saldo.
  it('receber que estoura o teto é recusado', () => {
    expect(erroDoTibar(TETO - 10, 'receber', 100, TETO)).toBe('O limite é T$ 1.000.000.')
    expect(erroDoTibar(TETO - 10, 'receber', 10, TETO)).toBeNull()
  })

  it('valor negativo ou não numérico é recusado antes de qualquer conta', () => {
    expect(erroDoTibar(100, 'receber', -1, TETO)).toBe('Informe um valor a partir de 0.')
    expect(erroDoTibar(100, 'corrigir', Number.NaN, TETO)).toBe('Informe um valor a partir de 0.')
  })
})
