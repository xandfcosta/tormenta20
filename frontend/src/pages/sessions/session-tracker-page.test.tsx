import { render, screen } from '@solidjs/testing-library'
import { QueryClient, QueryClientProvider } from '@tanstack/solid-query'
import {
  createMemoryHistory,
  createRootRoute,
  createRoute,
  createRouter,
  RouterProvider,
} from '@tanstack/solid-router'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { campaignMembersQueryOptions, campaignQueryOptions } from '@/entities/campaign/queries'
import { campaignSessionQueryOptions } from '@/entities/session/queries'
import { meQueryOptions } from '@/entities/user/queries'
import type { AuthUser, Campaign, Session } from '@/shared/api/api'
import type { SessionRealtime } from '@/shared/realtime/realtime'
import { UiProvider } from '@/shared/stores/ui-context'
import { createUiStore } from '@/shared/stores/ui-store'
import { FakeStorage } from '@/shared/test/fake-storage'
import { SessionTrackerPage } from './session-tracker-page'

/**
 * QUEM VÊ O QUÊ (ALE-186, bloco 4).
 *
 * Esta página escolhe entre a cena do MESTRE e a do JOGADOR, e nunca tinha sido
 * montada: as duas views têm testes próprios, a ESCOLHA entre elas não tinha
 * nenhum. Um `isGm` lido da query errada — ou lido antes de a campanha
 * assentar — passava a suíte inteira e entregava a mesa ao jogador.
 *
 * É a composição no sentido em que o CLAUDE.md usa a palavra: as peças estão
 * provadas, o que faltava era provar a montagem.
 */

/** O socket é criado DENTRO da página (ela é a dona da conexão da partida), então
 *  a única costura possível é o módulo. */
vi.mock('@/shared/realtime/realtime', async () => {
  const real = await vi.importActual<Record<string, unknown>>('@/shared/realtime/realtime')
  return {
    ...real,
    createSessionSocket: (): SessionRealtime =>
      ({
        state: () => ({ initiative: [], round: 1, turnIndex: -1 }),
        isConnected: () => true,
        error: () => null,
        hasPersistenceWarning: () => false,
        present: () => [],
        restFlash: () => null,
        board: () => null,
        listPlaces: () => Promise.resolve([]),
      }) as unknown as SessionRealtime,
  }
})

const SESSAO = {
  id: 5,
  campaignId: 1,
  sessionNumber: 3,
  status: 'active',
} as unknown as Session

function mesaCom(role: 'gm' | 'player'): Campaign {
  return { id: 1, name: 'Snapshot', ownerId: 1, role } as unknown as Campaign
}

/** Monta a página na rota real. `campaign` ausente = a campanha ainda voando. */
function renderTracker(campaign: Campaign | undefined) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  client.setQueryData(campaignSessionQueryOptions(1, 5).queryKey, SESSAO)
  client.setQueryData(campaignMembersQueryOptions(1).queryKey, [])
  const eu: AuthUser = { id: 9, email: 'eu@t20.local', name: null, isAdmin: false }
  client.setQueryData(meQueryOptions.queryKey, eu)
  if (campaign) client.setQueryData(campaignQueryOptions(1).queryKey, campaign)

  const root = createRootRoute()
  const route = createRoute({
    getParentRoute: () => root,
    path: '/campaigns/$id/sessions/$sid',
    component: SessionTrackerPage,
  })
  const router = createRouter({
    routeTree: root.addChildren([route]),
    history: createMemoryHistory({ initialEntries: ['/campaigns/1/sessions/5'] }),
  })
  return render(() => (
    <UiProvider store={createUiStore(new FakeStorage())}>
      <QueryClientProvider client={client}>
        {/* biome-ignore lint/suspicious/noExplicitAny: o router de teste tem uma rota só */}
        <RouterProvider router={router as any} />
      </QueryClientProvider>
    </UiProvider>
  ))
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

describe('SessionTrackerPage — o papel escolhe a cena', () => {
  it('o mestre recebe a cena do mestre', async () => {
    renderTracker(mesaCom('gm'))

    expect(await screen.findByRole('button', { name: 'Configurações da sessão' })).toBeVisible()
    // E não a do jogador: as duas na mesma tela seria pior que a errada.
    expect(screen.queryByRole('group', { name: 'O que ver na sessão' })).not.toBeInTheDocument()
  })

  it('o jogador recebe a cena do jogador', async () => {
    renderTracker(mesaCom('player'))

    expect(await screen.findByRole('group', { name: 'O que ver na sessão' })).toBeVisible()
    // O botão de configurações é do dono da mesa e não pode aparecer aqui.
    expect(
      screen.queryByRole('button', { name: 'Configurações da sessão' }),
    ).not.toBeInTheDocument()
  })

  it('enquanto a campanha não assenta, ninguém recebe cena nenhuma', async () => {
    renderTracker(undefined)

    // O papel só é conhecido quando a CAMPANHA chega, e `role` ausente lê como
    // "não é mestre": sem esta espera, o mestre veria a cena do jogador piscar
    // antes de trocar. O esqueleto é o estado certo desse instante.
    expect(await screen.findByRole('status', { name: 'Carregando a sessão' })).toBeVisible()
    expect(
      screen.queryByRole('button', { name: 'Configurações da sessão' }),
    ).not.toBeInTheDocument()
    expect(screen.queryByRole('group', { name: 'O que ver na sessão' })).not.toBeInTheDocument()
  })
})
