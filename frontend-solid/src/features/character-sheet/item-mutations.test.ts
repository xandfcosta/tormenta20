import { QueryClient } from '@tanstack/solid-query'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { characterQueryOptions } from '@/entities/character/queries'
import type { Character, CharacterItem, ConsumeItemResult } from '@/shared/api/api'
import {
  ItemRefused,
  addItem,
  consumeItem,
  consumeRefusal,
  equipRefusal,
  itemActions,
  optimisticItem,
  removeItem,
  settleAddedItem,
  settleConsume,
  settleUpdatedItem,
  updateItem,
} from './item-mutations'

const CHARACTER_ID = 1

function item(overrides: Partial<CharacterItem> = {}): CharacterItem {
  return {
    id: 1,
    catalogId: null,
    name: 'Corda',
    quantity: 1,
    slots: 1,
    equipped: null,
    improvements: '[]',
    material: null,
    ...overrides,
  }
}

function character(items: CharacterItem[], overrides: Partial<Character> = {}): Character {
  return {
    id: CHARACTER_ID,
    name: 'Tanque',
    level: 10,
    hpMax: 100,
    hpCurrent: 50,
    mpMax: 20,
    mpCurrent: 10,
    items,
    activeEffects: [],
    ...overrides,
  } as Character
}

afterEach(() => vi.restoreAllMocks())

describe('optimisticItem', () => {
  // Id negativo: a linha provisória não pode colidir com um id real do servidor.
  it('nasce com id temporário negativo', () => {
    expect(optimisticItem({ quantity: 1, name: 'Corda' }, -123).id).toBe(-123)
  })

  it('serializa as melhorias como o banco guarda', () => {
    const row = optimisticItem({ quantity: 1, name: 'Espada', improvements: ['afiada'] }, -1)
    expect(row.improvements).toBe('["afiada"]')
  })
})

describe('settleAddedItem', () => {
  it('troca a linha provisória pela que o servidor persistiu', () => {
    const temp = item({ id: -5, name: '...' })
    const saved = item({ id: 42, name: 'Corda' })
    const next = settleAddedItem(addItem(character([]), temp), -5, saved)
    expect(next.items).toEqual([saved])
  })
})

describe('updateItem vs settleUpdatedItem', () => {
  // O update recebe a forma DO FIO (string[]); o settle recebe a linha
  // guardada (JSON string). Tratar um como o outro corrompe o campo.
  it('update serializa melhorias; settle preserva a linha do servidor', () => {
    const start = character([item({ id: 1, improvements: '[]' })])
    expect(updateItem(start, 1, { improvements: ['afiada'] }).items[0].improvements).toBe(
      '["afiada"]',
    )
    const fromServer = item({ id: 1, improvements: '["afiada","leve"]' })
    expect(settleUpdatedItem(start, fromServer).items[0]).toEqual(fromServer)
  })

  it('não toca nos outros itens', () => {
    const start = character([item({ id: 1 }), item({ id: 2, name: 'Tocha' })])
    expect(updateItem(start, 1, { quantity: 9 }).items[1].name).toBe('Tocha')
  })
})

describe('consumeItem', () => {
  it('decrementa quando há mais de um', () => {
    const next = consumeItem(character([item({ id: 1, quantity: 3 })]), 1)
    expect(next.items[0].quantity).toBe(2)
  })

  // No último, a linha some — ficar com quantidade 0 na mochila é lixo visual.
  it('some com a linha no último', () => {
    expect(consumeItem(character([item({ id: 1, quantity: 1 })]), 1).items).toEqual([])
  })

  it('item inexistente não muda nada', () => {
    const start = character([item({ id: 1 })])
    expect(consumeItem(start, 99)).toBe(start)
  })
})

