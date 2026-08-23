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
  // A resposta do servidor é um ECO, e o teste que estava aqui não sabia disso.
  //
  // Ele afirmava "assenta pelo blob do SERVIDOR" e simulava uma resposta
  // DIVERGENTE do palpite — "como se o mestre tivesse aplicado atordoado no
  // mesmo instante". O servidor não pode fazer isso: `character_conditions.go`
  // devolve `marshalStrings(&body.ActiveConditions)`, que é o array que o
  // cliente mandou. O teste passava porque o mock mentia sobre o servidor, e
  // ele documentava um contrato que não existe (ALE-243).
  it('pinta a condição ANTES da resposta', async () => {
    const client = seeded()
    const api = await import('@/shared/api/api')
    let responder = (): void => {}
    const resposta = new Promise<{ activeConditions: string }>((resolve) => {
      responder = () => resolve({ activeConditions: '["cego"]' })
    })
    vi.spyOn(api.api.characters, 'updateConditions').mockReturnValue(resposta)

    const emVoo = conditionActions(client, CHARACTER_ID).set(['cego'])
    // Um tick: a pintura acontece depois do `cancelQueries`, que é await.
    await new Promise((resolve) => setTimeout(resolve, 0))
    expect(cached(client)?.activeConditions).toBe('["cego"]')

    responder()
    await emVoo
    expect(cached(client)?.activeConditions).toBe('["cego"]')
  })

  /**
   * TRÊS CLIQUES RÁPIDOS NÃO PERDEM A DO MEIO (ALE-243).
   *
   * O defeito: aplicar era fire-and-forget, o payload saía do CACHE, e a
   * resposta de cada escrita era assentada no cache assim que chegava. Com as
   * respostas atrasadas, a do PRIMEIRO clique pousava depois da pintura do
   * segundo e devolvia o cache a um item — e o terceiro clique montava o
   * pedido lendo esse cache, mandando dois.
   *
   * O teste segura as três respostas e as solta FORA DE ORDEM, que é o pior
   * caso. Não precisa de browser: a corrida é de promessas, e o jsdom as
   * ordena igual.
   *
   * A asserção que morde é a ÚLTIMA: o que foi para o fio na terceira escrita.
   * Afirmar só o cache final deixaria passar um conserto que arruma a tela e
   * continua gravando errado no banco.
   */
  it('com as respostas atrasadas e fora de ordem, nenhuma condição some', async () => {
    const client = seeded()
    const api = await import('@/shared/api/api')

    const soltar: (() => void)[] = []
    const enviados: string[][] = []
    vi.spyOn(api.api.characters, 'updateConditions').mockImplementation(
      (_id: number, conditions: string[]) => {
        enviados.push([...conditions])
        return new Promise<{ activeConditions: string }>((resolve) => {
          soltar.push(() => resolve({ activeConditions: JSON.stringify(conditions) }))
        })
      },
    )

    // O gesto real: quem clica não espera a resposta para clicar de novo, e o
    // conjunto de cada clique é lido do cache na hora.
    const daCache = () => JSON.parse(cached(client)?.activeConditions ?? '[]') as string[]
    const acoes = conditionActions(client, CHARACTER_ID)
    const emVoo: Promise<void>[] = []
    for (const nova of ['abalado', 'agarrado', 'cego']) {
      emVoo.push(acoes.set([...daCache(), nova] as never))
      await new Promise((resolve) => setTimeout(resolve, 0))
    }

    // Fora de ordem, e a primeira por ÚLTIMO — o pior caso.
    for (const i of [1, 2, 0]) soltar[i]?.()
    await Promise.all(emVoo)

    expect(daCache()).toEqual(['abalado', 'agarrado', 'cego'])
    expect(enviados[2], 'o terceiro pedido foi montado sobre um cache regredido').toEqual([
      'abalado',
      'agarrado',
      'cego',
    ])
  })

  it('falha devolve as condições anteriores', async () => {
    const client = seeded(character({ activeConditions: '["caido"]' }))
    const api = await import('@/shared/api/api')
    vi.spyOn(api.api.characters, 'updateConditions').mockRejectedValue(new Error('500'))

    await expect(conditionActions(client, CHARACTER_ID).set([])).rejects.toThrow('500')

    expect(cached(client)?.activeConditions).toBe('["caido"]')
  })
})
