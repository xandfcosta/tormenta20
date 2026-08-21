import { Brush, Eraser, Eye, Library, MapPin, Radar, Ruler as RulerIcon, Users, X } from 'lucide-solid'
import { Show, createEffect, createMemo, createSignal, on } from 'solid-js'
import { boardKeyAction } from '@/features/battle-board/board-keys'
import { createAreaTemplate } from '@/features/battle-board/area-template'
import { createRuler } from '@/features/battle-board/ruler'
import { pathBetween } from '@/features/battle-board/board-path'
import { AreaBar, MoveBar, PlayerLensBar, RulerBar, ViewControls } from '@/features/battle-board/board-bars'
import { BoardView } from '@/features/battle-board/board-view'
import { type BoardViewport, SQUARE_METRES } from '@/features/battle-board/board-viewport'
import { EmptyBoard } from '@/features/battle-board/empty-board'
import { PlaceEditor } from '@/features/battle-board/place-editor'
import { PlacesDialog } from '@/features/battle-board/places-list'
import { MarkerActions, TokenActions, nextMarkerText } from '@/features/battle-board/token-actions'
import { TokenDialog } from '@/features/battle-board/token-dialog'
import { SceneContainerProvider, useSceneContainer } from '@/shared/lib/scene-container'
import { createFullscreen } from '@/shared/lib/fullscreen'
import { boardReach } from '@/shared/lib/engine-wasm'
import type { BoardPlace, BoardState, BoardToken, SessionRealtime } from '@/shared/realtime/realtime'
import { Button } from '@/shared/ui/button'
import { ConfirmDialog } from '@/shared/ui/confirm-dialog'
import { TurnStrip } from './board-turn-strip'

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
  /** Linha sob o ponteiro na iniciativa: a peça dela acende (ALE-189). */
  highlightEntryId?: string | null
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
  /**
   * Montar um lugar do acervo (ALE-191, fatia 2). Enquanto o rascunho está
   * aberto ele OCUPA a região: montar é preparação, e desenhar duas cenas lado
   * a lado numa coluna que já divide espaço com a iniciativa seria dar metade
   * do mapa para cada uma. A mesa não fica sabendo — nada aqui é transmitido.
   */
  const [editing, setEditing] = createSignal<{ placeId: number; scene: BoardState } | null>(null)
  const editPlace = (placeId: number) => {
    void props.rt.placeScene(placeId).then((cena) => {
      if (cena) setEditing({ placeId, scene: cena })
    })
  }
  const savePlace = (cena: BoardState) => {
    const edicao = editing()
    if (!edicao) return
    void props.rt.savePlace(edicao.placeId, cena).then(setPlaces)
    setEditing(null)
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
  /**
   * A régua (ALE-124, fatia 6). É de TODO MUNDO e não só do mestre: "dá para
   * acertar daqui?" é pergunta de quem ataca, e hoje se responde contando
   * quadrado com o dedo na tela.
   *
   * Enquanto ela está ligada o clique numa casa MEDE em vez de pousar a peça —
   * mesma regra do pincel, e pela mesma razão: o clique já tem dono.
   */
  const ruler = createRuler()
  const [measuring, setMeasuring] = createSignal(false)
  const toggleRuler = () => {
    ruler.clear()
    setSelectedTokenId(null)
    setTemplating(false)
    setMeasuring((ligada) => !ligada)
  }

  /**
   * O gabarito de área (ALE-124, fatia 6b). Como a régua, é de todo mundo e é
   * local — desenhar não muda a cena. A FORMA vem do motor, transcrita da
   * figura da p225; aqui só mora a escolha e onde ela foi posta.
   */
  const area = createAreaTemplate(() => board()?.tokens ?? [])
  const [templating, setTemplating] = createSignal(false)
  const toggleTemplate = () => {
    area.clear()
    setSelectedTokenId(null)
    setMeasuring(false)
    setTemplating((ligado) => !ligado)
  }

  /**
   * Marcar um LUGAR do mapa que não é peça (ALE-195): a armadilha, a porta que
   * range. Enquanto o modo está ligado, a casa clicada RECEBE um marcador —
   * mesma gramática do pincel e da régua, porque o clique já tem dono.
   *
   * O rótulo nasce sozinho na próxima letra livre: quem está apontando a
   * armadilha no meio da cena não quer digitar, e "A", "B", "C" é como a mesa
   * fala de lugares num mapa.
   */
  const [marking, setMarking] = createSignal(false)
  const [selectedMarkerId, setSelectedMarkerId] = createSignal<string | null>(null)
  const toggleMarking = () => {
    setSelectedMarkerId(null)
    setSelectedTokenId(null)
    setMeasuring(false)
    setTemplating(false)
    setMarking((ligado) => !ligado)
  }
  const selectedMarker = () => board()?.markers?.find((m) => m.id === selectedMarkerId())
  const markAt = (x: number, y: number) => {
    props.rt.addMarker({ x, y, text: nextMarkerText(board()?.markers ?? []), color: 'ouro', hidden: true })
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
  /**
   * Quanto de PV resta em cada linha, para a peça mostrar (ALE-188). Derivado
   * da INICIATIVA, que é onde os vitais vivem: copiá-los para o tabuleiro seria
   * uma segunda verdade sobre a mesma criatura.
   *
   * A entrada sem números fica FORA do mapa, e é assim que a redação por papel
   * chega até a peça sem uma segunda política: quando o mestre oculta os PV, o
   * servidor já apaga `hpCurrent`/`hpMax` da cópia do jogador (`redactForPlayers`),
   * então o filete simplesmente não existe para ele.
   */
  const health = createMemo(() => {
    const porLinha = new Map<string, number>()
    for (const entry of props.rt.state().initiative) {
      if (entry.hpCurrent === undefined || entry.hpMax === undefined || entry.hpMax <= 0) continue
      porLinha.set(entry.id, Math.max(0, Math.round((entry.hpCurrent / entry.hpMax) * 100)))
    }
    return porLinha
  })
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

  const moveSelectedTo = (x: number, y: number) => {
    const token = selectedToken()
    if (!token) return
    if (props.isGm) {
      setOndeEstava((atual) => ({ ...atual, [token.id]: { x: token.x, y: token.y } }))
      props.rt.updateToken(token.id, { x, y })
    } else {
      props.rt.proposeMove(token.id, pathBetween({ x: token.x, y: token.y }, { x, y }))
    }
  }

  /** O clique POUSA: move e larga a peça. Sem largar, o clique seguinte num
   *  quadrado a moveria de novo sem querer. */
  const placeSelected = (x: number, y: number) => {
    moveSelectedTo(x, y)
    setSelectedTokenId(null)
  }

  /**
   * O teclado da superfície (ALE-194). Com peça na mão a seta move a PEÇA um
   * quadrado; sem peça, move a JANELA — a tecla sempre faz o que está em foco
   * na cena, e é assim que volta ao teclado o que os botões de seta davam antes
   * de saírem do cabeçalho (`c95d502`).
   *
   * A seta NÃO fura a regra: ela chama o mesmo caminho do clique, então o
   * jogador continua propondo (com a vez e o orçamento conferidos no servidor)
   * e o mestre continua posicionando livre. Um atalho que passa por fora da
   * regra do livro seria pior que atalho nenhum.
   */
  const onBoardKeyDown = (event: KeyboardEvent) => {
    const acao = boardKeyAction(event)
    if (!acao) return
    event.preventDefault()
    if (acao.kind === 'fit') return props.view.fit(board()?.tokens ?? [])
    if (acao.kind === 'zoom') return props.view.zoom(acao.deltaPx)
    stepSelectedOrView(acao.dx, acao.dy)
  }

  const stepSelectedOrView = (dx: number, dy: number) => {
    const token = selectedToken()
    if (!token || !movableTokenIds().has(token.id)) return props.view.pan(dx, dy)
    const de = nextStepOrigin(token, board()?.pending ?? null)
    moveSelectedTo(de.x + dx, de.y + dy)
  }

  /**
   * De onde a próxima seta conta: o FIM do caminho já proposto, quando há um, e
   * a posição da peça quando não há. O provisório não move a peça — ele a
   * promete —, então sem isto a segunda seta proporia o mesmo passo de novo e o
   * jogador ficaria preso a um quadrado de distância.
   */
  const nextStepOrigin = (token: BoardToken, move: BoardState['pending']) => {
    if (move && move.tokenId === token.id && move.path.length > 0) {
      return move.path[move.path.length - 1]
    }
    return { x: token.x, y: token.y }
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
      {/* Enquanto o rascunho está aberto, a cena da mesa sai da tela em vez de
          ficar escondida por CSS: duas cenas montadas ao mesmo tempo seriam
          duas peças com o mesmo nome acessível na mesma árvore, e o leitor de
          tela leria as duas. */}
      <Show when={editing()}>
        {(edicao) => (
          <PlaceEditor
            scene={edicao().scene}
            onTable={board()?.place ?? null}
            onSave={savePlace}
            onClose={() => setEditing(null)}
          />
        )}
      </Show>
      <Show when={!editing()}>
      <section
        ref={setSceneEl}
        // O escopo de tokens vai junto: em tela cheia o `::backdrop` é preto e
        // a seção precisa pintar o próprio fundo.
        class="scene-grimorio grimorio-frame @container flex w-full min-h-0 min-w-0 flex-1 flex-col bg-grimorio-panel"
      >
      <Show
        when={board()}
        fallback={
          <EmptyBoard
            isGm={props.isGm}
            onOpen={props.rt.openBoard}
            places={places()}
            onReopen={props.rt.reopenPlace}
            onEdit={editPlace}
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
                {/* Fora do bloco do mestre de propósito: medir é de quem joga. */}
                <Button
                  size="sm"
                  variant={measuring() ? 'default' : 'ghost'}
                  aria-pressed={measuring()}
                  aria-label="Régua"
                  title="Régua"
                  onClick={toggleRuler}
                >
                  <RulerIcon aria-hidden="true" class="size-4" />
                </Button>
                <Button
                  size="sm"
                  variant={templating() ? 'default' : 'ghost'}
                  aria-pressed={templating()}
                  aria-label="Gabarito de área"
                  title="Gabarito de área"
                  onClick={toggleTemplate}
                >
                  <Radar aria-hidden="true" class="size-4" />
                </Button>
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
                    variant={marking() ? 'default' : 'ghost'}
                    aria-pressed={marking()}
                    aria-label="Marcar um lugar"
                    title="Marcar um lugar"
                    onClick={toggleMarking}
                  >
                    <MapPin aria-hidden="true" class="size-4" />
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
                    onEdit={editPlace}
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
              highlightEntryId={props.highlightEntryId}
              health={health()}
              selectedTokenId={selectedTokenId()}
              movableTokenIds={movableTokenIds()}
              pending={cena(live()).pending}
              // Quem não pode mover NENHUMA peça também não seleciona: sem isso
              // a camada de quadrados nasceria com centenas de botões inertes
              // na árvore de quem só assiste.
              onSelectToken={movableTokenIds().size > 0 ? selectToken : undefined}
              // Com a ferramenta na mão o pincel é DONO da superfície: sem
              // isto o mesmo clique pintava duas vezes (o gesto e o botão da
              // casa), e com uma peça selecionada ele ainda a pousava no meio
              // do desenho.
              onSquareClick={
                measuring()
                  ? ruler.pick
                  : templating()
                    ? area.pick
                    : marking()
                      ? markAt
                      : !painting() && selectedTokenId()
                        ? placeSelected
                        : undefined
              }
              // Com a régua ligada, TODA casa responde: mede-se para onde não
              // se pode andar, que é justamente a pergunta do ataque.
              reachable={measuring() || templating() || marking() ? undefined : reachable()}
              onSelectMarker={props.isGm ? setSelectedMarkerId : undefined}
              selectedMarkerId={selectedMarkerId()}
              area={area.squares()}
              ruler={ruler.from() && ruler.to() ? { from: ruler.from()!, to: ruler.to()! } : null}
              difficult={cena(live()).difficult}
              // Arrastar com a ferramenta na mão PINTA em vez de mover a vista.
              onPaintSquare={painting() ? paintSquare : undefined}
              onKeyDown={onBoardKeyDown}
            />

            <Show when={props.isGm && selectedToken()}>
              {(token) => (
                <TokenActions
                  token={token()}
                  onUpdate={(patch) => props.rt.updateToken(token().id, patch)}
                  onDuplicate={() => props.rt.duplicateToken(token().id)}
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

            <Show when={templating()}>
              <AreaBar
                kind={area.kind()}
                onKind={area.chooseKind}
                size={area.size()}
                onSize={area.setSize}
                needsDirection={area.needsDirection()}
                hasOrigin={area.origin() !== null}
                hasDirection={area.direction() !== null}
                inside={area.inside()}
                onClose={toggleTemplate}
              />
            </Show>

            <Show when={ruler.reading()}>
              {(leitura) => <RulerBar reading={leitura()} onClose={toggleRuler} />}
            </Show>

            <Show when={props.isGm && selectedMarker()}>
              {(marker) => (
                <MarkerActions
                  marker={marker()}
                  onUpdate={(patch) => props.rt.updateMarker(marker().id, patch)}
                  onRemove={() => {
                    props.rt.removeMarker(marker().id)
                    setSelectedMarkerId(null)
                  }}
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
      </Show>
    </SceneContainerProvider>
  )
}
