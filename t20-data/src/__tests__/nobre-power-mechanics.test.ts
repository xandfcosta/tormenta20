import { describe, expect, it } from 'vitest'
import {
  NOBRE_ELECTIVES,
  activeNobrePowers,
  nobreElectives,
  nobrePowerById,
} from '../nobre-power-mechanics'

/**
 * PDF Cap 1 Nobre p79-82. Pinned:
 *  - 18 eletivos
 *  - Autoridade Feudal: varia 2 PM
 *  - Estrategista: padrao pmCost variavel
 *  - Favor: varia 5 PM
 *  - Grito Tirânico: completa
 *  - Inspirar Confiança: reacao 2 PM
 *  - Inspirar Glória: livre 5 PM cena
 *  - Jogo da Corte: livre 1 PM
 *  - Liderar pelo Exemplo: livre 2 PM
 *  - Língua de Ouro: padrao 4 PM
 *  - Língua de Prata: livre 2 PM
 */

describe('NOBRE_ELECTIVES — shape', () => {
  it('18 eletivos total', () => {
    expect(NOBRE_ELECTIVES.length).toBe(18)
  })

  it('frozen', () => {
    expect(Object.isFrozen(NOBRE_ELECTIVES)).toBe(true)
  })

  it('bookPage entre 79 e 80', () => {
    for (const p of NOBRE_ELECTIVES) {
      expect(p.bookPage).toBeGreaterThanOrEqual(79)
      expect(p.bookPage).toBeLessThanOrEqual(80)
    }
  })

  it('IDs únicos', () => {
    const ids = NOBRE_ELECTIVES.map((p) => p.id)
    expect(new Set(ids).size).toBe(ids.length)
  })
})

describe('NOBRE_ELECTIVES — pinned entries', () => {
  it('Autoridade Feudal: varia 2 PM', () => {
    const p = nobrePowerById('autoridade-feudal')!
    expect(p.action).toBe('varia')
    expect(p.pmCost).toBe(2)
  })

  it('Estrategista: padrao pmCost variavel', () => {
    const p = nobrePowerById('estrategista')!
    expect(p.action).toBe('padrao')
    expect(p.pmCost).toBe('variavel')
  })

  it('Favor: varia 5 PM', () => {
    const p = nobrePowerById('favor')!
    expect(p.action).toBe('varia')
    expect(p.pmCost).toBe(5)
  })

  it('Grito Tirânico: completa', () => {
    const p = nobrePowerById('grito-tiranico')!
    expect(p.action).toBe('completa')
  })

  it('Inspirar Confiança: reacao 2 PM', () => {
    const p = nobrePowerById('inspirar-confianca')!
    expect(p.action).toBe('reacao')
    expect(p.pmCost).toBe(2)
  })

  it('Inspirar Glória: livre 5 PM cena', () => {
    const p = nobrePowerById('inspirar-gloria')!
    expect(p.action).toBe('livre')
    expect(p.pmCost).toBe(5)
    expect(p.uses).toBe('cena')
  })

  it('Jogo da Corte: livre 1 PM', () => {
    const p = nobrePowerById('jogo-da-corte')!
    expect(p.action).toBe('livre')
    expect(p.pmCost).toBe(1)
  })

  it('Liderar pelo Exemplo: livre 2 PM', () => {
    const p = nobrePowerById('liderar-pelo-exemplo')!
    expect(p.action).toBe('livre')
    expect(p.pmCost).toBe(2)
  })

  it('Língua de Ouro: padrao 4 PM', () => {
    const p = nobrePowerById('lingua-de-ouro')!
    expect(p.action).toBe('padrao')
    expect(p.pmCost).toBe(4)
  })

  it('Língua de Prata: livre 2 PM', () => {
    const p = nobrePowerById('lingua-de-prata')!
    expect(p.action).toBe('livre')
    expect(p.pmCost).toBe(2)
  })
})

describe('nobrePowerById', () => {
  it('miss retorna undefined', () => {
    expect(nobrePowerById('inexistente')).toBeUndefined()
  })
})

describe('nobreElectives', () => {
  it('retorna todos', () => {
    expect(nobreElectives()).toBe(NOBRE_ELECTIVES)
  })
})

describe('activeNobrePowers', () => {
  it('exclui passivos', () => {
    for (const p of activeNobrePowers()) {
      expect(p.action).not.toBe('passivo')
    }
  })

  it('exatamente 10 poderes ativos', () => {
    expect(activeNobrePowers().length).toBe(10)
  })
})
