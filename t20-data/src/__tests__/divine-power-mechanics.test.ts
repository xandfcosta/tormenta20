import { describe, expect, it } from 'vitest'
import { DEUS_BY_ID, DEUSES } from '../abilities/deuses'
import {
  DIVINE_POWERS,
  activeDivinePowers,
  divinePowersOf,
  spellGrantingDivinePowers,
} from '../divine-power-mechanics'

/**
 * PDF Cap 2 Poderes p132-135 (Poderes Concedidos). Pinned:
 *  - 20 deuses maiores × 4 poderes = 80 entries.
 *  - Cada deusId existe em DEUSES.
 *  - Nomes de poder em DIVINE_POWERS correspondem aos listados em
 *    Deus.poderesConcedidos.
 */

const MAJOR_DEUSES = DEUSES.filter((d) => d.major && d.poderesConcedidos)

describe('DIVINE_POWERS — shape', () => {
  it('80 entries (20 deuses × 4 poderes)', () => {
    expect(DIVINE_POWERS.length).toBe(80)
  })

  it('frozen', () => {
    expect(Object.isFrozen(DIVINE_POWERS)).toBe(true)
  })

  it('cada deusId existe em DEUSES', () => {
    for (const p of DIVINE_POWERS) {
      expect(DEUS_BY_ID[p.deusId]).toBeDefined()
    }
  })

  it('bookPage entre 132 e 135', () => {
    for (const p of DIVINE_POWERS) {
      expect(p.bookPage).toBeGreaterThanOrEqual(132)
      expect(p.bookPage).toBeLessThanOrEqual(135)
    }
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
    for (const p of DIVINE_POWERS) {
      expect(valid.has(p.action)).toBe(true)
    }
  })
})

describe('DIVINE_POWERS — parity com deuses.ts', () => {
  it('cada deus maior tem exatamente 4 divine powers', () => {
    for (const d of MAJOR_DEUSES) {
      const powers = divinePowersOf(d.id)
      expect(powers.length).toBe(4)
    }
  })

  it('nomes divine powers batem com poderesConcedidos', () => {
    for (const d of MAJOR_DEUSES) {
      const powers = divinePowersOf(d.id).map((p) => p.name).sort()
      const expected = [...d.poderesConcedidos!].sort()
      expect(powers).toEqual(expected)
    }
  })
})

describe('DIVINE_POWERS — pinned entries', () => {
  it('Aharadak: Percepção Temporal livre/3PM/cena', () => {
    const p = DIVINE_POWERS.find(
      (p) => p.deusId === 'aharadak' && p.name === 'Percepção Temporal',
    )!
    expect(p.action).toBe('livre')
    expect(p.pmCost).toBe(3)
    expect(p.uses).toBe('cena')
  })

  it('Lena: 4 poderes passivos (Lena não ativa)', () => {
    for (const p of divinePowersOf('lena')) {
      expect(p.action).toBe('passivo')
      expect(p.pmCost).toBe(0)
    }
  })

  it('Wynna: Escudo Mágico reacao/0 PM (gatilho ao conjurar)', () => {
    const p = DIVINE_POWERS.find(
      (p) => p.deusId === 'wynna' && p.name === 'Escudo Mágico',
    )!
    expect(p.action).toBe('reacao')
    expect(p.pmCost).toBe(0)
  })

  it('Tenebra: Zumbificar completa/3PM', () => {
    const p = DIVINE_POWERS.find(
      (p) => p.deusId === 'tenebra' && p.name === 'Zumbificar',
    )!
    expect(p.action).toBe('completa')
    expect(p.pmCost).toBe(3)
  })

  it('Thyatis: Dom da Ressurreição completa/variavel/cena', () => {
    const p = DIVINE_POWERS.find(
      (p) => p.deusId === 'thyatis' && p.name === 'Dom da Ressurreição',
    )!
    expect(p.action).toBe('completa')
    expect(p.pmCost).toBe('variavel')
    expect(p.uses).toBe('cena')
  })

  it('Hyninn: Forma de Macaco completa/2PM/cena', () => {
    const p = DIVINE_POWERS.find(
      (p) => p.deusId === 'hyninn' && p.name === 'Forma de Macaco',
    )!
    expect(p.action).toBe('completa')
    expect(p.pmCost).toBe(2)
    expect(p.uses).toBe('cena')
  })

  it('Azgher: Fulgor Solar reacao/1PM/rodada', () => {
    const p = DIVINE_POWERS.find(
      (p) => p.deusId === 'azgher' && p.name === 'Fulgor Solar',
    )!
    expect(p.action).toBe('reacao')
    expect(p.uses).toBe('rodada')
  })

  it('Khalmyr: Reparar Injustiça reacao/2PM/rodada', () => {
    const p = DIVINE_POWERS.find(
      (p) => p.deusId === 'khalmyr' && p.name === 'Reparar Injustiça',
    )!
    expect(p.action).toBe('reacao')
    expect(p.uses).toBe('rodada')
  })
})

describe('divinePowersOf', () => {
  it('cobre todos os deuses maiores', () => {
    for (const d of MAJOR_DEUSES) {
      expect(divinePowersOf(d.id).length).toBe(4)
    }
  })

  it('miss retorna array vazio', () => {
    expect(divinePowersOf('inexistente')).toEqual([])
  })

  it('sentinels (panteao/paladino-do-bem) retornam vazio', () => {
    expect(divinePowersOf('panteao')).toEqual([])
    expect(divinePowersOf('paladino-do-bem')).toEqual([])
  })
})

describe('activeDivinePowers', () => {
  it('exclui passivos', () => {
    for (const p of activeDivinePowers('khalmyr')) {
      expect(p.action).not.toBe('passivo')
    }
  })

  it('Lena tem 0 ativos (100% passivos)', () => {
    expect(activeDivinePowers('lena').length).toBe(0)
  })

  it('Aharadak tem 2 ativos (Extase + Percepção)', () => {
    const active = activeDivinePowers('aharadak')
    expect(active.length).toBe(2)
  })

  it('Khalmyr tem 3 ativos (só Coragem Total é passivo)', () => {
    const active = activeDivinePowers('khalmyr').map((p) => p.name).sort()
    expect(active).toEqual([
      'Dom da Verdade',
      'Espada Justiceira',
      'Reparar Injustiça',
    ])
  })
})

describe('spellGrantingDivinePowers', () => {
  it('todos têm pmCost variavel', () => {
    for (const p of spellGrantingDivinePowers()) {
      expect(p.pmCost).toBe('variavel')
    }
  })

  it('mais de 10 poderes concedem magia', () => {
    expect(spellGrantingDivinePowers().length).toBeGreaterThan(10)
  })

  it('inclui Dedo Verde (Allihanna → Controlar Plantas)', () => {
    const names = spellGrantingDivinePowers().map((p) => p.name)
    expect(names).toContain('Dedo Verde')
  })

  it('inclui Centelha Mágica (Wynna → 1º círculo)', () => {
    const names = spellGrantingDivinePowers().map((p) => p.name)
    expect(names).toContain('Centelha Mágica')
  })
})
