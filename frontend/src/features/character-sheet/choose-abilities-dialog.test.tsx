import { QueryClient, QueryClientProvider } from '@tanstack/solid-query'
import { render, screen, waitFor, within } from '@solidjs/testing-library'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { makeCharacter } from '@/entities/character/__fixtures__/character'
import { characterQueryOptions } from '@/entities/character/queries'
import { type Character, api } from '@/shared/api/api'
import { ChooseAbilitiesDialog } from './choose-abilities-dialog'

/**
 * "Escolher poderes" — a administração da ficha, que era um MODO e virou
 * diálogo (ALE-217).
 *
 * O que este arquivo protege é o que o modo de edição protegia, mais a única
 * coisa que o diálogo acrescenta: ele ABRE na fonte que deve escolha. Quem abriu
 * veio pela pendência, e fazê-lo caçar a aba certa é gastar o clique que ele
 * acabou de dar.
 */

function renderDialog(char: Character = makeCharacter()) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  client.setQueryData(characterQueryOptions(char.id).queryKey, char)
  render(() => (
    <QueryClientProvider client={client}>
      <ChooseAbilitiesDialog character={char} />
    </QueryClientProvider>
  ))
  return { user: userEvent.setup(), client }
}

async function abre(user: ReturnType<typeof userEvent.setup>) {
  await user.click(screen.getByRole('button', { name: /^Escolher poderes/ }))
  return await screen.findByRole('dialog')
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

describe('Escolher poderes', () => {
  it('o botão abre o diálogo com o que falta escolher', async () => {
    const { user } = renderDialog()

    const dialog = await abre(user)

    expect(within(dialog).getByText('Faltam escolhas')).toBeInTheDocument()
  })

  /**
   * Abrir aterrissa na fonte que DEVE escolha, e não numa aba fixa.
   *
   * Quem abriu o diálogo veio pela pendência; fazê-lo caçar a aba certa é
   * gastar o clique que ele acabou de dar.
   *
   * São DOIS casos e não um, e isso foi decidido apanhando: com o fixture
   * cheio — que deve escolha nas três fontes — a primeira versão deste teste
   * passou VERDE com o diálogo abrindo fixo em "Classe", porque Classe também
   * devia. Um caso só não distingue "abriu onde deve" de "abriu sempre no
   * mesmo lugar". Com uma ficha que deve SÓ Origem e outra que deve SÓ Classe,
   * nenhuma aba fixa satisfaz as duas.
   */
  it('abre na Origem quando é só ela que deve', async () => {
    const { user } = renderDialog(soDeve('origem'))

    const dialog = await abre(user)

    expect(within(dialog).getByRole('button', { name: /^Origem,/ })).toHaveAttribute(
      'aria-pressed',
      'true',
    )
  })

  it('abre na Classe quando é só ela que deve', async () => {
    const { user } = renderDialog(soDeve('classe'))

    const dialog = await abre(user)

    expect(within(dialog).getByRole('button', { name: /^Classe,/ })).toHaveAttribute(
      'aria-pressed',
      'true',
    )
  })

  // Cada aba ANUNCIA o que o número significa: a pílula é `aria-hidden` e o
  // nome acessível diz "N escolhas pendentes" (ALE-173, P6). Antes o número era
  // lido cru, e "Origem 2" não diz nada a quem não vê a pílula.
  it('as abas dizem quantas escolhas cada fonte deve', async () => {
    const { user } = renderDialog()

    const dialog = await abre(user)

    expect(
      within(dialog).getByRole('button', { name: 'Origem, 1 escolha pendente' }),
    ).toBeInTheDocument()
  })

  it('trocar de aba mostra a seção daquela fonte', async () => {
    const { user } = renderDialog()
    const dialog = await abre(user)

    await user.click(within(dialog).getByRole('button', { name: /^Classe(,|$)/ }))

    expect(await within(dialog).findByText(/^Bardo 3$/)).toBeInTheDocument()
  })

  it('escolher um benefício de origem grava a lista', async () => {
    const update = vi
      .spyOn(api.characters, 'updateAbilityChoices')
      .mockResolvedValue({ originChoices: '[]' })
    const { user } = renderDialog()
    const dialog = await abre(user)

    await user.click(within(dialog).getByRole('button', { name: /^Origem(,|$)/ }))
    const [primeiro] = await within(dialog).findAllByRole('button', {
      name: /^Selecionar benefício/,
    })
    await user.click(primeiro)

    await waitFor(() => expect(update).toHaveBeenCalled())
    expect(update.mock.calls[0][1]).toHaveProperty('originChoices')
  })
})

/**
 * O +1 de raça que a forja deixou (ALE-169), agora dentro do diálogo.
 *
 * A forja oferece criar sem colocar os +1 — o passo de Resumo diz "dá para
 * criar assim e terminar na ficha" e nomeia a pendência. A ficha então não a
 * listava e não tinha onde colocá-los: o personagem ficava ilegal pelo livro
 * ("Sua raça modifica seus atributos", p18) sem conserto que não fosse refazer
 * a forja. O fixture é um Humano com `raceAttributeChoices` vazio, que é
 * exatamente o que a forja produz.
 */
describe('Escolher poderes — o +1 de raça', () => {
  it('mostra onde distribuir o bônus, com a conta do que falta', async () => {
    const { user } = renderDialog()
    const dialog = await abre(user)

    await user.click(within(dialog).getByRole('button', { name: /^Raça(,|$)/ }))

    expect(await within(dialog).findByText(/Distribua \+1 em 3 atributos · 0\/3/)).toBeInTheDocument()
  })

  it('clicar num atributo salva a escolha', async () => {
    const escrito: unknown[] = []
    vi.spyOn(api.characters, 'updateAbilityChoices').mockImplementation(async (_id, input) => {
      escrito.push(input)
      return { raceAttributeChoices: JSON.stringify(input.raceAttributeChoices) }
    })
    const { user } = renderDialog()
    const dialog = await abre(user)

    await user.click(within(dialog).getByRole('button', { name: /^Raça(,|$)/ }))
    await user.click(await within(dialog).findByRole('button', { name: 'FOR' }))

    await waitFor(() => expect(escrito).toHaveLength(1))
    expect(escrito[0]).toEqual({ raceAttributeChoices: { floatingPicks: ['strength'] } })
  })
})

/**
 * Uma ficha que deve escolha em UMA fonte só.
 *
 * Zerar uma fonte é tirá-la do personagem: sem raça não há +1 a distribuir,
 * sem origem não há benefício a escolher, sem classe não há poder a pegar.
 */
function soDeve(fonte: 'origem' | 'classe'): Character {
  const base = {
    races: [],
    raceAttributeChoices: '{"floatingPicks":["strength","dexterity","constitution"]}',
  }
  return fonte === 'origem'
    ? makeCharacter({ ...base, classes: [] })
    : makeCharacter({ ...base, origin: '', classes: [{ className: 'Bardo', level: 3 }] })
}
