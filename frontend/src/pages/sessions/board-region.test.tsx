import { render, screen } from '@solidjs/testing-library'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi } from 'vitest'
import type { BoardState, SessionRealtime } from '@/shared/realtime/realtime'
import { BoardRegion } from './board-region'

/**
 * O tabuleiro na cena da sessão (ALE-124). O que se prova aqui é o que alguém na
 * mesa notaria: a peça aparece onde o servidor disse, o mestre move clicando, e
 * quem não é mestre não ganha controle nenhum.
 *
 * Posição é afirmada pelo NOME ACESSÍVEL ("coluna 4, linha 3") e não por pixel:
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
  cols: 10,
  rows: 8,
  terrain: 'taverna',
  tokens: [
    { id: 't1', label: 'Ogro', x: 3, y: 2, footprint: 2, kind: 'npc', entryId: 'e1' },
    { id: 't2', label: 'Sílfide Ladina', x: 6, y: 5, footprint: 1, kind: 'character' },
  ],
}

function renderRegion(isGm: boolean, live: BoardState | null = TABULEIRO, activeEntryId?: string) {
  const rt = new FakeRealtime(live)
  render(() => <BoardRegion rt={rt.asRealtime()} isGm={isGm} activeEntryId={activeEntryId} />)
  return { rt, user: userEvent.setup() }
}

describe('o tabuleiro na cena', () => {
  it('as peças aparecem no quadrado que o servidor mandou', () => {
    renderRegion(true)

    expect(screen.getByRole('button', { name: 'Ogro, coluna 4, linha 3' })).toBeInTheDocument()
    expect(
      screen.getByRole('button', { name: 'Sílfide Ladina, coluna 7, linha 6' }),
    ).toBeInTheDocument()
  })

  // Duas ações, não uma: selecionar e pousar. Arrastar entra na fatia do
  // movimento — e mesmo lá continuará havendo o caminho de dois cliques, porque
  // gesto nunca é o único caminho.
  it('o mestre seleciona a peça e pousa num quadrado', async () => {
    const { rt, user } = renderRegion(true)

    await user.click(screen.getByRole('button', { name: 'Ogro, coluna 4, linha 3' }))
    await user.click(screen.getByRole('button', { name: 'Coluna 6, linha 4' }))

    expect(rt.updateToken).toHaveBeenCalledWith('t1', { x: 5, y: 3 })
  })

  // Clicar de novo na mesma peça a LARGA: sem isso não há como desistir, e o
  // próximo clique num quadrado moveria a peça errada.
  it('clicar de novo na mesma peça desiste do movimento', async () => {
    const { rt, user } = renderRegion(true)
    const ogro = screen.getByRole('button', { name: 'Ogro, coluna 4, linha 3' })

    await user.click(ogro)
    await user.click(ogro)
    await user.click(screen.getByRole('button', { name: 'Coluna 6, linha 4' }))

    expect(rt.updateToken).not.toHaveBeenCalled()
  })

  // A vez é a mesma informação da iniciativa, no mesmo vocabulário: quem está na
  // vez tem o anel dourado, e o teste afirma o ESTADO, não a classe.
  it('o jogador não recebe controle nenhum sobre as peças', () => {
    renderRegion(false)

    const ogro = screen.getByRole('button', { name: 'Ogro, coluna 4, linha 3' })
    expect(ogro).toBeDisabled()
    expect(screen.queryByRole('button', { name: 'Coluna 6, linha 4' })).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Encerrar o tabuleiro' })).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: /Trazer a iniciativa/ })).not.toBeInTheDocument()
  })

  it('sem tabuleiro, só o mestre vê como abrir um', () => {
    renderRegion(false, null)
    expect(screen.queryByRole('button', { name: 'Abrir tabuleiro' })).not.toBeInTheDocument()
    expect(screen.getByText(/ainda não abriu um tabuleiro/)).toBeInTheDocument()
  })

  it('o mestre abre um tabuleiro com lugar e grade', async () => {
    const { rt, user } = renderRegion(true, null)

    await user.click(screen.getByRole('button', { name: 'Abrir tabuleiro' }))
    await user.type(screen.getByLabelText('Lugar'), 'Cripta')
    await user.click(screen.getByRole('button', { name: 'Abrir' }))

    expect(rt.openBoard).toHaveBeenCalledWith('Cripta', 20, 15, 'pedra')
  })
})
