import { Brush, Check, Crosshair, LayoutGrid, Minus, Plus, Undo2, Users, X } from 'lucide-solid'
import { For, Show, createMemo, createSignal } from 'solid-js'
import { pathBetween } from '@/features/battle-board/board-path'
import { TokenActions } from '@/features/battle-board/token-actions'
import { TokenDialog } from '@/features/battle-board/token-dialog'
import { BoardView } from '@/features/battle-board/board-view'
import { type BoardViewport, SQUARE_METRES } from '@/features/battle-board/board-viewport'
import { OpenBoardDialog } from '@/features/battle-board/open-board-dialog'
import { upcomingTurns } from '@/features/session-tracker/tracker-rules'
import { boardReach } from '@/shared/lib/engine-wasm'
import type {
  BoardState,
  BoardToken,
  InitiativeEntry,
  SessionRealtime,
} from '@/shared/realtime/realtime'
import { cn } from '@/shared/lib/utils'
import { Button } from '@/shared/ui/button'
import { ConfirmDialog } from '@/shared/ui/confirm-dialog'

/**
 * O tabuleiro dentro da cena da sessão (ALE-124).
 *
 * Composição mora em `pages/`: liga o socket, a seleção que a iniciativa já usa
 * e os controles do mestre. A feature `battle-board` só desenha e avisa — ela
 * não conhece `session-tracker`, e é isso que mantém as dependências apontando
 * para baixo.
 *
 * É REGIÃO e não aba do workspace porque `Tabs` desmonta o conteúdo inativo: numa
 * aba, ir ao bestiário apagaria a seleção e o enquadramento — o mesmo defeito que
 * forçou o autosave das notas. Pela mesma razão a JANELA (origem e zoom) é dona
 * da página e chega por prop: ela precisa sobreviver à troca de região.
 */
