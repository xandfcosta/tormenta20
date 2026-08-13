import { QueryClient } from '@tanstack/solid-query'
import type { ActivationSpec } from '@tormenta20/t20-data'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { makeCharacter } from '@/entities/character/__fixtures__/character'
import { characterQueryOptions } from '@/entities/character/queries'
import { getActivation } from '@/shared/lib/activation-cache'
import type { ActiveEffect, Character } from '@/shared/api/api'
import { createConditionalsStore } from '@/shared/stores/conditionals-store'
import { createPowerUsesStore } from '@/shared/stores/power-uses-store'
import { createStanceActivationStore } from '@/shared/stores/stance-activation-store'
import { type PowerStores, powerActions } from './power-actions'

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

const CHARACTER_ID = 1
const furia = () => getActivation('class.barbaro.furia') as ActivationSpec

function barbaro(overrides: Partial<Character> = {}): Character {
  return makeCharacter({
    classes: [{ className: 'Bárbaro', level: 10 }],
    mpMax: 20,
    mpCurrent: 20,
    ...overrides,
  })
}

function stores(): PowerStores {
  return {
    conditionals: createConditionalsStore(new FakeStorage()),
    powerUses: createPowerUsesStore(new FakeStorage()),
    stanceActivations: createStanceActivationStore(new FakeStorage()),
  }
}

function seeded(char: Character) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  client.setQueryData(characterQueryOptions(CHARACTER_ID).queryKey, char)
  return client
}

const cached = (client: QueryClient) =>
  client.getQueryData<Character>(characterQueryOptions(CHARACTER_ID).queryKey)

let updateVitals: ReturnType<typeof vi.spyOn>

beforeEach(async () => {
  const api = await import('@/shared/api/api')
  // Echoes what was asked, like the real endpoint (which answers the CLAMPED
  // pair) — a fixed number here would hide that the cache settles on the
  // server's word, not on the optimistic guess.
  updateVitals = vi
    .spyOn(api.api.characters, 'updateVitals')
    .mockImplementation(async (_id, input) => ({
      hpCurrent: 10,
      mpCurrent: input.mpCurrent ?? 0,
    }))
})

afterEach(() => vi.restoreAllMocks())

describe('powerActions.activateStance', () => {
  it('paga o custo base e liga a flag da postura', async () => {
    const char = barbaro()
    const client = seeded(char)
    const store = stores()

    await powerActions(client, char, store).activateStance(furia(), 0)

    // Fúria p40: 2 PM base.
    expect(updateVitals).toHaveBeenCalledWith(CHARACTER_ID, { mpCurrent: 18 })
    expect(cached(client)?.mpCurrent).toBe(18)
    expect(store.stanceActivations.paidFor(CHARACTER_ID, 'furia')).toEqual({
      steps: 0,
      pmPaid: 2,
    })
  })

  // p40: "a cada 5 níveis, pode gastar +1 PM para aumentar o bônus em +1".
  it('cada degrau do stepper cobra 1 PM a mais', async () => {
    const char = barbaro()
    const client = seeded(char)
    const store = stores()

    await powerActions(client, char, store).activateStance(furia(), 2)

    expect(updateVitals).toHaveBeenCalledWith(CHARACTER_ID, { mpCurrent: 16 })
    expect(store.stanceActivations.paidFor(CHARACTER_ID, 'furia')?.pmPaid).toBe(4)
  })

  it('sem PM para o total, não entra na postura nem cobra', async () => {
    const char = barbaro({ mpCurrent: 1 })
    const client = seeded(char)
    const store = stores()

    await powerActions(client, char, store).activateStance(furia(), 0)

    expect(updateVitals).not.toHaveBeenCalled()
    expect(cached(client)?.mpCurrent).toBe(1)
    expect(store.stanceActivations.paidFor(CHARACTER_ID, 'furia')).toBeUndefined()
  })

  // O PM sai antes da resposta; se o servidor recusa, tem de voltar — senão o
  // jogador perde PM que nunca foi debitado de verdade.
  it('falha do servidor devolve o PM', async () => {
    const char = barbaro()
    const client = seeded(char)
    updateVitals.mockRejectedValue(new Error('500'))

    await powerActions(client, char, stores()).activateStance(furia(), 0)

    expect(cached(client)?.mpCurrent).toBe(20)
  })
})

