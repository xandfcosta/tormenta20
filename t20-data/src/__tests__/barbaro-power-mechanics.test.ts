import { describe, expect, it } from 'vitest'
import {
  BARBARO_ELECTIVES,
  activeBarbaroPowers,
  barbaroElectives,
  barbaroPowerById,
  furiaOnlyPowers,
} from '../barbaro-power-mechanics'

/**
 * PDF Cap 1 Bárbaro p41-42 (Poderes Eletivos). Pinned:
 *  - 20 eletivos
 *  - 4 requiresFuria: Espírito Inquebrável, Frenesi, Fúria Raivosa,
 *    Sangue dos Inimigos
 *  - 2 pmCost 'variavel': Totem Espiritual, Vigor Primal
 */

describe('BARBARO_ELECTIVES — shape', () => {
  it('20 eletivos total', () => {
    expect(BARBARO_ELECTIVES.length).toBe(20)
  })

  it('frozen', () => {
    expect(Object.isFrozen(BARBARO_ELECTIVES)).toBe(true)
  })

  it('bookPage entre 41 e 42', () => {
    for (const p of BARBARO_ELECTIVES) {
      expect(p.bookPage).toBeGreaterThanOrEqual(41)
      expect(p.bookPage).toBeLessThanOrEqual(42)
    }
  })

  it('IDs únicos', () => {
    const ids = BARBARO_ELECTIVES.map((p) => p.id)
    expect(new Set(ids).size).toBe(ids.length)
  })

  it('actions são values válidos', () => {
    const valid = new Set([
      'padrao',
      'movimento',
      'livre',
      'reacao',
      'gratuita',
      'completa',
      'passivo',
      'varia',
    ])
    for (const p of BARBARO_ELECTIVES) {
      expect(valid.has(p.action)).toBe(true)
    }
  })
})

describe('BARBARO_ELECTIVES — pinned entries', () => {
  it('Brado Assustador: movimento, 1 PM, cena', () => {
    const p = barbaroPowerById('brado-assustador')!
    expect(p.action).toBe('movimento')
    expect(p.pmCost).toBe(1)
    expect(p.uses).toBe('cena')
  })

  it('Frenesi: livre, 2 PM, rodada, requer Fúria', () => {
    const p = barbaroPowerById('frenesi')!
    expect(p.action).toBe('livre')
    expect(p.pmCost).toBe(2)
    expect(p.uses).toBe('rodada')
    expect(p.requiresFuria).toBe(true)
  })

  it('Vigor Primal: movimento, pmCost variavel', () => {
    const p = barbaroPowerById('vigor-primal')!
    expect(p.action).toBe('movimento')
    expect(p.pmCost).toBe('variavel')
  })

  it('Totem Espiritual: varia, pmCost variavel', () => {
    const p = barbaroPowerById('totem-espiritual')!
    expect(p.action).toBe('varia')
    expect(p.pmCost).toBe('variavel')
  })

  it('Força Indomável: livre, 1 PM, ilimitado', () => {
    const p = barbaroPowerById('forca-indomavel')!
    expect(p.action).toBe('livre')
    expect(p.pmCost).toBe(1)
    expect(p.uses).toBeNull()
  })

  it('Ímpeto: livre, 1 PM', () => {
    const p = barbaroPowerById('impeto')!
    expect(p.action).toBe('livre')
    expect(p.pmCost).toBe(1)
  })

  it('Superstição: passivo (resistência a magia +5)', () => {
    const p = barbaroPowerById('supersticao')!
    expect(p.action).toBe('passivo')
  })
})

describe('barbaroPowerById', () => {
  it('miss retorna undefined', () => {
    expect(barbaroPowerById('inexistente')).toBeUndefined()
  })
})

describe('barbaroElectives', () => {
  it('retorna todos', () => {
    expect(barbaroElectives()).toBe(BARBARO_ELECTIVES)
  })
})

describe('furiaOnlyPowers', () => {
  it('exatamente 4 poderes exigem Fúria', () => {
    expect(furiaOnlyPowers().length).toBe(4)
  })

  it('inclui Frenesi, Fúria Raivosa, Espírito Inquebrável, Sangue dos Inimigos', () => {
    const ids = furiaOnlyPowers().map((p) => p.id).sort()
    expect(ids).toEqual([
      'espirito-inquebravel',
      'frenesi',
      'furia-raivosa',
      'sangue-dos-inimigos',
    ])
  })
})

describe('activeBarbaroPowers', () => {
  it('exclui passivos', () => {
    for (const p of activeBarbaroPowers()) {
      expect(p.action).not.toBe('passivo')
    }
  })

  it('inclui Brado Assustador, Frenesi, Fúria Raivosa, Força Indomável, Golpe Poderoso, Ímpeto, Totem Espiritual, Vigor Primal', () => {
    const ids = activeBarbaroPowers().map((p) => p.id).sort()
    expect(ids).toEqual([
      'brado-assustador',
      'forca-indomavel',
      'frenesi',
      'furia-raivosa',
      'golpe-poderoso',
      'impeto',
      'totem-espiritual',
      'vigor-primal',
    ])
  })
})
