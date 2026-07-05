import { describe, expect, it } from 'vitest'
import {
  DOOR_FORCE_BREAK_CRITICAL_FAIL_DAMAGE,
  DOOR_FORCE_BREAK_CRITICAL_FAIL_MARGIN,
  MOVING_OBJECT_DEFENSE_BONUS,
  OBJECT_PV_SIZE_MULTIPLIER,
  OBJECT_STATS,
  doorForceBreakOutcome,
  movingObjectDefenseBonus,
  objectStatById,
  objectsByCategory,
  scaledObjectPv,
} from '../object-statistics'

/**
 * PDF livro p239 (Tabela 5-4) + p265 (Tabela 6-3 Portas + inline) + p268 (árvores).
 */

describe('Constantes', () => {
  it('multiplicador de PV por escala', () => {
    expect(Object.isFrozen(OBJECT_PV_SIZE_MULTIPLIER)).toBe(true)
    expect(OBJECT_PV_SIZE_MULTIPLIER.reduzido).toBe(0.5)
    expect(OBJECT_PV_SIZE_MULTIPLIER.comum).toBe(1)
    expect(OBJECT_PV_SIZE_MULTIPLIER.aumentado).toBe(2)
    expect(OBJECT_PV_SIZE_MULTIPLIER.gigante).toBe(5)
  })

  it('bônus de Defesa em movimento +5', () => {
    expect(MOVING_OBJECT_DEFENSE_BONUS).toBe(5)
  })

  it('falha arrombar por 5+ sofre 1d6', () => {
    expect(DOOR_FORCE_BREAK_CRITICAL_FAIL_DAMAGE).toBe('1d6')
    expect(DOOR_FORCE_BREAK_CRITICAL_FAIL_MARGIN).toBe(5)
  })
})

describe('OBJECT_STATS — shape', () => {
  it('frozen', () => {
    expect(Object.isFrozen(OBJECT_STATS)).toBe(true)
  })

  it('total: 11 gerais + 10 arma/armadura + 5 portas + 9 terreno = 35', () => {
    expect(OBJECT_STATS.length).toBe(35)
  })

  it('ids únicos', () => {
    const ids = OBJECT_STATS.map((o) => o.id)
    expect(new Set(ids).size).toBe(ids.length)
  })
})

describe('Tabela 5-4 gerais — spot checks verbatim (p239)', () => {
  it('pergaminho: Minúsculo Def 15 RD 0 PV 1', () => {
    const o = objectStatById('pergaminho')
    expect(o.size).toBe('Minúsculo')
    expect(o.defense).toBe(15)
    expect(o.rd).toBe(0)
    expect(o.pv).toBe(1)
  })

  it('corrente: RD 10 PV 2', () => {
    const o = objectStatById('corrente')
    expect(o.rd).toBe(10)
    expect(o.pv).toBe(2)
  })

  it('celeiro: Colossal Def 0 RD 5 PV 200', () => {
    const o = objectStatById('celeiro')
    expect(o.size).toBe('Colossal')
    expect(o.defense).toBe(0)
    expect(o.pv).toBe(200)
  })

  it('porta-ferro-p239: RD 10 PV 100', () => {
    const o = objectStatById('porta-ferro-p239')
    expect(o.rd).toBe(10)
    expect(o.pv).toBe(100)
  })
})

describe('Tabela 5-4 armas/armaduras — spot checks (p239)', () => {
  it('arma leve de metal: RD 10 PV 2', () => {
    const o = objectStatById('arma-leve-metal')
    expect(o.rd).toBe(10)
    expect(o.pv).toBe(2)
  })

  it('escudo pesado: RD 10 PV 20', () => {
    const o = objectStatById('escudo-pesado')
    expect(o.rd).toBe(10)
    expect(o.pv).toBe(20)
  })

  it('armadura pesada: RD 10 PV 40', () => {
    const o = objectStatById('armadura-pesada')
    expect(o.rd).toBe(10)
    expect(o.pv).toBe(40)
  })
})

describe('Tabela 6-3 Portas — spot checks verbatim (p265)', () => {
  it('porta-madeira: RD 5 PV 20 CD 15', () => {
    const o = objectStatById('porta-madeira')
    expect(o.rd).toBe(5)
    expect(o.pv).toBe(20)
    expect(o.cdForceBreak).toBe(15)
  })

  it('porta-madeira-reforcada: PV 30 CD 20', () => {
    const o = objectStatById('porta-madeira-reforcada')
    expect(o.pv).toBe(30)
    expect(o.cdForceBreak).toBe(20)
  })

  it('porta-pedra: RD 8 PV 100 CD 25', () => {
    const o = objectStatById('porta-pedra')
    expect(o.rd).toBe(8)
    expect(o.pv).toBe(100)
    expect(o.cdForceBreak).toBe(25)
  })

  it('porta-ferro: RD 10 PV 100 CD 25', () => {
    const o = objectStatById('porta-ferro')
    expect(o.rd).toBe(10)
    expect(o.pv).toBe(100)
    expect(o.cdForceBreak).toBe(25)
  })

  it('grade: RD 10 PV 60 CD 20', () => {
    const o = objectStatById('porta-grade')
    expect(o.rd).toBe(10)
    expect(o.pv).toBe(60)
    expect(o.cdForceBreak).toBe(20)
  })
})

