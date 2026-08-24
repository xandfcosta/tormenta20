import { cleanup, fireEvent, render, screen, waitFor, within } from '@solidjs/testing-library'
import userEvent from '@testing-library/user-event'
import { createSignal } from 'solid-js'
import { describe, expect, it, vi } from 'vitest'
import { createBoardViewport } from '@/features/battle-board/board-viewport'
import type { BoardPlace, BoardState, SessionRealtime } from '@/shared/realtime/realtime'
import { BoardRegion } from './board-region'

/**
 * O tabuleiro na cena da sessão (ALE-124). O que se prova aqui é o que alguém na
 * mesa notaria: a peça aparece onde o servidor disse, o mestre move clicando, e
 * quem não é mestre não ganha controle nenhum.
 *
 * Posição é afirmada pelo NOME ACESSÍVEL ("coluna 3, linha 2") e não por pixel:
 * em jsdom todo elemento mede zero, e uma asserção de layout aqui passaria verde
 * sobre um tabuleiro quebrado. O que o browser mede fica para o e2e.
 */
class FakeRealtime {
  readonly updateToken = vi.fn()
  readonly closeBoard = vi.fn()
  readonly populateBoard = vi.fn()
  readonly openBoard = vi.fn()
  readonly removeToken = vi.fn()
  readonly addToken = vi.fn()
  readonly paintTerrain = vi.fn()
  readonly reopenPlace = vi.fn()
  /** O acervo chega por PERGUNTA e não pelo snapshot (ALE-124, fatia 5). */
  places: BoardPlace[] = []
  /** A cópia que o SERVIDOR redige (ALE-193): aqui o fake faz o que o Go faz —
   *  `boardForRole("player", …)`, que apaga a peça escondida da cópia da mesa. */
  readonly boardAsPlayer = vi.fn(() => {
    const aberto = this.board()
    if (!aberto) return Promise.resolve(null)
    // A CORTINA vem antes da redação de peça, como no `BoardForRole` do Go
    // (ALE-202): com ela fechada a mesa não recebe cena nenhuma, e um fake que
    // só filtrasse peça escondida mentiria sobre o que o servidor manda.
    if (aberto.curtained) {
      return Promise.resolve({ version: aberto.version, curtained: true } as BoardState)
    }
    return Promise.resolve({ ...aberto, tokens: aberto.tokens.filter((peca) => !peca.hidden) })
  })
  readonly setCurtain = vi.fn()
  /** Montar um lugar do acervo (ALE-191, fatia 2). A cena guardada chega por
   *  PERGUNTA — a lista viaja só com nome e contagem. */
  scene: BoardState | null = null
  readonly savePlace = vi.fn(() => Promise.resolve(this.places))
  readonly duplicateToken = vi.fn()
  readonly addMarker = vi.fn()
  readonly updateMarker = vi.fn()
  readonly removeMarker = vi.fn()
  readonly proposeMove = vi.fn()
  readonly commitMove = vi.fn()
  readonly cancelMove = vi.fn()

  constructor(
    // Accessor e não valor: a cena muda no meio do teste (o mestre revela uma
    // peça), e é por essa mudança que a lente do ALE-193 tem de re-perguntar.
    private readonly board: () => BoardState | null,
    private readonly turnIndex = -1,
    /** Vitais por linha, quando o teste quer o filete de PV da peça (ALE-188).
     *  Ausentes = a linha não tem números, que é o que o servidor manda ao
     *  jogador quando o mestre oculta os PV. */
    private readonly vitals: Record<string, { hpCurrent: number; hpMax: number }> = {},
  ) {}

  asRealtime(): SessionRealtime {
    return {
      state: () => ({
        initiative: [
          { id: 'e1', label: 'Sílfide Ladina', initiative: 18, type: 'character', ...this.vitals.e1 },
          { id: 'e2', label: 'Ogro', initiative: 12, type: 'npc', ...this.vitals.e2 },
          { id: 'e3', label: 'Batedor Élfico', initiative: 9, type: 'character', ...this.vitals.e3 },
        ],
        round: 1,
        turnIndex: this.turnIndex,
      }),
      isConnected: () => true,
      error: () => null,
      board: this.board,
      updateToken: this.updateToken,
      duplicateToken: this.duplicateToken,
      addMarker: this.addMarker,
      updateMarker: this.updateMarker,
      removeMarker: this.removeMarker,
      removeToken: this.removeToken,
      addToken: this.addToken,
      closeBoard: this.closeBoard,
      populateBoard: this.populateBoard,
      paintTerrain: this.paintTerrain,
      setCurtain: this.setCurtain,
      listPlaces: () => Promise.resolve(this.places),
      boardAsPlayer: this.boardAsPlayer,
      placeScene: () => Promise.resolve(this.scene),
      savePlace: this.savePlace,
      reopenPlace: this.reopenPlace,
      removePlace: (placeId: number) => {
        this.places = this.places.filter((lugar) => lugar.id !== placeId)
        return Promise.resolve(this.places)
      },
      openBoard: this.openBoard,
      proposeMove: this.proposeMove,
      commitMove: this.commitMove,
      cancelMove: this.cancelMove,
    } as unknown as SessionRealtime
  }
}

const TABULEIRO: BoardState = {
  version: 3,
  place: 'Taverna do Javali',
  terrain: 'taverna',
  tokens: [
    { id: 't1', label: 'Ogro', x: 3, y: 2, footprint: 2, kind: 'npc', entryId: 'e1' },
    { id: 't2', label: 'Sílfide Ladina', x: 6, y: 5, footprint: 1, kind: 'character' },
    // Coordenada NEGATIVA é lugar legítimo num plano infinito — e o rótulo tem
    // de dizer o número que o servidor guarda, não um "+1" de planilha.
    { id: 't3', label: 'Batedor Élfico', x: -4, y: -3, footprint: 1, kind: 'character' },
  ],
}

/** A cena que o teste monta: fixa, ou um sinal quando ela muda no meio. */
type CenaDoTeste = BoardState | null | (() => BoardState | null)

function renderRegion(
  isGm: boolean,
  live: CenaDoTeste = TABULEIRO,
  activeEntryId?: string,
  jogador: {
    myCharacterIds?: ReadonlySet<number>
    turnIndex?: number
    places?: BoardPlace[]
    scene?: BoardState
    vitals?: Record<string, { hpCurrent: number; hpMax: number }>
  } = {},
  onOpenCombatant = vi.fn(),
) {
  const rt = new FakeRealtime(
    typeof live === 'function' ? live : () => live,
    jogador.turnIndex ?? -1,
    jogador.vitals,
  )
  rt.places = jogador.places ?? []
  rt.scene = jogador.scene ?? null
  // A janela nasce fora da região, como na página: ela precisa sobreviver à
  // troca de região, e o teste monta a mesma composição que a cena monta.
  const view = createBoardViewport()
  // A vista começa mostrando a origem menos um pedaço, para o teste alcançar
  // tanto o quadrado (3,2) quanto o negativo (−4,−3) sem depender de medição —
  // em jsdom todo elemento mede zero e o ResizeObserver nem existe.
  view.centerOn(0, 0)
  const view0 = render(() => (
    <BoardRegion
      rt={rt.asRealtime()}
      isGm={isGm}
      view={view}
      activeEntryId={activeEntryId}
      myCharacterIds={jogador.myCharacterIds}
      onOpenCombatant={onOpenCombatant}
    />
  ))
  return { ...view0, rt, user: userEvent.setup(), view, onOpenCombatant }
}

/** Um toque com o botão DIREITO, que é a borracha rápida. */
function apagaComOBotaoDireito(container: HTMLElement) {
  const plano = container.querySelector('[role="grid"]')
  if (!plano) throw new Error('o tabuleiro não montou')
  const toque = new Event('pointerdown', { bubbles: true })
  Object.assign(toque, { pointerId: 1, clientX: 10, clientY: 10, button: 2, buttons: 2 })
  plano.dispatchEvent(toque)
}

/** Um toque na superfície do tabuleiro. `PointerEvent` não existe em jsdom, e o
 *  que o gesto lê são `pointerId`/`clientX`/`clientY` — um `Event` com esses
 *  campos serve, e é o mesmo truque que o guia do front registra para
 *  `AnimationEvent`. */
function tocaOTabuleiro(container: HTMLElement) {
  const plano = container.querySelector('[role="grid"]')
  if (!plano) throw new Error('o tabuleiro não montou')
  const toque = new Event('pointerdown', { bubbles: true })
  Object.assign(toque, { pointerId: 1, clientX: 10, clientY: 10 })
  plano.dispatchEvent(toque)
}

