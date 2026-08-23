import { describe, expect, it, vi } from 'vitest'
import { createStanceActivationStore } from './stance-activation-store'

/**
 * O registro do que cada POSTURA custou, depois que o servidor virou dono
 * (ALE-222).
 *
 * A razão de este store existir não mudou: sair da postura não pode devolver
 * PM. Perder o registro é o pior dos três casos de falha, e é por isso que o
 * desfazer aqui tem teste próprio.
 */

const HEROI = 7
const PAGO = { steps: 2, pmPaid: 4 }

function servidorMudo() {
  return { set: vi.fn(async () => {}), clear: vi.fn(async () => {}) }
}

describe('store das posturas', () => {
  it('registra o pagamento na hora e o manda ao servidor', () => {
    const write = servidorMudo()
    const store = createStanceActivationStore(write)

    store.logActivation(HEROI, 'furia', PAGO)

    expect(store.paidFor(HEROI, 'furia')).toEqual(PAGO)
    expect(write.set).toHaveBeenCalledWith(HEROI, 'furia', PAGO)
  })

  it('postura que nunca foi paga não tem registro', () => {
    const store = createStanceActivationStore(servidorMudo())

    expect(store.paidFor(HEROI, 'furia')).toBeUndefined()
  })

  it('sair da postura apaga o registro e avisa o servidor', () => {
    const write = servidorMudo()
    const store = createStanceActivationStore(write)
    store.logActivation(HEROI, 'furia', PAGO)

    store.clearActivation(HEROI, 'furia')

    expect(store.paidFor(HEROI, 'furia')).toBeUndefined()
    expect(write.clear).toHaveBeenCalledWith(HEROI, 'furia')
  })

  it('DESFAZ o registro quando o servidor recusa a entrada', async () => {
    const store = createStanceActivationStore({
      set: () => Promise.reject(new Error('500')),
      clear: async () => {},
    })

    store.logActivation(HEROI, 'furia', PAGO)

    await vi.waitFor(() => expect(store.paidFor(HEROI, 'furia')).toBeUndefined())
  })

  it('DEVOLVE o registro quando o servidor recusa a saída', async () => {
    // Este é o caso perigoso: sem a volta, o pagamento some da tela e sair da
    // postura passa a devolver PM — exatamente o que este store impede.
    const store = createStanceActivationStore({
      set: async () => {},
      clear: () => Promise.reject(new Error('500')),
    })
    store.logActivation(HEROI, 'furia', PAGO)

    store.clearActivation(HEROI, 'furia')

    await vi.waitFor(() => expect(store.paidFor(HEROI, 'furia')).toEqual(PAGO))
  })

  it('hidrata substituindo o cache', () => {
    const store = createStanceActivationStore(servidorMudo())
    store.logActivation(HEROI, 'lixo', PAGO)

    store.hydrate(HEROI, [{ flag: 'furia', steps: 1, pmPaid: 2 }])

    expect(store.paidFor(HEROI, 'furia')).toEqual({ steps: 1, pmPaid: 2 })
    expect(store.paidFor(HEROI, 'lixo')).toBeUndefined()
  })
})
