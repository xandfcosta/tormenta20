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
import { api } from '@/shared/api/api'
import { DeleteSessionButton } from './delete-session-button'

/**
 * EXCLUIR A SESSÃO (ALE-197, grupo A).
 *
 * A porta destrutiva mais rara do app não tinha teste nenhum, enquanto a irmã
 * dela — o excluir da CAMPANHA — tinha quatro. E este era o caso MAIS difícil
 * dos dois: aquele recebia o `onConfirm` por prop, e este chama a api, invalida
 * a lista e NAVEGA — três coisas que só a montagem prova.
 *
 * A irmã não existe mais: a crônica virou cena do servidor na ALE-255 e o
 * `delete-campaign-button` foi junto. A comparação fica porque ela explica por
 * que ESTE teste é o caro; o caminho do arquivo saiu porque apontar para um
 * arquivo apagado é o defeito que a ALE-173 registrou — dois comentários do
 * `index.css` mandaram "validar live at /grimorio" por meses depois de a
 * página ter sido apagada.
 *
 * A navegação é afirmada pela tela de destino aparecendo, e não por um espião
 * no `useNavigate`: o que importa é que a pessoa saia da sessão que acabou de
 * apagar, não qual função foi chamada.
 */

function renderButton() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  const root = createRootRoute()
  const sessao = createRoute({
    getParentRoute: () => root,
    path: '/campaigns/$id/sessions/$sid',
    component: () => <DeleteSessionButton campaignId={7} sessionId={4} sessionNumber={3} />,
  })
  const cronica = createRoute({
    getParentRoute: () => root,
    path: '/campaigns/$id',
    component: () => <p>A campanha</p>,
  })
  const router = createRouter({
    routeTree: root.addChildren([sessao, cronica]),
    history: createMemoryHistory({ initialEntries: ['/campaigns/7/sessions/4'] }),
  })
  render(() => (
    <QueryClientProvider client={client}>
      {/* biome-ignore lint/suspicious/noExplicitAny: router de teste com duas rotas */}
      <RouterProvider router={router as any} />
    </QueryClientProvider>
  ))
  return userEvent.setup()
}

afterEach(() => {
  vi.restoreAllMocks()
  document.body.innerHTML = ''
})

describe('DeleteSessionButton', () => {
  it('o gatilho só abre a confirmação, não exclui', async () => {
    const remove = vi.spyOn(api.sessions, 'remove')
    const user = renderButton()

    await user.click(await screen.findByRole('button', { name: /Excluir/ }))

    expect(await screen.findByRole('dialog')).toHaveAccessibleName('Excluir sessão 3?')
    expect(remove).not.toHaveBeenCalled()
  })

  it('confirmar exclui a sessão certa e tira a pessoa de lá', async () => {
    const remove = vi.spyOn(api.sessions, 'remove').mockResolvedValue({ id: 4 })
    const user = renderButton()
    await user.click(await screen.findByRole('button', { name: /Excluir/ }))

    await user.click(await screen.findByRole('button', { name: 'Excluir' }))

    // Campanha E sessão: um id trocado apagaria a sessão errada, e nenhum
    // teste de helper pega isso.
    await waitFor(() => expect(remove).toHaveBeenCalledWith(7, 4))
    // Ficar na cena de uma sessão que não existe mais é o defeito clássico
    // desta família — a tela seguiria pedindo estado de um id apagado.
    expect(await screen.findByText('A campanha')).toBeInTheDocument()
  })

  it('cancelar fecha sem excluir', async () => {
    const remove = vi.spyOn(api.sessions, 'remove')
    const user = renderButton()
    await user.click(await screen.findByRole('button', { name: /Excluir/ }))

    await user.click(await screen.findByRole('button', { name: 'Cancelar' }))

    await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument())
    expect(remove).not.toHaveBeenCalled()
  })

  it('exclusão que FALHA não leva a pessoa embora', async () => {
    vi.spyOn(api.sessions, 'remove').mockRejectedValue(new Error('servidor fora do ar'))
    const user = renderButton()
    await user.click(await screen.findByRole('button', { name: /Excluir/ }))

    await user.click(await screen.findByRole('button', { name: 'Excluir' }))

    // Navegar depois de uma falha diria "apagamos" sobre uma sessão que
    // continua lá — a mesa voltaria para a campanha achando que acabou.
    await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument())
    expect(screen.queryByText('A campanha')).not.toBeInTheDocument()
  })
})