describe('o tabuleiro na cena', () => {
  it('as peças aparecem no quadrado que o servidor mandou', () => {
    renderRegion(true)

    expect(screen.getByRole('button', { name: 'Ogro, coluna 3, linha 2' })).toBeInTheDocument()
    expect(
      screen.getByRole('button', { name: 'Sílfide Ladina, coluna 6, linha 5' }),
    ).toBeInTheDocument()
  })

  // Duas ações, não uma: selecionar e pousar. Arrastar entra na fatia do
  // movimento — e mesmo lá continuará havendo o caminho de dois cliques, porque
  // gesto nunca é o único caminho.
  it('o mestre seleciona a peça e pousa num quadrado', async () => {
    const { rt, user } = renderRegion(true)

    await user.click(screen.getByRole('button', { name: 'Ogro, coluna 3, linha 2' }))
    await user.click(screen.getByRole('button', { name: 'Coluna 5, linha 3' }))

    expect(rt.updateToken).toHaveBeenCalledWith('t1', { x: 5, y: 3 })
  })

  // Clicar de novo na mesma peça a LARGA: sem isso não há como desistir, e o
  // próximo clique num quadrado moveria a peça errada.
  it('clicar de novo na mesma peça desiste do movimento', async () => {
    const { rt, user } = renderRegion(true)
    const ogro = screen.getByRole('button', { name: 'Ogro, coluna 3, linha 2' })

    await user.click(ogro)
    expect(screen.getByRole('button', { name: 'Coluna 5, linha 3' })).toBeInTheDocument()

    await user.click(ogro)

    // Sem peça na mão não há onde pousar: os quadrados só existem enquanto
    // alguém está movendo. Antes eles ficavam sempre na árvore, e eram
    // centenas de botões inertes no leitor de tela (ALE-124).
    expect(screen.queryByRole('button', { name: 'Coluna 5, linha 3' })).not.toBeInTheDocument()
    expect(rt.updateToken).not.toHaveBeenCalled()
  })

  // A vez é a mesma informação da iniciativa, no mesmo vocabulário: quem está na
  // vez tem o anel dourado, e o teste afirma o ESTADO, não a classe.
  it('o jogador não recebe controle nenhum sobre as peças', () => {
    renderRegion(false)

    const ogro = screen.getByRole('button', { name: 'Ogro, coluna 3, linha 2' })
    expect(ogro).toBeDisabled()
    expect(screen.queryByRole('button', { name: 'Coluna 5, linha 3' })).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Encerrar o tabuleiro' })).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: /Trazer a iniciativa/ })).not.toBeInTheDocument()
    // A lente do mestre é dele: quem já É a mesa não tem o que conferir.
    expect(screen.queryByRole('button', { name: 'Ver como jogador' })).not.toBeInTheDocument()
    // E o acervo de cenas é preparação do mestre (ALE-191).
    expect(screen.queryByRole('button', { name: 'Lugares da campanha' })).not.toBeInTheDocument()
  })

  it('sem tabuleiro, só o mestre vê como abrir um', () => {
    renderRegion(false, null)
    expect(screen.queryByRole('button', { name: 'Abrir tabuleiro' })).not.toBeInTheDocument()
    expect(screen.getByText(/ainda não abriu um tabuleiro/)).toBeInTheDocument()
  })

  // O PLANO É INFINITO: coordenada negativa é lugar legítimo, e o rótulo diz o
  // número que o servidor guarda. Traduzir para "coluna 1" seria mentir sobre
  // onde a peça está, e a mesa fala esse número em voz alta (ALE-124).
  it('a peça em coordenada negativa aparece com o número que o servidor guarda', () => {
    renderRegion(true)

    expect(
      screen.getByRole('button', { name: 'Batedor Élfico, coluna -4, linha -3' }),
    ).toBeInTheDocument()
  })

  // Andar com a janela é o que substitui a borda: sem isso, metade da cena fica
  // inalcançável num plano sem fim.
  it('a peça fora da janela some, e mover a vista a traz de volta', async () => {
    const longe: BoardState = {
      ...TABULEIRO,
      tokens: [{ id: 't9', label: 'Sentinela Distante', x: 60, y: 0, footprint: 1, kind: 'npc' }],
    }
    const { view } = renderRegion(true, longe)

    expect(screen.queryByRole('button', { name: /Sentinela Distante/ })).not.toBeInTheDocument()

    // A vista anda pela JANELA, que é da página — os botões de seta saíram
    // quando arrastar passou a fazer o mesmo melhor, e o arraste é pixel e
    // layout, que jsdom não tem (isso é e2e).
    view.centerOn(60, 0)

    expect(await screen.findByRole('button', { name: /Sentinela Distante/ })).toBeInTheDocument()
  })

  // "Centralizar" enquadra as PEÇAS e não a origem: num plano infinito o centro
  // não significa nada — o que o mestre quer é achar o grupo.
  it('centralizar acha o grupo longe da origem', async () => {
    const longe: BoardState = {
      ...TABULEIRO,
      tokens: [{ id: 't9', label: 'Sentinela Distante', x: 120, y: 80, footprint: 1, kind: 'npc' }],
    }
    const { user } = renderRegion(true, longe)

    await user.click(screen.getByRole('button', { name: 'Centralizar nas peças' }))

    expect(screen.getByRole('button', { name: /Sentinela Distante/ })).toBeInTheDocument()
  })

  it('o mestre abre um tabuleiro com lugar e cenário', async () => {
    const { rt, user } = renderRegion(true, null)

    await user.click(screen.getByRole('button', { name: 'Abrir tabuleiro' }))
    await user.type(screen.getByLabelText('Lugar'), 'Cripta')
    await user.click(screen.getByRole('button', { name: 'Abrir' }))

    expect(rt.openBoard).toHaveBeenCalledWith('Cripta', 'pedra')
  })
})


/**
 * Mover a peça (ALE-124, fatia 3).
 *
 * A trava de verdade é do servidor (`assertMovable`, provado em
 * `api/board_move_test.go`); o que se prova AQUI é a outra metade: que a tela
 * oferece o movimento a quem pode e não o oferece a quem não pode, e que o que
 * ela manda para o servidor é o caminho certo.
 */
const MEU_HEROI = new Set([77])

const COM_JOGADOR: BoardState = {
  ...TABULEIRO,
  tokens: [
    { id: 't1', label: 'Ogro', x: 3, y: 2, footprint: 2, kind: 'npc', entryId: 'e2' },
    {
      id: 't2',
      label: 'Sílfide Ladina',
      x: 6,
      y: 5,
      footprint: 1,
      kind: 'character',
      entryId: 'e1',
      characterId: 77,
      speedSquares: 6,
    },
  ],
}

describe('o jogador move a própria peça', () => {
  it('na própria vez, clicar numa casa acesa PROPÕE o caminho', async () => {
    const { rt, user } = renderRegion(false, COM_JOGADOR, 'e1', {
      myCharacterIds: MEU_HEROI,
      turnIndex: 0,
    })

    await user.click(screen.getByRole('button', { name: 'Sílfide Ladina, coluna 6, linha 5' }))
    await user.click(screen.getByRole('button', { name: 'Coluna 8, linha 5' }))

    // Caminho, e não destino: o custo depende do percurso (a diagonal custa o
    // dobro, T20 p238), e é o servidor que cobra.
    expect(rt.proposeMove).toHaveBeenCalledWith('t2', [
      { x: 6, y: 5 },
      { x: 7, y: 5 },
      { x: 8, y: 5 },
    ])
    // A peça NÃO é reposicionada por fora do fluxo de proposta.
    expect(rt.updateToken).not.toHaveBeenCalled()
  })

  // O losango é a regra na tela: com 6 quadrados dá para andar 6 em linha reta
  // e só 3 na diagonal (p238). A casa fora do alcance não responde ao clique.
  it('a casa além do deslocamento não aceita a peça', async () => {
    const { rt, user } = renderRegion(false, COM_JOGADOR, 'e1', {
      myCharacterIds: MEU_HEROI,
      turnIndex: 0,
    })

    await user.click(screen.getByRole('button', { name: 'Sílfide Ladina, coluna 6, linha 5' }))
    // (0,0) está DENTRO da janela e a 11 quadrados de (6,5) — bem além dos 6.
    const longe = screen.getByRole('button', { name: 'Coluna 0, linha 0' })

    expect(longe).toBeDisabled()
    await user.click(longe)
    expect(rt.proposeMove).not.toHaveBeenCalled()
  })

  /**
   * O brejo entra na conta ANTES do clique: com o chão pintado, o losango de
   * alcance encolhe sozinho. Sem isto o jogador via a casa acesa, clicava, e o
   * servidor recusava — a regra do livro chegava como erro em vez de chegar
   * como desenho (T20 p238, ALE-124 fatia 4).
   */
  it('o terreno difícil encolhe o losango antes do clique', async () => {
    const semBrejo = renderRegion(false, COM_JOGADOR, 'e1', {
      myCharacterIds: MEU_HEROI,
      turnIndex: 0,
    })
    await semBrejo.user.click(
      screen.getByRole('button', { name: 'Sílfide Ladina, coluna 6, linha 5' }),
    )
    // Seis quadrados em linha reta é exatamente o deslocamento: a última casa
    // do alcance responde. (A oeste porque a janela do teste vai de −10 a 9.)
    expect(screen.getByRole('button', { name: 'Coluna 0, linha 5' })).toBeEnabled()
    cleanup()

    // Uma casa de brejo no meio do caminho custa 2 em vez de 1, e a ponta do
    // losango deixa de caber no orçamento.
    // O brejo colado na peça: entrar nele custa 2, e contornar pela diagonal
    // custa 2 também (p238) — não há caminho de 6 até a ponta.
    const comBrejo: BoardState = { ...COM_JOGADOR, difficult: [{ x: 5, y: 5 }] }
    const { user } = renderRegion(false, comBrejo, 'e1', {
      myCharacterIds: MEU_HEROI,
      turnIndex: 0,
    })
    await user.click(screen.getByRole('button', { name: 'Sílfide Ladina, coluna 6, linha 5' }))

    expect(screen.getByRole('button', { name: 'Coluna 0, linha 5' })).toBeDisabled()
  })

  it('fora da própria vez a peça nem responde', () => {
    renderRegion(false, COM_JOGADOR, 'e2', { myCharacterIds: MEU_HEROI, turnIndex: 1 })

    expect(screen.getByRole('button', { name: 'Sílfide Ladina, coluna 6, linha 5' })).toBeDisabled()
  })

  it('a peça de outro não responde nem na vez dela', () => {
    renderRegion(false, COM_JOGADOR, 'e2', { myCharacterIds: MEU_HEROI, turnIndex: 1 })

    expect(screen.getByRole('button', { name: 'Ogro, coluna 3, linha 2' })).toBeDisabled()
  })
})

