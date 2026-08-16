import { QueryClient, QueryClientProvider } from '@tanstack/solid-query'
import {
  RouterProvider,
  createMemoryHistory,
  createRootRoute,
  createRoute,
  createRouter,
} from '@tanstack/solid-router'
import { render, screen } from '@solidjs/testing-library'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { z } from 'zod'
import { campaignMembersQueryOptions, campaignQueryOptions } from '@/entities/campaign/queries'
import { campaignSessionsQueryOptions } from '@/entities/session/queries'
import type { Campaign } from '@/shared/api/api'
import { UiProvider } from '@/shared/stores/ui-context'
import { createUiStore } from '@/shared/stores/ui-store'
import { FakeStorage } from '@/shared/test/fake-storage'
import { CampaignDetailPage } from './campaign-detail-page'

/**
 * A seção ativa mora no `?tab=` — é o que faz o deep link e o botão Voltar
 * funcionarem. Nada montava esta página com a URL, então a única prova de que a
 * URL escolhe a seção era um e2e; e a queda para "visão" quando o JOGADOR chega
 * com `?tab=config` (a seção que o rail dele não tem) não era testada em camada
 * nenhuma.
 */
function renderPage(role: 'gm' | 'player', search: string) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  const campaign = {
    id: 1,
    name: 'Snapshot',
    description: 'mesa de teste',
    ownerId: 1,
    role,
    createdAt: '2026-01-01T00:00:00.000Z',
    updatedAt: '2026-01-01T00:00:00.000Z',
  } as unknown as Campaign
  client.setQueryData(campaignQueryOptions(1).queryKey, campaign)
  client.setQueryData(campaignSessionsQueryOptions(1).queryKey, [])
  client.setQueryData(campaignMembersQueryOptions(1).queryKey, [])

  const root = createRootRoute()
  const route = createRoute({
    getParentRoute: () => root,
    path: '/campaigns/$id/',
    validateSearch: z.object({ tab: z.string().optional() }),
    component: CampaignDetailPage,
  })
  const router = createRouter({
    routeTree: root.addChildren([route]),
    history: createMemoryHistory({ initialEntries: [`/campaigns/1${search}`] }),
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

describe('CampaignDetailPage — a URL escolhe a seção', () => {
  // A cena consulta media queries; sem isto o jsdom derruba a página inteira no
  // ErrorBoundary e todo o teste vira "não achei o texto".
  beforeEach(() => {
    window.matchMedia = vi.fn().mockImplementation((media: string) => ({
      matches: false,
      media,
      onchange: null,
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
      addListener: vi.fn(),
      removeListener: vi.fn(),
      dispatchEvent: vi.fn(),
    }))
  })

  it('o mestre com ?tab=config cai na Config', async () => {
    renderPage('gm', '?tab=config')
    expect(await screen.findByText('Zona de perigo')).toBeInTheDocument()
  })

  // O jogador não tem Config no rail: chegar com a URL na mão não pode abrir
  // uma seção que ele não navega (a trava de verdade é do servidor; isto é UX).
  // Duas defesas independentes sustentam isto — a queda do `tab()` e o `Show`
  // do tomo — e o teste só fica vermelho quando as DUAS caem. Ele protege o
  // desfecho, não cada guarda: é de propósito.
  it('o jogador com ?tab=config cai na Visão', async () => {
    renderPage('player', '?tab=config')
    // Espera algo POSITIVO antes de afirmar a ausência: sobre a árvore ainda
    // vazia, "não achei a Zona de perigo" passaria sem a página ter montado.
    const rail = await screen.findAllByRole('tab')
    expect(rail.map((t) => t.textContent)).not.toContain('Config')
    expect(screen.queryByText('Zona de perigo')).not.toBeInTheDocument()
  })

  it('?tab= inventado cai na Visão em vez de quebrar', async () => {
    renderPage('gm', '?tab=lixo')
    const visao = await screen.findByRole('tab', { name: /Visão geral/i })
    expect(visao).toHaveAttribute('aria-selected', 'true')
    expect(screen.queryByText('Zona de perigo')).not.toBeInTheDocument()
  })
})
