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
export function createBoardGestures(
  view: () => BoardViewport,
  /** Quando devolve uma função, o arraste PINTA em vez de mover a vista — é o
   *  pincel do mestre (ALE-124). Devolver null é o estado normal. */
  painter: () => ((x: number, y: number, secondary: boolean) => void) | null = () => null,
) {
  const pointers = new Map<number, Point>()
  let travelled = 0
  let pinchSpread = 0
  /** A última casa pintada neste gesto: sem isto o mesmo quadrado é repintado a
   *  cada pixel do arraste, e cada repintura é uma mensagem no fio. */
  let lastPainted = ''
  /** O gesto começou com o botão direito: apaga em vez de pintar. */
  let secondaryDrag = false

  const onPointerDown = (event: PointerEvent) => {
    if (startsOnToken(event)) return
    pointers.set(event.pointerId, pointOf(event))
    travelled = 0
    lastPainted = ''
    const paint = painter()
    if (paint) {
      // Com o pincel na mão o gesto é OUTRO: captura já no toque, porque cada
      // pixel arrastado é tinta e não há clique nenhum a preservar.
      const host = event.currentTarget as HTMLElement
      capturar(host, event.pointerId)
      // `button` 2 é o direito no `pointerdown`; no `pointermove` ele vale −1 e
      // quem carrega a verdade é a máscara `buttons`.
      secondaryDrag = event.button === 2
      paintUnder(host, pointOf(event), paint, secondaryDrag)
      return
    }
    if (pointers.size === 2) pinchSpread = spreadOf(pointers)
  }

  const onPointerMove = (event: PointerEvent) => {
    const previous = pointers.get(event.pointerId)
    if (!previous) return
    const point = pointOf(event)
    pointers.set(event.pointerId, point)
    travelled += Math.abs(point.x - previous.x) + Math.abs(point.y - previous.y)
    const host = event.currentTarget as HTMLElement
    const paint = painter()
    if (paint) {
      paintUnder(host, point, paint, secondaryDrag || (event.buttons & 2) !== 0)
      return
    }
    capturarSeVirouArraste(host, event.pointerId)
    if (pointers.size >= 2) {
      pinch(host)
      return
    }
    view().panPixels(point.x - previous.x, point.y - previous.y)
  }

  /** A casa sob o ponteiro, pintada uma vez só por gesto. A conta é a da janela
   *  — origem mais pixels sobre o tamanho do quadrado — e não um
   *  `elementFromPoint`: a camada de casas clicáveis só existe para quem
   *  posiciona peça, e o pincel não depende dela. */
  const paintUnder = (
    host: HTMLElement,
    point: Point,
    paint: (x: number, y: number, secondary: boolean) => void,
    secondary: boolean,
  ) => {
    const local = anchorIn(host, point)
    const { x, y } = view().squareAt(local.x, local.y)
    const key = `${x},${y}`
    if (key === lastPainted) return
    lastPainted = key
    paint(x, y, secondary)
  }

  /** Ver a nota da função: capturar cedo demais mata o clique que pousa a peça. */
  const capturarSeVirouArraste = (host: HTMLElement, pointerId: number) => {
    if (travelled <= DRAG_SLOP_PX) return
    if (host.hasPointerCapture?.(pointerId)) return
    capturar(host, pointerId)
  }

  /** Capturar LANÇA quando o ponteiro não está mais ativo (o dedo saiu entre o
   *  evento e este código, ou o evento nem é de um ponteiro real). Sem o guarda,
   *  esse throw mata o handler no meio e o gesto inteiro se perde — foi
   *  exatamente o que aconteceu ao dirigir o pincel por evento sintético. */
  const capturar = (host: HTMLElement, pointerId: number) => {
    try {
      host.setPointerCapture?.(pointerId)
    } catch {
      // sem captura o arraste ainda funciona; só não sobrevive a sair da caixa
    }
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
    lastPainted = ''
    secondaryDrag = false
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

  /** Sem isto o botão direito abre o menu do browser no meio do apagar. */
  const onContextMenu = (event: MouseEvent) => {
    if (painter()) event.preventDefault()
  }

  return { onPointerDown, onPointerMove, onPointerUp, onWheel, onContextMenu }
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
