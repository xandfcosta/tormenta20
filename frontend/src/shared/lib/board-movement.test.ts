import { describe, expect, it } from 'vitest'
import { boardBudgetSquares, boardFootprint, boardPathCost } from './engine-wasm'

/**
 * A regra de movimento chega ao browser pelo MESMO Go que o servidor roda
 * (ALE-124). Este teste não repete a regra — ela é provada onde é autorada, em
 * `engine/board_movement_rules_test.go`, com a página do livro. O que se prova
 * aqui é a TRAVESSIA: o payload que o front monta e a resposta que ele lê, que é
 * onde uma regra certa vira número errado na tela.
 *
 * A suíte carrega o `.wasm` de verdade do disco (o hook `pretest` o constrói),
 * então isto exercita o motor de produção, não uma cópia.
 */
describe('a régua do tabuleiro atravessa para o browser', () => {
  it('caminho reto e diagonal voltam com o custo do livro', () => {
    const reto = boardPathCost(
      [
        { x: 0, y: 0 },
        { x: 1, y: 0 },
      ],
      [],
      6,
    )
    expect(reto).toMatchObject({ squares: 1, legal: true, stoppedAt: -1 })

    // A diagonal custa o dobro (p238) — se a travessia perdesse o formato do
    // payload, o motor mediria um caminho vazio e devolveria zero.
    expect(boardPathCost([{ x: 0, y: 0 }, { x: 1, y: 1 }], [], 6).squares).toBe(2)
  })

  it('o terreno difícil sai do array e chega como terreno', () => {
    const custo = boardPathCost(
      [
        { x: 0, y: 0 },
        { x: 1, y: 0 },
      ],
      [{ x: 1, y: 0 }],
      6,
    )

    expect(custo.squares).toBe(2)
  })

  it('estourar o deslocamento volta ilegal e dizendo onde parou', () => {
    const custo = boardPathCost(
      [
        { x: 0, y: 0 },
        { x: 1, y: 1 },
        { x: 2, y: 2 },
        { x: 3, y: 3 },
        { x: 4, y: 4 },
      ],
      [],
      6,
    )

    expect(custo.legal).toBe(false)
    expect(custo.stoppedAt).toBe(4)
    expect(custo.reason).toBeTruthy()
  })

  // Orçamento negativo = cena fora de combate: a régua mede e não recusa.
  it('sem orçamento, mede sem recusar', () => {
    const custo = boardPathCost(
      [
        { x: 0, y: 0 },
        { x: 1, y: 1 },
        { x: 2, y: 2 },
        { x: 3, y: 3 },
        { x: 4, y: 4 },
      ],
      [],
      -1,
    )

    expect(custo).toMatchObject({ squares: 8, legal: true })
  })

  it('deslocamento e tamanho viram quadrados', () => {
    expect(boardBudgetSquares(9)).toBe(6)
    expect(boardFootprint('Grande')).toBe(2)
    // A grafia do bestiário e a da ficha convergem no motor.
    expect(boardFootprint('medio')).toBe(1)
  })
})
