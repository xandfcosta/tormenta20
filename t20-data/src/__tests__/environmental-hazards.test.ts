import { describe, expect, it } from 'vitest'
import {
  BREATH_HOLD_ROUNDS_BASE,
  ENVIRONMENTAL_HAZARDS,
  FUMACA_BASE_CD,
  QUEDA_DAMAGE_PER_INTERVAL_METERS,
  QUEDA_MAX_DAMAGE_DICE,
  QUEDA_WATER_REDUCTION_DICE,
  SUFOCAMENTO_BASE_CD,
  TORMENTA_BASE_CD,
  breathHoldRounds,
  escalatingFortitudeCd,
  fallDamageDice,
  hazardById,
  hazardsByCategory,
  type HazardCategory,
} from '../environmental-hazards'

/**
 * PDF Cap 7 Aventura, p317-319 (+ Cap 6 Mestre p267-269). Pinned:
 *  - 14 perigos ambientais.
 *  - T20 NÃO tem afogamento separado (cai em Sufocamento).
 *  - T20 NÃO tem queimadura separada (cai em Fogo).
 *  - Queda: 1d6 por 1,5m, cap 40d6, água -4d6.
 *  - Prender respiração: 1 + Constituição rodadas.
 *  - Fortitude CD-base 15 escala +1 por teste anterior.
 *  - Tormenta CD-base 25 escala +2 por dia.
 */

const ALL_CATEGORIES: readonly HazardCategory[] = [
  'sufocamento',
  'fome-sede',
  'temperatura',
  'queda',
  'fogo',
  'meio-ambiente',
]

describe('ENVIRONMENTAL_HAZARDS — shape & invariants', () => {
  it('catálogo tem exatamente 14 perigos', () => {
    expect(ENVIRONMENTAL_HAZARDS.length).toBe(14)
  })

  it('todos ids únicos', () => {
    const ids = ENVIRONMENTAL_HAZARDS.map((h) => h.id)
    expect(new Set(ids).size).toBe(ids.length)
  })

  it('toda category na union conhecida', () => {
    for (const h of ENVIRONMENTAL_HAZARDS) {
      expect(ALL_CATEGORIES).toContain(h.category)
    }
  })

  it('catálogo frozen', () => {
    expect(Object.isFrozen(ENVIRONMENTAL_HAZARDS)).toBe(true)
  })

  it('NÃO contém afogamento separado (T20 usa Sufocamento)', () => {
    expect(hazardById('afogamento')).toBeUndefined()
  })

  it('NÃO contém queimadura separada (T20 usa Fogo)', () => {
    expect(hazardById('queimadura')).toBeUndefined()
  })
})

describe('ENVIRONMENTAL_HAZARDS — pinned canonical entries', () => {
  it('Queda: 1d6 por 1,5m, cap 40d6, água -4d6', () => {
    const h = hazardById('queda')!
    expect(h.damagePerInterval).toMatch(/1d6 de impacto por 1,5m/)
    expect(h.damagePerInterval).toMatch(/40d6/)
    expect(QUEDA_DAMAGE_PER_INTERVAL_METERS).toBe(1.5)
    expect(QUEDA_MAX_DAMAGE_DICE).toBe(40)
    expect(QUEDA_WATER_REDUCTION_DICE).toBe(4)
  })

  it('Sufocamento: Fortitude CD 15, prender respiração 1+Con', () => {
    const h = hazardById('sufocamento')!
    expect(h.saveType).toBe('fortitude')
    expect(h.saveCd).toBe(15)
    expect(h.effect).toMatch(/1 \+ Constituição/)
  })

  it('Tormenta: Vontade CD 25 +2/dia, lefeu imunes', () => {
    const h = hazardById('tormenta')!
    expect(h.saveType).toBe('vontade')
    expect(h.saveCd).toBe(25)
    expect(h.saveCdProgression).toMatch(/\+2/)
    expect(h.effect).toMatch(/Lefeu imunes/)
  })

  it('Fogo: Reflexos CD 15 fixa (não escala)', () => {
    const h = hazardById('fogo')!
    expect(h.saveType).toBe('reflexos')
    expect(h.saveCd).toBe(15)
    expect(h.saveCdProgression).toMatch(/não escalona/)
  })

  it('Lava: 2d6/rodada exposição, 20d6/rodada imersão', () => {
    const h = hazardById('lava')!
    expect(h.damagePerInterval).toMatch(/2d6 fogo por rodada/)
    expect(h.damagePerInterval).toMatch(/20d6/)
  })

  it('Fome e Sede: 1ª fatigado, 2ª exausto, 3ª inconsciente, 4ª morte', () => {
    const h = hazardById('fome-sede')!
    expect(h.effect).toMatch(/1ª falha = fatigado/)
    expect(h.effect).toMatch(/2ª = exausto/)
    expect(h.effect).toMatch(/4ª = letal/)
  })

  it('Fumaça: CD inicial 10', () => {
    const h = hazardById('fumaca')!
    expect(h.saveCd).toBe(10)
    expect(FUMACA_BASE_CD).toBe(10)
  })

  it('Areia Movediça: Sobrevivência CD 25 + Atletismo CD 25', () => {
    const h = hazardById('areia-movedica')!
    expect(h.effect).toMatch(/Sobrevivência CD 25/)
    expect(h.effect).toMatch(/Atletismo CD 25/)
  })

  it('Ácido: 1d6 exposição, 10d6 imersão (persiste 1 rodada)', () => {
    const h = hazardById('acido')!
    expect(h.damagePerInterval).toMatch(/10d6/)
    expect(h.effect).toMatch(/persiste 1 rodada/)
  })
})

