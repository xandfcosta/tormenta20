import { For, Show, onCleanup } from 'solid-js'
import type { BoardSquare } from '@/shared/lib/engine-wasm'
import type { BoardState, BoardToken, PendingMove } from '@/shared/realtime/realtime'
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
  /**
   * As casas que a peça selecionada ALCANÇA (T20 p238, medidas pelo motor).
   * Ausente = ninguém está medindo movimento, e todo quadrado aceita a peça —
   * é o caso do mestre, que posiciona sem orçamento.
   */
  reachable?: readonly BoardSquare[]
  /** O movimento proposto, desenhado para a mesa inteira ver. */
  pending?: PendingMove | null
  /** Peças que ESTE espectador pode pegar; as outras não respondem ao clique. */
  movableTokenIds?: ReadonlySet<string>
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
        {(place) => (
          <SquareLayer view={view()} onPlace={place()} reachable={props.reachable} />
        )}
      </Show>

      <Show when={props.pending}>{(move) => <PendingPath move={move()} view={view()} />}</Show>

      <For each={props.board.tokens.filter((token) => isVisible(token, window()))}>
        {(token) => (
          <TokenPiece
            token={token}
            view={view()}
            selected={props.selectedTokenId === token.id}
            movable={props.movableTokenIds?.has(token.id) ?? true}
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
function SquareLayer(props: {
  view: BoardViewport
  onPlace: (x: number, y: number) => void
  reachable?: readonly BoardSquare[]
}) {
  const squares = () => {
    const { cols, rows, originX, originY } = props.view
    return Array.from({ length: cols() * rows() }, (_, index) => ({
      x: originX() + (index % cols()),
      y: originY() + Math.floor(index / cols()),
    }))
  }

  // Um `Set` de chaves e não um `some` por quadrado: a janela tem centenas de
  // casas e o alcance dezenas, e o produto dos dois seria refeito a cada
  // reconciliação.
  const lit = () =>
    props.reachable && new Set(props.reachable.map((square) => `${square.x},${square.y}`))
  const canPlace = (x: number, y: number) => {
    const acesas = lit()
    return !acesas || acesas.has(`${x},${y}`)
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
            // A casa alcançável ACENDE, e a que passa do deslocamento não
            // responde: com a diagonal dobrada (T20 p238) o alcance é um
            // LOSANGO, e a forma ensina a regra sem ninguém explicar (ALE-124).
            // OURO e não `--primary`: nesta paleta o primary é o vermelho de
            // sangue, e um losango vermelho lê como "proibido" — exatamente o
            // contrário do que a casa acesa diz. Ouro é o vocabulário da vez na
            // iniciativa e do caminho proposto, que é a mesma conversa.
            class={cn(
              'cursor-pointer hover:bg-[color:var(--grimorio-gold)]/15',
              canPlace(square.x, square.y) &&
                lit() &&
                'bg-[color:var(--grimorio-gold)]/12 inset-ring-1 inset-ring-[color:var(--grimorio-gold)]/30',
              !canPlace(square.x, square.y) && 'cursor-default hover:bg-transparent',
            )}
            aria-label={`Coluna ${square.x}, linha ${square.y}`}
            disabled={!canPlace(square.x, square.y)}
            onClick={() => props.onPlace(square.x, square.y)}
          />
        )}
      </For>
    </div>
  )
}

/**
 * O caminho proposto, desenhado para a MESA inteira (ALE-124).
 *
 * SVG e não uma fileira de nós: a linha é uma coisa só, acompanha o zoom sem
 * recalcular nada e não entra na árvore acessível — quem lê o caminho por
 * leitor de tela lê o texto da barra de confirmação, que diz o custo em
 * quadrados, e não uma sequência de coordenadas.
 */
function PendingPath(props: { move: PendingMove; view: BoardViewport }) {
  const points = () => {
    const cell = props.view.cellPx()
    return props.move.path
      .map((square) => {
        const x = (square.x - props.view.originX()) * cell + cell / 2
        const y = (square.y - props.view.originY()) * cell + cell / 2
        return `${x},${y}`
      })
      .join(' ')
  }
  const destination = () => props.move.path[props.move.path.length - 1]

  return (
    <svg class="pointer-events-none absolute inset-0 size-full" aria-hidden="true">
      <polyline
        points={points()}
        fill="none"
        stroke="var(--grimorio-gold)"
        stroke-width="3"
        stroke-linejoin="round"
        stroke-dasharray="6 4"
      />
      <rect
        x={(destination().x - props.view.originX()) * props.view.cellPx()}
        y={(destination().y - props.view.originY()) * props.view.cellPx()}
        width={props.view.cellPx()}
        height={props.view.cellPx()}
        fill="var(--grimorio-gold)"
        opacity="0.25"
      />
    </svg>
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
  /** Este espectador pode pegar esta peça agora. */
  movable: boolean
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
        (!props.onSelect || !props.movable) && 'pointer-events-none',
      )}
      style={{ ...box(), background: hueGradient(props.token.label, 0.55, 0.15) }}
      // O nome acessível diz QUEM e ONDE, com o número que o servidor guarda:
      // num plano infinito a coordenada pode ser negativa, e traduzi-la para
      // "coluna 1" seria mentir sobre onde a peça está.
      aria-label={`${props.token.label}, coluna ${props.token.x}, linha ${props.token.y}`}
      aria-pressed={props.selected}
      disabled={!props.onSelect || !props.movable}
      onClick={() => props.onSelect?.(props.token.id)}
    >
      <span aria-hidden="true">{initials(props.token.label)}</span>
    </button>
  )
}
