import { describe, expect, it } from 'vitest'
import {
  CURTO_RANGE_METERS,
  DESLOCAMENTO_VOO_FALL_PER_ROUND_M,
  DESLOCAMENTO_VOO_TRIP_FALL_METERS_PER_D6,
  FARO_CAMUFLAGEM_TOTAL_FAILURE_RATE,
  HABILIDADE_AGARRAR_APRIMORADO,
  HABILIDADE_FARO,
  HABILIDADE_INCORPOREO,
  HABILIDADE_PERCEPCAO_AS_CEGAS,
  HABILIDADE_VISAO_NA_PENUMBRA,
  HABILIDADE_VISAO_NO_ESCURO,
  VULNERABILIDADE_DAMAGE_MULTIPLIER,
  habilidadeCuraAcelerada,
  habilidadeDeslocamento,
  habilidadeFortificacao,
  habilidadeImunidade,
  habilidadeReducaoDeDano,
  habilidadeResistenciaAEfeito,
  habilidadeVulnerabilidadeDano,
  voarQuedaDerrubadoM,
  vulnerabilidadeDamage,
} from '../habilidades-gerais'

/**
 * PDF livro p228-229 — Habilidades Gerais.
 */

describe('Constantes globais', () => {
  it('valores verbatim', () => {
    expect(CURTO_RANGE_METERS).toBe(9)
    expect(VULNERABILIDADE_DAMAGE_MULTIPLIER).toBe(1.5)
    expect(FARO_CAMUFLAGEM_TOTAL_FAILURE_RATE).toBe(0.2)
    expect(DESLOCAMENTO_VOO_FALL_PER_ROUND_M).toBe(150)
    expect(DESLOCAMENTO_VOO_TRIP_FALL_METERS_PER_D6).toBe(1.5)
  })
})

describe('Entradas fixas — frozen + estrutura', () => {
  it('Agarrar Aprimorado', () => {
    expect(Object.isFrozen(HABILIDADE_AGARRAR_APRIMORADO)).toBe(true)
    expect(HABILIDADE_AGARRAR_APRIMORADO.kind).toBe('agarrar-aprimorado')
    expect(HABILIDADE_AGARRAR_APRIMORADO.actionOnHit).toBe('livre')
    expect(HABILIDADE_AGARRAR_APRIMORADO.bookPage).toBe(228)
  })

  it('Faro', () => {
    expect(Object.isFrozen(HABILIDADE_FARO)).toBe(true)
    expect(HABILIDADE_FARO.camuflagemTotalFailureRate).toBe(0.2)
    expect(HABILIDADE_FARO.range).toBe('curto')
    expect(HABILIDADE_FARO.bookPage).toBe(229)
  })

  it('Incorpóreo', () => {
    expect(Object.isFrozen(HABILIDADE_INCORPOREO)).toBe(true)
    expect(HABILIDADE_INCORPOREO.onlyAffectedByMagical).toBe(true)
    expect(HABILIDADE_INCORPOREO.hasNoStrength).toBe(true)
  })

  it('Percepção às Cegas', () => {
    expect(Object.isFrozen(HABILIDADE_PERCEPCAO_AS_CEGAS)).toBe(true)
    expect(HABILIDADE_PERCEPCAO_AS_CEGAS.range).toBe('curto')
    expect(HABILIDADE_PERCEPCAO_AS_CEGAS.ignoresDarknessAndInvisibility).toBe(
      true,
    )
  })

  it('Visão na Penumbra / Visão no Escuro', () => {
    expect(Object.isFrozen(HABILIDADE_VISAO_NA_PENUMBRA)).toBe(true)
    expect(Object.isFrozen(HABILIDADE_VISAO_NO_ESCURO)).toBe(true)
    expect(HABILIDADE_VISAO_NA_PENUMBRA.worksInMagicalDarkness).toBe(false)
    expect(HABILIDADE_VISAO_NO_ESCURO.worksInMagicalDarkness).toBe(false)
  })
})

describe('habilidadeCuraAcelerada', () => {
  it('sem bloqueio → apenas hpPerTurn', () => {
    const h = habilidadeCuraAcelerada(5)
    expect(h.hpPerTurn).toBe(5)
    expect(h.blockedByDamageType).toBeUndefined()
    expect(h.healsMaxHpLoss).toBe(false)
    expect(h.bookPage).toBe(228)
  })

  it('com bloqueio /ácido', () => {
    const h = habilidadeCuraAcelerada(10, 'ácido')
    expect(h.blockedByDamageType).toBe('ácido')
  })

  it('hp ≤ 0 lança', () => {
    expect(() => habilidadeCuraAcelerada(0)).toThrow(/hpPerTurn must be > 0/)
    expect(() => habilidadeCuraAcelerada(-3)).toThrow(/hpPerTurn must be > 0/)
  })
})

