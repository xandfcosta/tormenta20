import { describe, expect, it } from 'vitest'
import {
  AQUATICO_FEATURES,
  ARTICO_FEATURES,
  BIOMES,
  COLINAS_FEATURES,
  FLORESTAS_FEATURES,
  MONTANHAS_FEATURES,
  biomeFeatures,
} from '../terrain-cds'

/**
 * PDF Cap 6 p268-269 — Terrenos. 8 biomas, cada um com 1-4 features
 * mecânicas (CDs, dano, cobertura, condições).
 */

describe('BIOMES (p268-269)', () => {
  it('lists 8 biomes in book order', () => {
    expect(BIOMES).toEqual([
      'colinas',
      'desertos',
      'florestas',
      'montanhas',
      'pantanos',
      'planicies',
      'artico',
      'aquatico',
    ])
  })
})

describe('biomeFeatures (p268-269)', () => {
  it('every biome has ≥ 1 feature', () => {
    for (const b of BIOMES) {
      expect(biomeFeatures(b).length).toBeGreaterThan(0)
    }
  })

  it('each feature has a non-empty label', () => {
    for (const b of BIOMES) {
      for (const f of biomeFeatures(b)) {
        expect(f.label.length).toBeGreaterThan(0)
      }
    }
  })
})

describe('Colinas (p268)', () => {
  it('has 3 features: suave / íngreme / penhasco', () => {
    expect(COLINAS_FEATURES).toEqual([
      'inclinacao-suave',
      'inclinacao-ingreme',
      'penhasco',
    ])
  })

  it('inclinação íngreme: descent CD 10 → 1d4×1,5m + 1d6/1,5m impacto', () => {
    const rows = biomeFeatures('colinas')
    const ingreme = rows.find((r) => r.feature === 'inclinacao-ingreme')!
    if (ingreme.feature !== 'inclinacao-ingreme')
      throw new Error('type narrow failed')
    expect(ingreme.descentTestCd).toBe(10)
    expect(ingreme.failureConsequence.rollDice).toBe('1d4')
    expect(ingreme.failureConsequence.damagePer1_5m).toBe('1d6')
  })

  it('penhasco: 1d6×3m altura, escalar CD 15', () => {
    const rows = biomeFeatures('colinas')
    const p = rows.find((r) => r.feature === 'penhasco')!
    if (p.feature !== 'penhasco') throw new Error('type narrow failed')
    expect(p.heightDice).toBe('1d6')
    expect(p.heightMultiplierMeters).toBe(3)
    expect(p.climbCd).toBe(15)
  })
})

describe('Desertos (p268)', () => {
  it('dunas: como inclinação íngreme mas cair não causa dano', () => {
    const rows = biomeFeatures('desertos')
    expect(rows).toHaveLength(1)
    const d = rows[0]!
    if (d.feature !== 'dunas') throw new Error('expected dunas')
    expect(d.worksAsInclinacaoIngreme).toBe(true)
    expect(d.fallDamage).toBe('nenhum')
  })
})

describe('Florestas (p268)', () => {
  it('has 4 features', () => {
    expect(FLORESTAS_FEATURES).toEqual([
      'arvore-estreita',
      'arvore-larga',
      'folhagens',
      'vegetacao-rasteira',
    ])
  })

  it('árvore estreita: RD 5, PV 100, escalar CD 15', () => {
    const rows = biomeFeatures('florestas')
    const t = rows.find((r) => r.feature === 'arvore-estreita')!
    if (t.feature !== 'arvore-estreita') throw new Error('narrow failed')
    expect(t.damageReduction).toBe(5)
    expect(t.hp).toBe(100)
    expect(t.climbCd).toBe(15)
    expect(t.cover).toBe('leve')
  })

  it('árvore larga: RD 5, PV 500, topo dá camuflagem leve', () => {
    const rows = biomeFeatures('florestas')
    const t = rows.find((r) => r.feature === 'arvore-larga')!
    if (t.feature !== 'arvore-larga') throw new Error('narrow failed')
    expect(t.hp).toBe(500)
    expect(t.topGrantsConcealment).toBe('leve')
  })

  it('vegetação rasteira: −2 Furtividade + terreno difícil', () => {
    const rows = biomeFeatures('florestas')
    const v = rows.find((r) => r.feature === 'vegetacao-rasteira')!
    if (v.feature !== 'vegetacao-rasteira') throw new Error('narrow failed')
    expect(v.stealthPenalty).toBe(-2)
    expect(v.difficultTerrain).toBe(true)
  })
})

