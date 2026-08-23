import { describe, expect, it } from 'vitest'
import { CABECALHO_DO_BLOCO_PX, ITEM_DA_FILA_PX, cabemNaFila, janelaDaFila } from './rail-geometry'

/**
 * A GEOMETRIA DO TRILHO (ALE-211).
 *
 * A fila recolhida não rola: mostra quem está na vez no centro, quem já agiu
 * acima e quem ainda vai abaixo. Quantos cabem é conta por viewport, e é a
 * parte que carrega REGRA — com um número errado de vizinhos, o trilho mente
 * sobre quem já jogou. É unitário por isso, e não porque a função é pequena.
 */

describe('cabemNaFila', () => {
  it('conta quantos itens entram em METADE do trilho', () => {
    // Metade de 900 são 450; menos o cabeçalho do bloco, 418; a 52 por item, 8.
    expect(cabemNaFila(900)).toBe(8)
  })

  /**
   * Zero é "ainda não medi", e a resposta é ZERO — nunca um palpite. Quem chama
   * mostra a fila inteira nesse quadro: ver demais por um quadro é melhor do
   * que afirmar uma vizinhança inventada, e o recorte do CSS segura o resto.
   */
  it('altura zero responde zero, e não um chute', () => {
    expect(cabemNaFila(0)).toBe(0)
  })

  // Trilho curto demais para um item sequer: a conta não devolve negativo.
  it('trilho apertado responde zero em vez de número negativo', () => {
    expect(cabemNaFila(CABECALHO_DO_BLOCO_PX)).toBe(0)
    expect(cabemNaFila(10)).toBe(0)
  })

  it('cresce com o trilho', () => {
    expect(cabemNaFila(1400)).toBeGreaterThan(cabemNaFila(700))
  })

  // A constante e a classe que a produz vivem no mesmo commit ou a fila passa a
  // mentir. Este guarda não copia o valor: ele afirma a RELAÇÃO.
  it('o item cabe inteiro no espaço que a conta reserva para ele', () => {
    const cabem = cabemNaFila(900)

    expect(cabem * ITEM_DA_FILA_PX + CABECALHO_DO_BLOCO_PX).toBeLessThanOrEqual(450)
  })
})

describe('janelaDaFila', () => {
  it('cabendo todo mundo, mostra todo mundo', () => {
    expect(janelaDaFila({ total: 4, turnIndex: 2, cabem: 8 })).toEqual({ inicio: 0, fim: 3 })
  })

  it('no meio da fila, a vez fica no centro', () => {
    // Cinco lugares: dois acima, a vez, dois abaixo.
    expect(janelaDaFila({ total: 9, turnIndex: 4, cabem: 5 })).toEqual({ inicio: 2, fim: 6 })
  })

  /**
   * No começo da rodada ninguém agiu, então NÃO há o que pôr acima — e reservar
   * espaço vazio ali seria desenhar um passado que não existe. O grampo devolve
   * a sobra para baixo, que é onde há gente.
   */
  it('no primeiro turno a vez fica no topo e a sobra vai para baixo', () => {
    expect(janelaDaFila({ total: 9, turnIndex: 0, cabem: 5 })).toEqual({ inicio: 0, fim: 4 })
  })

  it('no último turno a vez fica no pé e a sobra vai para cima', () => {
    expect(janelaDaFila({ total: 9, turnIndex: 8, cabem: 5 })).toEqual({ inicio: 4, fim: 8 })
  })

  /**
   * Fora de combate não há centro. A fila começa do TOPO, que é a ordem em que
   * ela vai ser jogada — centrar em ninguém esconderia justamente o primeiro,
   * que é quem o mestre está prestes a chamar.
   */
  it('sem vez de ninguém, mostra do começo', () => {
    expect(janelaDaFila({ total: 9, turnIndex: -1, cabem: 5 })).toEqual({ inicio: 0, fim: 4 })
  })

  // Antes da primeira medição: mostra tudo e deixa o CSS recortar. Esconder a
  // fila inteira por um quadro seria pior que mostrá-la demais.
  it('sem medida ainda, mostra a fila inteira', () => {
    expect(janelaDaFila({ total: 9, turnIndex: 4, cabem: 0 })).toEqual({ inicio: 0, fim: 8 })
  })

  it('fila vazia devolve uma janela vazia', () => {
    expect(janelaDaFila({ total: 0, turnIndex: -1, cabem: 5 })).toEqual({ inicio: 0, fim: -1 })
  })

  // Com um lugar só, ele é da VEZ: o trilho existe para responder de quem é.
  it('cabendo um só, é o da vez', () => {
    expect(janelaDaFila({ total: 9, turnIndex: 6, cabem: 1 })).toEqual({ inicio: 6, fim: 6 })
  })

  /**
   * A janela nunca sai da fila e nunca muda de tamanho. É a propriedade que
   * impede o trilho de desenhar um vizinho que não existe — e ela vale em toda
   * combinação, não só nas que eu lembrei de escrever.
   */
  it('a janela cabe na fila em qualquer turno', () => {
    const total = 9
    for (let cabem = 1; cabem <= 12; cabem++) {
      for (let turnIndex = -1; turnIndex < total; turnIndex++) {
        const { inicio, fim } = janelaDaFila({ total, turnIndex, cabem })

        expect(inicio, `cabem=${cabem} turno=${turnIndex}`).toBeGreaterThanOrEqual(0)
        expect(fim, `cabem=${cabem} turno=${turnIndex}`).toBeLessThan(total)
        expect(fim - inicio + 1).toBe(Math.min(cabem, total))
      }
    }
  })

  // E o da vez está SEMPRE dentro dela: é a razão de o trilho existir.
  it('quem está na vez nunca fica de fora', () => {
    const total = 9
    for (let cabem = 1; cabem <= 12; cabem++) {
      for (let turnIndex = 0; turnIndex < total; turnIndex++) {
        const { inicio, fim } = janelaDaFila({ total, turnIndex, cabem })

        expect(turnIndex, `cabem=${cabem} turno=${turnIndex}`).toBeGreaterThanOrEqual(inicio)
        expect(turnIndex).toBeLessThanOrEqual(fim)
      }
    }
  })
})
