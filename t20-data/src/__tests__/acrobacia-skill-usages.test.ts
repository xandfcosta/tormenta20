import { describe, expect, it } from 'vitest'
import {
  ACROBACIA_ARMOR_PENALTY,
  ACROBACIA_TRAINED_ONLY,
  ACROBACIA_USAGES,
  AMORTECER_QUEDA_CD,
  AMORTECER_QUEDA_EXTRA_DICE_PER_MARGIN,
  AMORTECER_QUEDA_TRAINED_ONLY,
  EQUILIBRIO_CD_ESCORREGADIO,
  EQUILIBRIO_CD_ESTREITA,
  EQUILIBRIO_CD_MUITO_ESTREITA,
  EQUILIBRIO_FALL_MARGIN,
  EQUILIBRIO_FULL_SPEED_PENALTY,
  ESCAPAR_CD_ALGEMAS,
  ESCAPAR_CD_REDES,
  ESCAPAR_CORDAS_BINDER_BONUS,
  ESPACO_APERTADO_CD,
  ESPACO_APERTADO_SIZE_STEPS_SMALLER,
  ESPACO_APERTADO_TRAINED_ONLY,
  LEVANTAR_SE_RAPIDAMENTE_CD,
  LEVANTAR_SE_RAPIDAMENTE_TRAINED_ONLY,
  PASSAR_POR_INIMIGO_OPPOSED_BY,
  acrobaciaUsageByKind,
  amortecerQuedaD6ReductionCount,
  amortecerQuedaOutcome,
  equilibrioCd,
  equilibrioFullSpeedPenalty,
  equilibrioOutcome,
  escaparCd,
  passarPorInimigoOpposedRoll,
  passarPorInimigoOutcome,
} from '../acrobacia-skill-usages'

/**
 * PDF livro p115 — Perícia Acrobacia (DES, com penalidade de armadura).
 *  1. Amortecer Queda — CD 15, apenas treinado, reação
 *  2. Equilíbrio — CDs 10/15/20 por superfície, movimento
 *  3. Escapar — cordas Des+10, redes 20, algemas 30, completa
 *  4. Levantar-se Rapidamente — CD 20, apenas treinado
 *  5. Passar por Espaço Apertado — CD 25, apenas treinado
 *  6. Passar por Inimigo — oposto vs melhor de Acrob/Ini/Luta
 */

describe('ACROBACIA_USAGES — shape', () => {
  it('exatamente 6 usos', () => {
    expect(ACROBACIA_USAGES.length).toBe(6)
  })

  it('frozen', () => {
    expect(Object.isFrozen(ACROBACIA_USAGES)).toBe(true)
  })

  it('ids canônicos', () => {
    expect(ACROBACIA_USAGES.map((u) => u.id).sort()).toEqual([
      'amortecer-queda',
      'equilibrio',
      'escapar',
      'levantar-se-rapidamente',
      'passar-por-espaco-apertado',
      'passar-por-inimigo',
    ])
  })

  it('todos em p115', () => {
    for (const u of ACROBACIA_USAGES) {
      expect(u.bookPage).toBe(115)
    }
  })
})

describe('Perícia flags — Tabela 2-1', () => {
  it('NÃO é somente treinada', () => {
    expect(ACROBACIA_TRAINED_ONLY).toBe(false)
  })

  it('com penalidade de armadura', () => {
    expect(ACROBACIA_ARMOR_PENALTY).toBe(true)
  })
})

describe('Amortecer Queda — p115', () => {
  const usage = acrobaciaUsageByKind('amortecer-queda')

  it('CD 15 apenas treinado reação', () => {
    expect(AMORTECER_QUEDA_CD).toBe(15)
    expect(AMORTECER_QUEDA_TRAINED_ONLY).toBe(true)
    if (usage.kind !== 'amortecer-queda') throw new Error('narrow failed')
    expect(usage.action).toBe('reacao')
    expect(usage.trainedOnly).toBe(true)
  })

  it('extra dice por 5 pts de margem', () => {
    expect(AMORTECER_QUEDA_EXTRA_DICE_PER_MARGIN).toBe(5)
  })

  it('dano zerado = cai de pé', () => {
    if (usage.kind !== 'amortecer-queda') throw new Error('narrow failed')
    expect(usage.zeroDamageLandsOnFeet).toBe(true)
  })
})

