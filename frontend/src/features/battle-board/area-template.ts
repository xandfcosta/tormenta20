import { type Accessor, createMemo, createSignal } from 'solid-js'
import { type BoardAreaKind, type BoardSquare, boardArea } from '@/shared/lib/engine-wasm'
import type { BoardToken } from '@/shared/realtime/realtime'

/**
 * O gabarito de área na mesa (ALE-124, fatia 6b): o cone, a esfera, a linha e o
 * quadrado do livro (T20 p225) desenhados sobre a cena — e, o que mais importa,
 * QUEM ESTÁ DENTRO.
 *
 * A pergunta que ele responde é a que trava o turno do conjurador: "se eu
 * soltar a bola de fogo aqui, quem pega?". Hoje ela se responde apontando o
 * dedo na tela e discutindo, e a discussão acontece com a mesa parada.
 *
 * A FORMA vem do motor Go, transcrita da figura da p225 — a tela não sabe
 * desenhar cone, e é bom que não saiba. Aqui mora só o estado da escolha (que
 * gabarito, de que tamanho, apontado para onde) e o cruzamento com as peças.
 *
 * Como a régua, é LOCAL: desenhar não muda a cena e não vai ao servidor.
 *
 * `create*` porque GUARDA estado entre chamadas: nasce uma vez no corpo do
 * componente, nunca por evento.
 */
export function createAreaTemplate(tokens: Accessor<readonly BoardToken[]>) {
  const [kind, setKind] = createSignal<BoardAreaKind>('esfera')
  const [size, setSize] = createSignal(2)
  const [origin, setOrigin] = createSignal<BoardSquare | null>(null)
  const [direction, setDirection] = createSignal<BoardSquare | null>(null)

  /** Cone e linha PRECISAM apontar para algum lado; esfera e quadrado, não. */
  const needsDirection = () => kind() === 'cone' || kind() === 'linha'

  /**
   * O primeiro clique põe a origem; o segundo, quando o gabarito aponta, diz
   * para onde. Depois disso o clique seguinte RECOMEÇA — a mesa posiciona a
   * mesma bola de fogo três vezes antes de decidir, e um botão de limpar entre
   * cada tentativa seria um clique a mais em cada uma.
   */
  const pick = (x: number, y: number) => {
    if (origin() && needsDirection() && !direction()) return setDirection(octant(origin(), { x, y }))
    setDirection(null)
    setOrigin({ x, y })
  }

  const clear = () => {
    setOrigin(null)
    setDirection(null)
  }

  /**
   * Trocar de gabarito LARGA o que estava posto. Sem isso, o primeiro clique
   * depois da troca caía na regra do segundo clique e APONTAVA o gabarito novo
   * a partir da origem do antigo — medido no browser, escolher "Cone" com uma
   * esfera na tela desenhava um cone apontado para o lado de onde se clicou.
   */
  const chooseKind = (next: BoardAreaKind) => {
    clear()
    setKind(next)
  }

  const squares = createMemo<readonly BoardSquare[]>(() => {
    const de = origin()
    if (!de) return []
    if (needsDirection() && !direction()) return []
    return boardArea(de, {
      kind: kind(),
      size: size(),
      direction: direction() ?? { x: 0, y: 0 },
    })
  })

  /**
   * Quem o gabarito pega. A peça entra se QUALQUER quadrado do corpo dela cair
   * na área — uma Colossal ocupa 6×6 (p107), e exigir que ela caiba inteira
   * deixaria o dragão de fora do próprio incêndio.
   */
  const inside = createMemo<readonly BoardToken[]>(() => {
    const area = new Set(squares().map((casa) => `${casa.x},${casa.y}`))
    if (area.size === 0) return []
    return tokens().filter((peca) => {
      const lado = peca.footprint > 0 ? peca.footprint : 1
      for (let dx = 0; dx < lado; dx++) {
        for (let dy = 0; dy < lado; dy++) {
          if (area.has(`${peca.x + dx},${peca.y + dy}`)) return true
        }
      }
      return false
    })
  })

  return { kind, chooseKind, size, setSize, origin, direction, needsDirection, pick, clear, squares, inside }
}

export type AreaTemplate = ReturnType<typeof createAreaTemplate>

/**
 * A direção do segundo clique, em passo unitário. Um clique quase em linha vira
 * ortogonal e um clique perto de 45° vira diagonal, porque o livro desenha o
 * cone nessas duas formas e não numa terceira: sem a zona morta, um pixel de
 * diferença no clique trocaria a forma inteira do gabarito debaixo do dedo.
 */
function octant(from: BoardSquare | null, to: BoardSquare): BoardSquare {
  if (!from) return { x: 1, y: 0 }
  const dx = to.x - from.x
  const dy = to.y - from.y
  if (dx === 0 && dy === 0) return { x: 1, y: 0 }
  if (Math.abs(dx) > 2 * Math.abs(dy)) return { x: Math.sign(dx), y: 0 }
  if (Math.abs(dy) > 2 * Math.abs(dx)) return { x: 0, y: Math.sign(dy) }
  return { x: Math.sign(dx) || 1, y: Math.sign(dy) || 1 }
}
