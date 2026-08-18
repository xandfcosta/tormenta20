import type { BoardSquare } from '@/shared/lib/engine-wasm'

/**
 * O caminho entre dois quadrados (ALE-124).
 *
 * Isto é GEOMETRIA, não regra: quem cobra o caminho é o motor Go
 * (`boardPathCost`), aqui só se desenha por onde a peça anda. A distinção é a
 * mesma da ALE-104 — uma segunda implementação da regra da diagonal seria uma
 * segunda verdade sobre o livro.
 *
 * **Por que não existe pathfinding:** com a diagonal custando o dobro (T20
 * p238), um passo diagonal (2) vale exatamente dois passos ortogonais (1+1).
 * Então TODO caminho monótono entre A e B custa `|dx| + |dy|`, e não há
 * caminho mais barato para procurar. O que muda isso é terreno difícil, que
 * chega na fatia do mapa — e aí quem responde continua sendo o Go.
 *
 * A diagonal vem PRIMEIRO porque é o que o olho espera de quem corta caminho;
 * o custo seria o mesmo em L.
 *
 * @example pathBetween({ x: 0, y: 0 }, { x: 3, y: 1 }) // 4 passos, custo 4
 */
export function pathBetween(from: BoardSquare, to: BoardSquare): BoardSquare[] {
  const path: BoardSquare[] = [from]
  let { x, y } = from
  while (x !== to.x || y !== to.y) {
    x += Math.sign(to.x - x)
    y += Math.sign(to.y - y)
    path.push({ x, y })
  }
  return path
}

/** O quadrado sob um ponto da tela, em coordenadas do TABULEIRO. */
export function squareAt(
  point: { x: number; y: number },
  view: { originX: number; originY: number; cellPx: number },
): BoardSquare {
  return {
    x: view.originX + Math.floor(point.x / view.cellPx),
    y: view.originY + Math.floor(point.y / view.cellPx),
  }
}
