import { describe, expect, it, vi } from 'vitest'
import { createConditionalsStore } from './conditionals-store'

/**
 * O store dos SITUACIONAIS depois que o servidor virou dono (ALE-222).
 *
 * O que este arquivo testava antes era a leitura defensiva de um blob de
 * `localStorage`. Esse blob não existe mais, e a regra que substituiu a antiga é
 * o OTIMISMO: a tela pinta antes de o servidor responder e volta atrás se ele
 * recusar. É isso que se protege aqui — não o CRUD, que o typechecker já
 * garante.
 */

const HEROI = 7

/** Um servidor que aceita e lembra o que recebeu. */
function servidorQueAceita() {
  return vi.fn(async () => {})
}

describe('store dos situacionais', () => {
  it('liga o situacional na hora, sem esperar o servidor', () => {
    // Um servidor que NUNCA responde: se a leitura dependesse dele, viria
    // vazia — e é exatamente esse o ponto do otimismo.
    const store = createConditionalsStore(() => new Promise(() => {}))

    store.toggle(HEROI, 'furia')

    expect([...store.active(HEROI)]).toEqual(['furia'])
  })

  it('manda ao servidor o CONJUNTO final, e não o que mudou', () => {
    const write = servidorQueAceita()
    const store = createConditionalsStore(write)

    store.toggle(HEROI, 'furia')
    store.toggle(HEROI, 'ataque-poderoso')

    expect(write).toHaveBeenLastCalledWith(HEROI, ['furia', 'ataque-poderoso'])
  })

  it('DESFAZ quando o servidor recusa', async () => {
    const store = createConditionalsStore(() => Promise.reject(new Error('500')))

    store.toggle(HEROI, 'furia')

    await vi.waitFor(() => expect([...store.active(HEROI)]).toEqual([]))
  })

  it('avisa quem escuta que a recusa aconteceu', async () => {
    // Sem esta metade o toggle voltaria sozinho e o jogador acharia que errou o
    // clique: desfazer em silêncio é pior que mostrar o erro.
    const aviso = vi.fn()
    const store = createConditionalsStore(() => Promise.reject(new Error('500')), aviso)

    store.toggle(HEROI, 'furia')

    await vi.waitFor(() => expect(aviso).toHaveBeenCalled())
  })

  it('hidrata SUBSTITUINDO o que estava no cache', () => {
    const store = createConditionalsStore(servidorQueAceita())
    store.toggle(HEROI, 'ligado-nesta-aba')

    store.hydrate(HEROI, ['furia'])

    // Mesclar deixaria sobreviver aqui um situacional que outra aba desligou, e
    // a ficha recomputaria contra um conjunto que não existe no servidor.
    expect([...store.active(HEROI)]).toEqual(['furia'])
  })

  it('cada personagem tem o próprio conjunto', () => {
    const store = createConditionalsStore(servidorQueAceita())

    store.toggle(HEROI, 'furia')
    store.toggle(99, 'ataque-poderoso')

    expect([...store.active(HEROI)]).toEqual(['furia'])
    expect([...store.active(99)]).toEqual(['ataque-poderoso'])
  })

  it('setMany liga e desliga em lote', () => {
    const store = createConditionalsStore(servidorQueAceita())

    store.setMany(HEROI, ['a', 'b', 'c'], true)
    store.setMany(HEROI, ['b'], false)

    expect([...store.active(HEROI)].sort()).toEqual(['a', 'c'])
  })
})
