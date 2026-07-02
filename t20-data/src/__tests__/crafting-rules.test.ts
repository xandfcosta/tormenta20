import { describe, expect, it } from 'vitest'
import {
  CONSUMABLE_DOUBLE_PENALTY,
  CRAFT_TIME_DAYS,
  ENCANTO_TIERS,
  ENGENHOCA_CRAFT_TIME_DAYS,
  MATERIAL_COST_FRACTION,
  OFICIO_CD_COMPLEX,
  OFICIO_CD_SIMPLE,
  REPAIR_MATERIAL_FRACTION,
  encantoCdBonus,
  encantoPriceBonus,
  engenhocaCd,
  engenhocaPrice,
  materialCost,
  repairCost,
  scrollPotionCd,
  scrollPotionPrice,
  tierFromEncantoCount,
} from '../crafting-rules'

/**
 * PDF Cap 2 p121 (Ofício Fabricar), Cap 8 p333-334 (pergaminho/poção
 * + itens mágicos + Tabela 8-7). Pinned:
 *  - Simple CD 15, Complex CD 20
 *  - Material = 1/3 preço; repair = 1/10
 *  - Consumable double penalty -5
 *  - Time 1/7/30 dias
 *  - Scroll/potion: T$ 30 × PM² / CD 20 + PM
 *  - Encantos: 1=T$18k/CD+10, 2=T$36k/CD+15, 3=T$72k/CD+20
 *  - Engenhoca: T$ 100 × PM / CD 20 + PM / 1 semana
 */

describe('OFICIO constants', () => {
  it('CD simple = 15', () => {
    expect(OFICIO_CD_SIMPLE).toBe(15)
  })

  it('CD complex = 20', () => {
    expect(OFICIO_CD_COMPLEX).toBe(20)
  })

  it('material 1/3', () => {
    expect(MATERIAL_COST_FRACTION).toBeCloseTo(1 / 3)
  })

  it('repair 1/10', () => {
    expect(REPAIR_MATERIAL_FRACTION).toBe(1 / 10)
  })

  it('consumable double = -5', () => {
    expect(CONSUMABLE_DOUBLE_PENALTY).toBe(-5)
  })
})

describe('CRAFT_TIME_DAYS', () => {
  it('consumable = 1 dia', () => {
    expect(CRAFT_TIME_DAYS.consumable).toBe(1)
  })

  it('non-consumable-common = 7 dias', () => {
    expect(CRAFT_TIME_DAYS['non-consumable-common']).toBe(7)
  })

  it('non-consumable-superior = 30 dias', () => {
    expect(CRAFT_TIME_DAYS['non-consumable-superior']).toBe(30)
  })
})

describe('materialCost', () => {
  it('T$ 300 → T$ 100', () => {
    expect(materialCost(300)).toBe(100)
  })

  it('T$ 999 → T$ 333', () => {
    expect(materialCost(999)).toBe(333)
  })

  it('T$ 100 → T$ 34 (ceil)', () => {
    expect(materialCost(100)).toBe(34)
  })

  it('T$ 0 → 0', () => {
    expect(materialCost(0)).toBe(0)
  })

  it('throws se negativo', () => {
    expect(() => materialCost(-1)).toThrow(/finalPrice/)
  })
})

describe('repairCost', () => {
  it('T$ 500 → T$ 50', () => {
    expect(repairCost(500)).toBe(50)
  })

  it('T$ 100 → T$ 10', () => {
    expect(repairCost(100)).toBe(10)
  })

  it('T$ 99 → T$ 10 (ceil)', () => {
    expect(repairCost(99)).toBe(10)
  })

  it('throws se negativo', () => {
    expect(() => repairCost(-1)).toThrow(/originalPrice/)
  })
})

describe('scrollPotionPrice', () => {
  it('1º círculo (1 PM) → T$ 30', () => {
    expect(scrollPotionPrice(1)).toBe(30)
  })

  it('2º círculo (3 PM) → T$ 270', () => {
    expect(scrollPotionPrice(3)).toBe(270)
  })

  it('3º círculo (6 PM) → T$ 1080', () => {
    expect(scrollPotionPrice(6)).toBe(1080)
  })

  it('0 PM → min 1', () => {
    expect(scrollPotionPrice(0)).toBe(1)
  })

  it('throws se negativo', () => {
    expect(() => scrollPotionPrice(-1)).toThrow(/pmCost/)
  })
})

describe('scrollPotionCd', () => {
  it('1 PM → CD 21', () => {
    expect(scrollPotionCd(1)).toBe(21)
  })

  it('6 PM → CD 26', () => {
    expect(scrollPotionCd(6)).toBe(26)
  })

  it('throws se negativo', () => {
    expect(() => scrollPotionCd(-1)).toThrow(/pmCost/)
  })
})

describe('ENCANTO_TIERS — Tabela 8-7', () => {
  it('1 encanto: T$ 18000 / CD +10', () => {
    expect(ENCANTO_TIERS[1]).toEqual({ priceBonus: 18000, cdBonus: 10 })
  })

  it('2 encantos: T$ 36000 / CD +15', () => {
    expect(ENCANTO_TIERS[2]).toEqual({ priceBonus: 36000, cdBonus: 15 })
  })

  it('3 encantos: T$ 72000 / CD +20', () => {
    expect(ENCANTO_TIERS[3]).toEqual({ priceBonus: 72000, cdBonus: 20 })
  })

  it('frozen', () => {
    expect(Object.isFrozen(ENCANTO_TIERS)).toBe(true)
  })
})

describe('encantoPriceBonus + encantoCdBonus', () => {
  it('helpers batem com Tabela 8-7', () => {
    expect(encantoPriceBonus(1)).toBe(18000)
    expect(encantoCdBonus(2)).toBe(15)
    expect(encantoPriceBonus(3)).toBe(72000)
  })
})

describe('tierFromEncantoCount', () => {
  it('1 → menor', () => {
    expect(tierFromEncantoCount(1)).toBe('menor')
  })

  it('2 → medio', () => {
    expect(tierFromEncantoCount(2)).toBe('medio')
  })

  it('3 → maior', () => {
    expect(tierFromEncantoCount(3)).toBe('maior')
  })
})

describe('engenhocaPrice', () => {
  it('1 PM → T$ 100', () => {
    expect(engenhocaPrice(1)).toBe(100)
  })

  it('3 PM → T$ 300', () => {
    expect(engenhocaPrice(3)).toBe(300)
  })

  it('throws se negativo', () => {
    expect(() => engenhocaPrice(-1)).toThrow(/pmCost/)
  })
})

describe('engenhocaCd', () => {
  it('3 PM → CD 23', () => {
    expect(engenhocaCd(3)).toBe(23)
  })

  it('0 PM → CD 20', () => {
    expect(engenhocaCd(0)).toBe(20)
  })

  it('throws se negativo', () => {
    expect(() => engenhocaCd(-1)).toThrow(/pmCost/)
  })
})

describe('ENGENHOCA_CRAFT_TIME_DAYS', () => {
  it('sempre 1 semana', () => {
    expect(ENGENHOCA_CRAFT_TIME_DAYS).toBe(7)
  })
})
