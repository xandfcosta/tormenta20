import { render, screen, waitFor } from '@solidjs/testing-library'
import { QueryClient, QueryClientProvider } from '@tanstack/solid-query'
import {
  createMemoryHistory,
  createRootRoute,
  createRoute,
  createRouter,
  RouterProvider,
} from '@tanstack/solid-router'
import { createSignal } from 'solid-js'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { campaignMembersQueryOptions, campaignQueryOptions } from '@/entities/campaign/queries'
import { campaignSessionQueryOptions } from '@/entities/session/queries'
import { meQueryOptions } from '@/entities/user/queries'
import type { AuthUser, Campaign, Session } from '@/shared/api/api'
import type { SessionRealtime } from '@/shared/realtime/realtime'
import { PowerUsesProvider } from '@/shared/stores/power-uses-context'
import { fakePowerUses } from '@/shared/test/play-stores'
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

/** O descanso que o MESTRE transmite, do lado de fora do mock: o teste o
 *  dispara como o socket faria, e a página reage. */
const [restFlash, setRestFlash] = createSignal<'scene' | 'day' | null>(null)

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
        restFlash: () => restFlash(),
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
  // Um membro MEU e um de outra pessoa: o descanso zera os contadores das
  // minhas fichas, e a segunda linha é o que prova que ele não sai zerando o
  // que não é meu (ALE-223).
  client.setQueryData(campaignMembersQueryOptions(1).queryKey, [
    { id: 1, campaignId: 1, characterId: 7, role: 'player', addedAt: '', character: { id: 7, ownerId: 9 } },
    { id: 2, campaignId: 1, characterId: 8, role: 'player', addedAt: '', character: { id: 8, ownerId: 99 } },
  ] as never)
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
  // O `PowerUsesProvider` é o mesmo que o `main.tsx` põe acima de tudo: a
  // página lê os contadores de uso para zerá-los no descanso do mestre
  // (ALE-223), e a ficha que ela monta já os lia antes.
  // Store real com o servidor MUDO (ALE-222): o cache local continua de
  // verdade, então o descanso do mestre é exercitado como na tela — só o que
  // iria pelo fio é que não vai.
  const powerUses = fakePowerUses()
  const rendered = render(() => (
    <UiProvider store={createUiStore(new FakeStorage())}>
      <PowerUsesProvider store={powerUses}>
        <QueryClientProvider client={client}>
          {/* biome-ignore lint/suspicious/noExplicitAny: o router de teste tem uma rota só */}
          <RouterProvider router={router as any} />
        </QueryClientProvider>
      </PowerUsesProvider>
    </UiProvider>
  ))
  return { ...rendered, powerUses }
}

beforeEach(() => {
  setRestFlash(null)
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

/**
 * O DESCANSO DO MESTRE zera os usos por cena/dia de quem está na sala (ALE-223).
 *
 * Os efeitos de escopo o servidor já expirava para a mesa inteira, mas "usado
 * 1/cena" é contador LOCAL e nada do lado do jogador o tocava — o botão
 * "Encerrar cena" da ficha mascarava o buraco, porque o jogador acabava zerando
 * sozinho. Com o botão fora da ficha (decisão do dono), este é o único caminho.
 *
 * O teste monta a PÁGINA e dispara o descanso pelo mesmo sinal que o socket
 * usa, porque é composição: o defeito não é a conta, é ela não chegar.
 */
describe('SessionTrackerPage — o descanso do mestre alcança os contadores locais', () => {
  it('descanso de cena zera o uso por cena da minha ficha', async () => {
    const { powerUses } = renderTracker(mesaCom('player'))
    await screen.findByRole('group', { name: 'O que ver na sessão' })
    powerUses.bump(7, 'class.bardo.inspiracao', 'scene')
    powerUses.bump(7, 'class.barbaro.furia', 'day')
    powerUses.bump(8, 'class.bardo.inspiracao', 'scene')

    setRestFlash('scene')

    await waitFor(() => {
      expect(powerUses.used(7, 'class.bardo.inspiracao').scene).toBe(0)
    })
    // Cena não é dia: quem descansa a cena não devolve o uso diário.
    expect(powerUses.used(7, 'class.barbaro.furia').day).toBe(1)
    // E a ficha de outro jogador não é minha para zerar: o contador é local, e
    // quem o zera é o navegador do dono dela.
    expect(powerUses.used(8, 'class.bardo.inspiracao').scene).toBe(1)
  })

  it('descanso de dia zera os dois escopos', async () => {
    const { powerUses } = renderTracker(mesaCom('player'))
    await screen.findByRole('group', { name: 'O que ver na sessão' })
    powerUses.bump(7, 'class.bardo.inspiracao', 'scene')
    powerUses.bump(7, 'class.barbaro.furia', 'day')

    setRestFlash('day')

    await waitFor(() => {
      expect(powerUses.used(7, 'class.barbaro.furia').day).toBe(0)
    })
    expect(powerUses.used(7, 'class.bardo.inspiracao').scene).toBe(0)
  })
})
