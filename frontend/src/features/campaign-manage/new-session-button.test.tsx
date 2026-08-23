import { render, screen } from '@solidjs/testing-library'
import userEvent from '@testing-library/user-event'
import { afterEach, describe, expect, it, vi } from 'vitest'
import type { Session } from '@/shared/api/api'
import { NewSessionAction, nextSessionNumber } from './new-session-button'

afterEach(() => {
  vi.clearAllMocks()
  document.body.innerHTML = ''
})

function session(sessionNumber: number): Session {
  return {
    id: sessionNumber * 10,
    campaignId: 1,
    title: null,
    sessionNumber,
    notes: null,
    status: 'ended',
    startedAt: null,
    endedAt: null,
    createdAt: '2026-01-01T00:00:00.000Z',
    updatedAt: '2026-01-01T00:00:00.000Z',
  }
}

describe('nextSessionNumber', () => {
  it('a primeira campanha abre na sessão 1', () => {
    expect(nextSessionNumber([])).toBe(1)
  })

  // O próximo número vem do MAIOR, não da contagem: apagar a sessão 2 de
  // [1,2,3] deixaria a contagem em 2 e recriaria um número já usado.
  it('segue o maior número já usado, não a contagem', () => {
    expect(nextSessionNumber([session(1), session(3)])).toBe(4)
  })

  it('não se confunde com a ordem de chegada', () => {
    expect(nextSessionNumber([session(3), session(1), session(2)])).toBe(4)
  })
})

describe('NewSessionAction', () => {
  it('nomeia a próxima sessão quando não recebe rótulo', () => {
    render(() => <NewSessionAction nextNumber={4} onCreate={vi.fn()} />)
    expect(screen.getByRole('button', { name: /Sessão 4/ })).toBeInTheDocument()
  })

  it('aceita um rótulo próprio para o estado vazio', () => {
    render(() => <NewSessionAction nextNumber={1} label="Abrir a primeira" onCreate={vi.fn()} />)
    expect(screen.getByRole('button', { name: /Abrir a primeira/ })).toBeInTheDocument()
  })

  it('cria a sessão no clique', async () => {
    const onCreate = vi.fn().mockResolvedValue(undefined)
    render(() => <NewSessionAction nextNumber={2} onCreate={onCreate} />)

    await userEvent.setup().click(screen.getByRole('button', { name: /Sessão 2/ }))

    expect(onCreate).toHaveBeenCalledOnce()
  })

  // Sem trava, dois cliques rápidos abrem duas sessões com o mesmo número.
  it('trava o botão enquanto cria', async () => {
    let release = () => {}
    const onCreate = vi.fn(() => new Promise<void>((resolve) => (release = resolve)))
    render(() => <NewSessionAction nextNumber={2} onCreate={onCreate} />)
    const button = screen.getByRole('button', { name: /Sessão 2/ })

    await userEvent.setup().click(button)

    expect(screen.getByRole('button', { name: /Criando/ })).toBeDisabled()
    release()
  })
})
