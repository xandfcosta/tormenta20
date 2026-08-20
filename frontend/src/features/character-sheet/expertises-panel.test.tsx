import { QueryClient, QueryClientProvider } from '@tanstack/solid-query'
import { render, screen } from '@solidjs/testing-library'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import type { Character, CharacterExpertise } from '@/shared/api/api'
import { ExpertisesPanel, expertiseDefsFor, filterExpertises } from './expertises-panel'

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

const custom = (name: string): CharacterExpertise => ({
  name,
  attribute: 'strength',
  trained: true,
  custom: true,
})

describe('expertiseDefsFor', () => {
  // "Teste de Reflexos CD 20!" é a consulta mais quente da mesa.
  it('fixa as resistências no topo', () => {
    const names = expertiseDefsFor(character()).slice(0, 3).map((d) => d.name)
    expect(names).toEqual(['Fortitude', 'Reflexos', 'Vontade'])
  })

  it('acrescenta os ofícios do personagem ao fim', () => {
    const defs = expertiseDefsFor(character({ expertises: [custom('Ferraria')] }))
    expect(defs.at(-1)).toMatchObject({ name: 'Ferraria', trainedOnly: true })
  })

  it('ignora as perícias padrão do personagem — elas já vêm do livro', () => {
    const withStandard = character({
      expertises: [{ name: 'Atletismo', attribute: 'strength', trained: true, custom: false }],
    })
    const atletismo = expertiseDefsFor(withStandard).filter((d) => d.name === 'Atletismo')
    expect(atletismo).toHaveLength(1)
  })
})

describe('filterExpertises', () => {
  const defs = expertiseDefsFor(character())

  it('sem busca, devolve tudo', () => {
    expect(filterExpertises(defs, '   ')).toHaveLength(defs.length)
  })

  // Ninguém digita acento no meio da mesa: "percepcao" tem de achar "Percepção",
  // e "SOBREV" tem de achar "Sobrevivência".
  it('acha ignorando acento e caixa', () => {
    expect(filterExpertises(defs, 'percepcao').map((d) => d.name)).toEqual(['Percepção'])
    expect(filterExpertises(defs, 'SOBREV').map((d) => d.name)).toEqual(['Sobrevivência'])
    expect(filterExpertises(defs, 'ATLET').map((d) => d.name)).toEqual(['Atletismo'])
  })
})

function renderPanel(char: Character = character(), glance = false) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  render(() => (
    <QueryClientProvider client={client}>
      <ExpertisesPanel character={char} glance={glance} />
    </QueryClientProvider>
  ))
  return client
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

describe('ExpertisesPanel', () => {
  it('mostra o cabeçalho com o treino e o meio-nível do personagem', () => {
    renderPanel()
    expect(screen.getByRole('heading', { name: 'Perícias' })).toBeInTheDocument()
    // Nível 10 → ½ nível 5.
    expect(screen.getByText(/½ nível 5/)).toBeInTheDocument()
  })

  it('lista as perícias do livro, resistências primeiro', () => {
    renderPanel()
    expect(screen.getByRole('switch', { name: 'Fortitude treinada' })).toBeInTheDocument()
    expect(screen.getByRole('switch', { name: 'Atletismo treinada' })).toBeInTheDocument()
  })

  it('a busca filtra a lista e explica quando não acha nada', async () => {
    renderPanel()
    const user = userEvent.setup()

    await user.type(screen.getByLabelText('Buscar perícia'), 'atlet')
    expect(screen.getByRole('switch', { name: 'Atletismo treinada' })).toBeInTheDocument()
    expect(screen.queryByRole('switch', { name: 'Fortitude treinada' })).not.toBeInTheDocument()

    await user.clear(screen.getByLabelText('Buscar perícia'))
    await user.type(screen.getByLabelText('Buscar perícia'), 'zzz')
    expect(screen.getByText(/Nenhuma perícia para "zzz"/)).toBeInTheDocument()
  })

  it('o toggle de treino anuncia seu estado', () => {
    renderPanel(
      character({
        expertises: [{ name: 'Atletismo', attribute: 'strength', trained: true, custom: false }],
      }),
    )
    expect(screen.getByRole('switch', { name: 'Atletismo treinada' })).toHaveAttribute(
      'aria-checked',
      'true',
    )
    expect(screen.getByRole('switch', { name: 'Furtividade treinada' })).toHaveAttribute(
      'aria-checked',
      'false',
    )
  })

  // Só ofício inventado pode ser apagado; perícia do livro, não.
  it('oferece remover apenas nos ofícios do jogador', () => {
    renderPanel(character({ expertises: [custom('Ferraria')] }))
    expect(screen.getByLabelText('Remover Ferraria')).toBeInTheDocument()
    expect(screen.queryByLabelText('Remover Atletismo')).not.toBeInTheDocument()
  })
})

/**
 * Na tela do mestre a perícia é uma CONSULTA: ele quer o número no meio do
 * turno, não a matemática dele (ALE-145). As chips da linha repetem o diálogo
 * de decomposição palavra por palavra e custam uma segunda linha por perícia.
 *
 * O que estes testes protegem não é "a chip sumiu" — é que a composição
 * continua ALCANÇÁVEL depois de sumir da linha. Uma asserção só de ausência
 * passaria verde se o diálogo tivesse ido junto.
 */