describe('confirmar o movimento proposto', () => {
  const PROPOSTO: BoardState = {
    ...COM_JOGADOR,
    pending: {
      tokenId: 't2',
      path: [
        { x: 6, y: 5 },
        { x: 7, y: 5 },
        { x: 8, y: 5 },
      ],
      cost: 2,
      budget: 6,
      // Caminho reto e limpo: nenhuma dobra a nomear (ALE-190).
      diagonals: 0,
      difficult: 0,
      byUserId: 42,
    },
  }

  // Quadrado é a unidade da REGRA (p236) e metro é a unidade da conversa na
  // mesa: a barra diz os dois, e diz o orçamento, porque "2" sem "de 6" não
  // responde o que o jogador está perguntando.
  it('a barra diz o custo em quadrados, em metros e contra o orçamento', () => {
    renderRegion(false, PROPOSTO, 'e1', { myCharacterIds: MEU_HEROI, turnIndex: 0 })

    expect(screen.getByText(/2 quadrados \(3,0m\) de 6/)).toBeInTheDocument()
  })

  // A REGRA ao lado do número (ALE-190). Sem ela o losango do alcance chega
  // como mágica: o jogador aceita a forma e não aprende a contar — e na mesa
  // presencial é ele quem vai contar quadrado no papel.
  it('caminho reto e limpo não anuncia regra nenhuma', () => {
    renderRegion(false, PROPOSTO, 'e1', { myCharacterIds: MEU_HEROI, turnIndex: 0 })

    // Nada a nomear: os dois passos são retos e fora do brejo.
    expect(screen.queryByText(/o dobro \(p238\)/)).not.toBeInTheDocument()
  })

  it('com diagonal e brejo no caminho, a barra nomeia as duas regras', () => {
    const dobrado: BoardState = {
      ...PROPOSTO,
      pending: { ...PROPOSTO.pending!, cost: 5, diagonals: 1, difficult: 1 },
    }
    renderRegion(false, dobrado, 'e1', { myCharacterIds: MEU_HEROI, turnIndex: 0 })

    // As contagens vêm do SERVIDOR, do mesmo laço que cobrou o caminho: a tela
    // escolhe palavras, nunca refaz a aritmética do livro (ALE-104).
    expect(screen.getByText('diagonal e terreno difícil custam o dobro (p238)')).toBeInTheDocument()
  })

  it('confirmar manda a VERSÃO que o cliente tinha na mão', async () => {
    const { rt, user } = renderRegion(false, PROPOSTO, 'e1', {
      myCharacterIds: MEU_HEROI,
      turnIndex: 0,
    })

    await user.click(screen.getByRole('button', { name: /Confirmar/ }))

    // Sem a versão o servidor não teria como recusar um commit escrito sobre
    // uma cena que já mudou.
    expect(rt.commitMove).toHaveBeenCalledWith(PROPOSTO.version)
  })

  it('o mestre confirma pelo jogador', async () => {
    const { rt, user } = renderRegion(true, PROPOSTO, 'e1')

    await user.click(screen.getByRole('button', { name: /Confirmar/ }))

    expect(rt.commitMove).toHaveBeenCalledWith(PROPOSTO.version)
  })

  // Quem não decide continua VENDO: é essa a razão de o provisório ser estado, e
  // não um arraste privado dentro do cliente de quem move.
  it('quem não decide vê o caminho, mas não os botões', () => {
    renderRegion(false, PROPOSTO, 'e1', { myCharacterIds: new Set([999]), turnIndex: 0 })

    expect(screen.getByText(/2 quadrados/)).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: /Confirmar/ })).not.toBeInTheDocument()
    expect(screen.getByText('Aguardando confirmação.')).toBeInTheDocument()
  })
})

/**
 * O que o mestre faz com a peça (ALE-178).
 *
 * O servidor já sabia fazer tudo isto desde a primeira fatia — os eventos
 * existem, com porta de mestre e teste — e nada tinha botão. O caso mais caro
 * era o esconder: a redação da emboscada está implementada e testada no Go, e
 * estava MORTA porque a tela não tinha como ligá-la.
 *
 * A redação em si não se repete aqui: ela é do Go e está congelada lá. O que se
 * prova nesta camada é que a tela EMITE a intenção certa, e que o jogador não
 * ganha o painel.
 */
describe('o painel da peça', () => {
  const comPecas: BoardState = {
    ...TABULEIRO,
    tokens: [
      { id: 't1', label: 'Ogro', x: 3, y: 2, footprint: 2, kind: 'npc', entryId: 'e1' },
      { id: 't4', label: 'Porta', x: 1, y: 1, footprint: 1, kind: 'object', hidden: true },
    ],
  }

  it('esconder manda a peça sumir da cópia do jogador', async () => {
    const { rt, user } = renderRegion(true, comPecas)

    await user.click(screen.getByRole('button', { name: 'Ogro, coluna 3, linha 2' }))
    await user.click(screen.getByRole('button', { name: 'Esconder Ogro' }))

    expect(rt.updateToken).toHaveBeenCalledWith('t1', { hidden: true })
  })

  // O rótulo diz o que vai ACONTECER, não o estado: "Esconder" numa peça já
  // escondida mandaria o mestre esconder o que ele quer revelar.
  it('na peça escondida, o botão oferece mostrar', async () => {
    const { rt, user } = renderRegion(true, comPecas)

    // O nome acessível da peça escondida ANUNCIA o segredo desde a ALE-179:
    // quem lê por leitor de tela tem o mesmo direito de saber o que a borda
    // tracejada conta para quem enxerga.
    await user.click(
      screen.getByRole('button', { name: 'Porta, coluna 1, linha 1, escondida dos jogadores' }),
    )
    await user.click(screen.getByRole('button', { name: 'Mostrar Porta' }))

    expect(rt.updateToken).toHaveBeenCalledWith('t4', { hidden: false })
  })

  it('tirar a peça pede confirmação e some com ela', async () => {
    const { rt, user } = renderRegion(true, comPecas)

    await user.click(screen.getByRole('button', { name: 'Ogro, coluna 3, linha 2' }))
    await user.click(screen.getByRole('button', { name: 'Tirar Ogro' }))
    const dialogo = await screen.findByRole('dialog')
    await user.click(within(dialogo).getByRole('button', { name: 'Tirar' }))

    expect(rt.removeToken).toHaveBeenCalledWith('t1')
  })

  // "Voltar para onde estava" é memória LOCAL da tela, não histórico no
  // servidor: o mestre erra o quadrado e desfaz sem que a mesa veja um estado
  // novo nascer só para isso.
  it('depois de pousar, dá para voltar a peça para onde estava', async () => {
    const { rt, user } = renderRegion(true, comPecas)

    await user.click(screen.getByRole('button', { name: 'Ogro, coluna 3, linha 2' }))
    await user.click(screen.getByRole('button', { name: 'Coluna 5, linha 3' }))
    expect(rt.updateToken).toHaveBeenCalledWith('t1', { x: 5, y: 3 })

    // A peça segue selecionada? Não: pousar solta. Seleciona de novo para desfazer.
    await user.click(screen.getByRole('button', { name: 'Ogro, coluna 3, linha 2' }))
    await user.click(screen.getByRole('button', { name: 'Voltar Ogro para onde estava' }))

    expect(rt.updateToken).toHaveBeenLastCalledWith('t1', { x: 3, y: 2 })
  })

  // Sem ter movido, não há para onde voltar — e um botão que não faz nada é
  // pior que botão nenhum.
  it('sem ter movido, não oferece desfazer', async () => {
    const { user } = renderRegion(true, comPecas)

    await user.click(screen.getByRole('button', { name: 'Ogro, coluna 3, linha 2' }))

    expect(screen.queryByRole('button', { name: /Voltar Ogro/ })).not.toBeInTheDocument()
  })

  it('o jogador não recebe o painel nem o botão de nova peça', () => {
    renderRegion(false, comPecas, undefined, { myCharacterIds: new Set([77]) })

    expect(screen.queryByRole('button', { name: /Esconder/ })).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: /Tirar/ })).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: '+ Peça' })).not.toBeInTheDocument()
  })

  /**
   * A emboscada da ALE-178 mora numa peça que o jogador NÃO recebe — e até a
   * ALE-179 ela era, no tabuleiro do mestre, idêntica a uma peça visível. O
   * segredo dependia de ele lembrar de cabeça quem estava escondido, com a mesa
   * esperando.
   */
  it('a peça escondida se anuncia ao mestre, e só a ele', () => {
    renderRegion(true, comPecas)

    expect(
      screen.getByRole('button', { name: /^Porta, .*escondida dos jogadores$/ }),
    ).toBeInTheDocument()
    // A visível não carrega o aviso: o anúncio é do ESTADO, não do vocabulário.
    expect(screen.getByRole('button', { name: 'Ogro, coluna 3, linha 2' })).toBeInTheDocument()
  })

  /**
   * Na tela do jogador o tabuleiro ocupa a superfície inteira, e até a ALE-179
   * saber se ele era o próximo custava TROCAR DE ABA — no meio do turno de
   * outra pessoa, que é justamente quando se decide o que fazer.
   */
  it('o jogador vê quem está na vez e quem vem depois sem sair do mapa', () => {
    renderRegion(false, TABULEIRO, 'e2', { turnIndex: 1 })

    const tira = screen.getByText('Na vez').parentElement
    if (!tira) throw new Error('a tira da vez não montou')
    // A ordem é a da mesa a partir de quem joga agora, e a lista é CIRCULAR:
    // depois do último vem o primeiro, com a rodada seguinte.
    expect(within(tira).getByText('Ogro')).toBeInTheDocument()
    expect(within(tira).getByText('Batedor Élfico')).toBeInTheDocument()
    expect(within(tira).getByText('Sílfide Ladina')).toBeInTheDocument()
  })

  // O mestre tem a iniciativa inteira numa coluna ao lado: repetir três nomes
  // sobre o mapa dele seria ruído sobre informação que já está na tela.
  it('o mestre não ganha a tira: ele já tem a iniciativa inteira ao lado', () => {
    renderRegion(true, TABULEIRO, 'e2', { turnIndex: 1 })

    expect(screen.queryByText('Na vez')).not.toBeInTheDocument()
  })

  it('fora de combate não há vez para anunciar', () => {
    renderRegion(false, TABULEIRO, undefined, { turnIndex: -1 })

    expect(screen.queryByText('Na vez')).not.toBeInTheDocument()
  })

  /**
   * O terreno difícil (T20 p238) é a fatia 4 da ALE-124. O motor sempre soube
   * cobrá-lo — o `PathCost` recebe o chão e o `boardReach` também — e o estado
   * não tinha onde guardá-lo: o mestre não tinha como DECLARAR o brejo.
   */
  it('com o pincel ligado, tocar o tabuleiro pinta em vez de pousar a peça', async () => {
    const { rt, user, container } = renderRegion(true)
    await user.click(screen.getByRole('button', { name: 'Ogro, coluna 3, linha 2' }))

    await user.click(screen.getByRole('button', { name: 'Terreno' }))
    tocaOTabuleiro(container)

    // A CASA exata é conta de pixel, e em jsdom tudo mede zero — a conversão
    // px→quadrado é provada em `board-viewport.test.ts` e o resto é e2e. O que
    // se prova aqui é a ESCOLHA: pintou, e não pousou.
    expect(rt.paintTerrain).toHaveBeenCalled()
    expect(rt.paintTerrain.mock.calls[0]?.[2]).toBe(true)
    expect(rt.updateToken).not.toHaveBeenCalled()
  })

  // Sem o pincel, a mesma casa continua sendo onde a peça pousa.
  it('sem o pincel, a casa continua pousando a peça', async () => {
    const { rt, user } = renderRegion(true)

    await user.click(screen.getByRole('button', { name: 'Ogro, coluna 3, linha 2' }))
    await user.click(screen.getByRole('button', { name: 'Coluna 1, linha 1' }))

    expect(rt.updateToken).toHaveBeenCalledWith('t1', { x: 1, y: 1 })
    expect(rt.paintTerrain).not.toHaveBeenCalled()
  })

  // O chão é público: o jogador precisa ver o que a régua vai lhe cobrar.
  it('a casa difícil se anuncia, e para o jogador também', async () => {
    const comBrejo: BoardState = { ...TABULEIRO, difficult: [{ x: 1, y: 1 }] }
    const { user } = renderRegion(true, comBrejo)

    await user.click(screen.getByRole('button', { name: 'Ogro, coluna 3, linha 2' }))

    expect(
      screen.getByRole('button', { name: 'Coluna 1, linha 1, terreno difícil' }),
    ).toBeInTheDocument()
  })

  it('o jogador não ganha o pincel: o chão é da cena, e a cena é do mestre', () => {
    renderRegion(false)

    expect(screen.queryByRole('button', { name: 'Terreno' })).not.toBeInTheDocument()
  })

  /**
   * A borracha é ferramenta IRMÃ do pincel, e não "clicar de novo apaga": com o
   * pincel pintando por arraste, alternar faria a casa piscar entre brejo e
   * chão limpo debaixo do dedo.
   */
  it('a borracha manda APAGAR, e é ferramenta irmã do pincel', async () => {
    const comBrejo: BoardState = { ...TABULEIRO, difficult: [{ x: 1, y: 1 }] }
    const { rt, user, container } = renderRegion(true, comBrejo)

    await user.click(screen.getByRole('button', { name: 'Terreno' }))
    await user.click(screen.getByRole('button', { name: 'Apagar terreno' }))
    tocaOTabuleiro(container)

    expect(rt.paintTerrain).toHaveBeenCalledTimes(1)
    expect(rt.paintTerrain.mock.calls[0]?.[2]).toBe(false)
  })

  // Sem ferramenta na mão, o tabuleiro volta a ser tabuleiro.
  it('sem pincel, tocar o tabuleiro não pinta nada', async () => {
    const { rt, user, container } = renderRegion(true)
    await user.click(screen.getByRole('button', { name: 'Ogro, coluna 3, linha 2' }))

    tocaOTabuleiro(container)

    expect(rt.paintTerrain).not.toHaveBeenCalled()
  })

  // As setas e o −/+ saíram: arrastar e a roda fazem o mesmo, melhor, e o
  // cabeçalho já quebrava linha no telefone com seis botões a mais.
  it('o cabeçalho não carrega mais os controles que o gesto faz', () => {
    renderRegion(true)

    expect(screen.queryByRole('button', { name: 'Mover a vista para a esquerda' })).toBeNull()
    expect(screen.queryByRole('button', { name: 'Aproximar' })).toBeNull()
    // O que gesto nenhum faz continua: achar o grupo num plano infinito.
    expect(screen.getByRole('button', { name: 'Centralizar nas peças' })).toBeInTheDocument()
  })

  /**
   * O botão direito apaga — é o gesto que todo editor de mapa tem — e a
   * ferramenta selecionada muda JUNTO: a tela não pode dizer "pincel" enquanto
   * a mão apaga.
   */
  it('o botão direito apaga, e a ferramenta na tela conta a verdade', async () => {
    const { rt, user, container } = renderRegion(true)
    await user.click(screen.getByRole('button', { name: 'Terreno' }))

    apagaComOBotaoDireito(container)

    expect(rt.paintTerrain.mock.calls[0]?.[2]).toBe(false)
    expect(screen.getByRole('button', { name: 'Apagar terreno' })).toHaveAttribute(
      'aria-pressed',
      'true',
    )
  })

  /**
   * A peça e a linha da iniciativa são a MESMA criatura: apontar a peça e ter de
   * procurar o nome na lista para ver os PV é trabalho que o app pode poupar.
   */
  /**
   * PEGAR a peça e LER a ficha são gestos diferentes (ALE-198).
   *
   * Eram o mesmo clique enquanto a ficha era uma aba escondida atrás de um
   * degrau de largura — abrir uma aba que ninguém está vendo não atrapalha
   * ninguém. A ficha virou DIÁLOGO, e um diálogo entre o pegar e o pousar cobre
   * o mapa: o clique do pousar caía dentro dele. Provado no e2e de dois
   * clientes, que passou a estourar por timeout com o seletor "Aplicar
   * efeito…" aberto por cima do tabuleiro no artefato.
   */
  it('escolher a peça NÃO abre a ficha: o clique é para pegar e mover', async () => {
    const { user, onOpenCombatant } = renderRegion(true)

    await user.click(screen.getByRole('button', { name: 'Ogro, coluna 3, linha 2' }))

    expect(onOpenCombatant).not.toHaveBeenCalled()
  })

  it('o painel da peça abre a ficha de quem ela representa', async () => {
    const { user, onOpenCombatant } = renderRegion(true)

    await user.click(screen.getByRole('button', { name: 'Ogro, coluna 3, linha 2' }))
    await user.click(screen.getByRole('button', { name: 'Ficha de Ogro' }))

    expect(onOpenCombatant).toHaveBeenCalledWith('e1')
  })

  // A peça avulsa (porta, baú) não tem linha na iniciativa — e não há ficha.
  it('a peça sem linha na iniciativa não oferece ficha nenhuma', async () => {
    const { user } = renderRegion(true)

    await user.click(screen.getByRole('button', { name: 'Sílfide Ladina, coluna 6, linha 5' }))

    expect(screen.queryByRole('button', { name: /^Ficha de/ })).not.toBeInTheDocument()
  })

})

