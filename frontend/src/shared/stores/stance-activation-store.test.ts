import { FakeStorage } from '@/shared/test/fake-storage'
import { describe, expect, it } from 'vitest'
import {
  STANCE_ACTIVATIONS_STORAGE_KEY,
  createStanceActivationStore,
  readStoredStanceActivations,
} from './stance-activation-store'


describe('readStoredStanceActivations', () => {
  it('lê a forma do zustand que o React grava', () => {
    const raw = JSON.stringify({
      state: { records: { '1': { furia: { steps: 1, pmPaid: 3 } } } },
    })

    expect(readStoredStanceActivations(raw)['1'].furia).toEqual({ steps: 1, pmPaid: 3 })
  })

  it('blob corrompido ou ausente não derruba a ficha', () => {
    expect(readStoredStanceActivations(null)).toEqual({})
    expect(readStoredStanceActivations('{quebrado')).toEqual({})
    expect(readStoredStanceActivations(JSON.stringify({ state: { records: 7 } }))).toEqual({})
  })

  // Registro sem os números não descreve pagamento nenhum — cai fora em vez de
  // virar NaN na tela de Posturas ativas.
  it('registro sem steps/pmPaid numéricos é descartado', () => {
    const raw = JSON.stringify({ state: { records: { '1': { furia: { steps: 'x' } } } } })

    expect(readStoredStanceActivations(raw)).toEqual({})
  })
})

describe('createStanceActivationStore', () => {
  it('guarda o que foi pago e persiste na chave do React', () => {
    const storage = new FakeStorage()
    const store = createStanceActivationStore(storage)

    store.logActivation(1, 'furia', { steps: 1, pmPaid: 3 })

    expect(store.paidFor(1, 'furia')).toEqual({ steps: 1, pmPaid: 3 })
    const persisted = readStoredStanceActivations(storage.getItem(STANCE_ACTIVATIONS_STORAGE_KEY))
    expect(persisted['1'].furia.pmPaid).toBe(3)
  })

  it('stance nunca ativada não tem registro', () => {
    const store = createStanceActivationStore(new FakeStorage())
    expect(store.paidFor(1, 'furia')).toBeUndefined()
  })

  it('personagens diferentes guardam pagamentos separados', () => {
    const store = createStanceActivationStore(new FakeStorage())
    store.logActivation(1, 'furia', { steps: 2, pmPaid: 4 })

    expect(store.paidFor(2, 'furia')).toBeUndefined()
  })

  it('encerrar a stance apaga o registro, sem tocar nas outras', () => {
    const store = createStanceActivationStore(new FakeStorage())
    store.logActivation(1, 'furia', { steps: 1, pmPaid: 3 })
    store.logActivation(1, 'inspiracao', { steps: 0, pmPaid: 2 })

    store.clearActivation(1, 'furia')

    expect(store.paidFor(1, 'furia')).toBeUndefined()
    expect(store.paidFor(1, 'inspiracao')).toEqual({ steps: 0, pmPaid: 2 })
  })

  it('encerrar stance que não estava ativa é inofensivo', () => {
    const store = createStanceActivationStore(new FakeStorage())
    expect(() => store.clearActivation(9, 'furia')).not.toThrow()
  })
})
