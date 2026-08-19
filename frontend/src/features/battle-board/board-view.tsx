import { For, Index, Show, onCleanup } from 'solid-js'
import type { BoardSquare } from '@/shared/lib/engine-wasm'
import type { BoardMarker, BoardState, BoardToken, PendingMove } from '@/shared/realtime/realtime'
import { cn } from '@/shared/lib/utils'
import { hpFillVar } from '@/shared/ui/vital-bar'
import { createPrefersReducedMotion } from '@/shared/lib/media-query'
import { TERRAIN_AMBIENCE, TERRAIN_STYLE, gridLinesFor } from './board-terrain'
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
   * Quanto de PV resta a cada linha da iniciativa, em porcentagem (ALE-188).
   * Ausente para quem não tem número — inclusive para o JOGADOR quando o mestre
   * ocultou os PV, e é assim que a redação por papel chega até a peça.
   */
  health?: ReadonlyMap<string, number>
  /** Peça cuja linha da iniciativa está sob o ponteiro (ALE-189): ela ACENDE,
   *  para o mestre parar de procurar o ogro entre nove peças com a mesa
   *  esperando. */
  highlightEntryId?: string | null
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
  /** As casas cobertas pelo gabarito de área, quando há um (T20 p225). */
  area?: readonly BoardSquare[]
  /** Ausente = os marcadores são só desenho (é a vista do jogador, e o lugar
   *  apontado não é para ele mexer). */
  onSelectMarker?: (markerId: string) => void
  selectedMarkerId?: string | null
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

      {/* O ambiente do lugar: UMA camada, entre o chão e tudo o mais. Ela não
          rola com a janela de propósito — é luz do lugar, não textura do
          terreno, e luz não anda quando a câmera anda (ALE-188). */}
      <Show when={TERRAIN_AMBIENCE[props.board.terrain]}>
        {(ambiente) => (
          <div
            aria-hidden="true"
            class="pointer-events-none absolute inset-0"
            style={{ 'background-image': ambiente() }}
          />
        )}
      </Show>

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

      {/* O gabarito vai ACIMA do terreno e ABAIXO das peças: ele é o que está
          acontecendo no chão, e esconder a peça que ele pega seria esconder a
          resposta que ele existe para dar. */}
      <AreaLayer view={view()} squares={props.area ?? []} />

      <Show when={props.pending}>{(move) => <PendingPath move={move()} view={view()} />}</Show>

      {/* Os LUGARES apontados (ALE-195). Vêm ANTES das peças na árvore: o
          marcador é chão, e a peça que pisa nele fica por cima. */}
      <Index each={props.board.markers ?? []}>
        {(marker) => (
          <Show when={isVisible({ x: marker().x, y: marker().y, footprint: 1 } as BoardToken, window())}>
            <MapMarker
              marker={marker()}
              view={view()}
              selected={props.selectedMarkerId === marker().id}
              onSelect={props.onSelectMarker}
            />
          </Show>
        )}
      </Index>

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
              highlighted={
                props.highlightEntryId !== undefined &&
                props.highlightEntryId !== null &&
                token().entryId === props.highlightEntryId
              }
              health={token().entryId ? props.health?.get(token().entryId ?? '') : undefined}
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
  const centro = (square: BoardSquare) => ({
    x: (square.x - props.view.originX()) * props.view.cellPx() + props.view.cellPx() / 2,
    y: (square.y - props.view.originY()) * props.view.cellPx() + props.view.cellPx() / 2,
  })
  const points = () => props.move.path.map((square) => `${centro(square).x},${centro(square).y}`).join(' ')
  const origem = () => centro(props.move.path[0] ?? { x: 0, y: 0 })
  const destino = () => centro(props.move.path[props.move.path.length - 1] ?? { x: 0, y: 0 })
  const meia = () => props.view.cellPx() / 2
  // Formigas marchando, e elas ficam SOB o `prefers-reduced-motion` por conta
  // própria: o bloco global de CSS que zera animação não alcança SMIL (ALE-188).
  const parado = createPrefersReducedMotion()

  return (
    <svg class="pointer-events-none absolute inset-0 size-full" aria-hidden="true">
      {/* De ONDE a peça saiu: confirmada a proposta ela some, e sem a marca
          ninguém lembra do lugar de onde ela veio. */}
      <circle
        cx={origem().x}
        cy={origem().y}
        r={meia() * 0.55}
        fill="none"
        stroke="var(--grimorio-gold)"
        stroke-width="1.5"
        stroke-dasharray="3 3"
        opacity="0.7"
      />
      <polyline
        points={points()}
        fill="none"
        stroke="var(--grimorio-gold)"
        stroke-width="3"
        stroke-linejoin="round"
        stroke-dasharray="6 4"
      >
        <Show when={!parado()}>
          <animate attributeName="stroke-dashoffset" from="20" to="0" dur="1s" repeatCount="indefinite" />
        </Show>
      </polyline>
      {/* O destino no LOSANGO, que é o vocabulário do alcance nesta casa — o
          retângulo cheio dizia "casa pintada", que é outra coisa. */}
      <polygon
        points={`${destino().x},${destino().y - meia() * 0.8} ${destino().x + meia() * 0.8},${destino().y} ${destino().x},${destino().y + meia() * 0.8} ${destino().x - meia() * 0.8},${destino().y}`}
        fill="var(--grimorio-gold)"
        opacity="0.3"
        stroke="var(--grimorio-gold)"
        stroke-width="1.5"
      />
    </svg>
  )
}

