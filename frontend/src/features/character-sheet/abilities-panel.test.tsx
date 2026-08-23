import { QueryClient, QueryClientProvider } from '@tanstack/solid-query'
import { render, screen } from '@solidjs/testing-library'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { makeCharacter } from '@/entities/character/__fixtures__/character'
import { characterQueryOptions } from '@/entities/character/queries'
import type { Character, CharacterSpell } from '@/shared/api/api'
import { ConditionalsProvider } from '@/shared/stores/conditionals-context'
import { PowerUsesProvider } from '@/shared/stores/power-uses-context'
import { StanceActivationProvider } from '@/shared/stores/stance-activation-context'
import { AbilitiesPanel } from './abilities-panel'
import { fakeConditionals, fakePowerUses, fakeStances } from '@/shared/test/play-stores'

/**
 * O bloco Poderes depois do redesenho (ALE-217).
 *
 * O painel tinha dois MODOS e o dono resumiu o resultado em "está difícil de
 * ser usada". A medição achou a causa principal e ela é a que o primeiro teste
 * aqui protege: a tela ABRIA no modo de edição sempre que havia pendência — o
 * estado normal de quem acabou de subir de nível —, então em sessão o jogador
 * caía na tela de administração em vez da de jogo.
 *
 * O que este arquivo prova é o painel: o que ele mostra ao abrir e o que ele
 * oferece. A administração mudou de casa e tem teste próprio
 * (`choose-abilities-dialog.test.tsx`).
 */

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
  /**
   * O DEFEITO Nº 1, e o teste mais importante deste arquivo.
   *
   * O fixture deve escolhas — é um personagem recém-criado, como todo mundo que
   * acabou de subir de nível. Antes, isso bastava para a tela abrir na
   * administração. A pendência continua sendo anunciada, mas por um CRACHÁ no
   * botão: ela chama, não sequestra.
   */
  it('abre na lista da mesa mesmo devendo escolhas', () => {
    renderPanel()

    expect(screen.getByText('Ações')).toBeInTheDocument()
    expect(screen.queryByText('Faltam escolhas')).not.toBeInTheDocument()
    // O crachá diz o que falta sem tomar a tela.
    expect(
      screen.getByRole('button', { name: /^Escolher poderes, \d+ escolhas? pendentes?$/ }),
    ).toBeInTheDocument()
  })

  // Sem nada a escolher o botão continua lá, sem crachá: escolher poder é uma
  // ação que existe sempre, não um aviso.
  it('sem pendência, o botão fica e o crachá some', () => {
    renderPanel(semPendencias())

    const botao = screen.getByRole('button', { name: 'Escolher poderes' })
    expect(botao).toBeInTheDocument()
    expect(botao).not.toHaveTextContent(/\d/)
  })

  it('a busca por nome atravessa as fontes', async () => {
    const { user } = renderPanel()

    await user.type(screen.getByRole('textbox', { name: 'Buscar poder ou habilidade' }), 'zzzz')

    expect(await screen.findByText(/Nenhum poder para "zzzz"/)).toBeInTheDocument()
  })

  it('a lista da mesa traz as ações com o afordance de usar', () => {
    renderPanel(makeCharacter({ classes: [{ className: 'Bárbaro', level: 6 }] }))

    expect(screen.getByText('Ações')).toBeInTheDocument()
    // Fúria é postura: entra com o chip de custo e o botão de ativar.
    expect(screen.getByRole('button', { name: 'Ativar Fúria' })).toBeInTheDocument()
  })
})

/**
 * A CLASSE SEM AÇÕES (ALE-217, decisão do dono).
 *
 * Medido: um Arcanista de nível 20 tem 26 habilidades e ZERO ações ativáveis.
 * Sumir com a seção faria a tela mudar de forma por classe, e duas pessoas na
 * mesma mesa veriam layouts diferentes — então ela fica e ENSINA onde está o
 * que o jogador procura.
 *
 * As duas metades importam. A frase só aponta para Magias a quem CONJURA:
 * mandar um guerreiro sem magia para uma aba vazia é trocar um beco sem saída
 * por outro.
 */
describe('AbilitiesPanel — quando a classe não tem ação nenhuma', () => {
  it('o conjurador é mandado para a aba Magias', () => {
    renderPanel(semAcoes({ spells: [magia()] }))

    expect(screen.getByText('Ações')).toBeInTheDocument()
    expect(
      screen.getByText(/Nenhuma ação ativável — suas magias estão na aba Magias\./),
    ).toBeInTheDocument()
  })

  it('quem não conjura não é mandado para lugar nenhum', () => {
    renderPanel(semAcoes({ spells: [] }))

    expect(screen.getByText(/Nenhuma ação ativável\. Suas habilidades são passivas\./)).toBeInTheDocument()
    expect(screen.queryByText(/aba Magias/)).not.toBeInTheDocument()
  })
})

/** Arcanista: 26 habilidades no nível 20, nenhuma delas ativável. */
function semAcoes(over: Partial<Character>): Character {
  return makeCharacter({ classes: [{ className: 'Arcanista', level: 6 }], ...over })
}

/** Um personagem com as escolhas do nascimento já feitas. */
function semPendencias(): Character {
  return makeCharacter({
    origin: '',
    races: [],
    classes: [],
    raceAttributeChoices: '{"floatingPicks":["strength","dexterity","constitution"]}',
  })
}

function magia(): CharacterSpell {
  return {
    id: 1,
    catalogSpellId: 'raio-de-fogo',
    prepared: true,
    learnedAt: '2026-01-01T00:00:00.000Z',
  }
}