describe('Perícias na tela do mestre (glance)', () => {
  it('a linha fica com o total e os controles, sem a composição', () => {
    renderPanel(character(), true)

    expect(screen.getByRole('switch', { name: 'Atletismo treinada' })).toBeInTheDocument()
    expect(screen.getByLabelText('Atletismo atributo')).toBeInTheDocument()
    // ½ nível 5 e treino +6 num personagem de nível 10: as chips diziam isso na
    // linha, uma vez por perícia.
    expect(screen.queryByText('½lvl')).not.toBeInTheDocument()
    expect(screen.queryByText('outros')).not.toBeInTheDocument()
  })

  it('a composição continua a um clique, no diálogo', async () => {
    renderPanel(character(), true)
    const user = userEvent.setup()

    await user.click(screen.getAllByRole('button', { name: /^Detalhar/ })[0])

    const dialog = await screen.findByRole('dialog')
    expect(dialog).toHaveTextContent('½ nível')
    expect(dialog).toHaveTextContent('Treino')
    expect(dialog).toHaveTextContent('Outros')
    expect(dialog).toHaveTextContent('Total')
  })

  it('na ficha do JOGADOR as chips continuam na linha', () => {
    renderPanel(character(), false)

    expect(screen.getAllByText('½lvl').length).toBeGreaterThan(0)
    expect(screen.getAllByText('outros').length).toBeGreaterThan(0)
  })
})

/**
 * No TELEFONE a linha usa o mesmo arranjo do mestre (ALE-183).
 *
 * Medido a 390px: as chips custavam 27px dos 108 de cada perícia, e a janela de
 * rolagem mostrava 4,3 perícias de vinte. Sem elas, 6,7 — rolar de quatro em
 * quatro, numa área que não é a rolagem da página, é caro na mesa.
 *
 * As chips não somem do APP: elas repetem, palavra por palavra, o diálogo que o
 * próprio total abre — e é por isso que dá para recuá-las sem perder a
 * auditoria. O que NÃO pode sumir é o seletor de atributo, que é controle e não
 * informação: ele sobe para a linha do nome, como no `glance`.
 */
describe('Perícias no telefone', () => {
  /** O gate é largura + ORIENTAÇÃO: o celular deitado tem 844px de largura com
   *  390 de altura, e é onde 27px por linha custam mais caro (ALE-162). */
  const comoTelefone = () =>
    vi.fn().mockImplementation((media: string) => ({
      matches: media.includes('max-width: 639px') || media.includes('orientation: landscape'),
      media,
      onchange: null,
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
      addListener: vi.fn(),
      removeListener: vi.fn(),
      dispatchEvent: vi.fn(),
    }))

  it('as chips saem da linha, e o seletor de atributo FICA', () => {
    window.matchMedia = comoTelefone()

    renderPanel(character(), false)

    expect(screen.queryByText('½lvl')).not.toBeInTheDocument()
    expect(screen.queryByText('outros')).not.toBeInTheDocument()
    // Controle, não informação: sem ele o jogador não muda o atributo que a
    // perícia usa, e isso não tem outro caminho na tela.
    expect(screen.getByLabelText('Atletismo atributo')).toBeInTheDocument()
    expect(screen.getByRole('switch', { name: 'Atletismo treinada' })).toBeInTheDocument()
  })

  it('a auditoria continua a um toque, no diálogo', async () => {
    window.matchMedia = comoTelefone()
    renderPanel(character(), false)
    const user = userEvent.setup()

    await user.click(screen.getAllByRole('button', { name: /^Detalhar/ })[0])

    const dialog = await screen.findByRole('dialog')
    expect(dialog).toHaveTextContent('½ nível')
    expect(dialog).toHaveTextContent('Treino')
    expect(dialog).toHaveTextContent('Outros')
  })
})

/**
 * "INDEFESO. […] falha automaticamente em testes de Reflexos" (p394).
 *
 * O motor responde QUAIS perícias falham automaticamente
 * (`sheet.autoFailExpertises`) em vez de a UI reinterpretar uma flag — a regra
 * de que paralisado, inconsciente e petrificado implicam indefeso mora no Go.
 * Aqui só se prova que a linha para de mostrar um número, porque um total ao
 * lado de "falha automática" é a leitura errada na mesa (ALE-115).
 */
describe('Reflexos sob falha automática', () => {
  it('a linha mostra falha automática em vez do total', async () => {
    renderPanel(character({ activeConditions: '["paralisado"]' }))

    const badge = await screen.findByRole('button', { name: /Falha automática/ })
    expect(badge).toBeInTheDocument()
    expect(badge).toHaveTextContent('—')
  })

  it('sem a condição, a mesma linha mostra o número', async () => {
    renderPanel(character())

    expect(screen.queryByRole('button', { name: /Falha automática/ })).not.toBeInTheDocument()
  })
})
