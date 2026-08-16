import { FakeStorage } from '@/shared/test/fake-storage'
import { QueryClient, QueryClientProvider } from '@tanstack/solid-query'
import { render, screen, waitFor } from '@solidjs/testing-library'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { makeCharacter } from '@/entities/character/__fixtures__/character'
import { characterQueryOptions } from '@/entities/character/queries'
import type { Character } from '@/shared/api/api'
import { ConditionalsProvider } from '@/shared/stores/conditionals-context'
import { createConditionalsStore } from '@/shared/stores/conditionals-store'
import { PowerUsesProvider } from '@/shared/stores/power-uses-context'
import { createPowerUsesStore } from '@/shared/stores/power-uses-store'
import { StanceActivationProvider } from '@/shared/stores/stance-activation-context'
import { createStanceActivationStore } from '@/shared/stores/stance-activation-store'
import { AbilitiesPanel } from './abilities-panel'


function renderPanel(char: Character = makeCharacter()) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  client.setQueryData(characterQueryOptions(char.id).queryKey, char)
  render(() => (
    <QueryClientProvider client={client}>
      <ConditionalsProvider store={createConditionalsStore(new FakeStorage())}>
        <PowerUsesProvider store={createPowerUsesStore(new FakeStorage())}>
          <StanceActivationProvider store={createStanceActivationStore(new FakeStorage())}>
            <AbilitiesPanel character={char} />
          </StanceActivationProvider>
        </PowerUsesProvider>
      </ConditionalsProvider>
    </QueryClientProvider>
  ))
  return { user: userEvent.setup(), client }
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

describe('AbilitiesPanel', () => {
  // Um personagem novo deve o essencial (benefícios de origem, poderes de
  // classe) — abrir no modo de jogo esconderia justamente o que falta fazer.
  it('abre em modo de edição quando há escolha pendente', () => {
    renderPanel()

    expect(screen.getByText('Faltam escolhas')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /Voltar ao jogo/ })).toBeInTheDocument()
  })

  it('as abas de fonte mostram quantas escolhas cada uma deve', () => {
    renderPanel()

    // Origem sempre deve 2 benefícios num personagem recém-criado.
    const origem = screen.getByRole('button', { name: /^Origem\s*\d*$/ })
    expect(origem.textContent).toMatch(/\d/)
  })

  it('trocar de aba mostra a seção daquela fonte', async () => {
    const { user } = renderPanel()

    await user.click(screen.getByRole('button', { name: /^Classe\s*\d*$/ }))

    expect(await screen.findByText(/^Bardo 3$/)).toBeInTheDocument()
  })

  it('a busca por nome atravessa as fontes', async () => {
    const { user } = renderPanel()

    await user.type(screen.getByRole('textbox', { name: 'Buscar poder ou habilidade' }), 'zzzz')

    expect(await screen.findByText(/Nenhum poder para "zzzz"/)).toBeInTheDocument()
  })

  // O modo de jogo é a lista da mesa: o que dá pra FAZER, com o afordance de
  // usar inline. É o padrão quando não há escolha pendente.
  it('voltar ao jogo mostra a lista de ações', async () => {
    const { user } = renderPanel(
      makeCharacter({ classes: [{ className: 'Bárbaro', level: 6 }] }),
    )

    await user.click(screen.getByRole('button', { name: /Voltar ao jogo/ }))

    expect(await screen.findByText('Ações')).toBeInTheDocument()
    // Fúria é postura: entra com o chip de custo e o botão de ativar.
    expect(screen.getByRole('button', { name: 'Ativar Fúria' })).toBeInTheDocument()
  })

  it('escolher um benefício de origem grava a lista', async () => {
    const api = await import('@/shared/api/api')
    const update = vi
      .spyOn(api.api.characters, 'updateAbilityChoices')
      .mockResolvedValue({ originChoices: '[]' })
    const { user } = renderPanel()

    await user.click(screen.getByRole('button', { name: /^Origem\s*\d*$/ }))
    const [first] = await screen.findAllByRole('button', { name: /^Selecionar benefício/ })
    await user.click(first)

    await waitFor(() => expect(update).toHaveBeenCalled())
    expect(update.mock.calls[0][1]).toHaveProperty('originChoices')
  })
})