/** As cores do marcador, do conjunto fechado que o servidor aceita. */
const MARKER_COLOR: Record<string, string> = {
  ouro: 'bg-grimorio-gold text-grimorio-parchment-ink',
  carmim: 'bg-[color:var(--primary)] text-white',
  azul: 'bg-[#3f6fb0] text-white',
  verde: 'bg-[#3f8f52] text-white',
}

/**
 * Um lugar marcado no mapa (ALE-195): duas letras num selo preso ao quadrado.
 *
 * Menor que a casa e encostado na quina, e isso é para ele NÃO parecer peça: o
 * marcador não ocupa quadrado, não é alvo e não entra na conta de área — quem
 * o confunde com criatura conta errado quem a bola de fogo pega.
 */
function MapMarker(props: {
  marker: BoardMarker
  view: BoardViewport
  selected: boolean
  onSelect?: (markerId: string) => void
}) {
  const lado = () => Math.max(14, props.view.cellPx() * 0.5)
  const canto = () => ({
    left: `${(props.marker.x - props.view.originX()) * props.view.cellPx() + 2}px`,
    top: `${(props.marker.y - props.view.originY()) * props.view.cellPx() + 2}px`,
    width: `${lado()}px`,
    height: `${lado()}px`,
    'font-size': `${Math.max(8, lado() * 0.5)}px`,
  })

  return (
    <button
      type="button"
      {...{ [TOKEN_PIECE_ATTR]: '' }}
      class={cn(
        'absolute grid place-items-center rounded-[3px] font-heading font-semibold uppercase leading-none shadow-[0_1px_2px_oklch(0_0_0/0.5)]',
        MARKER_COLOR[props.marker.color] ?? MARKER_COLOR.ouro,
        // Escondido: só o mestre o recebe, e ele precisa ver que a mesa não vê.
        props.marker.hidden && 'opacity-60 outline-1 outline-dashed outline-white/70',
        props.selected && 'ring-2 ring-white',
        !props.onSelect && 'pointer-events-none',
      )}
      style={canto()}
      aria-label={`Marcador ${props.marker.text || 'sem rótulo'}, coluna ${props.marker.x}, linha ${props.marker.y}${
        props.marker.hidden ? ', escondido dos jogadores' : ''
      }`}
      aria-pressed={props.selected}
      disabled={!props.onSelect}
      onClick={() => props.onSelect?.(props.marker.id)}
    >
      {props.marker.text}
    </button>
  )
}

