import { describe, expect, it } from 'vitest'
import {
  ATLETISMO_ARMOR_PENALTY,
  ATLETISMO_NATACAO_ARMOR_PENALTY,
  ATLETISMO_TRAINED_ONLY,
  ATLETISMO_USAGES,
  CORRIDA_BASE_SPEED_METERS,
  CORRIDA_FADIGA_CD_BASE,
  CORRIDA_MODIFIER_PER_SPEED_STEP,
  CORRIDA_SQUARE_METERS,
  ESCALAR_CATCH_ALLY_CD_BONUS,
  ESCALAR_CD_APOIOS,
  ESCALAR_CD_ARVORE,
  ESCALAR_CD_MURO_LISO,
  ESCALAR_CD_MURO_REENTRANCIAS,
  ESCALAR_FALL_MARGIN,
  ESCALAR_FULL_SPEED_PENALTY,
  NATACAO_CD_AGITADA,
  NATACAO_CD_CALMA,
  NATACAO_CD_TEMPESTUOSA,
  NATACAO_DROWN_STILL_SUBMERGED_HP_LOSS_D6,
  NATACAO_HOLD_BREATH_CD_BASE,
  NATACAO_SINK_MARGIN,
  SALTAR_HIGH_JUMP_CD_PER_1_5M,
  SALTAR_LONG_JUMP_CD_PER_1_5M,
  SALTAR_MIN_RUN_UP_METERS,
  SALTAR_NO_RUN_UP_CD_INCREASE,
  afogamentoCd,
  atletismoUsageByKind,
  corridaFadigaCd,
  corridaMaxRoundsBeforeFadiga,
  corridaMetersCovered,
  corridaMovementModifier,
  corridaSquaresCovered,
  escalarCatchAllyCd,
  escalarCatchAllyOutcome,
  escalarCd,
  escalarFullSpeedPenalty,
  escalarOutcome,
  natacaoCd,
  natacaoOutcome,
  prenderRespiracaoMaxRounds,
  saltarCd,
} from '../atletismo-skill-usages'

/**
 * PDF livro p116 — Perícia Atletismo (FOR, aberta).
 *  1. Corrida — completa; ±2 por 1,5m do deslocamento; Fortitude CD 15+
 *  2. Escalar — CDs 10/15/20/25; movimento; falha 5+ = cai
 *  3. Natação — CDs 10/15/20; movimento; falha 5+ = afunda; armor penalty aplica
 *  4. Saltar — CD 5 por 1,5m (longo) / 15 por 1,5m (altura); +10 sem 6m impulso
 */

describe('ATLETISMO_USAGES — shape', () => {
  it('exatamente 4 usos', () => {
    expect(ATLETISMO_USAGES.length).toBe(4)
  })

  it('frozen', () => {
    expect(Object.isFrozen(ATLETISMO_USAGES)).toBe(true)
  })

  it('ids canônicos', () => {
    expect(ATLETISMO_USAGES.map((u) => u.id).sort()).toEqual([
      'corrida',
      'escalar',
      'natacao',
      'saltar',
    ])
  })

  it('todos em p116', () => {
    for (const u of ATLETISMO_USAGES) {
      expect(u.bookPage).toBe(116)
    }
  })
})

describe('Perícia flags — Tabela 2-1', () => {
  it('NÃO é somente treinada', () => {
    expect(ATLETISMO_TRAINED_ONLY).toBe(false)
  })

  it('sem penalidade de armadura GLOBAL', () => {
    expect(ATLETISMO_ARMOR_PENALTY).toBe(false)
  })

  it('Natação AINDA aplica penalidade de armadura', () => {
    expect(ATLETISMO_NATACAO_ARMOR_PENALTY).toBe(true)
  })
})

describe('Corrida — p116', () => {
  const usage = atletismoUsageByKind('corrida')

  it('constantes verbatim', () => {
    expect(CORRIDA_SQUARE_METERS).toBe(1.5)
    expect(CORRIDA_BASE_SPEED_METERS).toBe(9)
    expect(CORRIDA_MODIFIER_PER_SPEED_STEP).toBe(2)
    expect(CORRIDA_FADIGA_CD_BASE).toBe(15)
  })

  it('linha reta / sem terreno difícil', () => {
    if (usage.kind !== 'corrida') throw new Error('narrow failed')
    expect(usage.straightLineOnly).toBe(true)
    expect(usage.cannotEnterDifficultTerrain).toBe(true)
  })
})

