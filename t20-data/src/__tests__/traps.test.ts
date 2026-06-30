import { describe, expect, it } from 'vitest'
import {
  TRAPS,
  TRAP_DISARM_BASE_TIME,
  TRAP_DISARM_CRITICAL_FAILURE_MARGIN,
  TRAP_DISARM_REQUIRES_TRAINED_LADINAGEM,
  magicalTraps,
  resolveDisarm,
  trapById,
  trapsByNd,
  trapsByTrigger,
  type TrapTriggerType,
} from '../traps'

/**
 * PDF Cap 7 Aventura, p317-318. Pinned:
 *  - 23 armadilhas catalogadas (NDs 1/4 a 8).
 *  - Detect skill = Investigação (PDF NÃO usa Percepção para armadilhas).
 *  - Disarm skill = Ladinagem (Apenas Treinada).
 *  - Falha de Sabotar por 5+ ativa (margem crítica).
 *  - ND = ND de criatura para fins de XP.
 *  - Tipo formal único = magica (itálico no PDF).
 */

const ALL_TRIGGERS: readonly TrapTriggerType[] = [
  'pressao',
  'tripwire',
  'proximidade',
  'manipulacao',
  'magica',
]

describe('TRAPS — shape & invariants', () => {
  it('catálogo tem exatamente 23 armadilhas', () => {
    expect(TRAPS.length).toBe(23)
  })

  it('todos ids únicos', () => {
    const ids = TRAPS.map((t) => t.id)
    expect(new Set(ids).size).toBe(ids.length)
  })

  it('toda armadilha usa detectSkill === "investigacao" (NUNCA percepção)', () => {
    for (const t of TRAPS) {
      expect(t.detectSkill).toBe('investigacao')
    }
  })

  it('toda armadilha tem triggerType em union conhecida', () => {
    for (const t of TRAPS) {
      expect(ALL_TRIGGERS).toContain(t.triggerType)
    }
  })

  it('toda armadilha tem bookPage 317 ou 318', () => {
    for (const t of TRAPS) {
      expect([317, 318]).toContain(t.bookPage)
    }
  })

  it('NDs válidos: 0.25 / 0.5 / 1 / 2 / 3 / 4 / 5 / 6 / 8', () => {
    const valid = new Set([0.25, 0.5, 1, 2, 3, 4, 5, 6, 8])
    for (const t of TRAPS) expect(valid.has(t.nd)).toBe(true)
  })

  it('catálogo frozen', () => {
    expect(Object.isFrozen(TRAPS)).toBe(true)
  })
})

describe('TRAPS — system constants per PDF', () => {
  it('TRAP_DISARM_REQUIRES_TRAINED_LADINAGEM = true (p111)', () => {
    expect(TRAP_DISARM_REQUIRES_TRAINED_LADINAGEM).toBe(true)
  })

  it('TRAP_DISARM_BASE_TIME = "1d4 rodadas" (p120)', () => {
    expect(TRAP_DISARM_BASE_TIME).toBe('1d4 rodadas')
  })

  it('TRAP_DISARM_CRITICAL_FAILURE_MARGIN = 5 (p120)', () => {
    expect(TRAP_DISARM_CRITICAL_FAILURE_MARGIN).toBe(5)
  })
})

describe('TRAPS — pinned canonical entries', () => {
  it('Agulha Envenenada: ND 1/4, manipulacao, det 25 / des 20', () => {
    const t = trapById('agulha-envenenada')!
    expect(t.nd).toBe(0.25)
    expect(t.triggerType).toBe('manipulacao')
    expect(t.detectCd).toBe(25)
    expect(t.disarmCd).toBe(20)
  })

  it('Símbolo da Morte: ND 8, mágica, Fortitude CD 30 parcial', () => {
    const t = trapById('simbolo-da-morte')!
    expect(t.nd).toBe(8)
    expect(t.magica).toBe(true)
    expect(t.save.type).toBe('fortitude')
    expect(t.save.cd).toBe(30)
    expect(t.save.effect).toBe('parcial')
  })

  it('Abismo da Morte: ND 8, queda 30m + estacas', () => {
    const t = trapById('abismo-da-morte')!
    expect(t.nd).toBe(8)
    expect(t.magica).toBe(false)
    expect(t.effect).toMatch(/30m/)
  })

  it('Bloco de Pedra: ND 1, pressão, 6d6 impacto', () => {
    const t = trapById('bloco-de-pedra')!
    expect(t.nd).toBe(1)
    expect(t.triggerType).toBe('pressao')
    expect(t.damage).toMatch(/6d6 impacto/)
  })

  it('Arame Farpado: sem teste de resistência (save.type = none)', () => {
    const t = trapById('arame-farpado')!
    expect(t.save.type).toBe('none')
    expect(t.save.effect).toBeNull()
  })

  it('Runa de Proteção: mágica, Reflexos metade', () => {
    const t = trapById('runa-de-protecao')!
    expect(t.magica).toBe(true)
    expect(t.save.type).toBe('reflexos')
    expect(t.save.effect).toBe('metade')
  })

  it('Gás Venenoso: dano recorrente 1d12/rodada por 2d4 rodadas', () => {
    const t = trapById('gas-venenoso')!
    expect(t.damage).toMatch(/1d12 PV de veneno por rodada/)
    expect(t.damage).toMatch(/2d4 rodadas/)
  })
})

