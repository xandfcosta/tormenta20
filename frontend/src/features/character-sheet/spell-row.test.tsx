import { FakeStorage } from '@/shared/test/fake-storage'
import { QueryClient, QueryClientProvider } from '@tanstack/solid-query'
import { render, screen, waitFor, within } from '@solidjs/testing-library'
import userEvent from '@testing-library/user-event'
import type { AttributeKey } from '@/shared/api/attribute-keys'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { makeCharacter } from '@/entities/character/__fixtures__/character'
import { characterQueryOptions } from '@/entities/character/queries'
import type { Character, CharacterSpell } from '@/shared/api/api'
import { spellCatalog } from '@/shared/lib/spell-cache'
import { ConditionalsProvider } from '@/shared/stores/conditionals-context'
import { createConditionalsStore } from '@/shared/stores/conditionals-store'
import { SpellRow } from './spell-row'

/**
 * SpellRow gets its own file because in the panel it lives inside a VIRTUALIZED
 * list, and jsdom measures every element as 0 — no row ever renders there
 * (migration trap: the panel test would pass green against an empty list).
 */


const CD: Record<AttributeKey, number> = {
  strength: 10,
  dexterity: 11,
  constitution: 12,
  intelligence: 18,
  wisdom: 15,
  charisma: 14,
}

function firstCircleSpell() {
  const spell = Object.values(spellCatalog()).find(
    (s) => s.circle === 1 && s.classes.includes('Arcanista'),
  )
  if (!spell) throw new Error('catálogo sem magia de Arcanista de 1º círculo')
  return spell
}

/** A 5th-circle spell an Arcanista 5 cannot reach yet. */
function highCircleSpell() {
  const spell = Object.values(spellCatalog()).find((s) => s.circle === 5)
  if (!spell) throw new Error('catálogo sem magia de 5º círculo')
  return spell
}