/**
 * Lugares da campanha (ALE-124, fatia 5). A épica prometia que encerrar ARQUIVA —
 * "taverna → masmorra → de volta à taverna, com tudo onde estava" — e até esta
 * fatia encerrar DESTRUÍA a cena montada. Era a única promessa que o código
 * contradizia.
 */
describe('os lugares guardados da campanha', () => {
  const TAVERNA: BoardPlace = {
    id: 7,
    name: 'Taverna do Javali',
    tokens: 9,
    updatedAt: '2026-08-19T00:00:00Z',
  }

  /**
   * O acervo é uma LISTA, e a tela vazia que o hospedava é a região mais
   * estreita da cena — 374px numa janela de 1920, porque sem tabuleiro a
   * coluna do meio recebe 3fr (ALE-161). Desenhada ali, a lista pedia os
   * 384px do `max-w-sm` e TRANSBORDAVA: a lixeira saía da tela. Na região
   * larga do laptop (~840px) a mesma medida fixa deixava 456px de vão morto.
   *
   * A lista sai da tela vazia e passa a viver só no diálogo que já existia
   * para trocar de cena — um desenho só do acervo, em vez de dois (ALE-198).
   */
  it('a tela vazia não desenha o acervo: ela oferece o botão que o abre', async () => {
    const { user } = renderRegion(true, null, undefined, { places: [TAVERNA] })

    const abrir = await screen.findByRole('button', { name: /Lugares da campanha/ })
    expect(screen.queryByText('9 peças')).not.toBeInTheDocument()

    await user.click(abrir)
    // A contagem é o que faz o mestre reconhecer a cena: "a taverna, aquela dos nove".
    expect(within(await screen.findByRole('dialog')).getByText('9 peças')).toBeInTheDocument()
  })

  it('sem tabuleiro, o mestre reabre uma cena guardada', async () => {
    const { rt, user } = renderRegion(true, null, undefined, { places: [TAVERNA] })

    await user.click(await screen.findByRole('button', { name: /Lugares da campanha/ }))
    const dialogo = await screen.findByRole('dialog')
    await user.click(within(dialogo).getByRole('button', { name: 'Reabrir' }))

    expect(rt.reopenPlace).toHaveBeenCalledWith(7)
  })

  // Apagar é o único caminho que destrói uma cena montada — e o único que
  // pergunta antes.
  it('apagar um lugar pede confirmação', async () => {
    const { user } = renderRegion(true, null, undefined, { places: [TAVERNA] })

    await user.click(await screen.findByRole('button', { name: /Lugares da campanha/ }))
    const acervo = await screen.findByRole('dialog')
    await user.click(within(acervo).getByRole('button', { name: 'Apagar Taverna do Javali' }))

    // O confirmar é modal: ele marca o acervo atrás como `aria-hidden`, e a
    // consulta por papel passa a devolver só ele (gotcha do guia do front).
    const dialogo = await screen.findByRole('dialog', { name: /Apagar Taverna do Javali\?/ })
    await user.click(within(dialogo).getByRole('button', { name: 'Apagar' }))
    await waitFor(() => expect(screen.queryByText('Taverna do Javali')).not.toBeInTheDocument())
  })

  // O acervo é preparação do mestre: saber que existe uma "Cripta do
  // Necromante" guardada é meio caminho da surpresa.
  it('o jogador não vê o acervo de cenas', async () => {
    renderRegion(false, null, undefined, { places: [TAVERNA] })

    await waitFor(() =>
      expect(screen.getByText('O mestre ainda não abriu um tabuleiro.')).toBeInTheDocument(),
    )
    expect(screen.queryByText('Lugares da campanha')).not.toBeInTheDocument()
  })
})

/**
 * "Ver como jogador" (ALE-193).
 *
 * A redação por papel já era forte e testada — a peça escondida some INTEIRA da
 * cópia da mesa (`redactBoardForPlayers`) —, mas o mestre não tinha como olhar
 * o resultado: conferir a emboscada exigia abrir DOIS navegadores com dois
 * logins, que foi literalmente como a ALE-178 foi verificada.
 *
 * O que se prova aqui é o que o mestre nota: a peça escondida some da cena
 * dele, a tira diz quantas são, e a lente ACOMPANHA a cena em vez de congelar
 * no instante em que foi ligada.
 */
