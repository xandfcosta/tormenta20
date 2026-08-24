import type { AttributeKey } from '@/shared/api/attribute-keys'
import { describe, expect, it } from 'vitest'
import { pointBuyStatusFor } from './point-buy'

const attrs = (patch: Partial<Record<AttributeKey, number>> = {}) => ({
  strength: 0,
  dexterity: 0,
  constitution: 0,
  intelligence: 0,
  wisdom: 0,
  charisma: 0,
  ...patch,
})

/**
 * FUMAÇA DA PONTE, e só isso (ALE-187).
 *
 * `pointBuyStatusFor` é literalmente `return enginePointBuyStatus(attrs)`: uma
 * linha sobre o WASM. A Tabela 1-1 (p17) tem dono em `engine/pointbuy_test.go`,
 * e re-afirmá-la aqui era a terceira cópia de uma regra do livro — o que a
 * ALE-104 apagou ao aposentar o `t20-data`.
 *
 * Sobraram DOIS casos, um por FORMA de resposta que a ponte pode devolver: um
 * número calculado e o nulo do modo livre. Se o motor não carregar, ou se a
 * travessia perder o valor, é aqui que aparece — e aparece rápido, antes de a
 * Forja inteira ficar vermelha por um motivo que não é dela.
 *
 * O que saiu e por onde está coberto: os avisos ("excedem o limite", "apenas UM
 * atributo") são texto que o jogador lê e vivem afirmados MONTADOS em
 * `pages/characters/forge/atributos-step.test.tsx`; o resto era a tabela.
 */
describe('pointBuyStatusFor — a ponte com o motor', () => {
  it('devolve o número que a Tabela 1-1 manda (1/2/4/7)', () => {
    expect(pointBuyStatusFor(attrs({ strength: 1 })).spent).toBe(1)
    expect(pointBuyStatusFor(attrs({ strength: 3 })).spent).toBe(4)
    expect(pointBuyStatusFor(attrs({ strength: 4 })).spent).toBe(7)
  })

  // Modo livre: o motor recusa base fora da faixa e devolve nulo em vez de
  // exceção, e a Forja mostra "—" no lugar do custo. É a outra forma que a
  // ponte devolve, e uma travessia que estourasse aqui derrubaria o passo.
  it('valor fora da tabela volta como nulo, não como exceção', () => {
    expect(pointBuyStatusFor(attrs({ strength: 9 })).spent).toBeNull()
  })
})