function renderRow(
  options: {
    learned?: CharacterSpell | null
    character?: Character
    spell?: ReturnType<typeof firstCircleSpell>
  } = {},
) {
  // A learned row only means "learned" when the character carries it too — the
  // shared guard reads `character.spells`, not this prop.
  const character =
    options.character ??
    makeCharacter({
      classes: [{ className: 'Arcanista', level: 5 }],
      spells: options.learned ? [options.learned] : [],
    })
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  client.setQueryData(characterQueryOptions(character.id).queryKey, character)
  render(() => (
    <QueryClientProvider client={client}>
      <ConditionalsProvider store={createConditionalsStore(new FakeStorage())}>
        <SpellRow
          spell={options.spell ?? firstCircleSpell()}
          character={character}
          learned={options.learned ?? null}
          spellCdByAttribute={CD}
        />
      </ConditionalsProvider>
    </QueryClientProvider>
  ))
  return { user: userEvent.setup() }
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

describe('SpellRow', () => {
  it('mostra a CD da classe que lança a magia', () => {
    renderRow()
    // Arcanista lança por Inteligência.
    expect(screen.getByText(`CD ${CD.intelligence}`)).toBeInTheDocument()
  })

  it('magia não aprendida não oferece Conjurar', () => {
    renderRow()
    expect(screen.queryByRole('button', { name: /^Conjurar/ })).not.toBeInTheDocument()
  })

  it('aprendida oferece Conjurar sem precisar expandir a linha', () => {
    renderRow({ learned: { id: 1, catalogSpellId: firstCircleSpell().id, prepared: false, learnedAt: '' } })

    expect(screen.getByRole('button', { name: /^Conjurar/ })).toBeInTheDocument()
  })

  // Círculo acima do alcançável: a linha aparece, mas conjurar fica travado.
  it('círculo alto demais desabilita o Conjurar', () => {
    const spell = highCircleSpell()
    renderRow({
      spell,
      learned: { id: 1, catalogSpellId: spell.id, prepared: false, learnedAt: '' },
    })

    expect(screen.getByRole('button', { name: `Conjurar ${spell.name}` })).toBeDisabled()
  })

  it('aprender chama o endpoint com a magia da linha', async () => {
    const spell = firstCircleSpell()
    const api = await import('@/shared/api/api')
    const learn = vi.spyOn(api.api.characters, 'learnSpell').mockResolvedValue({
      id: 1,
      catalogSpellId: spell.id,
      prepared: false,
      learnedAt: '',
    })
    const { user } = renderRow()

    await user.click(screen.getByRole('button', { expanded: false }))
    await user.click(await screen.findByRole('button', { name: 'Aprender' }))

    await waitFor(() => expect(learn).toHaveBeenCalledWith(1, spell.id))
  })

  it('preparar e despreparar trocam de rótulo', async () => {
    const spell = firstCircleSpell()
    const api = await import('@/shared/api/api')
    const setPrepared = vi
      .spyOn(api.api.characters, 'setSpellPrepared')
      .mockResolvedValue({ id: 1, catalogSpellId: spell.id, prepared: true, learnedAt: '' })
    const { user } = renderRow({
      learned: { id: 1, catalogSpellId: spell.id, prepared: false, learnedAt: '' },
    })

    await user.click(screen.getByRole('button', { expanded: false }))
    await user.click(await screen.findByRole('button', { name: 'Preparar' }))

    await waitFor(() => expect(setPrepared).toHaveBeenCalledWith(1, spell.id, true))
  })

  /**
   * Esquecer PERGUNTA antes (ALE-200).
   *
   * Ele é o único caminho que tira a magia da ficha, e ficava encostado no
   * "Despreparar", que alterna um estado reversível num clique. A ALE-200
   * separou os dois pela cor; a pergunta é a rede embaixo — um destrutivo de um
   * clique só é o mesmo defeito que a ALE-122 consertou afastando o "Reiniciar"
   * do "Próximo turno".
   *
   * São DOIS testes e não um com as duas metades, e o motivo é o overlay: o
   * Kobalte marca tudo atrás do modal como `aria-hidden` e não desfaz isso ao
   * fechar dentro do jsdom, então voltar a olhar a linha depois de cancelar
   * encontra uma árvore inteira fora do alcance. Montagem limpa em cada um.
   */
  it('esquecer pergunta antes de tirar a magia', async () => {
    const spell = firstCircleSpell()
    const api = await import('@/shared/api/api')
    const unlearn = vi
      .spyOn(api.api.characters, 'unlearnSpell')
      .mockResolvedValue({ catalogSpellId: spell.id, removed: 1 })
    const { user } = renderRow({
      learned: { id: 1, catalogSpellId: spell.id, prepared: false, learnedAt: '' },
    })

    await user.click(screen.getByRole('button', { expanded: false }))
    await user.click(await screen.findByRole('button', { name: 'Esquecer' }))

    const pergunta = await screen.findByRole('dialog')
    expect(within(pergunta).getByText(new RegExp(spell.name))).toBeInTheDocument()
    // A pergunta ABRIU e o endpoint ainda não foi chamado: é o clique que
    // deixou de ser destrutivo, não só a tela que ganhou um aviso.
    expect(unlearn, 'o clique tirou a magia antes de perguntar').not.toHaveBeenCalled()
  })

  it('confirmar a pergunta é o que tira a magia', async () => {
    const spell = firstCircleSpell()
    const api = await import('@/shared/api/api')
    const unlearn = vi
      .spyOn(api.api.characters, 'unlearnSpell')
      .mockResolvedValue({ catalogSpellId: spell.id, removed: 1 })
    const { user } = renderRow({
      learned: { id: 1, catalogSpellId: spell.id, prepared: false, learnedAt: '' },
    })

    await user.click(screen.getByRole('button', { expanded: false }))
    await user.click(await screen.findByRole('button', { name: 'Esquecer' }))
    const pergunta = await screen.findByRole('dialog')
    await user.click(within(pergunta).getByRole('button', { name: 'Esquecer' }))

    await waitFor(() => expect(unlearn).toHaveBeenCalledWith(1, spell.id))
  })

  // Magia concedida por poder não tem linha de grimório pra gerenciar.
  it('magia concedida não oferece Esquecer', async () => {
    const spell = firstCircleSpell()
    const client = new QueryClient()
    const character = makeCharacter({ classes: [{ className: 'Bárbaro', level: 5 }] })
    client.setQueryData(characterQueryOptions(character.id).queryKey, character)
    render(() => (
      <QueryClientProvider client={client}>
        <ConditionalsProvider store={createConditionalsStore(new FakeStorage())}>
          <SpellRow
            spell={spell}
            character={character}
            learned={null}
            spellCdByAttribute={CD}
            granted={{ sourcePower: 'Totem Espiritual', keyAttribute: 'wisdom' }}
          />
        </ConditionalsProvider>
      </QueryClientProvider>
    ))
    const user = userEvent.setup()

    expect(screen.getByText('Totem Espiritual')).toBeInTheDocument()
    // Concedida é conjurável mesmo para não-conjurador.
    expect(screen.getByRole('button', { name: /^Conjurar/ })).toBeEnabled()

    await user.click(screen.getByRole('button', { expanded: false }))
    expect(screen.queryByRole('button', { name: 'Esquecer' })).not.toBeInTheDocument()
  })
})
