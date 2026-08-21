import { QueryClient, QueryClientProvider } from '@tanstack/solid-query'
import {
  RouterProvider,
  createMemoryHistory,
  createRootRoute,
  createRoute,
  createRouter,
} from '@tanstack/solid-router'
import { render, screen } from '@solidjs/testing-library'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { characterOptionsQueryOptions } from '@/entities/character/queries'
import { wizardDefaults } from '@/features/character-build/wizard-steps'
import { CHARACTER_DRAFT_STORAGE_KEY } from '@/shared/stores/character-draft-store'
import type { CharacterOptions } from '@/shared/api/api'
import { UiProvider } from '@/shared/stores/ui-context'
import { createUiStore } from '@/shared/stores/ui-store'
import { FakeStorage } from '@/shared/test/fake-storage'
import { ForgePage } from './forge-page'

/**
 * A CAMINHADA da forja, montada com a URL de verdade (ALE-169).
 *
 * O passo de Poderes é uma tela preta com uma frase para todo personagem novo,
 * porque a forja cria nível 1 e a primeira vaga de poder é do SEGUNDO nível de
 * uma classe. A regra que o tira mora em `wizard-steps` e tem unitário; o que
 * só esta camada prova é a COMPOSIÇÃO — que a página anda pela caminhada
 * derivada em vez do catálogo, e que quem chega por um endereço guardado do
 * passo que saiu não fica preso nele.
 */
const OPTIONS = {
  races: ['Humano', 'Anão'],
  classes: ['Guerreiro', 'Arcanista'],
  origins: ['Artista'],
  gods: [],
  sizes: ['Médio'],
  expertises: [],
} as unknown as CharacterOptions

/** Um Guerreiro de nível 1: nenhuma vaga de poder, que é o caso de TODO
 *  personagem que a forja cria hoje. */
const RASCUNHO_NIVEL_1 = JSON.stringify({
  state: {
    values: {
      ...wizardDefaults,
      races: ['Humano'],
      classes: [{ className: 'Guerreiro', level: 1 }],
    },
    raceChoices: {},
    attributeMode: 'points',
  },
})

function renderForge(rota: string) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  client.setQueryData(characterOptionsQueryOptions.queryKey, OPTIONS)

  // A página chama `createCharacterDraftStore()` SEM argumento, e o padrão dele
  // é `globalThis.localStorage` — que o jsdom não expõe numa origem opaca. O
  // rascunho tem de estar onde a página vai procurar.
  const storage = new FakeStorage()
  storage.setItem(CHARACTER_DRAFT_STORAGE_KEY, RASCUNHO_NIVEL_1)
  vi.stubGlobal('localStorage', storage)

  const root = createRootRoute()
  const shell = createRoute({
    getParentRoute: () => root,
    path: '/characters/new',
    component: ForgePage,
  })
  const passo = createRoute({
    getParentRoute: () => shell,
    path: '$step',
    component: () => null,
  })
  const router = createRouter({
    routeTree: root.addChildren([shell.addChildren([passo])]),
    history: createMemoryHistory({ initialEntries: [rota] }),
  })

  render(() => (
    <UiProvider store={createUiStore(new FakeStorage())}>
      <QueryClientProvider client={client}>
        {/* biome-ignore lint/suspicious/noExplicitAny: o router de teste tem duas rotas */}
        <RouterProvider router={router as any} />
      </QueryClientProvider>
    </UiProvider>
  ))
  return { router }
}

describe('ForgePage — a caminhada pula o passo que nunca tem conteúdo', () => {
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

  // `stubGlobal` não se desfaz sozinho nesta configuração, e um `localStorage`
  // falso vazando para os outros arquivos é a regra de INDEPENDÊNCIA quebrada:
  // a suíte cheia caiu em três testes de outra pasta antes disto entrar.
  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it('a trilha não oferece Poderes num personagem de nível 1', async () => {
    renderForge('/characters/new/classe')

    const trilha = await screen.findByRole('list', { name: 'Passos da criação' })

    expect(trilha.textContent).not.toContain('Poderes')
    expect(await screen.findByText(/Passo 2 de 8/)).toBeInTheDocument()
  })

  it('o Próximo anda de Classe direto para Origem', async () => {
    const { router } = renderForge('/characters/new/classe')

    await userEvent.click(await screen.findByRole('button', { name: /Próximo/ }))

    expect(router.state.location.pathname).toBe('/characters/new/origem')
  })

  /**
   * Um endereço guardado do passo que saiu não pode virar beco sem saída — e a
   * saída não é redirecionar, que dentro de um efeito que observa a URL vira
   * laço. A caminhada inclui onde o jogador está, e o Próximo anda dali.
   */
  it('quem chega por URL em Poderes fica num lugar coerente e segue em frente', async () => {
    const { router } = renderForge('/characters/new/poderes')

    expect(await screen.findByText(/Passo 3 de 9/)).toBeInTheDocument()

    await userEvent.click(await screen.findByRole('button', { name: /Próximo/ }))

    expect(router.state.location.pathname).toBe('/characters/new/origem')
  })
})