export function BoardRegion(props: {
  rt: SessionRealtime
  isGm: boolean
  view: BoardViewport
  /** Linha da iniciativa na vez: a peça dela ganha o anel dourado. */
  activeEntryId?: string | null
  /** Os personagens DESTE espectador. Vazio para o mestre, que move qualquer um. */
  myCharacterIds?: ReadonlySet<number>
}) {
  const [selectedTokenId, setSelectedTokenId] = createSignal<string | null>(null)
  // O pincel é MODO e não gesto: enquanto ele está ligado, a casa clicada vira
  // terreno difícil em vez de receber a peça. Modo porque o clique numa casa já
  // tem dono (pousar), e porque um gesto com tecla não existe no toque.
  const [painting, setPainting] = createSignal(false)
  const board = () => props.rt.board()

  // A vez é do RASTREADOR: o tabuleiro pergunta, não guarda uma cópia — duas
  // cópias da vez divergiriam no primeiro turno passado com o tabuleiro fechado.
  const inCombat = () => props.rt.state().turnIndex >= 0
  const isMine = (token: BoardToken) =>
    token.characterId !== undefined && (props.myCharacterIds?.has(token.characterId) ?? false)

  // Espelho da regra do servidor (`assertMovable`), e só para AFFORDANCE: a
  // trava é do Go, e uma peça que não responde ao clique é UX, não segurança.
  const canMove = (token: BoardToken) =>
    props.isGm || (isMine(token) && (!inCombat() || token.entryId === props.activeEntryId))

  const movableTokenIds = createMemo(
    () => new Set((board()?.tokens ?? []).filter(canMove).map((token) => token.id)),
  )

  // Selecionar de novo a mesma peça DESSELECIONA: sem isso não há como largar a
  // peça sem posicioná-la, e o próximo clique num quadrado a moveria sem querer.
  const selectToken = (tokenId: string) =>
    setSelectedTokenId((current) => (current === tokenId ? null : tokenId))

  const selectedToken = () => board()?.tokens.find((token) => token.id === selectedTokenId())

  /**
   * As casas acesas. O mestre não tem nenhuma — ele posiciona sem orçamento, e
   * acender um losango para ele seria inventar um limite que o livro não lhe
   * impõe. Fora de combate também não há: ali não existe deslocamento de turno.
   */
  const reachable = createMemo(() => {
    const token = selectedToken()
    if (!token || props.isGm || !inCombat() || !token.speedSquares) return undefined
    // O losango passa a CONTORNAR o brejo sozinho: é o mesmo motor que o
    // servidor usa para cobrar o caminho (T20 p238).
    return boardReach({ x: token.x, y: token.y }, board()?.difficult ?? [], token.speedSquares)
  })

  /**
   * Pousar a peça. São dois caminhos porque são duas coisas diferentes: o
   * mestre POSICIONA (voo, empurrão, "pode ir"), e o jogador PROPÕE um
   * movimento que a mesa vê e alguém confirma.
   */
  // De onde cada peça veio no último posicionamento DESTA tela. Memória local
  // de propósito: um histórico no servidor seria muito estado para um problema
  // que uma posição anterior resolve, e o `PendingMove` já é o desfazer do
  // jogador (ALE-178).
  const [ondeEstava, setOndeEstava] = createSignal<Record<string, { x: number; y: number }>>({})

  const onSquare = (x: number, y: number) => {
    if (painting()) {
      props.rt.paintTerrain(x, y)
      return
    }
    placeSelected(x, y)
  }

  const placeSelected = (x: number, y: number) => {
    const token = selectedToken()
    if (!token) return
    if (props.isGm) {
      setOndeEstava((atual) => ({ ...atual, [token.id]: { x: token.x, y: token.y } }))
      props.rt.updateToken(token.id, { x, y })
    } else {
      props.rt.proposeMove(token.id, pathBetween({ x: token.x, y: token.y }, { x, y }))
    }
    setSelectedTokenId(null)
  }

  /**
   * Quem decide o provisório: o mestre por qualquer um — é ele quem toca a mesa
   * quando o jogador travou ou caiu da rede — e o jogador só sobre a própria
   * peça. Quem não decide continua VENDO o caminho: é essa a razão de o
   * provisório ser estado e não um arraste privado.
   */
  const decidesPending = () => {
    const move = board()?.pending
    if (!move) return false
    const token = board()?.tokens.find((piece) => piece.id === move.tokenId)
    return props.isGm || (token !== undefined && isMine(token))
  }

  return (
    // `w-full flex-1`: no rail do jogador o cartão é filho de um flex, e sem
    // isso ele encolhe para o conteúdo — medido em 138px de 352 disponíveis,
    // ou seja, o tabuleiro virava uma tira estreita (ALE-124).
    <section class="@container flex w-full min-h-0 min-w-0 flex-1 flex-col rounded-sm border border-grimorio-iron bg-[var(--grimorio-panel)]">
      <Show when={board()} fallback={<EmptyBoard isGm={props.isGm} onOpen={props.rt.openBoard} />}>
        {(live) => (
          <>
            <header class="flex shrink-0 flex-wrap items-center gap-2 border-b border-grimorio-iron px-3 py-2">
              <h2 class="min-w-0 truncate font-heading text-sm uppercase tracking-wide text-grimorio-gold">
                {live().place}
              </h2>
              <span class="font-mono text-[11px] tabular-nums text-muted-foreground">
                {live().tokens.length} peças · 1 quadrado = {SQUARE_METRES.toFixed(1).replace('.', ',')}m
              </span>

              {/* `flex-wrap` aqui também: o cabeçalho já quebrava, mas esta
                  fileira de controles não, e a 390px "Trazer a iniciativa" saía
                  cortado com o ✕ de encerrar INALCANÇÁVEL fora da tela
                  (ALE-178). */}
              <div class="ml-auto flex flex-wrap items-center justify-end gap-1">
                <ViewControls view={props.view} onFit={() => props.view.fit(live().tokens)} />
                <Show when={props.isGm}>
                  <Button
                    size="sm"
                    variant={painting() ? 'default' : 'ghost'}
                    aria-pressed={painting()}
                    onClick={() => setPainting((ligado) => !ligado)}
                  >
                    <Brush aria-hidden="true" class="size-4" />
                    Terreno
                  </Button>
                  <Button
                    size="sm"
                    variant="secondary"
                    disabled={!props.rt.isConnected()}
                    onClick={props.rt.populateBoard}
                  >
                    <Users aria-hidden="true" class="size-4" />
                    Trazer a iniciativa
                  </Button>
                  {/* A peça avulsa que a iniciativa não traz: a porta, o baú, o
                      aliado sem turno. É o `kind: "object"` que o servidor
                      sempre soube guardar e que nunca tinha como nascer. */}
                  <TokenDialog
                    onSave={(peca) => props.rt.addToken({ ...peca, x: 0, y: 0 })}
                    trigger={(open) => (
                      <Button size="sm" variant="secondary" onClick={open}>
                        + Peça
                      </Button>
                    )}
                  />
                  <ConfirmDialog
                    title="Encerrar o tabuleiro?"
                    description="As peças e as posições desta cena se perdem. A iniciativa e os PV continuam como estão."
                    confirmLabel="Encerrar"
                    destructive
                    onConfirm={props.rt.closeBoard}
                    trigger={(open) => (
                      <Button
                        size="sm"
                        variant="ghost"
                        aria-label="Encerrar o tabuleiro"
                        onClick={open}
                      >
                        <X aria-hidden="true" class="size-4" />
                      </Button>
                    )}
                  />
                </Show>
              </div>
            </header>

            <TurnStrip rt={props.rt} hidden={props.isGm} />

            <BoardView
              board={live()}
              view={props.view}
              activeEntryId={props.activeEntryId}
              selectedTokenId={selectedTokenId()}
              movableTokenIds={movableTokenIds()}
              reachable={reachable()}
              pending={live().pending}
              // Quem não pode mover NENHUMA peça também não seleciona: sem isso
              // a camada de quadrados nasceria com centenas de botões inertes
              // na árvore de quem só assiste.
              onSelectToken={movableTokenIds().size > 0 ? selectToken : undefined}
              // Com o pincel ligado, TODA casa responde — não só quando há peça
              // na mão: o mestre está pintando o chão, não posicionando.
              onPlaceToken={painting() || selectedTokenId() ? onSquare : undefined}
              difficult={live().difficult}
            />

            <Show when={props.isGm && selectedToken()}>
              {(token) => (
                <TokenActions
                  token={token()}
                  onUpdate={(patch) => props.rt.updateToken(token().id, patch)}
                  onRemove={() => {
                    props.rt.removeToken(token().id)
                    setSelectedTokenId(null)
                  }}
                  onUndo={
                    ondeEstava()[token().id] &&
                    (() => {
                      const antes = ondeEstava()[token().id]
                      props.rt.updateToken(token().id, antes)
                      setOndeEstava(({ [token().id]: _, ...resto }) => resto)
                    })
                  }
                />
              )}
            </Show>

            <Show when={live().pending}>
              {(move) => (
                <MoveBar
                  move={move()}
                  canDecide={decidesPending()}
                  onConfirm={() => props.rt.commitMove(live().version)}
                  onCancel={props.rt.cancelMove}
                />
              )}
            </Show>

            {/* A dica só existe no momento em que ela é ACIONÁVEL: como linha
                permanente, ela custava 26px de altura em todo formato para dizer
                o óbvio, e no celular deitado isso é uma fileira inteira de
                quadrados a menos (ALE-124). */}
            <Show when={selectedTokenId() && !live().pending}>
              <p class="shrink-0 border-t border-grimorio-iron px-3 py-1 text-[11px] text-grimorio-gold">
                {props.isGm
                  ? 'Clique num quadrado para pousar a peça.'
                  : 'Clique numa casa acesa para propor o movimento.'}
              </p>
            </Show>
          </>
        )}
      </Show>
    </section>
  )
}

