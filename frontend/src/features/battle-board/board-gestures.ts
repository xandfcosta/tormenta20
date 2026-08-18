import type { Anchor, BoardViewport } from './board-viewport'

/** Quanto o ponteiro precisa andar para o gesto virar ARRASTE. Abaixo disto é
 *  um toque, e um toque tem de continuar pousando a peça. */
const DRAG_SLOP_PX = 4

/** O passo do zoom por entalhe da roda. Fixo, e não proporcional ao `deltaY`:
 *  cada mouse e cada trackpad reportam uma escala diferente, e num deles o
 *  mesmo gesto saltaria dez quadrados. */
const WHEEL_STEP_PX = 6

/** A peça leva esta marca para o arraste saber onde NÃO começar. */
export const TOKEN_PIECE_ATTR = 'data-token-piece'

type Point = { x: number; y: number }

/**
 * Mouse e toque no tabuleiro (ALE-140): arrastar o fundo move a vista, roda e
 * pinça dão zoom ancorado no ponteiro.
 *
 * Os botões de seta e `−`/`+` continuam existindo, e isso não é detalhe de
 * compatibilidade: gesto NUNCA é o único caminho — é a regra da casa que
 * mantém a cena utilizável no teclado e no leitor de tela.
 *
 * Arrastar a partir de uma PEÇA não move a vista. A distinção é onde o gesto
 * COMEÇA, e é nesse mesmo lugar que o arraste da peça vai morar quando existir.
 *
 * `create*` porque guarda estado entre eventos (quais ponteiros estão no vidro):
 * nasce uma vez no corpo do componente, nunca por evento.
 *
 * A CAPTURA do ponteiro só acontece quando o gesto vira arraste, e essa ordem
 * é o coração deste arquivo: com o ponteiro capturado o browser reaponta o
 * `click` para a superfície, então capturar já no `pointerdown` fazia TODO
 * clique numa casa morrer — e clicar na casa é como o mestre pousa a peça.
 * Capturando só depois do limiar, o toque continua sendo toque e o arraste
 * ganha o que precisa: sobreviver a sair da caixa, e não terminar pousando a
 * peça onde o dedo saiu.
 *
 * @example const gestos = createBoardGestures(() => props.view)
 *          <div {...gestos} />
 */
export function createBoardGestures(view: () => BoardViewport) {
  const pointers = new Map<number, Point>()
  let travelled = 0
  let pinchSpread = 0

  const onPointerDown = (event: PointerEvent) => {
    if (startsOnToken(event)) return
    pointers.set(event.pointerId, pointOf(event))
    travelled = 0
    if (pointers.size === 2) pinchSpread = spreadOf(pointers)
  }

  const onPointerMove = (event: PointerEvent) => {
    const previous = pointers.get(event.pointerId)
    if (!previous) return
    const point = pointOf(event)
    pointers.set(event.pointerId, point)
    travelled += Math.abs(point.x - previous.x) + Math.abs(point.y - previous.y)
    const host = event.currentTarget as HTMLElement
    capturarSeVirouArraste(host, event.pointerId)
    if (pointers.size >= 2) {
      pinch(host)
      return
    }
    view().panPixels(point.x - previous.x, point.y - previous.y)
  }

  /** Ver a nota da função: capturar cedo demais mata o clique que pousa a peça. */
  const capturarSeVirouArraste = (host: HTMLElement, pointerId: number) => {
    if (travelled <= DRAG_SLOP_PX) return
    if (host.hasPointerCapture?.(pointerId)) return
    host.setPointerCapture?.(pointerId)
  }

  const pinch = (host: HTMLElement) => {
    const next = spreadOf(pointers)
    const previous = pinchSpread
    pinchSpread = next
    if (previous <= 0 || next <= 0) return
    view().zoomByFactor(next / previous, anchorIn(host, midpointOf(pointers)))
  }

  /** Serve `pointerup` e `pointercancel`: os dois querem dizer "este dedo saiu". */
  const onPointerUp = (event: PointerEvent) => {
    pointers.delete(event.pointerId)
    pinchSpread = 0
  }

  const onWheel = (event: WheelEvent) => {
    if (event.deltaY === 0) return
    // Sem isto o gesto rola a PÁGINA em vez de dar zoom, e a mesa perde a cena
    // de vista para achar o tabuleiro de novo.
    event.preventDefault()
    const host = event.currentTarget as HTMLElement
    const step = event.deltaY < 0 ? WHEEL_STEP_PX : -WHEEL_STEP_PX
    view().zoom(step, anchorIn(host, pointOf(event)))
  }

  return { onPointerDown, onPointerMove, onPointerUp, onWheel }
}

function pointOf(event: { clientX: number; clientY: number }): Point {
  return { x: event.clientX, y: event.clientY }
}

function startsOnToken(event: PointerEvent): boolean {
  const target = event.target as HTMLElement | null
  return target?.closest?.(`[${TOKEN_PIECE_ATTR}]`) != null
}

/** O ponto da vista, em pixels a partir da quina dela — que é o que o zoom
 *  ancorado espera receber. */
function anchorIn(host: HTMLElement, point: Point): Anchor {
  const box = host.getBoundingClientRect()
  return { x: point.x - box.left, y: point.y - box.top }
}

/** A distância entre os dois primeiros dedos: é ela que a pinça compara. */
function spreadOf(pointers: Map<number, Point>): number {
  const [first, second] = [...pointers.values()]
  if (!first || !second) return 0
  return Math.hypot(second.x - first.x, second.y - first.y)
}

function midpointOf(pointers: Map<number, Point>): Point {
  const [first, second] = [...pointers.values()]
  if (!first || !second) return first ?? { x: 0, y: 0 }
  return { x: (first.x + second.x) / 2, y: (first.y + second.y) / 2 }
}