describe('corridaMovementModifier', () => {
  it('9m base → 0', () => {
    expect(corridaMovementModifier(9)).toBe(0)
  })

  it('elfo 12m → +4', () => {
    expect(corridaMovementModifier(12)).toBe(4)
  })

  it('anão 6m → -4', () => {
    expect(corridaMovementModifier(6)).toBe(-4)
  })

  it('15m → +8', () => {
    expect(corridaMovementModifier(15)).toBe(8)
  })
})

describe('corridaSquaresCovered / corridaMetersCovered', () => {
  it('roll 20 → 20 quadrados / 30m', () => {
    expect(corridaSquaresCovered(20)).toBe(20)
    expect(corridaMetersCovered(20)).toBe(30)
  })
})

describe('corridaMaxRoundsBeforeFadiga', () => {
  it('CON 3 → 4 rodadas', () => {
    expect(corridaMaxRoundsBeforeFadiga(3)).toBe(4)
  })

  it('CON 0 → 1 rodada', () => {
    expect(corridaMaxRoundsBeforeFadiga(0)).toBe(1)
  })
})

describe('corridaFadigaCd', () => {
  it('sem testes anteriores → 15', () => {
    expect(corridaFadigaCd(0)).toBe(15)
  })

  it('após 3 testes → 18', () => {
    expect(corridaFadigaCd(3)).toBe(18)
  })
})

describe('Escalar — p116', () => {
  it('CDs verbatim', () => {
    expect(ESCALAR_CD_APOIOS).toBe(10)
    expect(ESCALAR_CD_ARVORE).toBe(15)
    expect(ESCALAR_CD_MURO_REENTRANCIAS).toBe(20)
    expect(ESCALAR_CD_MURO_LISO).toBe(25)
  })

  it('full-speed -5, margem 5, bônus pegar +10', () => {
    expect(ESCALAR_FULL_SPEED_PENALTY).toBe(-5)
    expect(ESCALAR_FALL_MARGIN).toBe(5)
    expect(ESCALAR_CATCH_ALLY_CD_BONUS).toBe(10)
  })

  it('desprevenido + dano força novo teste', () => {
    const usage = atletismoUsageByKind('escalar')
    if (usage.kind !== 'escalar') throw new Error('narrow failed')
    expect(usage.isFlatFooted).toBe(true)
    expect(usage.damageForcesNewTest).toBe(true)
  })
})

describe('escalarCd', () => {
  it.each([
    ['apoios-pes-e-maos', 10],
    ['arvore', 15],
    ['muro-com-reentrancias', 20],
    ['muro-liso', 25],
  ] as const)('%s → %s', (s, cd) => {
    expect(escalarCd(s)).toBe(cd)
  })
})

describe('escalarFullSpeedPenalty', () => {
  it('full → -5', () => {
    expect(escalarFullSpeedPenalty(true)).toBe(-5)
  })

  it('metade → 0', () => {
    expect(escalarFullSpeedPenalty(false)).toBe(0)
  })
})

describe('escalarCatchAllyCd', () => {
  it('muro liso → 35 (25+10)', () => {
    expect(escalarCatchAllyCd('muro-liso')).toBe(35)
  })

  it('árvore → 25 (15+10)', () => {
    expect(escalarCatchAllyCd('arvore')).toBe(25)
  })
})

describe('escalarOutcome', () => {
  it('sucesso', () => {
    expect(escalarOutcome(20, 20)).toBe('success')
    expect(escalarOutcome(25, 20)).toBe('success')
  })

  it('falha < 5 → no-progress', () => {
    expect(escalarOutcome(16, 20)).toBe('no-progress')
  })

  it('falha ≥ 5 → falls', () => {
    expect(escalarOutcome(15, 20)).toBe('falls')
  })
})

describe('escalarCatchAllyOutcome', () => {
  it('sucesso → caught', () => {
    expect(escalarCatchAllyOutcome(30, 30)).toBe('caught')
  })

  it('falha < 5 → missed', () => {
    expect(escalarCatchAllyOutcome(26, 30)).toBe('missed')
  })

  it('falha ≥ 5 → both-fall', () => {
    expect(escalarCatchAllyOutcome(25, 30)).toBe('both-fall')
  })
})