describe('a lente do mestre sobre a cena da mesa', () => {
  const ASSASSINO = {
    id: 't4',
    label: 'Assassino',
    x: 2,
    y: 2,
    footprint: 1,
    kind: 'npc' as const,
    hidden: true,
  }
  const EMBOSCADA: BoardState = { ...TABULEIRO, tokens: [...TABULEIRO.tokens, ASSASSINO] }

  it('a peça escondida some da cena, e a tira diz quantas a mesa não vê', async () => {
    const { user } = renderRegion(true, EMBOSCADA)
    expect(screen.getByRole('button', { name: /^Assassino,/ })).toBeInTheDocument()

    await user.click(screen.getByRole('button', { name: 'Ver como jogador' }))

    await waitFor(() =>
      expect(screen.queryByRole('button', { name: /^Assassino,/ })).not.toBeInTheDocument(),
    )
    // O número é a resposta que trouxe o mestre até aqui: contar o que sumiu da
    // tela não serve, porque ele não sabe o que não está vendo.
    expect(screen.getByRole('status')).toHaveTextContent('1 peça escondida não aparece')
    // A lente é sobre a CENA, não sobre as ferramentas: ele confere a emboscada
    // sem parar de montá-la.
    expect(screen.getByRole('button', { name: /Trazer a iniciativa/ })).toBeInTheDocument()
  })

  // Um modo que se esquece é pior que nenhum: o mestre que não percebe onde
  // está conclui que a peça que ele mesmo escondeu sumiu de verdade.
  it('a saída é a própria tira, e a cena do mestre volta inteira', async () => {
    const { user } = renderRegion(true, EMBOSCADA)
    await user.click(screen.getByRole('button', { name: 'Ver como jogador' }))
    await waitFor(() =>
      expect(screen.queryByRole('button', { name: /^Assassino,/ })).not.toBeInTheDocument(),
    )

    await user.click(screen.getByRole('button', { name: 'Voltar à vista do mestre' }))

    expect(await screen.findByRole('button', { name: /^Assassino,/ })).toBeInTheDocument()
    expect(screen.queryByRole('status')).not.toBeInTheDocument()
  })

  // A cena muda embaixo da lente o tempo todo — é o mestre montando. Uma lente
  // congelada no instante em que foi ligada responde a pergunta de dois
  // movimentos atrás, que é a pergunta errada.
  it('revelar a peça enquanto a lente está ligada mostra a mesa passando a vê-la', async () => {
    const [live, setLive] = createSignal<BoardState | null>(EMBOSCADA)
    const { user, rt } = renderRegion(true, live)

    await user.click(screen.getByRole('button', { name: 'Ver como jogador' }))
    await waitFor(() =>
      expect(screen.queryByRole('button', { name: /^Assassino,/ })).not.toBeInTheDocument(),
    )

    // O mestre revela a peça: é a mesma mutação que o servidor transmitiria,
    // com a versão subindo.
    setLive({
      ...EMBOSCADA,
      version: EMBOSCADA.version + 1,
      tokens: [...TABULEIRO.tokens, { ...ASSASSINO, hidden: false }],
    })

    expect(await screen.findByRole('button', { name: /^Assassino,/ })).toBeInTheDocument()
    expect(rt.boardAsPlayer).toHaveBeenCalledTimes(2)
  })
})

/**
 * Trocar de cena com a mesa jogando (ALE-191).
 *
 * Até aqui os Lugares só existiam na tela VAZIA: para levar o grupo da taverna
 * à cripta, o mestre tinha de ENCERRAR o tabuleiro — a mesa via a grade sumir e
 * voltar — e o `reopen`, por baixo, trocava a cena viva sem guardar o que estava
 * nela. Quem guarda agora é o servidor; o que se prova aqui é o caminho na tela.
 */
describe('mostrar outro lugar à mesa', () => {
  const CRIPTA: BoardPlace = {
    id: 12,
    name: 'Cripta do Necromante',
    tokens: 6,
    updatedAt: '2026-08-19T00:00:00Z',
  }
  // O acervo guarda TAMBÉM a cena que está na mesa: ela foi arquivada na última
  // vez que o mestre a encerrou, e é assim que ele a reconhece na lista.
  const TAVERNA: BoardPlace = {
    id: 7,
    name: 'Taverna do Javali',
    tokens: 9,
    updatedAt: '2026-08-19T00:00:00Z',
  }

  it('o mestre troca de cena pelo acervo, sem encerrar o tabuleiro', async () => {
    const { rt, user } = renderRegion(true, TABULEIRO, undefined, { places: [TAVERNA, CRIPTA] })

    await user.click(screen.getByRole('button', { name: 'Lugares da campanha' }))
    await user.click(await screen.findByRole('button', { name: /Mostrar à mesa/ }))

    // A pergunta nomeia as DUAS cenas: para onde a mesa vai, e o que acontece
    // com a que estava lá.
    const dialogos = await screen.findAllByRole('dialog')
    const pergunta = dialogos[dialogos.length - 1]
    expect(within(pergunta).getByText(/Mostrar Cripta do Necromante à mesa\?/)).toBeInTheDocument()
    expect(within(pergunta).getByText(/Taverna do Javali, vai para os Lugares/)).toBeInTheDocument()

    await user.click(within(pergunta).getByRole('button', { name: 'Mostrar à mesa' }))

    expect(rt.reopenPlace).toHaveBeenCalledWith(12)
  })

  // Mandar para a mesa o que já está nela é um caminho que só pode confundir —
  // e, no servidor, seria guardar a taverna para reabrir a taverna.
  it('a cena que já está na mesa não se oferece para ir à mesa', async () => {
    const { user } = renderRegion(true, TABULEIRO, undefined, { places: [TAVERNA, CRIPTA] })

    await user.click(screen.getByRole('button', { name: 'Lugares da campanha' }))
    const acervo = await screen.findByRole('dialog')

    expect(within(acervo).getByText('Na mesa')).toBeInTheDocument()
    expect(within(acervo).getAllByRole('button', { name: /Mostrar à mesa/ })).toHaveLength(1)
  })
})


/**
 * Montar a próxima cena com a mesa jogando (ALE-191, fatia 2).
 *
 * O rascunho é LOCAL: o mestre monta, e só a gravação chega ao servidor. Nada
 * daqui é transmitido — a mesa continua na taverna e não fica sabendo que a
 * cripta existe, que é meio caminho da surpresa.
 */
describe('montar um lugar do acervo', () => {
  const CRIPTA: BoardPlace = {
    id: 12,
    name: 'Cripta do Necromante',
    tokens: 1,
    updatedAt: '2026-08-19T00:00:00Z',
  }
  const CENA_DA_CRIPTA: BoardState = {
    version: 4,
    place: 'Cripta do Necromante',
    terrain: 'masmorra',
    tokens: [{ id: 'n1', label: 'Necromante', x: 1, y: 1, footprint: 1, kind: 'npc' }],
  }

  const montar = async () => {
    const tudo = renderRegion(true, TABULEIRO, undefined, { places: [CRIPTA], scene: CENA_DA_CRIPTA })
    await tudo.user.click(screen.getByRole('button', { name: 'Lugares da campanha' }))
    await tudo.user.click(await screen.findByRole('button', { name: 'Montar Cripta do Necromante' }))
    return tudo
  }

  it('o mestre monta a cripta enquanto a mesa continua na taverna', async () => {
    await montar()

    // A cena guardada está na tela para ser montada...
    expect(await screen.findByRole('button', { name: /^Necromante,/ })).toBeInTheDocument()
    // ...e o crachá diz, pelo NOME, o que a mesa está vendo enquanto isso.
    expect(screen.getByRole('status')).toHaveTextContent(/A mesa continua vendo Taverna do Javali/)
    // A cena da mesa sai da tela: são duas cenas, e desenhar as duas juntas
    // faria o mestre montar sobre o mapa errado.
    expect(screen.queryByRole('button', { name: /^Ogro,/ })).not.toBeInTheDocument()
  })

  it('a peça que nasce no rascunho pode ser posicionada, e a gravação leva tudo', async () => {
    const { rt, user } = await montar()
    await screen.findByRole('button', { name: /^Necromante,/ })

    // Posicionar a peça que já estava guardada: selecionar e pousar.
    await user.click(screen.getByRole('button', { name: /^Necromante,/ }))
    await user.click(screen.getByRole('button', { name: 'Coluna 3, linha 2' }))
    await user.click(screen.getByRole('button', { name: 'Guardar a cena' }))

    expect(rt.savePlace).toHaveBeenCalledTimes(1)
    const [placeId, cena] = rt.savePlace.mock.calls[0] as unknown as [number, BoardState]
    expect(placeId).toBe(12)
    expect(cena.tokens).toEqual([expect.objectContaining({ id: 'n1', x: 3, y: 2 })])
  })

  // Sair sem guardar é jogar o rascunho fora — e a mesa volta à tela inteira.
  it('sair sem guardar não grava nada e devolve a cena da mesa', async () => {
    const { rt, user } = await montar()
    await screen.findByRole('button', { name: /^Necromante,/ })

    await user.click(screen.getByRole('button', { name: 'Sair sem guardar' }))

    expect(rt.savePlace).not.toHaveBeenCalled()
    expect(await screen.findByRole('button', { name: /^Ogro,/ })).toBeInTheDocument()
  })
})


/**
 * O teclado na superfície (ALE-194).
 *
 * Duas dívidas se encontram aqui. A minha: ao tirar as quatro setas e o −/+ do
 * cabeçalho (`c95d502`), mover e ampliar passaram a existir só no ponteiro. E a
 * do teste prático do Roll20, que mostra o caminho certo — que não é devolver
 * os botões: a seta move a PEÇA um quadrado, que é a unidade do estado.
 *
 * A regra que não pode quebrar: a seta chama o MESMO caminho do clique. O
 * mestre posiciona livre, o jogador propõe — com a vez e o orçamento conferidos
 * no servidor. Atalho que fura a regra do livro é pior que atalho nenhum.
 */
