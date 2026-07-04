import { describe, expect, it } from 'vitest'
import {
  CAVALGAR_ARMOR_PENALTY,
  CAVALGAR_NO_SADDLE_PENALTY,
  CAVALGAR_TRAINED_ONLY,
  CAVALGAR_USAGES,
  CONDUZIR_CD_PERIGOSO,
  CONDUZIR_CD_RUIM,
  CONDUZIR_FALL_DAMAGE_DICE,
  GALOPAR_BASE_SPEED_METERS,
  GALOPAR_MODIFIER_PER_SPEED_STEP,
  GALOPAR_SQUARE_METERS,
  MONTAR_RAPIDAMENTE_CD,
  MONTAR_RAPIDAMENTE_FALL_MARGIN,
  cavalgarSaddlePenalty,
  cavalgarUsageByKind,
  conduzirCd,
  conduzirOutcome,
  galoparMetersCovered,
  galoparMovementModifier,
  galoparSquaresCovered,
  montarRapidamenteCd,
  montarRapidamenteOutcome,
} from '../cavalgar-skill-usages'

/**
 * PDF livro p116-117 — Perícia Cavalgar (DES, aberta).
 *  1. Conduzir — CD 15/25, parte do movimento; falha = cai + 1d6 dano
 *  2. Galopar — completa; ±2 por 1,5m do deslocamento base 9m
 *  3. Montar Rapidamente — CD 20; livre em sucesso; falha 5+ = cai no chão
 */

describe('CAVALGAR_USAGES — shape', () => {
  it('exatamente 3 usos', () => {
    expect(CAVALGAR_USAGES.length).toBe(3)
  })

  it('frozen', () => {
    expect(Object.isFrozen(CAVALGAR_USAGES)).toBe(true)
  })

  it('ids canônicos', () => {
    expect(CAVALGAR_USAGES.map((u) => u.id).sort()).toEqual([
      'conduzir',
      'galopar',
      'montar-rapidamente',
    ])
  })

  it('Conduzir em p116; Galopar e Montar em p117', () => {
    for (const u of CAVALGAR_USAGES) {
      const expected = u.id === 'conduzir' ? 116 : 117
      expect(u.bookPage).toBe(expected)
    }
  })
})

describe('Perícia flags — Tabela 2-1', () => {
  it('NÃO é somente treinada', () => {
    expect(CAVALGAR_TRAINED_ONLY).toBe(false)
  })

  it('sem penalidade de armadura', () => {
    expect(CAVALGAR_ARMOR_PENALTY).toBe(false)
  })
})

describe('Conduzir — p116', () => {
  it('CDs verbatim', () => {
    expect(CONDUZIR_CD_RUIM).toBe(15)
    expect(CONDUZIR_CD_PERIGOSO).toBe(25)
  })

  it('dano de queda 1d6', () => {
    expect(CONDUZIR_FALL_DAMAGE_DICE).toBe('1d6')
  })

  it('parte do movimento', () => {
    const usage = cavalgarUsageByKind('conduzir')
    if (usage.kind !== 'conduzir') throw new Error('narrow failed')
    expect(usage.action).toBe('parte-do-movimento')
  })
})

describe('conduzirCd', () => {
  it('ruim/pequeno → 15', () => {
    expect(conduzirCd('ruim-obstaculo-pequeno')).toBe(15)
  })

  it('perigoso/grande → 25', () => {
    expect(conduzirCd('perigoso-obstaculo-grande')).toBe(25)
  })
})

describe('conduzirOutcome', () => {
  it('sucesso → success', () => {
    expect(conduzirOutcome(15, 15)).toBe('success')
    expect(conduzirOutcome(20, 15)).toBe('success')
  })

  it('falha → falls-and-takes-1d6', () => {
    expect(conduzirOutcome(14, 15)).toBe('falls-and-takes-1d6')
    expect(conduzirOutcome(10, 25)).toBe('falls-and-takes-1d6')
  })
})

describe('Galopar — p117', () => {
  it('constantes verbatim', () => {
    expect(GALOPAR_SQUARE_METERS).toBe(1.5)
    expect(GALOPAR_BASE_SPEED_METERS).toBe(9)
    expect(GALOPAR_MODIFIER_PER_SPEED_STEP).toBe(2)
  })

  it('ação completa', () => {
    const usage = cavalgarUsageByKind('galopar')
    if (usage.kind !== 'galopar') throw new Error('narrow failed')
    expect(usage.action).toBe('completa')
  })
})

describe('galoparMovementModifier', () => {
  it('9m → 0', () => {
    expect(galoparMovementModifier(9)).toBe(0)
  })

  it('cavalo 12m → +4', () => {
    expect(galoparMovementModifier(12)).toBe(4)
  })

  it('trobo 15m → +8', () => {
    expect(galoparMovementModifier(15)).toBe(8)
  })

  it('montaria lenta 6m → -4', () => {
    expect(galoparMovementModifier(6)).toBe(-4)
  })
})

describe('galoparSquaresCovered / galoparMetersCovered', () => {
  it('roll 22 → 22 quadrados / 33m', () => {
    expect(galoparSquaresCovered(22)).toBe(22)
    expect(galoparMetersCovered(22)).toBe(33)
  })
})

describe('Montar Rapidamente — p117', () => {
  const usage = cavalgarUsageByKind('montar-rapidamente')

  it('CD 20 e margem de queda 5', () => {
    expect(MONTAR_RAPIDAMENTE_CD).toBe(20)
    expect(MONTAR_RAPIDAMENTE_FALL_MARGIN).toBe(5)
    if (usage.kind !== 'montar-rapidamente') throw new Error('narrow failed')
    expect(usage.dc).toBe(20)
  })

  it('sucesso vira ação livre', () => {
    if (usage.kind !== 'montar-rapidamente') throw new Error('narrow failed')
    expect(usage.successPromotesToFreeAction).toBe(true)
  })
})

describe('montarRapidamenteCd', () => {
  it('sempre 20', () => {
    expect(montarRapidamenteCd()).toBe(20)
  })
})

describe('montarRapidamenteOutcome', () => {
  it('sucesso → free-action', () => {
    expect(montarRapidamenteOutcome(20, 20)).toBe('free-action')
    expect(montarRapidamenteOutcome(25, 20)).toBe('free-action')
  })

  it('falha < 5 → normal-movement-action', () => {
    expect(montarRapidamenteOutcome(19, 20)).toBe('normal-movement-action')
    expect(montarRapidamenteOutcome(16, 20)).toBe('normal-movement-action')
  })

  it('falha ≥ 5 → falls-prone', () => {
    expect(montarRapidamenteOutcome(15, 20)).toBe('falls-prone')
    expect(montarRapidamenteOutcome(10, 20)).toBe('falls-prone')
  })
})

describe('cavalgarSaddlePenalty', () => {
  it('com sela → 0', () => {
    expect(cavalgarSaddlePenalty(true)).toBe(0)
  })

  it('sem sela → -5', () => {
    expect(cavalgarSaddlePenalty(false)).toBe(-5)
    expect(CAVALGAR_NO_SADDLE_PENALTY).toBe(-5)
  })
})

describe('cavalgarUsageByKind', () => {
  it('throws se kind inválido', () => {
    expect(() =>
      // @ts-expect-error — invalid kind on purpose
      cavalgarUsageByKind('encilhar'),
    ).toThrow(/unknown kind/)
  })

  it('resolve todos', () => {
    for (const k of ['conduzir', 'galopar', 'montar-rapidamente'] as const) {
      expect(cavalgarUsageByKind(k).kind).toBe(k)
    }
  })
})
