import {
  Brush,
  Check,
  Crosshair,
  Eraser,
  Eye,
  Library,
  LayoutGrid,
  Maximize,
  Minimize,
  Undo2,
  Users,
  X,
} from 'lucide-solid'
import { For, Show, createEffect, createMemo, createSignal, on } from 'solid-js'
import { pathBetween } from '@/features/battle-board/board-path'
import { TokenActions } from '@/features/battle-board/token-actions'
import { TokenDialog } from '@/features/battle-board/token-dialog'
import { BoardView } from '@/features/battle-board/board-view'
import { type BoardViewport, SQUARE_METRES } from '@/features/battle-board/board-viewport'
import { OpenBoardDialog } from '@/features/battle-board/open-board-dialog'
import { PlacesDialog, PlacesList } from '@/features/battle-board/places-list'
import { upcomingTurns } from '@/features/session-tracker/tracker-rules'
import { SceneContainerProvider, useSceneContainer } from '@/shared/lib/scene-container'
import { type FullscreenController, createFullscreen } from '@/shared/lib/fullscreen'
import { boardReach } from '@/shared/lib/engine-wasm'
import type {
  BoardPlace,
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
  /** Clicar na peça também ABRE o combatente dela: a peça e a linha da
   *  iniciativa são a mesma criatura, e o mestre não deveria ter de procurar o
   *  nome na lista para ver os PV de quem ele acabou de apontar. */
  onOpenCombatant?: (entryId: string) => void
}) {
  const [selectedTokenId, setSelectedTokenId] = createSignal<string | null>(null)
  // O acervo de cenas guardadas. Buscado sob demanda e não pelo snapshot: ele
  // não muda com a partida, e mandá-lo em toda mensagem seria carregar o
  // armário do mestre a cada peça movida (ALE-124, fatia 5).
  const [places, setPlaces] = createSignal<BoardPlace[]>([])
  const refreshPlaces = () => {
    if (!props.isGm) return
    void props.rt.listPlaces().then(setPlaces)
  }
  // Recarrega quando a mesa FICA sem tabuleiro: encerrar acabou de guardar uma
  // cena, e a lista tem de mostrá-la sem ninguém recarregar a página.
  createEffect(() => {
    if (props.rt.board() === null) refreshPlaces()
  })
  // E recarrega quando a MESA TROCA de cena: mostrar a cripta guardou a taverna,
  // e o acervo que o mestre abrir em seguida tem de mostrá-la (ALE-191).
  createEffect(
    on(
      () => props.rt.board()?.place,
      () => refreshPlaces(),
      { defer: true },
    ),
  )
  /**
   * "Ver como jogador" (ALE-193): a lente do mestre sobre a própria cena.
   *
   * A cópia vem do SERVIDOR e não de uma segunda regra escrita aqui — é
   * literalmente o que o broadcast manda à mesa, redigido pelo mesmo
   * `boardForRole` do Go. Até agora, conferir a emboscada exigia abrir DOIS
   * navegadores com dois logins; foi assim que a ALE-178 foi verificada.
   *
   * O modo É a cópia: ele acende quando ela chega, e por isso a peça escondida
   * nunca pisca na tela entre o clique e a resposta.
   */
  const [playerCopy, setPlayerCopy] = createSignal<BoardState | null>(null)
  const asPlayer = () => playerCopy() !== null
  // Cada pedido leva um número: a resposta de um pedido VELHO — o mestre saiu
  // da lente enquanto ela vinha — não pode reacendê-la sozinha.
  let pedido = 0
  const askPlayerCopy = () => {
    const meu = ++pedido
    void props.rt.boardAsPlayer().then((copia) => {
      if (meu === pedido) setPlayerCopy(copia)
    })
  }
  const sairDaLente = () => {
    pedido++
    setPlayerCopy(null)
  }
  const verComoJogador = () => {
    if (asPlayer()) return sairDaLente()
    // Largar a peça na mão: com a lente ligada ela pode estar invisível, e
    // pousar às cegas é o contrário de conferir.
    setSelectedTokenId(null)
    askPlayerCopy()
  }
  /**
   * Cada mutação re-pergunta: uma lente que mostra a cena de dois movimentos
   * atrás responde a pergunta errada. Até a resposta chegar fica a cópia
   * ANTERIOR, nunca a do mestre — a peça escondida piscando faria ele duvidar
   * do que a mesa viu.
   */
  createEffect(
    on(
      () => props.rt.board()?.version,
      (version) => {
        if (!asPlayer()) return
        if (version === undefined) return sairDaLente() // a cena acabou
        askPlayerCopy()
      },
      { defer: true },
    ),
  )

  // O pincel é MODO e não gesto: enquanto ele está ligado, a casa clicada (ou
  // arrastada) vira terreno difícil em vez de receber a peça. Modo porque o
  // clique numa casa já tem dono — pousar —, e porque gesto com tecla não
  // existe no toque. A BORRACHA é modo irmão em vez de "clicar de novo apaga":
  // arrastando, alternar faria a casa piscar debaixo do dedo.
  const [tool, setTool] = createSignal<'brush' | 'eraser' | null>(null)
  const painting = () => tool() !== null
  const useTool = (wanted: 'brush' | 'eraser') =>
    setTool((atual) => (atual === wanted ? null : wanted))
  /**
   * O botão direito apaga — o gesto que todo editor de mapa tem — e a ferramenta
   * SELECIONADA muda junto, senão a tela diria "pincel" enquanto a mão apaga.
   */
  const paintSquare = (x: number, y: number, secondary: boolean) => {
    if (secondary) setTool('eraser')
    props.rt.paintTerrain(x, y, !secondary && tool() === 'brush')
  }
  // A tela cheia é do TABULEIRO e não da página: pôr a página inteira em tela
  // cheia deixaria o mapa do mesmo tamanho, dividindo a tela com a iniciativa.
  const [sceneEl, setSceneEl] = createSignal<HTMLElement | null>(null)
  const fullscreen = createFullscreen(document, sceneEl)
  // Em tela cheia, o alvo dos overlays passa a ser o PRÓPRIO tabuleiro: o
  // elemento em tela cheia é o único que o browser desenha na top layer, e o
  // diálogo portava para a cena da partida, que está fora dele. Clicar em
  // "Encerrar" não mostrava nada, e o diálogo só aparecia ao SAIR da tela
  // cheia — com o mestre achando que o botão não funcionou.
  const cenaDaPartida = useSceneContainer()
  const alvoDosOverlays = () => (fullscreen.active() ? sceneEl() : cenaDaPartida())
  const board = () => props.rt.board()
  /** O que a tela DESENHA: a cópia da mesa enquanto a lente está ligada, e o
   *  tabuleiro do mestre no resto do tempo. Os CONTROLES continuam sendo os
   *  dele — a lente é sobre a cena, não sobre as ferramentas: ele confere a
   *  emboscada sem parar de montá-la. */
  const cena = (aberto: BoardState) => playerCopy() ?? aberto
  /** Quantas peças a mesa NÃO está vendo — é a resposta que o mestre foi
   *  procurar, e ela não se lê contando o que sumiu da tela. */
  const hiddenCount = () => (board()?.tokens ?? []).filter((peca) => peca.hidden).length

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
  const selectToken = (tokenId: string) => {
    const escolhida = selectedTokenId() !== tokenId
    setSelectedTokenId(escolhida ? tokenId : null)
    if (!escolhida) return
    const entryId = board()?.tokens.find((peca) => peca.id === tokenId)?.entryId
    if (entryId) props.onOpenCombatant?.(entryId)
  }

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
    <SceneContainerProvider element={alvoDosOverlays}>
      <section
        ref={setSceneEl}
        // O escopo de tokens vai junto: em tela cheia o `::backdrop` é preto e
        // a seção precisa pintar o próprio fundo.
        class="scene-grimorio @container flex w-full min-h-0 min-w-0 flex-1 flex-col rounded-sm border border-grimorio-iron bg-[var(--grimorio-panel)]"
      >
      <Show
        when={board()}
        fallback={
          <EmptyBoard
            isGm={props.isGm}
            onOpen={props.rt.openBoard}
            places={places()}
            onReopen={props.rt.reopenPlace}
            onRemovePlace={(placeId) => void props.rt.removePlace(placeId).then(setPlaces)}
          />
        }
      >
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
                <ViewControls
                  view={props.view}
                  onFit={() => props.view.fit(live().tokens)}
                  fullscreen={fullscreen}
                />
                <Show when={props.isGm}>
                  {/* Ícone só, como a borracha: o cabeçalho acabou de perder
                      seis botões por ocupar o lugar de quem trabalha (ALE-124),
                      e quem nomeia o modo em texto é a tira de aviso abaixo —
                      que só existe enquanto ele está ligado. */}
                  <Button
                    size="sm"
                    variant={asPlayer() ? 'default' : 'ghost'}
                    aria-pressed={asPlayer()}
                    aria-label="Ver como jogador"
                    title="Ver como jogador"
                    onClick={verComoJogador}
                  >
                    <Eye aria-hidden="true" class="size-4" />
                  </Button>
                  <Button
                    size="sm"
                    variant={tool() === 'brush' ? 'default' : 'ghost'}
                    aria-pressed={tool() === 'brush'}
                    onClick={() => useTool('brush')}
                  >
                    <Brush aria-hidden="true" class="size-4" />
                    Terreno
                  </Button>
                  <Button
                    size="sm"
                    variant={tool() === 'eraser' ? 'default' : 'ghost'}
                    aria-pressed={tool() === 'eraser'}
                    aria-label="Apagar terreno"
                    onClick={() => useTool('eraser')}
                  >
                    <Eraser aria-hidden="true" class="size-4" />
                  </Button>
                  {/* O acervo alcançável COM cena na mesa (ALE-191): trocar de
                      cena não exige mais ENCERRAR o tabuleiro primeiro, com a
                      mesa vendo a grade sumir e voltar. */}
                  <PlacesDialog
                    places={places()}
                    onTable={live().place}
                    onOpenList={refreshPlaces}
                    onReopen={props.rt.reopenPlace}
                    onRemove={(placeId) => void props.rt.removePlace(placeId).then(setPlaces)}
                    trigger={(open) => (
                      <Button size="sm" variant="ghost" aria-label="Lugares da crônica" onClick={open}>
                        <Library aria-hidden="true" class="size-4" />
                      </Button>
                    )}
                  />
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
                    description="A cena vai para os Lugares da crônica, com as peças onde estão, e você a reabre quando quiser. A iniciativa e os PV continuam como estão."
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

            <Show when={asPlayer()}>
              <PlayerLensBar hidden={hiddenCount()} onExit={sairDaLente} />
            </Show>

            <TurnStrip rt={props.rt} hidden={props.isGm} />

            <BoardView
              board={cena(live())}
              view={props.view}
              activeEntryId={props.activeEntryId}
              selectedTokenId={selectedTokenId()}
              movableTokenIds={movableTokenIds()}
              reachable={reachable()}
              pending={cena(live()).pending}
              // Quem não pode mover NENHUMA peça também não seleciona: sem isso
              // a camada de quadrados nasceria com centenas de botões inertes
              // na árvore de quem só assiste.
              onSelectToken={movableTokenIds().size > 0 ? selectToken : undefined}
              // Com a ferramenta na mão o pincel é DONO da superfície: sem
              // isto o mesmo clique pintava duas vezes (o gesto e o botão da
              // casa), e com uma peça selecionada ele ainda a pousava no meio
              // do desenho.
              onPlaceToken={!painting() && selectedTokenId() ? placeSelected : undefined}
              difficult={cena(live()).difficult}
              // Arrastar com a ferramenta na mão PINTA em vez de mover a vista.
              onPaintSquare={painting() ? paintSquare : undefined}
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
    </SceneContainerProvider>
  )
}

