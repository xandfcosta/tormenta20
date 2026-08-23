import { QueryClient, QueryClientProvider } from '@tanstack/solid-query'
import { render, screen, waitFor } from '@solidjs/testing-library'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { characterQueryOptions } from '@/entities/character/queries'
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
    conditionals: [],
    powerUses: [],
    stances: [],
    ...overrides,
  }
}

function renderPanel(char: Character = character()) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  // A escrita do tibar é otimista e pinta SOBRE o cache; sem a linha semeada
  // ela não teria o que repintar e o teste mediria só a chamada de rede.
  client.setQueryData(characterQueryOptions(char.id).queryKey, char)
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
  vi.restoreAllMocks()
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

  // A palavra "sobrecarga" sozinha não diz o que mudou na ficha. Os dois números
  // saem do motor (p141), que é quem já os aplicou no deslocamento e nas
  // perícias — a mochila MOSTRA a consequência, não a decora.
  it('a sobrecarga diz a consequência mecânica, não só a palavra', () => {
    renderPanel(character({ items: [item({ id: 1, quantity: 40, slots: 1 })] }))

    const aviso = screen.getByText(/Sobrecarregado \(p141\)/)
    expect(aviso).toHaveTextContent('-5 em Acrobacia, Furtividade e Ladinagem')
    expect(aviso).toHaveTextContent('-3m de deslocamento')
  })

  /**
   * A campanha que desligou a regra de carga (ALE-221).
   *
   * Este teste atravessa o motor WASM de verdade — o `ignoredRules` viaja na
   * ficha e é o Go que decide se há sobrecarga —, então ele protege o CANO
   * inteiro e não a formatação. Sem ele, o dia em que a ficha parar de carregar
   * o campo, a tela volta a punir uma mesa que dispensou a regra e nada acusa.
   *
   * As duas metades importam: a punição some E o número fica. O livro deixa o
   * mestre ignorar a regra "desde que os jogadores não abusem" (p141), e quem
   * vigia abuso precisa do número na tela.
   */
  it('com a regra desligada na campanha, não há sobrecarga — mas a conta continua', () => {
    renderPanel(
      character({
        items: [item({ id: 1, quantity: 40, slots: 1 })],
        ignoredRules: { carga: true },
      }),
    )

    expect(screen.queryByText('sobrecarga')).not.toBeInTheDocument()
    expect(screen.queryByText(/Sobrecarregado \(p141\)/)).not.toBeInTheDocument()
    expect(screen.getByText('a critério do mestre')).toBeInTheDocument()
    expect(screen.getByText('40')).toBeInTheDocument()
    expect(screen.getByText(/limite 18/)).toBeInTheDocument()
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

  // O dinheiro é do personagem e o kit inicial já o concede (Tabela 3-1, p140):
  // a mochila mostra o MESMO campo, não um segundo lugar onde ele mora.
  it('mostra quantos tibares o personagem tem', () => {
    renderPanel(character({ tibar: 1250.5 }))

    expect(screen.getByText('T$ 1.250,5')).toBeInTheDocument()
  })

  /**
   * O diálogo do dinheiro tem TRÊS gestos e o fio carrega SALDO (ALE-224).
   *
   * O que estes testes protegem é a tradução entre os dois: o jogador diz
   * "ganhei 350" e o servidor recebe o total resultante. Errar isso grava 350
   * em cima de 1.200 sem avisar ninguém — o erro silencioso que a prévia e o
   * verbo no botão existem para impedir.
   */
  it('receber SOMA ao saldo, e o fio leva o total', async () => {
    const api = await import('@/shared/api/api')
    const update = vi.spyOn(api.api.characters, 'updateTibar').mockResolvedValue({ tibar: 370 })
    const user = renderPanel(character({ tibar: 20 }))

    await user.click(screen.getByRole('button', { name: 'Editar tibares' }))
    const campo = screen.getByLabelText('T$')
    await user.clear(campo)
    await user.type(campo, '350')
    // A prévia diz o que o botão vai fazer, ANTES de ele ser apertado.
    expect(screen.getByText('T$ 20 → T$ 370')).toBeInTheDocument()
    await user.click(screen.getByRole('button', { name: 'Receber T$ 350' }))

    await waitFor(() => expect(update).toHaveBeenCalledWith(1, { tibar: 370 }))
  })

  it('gastar SUBTRAI, e o fio continua levando o total', async () => {
    const api = await import('@/shared/api/api')
    const update = vi.spyOn(api.api.characters, 'updateTibar').mockResolvedValue({ tibar: 120 })
    const user = renderPanel(character({ tibar: 200 }))

    await user.click(screen.getByRole('button', { name: 'Editar tibares' }))
    await user.click(screen.getByRole('button', { name: 'Gastar' }))
    const campo = screen.getByLabelText('T$')
    await user.clear(campo)
    await user.type(campo, '80')
    await user.click(screen.getByRole('button', { name: 'Gastar T$ 80' }))

    await waitFor(() => expect(update).toHaveBeenCalledWith(1, { tibar: 120 }))
  })

  it('corrigir ESCREVE o saldo, que é o gesto da Forja', async () => {
    const api = await import('@/shared/api/api')
    const update = vi.spyOn(api.api.characters, 'updateTibar').mockResolvedValue({ tibar: 300 })
    const user = renderPanel(character({ tibar: 20 }))

    await user.click(screen.getByRole('button', { name: 'Editar tibares' }))
    await user.click(screen.getByRole('button', { name: 'Corrigir' }))
    const campo = screen.getByLabelText('T$')
    await user.clear(campo)
    await user.type(campo, '300')
    await user.click(screen.getByRole('button', { name: 'Corrigir T$ 300' }))

    await waitFor(() => expect(update).toHaveBeenCalledWith(1, { tibar: 300 }))
  })

  // Recusa ANTES da rede: com a escrita otimista, deixar o servidor responder
  // faria o saldo (e a barra da mochila, que é carga) piscar e voltar.
  it('gastar mais do que se tem é recusado dizendo quanto se tem', async () => {
    const api = await import('@/shared/api/api')
    const update = vi.spyOn(api.api.characters, 'updateTibar').mockResolvedValue({ tibar: 0 })
    const user = renderPanel(character({ tibar: 50 }))

    await user.click(screen.getByRole('button', { name: 'Editar tibares' }))
    await user.click(screen.getByRole('button', { name: 'Gastar' }))
    const campo = screen.getByLabelText('T$')
    await user.clear(campo)
    await user.type(campo, '80')
    await user.click(screen.getByRole('button', { name: 'Gastar T$ 80' }))

    expect(await screen.findByText('Você tem T$ 50.')).toBeInTheDocument()
    expect(update).not.toHaveBeenCalled()
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