/**
 * A barra do movimento proposto (ALE-124).
 *
 * Diz o custo em QUADRADOS e em metros: quadrado é a unidade da regra (T20
 * p236) e metro é a unidade da conversa na mesa. E diz o orçamento ao lado,
 * porque "4" sem "de 6" não responde a pergunta que o jogador tem.
 *
 * Quem não decide continua lendo a barra: a mesa inteira vê para onde a peça
 * está indo, que é a razão de o provisório ser estado e não arraste privado.
 */
function MoveBar(props: {
  move: NonNullable<BoardState['pending']>
  canDecide: boolean
  onConfirm: () => void
  onCancel: () => void
}) {
  const metres = () => (props.move.cost * SQUARE_METRES).toFixed(1).replace('.', ',')

  return (
    <div class="flex shrink-0 flex-wrap items-center gap-2 border-t border-grimorio-iron px-3 py-1.5">
      <p class="font-mono text-[11px] tabular-nums text-grimorio-gold">
        {props.move.cost} {props.move.cost === 1 ? 'quadrado' : 'quadrados'} ({metres()}m)
        {props.move.budget >= 0 ? ` de ${props.move.budget}` : ' · sem limite de turno'}
      </p>
      <Show when={props.canDecide} fallback={<span class="text-[11px] text-muted-foreground">Aguardando confirmação.</span>}>
        <div class="ml-auto flex items-center gap-1">
          <Button size="sm" variant="ghost" onClick={() => props.onCancel()}>
            <Undo2 aria-hidden="true" class="size-4" />
            Refazer
          </Button>
          <Button size="sm" onClick={() => props.onConfirm()}>
            <Check aria-hidden="true" class="size-4" />
            Confirmar
          </Button>
        </div>
      </Show>
    </div>
  )
}