/**
 * A tira da lente do mestre (ALE-193).
 *
 * Existe porque um modo que se esquece é pior que nenhum: o mestre que não
 * percebe que está na vista da mesa não vê a peça que ele mesmo escondeu, e vai
 * concluir que ela sumiu. Por isso ela é PERSISTENTE, nomeia o modo em texto e
 * carrega a própria saída.
 *
 * E diz o NÚMERO de peças escondidas, que é a pergunta que trouxe o mestre até
 * aqui — "a emboscada está mesmo invisível?". Contar o que sumiu da tela não é
 * resposta: ele não sabe o que não está vendo.
 */
function PlayerLensBar(props: { hidden: number; onExit: () => void }) {
  return (
    <div
      role="status"
      class="flex shrink-0 flex-wrap items-center gap-2 border-b border-grimorio-gold/40 bg-grimorio-gold/10 px-3 py-1 text-[11px] text-grimorio-gold"
    >
      <Eye aria-hidden="true" class="size-3.5 shrink-0" />
      <p>
        Você está vendo a cena como a mesa.
        {props.hidden > 0
          ? ` ${props.hidden} ${props.hidden === 1 ? 'peça escondida não aparece' : 'peças escondidas não aparecem'}.`
          : ' Nenhuma peça escondida nesta cena.'}
      </p>
      <Button size="sm" variant="ghost" class="ml-auto" onClick={() => props.onExit()}>
        Voltar à vista do mestre
      </Button>
    </div>
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
/**
 * Os controles da vista. As quatro setas e o −/+ saíram (ALE-124): arrastar e a
 * roda/pinça fazem o mesmo melhor e sem ocupar seis lugares num cabeçalho que
 * já quebrava linha no telefone. Fica o que gesto nenhum faz — achar o grupo
 * num plano infinito — e entra a tela cheia, que é do TABULEIRO e não da
 * página: em tela cheia da página o mapa continuaria dividindo espaço com a
 * iniciativa, que é justamente o que se quer sair.
 */
function ViewControls(props: {
  view: BoardViewport
  onFit: () => void
  fullscreen: FullscreenController
}) {
  return (
    <div class="flex items-center gap-0.5">
      <Button size="sm" variant="ghost" aria-label="Centralizar nas peças" onClick={props.onFit}>
        <Crosshair aria-hidden="true" class="size-4" />
      </Button>
      <Show when={props.fullscreen.supported}>
        <Button
          size="sm"
          variant="ghost"
          aria-label={props.fullscreen.active() ? 'Sair da tela cheia' : 'Tabuleiro em tela cheia'}
          onClick={props.fullscreen.toggle}
        >
          <Show
            when={props.fullscreen.active()}
            fallback={<Maximize aria-hidden="true" class="size-4" />}
          >
            <Minimize aria-hidden="true" class="size-4" />
          </Show>
        </Button>
      </Show>
    </div>
  )
}


/**
 * Sessão sem tabuleiro. O estado vazio é do MESTRE: o jogador não abre cena, e
 * dizer a ele "abra um tabuleiro" seria oferecer um botão que não existe.
 */
function EmptyBoard(props: {
  isGm: boolean
  onOpen: (place: string, terrain: string) => void
  places: readonly BoardPlace[]
  onReopen: (placeId: number) => void
  onRemovePlace: (placeId: number) => void
}) {
  return (
    <div class="flex min-h-0 flex-1 flex-col items-center justify-center gap-3 overflow-y-auto p-6 text-center">
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
        {/* O acervo vem DEPOIS do botão de abrir: montar uma cena nova é o que
            se faz na primeira noite, e reabrir é o que se faz nas outras. */}
        <PlacesList
          places={props.places}
          onReopen={props.onReopen}
          onRemove={props.onRemovePlace}
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
