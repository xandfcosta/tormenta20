import { describe, expect, it } from 'vitest'
import { boardPathCost } from '@/shared/lib/engine-wasm'
import { pathBetween, squareAt } from './board-path'

/**
 * O caminho é geometria do cliente; o CUSTO é do motor Go. Por isso as
 * asserções passam pelo `boardPathCost`: o que se prova aqui é que o caminho
 * desenhado é aceito e cobrado pelo mesmo código que o servidor roda — um
 * caminho com buraco voltaria "ilegal" em vez de um número plausível.
 */
describe('pathBetween', () => {
  it('anda quadrado a quadrado até o destino', () => {
    const path = pathBetween({ x: 0, y: 0 }, { x: 3, y: 1 })

    expect(path[0]).toEqual({ x: 0, y: 0 })
    expect(path.at(-1)).toEqual({ x: 3, y: 1 })
    // O motor recusa passo que não seja de casa vizinha: se ele mediu, o
    // caminho é contíguo.
    expect(boardPathCost(path, [], -1).legal).toBe(true)
  })

  it('a diagonal vem primeiro, e o custo é o mesmo do L', () => {
    const diagonalPrimeiro = pathBetween({ x: 0, y: 0 }, { x: 3, y: 1 })
    const emL = [
      { x: 0, y: 0 },
      { x: 1, y: 0 },
      { x: 2, y: 0 },
      { x: 3, y: 0 },
      { x: 3, y: 1 },
    ]

    // A diagonal dobrada (T20 p238) é o que torna os dois iguais: um passo
    // diagonal custa exatamente dois ortogonais, e é por isso que esta tela não
    // precisa de pathfinding.
    expect(boardPathCost(diagonalPrimeiro, [], -1).squares).toBe(
      boardPathCost(emL, [], -1).squares,
    )
  })

  it('a peça que não sai do lugar tem caminho de um quadrado só', () => {
    expect(pathBetween({ x: 2, y: 2 }, { x: 2, y: 2 })).toEqual([{ x: 2, y: 2 }])
  })
})

describe('squareAt', () => {
  it('traduz o ponto da tela para o quadrado do tabuleiro, com a origem da janela', () => {
    const view = { originX: -3, originY: 5, cellPx: 40 }

    expect(squareAt({ x: 0, y: 0 }, view)).toEqual({ x: -3, y: 5 })
    expect(squareAt({ x: 95, y: 41 }, view)).toEqual({ x: -1, y: 6 })
  })
})