describe('o teclado do tabuleiro', () => {
  const superficie = (container: HTMLElement) => {
    const plano = container.querySelector('[role="grid"]')
    if (!plano) throw new Error('o tabuleiro não montou')
    return plano
  }

  it('com peça na mão, a seta move a peça um quadrado', async () => {
    const { rt, user, container } = renderRegion(true)

    await user.click(screen.getByRole('button', { name: 'Ogro, coluna 3, linha 2' }))
    fireEvent.keyDown(superficie(container), { key: 'ArrowRight' })

    expect(rt.updateToken).toHaveBeenCalledWith('t1', { x: 4, y: 2 })
  })

  // A peça continua na mão depois da seta: mover de um em um é o gesto, e
  // largar a cada passo obrigaria a selecionar de novo para cada quadrado.
  it('a peça continua selecionada, e a segunda seta continua de onde parou', async () => {
    const [live, setLive] = createSignal<BoardState | null>(TABULEIRO)
    const { rt, user, container } = renderRegion(true, live)

    await user.click(screen.getByRole('button', { name: 'Ogro, coluna 3, linha 2' }))
    fireEvent.keyDown(superficie(container), { key: 'ArrowRight' })
    // O servidor responde com a peça no lugar novo, como responderia de verdade.
    setLive({
      ...TABULEIRO,
      version: TABULEIRO.version + 1,
      tokens: TABULEIRO.tokens.map((peca) => (peca.id === 't1' ? { ...peca, x: 4 } : peca)),
    })
    fireEvent.keyDown(superficie(container), { key: 'ArrowDown' })

    expect(rt.updateToken).toHaveBeenLastCalledWith('t1', { x: 4, y: 3 })
  })

  // Sem peça na mão a seta move a JANELA: é o que os botões de seta davam ao
  // teclado antes de saírem, e num plano infinito sem isso metade da cena fica
  // inalcançável para quem não usa mouse.
  it('sem peça na mão, a seta move a vista', () => {
    const { container } = renderRegion(true)
    const antes = superficie(container).getAttribute('aria-label')

    fireEvent.keyDown(superficie(container), { key: 'ArrowRight' })

    expect(superficie(container).getAttribute('aria-label')).not.toBe(antes)
  })

  // A tecla que responde "onde está o grupo?" num plano sem bordas.
  it('Home enquadra as peças, inclusive as que estão fora da janela', async () => {
    const longe: BoardState = {
      ...TABULEIRO,
      tokens: [{ id: 't9', label: 'Sentinela Distante', x: 60, y: 0, footprint: 1, kind: 'npc' }],
    }
    const { container } = renderRegion(true, longe)
    expect(screen.queryByRole('button', { name: /Sentinela Distante/ })).not.toBeInTheDocument()

    fireEvent.keyDown(superficie(container), { key: 'Home' })

    expect(await screen.findByRole('button', { name: /Sentinela Distante/ })).toBeInTheDocument()
  })

  // O jogador PROPÕE pela seta, como propõe pelo clique: quem confere a vez e o
  // orçamento é o servidor, e o atalho não pode ser um caminho por fora.
  it('a seta do jogador propõe o caminho em vez de pousar a peça', async () => {
    const { rt, user, container } = renderRegion(false, COM_JOGADOR, 'e1', {
      myCharacterIds: MEU_HEROI,
      turnIndex: 0,
    })

    await user.click(screen.getByRole('button', { name: 'Sílfide Ladina, coluna 6, linha 5' }))
    fireEvent.keyDown(superficie(container), { key: 'ArrowRight' })

    expect(rt.proposeMove).toHaveBeenCalledWith('t2', [
      { x: 6, y: 5 },
      { x: 7, y: 5 },
    ])
    expect(rt.updateToken).not.toHaveBeenCalled()
  })

  /**
   * O foco sobrevive ao broadcast (ALE-194).
   *
   * O `For` do Solid reconcilia por REFERÊNCIA, e todo broadcast desta casa
   * troca o estado INTEIRO: cada peça virava um botão novo a cada mensagem da
   * mesa, e o foco caía no `body`. Com o teclado o sintoma ficou visível — a
   * primeira seta movia a peça e a segunda não fazia nada, porque a primeira
   * tinha acabado de destruir o botão em foco. Quem usa leitor de tela perdia o
   * lugar toda vez que QUALQUER peça se mexia.
   *
   * Achado no BROWSER, onde a tecla passa pelo foco de verdade; nos testes o
   * evento era disparado na superfície e não via nada.
   */
  it('a peça em foco continua em foco quando outra peça se mexe', () => {
    const [live, setLive] = createSignal<BoardState | null>(TABULEIRO)
    renderRegion(true, live)
    const ogro = screen.getByRole('button', { name: 'Ogro, coluna 3, linha 2' })
    ogro.focus()

    // O estado chega do SOCKET, então toda peça é um objeto NOVO — e é aí que
    // o `For` recria tudo. Clonar só a que mudou faria este teste passar verde
    // sobre o defeito, porque as referências das outras sobreviveriam.
    setLive({
      ...TABULEIRO,
      version: TABULEIRO.version + 1,
      tokens: TABULEIRO.tokens.map((peca) => ({ ...peca, x: peca.id === 't2' ? 7 : peca.x })),
    })

    expect(screen.getByRole('button', { name: 'Ogro, coluna 3, linha 2' })).toHaveFocus()
  })

  // A segunda seta continua do fim do caminho JÁ proposto: o provisório não
  // move a peça, ele a promete — sem isso o jogador ficaria preso a um
  // quadrado, propondo o mesmo passo de novo.
  it('com um caminho proposto, a seta seguinte o ESTENDE', async () => {
    const proposto: BoardState = {
      ...COM_JOGADOR,
      pending: {
        tokenId: 't2',
        path: [
          { x: 6, y: 5 },
          { x: 7, y: 5 },
        ],
        cost: 1,
        budget: 6,
        diagonals: 0,
        difficult: 0,
        byUserId: 1,
      },
    }
    const { rt, user, container } = renderRegion(false, proposto, 'e1', {
      myCharacterIds: MEU_HEROI,
      turnIndex: 0,
    })

    await user.click(screen.getByRole('button', { name: 'Sílfide Ladina, coluna 6, linha 5' }))
    fireEvent.keyDown(superficie(container), { key: 'ArrowRight' })

    expect(rt.proposeMove).toHaveBeenCalledWith('t2', [
      { x: 6, y: 5 },
      { x: 7, y: 5 },
      { x: 8, y: 5 },
    ])
  })
})


/**
 * A régua da mesa (ALE-124, fatia 6).
 *
 * "Dá para acertar daqui?" é pergunta de toda rodada, e até aqui se respondia
 * contando quadrado com o dedo na tela — que é justamente o que um tabuleiro
 * digital deveria poupar.
 *
 * Estes testes rodam o MOTOR GO de verdade pelo wasm, o mesmo que o servidor
 * usa: o que se prova não é a aritmética (essa está provada contra a p224 lá),
 * é a LIGAÇÃO — o clique vira medida e a medida vira a frase que a mesa lê.
 */
describe('a régua', () => {
  it('mede entre dois quadrados e diz a faixa de alcance do livro', async () => {
    const { user } = renderRegion(true)

    await user.click(screen.getByRole('button', { name: 'Régua' }))
    await user.click(screen.getByRole('button', { name: 'Coluna 0, linha 0' }))
    await user.click(screen.getByRole('button', { name: 'Coluna 4, linha 4' }))

    // Quatro passos na diagonal são OITO quadrados: a diagonal custa o dobro
    // (p238), e é a mesma régua do movimento — o alvo que outros jogos poriam
    // no alcance curto, aqui está no médio (p224).
    expect(screen.getByRole('status')).toHaveTextContent('8 quadrados (12,0m) · alcance médio')
  })

  // Medir é de quem joga, não só de quem mestra: quem pergunta "dá para acertar
  // daqui?" é quem vai atacar.
  it('o jogador também tem régua', async () => {
    const { user } = renderRegion(false, COM_JOGADOR, 'e1', {
      myCharacterIds: MEU_HEROI,
      turnIndex: 0,
    })

    await user.click(screen.getByRole('button', { name: 'Régua' }))
    await user.click(screen.getByRole('button', { name: 'Coluna 0, linha 0' }))
    await user.click(screen.getByRole('button', { name: 'Coluna 6, linha 0' }))

    expect(screen.getByRole('status')).toHaveTextContent('6 quadrados (9,0m) · alcance curto')
  })

  // Com a régua na mão o clique numa casa MEDE: o clique já tinha dono, e dois
  // donos para o mesmo gesto é como se perde a peça de lugar sem querer.
  it('com a régua ligada, o clique não pousa a peça', async () => {
    const { rt, user } = renderRegion(true)

    await user.click(screen.getByRole('button', { name: 'Ogro, coluna 3, linha 2' }))
    await user.click(screen.getByRole('button', { name: 'Régua' }))
    await user.click(screen.getByRole('button', { name: 'Coluna 5, linha 3' }))

    expect(rt.updateToken).not.toHaveBeenCalled()
  })
})


/**
 * Os gabaritos de área (ALE-124, fatia 6b — T20 p225).
 *
 * A pergunta que eles respondem trava o turno do conjurador: "se eu soltar a
 * bola de fogo aqui, quem pega?". Ela se responde hoje apontando o dedo na tela
 * e discutindo, com a mesa parada.
 *
 * A FORMA é do motor Go, transcrita da figura da p225 e provada lá contra as
 * contagens dela. O que se prova aqui é a ligação e o cruzamento com as peças —
 * e estes testes rodam o motor de verdade pelo wasm.
 */
