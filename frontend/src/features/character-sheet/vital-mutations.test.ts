import { QueryClient } from '@tanstack/solid-query'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { characterQueryOptions } from '@/entities/character/queries'
import type { ActiveEffect, Character } from '@/shared/api/api'
import { clampVital, createVitalActions, predictDamage } from './vital-mutations'

const CHARACTER_ID = 1
const WAIT = 350

/** An ActiveEffect carrying a temp-PV pool of `amount`. */
function pool(id: number, amount: number, source = 'Alma de Bronze'): ActiveEffect {
  return {
    id,
    catalogId: 'alma-de-bronze',
    scope: 'scene',
    modifiers: JSON.stringify([
      { target: { k: 'tempHp' }, amount, bonusType: 'untyped', note: source },
    ]),
    createdAt: '2026-01-01T00:00:00.000Z',
  }
}

function character(overrides: Partial<Character> = {}): Character {
  return {
    id: CHARACTER_ID,
    name: 'Tanque',
    level: 10,
    hpMax: 100,
    hpCurrent: 50,
    mpMax: 20,
    mpCurrent: 10,
    items: [],
    activeEffects: [],
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

const actionsFor = (client: QueryClient, char: Character) =>
  createVitalActions(client, () => cached(client) ?? char, { wait: WAIT })

describe('clampVital', () => {
  it('prende entre 0 e o máximo', () => {
    expect(clampVital(-5, 30)).toBe(0)
    expect(clampVital(99, 30)).toBe(30)
    expect(clampVital(12, 30)).toBe(12)
  })
})

describe('predictDamage', () => {
  it('sem pool temporário, tira tudo dos PV', () => {
    expect(predictDamage(character({ hpCurrent: 50 }), 7).hpCurrent).toBe(43)
  })

  // PV temporário é escudo: o dano come o pool primeiro (p256).
  it('drena o pool antes de tocar nos PV', () => {
    const after = predictDamage(character({ hpCurrent: 50, activeEffects: [pool(9, 10)] }), 4)
    expect(after.hpCurrent).toBe(50)
    expect(after.activeEffects[0].modifiers).toContain('"amount":6')
  })

  it('o que sobra depois de esvaziar o pool cai nos PV', () => {
    const after = predictDamage(character({ hpCurrent: 50, activeEffects: [pool(9, 10)] }), 14)
    expect(after.hpCurrent).toBe(46)
  })

  // Morrer é assunto do servidor; a ficha não pinta PV negativo.
  it('não deixa os PV passarem de zero', () => {
    expect(predictDamage(character({ hpCurrent: 3 }), 999).hpCurrent).toBe(0)
  })
})

beforeEach(() => vi.useFakeTimers())

afterEach(() => {
  vi.useRealTimers()
  vi.restoreAllMocks()
})

describe('createVitalActions — setHp/setMp', () => {
  it('pinta na hora e só manda uma vez no fim da rajada', async () => {
    const client = seeded()
    const api = await import('@/shared/api/api')
    const call = vi
      .spyOn(api.api.characters, 'updateVitals')
      .mockResolvedValue({ hpCurrent: 47, mpCurrent: 10 })
    const actions = actionsFor(client, character())

    actions.setHp(49)
    actions.setHp(48)
    actions.setHp(47)

    // Três cliques, zero requisições enquanto o dedo ainda está batendo.
    expect(cached(client)?.hpCurrent).toBe(47)
    expect(call).not.toHaveBeenCalled()

    await vi.advanceTimersByTimeAsync(WAIT)
    expect(call).toHaveBeenCalledTimes(1)
    expect(call).toHaveBeenCalledWith(CHARACTER_ID, { hpCurrent: 47 })
  })

  it('prende no máximo e ignora o valor que não mudou nada', async () => {
    const client = seeded()
    const api = await import('@/shared/api/api')
    const call = vi
      .spyOn(api.api.characters, 'updateVitals')
      .mockResolvedValue({ hpCurrent: 100, mpCurrent: 10 })
    const actions = actionsFor(client, character())

    actions.setHp(999)
    expect(cached(client)?.hpCurrent).toBe(100)

    await vi.advanceTimersByTimeAsync(WAIT)
    call.mockClear()

    actions.setHp(100)
    await vi.advanceTimersByTimeAsync(WAIT)
    expect(call).not.toHaveBeenCalled()
  })

  /**
   * Rollback silencioso é o pior caso possível numa ficha: o jogador acha que
   * o dano ficou salvo e joga a noite inteira com o PV errado.
   */
  it('falha da rede devolve o valor de ANTES da rajada inteira', async () => {
    const client = seeded()
    const api = await import('@/shared/api/api')
    vi.spyOn(api.api.characters, 'updateVitals').mockRejectedValue(new Error('offline'))
    const actions = actionsFor(client, character())

    actions.setHp(49)
    actions.setHp(48)
    await vi.advanceTimersByTimeAsync(WAIT)

    expect(cached(client)?.hpCurrent).toBe(50)
  })

  it('a resposta do servidor é a palavra final sobre PV e PM', async () => {
    const client = seeded()
    const api = await import('@/shared/api/api')
    vi.spyOn(api.api.characters, 'updateVitals').mockResolvedValue({
      hpCurrent: 42,
      mpCurrent: 7,
    })
    const actions = actionsFor(client, character())

    actions.setMp(9)
    await vi.advanceTimersByTimeAsync(WAIT)

    expect(cached(client)?.hpCurrent).toBe(42)
    expect(cached(client)?.mpCurrent).toBe(7)
  })
})

describe('createVitalActions — applyDamage', () => {
  it('pinta a previsão e depois assenta pelo delta do servidor', async () => {
    const client = seeded(character({ hpCurrent: 50, activeEffects: [pool(9, 10)] }))
    const api = await import('@/shared/api/api')
    vi.spyOn(api.api.characters, 'applyDamage').mockResolvedValue({
      hpCurrent: 46,
      tempHpRemaining: 0,
      drained: [{ effectId: 9, newAmount: 0, removed: true }],
    })

    await actionsFor(client, character()).applyDamage(14)

    expect(cached(client)?.hpCurrent).toBe(46)
    expect(cached(client)?.activeEffects).toEqual([])
  })

  it('falha recoloca o personagem que estava em cache', async () => {
    const client = seeded(character({ hpCurrent: 50 }))
    const api = await import('@/shared/api/api')
    vi.spyOn(api.api.characters, 'applyDamage').mockRejectedValue(new Error('500'))

    await expect(actionsFor(client, character()).applyDamage(7)).rejects.toThrow('500')

    expect(cached(client)?.hpCurrent).toBe(50)
  })
})

describe('createVitalActions — setManualTempHp', () => {
  it('aplica o pool devolvido pelo servidor', async () => {
    const client = seeded()
    const api = await import('@/shared/api/api')
    vi.spyOn(api.api.characters, 'applyEffect').mockResolvedValue({
      effect: pool(11, 12, 'PV temporários (manual)'),
      displaced: [],
    })

    await actionsFor(client, character()).setManualTempHp(12)

    expect(cached(client)?.activeEffects.map((e) => e.id)).toEqual([11])
  })

  /**
   * Vale-o-maior (p256): um pool maior já ativo vence, e o servidor responde
   * "superseded" sem escrever nada. A ficha não pode apagar o pool bom.
   */
  it('pool maior já ativo não mexe no cache', async () => {
    const client = seeded(character({ activeEffects: [pool(9, 20)] }))
    const api = await import('@/shared/api/api')
    vi.spyOn(api.api.characters, 'applyEffect').mockResolvedValue({
      superseded: true,
      keptEffectId: 9,
      keptAmount: 20,
    })

    await actionsFor(client, character()).setManualTempHp(5)

    expect(cached(client)?.activeEffects.map((e) => e.id)).toEqual([9])
  })

  it('zerar limpa o pool manual', async () => {
    const client = seeded(character({ activeEffects: [pool(11, 12, 'manual')] }))
    const api = await import('@/shared/api/api')
    vi.spyOn(api.api.characters, 'applyEffect').mockResolvedValue({
      cleared: true,
      removedEffectIds: [11],
    })

    await actionsFor(client, character()).setManualTempHp(0)

    expect(cached(client)?.activeEffects).toEqual([])
  })
})
