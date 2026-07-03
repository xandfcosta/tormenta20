import { describe, expect, it } from 'vitest'
import {
  BAD_WEATHER_MULTIPLIER,
  DIFFICULT_TERRAIN_MULTIPLIER,
  KM_PER_HOUR_PER_METER,
  MARCHA_FORCADA_CD_PROGRESSAO,
  MARCHA_FORCADA_DANO_FALHA,
  MARCHA_FORCADA_FORT_CD_BASE,
  MARCHA_FORCADA_MULTIPLIER,
  TABELA_6_4,
  TRAVEL_HOURS_PER_DAY,
  applyTerrainWeatherModifier,
  kmPerDay,
  kmPerHour,
  kmPerHourForced,
  marchaForcadaFortCd,
  partyKmPerHour,
} from '../travel-marcha'

/**
 * PDF Cap 6 p270. Pinned Tabela 6-4:
 *  - 4,5m → 2,25 km/h / 18 km/dia
 *  - 6m → 3 km/h / 24 km/dia
 *  - 9m → 4,5 km/h / 36 km/dia
 *  - 12m → 6 km/h / 48 km/dia
 * Marcha forçada dobra distância/hora + Fort CD 15+n ou 1d6 PV.
 * Terreno difícil OR clima ruim: × ½ cumulativo.
 */

describe('constantes p270', () => {
  it('8h de marcha por dia', () => {
    expect(TRAVEL_HOURS_PER_DAY).toBe(8)
  })

  it('fator 0,5 km/h por metro', () => {
    expect(KM_PER_HOUR_PER_METER).toBe(0.5)
  })

  it('marcha forçada dobra', () => {
    expect(MARCHA_FORCADA_MULTIPLIER).toBe(2)
  })

  it('Fort CD base 15', () => {
    expect(MARCHA_FORCADA_FORT_CD_BASE).toBe(15)
  })

  it('progressão CD +1', () => {
    expect(MARCHA_FORCADA_CD_PROGRESSAO).toBe(1)
  })

  it('dano falha 1d6', () => {
    expect(MARCHA_FORCADA_DANO_FALHA).toBe('1d6')
  })

  it('terreno difícil × ½', () => {
    expect(DIFFICULT_TERRAIN_MULTIPLIER).toBe(0.5)
  })

  it('clima ruim × ½', () => {
    expect(BAD_WEATHER_MULTIPLIER).toBe(0.5)
  })
})

describe('TABELA_6_4 — verbatim', () => {
  it('5 linhas canônicas', () => {
    expect(TABELA_6_4.length).toBe(5)
  })

  it('frozen', () => {
    expect(Object.isFrozen(TABELA_6_4)).toBe(true)
  })

  it.each([
    [4.5, 2.25, 18],
    [6, 3, 24],
    [7.5, 3.75, 30],
    [9, 4.5, 36],
    [12, 6, 48],
  ])('desloc %sm → %s km/h / %s km/dia', (m, hourKm, dayKm) => {
    const row = TABELA_6_4.find((r) => r.deslocamentoM === m)!
    expect(row.kmPerHour).toBe(hourKm)
    expect(row.kmPerDay).toBe(dayKm)
  })
})

describe('kmPerHour + kmPerDay — fórmula', () => {
  it('desloc 6m → 3 km/h', () => {
    expect(kmPerHour(6)).toBe(3)
  })

  it('desloc 12m → 6 km/h', () => {
    expect(kmPerHour(12)).toBe(6)
  })

  it('extrapola: desloc 15m → 7,5 km/h', () => {
    expect(kmPerHour(15)).toBe(7.5)
  })

  it('extrapola: desloc 18m → 9 km/h', () => {
    expect(kmPerHour(18)).toBe(9)
  })

  it('desloc 0m → 0 km/h', () => {
    expect(kmPerHour(0)).toBe(0)
  })

  it('throws se deslocamento negativo', () => {
    expect(() => kmPerHour(-1)).toThrow(/deslocamentoM/)
  })

  it('desloc 9m default → 36 km/dia', () => {
    expect(kmPerDay(9)).toBe(36)
  })

  it('desloc 12m com 12h → 72 km', () => {
    expect(kmPerDay(12, 12)).toBe(72)
  })

  it('throws se hoursPerDay negativo', () => {
    expect(() => kmPerDay(9, -1)).toThrow(/hoursPerDay/)
  })
})

describe('kmPerHourForced — marcha forçada', () => {
  it('desloc 9m: dobra para 9 km/h', () => {
    expect(kmPerHourForced(9)).toBe(9)
  })

  it('desloc 12m: dobra para 12 km/h', () => {
    expect(kmPerHourForced(12)).toBe(12)
  })
})

describe('marchaForcadaFortCd — CD 15+n', () => {
  it('primeira hora → 15', () => {
    expect(marchaForcadaFortCd(0)).toBe(15)
  })

  it('segunda → 16', () => {
    expect(marchaForcadaFortCd(1)).toBe(16)
  })

  it('quinta → 19', () => {
    expect(marchaForcadaFortCd(4)).toBe(19)
  })

  it('throws se previousTests negativo', () => {
    expect(() => marchaForcadaFortCd(-1)).toThrow(/previousTests/)
  })
})

describe('applyTerrainWeatherModifier — p270', () => {
  it('sem opts: distância intacta', () => {
    expect(applyTerrainWeatherModifier(36)).toBe(36)
  })

  it('só terreno difícil: × ½ → 18', () => {
    expect(applyTerrainWeatherModifier(36, { difficultTerrain: true })).toBe(18)
  })

  it('só clima ruim: × ½ → 18', () => {
    expect(applyTerrainWeatherModifier(36, { badWeather: true })).toBe(18)
  })

  it('ambos cumulativos: × ¼ → 9', () => {
    expect(
      applyTerrainWeatherModifier(36, {
        difficultTerrain: true,
        badWeather: true,
      }),
    ).toBe(9)
  })
})

describe('partyKmPerHour — velocidade do mais lento', () => {
  it('grupo de 3 (9m, 6m, 12m) → 3 km/h (usa 6m)', () => {
    expect(partyKmPerHour([9, 6, 12])).toBe(3)
  })

  it('grupo de 1 (9m) → 4,5 km/h', () => {
    expect(partyKmPerHour([9])).toBe(4.5)
  })

  it('throws se array vazio', () => {
    expect(() => partyKmPerHour([])).toThrow(/at least one member/)
  })
})
