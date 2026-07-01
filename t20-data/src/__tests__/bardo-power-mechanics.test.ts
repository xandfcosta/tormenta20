import { describe, expect, it } from 'vitest'
import {
  BARDO_ELECTIVES,
  activeBardoPowers,
  bardoElectives,
  bardoPowerById,
  inspiracaoOnlyBardoPowers,
  musicBardoPowers,
} from '../bardo-power-mechanics'

/**
 * PDF Cap 1 Bardo p42-45. Pinned:
 *  - 20 eletivos.
 *  - 4 requiresInspiracao: Arte Mágica, Esgrima Mágica, Golpe Elemental,
 *    Golpe Mágico.
 *  - 4 isMusic: Balada Fascinante, Canção Assustadora, Melodia Curativa,
 *    Melodia Restauradora.
 *  - Paródia: reação, 1/rodada, pmCost variavel (1 + custo magia).
 *  - Prestidigitação: padrão, pmCost variavel (custo da magia).
 */

describe('BARDO_ELECTIVES — shape', () => {
  it('20 eletivos total', () => {
    expect(BARDO_ELECTIVES.length).toBe(20)
  })

  it('frozen', () => {
    expect(Object.isFrozen(BARDO_ELECTIVES)).toBe(true)
  })

  it('bookPage entre 44 e 45', () => {
    for (const p of BARDO_ELECTIVES) {
      expect(p.bookPage).toBeGreaterThanOrEqual(44)
      expect(p.bookPage).toBeLessThanOrEqual(45)
    }
  })

  it('IDs únicos', () => {
    const ids = BARDO_ELECTIVES.map((p) => p.id)
    expect(new Set(ids).size).toBe(ids.length)
  })
})

describe('BARDO_ELECTIVES — pinned entries', () => {
  it('Balada Fascinante: padrão, 1 PM, Música', () => {
    const p = bardoPowerById('musica-balada-fascinante')!
    expect(p.action).toBe('padrao')
    expect(p.pmCost).toBe(1)
    expect(p.isMusic).toBe(true)
  })

  it('Melodia Curativa: padrão, 1 PM base, Música', () => {
    const p = bardoPowerById('musica-melodia-curativa')!
    expect(p.action).toBe('padrao')
    expect(p.pmCost).toBe(1)
    expect(p.isMusic).toBe(true)
  })

  it('Paródia: reacao, pmCost variavel, 1/rodada', () => {
    const p = bardoPowerById('parodia')!
    expect(p.action).toBe('reacao')
    expect(p.pmCost).toBe('variavel')
    expect(p.uses).toBe('rodada')
  })

  it('Prestidigitação: padrão, pmCost variavel', () => {
    const p = bardoPowerById('prestidigitacao')!
    expect(p.action).toBe('padrao')
    expect(p.pmCost).toBe('variavel')
  })

  it('Dança das Lâminas: livre, 1 PM', () => {
    const p = bardoPowerById('danca-das-laminas')!
    expect(p.action).toBe('livre')
    expect(p.pmCost).toBe(1)
  })

  it('Golpe Elemental: livre, 1 PM, requer Inspiração', () => {
    const p = bardoPowerById('golpe-elemental')!
    expect(p.action).toBe('livre')
    expect(p.pmCost).toBe(1)
    expect(p.requiresInspiracao).toBe(true)
  })

  it('Manipular: padrão, 1 PM', () => {
    const p = bardoPowerById('manipular')!
    expect(p.action).toBe('padrao')
    expect(p.pmCost).toBe(1)
  })

  it('Lendas e Histórias: livre, 1 PM (reroll identify)', () => {
    const p = bardoPowerById('lendas-e-historias')!
    expect(p.action).toBe('livre')
    expect(p.pmCost).toBe(1)
  })
})

describe('bardoPowerById', () => {
  it('miss retorna undefined', () => {
    expect(bardoPowerById('inexistente')).toBeUndefined()
  })
})

describe('bardoElectives', () => {
  it('retorna todos', () => {
    expect(bardoElectives()).toBe(BARDO_ELECTIVES)
  })
})

describe('inspiracaoOnlyBardoPowers', () => {
  it('exatamente 4 poderes exigem Inspiração', () => {
    expect(inspiracaoOnlyBardoPowers().length).toBe(4)
  })

  it('IDs corretos', () => {
    const ids = inspiracaoOnlyBardoPowers().map((p) => p.id).sort()
    expect(ids).toEqual([
      'arte-magica',
      'esgrima-magica',
      'golpe-elemental',
      'golpe-magico',
    ])
  })
})

describe('musicBardoPowers', () => {
  it('exatamente 4 poderes de Música', () => {
    expect(musicBardoPowers().length).toBe(4)
  })

  it('todos começam com "Música:" no nome', () => {
    for (const p of musicBardoPowers()) {
      expect(p.name.startsWith('Música:')).toBe(true)
    }
  })

  it('IDs corretos', () => {
    const ids = musicBardoPowers().map((p) => p.id).sort()
    expect(ids).toEqual([
      'musica-balada-fascinante',
      'musica-cancao-assustadora',
      'musica-melodia-curativa',
      'musica-melodia-restauradora',
    ])
  })
})

describe('activeBardoPowers', () => {
  it('exclui passivos', () => {
    for (const p of activeBardoPowers()) {
      expect(p.action).not.toBe('passivo')
    }
  })
})