describe('amortecerQuedaD6ReductionCount', () => {
  it('roll 14 → 0 (falha)', () => {
    expect(amortecerQuedaD6ReductionCount(14)).toBe(0)
  })

  it('roll 15 → 1', () => {
    expect(amortecerQuedaD6ReductionCount(15)).toBe(1)
  })

  it('roll 19 → 1 (só 4 acima)', () => {
    expect(amortecerQuedaD6ReductionCount(19)).toBe(1)
  })

  it('roll 20 → 2 (5 acima = +1d6)', () => {
    expect(amortecerQuedaD6ReductionCount(20)).toBe(2)
  })

  it('roll 30 → 4 (15 acima = +3d6)', () => {
    expect(amortecerQuedaD6ReductionCount(30)).toBe(4)
  })
})

describe('amortecerQuedaOutcome', () => {
  it('roll 14 → failed', () => {
    expect(amortecerQuedaOutcome(14, 20)).toBe('failed')
  })

  it('roll 20, dano final 5 → reduced', () => {
    expect(amortecerQuedaOutcome(20, 5)).toBe('reduced')
  })

  it('roll 25, dano final 0 → landed-on-feet', () => {
    expect(amortecerQuedaOutcome(25, 0)).toBe('landed-on-feet')
  })

  it('roll 25, dano final -3 → landed-on-feet', () => {
    expect(amortecerQuedaOutcome(25, -3)).toBe('landed-on-feet')
  })
})

describe('Equilíbrio — p115', () => {
  it('CDs verbatim', () => {
    expect(EQUILIBRIO_CD_ESCORREGADIO).toBe(10)
    expect(EQUILIBRIO_CD_ESTREITA).toBe(15)
    expect(EQUILIBRIO_CD_MUITO_ESTREITA).toBe(20)
  })

  it('full-speed -5 e margem de queda 5', () => {
    expect(EQUILIBRIO_FULL_SPEED_PENALTY).toBe(-5)
    expect(EQUILIBRIO_FALL_MARGIN).toBe(5)
  })

  it('desprevenido + dano força novo teste', () => {
    const usage = acrobaciaUsageByKind('equilibrio')
    if (usage.kind !== 'equilibrio') throw new Error('narrow failed')
    expect(usage.isFlatFooted).toBe(true)
    expect(usage.damageForcesNewTest).toBe(true)
  })
})

describe('equilibrioCd', () => {
  it('piso-escorregadio → 10', () => {
    expect(equilibrioCd('piso-escorregadio')).toBe(10)
  })

  it('estreita → 15', () => {
    expect(equilibrioCd('estreita')).toBe(15)
  })

  it('muito-estreita → 20', () => {
    expect(equilibrioCd('muito-estreita')).toBe(20)
  })
})

describe('equilibrioFullSpeedPenalty', () => {
  it('full speed → -5', () => {
    expect(equilibrioFullSpeedPenalty(true)).toBe(-5)
  })

  it('metade → 0', () => {
    expect(equilibrioFullSpeedPenalty(false)).toBe(0)
  })
})

describe('equilibrioOutcome', () => {
  it('roll ≥ CD → success', () => {
    expect(equilibrioOutcome(15, 15)).toBe('success')
    expect(equilibrioOutcome(20, 15)).toBe('success')
  })

  it('falha por < 5 → no-progress', () => {
    expect(equilibrioOutcome(14, 15)).toBe('no-progress')
    expect(equilibrioOutcome(11, 15)).toBe('no-progress')
  })

  it('falha por 5 → falls', () => {
    expect(equilibrioOutcome(10, 15)).toBe('falls')
  })

  it('falha por 10 → falls', () => {
    expect(equilibrioOutcome(10, 20)).toBe('falls')
  })
})

describe('Escapar — p115', () => {
  it('CDs verbatim', () => {
    expect(ESCAPAR_CD_REDES).toBe(20)
    expect(ESCAPAR_CD_ALGEMAS).toBe(30)
    expect(ESCAPAR_CORDAS_BINDER_BONUS).toBe(10)
  })

  it('ação completa', () => {
    const usage = acrobaciaUsageByKind('escapar')
    if (usage.kind !== 'escapar') throw new Error('narrow failed')
    expect(usage.action).toBe('completa')
  })
})