describe('habilidadeDeslocamento', () => {
  it('escalada 9m → p228', () => {
    const h = habilidadeDeslocamento('deslocamento-escalada', 9)
    expect(h.speedMeters).toBe(9)
    expect(h.name).toBe('Deslocamento de Escalada')
    expect(h.bookPage).toBe(228)
  })

  it('escavação 6m', () => {
    const h = habilidadeDeslocamento('deslocamento-escavacao', 6)
    expect(h.speedMeters).toBe(6)
    expect(h.name).toBe('Deslocamento de Escavação')
    expect(h.bookPage).toBe(228)
  })

  it('natação 15m', () => {
    const h = habilidadeDeslocamento('deslocamento-natacao', 15)
    expect(h.name).toBe('Deslocamento de Natação')
    expect(h.bookPage).toBe(228)
  })

  it('voo 18m → p229', () => {
    const h = habilidadeDeslocamento('deslocamento-voo', 18)
    expect(h.speedMeters).toBe(18)
    expect(h.name).toBe('Deslocamento de Voo')
    expect(h.bookPage).toBe(229)
  })

  it('velocidade ≤ 0 lança', () => {
    expect(() =>
      habilidadeDeslocamento('deslocamento-voo', 0),
    ).toThrow(/speedMeters must be > 0/)
  })
})

describe('habilidadeImunidade', () => {
  it('tipo-de-dano fogo → sem crítico->normal', () => {
    const h = habilidadeImunidade('tipo-de-dano', 'fogo')
    expect(h.target).toBe('tipo-de-dano')
    expect(h.specificTarget).toBe('fogo')
    expect(h.criticalConvertsToNormal).toBe(false)
  })

  it('habilidade + acerto-critico → crítico vira normal', () => {
    const h = habilidadeImunidade('habilidade', 'acerto-critico')
    expect(h.criticalConvertsToNormal).toBe(true)
  })

  it('condição sono', () => {
    const h = habilidadeImunidade('condicao', 'sono')
    expect(h.target).toBe('condicao')
    expect(h.specificTarget).toBe('sono')
  })
})

describe('habilidadeReducaoDeDano', () => {
  it('RD 5 sem bypass', () => {
    const h = habilidadeReducaoDeDano(5)
    expect(h.value).toBe(5)
    expect(h.bypassedBy).toBeUndefined()
  })

  it('RD 10/mágico', () => {
    const h = habilidadeReducaoDeDano(10, 'mágico')
    expect(h.value).toBe(10)
    expect(h.bypassedBy).toBe('mágico')
  })

  it('valor ≤ 0 lança', () => {
    expect(() => habilidadeReducaoDeDano(0)).toThrow(/value must be > 0/)
  })
})

describe('habilidadeResistenciaAEfeito', () => {
  it('Resistência a magia +2', () => {
    const h = habilidadeResistenciaAEfeito('magia', 2)
    expect(h.effectType).toBe('magia')
    expect(h.bonus).toBe(2)
  })
})

describe('habilidadeVulnerabilidadeDano', () => {
  it('vulnerabilidade a frio → 1.5×', () => {
    const h = habilidadeVulnerabilidadeDano('frio')
    expect(h.damageType).toBe('frio')
    expect(h.damageMultiplier).toBe(1.5)
  })
})

describe('habilidadeFortificacao', () => {
  it('50% → 0.5', () => {
    const h = habilidadeFortificacao(50)
    expect(h.ignoreExtraDamageChance).toBe(0.5)
    expect(h.name).toBe('Fortificação 50%')
  })

  it('percentual fora de (0, 100] lança', () => {
    expect(() => habilidadeFortificacao(0)).toThrow(/percent must be/)
    expect(() => habilidadeFortificacao(101)).toThrow(/percent must be/)
    expect(() => habilidadeFortificacao(-5)).toThrow(/percent must be/)
  })
})

describe('vulnerabilidadeDamage — exemplo do livro (15 × 1.5 = 22)', () => {
  it('15 → 22 (floor)', () => {
    expect(vulnerabilidadeDamage(15)).toBe(22)
  })

  it('10 → 15', () => {
    expect(vulnerabilidadeDamage(10)).toBe(15)
  })

  it('0 → 0', () => {
    expect(vulnerabilidadeDamage(0)).toBe(0)
  })

  it('dano negativo lança', () => {
    expect(() => vulnerabilidadeDamage(-1)).toThrow(/baseDamage must be ≥ 0/)
  })
})

describe('voarQuedaDerrubadoM — 1d6 × 1,5m', () => {
  it('d6 = 1 → 1.5m', () => {
    expect(voarQuedaDerrubadoM(1)).toBe(1.5)
  })

  it('d6 = 6 → 9m', () => {
    expect(voarQuedaDerrubadoM(6)).toBe(9)
  })

  it('d6 fora de 1..6 lança', () => {
    expect(() => voarQuedaDerrubadoM(0)).toThrow(/d6Roll must be 1..6/)
    expect(() => voarQuedaDerrubadoM(7)).toThrow(/d6Roll must be 1..6/)
  })
})