describe('o gabarito de área', () => {
  it('a esfera pega quem está debaixo dela, e a barra diz o nome', async () => {
    const { user } = renderRegion(true)

    await user.click(screen.getByRole('button', { name: 'Gabarito de área' }))
    // O padrão é esfera de raio 2 — o "Raio de 3m" da figura, 12 casas.
    // A interseção em (5,4) põe a área sobre (4,3) e (3,3), que são CORPO do
    // Ogro (2×2 a partir de (3,2)), e deixa de fora a casa do canto dele: quem
    // olhasse só a coordenada da peça diria que ela escapou.
    await user.click(screen.getByRole('button', { name: 'Coluna 5, linha 4' }))

    // Basta UM quadrado do corpo na área — exigir a peça inteira deixaria a
    // Colossal de fora do próprio incêndio (p107).
    expect(screen.getByRole('status')).toHaveTextContent('Pega 1 peça: Ogro')
  })

  // Cone e linha PRECISAM apontar, e a barra diz isso em vez de desenhar um
  // gabarito para um lado que ninguém escolheu.
  it('o cone pede a direção no segundo clique', async () => {
    const { user } = renderRegion(true)

    await user.click(screen.getByRole('button', { name: 'Gabarito de área' }))
    await user.click(screen.getByRole('button', { name: 'Cone' }))
    await user.click(screen.getByRole('button', { name: 'Aumentar alcance' }))
    await user.click(screen.getByRole('button', { name: 'Coluna 0, linha 2' }))

    expect(screen.getByRole('status')).toHaveTextContent('Clique de novo para apontar')

    // Apontado para a direita, o cone de 3 quadrados alcança a coluna 3 e pega
    // o Ogro, que ocupa (3,2).
    await user.click(screen.getByRole('button', { name: 'Coluna 5, linha 2' }))

    expect(screen.getByRole('status')).toHaveTextContent('Pega 1 peça: Ogro')
  })

  // Achado no BROWSER: trocar de gabarito mantinha a origem do anterior, e o
  // primeiro clique depois da troca caía na regra do SEGUNDO clique — escolher
  // "Cone" com uma esfera na tela desenhava um cone apontado a partir dela.
  it('trocar de gabarito larga o que estava posto', async () => {
    const { user } = renderRegion(true)

    await user.click(screen.getByRole('button', { name: 'Gabarito de área' }))
    await user.click(screen.getByRole('button', { name: 'Coluna 5, linha 4' }))
    expect(screen.getByRole('status')).toHaveTextContent('Pega 1 peça: Ogro')

    await user.click(screen.getByRole('button', { name: 'Cone' }))

    expect(screen.getByRole('status')).toHaveTextContent('Clique numa casa para pôr o gabarito')
  })

  // Como a régua: quem pergunta "quem pega?" é quem vai conjurar.
  it('o jogador também tem gabarito', async () => {
    renderRegion(false, COM_JOGADOR, 'e1', { myCharacterIds: MEU_HEROI, turnIndex: 0 })

    expect(screen.getByRole('button', { name: 'Gabarito de área' })).toBeInTheDocument()
  })
})


/**
 * Duplicar peça (ALE-192).
 *
 * "Mais um zumbi" é a operação mais repetida ao montar encontro, e ela custava
 * abrir a forma e digitar o nome de uma criatura idêntica à que já está ali ao
 * lado. Quem NUMERA a cópia é o servidor — a regra e a prova de que "Zumbi 1"
 * vira "Zumbi 3" estão no Go, com a mesma tabela de exemplos do parser de
 * aparência. Aqui se prova o caminho: o mestre tem o botão e o jogador não.
 */
describe('duplicar peça', () => {
  it('o mestre duplica a peça selecionada', async () => {
    const { rt, user } = renderRegion(true)

    await user.click(screen.getByRole('button', { name: 'Ogro, coluna 3, linha 2' }))
    await user.click(screen.getByRole('button', { name: 'Duplicar Ogro' }))

    expect(rt.duplicateToken).toHaveBeenCalledWith('t1')
  })

  it('o jogador não duplica peça nenhuma', async () => {
    const { user } = renderRegion(false, COM_JOGADOR, 'e1', {
      myCharacterIds: MEU_HEROI,
      turnIndex: 0,
    })

    await user.click(screen.getByRole('button', { name: 'Sílfide Ladina, coluna 6, linha 5' }))

    expect(screen.queryByRole('button', { name: /Duplicar/ })).not.toBeInTheDocument()
  })
})


/**
 * PV e caído na peça (ALE-188).
 *
 * O mestre olha o MAPA para decidir o foco do turno — "quem eu derrubo neste
 * turno" é a decisão tática da rodada — e precisava voltar os olhos para a
 * LISTA só para saber quem está caindo.
 *
 * O filete de PV é desenho, e a casa não afirma classe em teste. O que se prova
 * aqui é o que muda para quem LÊ a tela: a peça caída se anuncia, e a redação
 * por papel a alcança.
 */
describe('a peça conta os PV', () => {
  it('a 0 PV a peça se anuncia caída', () => {
    renderRegion(true, TABULEIRO, undefined, { vitals: { e1: { hpCurrent: 0, hpMax: 130 } } })

    // O Ogro é a peça da linha `e1` neste tabuleiro.
    expect(screen.getByRole('button', { name: /^Ogro,.*caída$/ })).toBeInTheDocument()
  })

  // O PV oculto é redação por papel: o servidor apaga os números da cópia do
  // jogador (`redactForPlayers`), então a peça não tem o que mostrar — e é
  // assim que a política chega ao tabuleiro sem uma SEGUNDA política.
  it('sem os números, a peça do jogador não diz nada de PV', () => {
    renderRegion(false, TABULEIRO, undefined, {})

    expect(screen.queryByRole('button', { name: /caída/ })).not.toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Ogro, coluna 3, linha 2' })).toBeInTheDocument()
  })
})


/**
 * O lugar marcado no mapa (ALE-195).
 *
 * Nem tudo que importa é criatura ou móvel: a armadilha, a porta que range, o
 * ponto de encontro. A saída de hoje é criar uma PEÇA de cenário — e peça ocupa
 * quadrado, vira alvo e entra na conta de quem o gabarito pega.
 */
describe('marcar um lugar', () => {
  const COM_MARCADOR: BoardState = {
    ...TABULEIRO,
    markers: [{ id: 'm1', x: 1, y: 1, text: 'A', color: 'ouro', hidden: true }],
  }

  it('o mestre marca a casa, e o marcador nasce ESCONDIDO com a próxima letra', async () => {
    const { rt, user } = renderRegion(true, COM_MARCADOR)

    await user.click(screen.getByRole('button', { name: 'Marcar um lugar' }))
    await user.click(screen.getByRole('button', { name: 'Coluna 5, linha 3' }))

    // "A" já está em uso: o próximo é o "B". Quem aponta a armadilha no meio da
    // cena não quer digitar.
    expect(rt.addMarker).toHaveBeenCalledWith({ x: 5, y: 3, text: 'B', color: 'ouro', hidden: true })
  })

  // Revelar é o verbo que importa: marcar antes e mostrar quando alguém pisa.
  it('o mestre revela o marcador escondido', async () => {
    const { rt, user } = renderRegion(true, COM_MARCADOR)

    await user.click(screen.getByRole('button', { name: /^Marcador A,.*escondido dos jogadores$/ }))
    await user.click(screen.getByRole('button', { name: 'Mostrar marcador A' }))

    expect(rt.updateMarker).toHaveBeenCalledWith('m1', { hidden: false })
  })

  // O marcador do jogador é DESENHO: o lugar apontado não é para ele mexer, e o
  // escondido nem chega até ele (o servidor o apaga da cópia).
  it('o jogador não mexe no marcador', () => {
    const visivel: BoardState = {
      ...TABULEIRO,
      markers: [{ id: 'm2', x: 1, y: 1, text: 'C', color: 'carmim' }],
    }
    renderRegion(false, visivel)

    expect(screen.getByRole('button', { name: /^Marcador C,/ })).toBeDisabled()
    expect(screen.queryByRole('button', { name: 'Marcar um lugar' })).not.toBeInTheDocument()
  })
})

/**
 * QUEM vem para o tabuleiro (ALE-204).
 *
 * "Trazer a iniciativa" trazia a fila INTEIRA, e a fila inclui quem o mestre
 * montou para aparecer depois: num clique a emboscada virava peça na tela da
 * mesa, e desfazer era peça por peça.
 *
 * A fixture já tem o caso todo: a Sílfide (`e1`) JÁ tem peça, o Batedor (`e3`)
 * é jogador e não tem, e o Ogro (`e2`) é do mestre.
 */
describe('quem vem para o tabuleiro', () => {
  const abrirATrazer = () => screen.getByRole('button', { name: /Trazer a iniciativa/ })

  // O atalho: pôr os PCs no mapa é o gesto de abrir cena e não revela nada —
  // eles sabem onde estão. O Ogro é informação do mestre, e informação do
  // mestre não sai por atalho.
  it('o clique direito traz só os jogadores', () => {
    const { rt } = renderRegion(true)

    fireEvent.contextMenu(abrirATrazer())

    expect(rt.populateBoard).toHaveBeenCalledWith(['e3'])
    // E o atalho não abre nada: ele é o caminho de um gesto só.
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
  })

  // O cuidado que a issue nomeia: o clique direito JÁ tem dono no tabuleiro —
  // ele apaga terreno (ALE-192/178). O botão vive no CABEÇALHO, que não é
  // ancestral da superfície, e é isso que mantém os dois gestos separados; pôr
  // o botão dentro do plano faria o mesmo gesto trazer peça E apagar chão.
  it('o clique direito no botão não apaga terreno', async () => {
    const { rt, user } = renderRegion(true)
    await user.click(screen.getByRole('button', { name: 'Apagar terreno' }))

    const botao = abrirATrazer()
    const toque = new Event('pointerdown', { bubbles: true })
    Object.assign(toque, { pointerId: 1, clientX: 10, clientY: 10, button: 2, buttons: 2 })
    botao.dispatchEvent(toque)
    fireEvent.contextMenu(botao)

    expect(rt.populateBoard).toHaveBeenCalledWith(['e3'])
    expect(rt.paintTerrain).not.toHaveBeenCalled()
  })

  // O defeito da issue, em uma linha: o clique esquerdo PERGUNTA.
  it('o clique esquerdo não põe ninguém no mapa — ele pergunta', async () => {
    const { rt, user } = renderRegion(true)

    await user.click(abrirATrazer())

    expect(screen.getByRole('dialog')).toBeInTheDocument()
    expect(rt.populateBoard).not.toHaveBeenCalled()
  })

  // A regra da casa: gesto NUNCA é o único caminho. Abrir e confirmar faz
  // exatamente o que o clique direito faz, sem gesto nenhum.
  it('abrir e confirmar faz o mesmo que o atalho', async () => {
    const { rt, user } = renderRegion(true)

    await user.click(abrirATrazer())
    await user.click(screen.getByRole('button', { name: /^Trazer \d/ }))

    expect(rt.populateBoard).toHaveBeenCalledWith(['e3'])
  })

  // E o diálogo sabe fazer MAIS: o vilão vem quando o mestre o marca, e só aí.
  it('o vilão só entra na lista depois de marcado', async () => {
    const { rt, user } = renderRegion(true)

    await user.click(abrirATrazer())
    const ogro = screen.getByRole('button', { name: 'Ogro' })
    expect(ogro).toHaveAttribute('aria-pressed', 'false')

    await user.click(ogro)
    await user.click(screen.getByRole('button', { name: /^Trazer \d/ }))

    expect([...rt.populateBoard.mock.calls[0][0]].sort()).toEqual(['e2', 'e3'])
  })

  // Quem já tem peça aparece e não se escolhe: trazer de novo não faria nada, e
  // esconder a linha faria o mestre procurar um nome que ele acabou de ver.
  it('quem já está no mapa fica travado na lista', async () => {
    const { user } = renderRegion(true)

    await user.click(abrirATrazer())

    // Escopado no diálogo: a peça DELA continua no tabuleiro atrás, com o mesmo
    // nome no começo do rótulo.
    const lista = within(screen.getByRole('dialog'))
    expect(lista.getByRole('button', { name: /Sílfide Ladina/ })).toBeDisabled()
  })
})

