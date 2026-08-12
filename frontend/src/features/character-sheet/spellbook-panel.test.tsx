import { QueryClient, QueryClientProvider } from '@tanstack/solid-query'
import { render, screen } from '@solidjs/testing-library'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { makeCharacter } from '@/entities/character/__fixtures__/character'
import { characterQueryOptions } from '@/entities/character/queries'
import type { Character, CharacterSpell } from '@/shared/api/api'
import { spellCatalog } from '@/shared/lib/spell-cache'
import { ConditionalsProvider } from '@/shared/stores/conditionals-context'
import { createConditionalsStore } from '@/shared/stores/conditionals-store'
import { SpellbookPanel } from './spellbook-panel'

/** In-memory Storage double, so no test reaches a real localStorage. */
class FakeStorage implements Storage {
  private entries = new Map<string, string>()
  get length() {
    return this.entries.size
  }
  clear() {
    this.entries.clear()
  }
  getItem(key: string) {
    return this.entries.get(key) ?? null
  }
  key(index: number) {
    return [...this.entries.keys()][index] ?? null
  }
  removeItem(key: string) {
    this.entries.delete(key)
  }
  setItem(key: string, value: string) {
    this.entries.set(key, value)
  }
}

/** A real 1st-circle Arcanista spell out of the primed catalog. */
function anArcanistaSpell() {
  const spell = Object.values(spellCatalog()).find(
    (s) => s.circle === 1 && s.classes.includes('Arcanista'),
  )
  if (!spell) throw new Error('catálogo sem magia de Arcanista de 1º círculo')
  return spell
}

const arcanista = (spells: CharacterSpell[] = []) =>
  makeCharacter({ classes: [{ className: 'Arcanista', level: 5 }], spells })

function renderPanel(char: Character) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  client.setQueryData(characterQueryOptions(char.id).queryKey, char)
  render(() => (
    <QueryClientProvider client={client}>
      <ConditionalsProvider store={createConditionalsStore(new FakeStorage())}>
        <SpellbookPanel character={char} />
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

describe('SpellbookPanel', () => {
  it('grimório vazio ensina como enchê-lo', () => {
    renderPanel(arcanista())

    expect(screen.getByText(/Nenhuma magia aprendida/)).toBeInTheDocument()
    expect(screen.getByText('0 aprendidas')).toBeInTheDocument()
  })

  // Sem classe conjuradora não há o que aprender — nem botão.
  it('não-conjurador não vê "Aprender"', () => {
    renderPanel(makeCharacter({ classes: [{ className: 'Guerreiro', level: 5 }] }))

    expect(screen.queryByRole('button', { name: 'Aprender' })).not.toBeInTheDocument()
    expect(screen.getByText(/não tem classe conjuradora/)).toBeInTheDocument()
  })

  it('conta as magias aprendidas', () => {
    const spell = anArcanistaSpell()
    renderPanel(
      arcanista([{ id: 1, catalogSpellId: spell.id, prepared: false, learnedAt: '' }]),
    )

    expect(screen.getByText('1 aprendida')).toBeInTheDocument()
  })

  it('o diálogo de aprender filtra o catálogo pelo nome', async () => {
    const spell = anArcanistaSpell()
    const { user } = renderPanel(arcanista())

    await user.click(screen.getByRole('button', { name: 'Aprender' }))
    await user.type(await screen.findByRole('searchbox', { name: 'Buscar magia' }), 'zzzz')

    expect(await screen.findByText(/Nenhuma magia para "zzzz"/)).toBeInTheDocument()
    expect(screen.queryByText(spell.name)).not.toBeInTheDocument()
  })
})
