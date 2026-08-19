import { createMemo, createSignal } from 'solid-js'
import { type BoardMeasurement, type BoardSquare, boardMeasure } from '@/shared/lib/engine-wasm'

/**
 * A régua da mesa (ALE-124, fatia 6): dois quadrados e a distância entre eles,
 * com a faixa de alcance do livro (T20 p224).
 *
 * A pergunta que ela responde é feita em voz alta toda rodada — "dá para
 * acertar daqui?" —, e hoje se responde contando quadrado com o dedo na tela,
 * que é justamente o que um tabuleiro digital deveria poupar.
 *
 * NÃO vai ao servidor e NÃO é transmitida: medir não muda a cena, e a régua de
 * um jogador não é assunto da mesa. É a diferença entre isto e o movimento
 * proposto, que é estado porque a mesa inteira decide sobre ele.
 *
 * `create*` porque GUARDA estado entre chamadas (as duas pontas): nasce uma vez
 * no corpo do componente, nunca por evento.
 *
 * @example const regua = createRuler(); regua.pick(3, 2); regua.pick(9, 2)
 */
export function createRuler() {
  const [from, setFrom] = createSignal<BoardSquare | null>(null)
  const [to, setTo] = createSignal<BoardSquare | null>(null)

  /**
   * Cada clique é uma ponta: o primeiro fixa a origem, o segundo fecha a
   * medida, e o terceiro RECOMEÇA de onde clicou. Sem o recomeço, medir a
   * distância seguinte exigiria um botão de limpar — e a mesa mede muitas vezes
   * seguidas, sempre a partir de outro lugar.
   */
  const pick = (x: number, y: number) => {
    if (from() && !to()) return setTo({ x, y })
    setTo(null)
    setFrom({ x, y })
  }

  const clear = () => {
    setFrom(null)
    setTo(null)
  }

  /** A leitura, quando as duas pontas existem. A conta é do MOTOR GO, o mesmo
   *  que o servidor usa para cobrar caminho — a tela não sabe somar diagonal, e
   *  é bom que não saiba. */
  const reading = createMemo<BoardMeasurement | null>(() => {
    const inicio = from()
    const fim = to()
    if (!inicio || !fim) return null
    return boardMeasure(inicio, fim)
  })

  return { from, to, pick, clear, reading }
}

export type Ruler = ReturnType<typeof createRuler>
