import { Brush, Eraser, Hammer, X } from 'lucide-solid'
import { Show, createMemo, createSignal, onMount } from 'solid-js'
import type { BoardState, BoardToken } from '@/shared/realtime/realtime'
import { Button } from '@/shared/ui/button'
import { ViewControls } from './board-bars'
import { boardKeyAction } from './board-keys'
import { BoardView } from './board-view'
import { createBoardViewport } from './board-viewport'
import { TokenActions } from './token-actions'
import { TokenDialog } from './token-dialog'
import { createFullscreen } from '@/shared/lib/fullscreen'
import { SceneContainerProvider, useSceneContainer } from '@/shared/lib/scene-container'

/**
 * O mestre monta um lugar do acervo sem pôr nada na mesa (ALE-191, fatia 2).
 *
 * Até aqui só existia UMA cena: a que abria era a que todo mundo via, no
 * instante em que abria. Montar a emboscada da próxima sala — que é o que o
 * mestre faz enquanto a mesa joga — obrigava a mostrá-la à mesa antes de estar
 * pronta.
 *
 * O rascunho é LOCAL e sai inteiro numa gravação, ao contrário do tabuleiro
 * vivo, onde cada gesto é uma mensagem. A diferença não é preguiça: aqui não há
 * ninguém para transmitir, não há vez de ninguém e não há concorrência — o
 * protocolo do vivo existe para manter uma mesa em sincronia, e a mesa não está
 * olhando. De brinde, "Cancelar" é jogar o rascunho fora.
 *
 * @example <PlaceEditor scene={cena()} onTable="Taverna do Javali" onSave={guardar} onClose={fechar} />
 */
export function PlaceEditor(props: {
  scene: BoardState
  /** O nome da cena que a mesa está vendo agora, ou `null` se não há tabuleiro. */
  onTable: string | null
  onSave: (scene: BoardState) => void
  onClose: () => void
}) {
  const [tokens, setTokens] = createSignal<BoardToken[]>([...props.scene.tokens])
  const [difficult, setDifficult] = createSignal([...(props.scene.difficult ?? [])])
  const [selectedTokenId, setSelectedTokenId] = createSignal<string | null>(null)
  const [tool, setTool] = createSignal<'brush' | 'eraser' | null>(null)

  // Janela PRÓPRIA: a cripta é outra cena, e herdar o enquadramento da mesa
  // deixaria o mestre olhando o canto vazio de um plano infinito.
  const view = createBoardViewport()
  const [sceneEl, setSceneEl] = createSignal<HTMLElement | null>(null)
  const fullscreen = createFullscreen(document, sceneEl)
  // Em tela cheia o alvo dos overlays passa a ser o PRÓPRIO editor: o elemento
  // em tela cheia é o único que o browser desenha na top layer, e um diálogo
  // portado para fora dele não apareceria (ALE-124).
  const cenaDaPartida = useSceneContainer()
  const alvoDosOverlays = () => (fullscreen.active() ? sceneEl() : cenaDaPartida())
  onMount(() => view.fit(props.scene.tokens))

  const draft = createMemo<BoardState>(() => ({
    ...props.scene,
    tokens: tokens(),
    difficult: difficult(),
    pending: null,
  }))

  const selectedToken = () => tokens().find((peca) => peca.id === selectedTokenId())
  const patchToken = (tokenId: string, patch: Partial<BoardToken>) =>
    setTokens((atual) =>
      atual.map((peca) => (peca.id === tokenId ? { ...peca, ...patch } : peca)),
    )

  // Selecionar de novo LARGA a peça, como no tabuleiro vivo: sem isso não há
  // como desistir, e o próximo clique num quadrado a moveria sem querer.
  const selectToken = (tokenId: string) =>
    setSelectedTokenId((atual) => (atual === tokenId ? null : tokenId))

  const placeSelected = (x: number, y: number) => {
    const peca = selectedToken()
    if (!peca) return
    patchToken(peca.id, { x, y })
    setSelectedTokenId(null)
  }

  const addToken = (peca: Omit<BoardToken, 'id' | 'x' | 'y'>) =>
    setTokens((atual) => [...atual, { ...peca, id: draftTokenId(atual), x: 0, y: 0 }])

  /** A mesma gramática de teclas do tabuleiro vivo (ALE-194): com peça na mão a
   *  seta move a peça, sem peça move a janela. Aqui não há vez nem orçamento —
   *  montar é posicionar livre —, então o passo é direto. */
  const onKeyDown = (event: KeyboardEvent) => {
    const acao = boardKeyAction(event)
    if (!acao) return
    event.preventDefault()
    if (acao.kind === 'fit') return view.fit(tokens())
    if (acao.kind === 'zoom') return view.zoom(acao.deltaPx)
    const peca = selectedToken()
    if (!peca) return view.pan(acao.dx, acao.dy)
    patchToken(peca.id, { x: peca.x + acao.dx, y: peca.y + acao.dy })
  }

  const paintSquare = (x: number, y: number, secondary: boolean) => {
    if (secondary) setTool('eraser')
    const apagar = secondary || tool() === 'eraser'
    setDifficult((casas) => {
      const resto = casas.filter((casa) => casa.x !== x || casa.y !== y)
      return apagar ? resto : [...resto, { x, y }]
    })
  }

  return (
    <SceneContainerProvider element={alvoDosOverlays}>
    <section
      ref={setSceneEl}
      class="scene-grimorio @container flex w-full min-h-0 min-w-0 flex-1 flex-col rounded-none border border-grimorio-gold/50 bg-grimorio-panel"
    >
      <header class="flex shrink-0 flex-wrap items-center gap-2 border-b border-grimorio-iron px-3 py-2">
        <h2 class="min-w-0 truncate font-heading text-sm uppercase tracking-wide text-grimorio-gold">
          <Hammer aria-hidden="true" class="mr-1 inline size-4" />
          Montando {props.scene.place}
        </h2>
        <div class="ml-auto flex flex-wrap items-center justify-end gap-1">
          <ViewControls view={view} onFit={() => view.fit(tokens())} fullscreen={fullscreen} />
          <Button
            size="sm"
            variant={tool() === 'brush' ? 'default' : 'ghost'}
            aria-pressed={tool() === 'brush'}
            onClick={() => setTool((atual) => (atual === 'brush' ? null : 'brush'))}
          >
            <Brush aria-hidden="true" class="size-4" />
            Terreno
          </Button>
          <Button
            size="sm"
            variant={tool() === 'eraser' ? 'default' : 'ghost'}
            aria-pressed={tool() === 'eraser'}
            aria-label="Apagar terreno"
            onClick={() => setTool((atual) => (atual === 'eraser' ? null : 'eraser'))}
          >
            <Eraser aria-hidden="true" class="size-4" />
          </Button>
          <TokenDialog
            onSave={addToken}
            trigger={(open) => (
              <Button size="sm" variant="secondary" onClick={open}>
                + Peça
              </Button>
            )}
          />
          <Button size="sm" onClick={() => props.onSave(draft())}>
            Guardar a cena
          </Button>
          <Button size="sm" variant="ghost" aria-label="Sair sem guardar" onClick={props.onClose}>
            <X aria-hidden="true" class="size-4" />
          </Button>
        </div>
      </header>

      <TableWatchingBar onTable={props.onTable} />

      <BoardView
        board={draft()}
        view={view}
        selectedTokenId={selectedTokenId()}
        movableTokenIds={new Set(tokens().map((peca) => peca.id))}
        onSelectToken={selectToken}
        onSquareClick={tool() === null && selectedTokenId() ? placeSelected : undefined}
        difficult={difficult()}
        onPaintSquare={tool() !== null ? paintSquare : undefined}
        onKeyDown={onKeyDown}
      />

      <Show when={selectedToken()}>
        {(peca) => (
          <TokenActions
            token={peca()}
            onUpdate={(patch) => patchToken(peca().id, patch)}
            onRemove={() => {
              setTokens((atual) => atual.filter((outra) => outra.id !== peca().id))
              setSelectedTokenId(null)
            }}
          />
        )}
      </Show>
    </section>
    </SceneContainerProvider>
  )
}

