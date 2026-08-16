import { QueryClient } from '@tanstack/solid-query'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { characterQueryOptions } from '@/entities/character/queries'
import type { Character } from '@/shared/api/api'
import {
  classDefaults,
  groupProficiencies,
  ownedProficiencies,
  proficiencyActions,
  proficiencyCatalog,
  toggleProficiency,
} from './proficiency-mutations'

const CHARACTER_ID = 1

function character(overrides: Partial<Character> = {}): Character {
  return {
    id: CHARACTER_ID,
    name: 'Tanque',
    proficiencies: '[]',
    classes: [{ className: 'Guerreiro', level: 3 }],
    items: [],
    activeEffects: [],
    expertises: [],
    spells: [],
    ...overrides,
  } as Character
}

afterEach(() => vi.restoreAllMocks())

describe('ownedProficiencies', () => {
  it('lê o blob guardado', () => {
    const owned = ownedProficiencies(character({ proficiencies: '["armas-simples"]' }))
    expect(owned.has('armas-simples')).toBe(true)
  })

  // Blob corrompido não pode derrubar a ficha que o jogador está tentando ler.
  it('blob ilegível vira conjunto vazio', () => {
    expect(ownedProficiencies(character({ proficiencies: '{quebrado' })).size).toBe(0)
    expect(ownedProficiencies(character({ proficiencies: '"nem array"' })).size).toBe(0)
  })
})

describe('proficiencyCatalog / groupProficiencies', () => {
  it('o Guerreiro traz armas e armaduras do livro', () => {
    const entries = proficiencyCatalog(character())
    const { weapons, armors } = groupProficiencies(entries)
    expect(weapons.length).toBeGreaterThan(0)
    expect(armors.some((e) => e.category === 'escudos')).toBe(true)
    // Todo id cai em um dos dois grupos — nenhum some da tela.
    expect(weapons.length + armors.length).toBe(entries.length)
  })
})

describe('toggleProficiency', () => {
  it('liga o que falta e desliga o que tem', () => {
    expect(toggleProficiency(new Set(), 'escudos')).toEqual(['escudos'])
    expect(toggleProficiency(new Set(['escudos', 'armas-simples']), 'escudos')).toEqual([
      'armas-simples',
    ])
  })
})

describe('classDefaults', () => {
  it('devolve só o que a classe concede', () => {
    const defaults = classDefaults(character())
    const granted = proficiencyCatalog(character()).filter((e) => e.granted)
    expect(defaults).toEqual(granted.map((e) => e.category))
  })
})

function seeded(char: Character = character()) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  client.setQueryData(characterQueryOptions(CHARACTER_ID).queryKey, char)
  return client
}

const cached = (client: QueryClient) =>
  client.getQueryData<Character>(characterQueryOptions(CHARACTER_ID).queryKey)

describe('proficiencyActions', () => {
  // O servidor responde DIFERENTE do palpite de propósito: com a resposta igual
  // ao que se pintou, o teste não distinguia a pintura do que o servidor mandou
  // e passaria verde mesmo se a resposta fosse descartada.
  it('pinta antes da resposta e assenta pelo blob do SERVIDOR', async () => {
    const client = seeded()
    const api = await import('@/shared/api/api')
    let responder = (): void => {}
    const resposta = new Promise<{ proficiencies: string }>((resolve) => {
      responder = () => resolve({ proficiencies: '["armas-marciais","escudos"]' })
    })
    vi.spyOn(api.api.characters, 'updateProficiencies').mockReturnValue(resposta)

    const emVoo = proficiencyActions(client, CHARACTER_ID).set(['armas-marciais'])
    await new Promise((resolve) => setTimeout(resolve, 0))
    expect(cached(client)?.proficiencies).toBe('["armas-marciais"]')

    responder()
    await emVoo
    expect(cached(client)?.proficiencies).toBe('["armas-marciais","escudos"]')
  })

  it('falha devolve o blob anterior', async () => {
    const client = seeded(character({ proficiencies: '["escudos"]' }))
    const api = await import('@/shared/api/api')
    vi.spyOn(api.api.characters, 'updateProficiencies').mockRejectedValue(new Error('500'))

    await expect(proficiencyActions(client, CHARACTER_ID).set([])).rejects.toThrow('500')

    expect(cached(client)?.proficiencies).toBe('["escudos"]')
  })
})
