import { describe, expect, it } from 'vitest'
import {
  COMPONENTE_MATERIAL_MUST_BE_IN_HAND,
  COST_PAID_ON_ACTIVATION,
  applyPenalidadePm,
  applySacrificioPm,
  componenteMaterial,
  describeSpecialCost,
  isPaidOnFailure,
  penalidadePm,
  removePenalidadePm,
  sacrificioPm,
} from '../ability-special-costs'

/**
 * PDF livro p224 — Custos Especiais de habilidade.
 */

describe('Constantes', () => {
  it('custo é pago na ativação (mesmo em falha)', () => {
    expect(COST_PAID_ON_ACTIVATION).toBe(true)
  })

  it('componente material precisa estar na mão', () => {
    expect(COMPONENTE_MATERIAL_MUST_BE_IN_HAND).toBe(true)
  })
})

describe('componenteMaterial', () => {
  it('sem valor em T$', () => {
    const c = componenteMaterial('pó de esmeralda')
    expect(c.kind).toBe('componente-material')
    expect(c.ingredient).toBe('pó de esmeralda')
    expect(c.tibarValue).toBeUndefined()
    expect(c.mustBeInHand).toBe(true)
    expect(c.consumedOnFailure).toBe(true)
    expect(c.bookPage).toBe(224)
  })

  it('com valor em T$', () => {
    const c = componenteMaterial('pó de diamante', 500)
    expect(c.tibarValue).toBe(500)
    expect(c.description).toContain('T$ 500')
  })

  it('T$ negativo lança', () => {
    expect(() => componenteMaterial('pó', -1)).toThrow(/tibarValue must be ≥ 0/)
  })
})

describe('penalidadePm', () => {
  it('cria com amount', () => {
    const p = penalidadePm(3)
    expect(p.kind).toBe('penalidade-pm')
    expect(p.amount).toBe(3)
    expect(p.temporary).toBe(true)
    expect(p.consumedOnFailure).toBe(true)
    expect(p.description).toContain('-3 PM')
  })

  it('amount ≤ 0 lança', () => {
    expect(() => penalidadePm(0)).toThrow(/amount must be > 0/)
    expect(() => penalidadePm(-2)).toThrow(/amount must be > 0/)
  })
})

describe('sacrificioPm', () => {
  it('cria com amount', () => {
    const s = sacrificioPm(2)
    expect(s.kind).toBe('sacrificio-pm')
    expect(s.amount).toBe(2)
    expect(s.permanent).toBe(true)
    expect(s.description).toContain('permanente')
  })

  it('amount ≤ 0 lança', () => {
    expect(() => sacrificioPm(0)).toThrow(/amount must be > 0/)
  })
})

describe('applyPenalidadePm', () => {
  it('reduz PM máximo', () => {
    expect(applyPenalidadePm(20, penalidadePm(5))).toBe(15)
  })

  it('não desce abaixo de 0', () => {
    expect(applyPenalidadePm(3, penalidadePm(5))).toBe(0)
  })

  it('currentMaxPm negativo lança', () => {
    expect(() => applyPenalidadePm(-1, penalidadePm(3))).toThrow(
      /currentMaxPm must be ≥ 0/,
    )
  })
})

describe('removePenalidadePm — reversão ao fim da duração', () => {
  it('restaura o PM máximo', () => {
    const p = penalidadePm(5)
    const reduced = applyPenalidadePm(20, p)
    expect(removePenalidadePm(reduced, p)).toBe(20)
  })

  it('funciona mesmo se atingiu o piso 0', () => {
    const p = penalidadePm(5)
    // 3 - 5 = 0 (clamp); remove volta 0 + 5 = 5
    const reduced = applyPenalidadePm(3, p)
    expect(removePenalidadePm(reduced, p)).toBe(5)
  })
})

describe('applySacrificioPm — permanente', () => {
  it('reduz PM máximo permanentemente', () => {
    expect(applySacrificioPm(20, sacrificioPm(2))).toBe(18)
  })

  it('não desce abaixo de 0', () => {
    expect(applySacrificioPm(1, sacrificioPm(5))).toBe(0)
  })
})

describe('isPaidOnFailure', () => {
  it('todos os custos especiais são consumidos em falha', () => {
    expect(isPaidOnFailure(componenteMaterial('pó'))).toBe(true)
    expect(isPaidOnFailure(penalidadePm(3))).toBe(true)
    expect(isPaidOnFailure(sacrificioPm(2))).toBe(true)
  })
})

describe('describeSpecialCost', () => {
  it('retorna descrição gerada', () => {
    expect(describeSpecialCost(componenteMaterial('sangue', 100))).toContain(
      'sangue',
    )
    expect(describeSpecialCost(penalidadePm(3))).toContain('Penalidade')
    expect(describeSpecialCost(sacrificioPm(1))).toContain('Sacrifício')
  })
})
