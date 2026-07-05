import { describe, expect, it } from 'vitest'
import {
  COLD_EXTREME_C,
  COLD_THRESHOLD_C,
  HEAT_EXTREME_C,
  HEAT_THRESHOLD_C,
  PRECIPITATIONS,
  PRECIPITATION_TABLE,
  WINDS,
  WIND_TABLE,
  heatColdCdAfter,
  precipitationRow,
  temperatureExposure,
  windRow,
} from '../weather-cd'

/**
 * PDF Cap 6 p267 — "Clima". Three subsystems: temperature exposure,
 * precipitation, wind.
 */

describe('temperature thresholds (p267)', () => {
  it('constants match book text', () => {
    expect(HEAT_THRESHOLD_C).toBe(50)
    expect(HEAT_EXTREME_C).toBe(60)
    expect(COLD_THRESHOLD_C).toBe(-10)
    expect(COLD_EXTREME_C).toBe(-20)
  })
})

describe('temperatureExposure (p267)', () => {
  it('20°C is safe', () => {
    expect(temperatureExposure(20).kind).toBe('safe')
  })

  it('exactly 50°C is safe (threshold is strict >)', () => {
    expect(temperatureExposure(50).kind).toBe('safe')
  })

  it('55°C → hot, non-extreme (test per day)', () => {
    const e = temperatureExposure(55)
    expect(e.kind).toBe('hot')
    if (e.kind === 'hot') {
      expect(e.extreme).toBe(false)
      expect(e.damageType).toBe('fogo')
    }
  })

  it('60°C → hot extreme (test per minute)', () => {
    const e = temperatureExposure(60)
    expect(e.kind).toBe('hot')
    if (e.kind === 'hot') expect(e.extreme).toBe(true)
  })

  it('exactly -10°C is safe (threshold is strict <)', () => {
    expect(temperatureExposure(-10).kind).toBe('safe')
  })

  it('-15°C → cold, non-extreme', () => {
    const e = temperatureExposure(-15)
    expect(e.kind).toBe('cold')
    if (e.kind === 'cold') {
      expect(e.extreme).toBe(false)
      expect(e.damageType).toBe('frio')
    }
  })

  it('-20°C → cold extreme (per minute)', () => {
    const e = temperatureExposure(-20)
    expect(e.kind).toBe('cold')
    if (e.kind === 'cold') expect(e.extreme).toBe(true)
  })

  it('base CD is 15 for both hot and cold', () => {
    const hot = temperatureExposure(55)
    const cold = temperatureExposure(-15)
    if (hot.kind === 'hot') expect(hot.baseCd).toBe(15)
    if (cold.kind === 'cold') expect(cold.baseCd).toBe(15)
  })
})

describe('heatColdCdAfter (p267 — "CD 15 +1 por teste anterior")', () => {
  it('starts at 15', () => {
    expect(heatColdCdAfter(0)).toBe(15)
  })

  it('grows by 1 per prior test', () => {
    expect(heatColdCdAfter(1)).toBe(16)
    expect(heatColdCdAfter(5)).toBe(20)
    expect(heatColdCdAfter(10)).toBe(25)
  })

  it('rejects negatives / non-integers', () => {
    expect(() => heatColdCdAfter(-1)).toThrow()
    expect(() => heatColdCdAfter(1.5)).toThrow()
  })
})

// ─── Precipitation ──────────────────────────────────────────────────

describe('PRECIPITATIONS (p267)', () => {
  it('lists 4 kinds in book order', () => {
    expect(PRECIPITATIONS).toEqual(['chuva', 'granizo', 'neve', 'tempestade'])
  })

  it('table matches enum 1:1', () => {
    expect(PRECIPITATION_TABLE.map((r) => r.kind)).toEqual([...PRECIPITATIONS])
  })
})

