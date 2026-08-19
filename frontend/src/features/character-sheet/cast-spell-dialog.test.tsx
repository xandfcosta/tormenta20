import { render, screen, waitFor, within } from '@solidjs/testing-library'
import { QueryClient, QueryClientProvider } from '@tanstack/solid-query'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { makeCharacter } from '@/entities/character/__fixtures__/character'
import { characterQueryOptions } from '@/entities/character/queries'
import { ApiError, api, type Character } from '@/shared/api/api'
import type { CatalogSpell } from '@/shared/api/catalog-types'
import { spellCatalog } from '@/shared/lib/spell-cache'
import { CastSpellDialog } from './cast-spell-dialog'

/**
 * CONJURAR, do clique ao corpo da requisição (ALE-186, bloco 4).
 *
 * O fluxo mais rico da ficha tinha só os helpers puros provados: o cálculo de
 * PM, os picks e o limite. A COMPOSIÇÃO — abrir, escolher degraus, ver o preço
 * mudar, mandar — nunca tinha sido montada, e é onde os defeitos deste app
 * moram. Um `augmentIndex` trocado pelo `stacks` passaria em todos os testes
 * de helper.
 *
 * Um caso aqui é armadilha documentada: o erro do servidor tem de aparecer
 * INLINE. Um toast disparado de dentro de um modal NÃO é anunciado — Kobalte
 * marca os irmãos do modal como `aria-hidden`, e a região do sonner é irmã.
 */

/** Uma magia REAL do catálogo com um aprimoramento que soma degraus. */
function magiaComDegrau(): CatalogSpell {
  const spell = spellCatalog()['sono']
  if (!spell) throw new Error('catálogo sem a magia "sono"')
  return spell
}

/** "Enfeitiçar" tem um aprimoramento que exige 3º círculo — o do teste da trava. */
function magiaComAprimoramentoDeCirculoAlto(): CatalogSpell {
  const spell = spellCatalog()['enfeiticar']
  if (!spell) throw new Error('catálogo sem a magia "enfeitiçar"')
  return spell
}

function arcanista(level: number, overrides: Partial<Character> = {}): Character {
  return makeCharacter({
    level,
    classes: [{ className: 'Arcanista', level }],
    intelligence: 4,
    mpMax: 40,
    mpCurrent: 40,
    ...overrides,
  })
}

function renderDialog(spell: CatalogSpell, character: Character) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  client.setQueryData(characterQueryOptions(character.id).queryKey, character)
  render(() => (
    <QueryClientProvider client={client}>
      <CastSpellDialog spell={spell} character={character} />
    </QueryClientProvider>
  ))
  return userEvent.setup()
}

/** Abre o diálogo e devolve o `user` — todo caso começa por aqui. */
async function abrirConjurar(spell: CatalogSpell, character: Character) {
  const user = renderDialog(spell, character)
  await user.click(screen.getByRole('button', { name: `Conjurar ${spell.name}` }))
  await screen.findByRole('dialog')
  return user
}

/** Um degrau pelo CAMPO, que é o que o teclado alcança: os botões do spinner
 *  têm `tabIndex={-1}` e nome acessível repetido entre aprimoramentos. */
async function somarUmDegrau(user: ReturnType<typeof userEvent.setup>) {
  const campo = screen.getByRole('spinbutton', { name: 'Aprimoramento 1 — degraus' })
  await user.clear(campo)
  await user.type(campo, '1')
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

describe('CastSpellDialog', () => {
  it('somar um degrau soma o PM do aprimoramento ao custo mostrado', async () => {
    const spell = magiaComDegrau()
    const user = await abrirConjurar(spell, arcanista(5))
    const degrau = spell.augments[0]
    expect(screen.getByText('1 PM')).toBeInTheDocument()

    await somarUmDegrau(user)

    // O preço na tela é o que a mesa lê antes de decidir: base + o degrau.
    expect(await screen.findByText(`${1 + degrau.pmCost} PM`)).toBeInTheDocument()
  })

  it('manda o índice do aprimoramento e quantos degraus', async () => {
    const cast = vi
      .spyOn(api.characters, 'castSpell')
      .mockResolvedValue({ mpCurrent: 37, removedEffectIds: [] })
    const spell = magiaComDegrau()
    const user = await abrirConjurar(spell, arcanista(5))

    await somarUmDegrau(user)
    await user.click(screen.getByRole('button', { name: /^Conjurar$/ }))

    // O corpo é a única coisa que o servidor vê: índice e degraus, não o preço
    // — quem recalcula o custo é ele.
    await waitFor(() =>
      expect(cast).toHaveBeenCalledWith(1, spell.id, [{ augmentIndex: 0, stacks: 1 }]),
    )
  })

  it('aprimoramento acima do círculo alcançável não é escolhível', async () => {
    const spell = magiaComAprimoramentoDeCirculoAlto()
    const travado = spell.augments.findIndex((a) => a.requiresCircle === 3)
    // Arcanista 1 chega ao 1º círculo: o aprimoramento de 3º está fora do
    // alcance dela, e a tela não pode oferecer o que o servidor vai recusar.
    await abrirConjurar(spell, arcanista(1))

    const caixa = screen.getByRole('checkbox', {
      name: new RegExp(spell.augments[travado].description.slice(0, 20)),
    })

    expect(caixa).toBeDisabled()
    expect(screen.getByText(/requer 3º círculo/)).toBeInTheDocument()
  })

  it('a recusa do servidor aparece DENTRO do diálogo', async () => {
    vi.spyOn(api.characters, 'castSpell').mockRejectedValue(
      new ApiError(400, 'PM insuficiente para conjurar'),
    )
    const spell = magiaComDegrau()
    const user = await abrirConjurar(spell, arcanista(5))

    await user.click(screen.getByRole('button', { name: /^Conjurar$/ }))

    // Dentro do diálogo, e o diálogo CONTINUA aberto: um toast aqui não seria
    // anunciado, e fechar levaria embora a escolha de degraus que a pessoa fez.
    const dialog = await screen.findByRole('dialog')
    expect(await within(dialog).findByText(/PM insuficiente/)).toBeVisible()
  })
})
