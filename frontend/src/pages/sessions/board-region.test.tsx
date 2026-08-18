import { render, screen, within } from '@solidjs/testing-library'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi } from 'vitest'
import { createBoardViewport } from '@/features/battle-board/board-viewport'
import type { BoardState, SessionRealtime } from '@/shared/realtime/realtime'
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
  readonly proposeMove = vi.fn()
  readonly commitMove = vi.fn()
  readonly cancelMove = vi.fn()

  constructor(
    private readonly live: BoardState | null,
    private readonly turnIndex = -1,
  ) {}

  asRealtime(): SessionRealtime {
    return {
      state: () => ({
        initiative: [{ id: 'e1', label: 'Sílfide Ladina', initiative: 18, type: 'character' }],
        round: 1,
        turnIndex: this.turnIndex,
      }),
      isConnected: () => true,
      error: () => null,
      board: () => this.live,
      updateToken: this.updateToken,
      removeToken: this.removeToken,
      addToken: this.addToken,
      closeBoard: this.closeBoard,
      populateBoard: this.populateBoard,
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

function renderRegion(
  isGm: boolean,
  live: BoardState | null = TABULEIRO,
  activeEntryId?: string,
  jogador: { myCharacterIds?: ReadonlySet<number>; turnIndex?: number } = {},
) {
  const rt = new FakeRealtime(live, jogador.turnIndex ?? -1)
  // A janela nasce fora da região, como na página: ela precisa sobreviver à
  // troca de região, e o teste monta a mesma composição que a cena monta.
  const view = createBoardViewport()
  // A vista começa mostrando a origem menos um pedaço, para o teste alcançar
  // tanto o quadrado (3,2) quanto o negativo (−4,−3) sem depender de medição —
  // em jsdom todo elemento mede zero e o ResizeObserver nem existe.
  view.centerOn(0, 0)
  render(() => (
    <BoardRegion
      rt={rt.asRealtime()}
      isGm={isGm}
      view={view}
      activeEntryId={activeEntryId}
      myCharacterIds={jogador.myCharacterIds}
    />
  ))
  return { rt, user: userEvent.setup(), view }
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
    const { user, view } = renderRegion(true, longe)

    expect(screen.queryByRole('button', { name: /Sentinela Distante/ })).not.toBeInTheDocument()

    const passos = Math.ceil(60 / Math.max(1, Math.floor(view.cols() / 3)))
    for (let i = 0; i < passos; i++) {
      await user.click(screen.getByRole('button', { name: 'Mover a vista para a direita' }))
    }

    expect(screen.getByRole('button', { name: /Sentinela Distante/ })).toBeInTheDocument()
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

    await user.click(screen.getByRole('button', { name: 'Porta, coluna 1, linha 1' }))
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
})
