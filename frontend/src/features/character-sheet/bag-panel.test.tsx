import { QueryClient, QueryClientProvider } from '@tanstack/solid-query'
import { render, screen } from '@solidjs/testing-library'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import type { Character, CharacterItem } from '@/shared/api/api'
import { BagPanel } from './bag-panel'

function item(overrides: Partial<CharacterItem> = {}): CharacterItem {
  return {
    id: 1,
    catalogId: null,
    name: 'Corda de cânhamo',
    quantity: 1,
    slots: 1,
    equipped: null,
    improvements: '[]',
    material: null,
    ...overrides,
  }
}

function character(overrides: Partial<Character> = {}): Character {
  return {
    id: 1,
    ownerId: 1,
    name: 'Tanque Placas',
    origin: 'Soldado',
    god: null,
    godPower: '',
    tibar: 0,
    level: 10,
    hpMax: 100,
    hpCurrent: 100,
    mpMax: 20,
    mpCurrent: 20,
    strength: 4,
    dexterity: 1,
    constitution: 4,
    intelligence: 1,
    wisdom: 2,
    charisma: 1,
    size: 'Médio',
    displacement: 9,
    proficiencies: '[]',
    raceAbilityChoices: '[]',
    activeConditions: '[]',
    raceAttributeChoices: '{}',
    secondaryRaceChoices: '[]',
    originChoices: '[]',
    classPowers: '[]',
    classChoices: '{}',
    powerChoices: '{}',
    createdAt: '2026-01-01T00:00:00.000Z',
    updatedAt: '2026-01-01T00:00:00.000Z',
    races: [{ race: 'Humano' }],
    classes: [{ className: 'Guerreiro', level: 10 }],
    expertises: [],
    items: [],
    activeEffects: [],
    spells: [],
    ...overrides,
  }
}

function renderPanel(char: Character = character()) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  render(() => (
    <QueryClientProvider client={client}>
      <BagPanel character={char} />
    </QueryClientProvider>
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
  vi.clearAllMocks()
  document.body.innerHTML = ''
})

describe('BagPanel', () => {
  it('mostra a carga usada contra o limite do personagem', () => {
    renderPanel(
      character({ items: [item({ id: 1, quantity: 3, slots: 0.5 })] }),
    )
    // 3 × 0,5 = 1,5 espaço; limite = 10 + 2×For(+4) = 18.
    expect(screen.getByText('1,5')).toBeInTheDocument()
    expect(screen.getByText(/limite 18 · 10 \+ 2×For \+4/)).toBeInTheDocument()
  })

  it('avisa a sobrecarga quando a carga passa do limite', () => {
    renderPanel(character({ items: [item({ id: 1, quantity: 40, slots: 1 })] }))
    expect(screen.getByText('sobrecarga')).toBeInTheDocument()
  })

  it('a mochila vazia explica como encher', () => {
    renderPanel()
    expect(screen.getByText(/Mochila vazia/)).toBeInTheDocument()
  })

  it('mostra os equipados nas duas caixas de capacidade', () => {
    renderPanel(
      character({
        items: [
          item({ id: 1, catalogId: 'espada-longa', name: 'Espada longa', equipped: 'wielded' }),
          item({ id: 2, catalogId: 'couraca', name: 'Couraça', equipped: 'vested' }),
        ],
      }),
    )
    expect(screen.getByText('1/2')).toBeInTheDocument()
    expect(screen.getByText('1/4')).toBeInTheDocument()
    expect(screen.getByLabelText('Abrir Espada longa')).toBeInTheDocument()
    expect(screen.getByLabelText('Desequipar Couraça')).toBeInTheDocument()
  })

  // Uma arma de duas mãos ocupa as DUAS — a caixa tem de dizer 2/2, senão o
  // jogador acha que ainda pode empunhar um escudo.
  it('arma de duas mãos ocupa as duas mãos', () => {
    renderPanel(
      character({
        items: [item({ id: 1, catalogId: 'montante', name: 'Montante', equipped: 'wielded2' })],
      }),
    )
    expect(screen.getByText('2/2')).toBeInTheDocument()
    expect(screen.getByText('Duas mãos')).toBeInTheDocument()
  })

  it('só lista na grade o que está guardado', () => {
    renderPanel(
      character({
        items: [
          item({ id: 1, name: 'Espada longa', catalogId: 'espada-longa', equipped: 'wielded' }),
          item({ id: 2, name: 'Corda de cânhamo' }),
        ],
      }),
    )
    expect(screen.getByLabelText('Abrir Corda de cânhamo')).toBeInTheDocument()
    expect(screen.getByText('1 item')).toBeInTheDocument()
  })

  it('a busca e o filtro de categoria estreitam a grade', async () => {
    const user = renderPanel(
      character({
        items: [
          item({ id: 1, name: 'Espada longa', catalogId: 'espada-longa' }),
          item({ id: 2, name: 'Bálsamo restaurador', catalogId: 'balsamo-restaurador' }),
        ],
      }),
    )

    await user.type(screen.getByLabelText('Buscar item na mochila'), 'balsamo')
    expect(screen.queryByLabelText('Abrir Espada longa')).not.toBeInTheDocument()
    expect(screen.getByLabelText('Abrir Bálsamo restaurador')).toBeInTheDocument()

    await user.clear(screen.getByLabelText('Buscar item na mochila'))
    await user.click(screen.getByRole('button', { name: 'armas' }))
    expect(screen.getByLabelText('Abrir Espada longa')).toBeInTheDocument()
    expect(screen.queryByLabelText('Abrir Bálsamo restaurador')).not.toBeInTheDocument()

    await user.click(screen.getByRole('button', { name: 'defesa' }))
    expect(screen.getByText('Nenhum item para esse filtro.')).toBeInTheDocument()
  })

  it('abrir um item leva à ficha dele com as ações de equipar', async () => {
    const user = renderPanel(
      character({
        items: [item({ id: 1, name: 'Espada longa', catalogId: 'espada-longa' })],
      }),
    )

    await user.click(screen.getByLabelText('Abrir Espada longa'))
    expect(screen.getByRole('dialog')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Empunhar (1 mão)' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Remover Espada longa' })).toBeInTheDocument()
  })
})
