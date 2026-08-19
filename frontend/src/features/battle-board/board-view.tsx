import { For, Index, Show, onCleanup } from 'solid-js'
import type { BoardSquare } from '@/shared/lib/engine-wasm'
import type { BoardState, BoardToken, PendingMove } from '@/shared/realtime/realtime'
import { cn } from '@/shared/lib/utils'
import { TERRAIN_STYLE, gridLinesFor } from './board-terrain'
import { TOKEN_PIECE_ATTR, createBoardGestures } from './board-gestures'
import { type BoardViewport, isVisible } from './board-viewport'
import { tokenAppearance } from './token-appearance'

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
 * @example <BoardView board={board()} view={viewport} onSquareClick={place} />
 */
export function BoardView(props: {
  board: BoardState
  view: BoardViewport
  /** Ausente = ninguém seleciona nada (a vista do jogador nesta fatia). */
  onSelectToken?: (tokenId: string) => void
  selectedTokenId?: string | null
  /** O que um clique num QUADRADO faz — pousar a peça na mão, ou fixar uma
   *  ponta da régua. Ausente = a camada de casas nem existe, e com ela some um
   *  campo minado de centenas de botões no leitor de tela. */
  onSquareClick?: (x: number, y: number) => void
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
  /** As casas que custam o dobro (T20 p238). Público: chão acidentado é coisa
   *  que se vê, e a mesa precisa ver o mesmo mapa que a régua cobra. */
  difficult?: readonly BoardSquare[]
  /** Presente = ferramenta na mão: arrastar PINTA em vez de mover a vista. */
  onPaintSquare?: (x: number, y: number, secondary: boolean) => void
  /** As duas pontas da régua, quando a mesa está medindo (T20 p224). */
  ruler?: { from: BoardSquare; to: BoardSquare } | null
  /** O teclado da superfície (ALE-194): quem interpreta as teclas é a cena, que
   *  é quem sabe se há peça na mão e quem pode movê-la. */
  onKeyDown?: (event: KeyboardEvent) => void
}) {
  const view = () => props.view
  const window = () => ({
    originX: view().originX(),
    originY: view().originY(),
    cols: view().cols(),
    rows: view().rows(),
  })

  // Nasce UMA vez: guarda quais ponteiros estão no vidro entre um evento e o
  // seguinte (ALE-140).
  const gestures = createBoardGestures(view, () => props.onPaintSquare ?? null)

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
        // O CHÃO fica no hospedeiro e SEM `background-size`: é o que devolve ao
        // lugar o gradiente inteiro em vez de um ladrilho de 44px (ALE-179).
        TERRAIN_STYLE[props.board.terrain] ?? TERRAIN_STYLE.pedra,
      )}
      style={{
        // Só nesta superfície: sem isto, no telefone o arraste rola a PÁGINA em
        // vez de mover a vista, e o tabuleiro fica navegável só pelos botões.
        'touch-action': 'none',
      }}
      onPointerDown={gestures.onPointerDown}
      onPointerMove={gestures.onPointerMove}
      onPointerUp={gestures.onPointerUp}
      onPointerCancel={gestures.onPointerUp}
      onWheel={gestures.onWheel}
      onContextMenu={gestures.onContextMenu}
      onKeyDown={(event) => props.onKeyDown?.(event)}
      // A superfície é FOCÁVEL para o teclado ter onde acontecer: quem clica no
      // mapa e aperta uma seta espera que ela valha para o mapa, e sem foco a
      // tecla vai para o `body` e rola a página (ALE-194). É UM ponto de
      // tabulação antes das peças, que são botões e já vinham de graça.
      tabIndex={0}
      role="grid"
      // A coluna e a linha são arredondadas porque a origem passou a ser
      // fracionária com o arraste (ALE-140), e "janela em coluna −7,3125" não é
      // uma frase que se ouça.
      aria-label={`Tabuleiro: ${props.board.place}, janela em coluna ${Math.round(view().originX())}, linha ${Math.round(view().originY())}`}
    >
      {/* A grade é FUNDO, não nós: duas faixas cruzadas cobrem a janela inteira
          em ZERO elementos por quadrado, e o deslocamento da origem entra como
          `background-position` — o número nunca cresce com o pan. Camada
          própria desde a ALE-179, para o `background-size` do quadrado não
          ladrilhar também o chão. */}
      <div
        aria-hidden="true"
        class="pointer-events-none absolute inset-0"
        style={{
          'background-image': gridLinesFor(props.board.terrain),
          'background-size': `${view().cellPx()}px ${view().cellPx()}px`,
          'background-position': `${-view().originX() * view().cellPx()}px ${-view().originY() * view().cellPx()}px`,
        }}
      />

      <Show when={props.onSquareClick}>
        {(place) => (
          <SquareLayer
            view={view()}
            onPlace={place()}
            reachable={props.reachable}
            difficult={props.difficult}
          />
        )}
      </Show>

      <DifficultLayer view={view()} squares={props.difficult ?? []} />

      <Show when={props.pending}>{(move) => <PendingPath move={move()} view={view()} />}</Show>

      <Show when={props.ruler}>{(linha) => <RulerLine ends={linha()} view={view()} />}</Show>

      {/* `Index` sobre a lista INTEIRA do servidor, com a visibilidade num
          `Show` por peça — e as duas metades desta linha são consertos.
          O `For` reconcilia por REFERÊNCIA e todo broadcast troca o estado
          inteiro: cada peça virava um botão NOVO a cada mensagem da mesa e o
          foco caía no `body` (com o teclado da ALE-194 isso aparecia como "a
          primeira seta move e a segunda não faz nada"; para quem usa leitor de
          tela, era perder o lugar sempre que qualquer peça se mexia).
          E o `Index` sobre a lista FILTRADA tinha o defeito espelhado, medido no
          browser: como quem ocupa cada índice muda quando a janela anda, o foco
          escorregava para OUTRA peça em silêncio. Sobre a lista inteira o índice
          é o do servidor, que não muda quando a vista anda. */}
      <Index each={props.board.tokens}>
        {(token) => (
          <Show when={isVisible(token(), window())}>
            <TokenPiece
              token={token()}
              view={view()}
              selected={props.selectedTokenId === token().id}
              movable={props.movableTokenIds?.has(token().id) ?? true}
              onTurn={
                props.activeEntryId !== undefined &&
                props.activeEntryId !== null &&
                token().entryId === props.activeEntryId
              }
              onSelect={props.onSelectToken}
            />
          </Show>
        )}
      </Index>
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
  difficult?: readonly BoardSquare[]
}) {
  // A origem é fracionária desde o arraste (ALE-140), mas casa é coisa inteira:
  // a camada começa no quadrado de baixo e se desloca pelo RESTO, com uma
  // coluna e uma linha a mais para cobrir a fresta que o deslocamento abre.
  const first = () => ({ x: Math.floor(props.view.originX()), y: Math.floor(props.view.originY()) })
  const cols = () => props.view.cols() + 1
  const rows = () => props.view.rows() + 1
  const squares = () =>
    Array.from({ length: cols() * rows() }, (_, index) => ({
      x: first().x + (index % cols()),
      y: first().y + Math.floor(index / cols()),
    }))

  // Um `Set` de chaves e não um `some` por quadrado: a janela tem centenas de
  // casas e o alcance dezenas, e o produto dos dois seria refeito a cada
  // reconciliação.
  const lit = () =>
    props.reachable && new Set(props.reachable.map((square) => `${square.x},${square.y}`))
  const hardKeys = () =>
    props.difficult && new Set(props.difficult.map((square) => `${square.x},${square.y}`))
  const canPlace = (x: number, y: number) => {
    const acesas = lit()
    return !acesas || acesas.has(`${x},${y}`)
  }

  return (
    <div
      class="absolute inset-0 grid"
      style={{
        'grid-template-columns': `repeat(${cols()}, ${props.view.cellPx()}px)`,
        'grid-auto-rows': `${props.view.cellPx()}px`,
        transform: `translate(${(first().x - props.view.originX()) * props.view.cellPx()}px, ${(first().y - props.view.originY()) * props.view.cellPx()}px)`,
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
            aria-label={`Coluna ${square.x}, linha ${square.y}${
              hardKeys()?.has(`${square.x},${square.y}`) ? ', terreno difícil' : ''
            }`}
            disabled={!canPlace(square.x, square.y)}
            onClick={() => props.onPlace(square.x, square.y)}
          />
        )}
      </For>
    </div>
  )
}

