import { QueryClient, QueryClientProvider } from '@tanstack/solid-query'
import { render, screen } from '@solidjs/testing-library'
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
 * A cena do jogador com o SERVIDOR FORA DO AR (ALE-143).
 *
 * O defeito que este teste congela matava a aba: a decisão "existe ficha para
 * abrir?" olhava o estado de carga da lista de membros, e o ramo de baixo
 * montava quem consulta a MESMA lista. Com a lista falhando, montar refazia a
 * consulta, a consulta voltava a "carregando", os ramos se alternavam — e o
 * laço só terminava quando a memória acabava. Reduzido a doze linhas antes do
 * conserto; aqui ele fica preso na cena de verdade.
 *
 * Este teste não afirma um texto: ele afirma que a cena TERMINA. Antes do
 * conserto, o worker do vitest morria com "JavaScript heap out of memory".
 */
vi.mock('@/shared/api/api', async () => {
  const real = await vi.importActual<Record<string, unknown>>('@/shared/api/api')
  return {
    ...real,
    api: {
      ...(real.api as Record<string, unknown>),
      members: { list: () => Promise.reject(new Error('servidor fora do ar')) },
      characters: { get: () => Promise.reject(new Error('servidor fora do ar')) },
    },
  }
})

const SESSAO = { id: 4, campaignId: 1, sessionNumber: 4, status: 'active' } as unknown as Session

const RT = {
  state: () => ({ initiative: [], round: 2, turnIndex: -1 }),
  isConnected: () => true,
  error: () => null,
  hasPersistenceWarning: () => false,
  present: () => [],
  restFlash: () => null,
  board: () => null,
  // O acervo de Lugares é do mestre e chega por PERGUNTA, não pelo snapshot
  // (ALE-124, fatia 5): o fake responde vazio.
  listPlaces: () => Promise.resolve([]),
} as unknown as SessionRealtime

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

describe('a cena do jogador com o servidor fora do ar', () => {
  it('assenta em vez de entrar em laço', async () => {
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    const root = createRootRoute({
      component: () => (
        <QueryClientProvider client={client}>
          <SessionPlayerView
            campaignId={1}
            session={SESSAO}
            rt={RT}
            myCharacterIds={new Set<number>()}
          />
        </QueryClientProvider>
      ),
    })
    const router = createRouter({ routeTree: root, history: createMemoryHistory() })
    render(() => <RouterProvider router={router} />)

    // Tempo de sobra para o laço nascer, se ele existisse.
    await new Promise((resolve) => setTimeout(resolve, 500))

    // A cena continua de pé e navegável: é isso que o jogador precisa quando o
    // servidor cai no meio da sessão.
    expect(await screen.findByRole('button', { name: /Mesa/ })).toBeInTheDocument()
    expect(screen.getByText(/Você não tem personagem nesta mesa/)).toBeInTheDocument()
  })
})