/**
 * Mover a vista e o zoom. Num plano INFINITO isto não é enfeite: sem uma forma
 * de andar com a janela, metade da cena fica inalcançável.
 *
 * "Centralizar" enquadra as PEÇAS, e não a origem: o centro de um plano infinito
 * não significa nada — o que o mestre quer é achar o grupo.
 */
function ViewControls(props: { view: BoardViewport; onFit: () => void }) {
  const step = () => Math.max(1, Math.floor(props.view.cols() / 3))

  return (
    <div class="flex items-center gap-0.5">
      <PanButton label="Mover a vista para a esquerda" onClick={() => props.view.pan(-step(), 0)}>
        ←
      </PanButton>
      <PanButton label="Mover a vista para cima" onClick={() => props.view.pan(0, -step())}>
        ↑
      </PanButton>
      <PanButton label="Mover a vista para baixo" onClick={() => props.view.pan(0, step())}>
        ↓
      </PanButton>
      <PanButton label="Mover a vista para a direita" onClick={() => props.view.pan(step(), 0)}>
        →
      </PanButton>
      <Button size="sm" variant="ghost" aria-label="Afastar" onClick={() => props.view.zoom(-8)}>
        <Minus aria-hidden="true" class="size-4" />
      </Button>
      <Button size="sm" variant="ghost" aria-label="Aproximar" onClick={() => props.view.zoom(8)}>
        <Plus aria-hidden="true" class="size-4" />
      </Button>
      <Button size="sm" variant="ghost" aria-label="Centralizar nas peças" onClick={props.onFit}>
        <Crosshair aria-hidden="true" class="size-4" />
      </Button>
    </div>
  )
}

function PanButton(props: { label: string; onClick: () => void; children: string }) {
  return (
    <Button
      size="sm"
      variant="ghost"
      class="h-8 w-8 font-mono"
      aria-label={props.label}
      onClick={props.onClick}
    >
      <span aria-hidden="true">{props.children}</span>
    </Button>
  )
}

/**
 * Sessão sem tabuleiro. O estado vazio é do MESTRE: o jogador não abre cena, e
 * dizer a ele "abra um tabuleiro" seria oferecer um botão que não existe.
 */
function EmptyBoard(props: { isGm: boolean; onOpen: (place: string, terrain: string) => void }) {
  return (
    <div class="flex min-h-0 flex-1 flex-col items-center justify-center gap-3 p-6 text-center">
      <LayoutGrid aria-hidden="true" class="size-8 text-muted-foreground" />
      <p class="text-sm text-muted-foreground">
        {props.isGm
          ? 'Nenhum tabuleiro aberto. Vale para combate e para cena de interpretação.'
          : 'O mestre ainda não abriu um tabuleiro.'}
      </p>
      <Show when={props.isGm}>
        <OpenBoardDialog
          onOpen={props.onOpen}
          trigger={(open) => <Button onClick={open}>Abrir tabuleiro</Button>}
        />
      </Show>
    </div>
  )
}

/**
 * Quem está na vez e quem vem depois, sem sair do mapa (ALE-179).
 *
 * Só para o JOGADOR: o mestre tem a iniciativa inteira numa coluna ao lado, e
 * repetir três nomes ali seria ruído. Na tela do jogador o tabuleiro ocupa a
 * superfície toda, e para saber se ele é o próximo ele precisava trocar de aba
 * — no meio do turno de outra pessoa, que é quando se decide o que fazer.
 *
 * Fora de combate a tira some: não há vez de ninguém para anunciar.
 */
function TurnStrip(props: { rt: SessionRealtime; hidden: boolean }) {
  const fila = createMemo(() =>
    props.hidden ? [] : upcomingTurns(props.rt.state().initiative, props.rt.state().turnIndex, 3),
  )

  return (
    <Show when={fila().length > 0}>
      <div class="flex shrink-0 items-center gap-2 overflow-hidden border-b border-grimorio-iron px-3 py-1.5 text-[11px]">
        <span class="shrink-0 font-heading uppercase tracking-wide text-grimorio-gold">
          Na vez
        </span>
        <For each={fila()}>
          {(entrada: InitiativeEntry, posicao: () => number) => (
            <>
              <Show when={posicao() > 0}>
                <span aria-hidden="true" class="shrink-0 text-muted-foreground">
                  ›
                </span>
              </Show>
              <span
                class={cn(
                  'min-w-0 truncate',
                  posicao() === 0 ? 'font-semibold text-foreground' : 'text-muted-foreground',
                )}
              >
                {entrada.label}
              </span>
            </>
          )}
        </For>
      </div>
    </Show>
  )
}
