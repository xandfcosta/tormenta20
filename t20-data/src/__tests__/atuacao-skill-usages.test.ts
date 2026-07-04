import { describe, expect, it } from 'vitest'
import {
  APRESENTACAO_BASE_TIBAR_DICE,
  APRESENTACAO_CD,
  APRESENTACAO_EXTRA_DIE_PER_MARGIN,
  ATUACAO_ARMOR_PENALTY,
  ATUACAO_TRAINED_ONLY,
  ATUACAO_USAGES,
  IMPRESSIONAR_FAILURE_CHARISMA_PENALTY,
  IMPRESSIONAR_SUCCESS_CHARISMA_BONUS,
  LOCATION_QUALITY_MULTIPLIER,
  apresentacaoD6Count,
  apresentacaoLocationMultiplier,
  apresentacaoPayout,
  atuacaoUsageByKind,
  impressionarAudienceCd,
  impressionarCharismaModifier,
  impressionarOutcome,
} from '../atuacao-skill-usages'

/**
 * PDF livro p116 — Perícia Atuação (CAR, treinada).
 *  1. Apresentação — CD 20, 1 dia/noite, T$ 1d6 base + 1d6/5 acima; local ½/1×/2×
 *  2. Impressionar — oposto vs Vontade; +2/-2 em perícias CAR no dia; plateia usa melhor Vontade
 */

describe('ATUACAO_USAGES — shape', () => {
  it('exatamente 2 usos', () => {
    expect(ATUACAO_USAGES.length).toBe(2)
  })

  it('frozen', () => {
    expect(Object.isFrozen(ATUACAO_USAGES)).toBe(true)
  })

  it('ids canônicos', () => {
    expect(ATUACAO_USAGES.map((u) => u.id).sort()).toEqual([
      'apresentacao',
      'impressionar',
    ])
  })

  it('todos em p116', () => {
    for (const u of ATUACAO_USAGES) {
      expect(u.bookPage).toBe(116)
    }
  })
})

describe('Perícia flags — Tabela 2-1', () => {
  it('apenas treinada', () => {
    expect(ATUACAO_TRAINED_ONLY).toBe(true)
  })

  it('sem penalidade de armadura', () => {
    expect(ATUACAO_ARMOR_PENALTY).toBe(false)
  })
})

describe('Apresentação — p116', () => {
  it('constantes verbatim', () => {
    expect(APRESENTACAO_CD).toBe(20)
    expect(APRESENTACAO_BASE_TIBAR_DICE).toBe('1d6')
    expect(APRESENTACAO_EXTRA_DIE_PER_MARGIN).toBe(5)
  })

  it('duração 1 dia/noite', () => {
    const usage = atuacaoUsageByKind('apresentacao')
    if (usage.kind !== 'apresentacao') throw new Error('narrow failed')
    expect(usage.duration).toBe('um-dia-ou-noite')
  })
})

describe('apresentacaoD6Count', () => {
  it('roll 19 → 0 (falha)', () => {
    expect(apresentacaoD6Count(19)).toBe(0)
  })

  it('roll 20 → 1', () => {
    expect(apresentacaoD6Count(20)).toBe(1)
  })

  it('roll 24 → 1 (4 acima; falta 1 para +1d6)', () => {
    expect(apresentacaoD6Count(24)).toBe(1)
  })

  it('roll 25 → 2 (5 acima = +1d6)', () => {
    expect(apresentacaoD6Count(25)).toBe(2)
  })

  it('roll 40 → 5 (20 acima = +4d6)', () => {
    expect(apresentacaoD6Count(40)).toBe(5)
  })
})

describe('LOCATION_QUALITY_MULTIPLIER / apresentacaoLocationMultiplier', () => {
  it('frozen', () => {
    expect(Object.isFrozen(LOCATION_QUALITY_MULTIPLIER)).toBe(true)
  })

  it('multiplicadores verbatim', () => {
    expect(apresentacaoLocationMultiplier('inadequado')).toBe(0.5)
    expect(apresentacaoLocationMultiplier('propicio')).toBe(1)
    expect(apresentacaoLocationMultiplier('especialmente-propicio')).toBe(2)
  })
})

describe('apresentacaoPayout', () => {
  it('propício T$ 10 → 10', () => {
    expect(apresentacaoPayout(10, 'propicio')).toBe(10)
  })

  it('inadequado T$ 10 → 5', () => {
    expect(apresentacaoPayout(10, 'inadequado')).toBe(5)
  })

  it('especialmente propício T$ 10 → 20', () => {
    expect(apresentacaoPayout(10, 'especialmente-propicio')).toBe(20)
  })

  it('T$ negativo lança', () => {
    expect(() => apresentacaoPayout(-1, 'propicio')).toThrow(
      /d6TotalTibar must be ≥ 0/,
    )
  })
})

describe('Impressionar — p116', () => {
  const usage = atuacaoUsageByKind('impressionar')

  it('oposto por Vontade', () => {
    if (usage.kind !== 'impressionar') throw new Error('narrow failed')
    expect(usage.opposedBy).toBe('vontade')
  })

  it('modificadores verbatim', () => {
    expect(IMPRESSIONAR_SUCCESS_CHARISMA_BONUS).toBe(2)
    expect(IMPRESSIONAR_FAILURE_CHARISMA_PENALTY).toBe(-2)
  })

  it('falha trava tentativas no dia + plateia usa melhor', () => {
    if (usage.kind !== 'impressionar') throw new Error('narrow failed')
    expect(usage.failureLocksOutSameDay).toBe(true)
    expect(usage.audienceUsesBestValue).toBe(true)
  })
})

describe('impressionarOutcome', () => {
  it('atuação > vontade → success', () => {
    expect(impressionarOutcome(20, 15)).toBe('success')
  })

  it('empate → success (atacante vence)', () => {
    expect(impressionarOutcome(20, 20)).toBe('success')
  })

  it('atuação < vontade → failed', () => {
    expect(impressionarOutcome(15, 20)).toBe('failed')
  })
})

describe('impressionarCharismaModifier', () => {
  it('success → +2', () => {
    expect(impressionarCharismaModifier('success')).toBe(2)
  })

  it('failed → -2', () => {
    expect(impressionarCharismaModifier('failed')).toBe(-2)
  })
})

describe('impressionarAudienceCd', () => {
  it('pega maior Vontade da plateia', () => {
    expect(impressionarAudienceCd([15, 22, 18])).toBe(22)
    expect(impressionarAudienceCd([10])).toBe(10)
  })

  it('plateia vazia lança', () => {
    expect(() => impressionarAudienceCd([])).toThrow(
      /vontadeRolls must be non-empty/,
    )
  })
})

describe('atuacaoUsageByKind', () => {
  it('throws se kind inválido', () => {
    expect(() =>
      // @ts-expect-error — invalid kind on purpose
      atuacaoUsageByKind('improvisar'),
    ).toThrow(/unknown kind/)
  })

  it('resolve todos', () => {
    for (const k of ['apresentacao', 'impressionar'] as const) {
      expect(atuacaoUsageByKind(k).kind).toBe(k)
    }
  })
})