describe('Montanhas (p268)', () => {
  it('has 4 features', () => {
    expect(MONTANHAS_FEATURES).toEqual([
      'abismo',
      'altitude',
      'paredao',
      'seixos',
    ])
  })

  it('abismo: 1d4×1,5m largura, 2d4×3m profundidade, escalar CD 20', () => {
    const rows = biomeFeatures('montanhas')
    const a = rows.find((r) => r.feature === 'abismo')!
    if (a.feature !== 'abismo') throw new Error('narrow failed')
    expect(a.widthDice).toBe('1d4')
    expect(a.depthDice).toBe('2d4')
    expect(a.climbOutCd).toBe(20)
  })

  it('altitude: Fort CD 15 por dia (falha = fatigado/exausto)', () => {
    const rows = biomeFeatures('montanhas')
    const alt = rows.find((r) => r.feature === 'altitude')!
    if (alt.feature !== 'altitude') throw new Error('narrow failed')
    expect(alt.fortCdBase).toBe(15)
    expect(alt.period).toBe('dia')
    expect(alt.failureEffect).toBe('fatigado-ou-exausto')
  })

  it('paredão: 2d6×3m altura, escalar CD 25 (maior CD de escalada do bioma)', () => {
    const rows = biomeFeatures('montanhas')
    const p = rows.find((r) => r.feature === 'paredao')!
    if (p.feature !== 'paredao') throw new Error('narrow failed')
    expect(p.heightDice).toBe('2d6')
    expect(p.climbCd).toBe(25)
  })

  it('seixos: descent CD sobe para 15 (vs 10 padrão de inclinação íngreme)', () => {
    const rows = biomeFeatures('montanhas')
    const s = rows.find((r) => r.feature === 'seixos')!
    if (s.feature !== 'seixos') throw new Error('narrow failed')
    expect(s.descentTestCd).toBe(15)
  })
})

describe('Pântanos (p268)', () => {
  it('lodaçal: terreno difícil + vulnerável', () => {
    const rows = biomeFeatures('pantanos')
    expect(rows).toHaveLength(1)
    const l = rows[0]!
    if (l.feature !== 'lodacal') throw new Error('expected lodacal')
    expect(l.difficultTerrain).toBe(true)
    expect(l.conditionApplied).toBe('vulneravel')
  })
})

describe('Planícies (p268)', () => {
  it('trincheira: cobertura leve contra ranged; sair = terreno difícil', () => {
    const rows = biomeFeatures('planicies')
    expect(rows).toHaveLength(1)
    const t = rows[0]!
    if (t.feature !== 'trincheira') throw new Error('expected trincheira')
    expect(t.cover).toBe('leve')
    expect(t.coverScope).toBe('ranged')
    expect(t.exitAsDifficultTerrain).toBe(true)
  })
})

describe('Ártico (p269)', () => {
  it('has 2 features', () => {
    expect(ARTICO_FEATURES).toEqual(['gelo', 'rio-congelado'])
  })

  it('gelo: metade do deslocamento free; Acrobacia CD 15 mov / = dano', () => {
    const rows = biomeFeatures('artico')
    const g = rows.find((r) => r.feature === 'gelo')!
    if (g.feature !== 'gelo') throw new Error('narrow failed')
    expect(g.halfSpeedFree).toBe(true)
    expect(g.acrobaticsCd.movement).toBe(15)
    expect(g.acrobaticsCd.damage).toBe('equal-to-damage')
    expect(g.failureSlide.rollDice).toBe('1d4')
  })

  it('rio congelado: gelo quebra em roll 1, submerso = 1d6 frio/round, break = 10 impacto/fogo', () => {
    const rows = biomeFeatures('artico')
    const r = rows.find((f) => f.feature === 'rio-congelado')!
    if (r.feature !== 'rio-congelado') throw new Error('narrow failed')
    expect(r.iceCracksOnSlideRoll).toBe(1)
    expect(r.submergedDamagePerRound.type).toBe('frio')
    expect(r.breakThroughDamage.amount).toBe(10)
    expect(r.breakThroughDamage.types).toEqual(['impacto', 'fogo'])
  })
})

describe('Aquático (p269)', () => {
  it('has 3 features', () => {
    expect(AQUATICO_FEATURES).toEqual([
      'agua-corrente',
      'agua-parada',
      'submerso',
    ])
  })

  it('água corrente: 1d6×3m/rodada; swim CD 15/20; sair CD 20', () => {
    const rows = biomeFeatures('aquatico')
    const ac = rows.find((r) => r.feature === 'agua-corrente')!
    if (ac.feature !== 'agua-corrente') throw new Error('narrow failed')
    expect(ac.currentSpeedDice).toBe('1d6')
    expect(ac.currentMultiplierMeters).toBe(3)
    expect(ac.swimCd.slowCorrenteza).toBe(15)
    expect(ac.swimCd.fastCorrenteza).toBe(20)
    expect(ac.exitCd).toBe(20)
  })

  it('submerso: -2 attack, -5 Perc, sem magias verbais, ranged proibido salvo perfuração/bestas/redes', () => {
    const rows = biomeFeatures('aquatico')
    const s = rows.find((r) => r.feature === 'submerso')!
    if (s.feature !== 'submerso') throw new Error('narrow failed')
    expect(s.attackPenalty).toBe(-2)
    expect(s.perceptionPenalty).toBe(-5)
    expect(s.cannotSpeak).toBe(true)
    expect(s.rangedForbiddenExcept).toEqual([
      'arremesso-perfuracao',
      'bestas',
      'redes',
    ])
    expect(s.slashImpactHalfDamage).toBe(true)
    expect(s.againstAboveWaterConcealment).toBe('leve')
    expect(s.againstAboveWaterCover).toBe('leve')
  })
})