describe('Natação — p116', () => {
  const usage = atletismoUsageByKind('natacao')

  it('CDs verbatim', () => {
    expect(NATACAO_CD_CALMA).toBe(10)
    expect(NATACAO_CD_AGITADA).toBe(15)
    expect(NATACAO_CD_TEMPESTUOSA).toBe(20)
  })

  it('margens verbatim', () => {
    expect(NATACAO_SINK_MARGIN).toBe(5)
    expect(NATACAO_HOLD_BREATH_CD_BASE).toBe(15)
    expect(NATACAO_DROWN_STILL_SUBMERGED_HP_LOSS_D6).toBe(3)
  })

  it('armor penalty aplica dentro do uso', () => {
    if (usage.kind !== 'natacao') throw new Error('narrow failed')
    expect(usage.armorPenaltyApplies).toBe(true)
  })
})

describe('natacaoCd', () => {
  it.each([
    ['calma', 10],
    ['agitada', 15],
    ['tempestuosa', 20],
  ] as const)('%s → %s', (w, cd) => {
    expect(natacaoCd(w)).toBe(cd)
  })
})

describe('natacaoOutcome', () => {
  it('sucesso → advances', () => {
    expect(natacaoOutcome(15, 15)).toBe('advances')
    expect(natacaoOutcome(20, 15)).toBe('advances')
  })

  it('falha < 5 → floats', () => {
    expect(natacaoOutcome(11, 15)).toBe('floats')
  })

  it('falha ≥ 5 → sinks', () => {
    expect(natacaoOutcome(10, 15)).toBe('sinks')
  })
})

describe('prenderRespiracaoMaxRounds', () => {
  it('CON 5 → 6 rodadas', () => {
    expect(prenderRespiracaoMaxRounds(5)).toBe(6)
  })
})

describe('afogamentoCd', () => {
  it('sem anteriores → 15', () => {
    expect(afogamentoCd(0)).toBe(15)
  })

  it('após 4 testes → 19', () => {
    expect(afogamentoCd(4)).toBe(19)
  })
})

describe('Saltar — p116', () => {
  it('constantes verbatim', () => {
    expect(SALTAR_LONG_JUMP_CD_PER_1_5M).toBe(5)
    expect(SALTAR_HIGH_JUMP_CD_PER_1_5M).toBe(15)
    expect(SALTAR_MIN_RUN_UP_METERS).toBe(6)
    expect(SALTAR_NO_RUN_UP_CD_INCREASE).toBe(10)
  })

  it('parte do movimento', () => {
    const usage = atletismoUsageByKind('saltar')
    if (usage.kind !== 'saltar') throw new Error('narrow failed')
    expect(usage.action).toBe('parte-do-movimento')
  })
})

describe('saltarCd', () => {
  it('salto longo 3m → 10', () => {
    expect(saltarCd('longo', 3)).toBe(10)
  })

  it('salto longo 4.5m → 15', () => {
    expect(saltarCd('longo', 4.5)).toBe(15)
  })

  it('salto longo 6m → 20', () => {
    expect(saltarCd('longo', 6)).toBe(20)
  })

  it('salto altura 3m → 30', () => {
    expect(saltarCd('altura', 3)).toBe(30)
  })

  it('salto altura 4.5m → 45', () => {
    expect(saltarCd('altura', 4.5)).toBe(45)
  })

  it('salto longo 3m sem impulso → 20', () => {
    expect(saltarCd('longo', 3, false)).toBe(20)
  })

  it('salto altura 3m sem impulso → 40', () => {
    expect(saltarCd('altura', 3, false)).toBe(40)
  })
})

describe('atletismoUsageByKind', () => {
  it('throws se kind inválido', () => {
    expect(() =>
      // @ts-expect-error — invalid kind on purpose
      atletismoUsageByKind('lancar-peso'),
    ).toThrow(/unknown kind/)
  })

  it('resolve todos', () => {
    for (const k of ['corrida', 'escalar', 'natacao', 'saltar'] as const) {
      expect(atletismoUsageByKind(k).kind).toBe(k)
    }
  })
})
