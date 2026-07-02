import { describe, expect, it } from 'vitest'
import {
  INVENTOR_ELECTIVES,
  activeInventorPowers,
  inventorElectives,
  inventorPowerById,
} from '../inventor-power-mechanics'

/**
 * PDF Cap 1 Inventor p68-71. Pinned:
 *  - 30 eletivos
 *  - Actives: Ajuste de Mira (padrao variavel cena), Autômato Prototipado
 *    (padrao 2 PM), Catalisador Instável (completa 3 PM), Chutes e Palavrões
 *    (livre 1 PM rodada), Maestria em Perícia (livre 1 PM), Oficina de Campo
 *    (varia 2 PM dia), Pedra de Amolar (movimento variavel cena)
 */

describe('INVENTOR_ELECTIVES — shape', () => {
  it('30 eletivos total', () => {
    expect(INVENTOR_ELECTIVES.length).toBe(30)
  })

  it('frozen', () => {
    expect(Object.isFrozen(INVENTOR_ELECTIVES)).toBe(true)
  })

  it('bookPage entre 68 e 71', () => {
    for (const p of INVENTOR_ELECTIVES) {
      expect(p.bookPage).toBeGreaterThanOrEqual(68)
      expect(p.bookPage).toBeLessThanOrEqual(71)
    }
  })

  it('IDs únicos', () => {
    const ids = INVENTOR_ELECTIVES.map((p) => p.id)
    expect(new Set(ids).size).toBe(ids.length)
  })
})

describe('INVENTOR_ELECTIVES — pinned entries', () => {
  it('Ajuste de Mira: padrao pmCost variavel cena', () => {
    const p = inventorPowerById('ajuste-de-mira')!
    expect(p.action).toBe('padrao')
    expect(p.pmCost).toBe('variavel')
    expect(p.uses).toBe('cena')
  })

  it('Autômato Prototipado: padrao 2 PM', () => {
    const p = inventorPowerById('automato-prototipado')!
    expect(p.action).toBe('padrao')
    expect(p.pmCost).toBe(2)
  })

  it('Catalisador Instável: completa 3 PM', () => {
    const p = inventorPowerById('catalisador-instavel')!
    expect(p.action).toBe('completa')
    expect(p.pmCost).toBe(3)
  })

  it('Chutes e Palavrões: livre 1 PM rodada', () => {
    const p = inventorPowerById('chutes-e-palavroes')!
    expect(p.action).toBe('livre')
    expect(p.pmCost).toBe(1)
    expect(p.uses).toBe('rodada')
  })

  it('Maestria em Perícia: livre 1 PM', () => {
    const p = inventorPowerById('maestria-em-pericia')!
    expect(p.action).toBe('livre')
    expect(p.pmCost).toBe(1)
  })

  it('Oficina de Campo: varia 2 PM dia', () => {
    const p = inventorPowerById('oficina-de-campo')!
    expect(p.action).toBe('varia')
    expect(p.pmCost).toBe(2)
    expect(p.uses).toBe('dia')
  })

  it('Pedra de Amolar: movimento pmCost variavel cena', () => {
    const p = inventorPowerById('pedra-de-amolar')!
    expect(p.action).toBe('movimento')
    expect(p.pmCost).toBe('variavel')
    expect(p.uses).toBe('cena')
  })
})

describe('inventorPowerById', () => {
  it('miss retorna undefined', () => {
    expect(inventorPowerById('inexistente')).toBeUndefined()
  })
})

describe('inventorElectives', () => {
  it('retorna todos', () => {
    expect(inventorElectives()).toBe(INVENTOR_ELECTIVES)
  })
})

describe('activeInventorPowers', () => {
  it('exclui passivos', () => {
    for (const p of activeInventorPowers()) {
      expect(p.action).not.toBe('passivo')
    }
  })

  it('exatamente 7 poderes ativos', () => {
    expect(activeInventorPowers().length).toBe(7)
  })

  it('IDs corretos', () => {
    const ids = activeInventorPowers().map((p) => p.id).sort()
    expect(ids).toEqual([
      'ajuste-de-mira',
      'automato-prototipado',
      'catalisador-instavel',
      'chutes-e-palavroes',
      'maestria-em-pericia',
      'oficina-de-campo',
      'pedra-de-amolar',
    ])
  })
})