describe('powerActions.deactivateStance', () => {
  it('encerra de graça: desliga, apaga o registro e não devolve PM', async () => {
    const char = barbaro()
    const client = seeded(char)
    const store = stores()
    await powerActions(client, char, store).activateStance(furia(), 1)
    updateVitals.mockClear()

    await powerActions(client, cached(client) as Character, store).deactivateStance('furia')

    expect(updateVitals).not.toHaveBeenCalled()
    expect(store.stanceActivations.paidFor(CHARACTER_ID, 'furia')).toBeUndefined()
  })
})

describe('powerActions.use', () => {
  // Um instantâneo limitado gasta PM e queima o uso da cena.
  it('gasta o PM e marca o uso quando o poder é limitado', async () => {
    const spec: ActivationSpec = {
      id: 'class.teste.instantaneo',
      name: 'Instantâneo',
      kind: 'instant',
      action: 'padrao',
      pmCost: 3,
      uses: 'cena',
      bookPage: 1,
    }
    const char = barbaro()
    const client = seeded(char)
    const store = stores()

    await powerActions(client, char, store).use(spec)

    expect(updateVitals).toHaveBeenCalledWith(CHARACTER_ID, { mpCurrent: 17 })
    expect(store.powerUses.used(CHARACTER_ID, spec.id).scene).toBe(1)
  })

  it('poder de 0 PM não faz requisição nenhuma', async () => {
    const spec: ActivationSpec = {
      id: 'class.teste.livre',
      name: 'Livre',
      kind: 'instant',
      action: 'livre',
      pmCost: 0,
      uses: null,
      bookPage: 1,
    }
    const char = barbaro()

    await powerActions(seeded(char), char, stores()).use(spec)

    expect(updateVitals).not.toHaveBeenCalled()
  })

  it('segundo uso na mesma cena é recusado', async () => {
    const spec: ActivationSpec = {
      id: 'class.teste.limitado',
      name: 'Limitado',
      kind: 'instant',
      action: 'padrao',
      pmCost: 1,
      uses: 'cena',
      bookPage: 1,
    }
    const char = barbaro()
    const client = seeded(char)
    const store = stores()
    await powerActions(client, char, store).use(spec)
    updateVitals.mockClear()

    await powerActions(client, cached(client) as Character, store).use(spec)

    expect(updateVitals).not.toHaveBeenCalled()
    expect(store.powerUses.used(CHARACTER_ID, spec.id).scene).toBe(1)
  })
})

describe('powerActions — concessões da postura', () => {
  const almaDeBronze = 'class.barbaro.alma-de-bronze'
  const grantEffect: ActiveEffect = {
    id: 42,
    catalogId: almaDeBronze,
    scope: 'scene',
    modifiers: '[]',
    createdAt: '2026-01-01',
  }

  it('entrar na Fúria aplica a concessão que o personagem tem', async () => {
    const char = barbaro({ classPowers: JSON.stringify([almaDeBronze]) })
    const client = seeded(char)
    const api = await import('@/shared/api/api')
    const applyEffect = vi
      .spyOn(api.api.characters, 'applyEffect')
      .mockResolvedValue(grantEffect)

    await powerActions(client, char, stores()).activateStance(furia(), 0)

    expect(applyEffect).toHaveBeenCalledWith(CHARACTER_ID, { powerId: almaDeBronze })
  })

  it('quem não tem o poder não ganha a concessão', async () => {
    const char = barbaro({ classPowers: '[]' })
    const api = await import('@/shared/api/api')
    const applyEffect = vi.spyOn(api.api.characters, 'applyEffect')

    await powerActions(seeded(char), char, stores()).activateStance(furia(), 0)

    expect(applyEffect).not.toHaveBeenCalled()
  })

  it('sair da Fúria remove o efeito que ela concedeu', async () => {
    const char = barbaro({
      classPowers: JSON.stringify([almaDeBronze]),
      activeEffects: [grantEffect],
    })
    const api = await import('@/shared/api/api')
    const removeEffect = vi
      .spyOn(api.api.characters, 'removeActiveEffect')
      .mockResolvedValue({ id: 42 })

    await powerActions(seeded(char), char, stores()).deactivateStance('furia')

    expect(removeEffect).toHaveBeenCalledWith(CHARACTER_ID, 42)
  })
})
