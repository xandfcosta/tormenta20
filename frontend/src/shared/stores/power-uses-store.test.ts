import { describe, expect, it, vi } from 'vitest'
import { createPowerUsesStore } from './power-uses-store'

/**
 * Os contadores de USO depois que o servidor virou dono (ALE-222).
 *
 * A regra que este arquivo protege é a mesma do situacional — otimismo e volta
 * atrás — mais uma que é só daqui: o que vai no fio é "gastei MAIS UM", nunca o
 * total. Um store que mandasse o total perderia um uso a cada par de cliques
 * rápidos, e o teste que só olhasse o contador local não veria isso.
 */

const HEROI = 7

describe('store dos usos de poder', () => {
  it('conta o uso na hora, sem esperar o servidor', () => {
    const store = createPowerUsesStore(() => new Promise(() => {}))

    store.bump(HEROI, 'furia', 'day')

    expect(store.used(HEROI, 'furia')).toEqual({ scene: 0, day: 1 })
  })

  it('manda "mais um" com o escopo, e nunca o total', () => {
    const write = vi.fn(async () => {})
    const store = createPowerUsesStore(write)

    store.bump(HEROI, 'furia', 'day')
    store.bump(HEROI, 'furia', 'day')

    // Duas chamadas iguais — é o servidor que soma. Se o store mandasse o total,
    // a segunda chamada traria 2 e dois cliques simultâneos gravariam 2 duas
    // vezes, perdendo um uso.
    expect(write).toHaveBeenCalledTimes(2)
    expect(write).toHaveBeenNthCalledWith(1, HEROI, 'furia', 'day')
    expect(write).toHaveBeenNthCalledWith(2, HEROI, 'furia', 'day')
  })

  it('as contas de cena e de dia do MESMO poder são independentes', () => {
    const store = createPowerUsesStore(async () => {})

    store.bump(HEROI, 'furia', 'scene')
    store.bump(HEROI, 'furia', 'day')
    store.bump(HEROI, 'furia', 'day')

    expect(store.used(HEROI, 'furia')).toEqual({ scene: 1, day: 2 })
  })

  it('DESFAZ o contador quando o servidor recusa', async () => {
    const store = createPowerUsesStore(() => Promise.reject(new Error('500')))

    store.bump(HEROI, 'furia', 'day')

    // Aqui o desfazer importa mais que no situacional: um contador que ficou
    // alto sozinho faz o jogador achar que gastou um poder que ainda tem.
    await vi.waitFor(() => expect(store.used(HEROI, 'furia').day).toBe(0))
  })

  it('o descanso de cena zera só a cena', () => {
    const store = createPowerUsesStore(async () => {})
    store.bump(HEROI, 'furia', 'scene')
    store.bump(HEROI, 'milagre', 'day')

    store.resetScene(HEROI)

    expect(store.used(HEROI, 'furia').scene).toBe(0)
    // A metade que importa: o "1/dia" NÃO volta, senão o descanso de cena
    // devolveria um poder que o livro não devolve.
    expect(store.used(HEROI, 'milagre').day).toBe(1)
  })

  it('o descanso de dia zera os dois escopos', () => {
    const store = createPowerUsesStore(async () => {})
    store.bump(HEROI, 'furia', 'scene')
    store.bump(HEROI, 'milagre', 'day')

    store.resetDay(HEROI)

    expect(store.used(HEROI, 'furia').scene).toBe(0)
    expect(store.used(HEROI, 'milagre').day).toBe(0)
  })

  it('o reset NÃO fala com o servidor — quem chama já falou', () => {
    // O `/end-scene` limpa lá; repetir a chamada aqui seria a mesma decisão
    // escrita em dois lugares, e um dia elas discordariam.
    const write = vi.fn(async () => {})
    const store = createPowerUsesStore(write)
    store.bump(HEROI, 'furia', 'scene')
    write.mockClear()

    store.resetScene(HEROI)
    store.resetDay(HEROI)

    expect(write).not.toHaveBeenCalled()
  })

  it('hidrata substituindo o cache', () => {
    const store = createPowerUsesStore(async () => {})
    store.bump(HEROI, 'lixo', 'day')

    store.hydrate(HEROI, [{ powerId: 'furia', scope: 'scene', used: 3 }])

    expect(store.used(HEROI, 'furia')).toEqual({ scene: 3, day: 0 })
    expect(store.used(HEROI, 'lixo')).toEqual({ scene: 0, day: 0 })
  })
})