/**
 * A CORTINA (ALE-202): o tabuleiro existe para o mestre e a mesa vê uma cortina
 * no lugar dele — montar a taverna enquanto eles olham a cripta.
 *
 * O que se prova aqui é a composição, que é onde o defeito moraria: a redação em
 * si é do servidor e já está provada no Go (`TestACortinaEscondeACenaInteiraDaMesa`).
 * A tela não decide o que esconder; ela desenha o que recebeu, e o risco é ela
 * desenhar a coisa errada — ou desenhar a cortina para quem não devia.
 */
describe('a cortina sobre a cena', () => {
  /** O que o servidor manda à mesa com a cortina fechada: cena NENHUMA, e a
   *  lista de peças VAZIA (nunca nula — o cabeçalho indexa `tokens.length`). */
  const SOB_CORTINA: BoardState = { version: 4, place: '', terrain: '', tokens: [], curtained: true }

  it('a mesa vê a cortina, e não a cena', () => {
    renderRegion(false, SOB_CORTINA)

    expect(screen.getByText('O mestre está montando a cena.')).toBeInTheDocument()
    // A grade é o que denunciaria a cena: tamanho, terreno, onde tem peça.
    expect(screen.queryByRole('grid')).not.toBeInTheDocument()
    // "O mestre ainda não abriu um tabuleiro" é OUTRO estado, e o jogador o
    // resolve de outro jeito — este é esperar, aquele é cutucar o mestre.
    expect(screen.queryByText(/ainda não abriu/)).not.toBeInTheDocument()
  })

  // O CONTROLE do teste de cima: sem ele, um `BoardRegion` que escondesse a
  // grade sempre passaria nas asserções de ausência e a mesa nunca mais veria
  // tabuleiro nenhum. Ausência só é evidência quando se prova que o canal
  // estaria lá.
  it('sem cortina, a mesma mesa vê a grade', () => {
    renderRegion(false, TABULEIRO)

    expect(screen.getByRole('grid')).toBeInTheDocument()
    expect(screen.queryByText('O mestre está montando a cena.')).not.toBeInTheDocument()
  })

  it('o mestre continua vendo a cena que está montando, com a tira avisando', async () => {
    const CENA_FECHADA: BoardState = { ...TABULEIRO, curtained: true }
    const { rt, user } = renderRegion(true, CENA_FECHADA)

    // O ponto todo da cortina: o mestre trabalha no mapa inteiro.
    expect(screen.getByRole('grid')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /^Ogro,/ })).toBeInTheDocument()

    // A tira existe porque o esquecimento é o modo de falha caro: o mestre
    // narra a taverna para uma mesa que está olhando um aviso.
    const tira = screen.getByText('A cortina está fechada: a mesa não vê esta cena.')
    expect(tira).toBeInTheDocument()

    await user.click(screen.getByRole('button', { name: 'Abrir a cortina' }))
    expect(rt.setCurtain).toHaveBeenCalledWith(false)
  })

  it('o mestre fecha a cortina pelo cabeçalho', async () => {
    const { rt, user } = renderRegion(true, TABULEIRO)

    const botao = screen.getByRole('button', { name: 'Fechar a cortina' })
    expect(botao).toHaveAttribute('aria-pressed', 'false')

    await user.click(botao)

    expect(rt.setCurtain).toHaveBeenCalledWith(true)
  })

  // A lente da ALE-193 mostra a cortina SEM uma linha própria: ela pergunta ao
  // servidor a cópia da mesa, e a cópia vem com a cortina. Uma segunda regra
  // escrita na tela poderia divergir do servidor — que é a classe de defeito
  // que a lente existe para não ter.
  it('a lente do mestre mostra a cortina que a mesa está vendo', async () => {
    const CENA_FECHADA: BoardState = { ...TABULEIRO, curtained: true }
    const { user } = renderRegion(true, CENA_FECHADA)

    await user.click(screen.getByRole('button', { name: 'Ver como jogador' }))

    await waitFor(() =>
      expect(screen.getByText('O mestre está montando a cena.')).toBeInTheDocument(),
    )
    // A tira da cortina sobrevive à lente: é aí que o mestre mais precisa saber
    // que a cortina é DELE, senão a lente parece quebrada.
    expect(screen.getByText('A cortina está fechada: a mesa não vê esta cena.')).toBeInTheDocument()
  })

  // O jogador não tem como abrir a própria cortina, e a trava de verdade é do
  // SERVIDOR (provado no Go, `gmGate`). Aqui se afirma só que a tela não oferece
  // o gesto — é UX, e não a fronteira de segurança.
  it('a mesa não ganha botão de cortina', () => {
    renderRegion(false, SOB_CORTINA)

    expect(screen.queryByRole('button', { name: 'Fechar a cortina' })).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Abrir a cortina' })).not.toBeInTheDocument()
  })
})

/**
 * PONTOS DE PARADA (ALE-266).
 *
 * O defeito: o caminho que ia no fio era sempre uma reta monótona que o cliente
 * desenhava sozinho, então o jogador nunca escolhia a rota. Não doía enquanto
 * todo caminho entre A e B custasse o mesmo — com a diagonal valendo o dobro
 * (T20 p238) isso é verdade —, e passou a doer quando o terreno difícil chegou:
 * contornar duas casas de lama custa diferente de atravessá-las, e o jogador
 * pode QUERER a rota mais cara para não passar ao lado de um inimigo.
 *
 * O que se prova aqui é o desfecho: cada solta acrescenta uma perna, a perna sai
 * da última parada, e desfazer volta uma parada de cada vez. O CUSTO não é
 * afirmado — ele vem do servidor, e refazer a aritmética da diagonal e do brejo
 * na tela seria a segunda implementação da regra que a ALE-104 apagou.
 */
describe('o caminho é feito de paradas, e não de uma reta', () => {
  const PARADO_EM_8: BoardState = {
    ...COM_JOGADOR,
    pending: {
      tokenId: 't2',
      path: [
        { x: 6, y: 5 },
        { x: 7, y: 5 },
        { x: 8, y: 5 },
      ],
      cost: 2,
      budget: 6,
      diagonals: 0,
      difficult: 0,
      byUserId: 42,
    },
  }

  it('a segunda parada EMENDA na primeira, em vez de recomeçar da peça', async () => {
    const { rt, user } = renderRegion(false, PARADO_EM_8, 'e1', {
      myCharacterIds: MEU_HEROI,
      turnIndex: 0,
    })

    await user.click(screen.getByRole('button', { name: 'Sílfide Ladina, coluna 6, linha 5' }))
    await user.click(screen.getByRole('button', { name: 'Coluna 8, linha 4' }))

    // Sai da ÚLTIMA parada (8,5) e não do lugar da peça (6,5) — se recomeçasse
    // da peça, o caminho teria três quadrados e o desvio do jogador sumiria.
    expect(rt.proposeMove).toHaveBeenCalledWith('t2', [
      { x: 6, y: 5 },
      { x: 7, y: 5 },
      { x: 8, y: 5 },
      { x: 8, y: 4 },
    ])
  })

  it('desfazer volta UMA parada, e a última vira cancelar', async () => {
    // A cena muda no meio porque o servidor é quem devolve a proposta: o fake
    // não a inventa, e sem ela a faixa do movimento não monta. É a mesma
    // mutação que o broadcast traria, com a versão subindo.
    const [live, setLive] = createSignal<BoardState | null>(COM_JOGADOR)
    const { rt, user } = renderRegion(false, live, 'e1', {
      myCharacterIds: MEU_HEROI,
      turnIndex: 0,
    })

    await user.click(screen.getByRole('button', { name: 'Sílfide Ladina, coluna 6, linha 5' }))
    await user.click(screen.getByRole('button', { name: 'Coluna 8, linha 5' }))
    expect(rt.proposeMove).toHaveBeenCalledTimes(1)
    setLive({ ...PARADO_EM_8, version: COM_JOGADOR.version + 1 })

    // Com uma parada só, desfazê-la não deixa proposta nenhuma de pé — e
    // proposta sem perna não é proposta.
    await user.click(await screen.findByRole('button', { name: 'Desfazer parada' }))
    expect(rt.cancelMove).toHaveBeenCalled()
    expect(rt.proposeMove).toHaveBeenCalledTimes(1)
  })

  /**
   * A metade que o dono fez questão de separar: sem ver o que sobrou, o jogador
   * empilha paradas, passa do deslocamento e não sabe o que desfazer.
   *
   * O número sai da conta do SERVIDOR (`pending.cost` contra `speedSquares`),
   * nunca de uma refeita aqui.
   */
  it('a faixa diz quanto AINDA cabe, e não só o que já se gastou', async () => {
    const { user } = renderRegion(false, PARADO_EM_8, 'e1', {
      myCharacterIds: MEU_HEROI,
      turnIndex: 0,
    })

    await user.click(screen.getByRole('button', { name: 'Sílfide Ladina, coluna 6, linha 5' }))

    // 6 de deslocamento menos os 2 que o servidor cobrou pelas duas primeiras
    // casas.
    expect(await screen.findByText(/resta 4 quadrados/)).toBeInTheDocument()
  })
})
