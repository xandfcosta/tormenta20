import { QueryClient } from '@tanstack/solid-query'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { characterQueryOptions } from '@/entities/character/queries'
import type { ActiveEffect, Character } from '@/shared/api/api'
import { conditionActions, dropClearedScopes, effectActions } from './effect-mutations'

const CHARACTER_ID = 1

function effect(id: number, scope: 'scene' | 'day', catalogId = 'pocao'): ActiveEffect {
  return { id, catalogId, scope, modifiers: '[]', createdAt: '2026-08-11' }
}

function character(overrides: Partial<Character> = {}): Character {
  return {
    id: CHARACTER_ID,
    name: 'Bruxo',
    activeConditions: '[]',
    activeEffects: [],
    classes: [{ className: 'Bardo', level: 3 }],
    items: [],
    expertises: [],
    spells: [],
    ...overrides,
  } as Character
}

function seeded(char: Character = character()) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  client.setQueryData(characterQueryOptions(CHARACTER_ID).queryKey, char)
  return client
}

const cached = (client: QueryClient) =>
  client.getQueryData<Character>(characterQueryOptions(CHARACTER_ID).queryKey)

afterEach(() => vi.restoreAllMocks())

describe('dropClearedScopes', () => {
  it('tira só os efeitos dos escopos que o servidor limpou', () => {
    const char = character({ activeEffects: [effect(1, 'scene'), effect(2, 'day')] })

    const next = dropClearedScopes(char, ['scene'])

    expect(next.activeEffects.map((e) => e.id)).toEqual([2])
  })

  // Encerrar o dia limpa os dois escopos — o servidor diz quais, não o botão.
  it('encerrar dia leva cena e dia', () => {
    const char = character({ activeEffects: [effect(1, 'scene'), effect(2, 'day')] })

    expect(dropClearedScopes(char, ['scene', 'day']).activeEffects).toEqual([])
  })

  it('escopo nenhum limpo devolve a lista intacta', () => {
    const char = character({ activeEffects: [effect(1, 'scene')] })

    expect(dropClearedScopes(char, []).activeEffects).toHaveLength(1)
  })
})

describe('effectActions.remove', () => {
  it('some com a linha depois que o servidor confirma o id', async () => {
    const client = seeded(character({ activeEffects: [effect(7, 'scene'), effect(8, 'day')] }))
    const api = await import('@/shared/api/api')
    vi.spyOn(api.api.characters, 'removeActiveEffect').mockResolvedValue({ id: 7 })

    await effectActions(client, CHARACTER_ID).remove(7)

    expect(cached(client)?.activeEffects.map((e) => e.id)).toEqual([8])
  })

  // Sem otimismo aqui de propósito: o efeito só sai quando o servidor confirma,
  // porque um id que não existe mais é 404 e a linha teria de voltar.
  it('falha mantém o efeito na ficha', async () => {
    const client = seeded(character({ activeEffects: [effect(7, 'scene')] }))
    const api = await import('@/shared/api/api')
    vi.spyOn(api.api.characters, 'removeActiveEffect').mockRejectedValue(new Error('404'))

    await expect(effectActions(client, CHARACTER_ID).remove(7)).rejects.toThrow('404')

    expect(cached(client)?.activeEffects).toHaveLength(1)
  })
})

describe('effectActions.endScene / endDay', () => {
  it('encerrar cena derruba os efeitos de cena no cache', async () => {
    const client = seeded(character({ activeEffects: [effect(1, 'scene'), effect(2, 'day')] }))
    const api = await import('@/shared/api/api')
    vi.spyOn(api.api.characters, 'endScene').mockResolvedValue({ clearedScopes: ['scene'] })

    await effectActions(client, CHARACTER_ID).endScene()

    expect(cached(client)?.activeEffects.map((e) => e.id)).toEqual([2])
  })

  it('encerrar dia derruba os dois escopos', async () => {
    const client = seeded(character({ activeEffects: [effect(1, 'scene'), effect(2, 'day')] }))
    const api = await import('@/shared/api/api')
    vi.spyOn(api.api.characters, 'endDay').mockResolvedValue({ clearedScopes: ['scene', 'day'] })

    await effectActions(client, CHARACTER_ID).endDay()

    expect(cached(client)?.activeEffects).toEqual([])
  })
})

describe('conditionActions.set', () => {
  it('pinta a condição antes da resposta e assenta pelo blob do servidor', async () => {
    const client = seeded()
    const api = await import('@/shared/api/api')
    vi.spyOn(api.api.characters, 'updateConditions').mockResolvedValue({
      activeConditions: '["cego"]',
    })

    await conditionActions(client, CHARACTER_ID).set(['cego'])

    expect(cached(client)?.activeConditions).toBe('["cego"]')
  })

  it('falha devolve as condições anteriores', async () => {
    const client = seeded(character({ activeConditions: '["caido"]' }))
    const api = await import('@/shared/api/api')
    vi.spyOn(api.api.characters, 'updateConditions').mockRejectedValue(new Error('500'))

    await expect(conditionActions(client, CHARACTER_ID).set([])).rejects.toThrow('500')

    expect(cached(client)?.activeConditions).toBe('["caido"]')
  })
})
