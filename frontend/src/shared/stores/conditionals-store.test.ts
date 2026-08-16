import { FakeStorage } from '@/shared/test/fake-storage'
import { describe, expect, it } from 'vitest'
import { CONDITIONALS_STORAGE_KEY, createConditionalsStore } from './conditionals-store'


const HERO = 1
const OTHER = 2

describe('createConditionalsStore', () => {
  it('começa sem nada ativo', () => {
    const store = createConditionalsStore(new FakeStorage())
    expect([...store.active(HERO)]).toEqual([])
  })

  it('alterna um condicional para ligado e para desligado', () => {
    const store = createConditionalsStore(new FakeStorage())

    store.toggle(HERO, 'furia')
    expect(store.active(HERO).has('furia')).toBe(true)

    store.toggle(HERO, 'furia')
    expect(store.active(HERO).has('furia')).toBe(false)
  })

  // Dois personagens abertos na mesma máquina não podem compartilhar a Fúria.
  it('mantém os personagens separados', () => {
    const store = createConditionalsStore(new FakeStorage())

    store.toggle(HERO, 'furia')
    expect(store.active(OTHER).has('furia')).toBe(false)
  })

  it('setMany liga e desliga um lote de uma vez', () => {
    const store = createConditionalsStore(new FakeStorage())

    store.setMany(HERO, ['furia', 'ataque-poderoso'], true)
    expect([...store.active(HERO)].sort()).toEqual(['ataque-poderoso', 'furia'])

    store.setMany(HERO, ['furia'], false)
    expect([...store.active(HERO)]).toEqual(['ataque-poderoso'])
  })

  it('clear zera só o personagem pedido', () => {
    const store = createConditionalsStore(new FakeStorage())
    store.toggle(HERO, 'furia')
    store.toggle(OTHER, 'esquiva')

    store.clear(HERO)

    expect([...store.active(HERO)]).toEqual([])
    expect(store.active(OTHER).has('esquiva')).toBe(true)
  })

  /**
   * Mesma chave e mesma forma do zustand do app React (`t20-conditionals` →
   * `{ state: { active } }`): durante a migração o jogador alterna entre os
   * dois fronts, e o que está ligado na mesa não pode se perder na troca.
   */
  it('persiste na chave e na forma que o app React usava', () => {
    const storage = new FakeStorage()
    createConditionalsStore(storage).toggle(HERO, 'furia')

    expect(JSON.parse(storage.getItem(CONDITIONALS_STORAGE_KEY) ?? '{}')).toEqual({
      state: { active: { '1': ['furia'] } },
    })
  })

  it('relê o que ficou salvo', () => {
    const storage = new FakeStorage()
    storage.setItem(
      CONDITIONALS_STORAGE_KEY,
      JSON.stringify({ state: { active: { '1': ['furia'] } } }),
    )

    expect(createConditionalsStore(storage).active(HERO).has('furia')).toBe(true)
  })

  // Storage corrompido (ou de uma versão antiga) não pode derrubar a ficha.
  it('ignora um storage ilegível em vez de estourar', () => {
    const storage = new FakeStorage()
    storage.setItem(CONDITIONALS_STORAGE_KEY, '{isto não é json')

    expect([...createConditionalsStore(storage).active(HERO)]).toEqual([])
  })
})
