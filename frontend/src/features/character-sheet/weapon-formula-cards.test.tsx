import { render, screen } from '@solidjs/testing-library'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { makeCharacter } from '@/entities/character/__fixtures__/character'
import type { Character, CharacterItem } from '@/shared/api/api'
import { WeaponFormulaCards } from './weapon-formula-cards'

/**
 * A FÓRMULA PRONTA DE ATAQUE (ALE-197, grupo C).
 *
 * Os números vêm do motor Go pelo WASM — a suíte roda o MESMO motor da
 * produção —, então o que se prova aqui não é a conta e sim a LEITURA: a tira
 * que a pessoa olha no meio do turno para não trocar de bloco atrás do dado de
 * dano, e o detalhamento que diz de onde cada número veio.
 *
 * O `critLabel` e o `weaponCardRows` não tinham teste em camada nenhuma. Eles
 * ganham rede AQUI, na tela, e não em teste de função: o valor deles é o texto
 * que aparece no card.
 */

function arma(overrides: Partial<CharacterItem>): CharacterItem {
  return {
    id: 1,
    catalogId: 'espada-longa',
    name: 'Espada longa',
    quantity: 1,
    slots: 1,
    equipped: 'wielded',
    improvements: '[]',
    material: null,
    ...overrides,
  }
}

function renderCards(character: Character) {
  render(() => <WeaponFormulaCards character={character} activeConditionals={new Set()} />)
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

describe('WeaponFormulaCards', () => {
  it('de mãos vazias, o lugar não fica quebrado', () => {
    renderCards(makeCharacter({ items: [] }))

    // O espaço vazio de um bloco que existe lê como bug; a frase diz que está
    // certo assim.
    expect(screen.getByText('Nenhuma arma empunhada.')).toBeInTheDocument()
  })

  it('a arma na mão vira ataque, dano e crítico numa linha só', () => {
    renderCards(makeCharacter({ strength: 4, items: [arma({})] }))

    expect(screen.getByLabelText('Detalhamento de Espada longa')).toBeInTheDocument()
    // 19-20/x2 é o crítico da espada longa no livro; o dado e os bônus vêm do
    // motor Go, o mesmo que o servidor usa.
    expect(screen.getByText('19-20/x2')).toBeInTheDocument()
  })

  it('crítico 20 é escrito "20", nunca "20-20"', () => {
    renderCards(
      makeCharacter({
        items: [arma({ catalogId: 'machado-batalha', name: 'Machado de batalha' })],
      }),
    )

    // A exceção de escrita do `critLabel`, que é a armadilha da função — e a
    // única maneira de ver que ela vale é lendo o card.
    expect(screen.getByText('20/x3')).toBeInTheDocument()
    expect(screen.queryByText('20-20/x3')).not.toBeInTheDocument()
  })

  it('abrir o card mostra DE ONDE vem o ataque', async () => {
    const user = renderCards(makeCharacter({ strength: 4, items: [arma({})] }))

    await user.click(screen.getByLabelText('Detalhamento de Espada longa'))

    // A tira dá o número; o detalhamento dá a conta. Sem ele, o jogador que
    // discorda do total não tem como conferir com o mestre.
    const dialog = await screen.findByRole('dialog')
    expect(dialog).toHaveTextContent(/Ataque \(Luta\)/)
    expect(dialog).toHaveTextContent(/FOR/)
  })

  it('duas armas empunhadas viram dois cards', () => {
    renderCards(
      makeCharacter({
        items: [arma({}), arma({ id: 2, catalogId: 'adaga', name: 'Adaga' })],
      }),
    )

    expect(screen.getByLabelText('Detalhamento de Espada longa')).toBeInTheDocument()
    expect(screen.getByLabelText('Detalhamento de Adaga')).toBeInTheDocument()
  })
})
