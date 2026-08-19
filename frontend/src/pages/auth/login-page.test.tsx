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
import { afterEach, describe, expect, it, vi } from 'vitest'
import { z } from 'zod'
import { meQueryOptions } from '@/entities/user/queries'
import { type AuthUser, api } from '@/shared/api/api'
import { LoginPage } from './login-page'

/**
 * A VOLTA depois de entrar (ALE-197, grupo A).
 *
 * O `requireSession` manda o anônimo para `/login?redirect=<onde ele ia>`, e o
 * `-guards.test.ts` prova essa ESCRITA. Ninguém provava a LEITURA: quem clica o
 * link da mesa, entra e cai na home perdeu o caminho, e o guarda continuaria
 * verde — as duas metades vivem em arquivos diferentes e nenhuma vê a outra.
 *
 * O destino é afirmado pela tela que aparece, não por um espião na navegação.
 */

const EU: AuthUser = { id: 9, email: 'eu@t20.local', name: null, isAdmin: false }

function renderLogin(url: string) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  const root = createRootRoute()
  const login = createRoute({
    getParentRoute: () => root,
    path: '/login',
    validateSearch: z.object({ redirect: z.string().optional() }),
    component: LoginPage,
  })
  const sessao = createRoute({
    getParentRoute: () => root,
    path: '/campaigns/$id/sessions/$sid',
    component: () => <p>A cena da sessão</p>,
  })
  const home = createRoute({
    getParentRoute: () => root,
    path: '/',
    component: () => <p>O saguão</p>,
  })
  const router = createRouter({
    routeTree: root.addChildren([login, sessao, home]),
    history: createMemoryHistory({ initialEntries: [url] }),
  })
  render(() => (
    <QueryClientProvider client={client}>
      {/* biome-ignore lint/suspicious/noExplicitAny: router de teste com três rotas */}
      <RouterProvider router={router as any} />
    </QueryClientProvider>
  ))
  return { user: userEvent.setup(), client }
}

async function entrar(user: ReturnType<typeof userEvent.setup>) {
  await user.type(await screen.findByLabelText('E-mail'), 'eu@t20.local')
  await user.type(screen.getByLabelText('Senha'), 'segredo123')
  await user.click(screen.getByRole('button', { name: 'Entrar' }))
}

afterEach(() => {
  vi.restoreAllMocks()
  document.body.innerHTML = ''
})

describe('LoginPage', () => {
  it('entra e volta para onde a pessoa ia', async () => {
    vi.spyOn(api.auth, 'login').mockResolvedValue(EU)
    const { user } = renderLogin('/login?redirect=%2Fcampaigns%2F7%2Fsessions%2F4')

    await entrar(user)

    expect(await screen.findByText('A cena da sessão')).toBeInTheDocument()
  })

  it('sem destino guardado, entra no saguão', async () => {
    vi.spyOn(api.auth, 'login').mockResolvedValue(EU)
    const { user } = renderLogin('/login')

    await entrar(user)

    expect(await screen.findByText('O saguão')).toBeInTheDocument()
  })

  it('semeia a sessão no cache ANTES de navegar', async () => {
    vi.spyOn(api.auth, 'login').mockResolvedValue(EU)
    const { user, client } = renderLogin('/login?redirect=%2Fcampaigns%2F7%2Fsessions%2F4')

    await entrar(user)

    // Semear em vez de invalidar é decisão registrada no código: o guarda da
    // próxima rota lê isto IMEDIATAMENTE, e um refetch correria com a
    // navegação — o usuário recém-logado tomaria um 401 do próprio guarda.
    await waitFor(() => expect(client.getQueryData(meQueryOptions.queryKey)).toEqual(EU))
  })

  it('credencial recusada não navega para lugar nenhum', async () => {
    vi.spyOn(api.auth, 'login').mockRejectedValue(new Error('Credenciais inválidas'))
    const { user } = renderLogin('/login?redirect=%2Fcampaigns%2F7%2Fsessions%2F4')

    await entrar(user)

    await waitFor(() => expect(screen.getByRole('button', { name: 'Entrar' })).toBeEnabled())
    expect(screen.queryByText('A cena da sessão')).not.toBeInTheDocument()
  })
})
