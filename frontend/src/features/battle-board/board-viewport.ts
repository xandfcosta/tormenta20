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

/**
 * A JANELA sobre o plano infinito (ALE-124).
 *
 * O tabuleiro não tem bordas; quem tem tamanho é isto aqui. A origem é o
 * quadrado do canto superior-esquerdo da vista e pode ser negativa — empurrar a
 * cena para a esquerda é andar com a janela, não acabar o mapa.
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
    const nextCols = Math.max(1, Math.floor(width / cellPx()))
    const nextRows = Math.max(1, Math.floor(height / cellPx()))
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
    setOriginX(Math.round(centerX - nextCols / 2))
    setOriginY(Math.round(centerY - nextRows / 2))
  }

  const pan = (dxSquares: number, dySquares: number) => {
    setOriginX((x) => x + dxSquares)
    setOriginY((y) => y + dySquares)
  }

  /** Põe o quadrado no CENTRO da janela, não no canto: quem pede para centralizar
   *  quer ver o que está em volta. */
  const centerOn = (x: number, y: number) => {
    setOriginX(x - Math.floor(cols() / 2))
    setOriginY(y - Math.floor(rows() / 2))
  }

  /** Aproxima ou afasta. Só mexe no tamanho do quadrado — manter o centro é
   *  trabalho da `measure`, que é quem descobre quantos quadrados passaram a
   *  caber. */
  const zoom = (deltaPx: number) => {
    setCellPx((current) => clamp(current + deltaPx, MIN_CELL_PX, MAX_CELL_PX))
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
    centerOn,
    zoom,
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