/**
 * As casas do gabarito de área (T20 p225). Camada de pintura, sem nós
 * clicáveis: quem responde ao clique continua sendo a `SquareLayer`.
 */
function AreaLayer(props: { view: BoardViewport; squares: readonly BoardSquare[] }) {
  return (
    <For each={props.squares}>
      {(casa) => (
        <div
          aria-hidden="true"
          class="pointer-events-none absolute bg-grimorio-gold/25 ring-1 ring-inset ring-grimorio-gold/40"
          style={{
            left: `${(casa.x - props.view.originX()) * props.view.cellPx()}px`,
            top: `${(casa.y - props.view.originY()) * props.view.cellPx()}px`,
            width: `${props.view.cellPx()}px`,
            height: `${props.view.cellPx()}px`,
          }}
        />
      )}
    </For>
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
  /** A linha desta peça está sob o ponteiro na iniciativa (ALE-189). */
  highlighted: boolean
  /** PV restante em porcentagem, quando esta peça tem linha na iniciativa e
   *  quem está olhando pode ver os números (ALE-188). */
  health?: number
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
        // APONTADA da lista: um terceiro sinal, que não pode colidir com os dois
        // que já existem — o ouro é a VEZ e o erguer é a peça na mão (ALE-179).
        // Contorno com folga, desenhado FORA da caixa: ele não muda o tamanho da
        // peça nem disputa a borda, e some no `mouseleave`.
        props.highlighted && 'outline-2 outline-offset-2 outline-white',
        // Escondida: o mestre é o ÚNICO que a recebe, e até agora ela era
        // idêntica a uma peça visível — a emboscada dependia de ele lembrar de
        // cabeça quem estava escondido (ALE-178, ALE-179).
        props.token.hidden && 'border-dashed opacity-55',
        // CAÍDA a 0 PV: a peça perde a cor e o monograma deita. O mestre olha o
        // MAPA para decidir o foco do turno, e até agora precisava voltar os
        // olhos para a lista só para saber quem já está no chão (ALE-188).
        props.health === 0 && 'saturate-0 opacity-70',
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
      // segredo da emboscada — e o mesmo vale para quem caiu, que é a peça que
      // o mestre precisa parar de atacar (ALE-188). O FERIDO fica só no filete:
      // anunciar "ferido" em toda peça machucada encheria a leitura de uma
      // informação que a iniciativa já dá com número exato.
      aria-label={`${props.token.label}, coluna ${props.token.x}, linha ${props.token.y}${
        props.token.hidden ? ', escondida dos jogadores' : ''
      }${props.health === 0 ? ', caída' : ''}`}
      aria-pressed={props.selected}
      disabled={!props.onSelect || !props.movable}
      onClick={() => props.onSelect?.(props.token.id)}
    >
      <Show when={showsMonogram()}>
        <span
          aria-hidden="true"
          class={cn(props.health === 0 && 'rotate-90')}
          style={{ 'text-shadow': '0 1px 2px oklch(0 0 0/0.7)' }}
        >
          {look().monogram}
        </span>
      </Show>

      {/* O FILETE de PV: 3px no rodapé da peça, com a mesma cor que a barra da
          ficha e da iniciativa usam na mesma fração (`hpFillVar`) — uma segunda
          régua de "quão mal" seria duas verdades sobre o mesmo personagem.
          Só aparece abaixo do cheio: a peça inteira já diz que está inteira, e
          um filete verde em nove peças é ruído em toda rodada. */}
      <Show when={props.health !== undefined && props.health < 100}>
        <span
          aria-hidden="true"
          class="absolute inset-x-0.5 bottom-0.5 h-[3px] overflow-hidden rounded-full bg-black/50"
        >
          <span
            class="block h-full rounded-full"
            style={{
              width: `${Math.max(0, props.health ?? 0)}%`,
              background: `var(${hpFillVar(props.health ?? 0)})`,
            }}
          />
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
