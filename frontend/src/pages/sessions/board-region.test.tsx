import { render, screen } from '@solidjs/testing-library'
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

  constructor(private readonly live: BoardState | null) {}

  asRealtime(): SessionRealtime {
    return {
      state: () => ({ initiative: [], round: 1, turnIndex: -1 }),
      isConnected: () => true,
      error: () => null,
      board: () => this.live,
      updateToken: this.updateToken,
      closeBoard: this.closeBoard,
      populateBoard: this.populateBoard,
      openBoard: this.openBoard,
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

function renderRegion(isGm: boolean, live: BoardState | null = TABULEIRO, activeEntryId?: string) {
  const rt = new FakeRealtime(live)
  // A janela nasce fora da região, como na página: ela precisa sobreviver à
  // troca de região, e o teste monta a mesma composição que a cena monta.
  const view = createBoardViewport()
  // A vista começa mostrando a origem menos um pedaço, para o teste alcançar
  // tanto o quadrado (3,2) quanto o negativo (−4,−3) sem depender de medição —
  // em jsdom todo elemento mede zero e o ResizeObserver nem existe.
  view.centerOn(0, 0)
  render(() => (
    <BoardRegion rt={rt.asRealtime()} isGm={isGm} view={view} activeEntryId={activeEntryId} />
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
    await user.click(ogro)
    await user.click(screen.getByRole('button', { name: 'Coluna 5, linha 3' }))

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