describe('Árvores e Pilares — p268/p265', () => {
  it('árvore estreita: RD 5 PV 100', () => {
    const o = objectStatById('arvore-estreita')
    expect(o.rd).toBe(5)
    expect(o.pv).toBe(100)
    expect(o.bookPage).toBe(268)
    expect(o.cdAtletismo).toBe(15)
  })

  it('árvore larga: RD 5 PV 500', () => {
    expect(objectStatById('arvore-larga').pv).toBe(500)
  })

  it('pilar estreito: RD 8 PV 100', () => {
    const o = objectStatById('pilar-estreito')
    expect(o.rd).toBe(8)
    expect(o.pv).toBe(100)
    expect(o.bookPage).toBe(265)
  })

  it('pilar largo: RD 8 PV 500', () => {
    expect(objectStatById('pilar-largo').pv).toBe(500)
  })
})

describe('Paredes / tapeçaria / altar (p265)', () => {
  it('parede alvenaria: RD 8 PV 200 CD Atletismo 20', () => {
    const o = objectStatById('parede-alvenaria')
    expect(o.rd).toBe(8)
    expect(o.pv).toBe(200)
    expect(o.cdAtletismo).toBe(20)
  })

  it('parede pedra bruta: PV 500 CD Atletismo 15', () => {
    const o = objectStatById('parede-pedra-bruta')
    expect(o.pv).toBe(500)
    expect(o.cdAtletismo).toBe(15)
  })

  it('tapeçaria: RD 0 PV 10', () => {
    const o = objectStatById('tapecaria')
    expect(o.rd).toBe(0)
    expect(o.pv).toBe(10)
  })

  it('altar típico: RD 8 PV 200', () => {
    const o = objectStatById('altar-tipico')
    expect(o.rd).toBe(8)
    expect(o.pv).toBe(200)
  })
})

describe('objectsByCategory', () => {
  it('door: 5 portas', () => {
    expect(objectsByCategory('door').length).toBe(5)
  })

  it('tree: 2 árvores', () => {
    expect(objectsByCategory('tree').length).toBe(2)
  })

  it('pillar: 2 pilares', () => {
    expect(objectsByCategory('pillar').length).toBe(2)
  })

  it('wall: 3 paredes', () => {
    expect(objectsByCategory('wall').length).toBe(3)
  })
})

describe('objectStatById — throws on unknown', () => {
  it('lança se id desconhecido', () => {
    expect(() => objectStatById('foo-bar')).toThrow(/unknown id/)
  })
})

describe('scaledObjectPv', () => {
  it('comum → PV base', () => {
    expect(scaledObjectPv(100, 'comum')).toBe(100)
  })

  it('reduzido → floor(50)', () => {
    expect(scaledObjectPv(100, 'reduzido')).toBe(50)
  })

  it('aumentado → 200', () => {
    expect(scaledObjectPv(100, 'aumentado')).toBe(200)
  })

  it('gigante → 500', () => {
    expect(scaledObjectPv(100, 'gigante')).toBe(500)
  })

  it('reduzido de PV ímpar → floor', () => {
    expect(scaledObjectPv(11, 'reduzido')).toBe(5)
  })

  it('PV negativo lança', () => {
    expect(() => scaledObjectPv(-1, 'comum')).toThrow(/basePv must be ≥ 0/)
  })
})

describe('movingObjectDefenseBonus', () => {
  it('em movimento → +5', () => {
    expect(movingObjectDefenseBonus(true)).toBe(5)
  })

  it('parado → 0', () => {
    expect(movingObjectDefenseBonus(false)).toBe(0)
  })
})

describe('doorForceBreakOutcome — p265', () => {
  it('sucesso → broken', () => {
    expect(doorForceBreakOutcome(15, 15)).toBe('broken')
    expect(doorForceBreakOutcome(20, 15)).toBe('broken')
  })

  it('falha por < 5 → failed', () => {
    expect(doorForceBreakOutcome(14, 15)).toBe('failed')
    expect(doorForceBreakOutcome(11, 15)).toBe('failed')
  })

  it('falha por ≥ 5 → failed-and-hurt (1d6 impacto)', () => {
    expect(doorForceBreakOutcome(10, 15)).toBe('failed-and-hurt')
    expect(doorForceBreakOutcome(15, 25)).toBe('failed-and-hurt')
  })
})
