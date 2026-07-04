import { describe, expect, it } from 'vitest'
import {
  ARMOR_SHIELD_PLUS_ONE_ITEM_STACKS,
  MIN_HIT_CHANCE,
  MISS_CHANCE_CAP,
  MODIFIER_TIMING,
  NON_SELF_STACKING_SOURCES,
  PM_COST_FLOOR,
  ROUNDING_RULE,
  SELF_STACKING_SOURCES,
  bestBonusOnly,
  capMissChance,
  combineMultipliers,
  floorPmCost,
  hitChanceAfterMissCap,
  roundDown,
  sameEffectStacks,
  stacksWith,
  sumStackingBonuses,
} from '../modifier-stacking'

/**
 * PDF livro p226 — Acumulando Efeitos.
 */

describe('Constantes', () => {
  it('valores verbatim livro', () => {
    expect(MISS_CHANCE_CAP).toBe(0.75)
    expect(MIN_HIT_CHANCE).toBe(0.25)
    expect(PM_COST_FLOOR).toBe(1)
    expect(MODIFIER_TIMING).toBe('before-roll')
    expect(ROUNDING_RULE).toBe('floor')
    expect(ARMOR_SHIELD_PLUS_ONE_ITEM_STACKS).toBe(true)
  })
})

describe('SELF_STACKING_SOURCES / NON_SELF_STACKING_SOURCES', () => {
  it('habilidades e perícias empilham consigo mesmas (efeitos distintos)', () => {
    expect(SELF_STACKING_SOURCES.has('habilidade')).toBe(true)
    expect(SELF_STACKING_SOURCES.has('pericia')).toBe(true)
  })

  it('itens/magias/parceiros/ambiente NÃO empilham consigo mesmos', () => {
    expect(NON_SELF_STACKING_SOURCES.has('item')).toBe(true)
    expect(NON_SELF_STACKING_SOURCES.has('magia')).toBe(true)
    expect(NON_SELF_STACKING_SOURCES.has('parceiro')).toBe(true)
    expect(NON_SELF_STACKING_SOURCES.has('ambiente')).toBe(true)
  })
})

describe('stacksWith — matriz p226', () => {
  it('fontes diferentes SEMPRE empilham (item + magia = 2)', () => {
    expect(stacksWith('item', 'magia')).toBe(true)
    expect(stacksWith('habilidade', 'ambiente')).toBe(true)
    expect(stacksWith('pericia', 'parceiro')).toBe(true)
  })

  it('mesma fonte item: NÃO empilha', () => {
    expect(stacksWith('item', 'item')).toBe(false)
  })

  it('mesma fonte magia: NÃO empilha', () => {
    expect(stacksWith('magia', 'magia')).toBe(false)
  })

  it('mesma fonte parceiro/ambiente: NÃO empilha', () => {
    expect(stacksWith('parceiro', 'parceiro')).toBe(false)
    expect(stacksWith('ambiente', 'ambiente')).toBe(false)
  })

  it('mesma fonte habilidade: empilha (efeitos distintos)', () => {
    expect(stacksWith('habilidade', 'habilidade')).toBe(true)
  })

  it('mesma fonte perícia: empilha (efeitos distintos)', () => {
    expect(stacksWith('pericia', 'pericia')).toBe(true)
  })
})

describe('sameEffectStacks', () => {
  it('mesmo effectId → NÃO empilha (única aplicação)', () => {
    expect(sameEffectStacks('inspiracao-heroica', 'inspiracao-heroica')).toBe(
      false,
    )
  })

  it('effectIds diferentes → empilha (caller ainda checa stacksWith)', () => {
    expect(sameEffectStacks('inspiracao-heroica', 'ataque-preciso')).toBe(true)
  })
})

describe('capMissChance', () => {
  it('acima do cap → 0.75', () => {
    expect(capMissChance(0.9)).toBe(0.75)
    expect(capMissChance(1.0)).toBe(0.75)
  })

  it('abaixo do cap → sem alteração', () => {
    expect(capMissChance(0.5)).toBe(0.5)
    expect(capMissChance(0)).toBe(0)
  })

  it('negativo lança', () => {
    expect(() => capMissChance(-0.1)).toThrow(/chance must be ≥ 0/)
  })
})

describe('hitChanceAfterMissCap', () => {
  it('miss ≥ 75% → hit = 25% (mínimo garantido)', () => {
    expect(hitChanceAfterMissCap(0.8)).toBe(0.25)
    expect(hitChanceAfterMissCap(1.0)).toBe(0.25)
  })

  it('miss 50% → hit 50%', () => {
    expect(hitChanceAfterMissCap(0.5)).toBe(0.5)
  })

  it('miss 0% → hit 100%', () => {
    expect(hitChanceAfterMissCap(0)).toBe(1)
  })
})

describe('floorPmCost', () => {
  it('reducedCost 3 → 3', () => {
    expect(floorPmCost(3)).toBe(3)
  })

  it('reducedCost 0 → 1 (piso)', () => {
    expect(floorPmCost(0)).toBe(1)
  })

  it('reducedCost negativo → 1', () => {
    expect(floorPmCost(-2)).toBe(1)
  })
})

describe('combineMultipliers — p226', () => {
  it('lista vazia → 1', () => {
    expect(combineMultipliers([])).toBe(1)
  })

  it('um multiplicador → aplica cheio', () => {
    expect(combineMultipliers([2])).toBe(2)
    expect(combineMultipliers([3])).toBe(3)
  })

  it('2× + 2× → 3× (dobrar duas vezes = triplicar)', () => {
    expect(combineMultipliers([2, 2])).toBe(3)
  })

  it('2× + 3× → 4×', () => {
    expect(combineMultipliers([2, 3])).toBe(4)
  })

  it('2× + 2× + 2× → 4×', () => {
    expect(combineMultipliers([2, 2, 2])).toBe(4)
  })

  it('3× + 3× → 5×', () => {
    expect(combineMultipliers([3, 3])).toBe(5)
  })

  it('multiplier < 1 lança', () => {
    expect(() => combineMultipliers([2, 0.5])).toThrow(/must be ≥ 1/)
  })
})

describe('roundDown', () => {
  it('5.9 → 5', () => {
    expect(roundDown(5.9)).toBe(5)
  })

  it('-0.5 → -1 (floor)', () => {
    expect(roundDown(-0.5)).toBe(-1)
  })
})

describe('bestBonusOnly', () => {
  it('vazia → 0', () => {
    expect(bestBonusOnly([])).toBe(0)
  })

  it('pega o maior', () => {
    expect(bestBonusOnly([1, 3, 2])).toBe(3)
  })

  it('funciona com negativos', () => {
    expect(bestBonusOnly([-2, -5, -1])).toBe(-1)
  })
})

describe('sumStackingBonuses', () => {
  it('soma tudo', () => {
    expect(sumStackingBonuses([1, 2, 3])).toBe(6)
  })

  it('vazia → 0', () => {
    expect(sumStackingBonuses([])).toBe(0)
  })
})
