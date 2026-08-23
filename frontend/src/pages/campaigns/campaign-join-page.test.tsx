import { render, screen, waitFor } from '@solidjs/testing-library'
import { QueryClient, QueryClientProvider } from '@tanstack/solid-query'
import {
  createMemoryHistory,
  createRootRoute,
  createRoute,
  createRouter,
  RouterProvider,
} from '@tanstack/solid-router'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { z } from 'zod'
import { inviteQueryOptions } from '@/entities/campaign/queries'
import { makeCharacter } from '@/entities/character/__fixtures__/character'
import { charactersQueryOptions } from '@/entities/character/queries'
import { ApiError, api, type CampaignMember } from '@/shared/api/api'
import { UiProvider } from '@/shared/stores/ui-context'
import { createUiStore } from '@/shared/stores/ui-store'
import { FakeStorage } from '@/shared/test/fake-storage'
import { CampaignJoinPage } from './campaign-join-page'

/**
 * ENTRAR NA MESA (ALE-197, grupo A).
 *
 * A cena tem duas portas — o link de convite e o número que o mestre leu em voz
 * alta — e o alvo é DERIVADO delas (`joinTargetId`, que tem teste próprio). O
 * que nunca foi montado é a composição: qual porta aparece, o que o botão
 * espera para destravar, e qual campanha recebe o `add` no fim. A versão React
 * espelhava o id do convite em estado de formulário e o espelho podia
 * discordar do convite (ALE-20); o teste que faltava é justamente o que
 * flagraria esse desacordo.
 */

const HEROI = makeCharacter({ id: 12, name: 'Arwen' })

function renderJoin(url: string, options: { invite?: { campaignId: number; campaignName: string } } = {}) {
  // `staleTime: Infinity` não é conforto de teste: sem ele o dado semeado do
  // convite é refetchado na montagem, o fetch real falha no jsdom e o
  // `inviteInvalid()` trava o botão — a cena mediria a rede, não a regra.
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false, staleTime: Number.POSITIVE_INFINITY } },
  })
  client.setQueryData(charactersQueryOptions.queryKey, [HEROI])
  if (options.invite) {
    client.setQueryData(inviteQueryOptions('abc').queryKey, options.invite)
  }

  const root = createRootRoute()
  const join = createRoute({
    getParentRoute: () => root,
    path: '/campaigns/join',
    validateSearch: z.object({ token: z.string().optional() }),
    component: CampaignJoinPage,
  })
  const cronica = createRoute({
    getParentRoute: () => root,
    path: '/campaigns/$id',
    component: () => <p>A campanha</p>,
  })
  const router = createRouter({
    routeTree: root.addChildren([join, cronica]),
    history: createMemoryHistory({ initialEntries: [url] }),
  })
  render(() => (
    <UiProvider store={createUiStore(new FakeStorage())}>
      <QueryClientProvider client={client}>
        {/* biome-ignore lint/suspicious/noExplicitAny: router de teste com duas rotas */}
        <RouterProvider router={router as any} />
      </QueryClientProvider>
    </UiProvider>
  ))
  return userEvent.setup()
}

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

afterEach(() => {
  vi.restoreAllMocks()
  document.body.innerHTML = ''
})

describe('CampaignJoinPage', () => {
  it('com convite, a carta nomeia a mesa e o número não é pedido', async () => {
    renderJoin('/campaigns/join?token=abc', {
      invite: { campaignId: 7, campaignName: 'A Tumba de Sszzaas' },
    })

    expect(await screen.findByText('A Tumba de Sszzaas')).toBeInTheDocument()
    // As duas portas são exclusivas: pedir o número a quem chegou por convite
    // seria oferecer um jeito de contradizer o próprio link.
    expect(screen.queryByLabelText('Número da campanha')).not.toBeInTheDocument()
  })

  it('entra na campanha DO CONVITE, levando o token junto', async () => {
    const add = vi.spyOn(api.members, 'add').mockResolvedValue({ id: 1 } as CampaignMember)
    const user = renderJoin('/campaigns/join?token=abc', {
      invite: { campaignId: 7, campaignName: 'A Tumba de Sszzaas' },
    })

    await user.click(await screen.findByRole('button', { name: /Arwen/ }))
    await user.click(screen.getByRole('button', { name: 'Entrar na mesa' }))

    // O id vem do CONVITE, não de nada que a tela guardou; e o token viaja
    // junto porque é ele que autoriza a entrada numa mesa que não é pública.
    await waitFor(() =>
      expect(add).toHaveBeenCalledWith(7, {
        characterId: 12,
        role: 'player',
        inviteToken: 'abc',
      }),
    )
    expect(await screen.findByText('A campanha')).toBeInTheDocument()
  })

  it('sem herói escolhido, não dá para entrar', async () => {
    renderJoin('/campaigns/join?token=abc', {
      invite: { campaignId: 7, campaignName: 'A Tumba de Sszzaas' },
    })

    // O botão é a única barreira: entrar sem personagem criaria um membro sem
    // ficha, e a mesa mostraria uma linha vazia na iniciativa.
    expect(await screen.findByRole('button', { name: 'Entrar na mesa' })).toBeDisabled()
  })

  it('pelo número digitado, entra na campanha digitada e SEM token', async () => {
    const add = vi.spyOn(api.members, 'add').mockResolvedValue({ id: 1 } as CampaignMember)
    const user = renderJoin('/campaigns/join')

    await user.type(await screen.findByLabelText('Número da campanha'), '3')
    await user.click(screen.getByRole('button', { name: /Arwen/ }))
    await user.click(screen.getByRole('button', { name: 'Entrar na mesa' }))

    await waitFor(() =>
      expect(add).toHaveBeenCalledWith(3, { characterId: 12, role: 'player' }),
    )
  })

  it('a recusa do servidor aparece na cena e ninguém sai do lugar', async () => {
    vi.spyOn(api.members, 'add').mockRejectedValue(
      new ApiError(409, 'Você já joga nesta mesa'),
    )
    const user = renderJoin('/campaigns/join')

    await user.type(await screen.findByLabelText('Número da campanha'), '3')
    await user.click(screen.getByRole('button', { name: /Arwen/ }))
    await user.click(screen.getByRole('button', { name: 'Entrar na mesa' }))

    // A mensagem é a DO SERVIDOR: "não foi possível" é o último recurso, e
    // trocar o motivo real por ele faria a pessoa tentar de novo à toa.
    expect(await screen.findByText('Você já joga nesta mesa')).toBeInTheDocument()
    expect(screen.queryByText('A campanha')).not.toBeInTheDocument()
  })
})