/**
 * O chão que custa o dobro (T20 p238), desenhado para a mesa INTEIRA — inclusive
 * para quem não posiciona peça, porque terreno acidentado é coisa que se vê e o
 * jogador precisa enxergar o que a régua vai lhe cobrar (ALE-124, fatia 4).
 *
 * Camada só de pintura, sem nós clicáveis: quem clica é a `SquareLayer`, que só
 * existe para quem pode agir.
 *
 * NÃO é hachura diagonal, e isso foi corrigido depois de ver na tela: listras
 * na diagonal são o vocabulário universal de "desabilitado, não passe", e
 * terreno difícil é o contrário disso — passa-se, só que devagar. O que ficou é
 * chão MOLE: um borrão quente por baixo e tufos claros espalhados, que leem
 * como lama e mato em vez de fita de obra.
 */
function DifficultLayer(props: { view: BoardViewport; squares: readonly BoardSquare[] }) {
  const cell = () => props.view.cellPx()
  return (
    <div aria-hidden="true" class="pointer-events-none absolute inset-0">
      <For each={props.squares}>
        {(square) => (
          <div
            class="absolute bg-[radial-gradient(circle_at_28%_32%,rgba(255,255,255,.22)_0_1.5px,transparent_2px),radial-gradient(circle_at_68%_58%,rgba(255,255,255,.18)_0_1.5px,transparent_2px),radial-gradient(circle_at_45%_80%,rgba(255,255,255,.14)_0_1px,transparent_1.5px),radial-gradient(ellipse_at_center,oklch(0.55_0.07_75/0.30),oklch(0.45_0.05_75/0.16))]"
            style={{
              left: `${(square.x - props.view.originX()) * cell()}px`,
              top: `${(square.y - props.view.originY()) * cell()}px`,
              width: `${cell()}px`,
              height: `${cell()}px`,
            }}
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
 * A linha da régua (ALE-124, fatia 6). Reta e contínua, ao contrário do caminho
 * proposto, que é tracejado e dobra pelos quadrados: são coisas diferentes —
 * um é por onde a peça VAI, a outra é a distância até onde se quer acertar.
 */
function RulerLine(props: { ends: { from: BoardSquare; to: BoardSquare }; view: BoardViewport }) {
  const centro = (square: BoardSquare) => ({
    x: (square.x - props.view.originX()) * props.view.cellPx() + props.view.cellPx() / 2,
    y: (square.y - props.view.originY()) * props.view.cellPx() + props.view.cellPx() / 2,
  })

  return (
    <svg class="pointer-events-none absolute inset-0 size-full" aria-hidden="true">
      <line
        x1={centro(props.ends.from).x}
        y1={centro(props.ends.from).y}
        x2={centro(props.ends.to).x}
        y2={centro(props.ends.to).y}
        stroke="var(--grimorio-gold)"
        stroke-width="2"
      />
      <circle cx={centro(props.ends.from).x} cy={centro(props.ends.from).y} r="4" fill="var(--grimorio-gold)" />
      <circle
        cx={centro(props.ends.to).x}
        cy={centro(props.ends.to).y}
        r="6"
        fill="none"
        stroke="var(--grimorio-gold)"
        stroke-width="2"
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
  const sizePx = () => side() * props.view.cellPx()
  const look = () => tokenAppearance(props.token.label)
  const box = () => {
    const cell = props.view.cellPx()
    return {
      left: `${(props.token.x - props.view.originX()) * cell}px`,
      top: `${(props.token.y - props.view.originY()) * cell}px`,
      width: `${sizePx()}px`,
      height: `${sizePx()}px`,
    }
  }

  // O monograma some quando a peça fica pequena demais para ele: duas letras
  // espremidas num disco de 20px viram mancha, e o nome acessível continua no
  // botão de qualquer forma.
  const showsMonogram = () => sizePx() >= 26
  // Tipo derivado da CAIXA, e não fixo: o mesmo 9,6px servia o disco de 20px e
  // a Colossal de 576px (6×6 quadrados no zoom máximo), sussurrando nos dois.
  const fontPx = () => Math.min(44, Math.max(10, sizePx() * 0.3))

  return (
    <button
      type="button"
      // Marca que o arraste consulta: começar o gesto NA PEÇA não move a vista.
      {...{ [TOKEN_PIECE_ATTR]: '' }}
      class={cn(
        'absolute flex items-center justify-center border-2 p-0.5 font-heading font-semibold uppercase leading-none text-white transition-[transform,box-shadow]',
        // A FORMA é canal de informação, e canal que não é cor: varrer o
        // tabuleiro e separar "nós" de "eles" sem ler rótulo é o que o mestre
        // faz vinte vezes por rodada. Objeto não é criatura — a porta e o baú
        // saem do vocabulário de matiz e ficam em ferro (ALE-179).
        props.token.kind === 'character' && 'rounded-full',
        props.token.kind === 'npc' && 'rounded-[3px]',
        props.token.kind === 'object' &&
          'rounded-none border-grimorio-iron-light bg-[var(--grimorio-panel)] text-muted-foreground',
        // Brilho no topo e sombra de assento: é o que faz o disco ler como peça
        // pousada na mesa em vez de círculo de dashboard.
        props.onTurn
          ? 'border-grimorio-gold shadow-[0_0_0_3px_var(--grimorio-gold)]'
          : 'border-grimorio-iron shadow-[inset_0_1px_0_oklch(1_0_0/0.18),0_1px_2px_oklch(0_0_0/0.5)]',
        // SELECIONADA se ERGUE, e isso não é enfeite: o anel era
        // `--primary`, que nesta paleta é o carmim de sangue — a mesma cor que
        // o código já tinha recusado para a casa acesa porque "lê como
        // proibido". Com a peça na mão, o próximo clique MOVE; o estado mais
        // perigoso do tabuleiro merece o sinal mais claro, e geometria não
        // disputa canal com o ouro da vez.
        props.selected && '-translate-y-0.5 scale-105 shadow-lg shadow-black/60 ring-2 ring-white/80',
        // Escondida: o mestre é o ÚNICO que a recebe, e até agora ela era
        // idêntica a uma peça visível — a emboscada dependia de ele lembrar de
        // cabeça quem estava escondido (ALE-178, ALE-179).
        props.token.hidden && 'border-dashed opacity-55',
        (!props.onSelect || !props.movable) && 'pointer-events-none',
      )}
      style={{
        ...box(),
        ...(props.token.kind === 'object' ? {} : { background: look().background }),
        'font-size': `${fontPx()}px`,
      }}
      // O nome acessível diz QUEM e ONDE, com o número que o servidor guarda:
      // num plano infinito a coordenada pode ser negativa, e traduzi-la para
      // "coluna 1" seria mentir sobre onde a peça está. E diz também o que só a
      // aparência dizia: quem lê por leitor de tela tem o mesmo direito ao
      // segredo da emboscada.
      aria-label={`${props.token.label}, coluna ${props.token.x}, linha ${props.token.y}${
        props.token.hidden ? ', escondida dos jogadores' : ''
      }`}
      aria-pressed={props.selected}
      disabled={!props.onSelect || !props.movable}
      onClick={() => props.onSelect?.(props.token.id)}
    >
      <Show when={showsMonogram()}>
        <span aria-hidden="true" style={{ 'text-shadow': '0 1px 2px oklch(0 0 0/0.7)' }}>
          {look().monogram}
        </span>
      </Show>
      {/* O número da instância em PERGAMINHO: "eu ataco o Zumbi 3" precisa de
          resposta num relance, e o monograma sozinho não distingue os três. */}
      <Show when={look().instance && showsMonogram()}>
        <span
          aria-hidden="true"
          class="absolute -bottom-0.5 -right-0.5 grid place-items-center rounded-full bg-grimorio-parchment font-mono text-grimorio-parchment-ink"
          style={{
            width: `${Math.max(12, sizePx() * 0.34)}px`,
            height: `${Math.max(12, sizePx() * 0.34)}px`,
            'font-size': `${Math.max(8, sizePx() * 0.22)}px`,
          }}
        >
          {look().instance}
        </span>
      </Show>
    </button>
  )
}
