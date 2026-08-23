import { QueryClient, QueryClientProvider } from '@tanstack/solid-query'
import { render, screen, waitFor } from '@solidjs/testing-library'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { makeCharacter } from '@/entities/character/__fixtures__/character'
import { characterQueryOptions } from '@/entities/character/queries'
import { type Character, api } from '@/shared/api/api'
import { ConditionalsProvider } from '@/shared/stores/conditionals-context'
import { PowerUsesProvider } from '@/shared/stores/power-uses-context'
import { StanceActivationProvider } from '@/shared/stores/stance-activation-context'
import { AbilitiesPanel } from './abilities-panel'
import { fakeConditionals, fakePowerUses, fakeStances } from '@/shared/test/play-stores'


function renderPanel(char: Character = makeCharacter()) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  client.setQueryData(characterQueryOptions(char.id).queryKey, char)
  render(() => (
    <QueryClientProvider client={client}>
      <ConditionalsProvider store={fakeConditionals()}>
        <PowerUsesProvider store={fakePowerUses()}>
          <StanceActivationProvider store={fakeStances()}>
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

    // Origem sempre deve 2 benefícios num personagem recém-criado, e a aba
    // ANUNCIA o que o número significa: a pílula é `aria-hidden` e a linha
    // `sr-only` ao lado diz "N escolhas pendentes" (ALE-173, P6). Antes o
    // número era lido cru, e "Origem 2" não diz nada a quem não vê a pílula.
    expect(screen.getByRole('button', { name: 'Origem, 1 escolha pendente' })).toBeTruthy()
  })

  it('trocar de aba mostra a seção daquela fonte', async () => {
    const { user } = renderPanel()

    await user.click(screen.getByRole('button', { name: /^Classe, / }))

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

    await user.click(screen.getByRole('button', { name: 'Origem, 1 escolha pendente' }))
    const [first] = await screen.findAllByRole('button', { name: /^Selecionar benefício/ })
    await user.click(first)

    await waitFor(() => expect(update).toHaveBeenCalled())
    expect(update.mock.calls[0][1]).toHaveProperty('originChoices')
  })
})

/**
 * O bônus de atributo da raça, terminado na FICHA (ALE-169).
 *
 * A forja oferece criar sem colocar os +1 — o passo de Resumo diz "dá para
 * criar assim e terminar na ficha" e nomeia a pendência. A ficha então não a
 * listava e não tinha onde colocá-los: o personagem ficava ilegal pelo livro
 * ("Sua raça modifica seus atributos", p18) sem conserto que não fosse refazer
 * a forja. O fixture é um Humano com `raceAttributeChoices` vazio, que é
 * exatamente o que a forja produz.
 */
describe('AbilitiesPanel — o +1 de raça que a forja deixou', () => {
  it('mostra onde distribuir o bônus, com a conta do que falta', async () => {
    renderPanel()

    expect(await screen.findByText(/Distribua \+1 em 3 atributos · 0\/3/)).toBeInTheDocument()
  })

  it('clicar num atributo salva a escolha', async () => {
    const { user, client } = renderPanel()
    const escrito: unknown[] = []
    vi.spyOn(api.characters, 'updateAbilityChoices').mockImplementation(
      async (_id, input) => {
        escrito.push(input)
        return { raceAttributeChoices: JSON.stringify(input.raceAttributeChoices) }
      },
    )

    await user.click(await screen.findByRole('button', { name: 'FOR' }))

    await waitFor(() => expect(escrito).toHaveLength(1))
    expect(escrito[0]).toEqual({ raceAttributeChoices: { floatingPicks: ['strength'] } })
    expect(client).toBeTruthy()
  })
})