/**
 * O crachá que a issue pede: diz, PELO NOME, o que a mesa está vendo enquanto o
 * mestre monta outra coisa.
 *
 * Sem ele, montar às escondidas é uma aposta — o mestre não tem como saber, sem
 * sair da tela, se o que ele está desenhando já está na frente de todo mundo. É
 * o widget de fita dourada do Roll20 traduzido para cá.
 */
function TableWatchingBar(props: { onTable: string | null }) {
  return (
    <div
      role="status"
      class="flex shrink-0 flex-wrap items-center gap-2 border-b border-grimorio-gold/40 bg-grimorio-gold/10 px-3 py-1 text-[11px] text-grimorio-gold"
    >
      <p>
        {props.onTable
          ? `A mesa continua vendo ${props.onTable}. Nada daqui aparece para ela até você mostrar esta cena à mesa.`
          : 'A mesa está sem tabuleiro. Nada daqui aparece para ela até você mostrar esta cena à mesa.'}
      </p>
    </div>
  )
}

/**
 * Id da peça criada no rascunho. O servidor cunha o id de quem chega sem um,
 * mas a tela precisa de um AGORA — é por ele que se seleciona e se move a peça
 * que acabou de nascer. `rascunho-N` continua depois do maior que já existe na
 * cena, para uma segunda sessão de montagem não colidir com a primeira.
 */
function draftTokenId(tokens: readonly BoardToken[]): string {
  const usados = tokens
    .map((peca) => Number(/^rascunho-(\d+)$/.exec(peca.id)?.[1]))
    .filter((numero) => Number.isFinite(numero))
  return `rascunho-${Math.max(0, ...usados) + 1}`
}
