import { createSignal } from 'solid-js'
import type { BoardToken } from '@/shared/realtime/realtime'

/** O quadrado vale 1,5m (T20 p236) — a conversão da tela sai de um lugar só. */
export const SQUARE_METRES = 1.5

/**
 * Os limites do zoom, em pixels por quadrado. Abaixo de 20 a peça vira um ponto
 * e o rótulo some; acima de 96 uma tela de 1024 mostra 10 quadrados, menos que
 * dois deslocamentos padrão (9m = 6 quadrados, p106) — e aí o mestre deixa de
 * ver para onde dá para andar.
 */
export const MIN_CELL_PX = 20
export const MAX_CELL_PX = 96
const DEFAULT_CELL_PX = 44

/**
 * A janela padrão em quadrados, usada até a medição real chegar. Existe porque o
 * plano é INFINITO: não há "tamanho do tabuleiro" para derivar, e um zero
 * enquanto o ResizeObserver não disparou deixaria a cena em branco no primeiro
 * quadro — e em jsdom, onde tudo mede zero, para sempre.
 */
const DEFAULT_COLS = 20
const DEFAULT_ROWS = 14

export type BoardViewport = ReturnType<typeof createBoardViewport>

/** Um ponto da VISTA em pixels, medido da quina superior-esquerda dela. */
export type Anchor = { x: number; y: number }

/**
 * A JANELA sobre o plano infinito (ALE-124).
 *
 * O tabuleiro não tem bordas; quem tem tamanho é isto aqui. A origem é o
 * quadrado do canto superior-esquerdo da vista e pode ser negativa — empurrar a
 * cena para a esquerda é andar com a janela, não acabar o mapa.
 *
 * A origem é FRACIONÁRIA desde a ALE-140. O estado do servidor continua em
 * quadrados inteiros — quem ganhou casas decimais foi só o enquadramento, e ele
 * precisa delas: arrastar com o dedo produz pixels, e uma origem que só anda de
 * quadrado em quadrado faria a cena pular 44px atrás do dedo em vez de segui-lo.
 * Quem precisa de quadrado inteiro (a camada de casas clicáveis) arredonda para
 * baixo e desloca a grade pelo resto.
 *
 * `create*` porque GUARDA estado entre chamadas (origem e zoom): nasce uma vez
 * no corpo do componente, nunca por evento. E nasce na PÁGINA, não na feature,
 * porque `Show`/`Tabs` desmontam o conteúdo inativo e o enquadramento morreria a
 * cada troca de região.
 *
 * @example const view = createBoardViewport(); view.centerOn(3, -2)
 */
