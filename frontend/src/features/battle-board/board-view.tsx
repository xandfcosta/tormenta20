import { For, Show } from 'solid-js'
import type { BoardState, BoardToken } from '@/shared/realtime/realtime'
import { hueGradient } from '@/shared/lib/hue-from-name'
import { initials } from '@/shared/lib/initials'
import { cn } from '@/shared/lib/utils'
import { TERRAIN_STYLE } from './board-terrain'

/**
 * O menor quadrado que ainda se acerta com o dedo. Não são os 44px do alvo
 * tocável ideal de propósito: com 44, um tabuleiro de 20 colunas exigiria 880px
 * e até o tablet passaria a sessão rolando. 28 é o número medido — é o que faz
 * um tabuleiro de 20 colunas caber inteiro na coluna de 576px que a cena dá a
 * 1024px, e ainda dá um alvo utilizável no celular, onde ele rola dentro da
 * própria região. Some na fatia do zoom (ALE-124).
 */
const MIN_CELL_PX = 28

/**
 * O tabuleiro tático desenhado (ALE-124).
 *
 * DOM, não canvas: o servidor limita a sessão a 50 combatentes, uma mesa real
 * tem ~20 peças, e cada peça é um `<button>` de verdade com nome acessível e
 * `aria-pressed` — coisa que num canvas teria de ser reconstruída num DOM
 * espelho invisível. A grade em si não custa nó nenhum: é um
 * `repeating-linear-gradient` de fundo.
 *
 * A geometria é PORCENTAGEM da grade, não pixel: o quadrado é a unidade da
 * regra (T20 p236, 1 quadrado = 1,5m), e uma posição em pixel faria o celular e
 * o desktop discordarem sobre onde o ogro está. Também é o que deixa o tamanho
 * do quadrado se ajustar à região sem medir DOM nenhum — e é por isso que o
 * teste consegue afirmar posição sem browser.
 *
 * @example <BoardView board={board()} onSelectToken={select} selectedTokenId={id()} />
 */
export function BoardView(props: {
  board: BoardState
  /** Ausente = ninguém seleciona nada (a vista do jogador nesta fatia). */
  onSelectToken?: (tokenId: string) => void
  selectedTokenId?: string | null
  /** Ausente para quem não posiciona peça — só o mestre, nesta fatia. */
  onPlaceToken?: (x: number, y: number) => void
  /** Peça cuja linha está na vez: o anel dourado é o mesmo sinal da iniciativa. */
  activeEntryId?: string | null
}) {
  const cols = () => props.board.cols
  const rows = () => props.board.rows

  return (
    // `m-auto` no filho e não `items-center` no pai: com centralização por flex,
    // um conteúdo maior que a caixa tem a BORDA ESQUERDA cortada e ninguém
    // alcança a primeira coluna nem rolando.
    <div class="min-h-0 min-w-0 flex-1 overflow-auto p-1">
      <div
        class={cn(
          'relative m-auto rounded-sm border border-grimorio-iron',
          TERRAIN_STYLE[props.board.terrain] ?? TERRAIN_STYLE.pedra,
        )}
        style={{
          'aspect-ratio': `${cols()} / ${rows()}`,
          // Piso de MIN_CELL por quadrado: a 390px de largura, 20 colunas dariam
          // 18px e ninguém acerta um quadrado desses com o dedo. Abaixo do piso o
          // tabuleiro ROLA dentro da própria região — a página continua sem rolar.
          width: `max(calc(${cols()} * ${MIN_CELL_PX}px), min(100%, calc((100cqh - 0.5rem) * ${cols() / rows()})))`,
          'background-size': `calc(100% / ${cols()}) calc(100% / ${rows()})`,
        }}
        role="grid"
        aria-label={`Tabuleiro: ${props.board.place}, ${cols()} por ${rows()} quadrados`}
      >
        <Show when={props.onPlaceToken}>
          {(place) => <SquareLayer cols={cols()} rows={rows()} onPlace={place()} />}
        </Show>

        <For each={props.board.tokens}>
          {(token) => (
            <TokenPiece
              token={token}
              cols={cols()}
              rows={rows()}
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
    </div>
  )
}

/**
 * Os quadrados clicáveis, para o mestre pousar a peça selecionada. Só existem
 * quando alguém pode posicionar — sem isso seriam 300 botões inertes na árvore,
 * e no leitor de tela um campo minado de "botão, botão, botão".
 */
function SquareLayer(props: { cols: number; rows: number; onPlace: (x: number, y: number) => void }) {
  const squares = () =>
    Array.from({ length: props.cols * props.rows }, (_, index) => ({
      x: index % props.cols,
      y: Math.floor(index / props.cols),
    }))

  return (
    <div
      class="absolute inset-0 grid"
      style={{
        'grid-template-columns': `repeat(${props.cols}, minmax(0, 1fr))`,
        'grid-template-rows': `repeat(${props.rows}, minmax(0, 1fr))`,
      }}
    >
      <For each={squares()}>
        {(square) => (
          <button
            type="button"
            class="cursor-pointer hover:bg-[color:var(--primary)]/15"
            aria-label={`Coluna ${square.x + 1}, linha ${square.y + 1}`}
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
 *
 * O rosto é o mesmo gradiente determinístico por nome que o app usa nas outras
 * telas: mesmo goblin, mesma cor, em qualquer sessão.
 */
function TokenPiece(props: {
  token: BoardToken
  cols: number
  rows: number
  selected: boolean
  onTurn: boolean
  onSelect?: (tokenId: string) => void
}) {
  const side = () => Math.max(1, props.token.footprint)
  const box = () => ({
    left: `${(props.token.x / props.cols) * 100}%`,
    top: `${(props.token.y / props.rows) * 100}%`,
    width: `${(side() / props.cols) * 100}%`,
    height: `${(side() / props.rows) * 100}%`,
  })

  return (
    <button
      type="button"
      class={cn(
        'absolute flex items-center justify-center rounded-full border-2 p-0.5 text-[0.6rem] font-semibold uppercase text-white transition-shadow',
        props.onTurn ? 'border-grimorio-gold shadow-[0_0_0_3px_var(--grimorio-gold)]' : 'border-black/40',
        props.selected && 'ring-2 ring-[color:var(--primary)] ring-offset-1 ring-offset-black/40',
        !props.onSelect && 'pointer-events-none',
      )}
      style={{
        ...box(),
        background: hueGradient(props.token.label, 0.55, 0.15),
      }}
      // O nome acessível diz QUEM e ONDE: num tabuleiro, a posição é metade da
      // informação, e o quadrado é o que a mesa fala em voz alta.
      aria-label={`${props.token.label}, coluna ${props.token.x + 1}, linha ${props.token.y + 1}`}
      aria-pressed={props.selected}
      disabled={!props.onSelect}
      onClick={() => props.onSelect?.(props.token.id)}
    >
      <span aria-hidden="true">{initials(props.token.label)}</span>
    </button>
  )
}
