import { QueryClient, QueryClientProvider } from '@tanstack/solid-query'
import { render, screen } from '@solidjs/testing-library'
import userEvent from '@testing-library/user-event'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import type { Session } from '@/shared/api/api'
import type { InitiativeEntry, SessionRealtime } from '@/shared/realtime/realtime'
import { SessionGmView } from './session-gm-view'

/**
 * A faixa do turno — o que o mestre mais clica na sessão.
 *
 * "Turno anterior" existe porque um "Próximo turno" a mais é o erro mais comum
 * da mesa, e o conserto até aqui era dar a volta na iniciativa inteira, o que
 * empurrava a RODADA junto (ALE-122).
 */
class FakeRealtime {
  readonly nextTurn = vi.fn()
  readonly previousTurn = vi.fn()
  readonly resetInitiative = vi.fn()

  asRealtime(): SessionRealtime {
    return {
      state: () => ({ initiative: [] as InitiativeEntry[], round: 3, turnIndex: -1 }),
      isConnected: () => true,
      // A cena do mestre passou a montar a região do tabuleiro (ALE-124): sem
      // este acessor o fake mente sobre a forma do contrato e a tela quebra.
      board: () => null,
      // O acervo de Lugares é do mestre e chega por PERGUNTA, não pelo snapshot
      // (ALE-124, fatia 5): o fake responde vazio.
      listPlaces: () => Promise.resolve([]),
      error: () => null,
      hasPersistenceWarning: () => false,
      present: () => [],
      nextTurn: this.nextTurn,
      previousTurn: this.previousTurn,
      resetInitiative: this.resetInitiative,
      addEntry: vi.fn(),
      updateEntry: vi.fn(),
      removeEntry: vi.fn(),
      populateParty: vi.fn(),
      deltaVitals: vi.fn(),
      applyEffect: vi.fn(),
      rest: vi.fn(),
    } as unknown as SessionRealtime
  }
}

const SESSION = { id: 4, campaignId: 1, sessionNumber: 4, title: null } as unknown as Session

function renderScene() {
  const rt = new FakeRealtime()
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  render(() => (
    <QueryClientProvider client={client}>
      <SessionGmView
        campaignId={1}
        sessionId={4}
        session={SESSION}
        rt={rt.asRealtime()}
        myCharacterIds={new Set<number>()}
      />
    </QueryClientProvider>
  ))
  return { rt, user: userEvent.setup() }
}

beforeEach(() => {
  window.matchMedia = vi.fn().mockImplementation((media: string) => ({
    matches: true,
    media,
    onchange: null,
    addEventListener: vi.fn(),
    removeEventListener: vi.fn(),
    addListener: vi.fn(),
    removeListener: vi.fn(),
    dispatchEvent: vi.fn(),
  }))
})

describe('faixa do turno', () => {
  it('desfazer um turno pede o turno anterior ao servidor', async () => {
    const { rt, user } = renderScene()

    await user.click(screen.getByRole('button', { name: 'Turno anterior' }))

    expect(rt.previousTurn).toHaveBeenCalledTimes(1)
    expect(rt.nextTurn).not.toHaveBeenCalled()
  })

  // Reiniciar apaga o combate inteiro e fica longe do avanço, atrás de
  // confirmação — a proteção estava no lugar errado antes (ALE-122). Na
  // ALE-184 ele saiu da FAIXA e foi para o menu da sessão: continua atrás de
  // confirmação, e agora também atrás de um menu, porque ocupava um lugar na
  // fileira mais disputada da cena para algo que se usa quase nunca.
  it('reiniciar saiu da faixa e continua pedindo confirmação', async () => {
    const { rt, user } = renderScene()

    expect(screen.queryByRole('button', { name: 'Reiniciar' })).not.toBeInTheDocument()

    await user.click(screen.getByRole('button', { name: 'Configurações da sessão' }))
    await user.click(await screen.findByRole('button', { name: 'Reiniciar' }))

    expect(rt.resetInitiative).not.toHaveBeenCalled()
    expect(await screen.findByRole('dialog')).toHaveTextContent('Reiniciar o combate?')
  })

  // O avanço passa a dizer PARA ONDE vai (ALE-184): com a lista vazia não há
  // destino, e prometer um nome seria inventá-lo.
  it('sem combatentes, o avanço não promete nome nenhum', () => {
    renderScene()

    expect(screen.getAllByRole('button', { name: 'Próximo turno' })[0]).toBeDisabled()
  })
})