export function createBoardViewport() {
  const [originX, setOriginX] = createSignal(0)
  const [originY, setOriginY] = createSignal(0)
  const [cellPx, setCellPx] = createSignal(DEFAULT_CELL_PX)
  const [cols, setCols] = createSignal(DEFAULT_COLS)
  const [rows, setRows] = createSignal(DEFAULT_ROWS)

  // A primeira medição também CENTRALIZA na origem. Sem isso a vista nasce com
  // a origem no canto superior-esquerdo e a fileira de peças novas aparece
  // colada na quina, com três quartos da tela mostrando plano vazio.
  let centered = false
  // O último tamanho medido, em pixels. Guardado porque quantos quadrados cabem
  // depende do ZOOM, e o zoom muda sem a caixa mudar de tamanho — o
  // ResizeObserver não dispara, e sem isto `cols`/`rows` ficavam com a contagem
  // do zoom anterior até alguém redimensionar a janela do browser (ALE-140).
  let widthPx = 0
  let heightPx = 0

  /**
   * Mede a área visível em quadrados. Tamanho zero (jsdom, primeiro quadro) NÃO
   * apaga a janela: mantém a anterior, senão a cena pisca vazia.
   *
   * É AQUI que o centro é preservado, e não no zoom: a janela é medida em
   * quadrados, então mudar o zoom não move nada sozinho — quem muda quantos
   * quadrados cabem é a remedição que vem depois, e é ela que jogaria a cena
   * para o canto. Vale igual para redimensionar a janela do browser.
   */
  const measure = (width: number, height: number) => {
    if (width <= 0 || height <= 0) return
    widthPx = width
    heightPx = height
    const nextCols = squaresIn(width)
    const nextRows = squaresIn(height)
    if (!centered) {
      centered = true
      setCols(nextCols)
      setRows(nextRows)
      centerOn(0, 0)
      return
    }
    const centerX = originX() + cols() / 2
    const centerY = originY() + rows() / 2
    setCols(nextCols)
    setRows(nextRows)
    setOriginX(centerX - nextCols / 2)
    setOriginY(centerY - nextRows / 2)
  }

  const squaresIn = (lengthPx: number) => Math.max(1, Math.floor(lengthPx / cellPx()))

  /** Reconta quantos quadrados cabem, com o tamanho já medido. Não mexe na
   *  origem: quem decide para onde a vista olha é quem chamou. */
  const recount = () => {
    if (widthPx <= 0 || heightPx <= 0) return
    setCols(squaresIn(widthPx))
    setRows(squaresIn(heightPx))
  }

  const pan = (dxSquares: number, dySquares: number) => {
    setOriginX((x) => x + dxSquares)
    setOriginY((y) => y + dySquares)
  }

  /** O quadrado sob um ponto da vista, em pixels a partir da quina dela. A
   *  conta mora aqui pelo mesmo motivo do `panPixels`: é o único lugar que
   *  conhece o tamanho do quadrado na tela. `floor` e não `round` porque o
   *  quadrado é a CASA em que o ponto caiu, não a mais próxima. */
  const squareAt = (px: number, py: number) => ({
    x: Math.floor(originX() + px / cellPx()),
    y: Math.floor(originY() + py / cellPx()),
  })

  /** Arrastar entrega PIXELS, e a conversão para quadrado mora aqui — é o único
   *  lugar que conhece o tamanho do quadrado na tela. O sinal é invertido de
   *  propósito: puxar o mapa para a direita mostra o que está à ESQUERDA. */
  const panPixels = (dxPx: number, dyPx: number) => {
    pan(-dxPx / cellPx(), -dyPx / cellPx())
  }

  /** Põe o quadrado no CENTRO da janela, não no canto: quem pede para centralizar
   *  quer ver o que está em volta. */
  const centerOn = (x: number, y: number) => {
    setOriginX(x - Math.floor(cols() / 2))
    setOriginY(y - Math.floor(rows() / 2))
  }

  /**
   * Aproxima ou afasta, ANCORADO num ponto da tela: o quadrado que está sob o
   * ponteiro continua sob o ponteiro. É a diferença entre navegar e se perder —
   * sem âncora, dar zoom na briga joga a briga para fora da tela e o mestre
   * gasta o turno procurando (ALE-140).
   *
   * `anchor` vem em pixels a partir da quina superior-esquerda da vista. Sem
   * âncora, o ponto fixo é o CENTRO, que é o que os botões `−`/`+` querem.
   */
  const zoom = (deltaPx: number, anchor?: Anchor) => {
    zoomTo(cellPx() + deltaPx, anchor)
  }

  /** A forma natural da PINÇA: a distância entre os dedos dobrar dobra o
   *  quadrado. Somar pixels aqui daria um salto que depende do tamanho da mão. */
  const zoomByFactor = (factor: number, anchor?: Anchor) => {
    if (!Number.isFinite(factor) || factor <= 0) return
    zoomTo(cellPx() * factor, anchor)
  }

  const zoomTo = (nextCellPx: number, anchor?: Anchor) => {
    const before = cellPx()
    const after = clamp(nextCellPx, MIN_CELL_PX, MAX_CELL_PX)
    if (after === before) return
    const ax = anchor?.x ?? (cols() * before) / 2
    const ay = anchor?.y ?? (rows() * before) / 2
    setCellPx(after)
    // O quadrado sob a âncora é `origem + a/celula`; manter esse número com a
    // célula nova é toda a conta.
    setOriginX((x) => x + ax / before - ax / after)
    setOriginY((y) => y + ay / before - ay / after)
    recount()
  }

  /** Enquadra todas as peças. Num plano infinito, "voltar ao começo" não
   *  significa nada — o que o mestre quer é achar o grupo. */
  const fit = (tokens: readonly BoardToken[]) => {
    if (tokens.length === 0) {
      setOriginX(-Math.floor(cols() / 2))
      setOriginY(-Math.floor(rows() / 2))
      return
    }
    const box = boundingBox(tokens)
    centerOn(Math.round((box.minX + box.maxX) / 2), Math.round((box.minY + box.maxY) / 2))
  }

  return {
    originX,
    originY,
    cellPx,
    cols,
    rows,
    measure,
    pan,
    panPixels,
    squareAt,
    centerOn,
    zoom,
    zoomByFactor,
    fit,
  }
}

/** A caixa que contém todas as peças, em quadrados (o canto de cada peça mais o
 *  corpo dela: uma Colossal ocupa 6×6, p107). */
export function boundingBox(tokens: readonly BoardToken[]) {
  const xs = tokens.map((token) => token.x)
  const ys = tokens.map((token) => token.y)
  const rights = tokens.map((token) => token.x + Math.max(1, token.footprint) - 1)
  const bottoms = tokens.map((token) => token.y + Math.max(1, token.footprint) - 1)
  return {
    minX: Math.min(...xs),
    minY: Math.min(...ys),
    maxX: Math.max(...rights),
    maxY: Math.max(...bottoms),
  }
}

/** A peça aparece na janela? Num plano infinito, desenhar o que está fora seria
 *  desenhar o infinito. */
export function isVisible(
  token: BoardToken,
  window: { originX: number; originY: number; cols: number; rows: number },
): boolean {
  const side = Math.max(1, token.footprint)
  return (
    token.x + side > window.originX &&
    token.x < window.originX + window.cols &&
    token.y + side > window.originY &&
    token.y < window.originY + window.rows
  )
}

function clamp(value: number, min: number, max: number): number {
  return Math.min(max, Math.max(min, value))
}
