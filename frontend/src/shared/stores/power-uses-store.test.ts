import { FakeStorage } from '@/shared/test/fake-storage'
import { describe, expect, it } from 'vitest'
import {
  POWER_USES_STORAGE_KEY,
  createPowerUsesStore,
  readStoredPowerUses,
} from './power-uses-store'


const POWER = 'class.barbaro.golpe-poderoso'

describe('readStoredPowerUses', () => {
  it('lê a forma do zustand que o React grava', () => {
    const raw = JSON.stringify({ state: { uses: { '1': { scene: { [POWER]: 2 }, day: {} } } } })
    expect(readStoredPowerUses(raw)['1'].scene[POWER]).toBe(2)
  })

  it('blob corrompido ou ausente não derruba a ficha', () => {
    expect(readStoredPowerUses(null)).toEqual({})
    expect(readStoredPowerUses('{quebrado')).toEqual({})
    expect(readStoredPowerUses(JSON.stringify({ state: { uses: 'nem objeto' } }))).toEqual({})
  })
})

describe('createPowerUsesStore', () => {
  it('conta os usos por escopo e persiste na chave do React', () => {
    const storage = new FakeStorage()
    const store = createPowerUsesStore(storage)

    store.bump(1, POWER, 'scene')
    store.bump(1, POWER, 'scene')
    store.bump(1, POWER, 'day')

    expect(store.used(1, POWER)).toEqual({ scene: 2, day: 1 })
    expect(readStoredPowerUses(storage.getItem(POWER_USES_STORAGE_KEY))['1'].scene[POWER]).toBe(2)
  })

  it('personagens diferentes contam separado', () => {
    const store = createPowerUsesStore(new FakeStorage())
    store.bump(1, POWER, 'scene')
    expect(store.used(2, POWER)).toEqual({ scene: 0, day: 0 })
  })

  it('encerrar cena zera só a cena', () => {
    const store = createPowerUsesStore(new FakeStorage())
    store.bump(1, POWER, 'scene')
    store.bump(1, POWER, 'day')

    store.resetScene(1)

    expect(store.used(1, POWER)).toEqual({ scene: 0, day: 1 })
  })

  // Encerrar o dia encerra a cena que estava rolando (descanso do livro).
  it('encerrar dia zera os dois escopos', () => {
    const store = createPowerUsesStore(new FakeStorage())
    store.bump(1, POWER, 'scene')
    store.bump(1, POWER, 'day')

    store.resetDay(1)

    expect(store.used(1, POWER)).toEqual({ scene: 0, day: 0 })
  })

  it('encerrar cena de um personagem sem uso nenhum é inofensivo', () => {
    const store = createPowerUsesStore(new FakeStorage())
    expect(() => store.resetScene(99)).not.toThrow()
    expect(store.used(99, POWER)).toEqual({ scene: 0, day: 0 })
  })
})
