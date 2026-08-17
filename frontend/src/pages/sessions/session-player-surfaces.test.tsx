import { QueryClient, QueryClientProvider } from '@tanstack/solid-query'
import { render, screen } from '@solidjs/testing-library'
import userEvent from '@testing-library/user-event'
import {
  RouterProvider,
  createMemoryHistory,
  createRootRoute,
  createRouter,
} from '@tanstack/solid-router'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import type { Session } from '@/shared/api/api'
import type { SessionRealtime } from '@/shared/realtime/realtime'
import { SessionPlayerView } from './session-player-view'

/**
 * As três superfícies do jogador (ALE-129).
 *
 * A cena era a ficha ocupando a tela com a sessão espremida num rail de 22rem —
 * e ali o tabuleiro cabia em 8×4 quadrados. Agora ficha, mesa e tabuleiro são
 * superfícies irmãs, cada uma ocupando a tela, com o seletor ancorado no topo.
 *
 * O que se prova aqui é a TROCA: escolher uma superfície mostra o conteúdo dela
 * e esconde o das outras. Que ela CABE na tela é medida do browser, e mora no
 * e2e — em jsdom tudo mede zero.
 */
class FakeRealtime {
  asRealtime(): SessionRealtime {
    return {
      state: () => ({ initiative: [], round: 2, turnIndex: -1 }),
      isConnected: () => true,
      error: () => null,
      hasPersistenceWarning: () => false,
      present: () => [],
      restFlash: () => null,
      board: () => null,
      rollSelfInitiative: vi.fn(),
      openBoard: vi.fn(),
    } as unknown as SessionRealtime
  }
}

const SESSAO = { id: 4, campaignId: 1, sessionNumber: 4, status: 'active' } as unknown as Session

function renderCena() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  const root = createRootRoute({
    component: () => (
      <QueryClientProvider client={client}>
        <SessionPlayerView
          campaignId={1}
          session={SESSAO}
          rt={new FakeRealtime().asRealtime()}
          myCharacterIds={new Set<number>()}
        />
      </QueryClientProvider>
    ),
  })
  const router = createRouter({ routeTree: root, history: createMemoryHistory() })
  render(() => <RouterProvider router={router} />)
  return userEvent.setup()
}

beforeEach(() => {
  window.matchMedia = vi.fn().mockImplementation((media: string) => ({
    matches: true,
    media,
    onchange: null,
    addEventListener: vi.fn(),
    removeEventListener: vi.fn(),
    dispatchEvent: vi.fn(),
  })) as unknown as typeof window.matchMedia
})

describe('as superfícies da cena do jogador', () => {
  // O elenco vazio (o fake não tem membro nenhum) cai no ramo "sem ficha", que
  // mostra a MESA ocupando a tela. O seletor continua sendo a forma de trocar.
  it('o seletor fica na tela, com as três', async () => {
    renderCena()

    expect(await screen.findByRole('button', { name: /Minha ficha/ })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /Mesa/ })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /Tabuleiro/ })).toBeInTheDocument()
  })

  // A mesa é uma superfície inteira, não um rail: a iniciativa cabe nela.
  it('escolher a mesa mostra a iniciativa ocupando a cena', async () => {
    const user = renderCena()

    await user.click(await screen.findByRole('button', { name: /Mesa/ }))

    expect(screen.getByRole('heading', { name: 'Iniciativa' })).toBeInTheDocument()
    expect(screen.queryByText(/O mestre ainda não abriu um tabuleiro/)).not.toBeInTheDocument()
  })

  it('escolher o tabuleiro troca a superfície inteira', async () => {
    const user = renderCena()

    await user.click(await screen.findByRole('button', { name: /Tabuleiro/ }))

    expect(screen.getByText(/O mestre ainda não abriu um tabuleiro/)).toBeInTheDocument()
    // A mesa saiu: superfícies são irmãs, não empilhadas.
    expect(screen.queryByRole('heading', { name: 'Iniciativa' })).not.toBeInTheDocument()
  })

  // A escolha atual precisa ser legível sem cor: `aria-pressed` é o que o leitor
  // de tela lê e o que o teste afirma.
  it('a superfície escolhida se anuncia', async () => {
    const user = renderCena()

    await user.click(await screen.findByRole('button', { name: /Mesa/ }))

    expect(screen.getByRole('button', { name: /Mesa/ })).toHaveAttribute('aria-pressed', 'true')
    expect(screen.getByRole('button', { name: /Minha ficha/ })).toHaveAttribute(
      'aria-pressed',
      'false',
    )
  })
})