describe('escaparCd', () => {
  it('cordas → Des roll + 10', () => {
    expect(escaparCd({ kind: 'cordas', binderDestrezaRoll: 18 })).toBe(28)
  })

  it('redes → 20', () => {
    expect(escaparCd({ kind: 'redes' })).toBe(20)
  })

  it('algemas → 30', () => {
    expect(escaparCd({ kind: 'algemas' })).toBe(30)
  })
})

describe('Levantar-se Rapidamente — p115', () => {
  const usage = acrobaciaUsageByKind('levantar-se-rapidamente')

  it('CD 20 apenas treinado', () => {
    expect(LEVANTAR_SE_RAPIDAMENTE_CD).toBe(20)
    expect(LEVANTAR_SE_RAPIDAMENTE_TRAINED_ONLY).toBe(true)
    if (usage.kind !== 'levantar-se-rapidamente') throw new Error('narrow failed')
    expect(usage.dc).toBe(20)
    expect(usage.trainedOnly).toBe(true)
  })

  it('sucesso vira ação livre', () => {
    if (usage.kind !== 'levantar-se-rapidamente') throw new Error('narrow failed')
    expect(usage.successPromotesToFreeAction).toBe(true)
  })
})

describe('Passar por Espaço Apertado — p115', () => {
  const usage = acrobaciaUsageByKind('passar-por-espaco-apertado')

  it('CD 25 apenas treinado', () => {
    expect(ESPACO_APERTADO_CD).toBe(25)
    expect(ESPACO_APERTADO_TRAINED_ONLY).toBe(true)
    if (usage.kind !== 'passar-por-espaco-apertado') throw new Error('narrow failed')
    expect(usage.dc).toBe(25)
  })

  it('1 categoria menor + metade deslocamento', () => {
    expect(ESPACO_APERTADO_SIZE_STEPS_SMALLER).toBe(1)
    if (usage.kind !== 'passar-por-espaco-apertado') throw new Error('narrow failed')
    expect(usage.sizeCategoriesSmaller).toBe(1)
    expect(usage.halfSpeedMovement).toBe(true)
  })
})

describe('Passar por Inimigo — p115', () => {
  const usage = acrobaciaUsageByKind('passar-por-inimigo')

  it('oposto vs Acrobacia/Iniciativa/Luta (o melhor)', () => {
    expect(PASSAR_POR_INIMIGO_OPPOSED_BY).toEqual([
      'acrobacia',
      'iniciativa',
      'luta',
    ])
    if (usage.kind !== 'passar-por-inimigo') throw new Error('narrow failed')
    expect(usage.opposedByBestOf).toEqual(['acrobacia', 'iniciativa', 'luta'])
  })

  it('espaço = terreno difícil', () => {
    if (usage.kind !== 'passar-por-inimigo') throw new Error('narrow failed')
    expect(usage.countsAsDifficultTerrain).toBe(true)
  })

  it('opposed frozen', () => {
    expect(Object.isFrozen(PASSAR_POR_INIMIGO_OPPOSED_BY)).toBe(true)
  })
})

describe('passarPorInimigoOpposedRoll', () => {
  it('pega o maior dos três', () => {
    expect(passarPorInimigoOpposedRoll(10, 15, 12)).toBe(15)
    expect(passarPorInimigoOpposedRoll(20, 15, 12)).toBe(20)
    expect(passarPorInimigoOpposedRoll(10, 5, 25)).toBe(25)
  })
})

describe('passarPorInimigoOutcome', () => {
  it('roll > oponente → crosses', () => {
    expect(passarPorInimigoOutcome(20, 15)).toBe('crosses')
  })

  it('empate → crosses (atacante vence)', () => {
    expect(passarPorInimigoOutcome(15, 15)).toBe('crosses')
  })

  it('roll < oponente → blocked', () => {
    expect(passarPorInimigoOutcome(10, 15)).toBe('blocked')
  })
})

describe('acrobaciaUsageByKind', () => {
  it('throws se kind inválido', () => {
    expect(() =>
      // @ts-expect-error — invalid kind on purpose
      acrobaciaUsageByKind('cambalhota'),
    ).toThrow(/unknown kind/)
  })

  it('resolve todos', () => {
    for (const k of [
      'amortecer-queda',
      'equilibrio',
      'escapar',
      'levantar-se-rapidamente',
      'passar-por-espaco-apertado',
      'passar-por-inimigo',
    ] as const) {
      expect(acrobaciaUsageByKind(k).kind).toBe(k)
    }
  })
})