describe('hazardsByCategory — distribuição', () => {
  it('fogo: 2 entradas (Fogo + Lava)', () => {
    const ids = hazardsByCategory('fogo').map((h) => h.id).sort()
    expect(ids).toEqual(['fogo', 'lava'])
  })

  it('sufocamento: 2 entradas (Sufocamento + Fumaça)', () => {
    const ids = hazardsByCategory('sufocamento').map((h) => h.id).sort()
    expect(ids).toEqual(['fumaca', 'sufocamento'])
  })

  it('temperatura: 3 entradas (Calor/Frio + Altitude + Gelo)', () => {
    expect(hazardsByCategory('temperatura').length).toBe(3)
  })

  it('queda: apenas Queda', () => {
    expect(hazardsByCategory('queda')).toHaveLength(1)
  })

  it('fome-sede: Fome e Sede + Sono', () => {
    const ids = hazardsByCategory('fome-sede').map((h) => h.id).sort()
    expect(ids).toEqual(['fome-sede', 'sono'])
  })
})

describe('fallDamageDice — fórmula PDF p319', () => {
  it('queda 0m = 0 dado', () => {
    expect(fallDamageDice(0)).toBe(0)
  })

  it('queda 1,5m = 1d6', () => {
    expect(fallDamageDice(1.5)).toBe(1)
  })

  it('queda 6m = 4d6', () => {
    expect(fallDamageDice(6)).toBe(4)
  })

  it('queda 60m = 40d6 (cap PDF)', () => {
    expect(fallDamageDice(60)).toBe(40)
  })

  it('queda 100m = 40d6 (cap clamp)', () => {
    expect(fallDamageDice(100)).toBe(40)
  })

  it('queda 60m em água = 36d6 (-4d6)', () => {
    expect(fallDamageDice(60, true)).toBe(36)
  })

  it('queda 6m em água = 0 dado (-4d6 ≤ 0)', () => {
    expect(fallDamageDice(6, true)).toBe(0)
  })

  it('queda 9m em água = 2d6 (6 - 4)', () => {
    expect(fallDamageDice(9, true)).toBe(2)
  })
})

describe('breathHoldRounds — PDF p319', () => {
  it('base 1 + 0 mod = 1 rodada', () => {
    expect(breathHoldRounds(0)).toBe(BREATH_HOLD_ROUNDS_BASE)
  })

  it('Constituição +2 = 3 rodadas', () => {
    expect(breathHoldRounds(2)).toBe(3)
  })

  it('Constituição +5 = 6 rodadas', () => {
    expect(breathHoldRounds(5)).toBe(6)
  })
})

describe('escalatingFortitudeCd', () => {
  it('Fortitude CD-base 15 + 0 testes anteriores = 15', () => {
    expect(escalatingFortitudeCd(SUFOCAMENTO_BASE_CD, 0)).toBe(15)
  })

  it('Fortitude CD-base 15 + 3 testes anteriores = 18', () => {
    expect(escalatingFortitudeCd(SUFOCAMENTO_BASE_CD, 3)).toBe(18)
  })

  it('Fumaça CD-base 10 + 5 testes = 15', () => {
    expect(escalatingFortitudeCd(FUMACA_BASE_CD, 5)).toBe(15)
  })
})

describe('Constantes — valores PDF', () => {
  it('SUFOCAMENTO_BASE_CD = 15', () => {
    expect(SUFOCAMENTO_BASE_CD).toBe(15)
  })

  it('FUMACA_BASE_CD = 10', () => {
    expect(FUMACA_BASE_CD).toBe(10)
  })

  it('TORMENTA_BASE_CD = 25', () => {
    expect(TORMENTA_BASE_CD).toBe(25)
  })

  it('BREATH_HOLD_ROUNDS_BASE = 1', () => {
    expect(BREATH_HOLD_ROUNDS_BASE).toBe(1)
  })
})