describe('precipitationRow (p267)', () => {
  it('chuva: −5 Percepção + vento forte + sem efeito por rodada', () => {
    const r = precipitationRow('chuva')
    expect(r.perceptionPenalty).toBe(-5)
    expect(r.windEquivalent).toBe('vento-forte')
    expect(r.perRoundEffect.kind).toBe('none')
  })

  it('granizo: como chuva + 1 impacto/rodada', () => {
    const r = precipitationRow('granizo')
    expect(r.perceptionPenalty).toBe(-5)
    expect(r.windEquivalent).toBe('vento-forte')
    expect(r.perRoundEffect.kind).toBe('impact')
  })

  it('neve: como chuva + terreno difícil', () => {
    const r = precipitationRow('neve')
    expect(r.perRoundEffect.kind).toBe('difficult-terrain')
  })

  it('tempestade: −10 Percepção + vendaval + 10% raio 8d10', () => {
    const r = precipitationRow('tempestade')
    expect(r.perceptionPenalty).toBe(-10)
    expect(r.windEquivalent).toBe('vendaval')
    if (r.perRoundEffect.kind === 'lightning-strike') {
      expect(r.perRoundEffect.chancePct).toBe(10)
      expect(r.perRoundEffect.damage).toBe('8d10')
      expect(r.perRoundEffect.damageType).toBe('eletricidade')
    } else {
      throw new Error('expected lightning-strike')
    }
  })
})

// ─── Wind ───────────────────────────────────────────────────────────

describe('WINDS (p267)', () => {
  it('lists 4 kinds in ascending intensity', () => {
    expect(WINDS).toEqual(['vento-forte', 'vendaval', 'furacao', 'tornado'])
  })

  it('table matches enum 1:1', () => {
    expect(WIND_TABLE.map((r) => r.kind)).toEqual([...WINDS])
  })
})

describe('windRow (p267)', () => {
  it('vento-forte: −2 ranged, 50% chance apaga chamas', () => {
    const r = windRow('vento-forte')
    expect(r.rangedAttackPenalty).toBe(-2)
    expect(r.chamas).toBe('50pct-por-rodada-apaga')
    expect(r.knockdown).toBeNull()
  })

  it('vendaval: −5 ranged, apaga chamas', () => {
    const r = windRow('vendaval')
    expect(r.rangedAttackPenalty).toBe(-5)
    expect(r.chamas).toBe('apaga')
    expect(r.knockdown).toBeNull()
  })

  it('furacao: ranged impossivel + Fort CD 15 (Médias-) → cai + 1d4×1,5m direção-do-vento', () => {
    const r = windRow('furacao')
    expect(r.rangedAttackPenalty).toBe('impossivel')
    if (r.knockdown) {
      expect(r.knockdown.fortCd).toBe(15)
      expect(r.knockdown.maxAffectedSize).toBe('medio')
      expect(r.knockdown.dragDice).toBe('1d4')
      expect(r.knockdown.dragDirection).toBe('direcao-do-vento')
      expect(r.knockdown.damagePer1_5m).toBe('1d6')
    } else {
      throw new Error('furacao must have knockdown')
    }
  })

  it('tornado: ranged impossivel + Fort CD 25 (Grandes-) → cai + 1d12×1,5m aleatoria', () => {
    const r = windRow('tornado')
    expect(r.rangedAttackPenalty).toBe('impossivel')
    if (r.knockdown) {
      expect(r.knockdown.fortCd).toBe(25)
      expect(r.knockdown.maxAffectedSize).toBe('grande')
      expect(r.knockdown.dragDice).toBe('1d12')
      expect(r.knockdown.dragDirection).toBe('aleatoria')
    } else {
      throw new Error('tornado must have knockdown')
    }
  })

  it('ranged penalty is monotonically worsening', () => {
    const rows = WINDS.map((k) => windRow(k))
    expect(rows[0]!.rangedAttackPenalty).toBe(-2)
    expect(rows[1]!.rangedAttackPenalty).toBe(-5)
    expect(rows[2]!.rangedAttackPenalty).toBe('impossivel')
    expect(rows[3]!.rangedAttackPenalty).toBe('impossivel')
  })
})
