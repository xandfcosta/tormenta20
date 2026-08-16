import { For, Show, onCleanup } from 'solid-js'
import type { BoardState, BoardToken } from '@/shared/realtime/realtime'
import { hueGradient } from '@/shared/lib/hue-from-name'
import { initials } from '@/shared/lib/initials'
import { cn } from '@/shared/lib/utils'
import { TERRAIN_STYLE } from './board-terrain'
import { type BoardViewport, isVisible } from './board-viewport'

/**
 * O tabuleiro tático desenhado (ALE-124).
 *
 * O plano é INFINITO: o que se desenha é a JANELA que o `BoardViewport` define,
 * e a peça fora dela não vira nó nenhum — desenhar o que está fora seria
 * desenhar o infinito. Coordenada negativa é lugar legítimo, então o rótulo usa
 * o número COM SINAL que o servidor guarda: num plano sem bordas, o "+1" de
 * planilha mente sobre onde a peça está.
 *
 * DOM e não canvas: o servidor limita a sessão a 50 combatentes, e cada peça é
 * um `<button>` de verdade com nome acessível que diz quem e onde — num canvas
 * isso viraria um DOM espelho invisível, escrito duas vezes.
 *
 * @example <BoardView board={board()} view={viewport} onPlaceToken={place} />
 */
export function BoardView(props: {
  board: BoardState
  view: BoardViewport
  /** Ausente = ninguém seleciona nada (a vista do jogador nesta fatia). */
  onSelectToken?: (tokenId: string) => void
  selectedTokenId?: string | null
  /** Ausente para quem não posiciona peça — só o mestre, nesta fatia. */
  onPlaceToken?: (x: number, y: number) => void
  /** Peça cuja linha está na vez: o anel dourado é o mesmo sinal da iniciativa. */
  activeEntryId?: string | null
}) {
  const view = () => props.view
  const window = () => ({
    originX: view().originX(),
    originY: view().originY(),
    cols: view().cols(),
    rows: view().rows(),
  })

  let host: HTMLDivElement | undefined
  const observe = (element: HTMLDivElement) => {
    host = element
    if (typeof ResizeObserver === 'undefined') return
    const watcher = new ResizeObserver(() => {
      if (host) view().measure(host.clientWidth, host.clientHeight)
    })
    watcher.observe(element)
    onCleanup(() => watcher.disconnect())
  }

  return (
    <div
      ref={observe}
      class={cn(
        // `min-h-48` e não só `flex-1`: num pai sem altura definida (o rail do
        // jogador é um bloco que rola) o `flex-1` resolve para ZERO e o plano
        // some — a peça chega pelo socket e não há grade para desenhá-la. Um
        // tabuleiro nunca pode ser uma caixa de altura zero (ALE-124).
        'relative min-h-48 min-w-0 flex-1 overflow-hidden',
        TERRAIN_STYLE[props.board.terrain] ?? TERRAIN_STYLE.pedra,
      )}
      style={{
        // A grade é FUNDO, não nós: um `repeating-linear-gradient` cobre a
        // janela inteira em zero elementos, e o deslocamento da origem entra
        // como `background-position` — o número nunca cresce com o pan.
        'background-size': `${view().cellPx()}px ${view().cellPx()}px`,
        'background-position': `${-view().originX() * view().cellPx()}px ${-view().originY() * view().cellPx()}px`,
      }}
      role="grid"
      aria-label={`Tabuleiro: ${props.board.place}, janela em coluna ${view().originX()}, linha ${view().originY()}`}
    >
      <Show when={props.onPlaceToken}>
        {(place) => <SquareLayer view={view()} onPlace={place()} />}
      </Show>

      <For each={props.board.tokens.filter((token) => isVisible(token, window()))}>
        {(token) => (
          <TokenPiece
            token={token}
            view={view()}
            selected={props.selectedTokenId === token.id}
            onTurn={
              props.activeEntryId !== undefined &&
              props.activeEntryId !== null &&
              token.entryId === props.activeEntryId
            }
            onSelect={props.onSelectToken}
          />
        )}
      </For>
    </div>
  )
}

/**
 * Os quadrados clicáveis da janela, para o mestre pousar a peça selecionada. Só
 * existem quando alguém pode posicionar — sem isso seriam centenas de botões
 * inertes na árvore e um campo minado no leitor de tela.
 */
function SquareLayer(props: { view: BoardViewport; onPlace: (x: number, y: number) => void }) {
  const squares = () => {
    const { cols, rows, originX, originY } = props.view
    return Array.from({ length: cols() * rows() }, (_, index) => ({
      x: originX() + (index % cols()),
      y: originY() + Math.floor(index / cols()),
    }))
  }

  return (
    <div
      class="absolute inset-0 grid"
      style={{
        'grid-template-columns': `repeat(${props.view.cols()}, ${props.view.cellPx()}px)`,
        'grid-auto-rows': `${props.view.cellPx()}px`,
      }}
    >
      <For each={squares()}>
        {(square) => (
          <button
            type="button"
            class="cursor-pointer hover:bg-[color:var(--primary)]/15"
            aria-label={`Coluna ${square.x}, linha ${square.y}`}
            onClick={() => props.onPlace(square.x, square.y)}
          />
        )}
      </For>
    </div>
  )
}

/**
 * Uma peça. O tamanho vem do `footprint` em quadrados (T20 p107: Grande ocupa
 * 2×2, Enorme 3×3, Colossal 6×6), nunca de uma alça de escala livre — um
 * tabuleiro onde o rato fica maior que o dragão não responde "está ao alcance?".
 */
function TokenPiece(props: {
  token: BoardToken
  view: BoardViewport
  selected: boolean
  onTurn: boolean
  onSelect?: (tokenId: string) => void
}) {
  const side = () => Math.max(1, props.token.footprint)
  const box = () => {
    const cell = props.view.cellPx()
    return {
      left: `${(props.token.x - props.view.originX()) * cell}px`,
      top: `${(props.token.y - props.view.originY()) * cell}px`,
      width: `${side() * cell}px`,
      height: `${side() * cell}px`,
    }
  }

  return (
    <button
      type="button"
      class={cn(
        'absolute flex items-center justify-center rounded-full border-2 p-0.5 text-[0.6rem] font-semibold uppercase text-white transition-shadow',
        props.onTurn
          ? 'border-grimorio-gold shadow-[0_0_0_3px_var(--grimorio-gold)]'
          : 'border-black/40',
        props.selected && 'ring-2 ring-[color:var(--primary)] ring-offset-1 ring-offset-black/40',
        !props.onSelect && 'pointer-events-none',
      )}
      style={{ ...box(), background: hueGradient(props.token.label, 0.55, 0.15) }}
      // O nome acessível diz QUEM e ONDE, com o número que o servidor guarda:
      // num plano infinito a coordenada pode ser negativa, e traduzi-la para
      // "coluna 1" seria mentir sobre onde a peça está.
      aria-label={`${props.token.label}, coluna ${props.token.x}, linha ${props.token.y}`}
      aria-pressed={props.selected}
      disabled={!props.onSelect}
      onClick={() => props.onSelect?.(props.token.id)}
    >
      <span aria-hidden="true">{initials(props.token.label)}</span>
    </button>
  )
}