describe('settleConsume', () => {
  it('aplica o delta do servidor: contagem, efeito e vitais', () => {
    const delta: ConsumeItemResult = {
      item: { id: 1, quantity: 2, removed: false },
      effect: {
        id: 7,
        catalogId: 'pocao',
        scope: 'scene',
        modifiers: '[]',
        createdAt: '2026-01-01T00:00:00.000Z',
      },
      hpCurrent: 80,
      mpCurrent: 12,
    }
    const next = settleConsume(character([item({ id: 1, quantity: 3 })]), delta)
    expect(next.items[0].quantity).toBe(2)
    expect(next.activeEffects.map((e) => e.id)).toEqual([7])
    expect(next).toMatchObject({ hpCurrent: 80, mpCurrent: 12 })
  })

  it('removed: true tira a linha', () => {
    const delta: ConsumeItemResult = {
      item: { id: 1, quantity: 0, removed: true },
      effect: null,
      hpCurrent: 50,
      mpCurrent: 10,
    }
    expect(settleConsume(character([item({ id: 1 })]), delta).items).toEqual([])
  })
})

/**
 * A pré-validação é o que torna o otimismo honesto: só pintamos o que o
 * servidor vai aceitar. As regras vêm do t20-data, as mesmas que o backend usa.
 */
describe('equipRefusal', () => {
  it('libera quando ainda cabe', () => {
    expect(equipRefusal(character([item({ id: 1 })]), 1, 'wielded')).toBeNull()
  })

  it('desequipar nunca é recusado', () => {
    expect(equipRefusal(character([item({ id: 1 })]), 1, null)).toBeNull()
  })

  it('recusa uma terceira mão', () => {
    const cheio = character([
      item({ id: 1, equipped: 'wielded' }),
      item({ id: 2, equipped: 'wielded2' }),
      item({ id: 3 }),
    ])
    expect(equipRefusal(cheio, 3, 'wielded')).toBeTruthy()
  })
})

describe('consumeRefusal', () => {
  it('recusa consumir o que não existe mais', () => {
    expect(consumeRefusal(item({ quantity: 0 }))).toBeTruthy()
    expect(consumeRefusal(item({ quantity: 1 }))).toBeNull()
  })
})

function seeded(items: CharacterItem[]) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  client.setQueryData(characterQueryOptions(CHARACTER_ID).queryKey, character(items))
  return client
}

const cached = (client: QueryClient) =>
  client.getQueryData<Character>(characterQueryOptions(CHARACTER_ID).queryKey)

describe('itemActions', () => {
  it('remover pinta antes da resposta e desfaz na falha', async () => {
    const client = seeded([item({ id: 1 }), item({ id: 2 })])
    const api = await import('@/shared/api/api')
    vi.spyOn(api.api.characters, 'deleteItem').mockRejectedValue(new Error('500'))

    await expect(itemActions(client, CHARACTER_ID).remove(1)).rejects.toThrow('500')

    expect(cached(client)?.items.map((i) => i.id)).toEqual([1, 2])
  })

  // A recusa acontece ANTES da rede: nada é pintado e nada é enviado.
  it('equipar acima do limite nem chega a chamar o backend', async () => {
    const client = seeded([
      item({ id: 1, equipped: 'wielded' }),
      item({ id: 2, equipped: 'wielded2' }),
      item({ id: 3 }),
    ])
    const api = await import('@/shared/api/api')
    const call = vi.spyOn(api.api.characters, 'updateItem')

    await expect(
      itemActions(client, CHARACTER_ID).change(3, { equipped: 'wielded' }),
    ).rejects.toBeInstanceOf(ItemRefused)

    expect(call).not.toHaveBeenCalled()
    expect(cached(client)?.items[2].equipped).toBeNull()
  })

  it('consumir sem unidade nem chega a chamar o backend', async () => {
    const client = seeded([item({ id: 1, quantity: 0 })])
    const api = await import('@/shared/api/api')
    const call = vi.spyOn(api.api.characters, 'consumeItem')

    await expect(
      itemActions(client, CHARACTER_ID).consume(item({ id: 1, quantity: 0 })),
    ).rejects.toBeInstanceOf(ItemRefused)

    expect(call).not.toHaveBeenCalled()
  })
})