describe('magicalTraps — armadilhas com magica=true', () => {
  it('todas as Runas e Símbolos são mágicas', () => {
    const magicas = magicalTraps()
    const ids = magicas.map((t) => t.id)
    expect(ids).toContain('runa-de-protecao')
    expect(ids).toContain('simbolo-do-medo')
    expect(ids).toContain('simbolo-do-sono')
    expect(ids).toContain('simbolo-da-dor')
    expect(ids).toContain('simbolo-do-atordoamento')
    expect(ids).toContain('simbolo-da-insanidade')
    expect(ids).toContain('simbolo-da-morte')
  })

  it('total de armadilhas mágicas = 7 (1 Runa + 6 Símbolos)', () => {
    expect(magicalTraps().length).toBe(7)
  })

  it('armadilhas não-mágicas: 16 (23 - 7)', () => {
    expect(TRAPS.filter((t) => !t.magica).length).toBe(16)
  })
})

describe('trapsByTrigger', () => {
  it('manipulacao: apenas Agulha Envenenada', () => {
    const ids = trapsByTrigger('manipulacao').map((t) => t.id)
    expect(ids).toEqual(['agulha-envenenada'])
  })

  it('magica retorna o mesmo conjunto que magicalTraps()', () => {
    expect(trapsByTrigger('magica').length).toBe(magicalTraps().length)
  })
})

describe('trapsByNd — XP scaling baseado em ND de criatura', () => {
  it('NDs <1 (1/4 e 1/2) têm 5 + 2 = 7 armadilhas', () => {
    const baixos = TRAPS.filter((t) => t.nd < 1).length
    expect(baixos).toBe(7) // 5 ND 1/4 + 2 ND 1/2
  })

  it('ND 8 tem exatamente 2 armadilhas (Abismo + Símbolo da Morte)', () => {
    const ids = trapsByNd(8).map((t) => t.id).sort()
    expect(ids).toEqual(['abismo-da-morte', 'simbolo-da-morte'])
  })

  it('ND 1/4 (0.25) tem 5 entradas', () => {
    expect(trapsByNd(0.25).length).toBe(5)
  })
})

describe('resolveDisarm', () => {
  const t = trapById('bloco-de-pedra')! // disarmCd 20

  it('total ≥ CD = sucesso', () => {
    expect(resolveDisarm(t, 20)).toBe('sucesso')
    expect(resolveDisarm(t, 25)).toBe('sucesso')
  })

  it('total dentro de [CD-4, CD-1] = falha simples', () => {
    expect(resolveDisarm(t, 19)).toBe('falha')
    expect(resolveDisarm(t, 16)).toBe('falha')
  })

  it('total ≤ CD-5 = falha-critica (ativa armadilha)', () => {
    expect(resolveDisarm(t, 15)).toBe('falha-critica')
    expect(resolveDisarm(t, 10)).toBe('falha-critica')
    expect(resolveDisarm(t, 0)).toBe('falha-critica')
  })

  it('Símbolo da Morte (CD 30): margem crítica em 25', () => {
    const sm = trapById('simbolo-da-morte')!
    expect(resolveDisarm(sm, 25)).toBe('falha-critica')
    expect(resolveDisarm(sm, 26)).toBe('falha')
    expect(resolveDisarm(sm, 30)).toBe('sucesso')
  })
})

describe('TRAPS — Investigação CD ≠ Ladinagem CD em algumas entradas', () => {
  it('Agulha Envenenada / Virote / Lâmina na Parede / Pêndulo: detect 25, disarm 20', () => {
    for (const id of [
      'agulha-envenenada',
      'virote',
      'lamina-na-parede',
      'pendulo-de-teto',
    ]) {
      const t = trapById(id)!
      expect(t.detectCd).toBe(25)
      expect(t.disarmCd).toBe(20)
    }
  })

  it('Arame Farpado: detect 10 (mais baixo do catálogo)', () => {
    expect(trapById('arame-farpado')!.detectCd).toBe(10)
  })
})
