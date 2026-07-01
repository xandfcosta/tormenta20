import { describe, expect, it } from 'vitest'
import {
  mulberry32,
  pickOne,
  randInt,
  rollDice,
  rollDie,
  rollFormula,
  rollPercentile,
} from '../loot-rng'

describe('mulberry32 — seedable PRNG', () => {
  it('produz mesma sequência para mesma seed', () => {
    const a = mulberry32(42)
    const b = mulberry32(42)
    for (let i = 0; i < 100; i++) {
      expect(a.next()).toBe(b.next())
    }
  })

  it('produz sequências diferentes para seeds diferentes', () => {
    const a = mulberry32(1)
    const b = mulberry32(2)
    let differentCount = 0
    for (let i = 0; i < 100; i++) {
      if (a.next() !== b.next()) differentCount++
    }
    expect(differentCount).toBeGreaterThan(95)
  })

  it('valores em [0, 1)', () => {
    const rng = mulberry32(123)
    for (let i = 0; i < 1000; i++) {
      const v = rng.next()
      expect(v).toBeGreaterThanOrEqual(0)
      expect(v).toBeLessThan(1)
    }
  })
})

describe('randInt', () => {
  it('range [1, 6] inclusive', () => {
    const rng = mulberry32(7)
    const seen = new Set<number>()
    for (let i = 0; i < 200; i++) {
      const v = randInt(rng, 1, 6)
      expect(v).toBeGreaterThanOrEqual(1)
      expect(v).toBeLessThanOrEqual(6)
      seen.add(v)
    }
    expect(seen.size).toBe(6) // todos os 6 valores aparecem
  })

  it('throws quando min > max', () => {
    expect(() => randInt(mulberry32(1), 5, 1)).toThrow(/min 5/)
  })

  it('min === max sempre retorna esse valor', () => {
    const rng = mulberry32(42)
    for (let i = 0; i < 10; i++) {
      expect(randInt(rng, 7, 7)).toBe(7)
    }
  })
})

describe('rollDie', () => {
  it('d6 em [1, 6]', () => {
    const rng = mulberry32(9)
    for (let i = 0; i < 100; i++) {
      const v = rollDie(rng, 6)
      expect(v).toBeGreaterThanOrEqual(1)
      expect(v).toBeLessThanOrEqual(6)
    }
  })

  it('throws quando sides < 1', () => {
    expect(() => rollDie(mulberry32(1), 0)).toThrow(/sides/)
  })
})

describe('rollDice', () => {
  it('2d6 em [2, 12]', () => {
    const rng = mulberry32(13)
    for (let i = 0; i < 100; i++) {
      const v = rollDice(rng, 2, 6)
      expect(v).toBeGreaterThanOrEqual(2)
      expect(v).toBeLessThanOrEqual(12)
    }
  })

  it('0 dados = 0', () => {
    expect(rollDice(mulberry32(1), 0, 6)).toBe(0)
  })

  it('throws quando count < 0', () => {
    expect(() => rollDice(mulberry32(1), -1, 6)).toThrow(/count/)
  })
})

describe('rollPercentile', () => {
  it('valores em [1, 100]', () => {
    const rng = mulberry32(99)
    for (let i = 0; i < 1000; i++) {
      const v = rollPercentile(rng)
      expect(v).toBeGreaterThanOrEqual(1)
      expect(v).toBeLessThanOrEqual(100)
    }
  })

  it('determinismo: mesma seed → mesma sequência', () => {
    const a = mulberry32(2026)
    const b = mulberry32(2026)
    for (let i = 0; i < 10; i++) {
      expect(rollPercentile(a)).toBe(rollPercentile(b))
    }
  })
})

describe('rollFormula — parse + roll', () => {
  it('"4d6×10" em range [40, 240]', () => {
    const rng = mulberry32(33)
    for (let i = 0; i < 50; i++) {
      const v = rollFormula(rng, '4d6×10')
      expect(v).toBeGreaterThanOrEqual(40)
      expect(v).toBeLessThanOrEqual(240)
    }
  })

  it('"1d4×10" em range [10, 40]', () => {
    const rng = mulberry32(34)
    for (let i = 0; i < 50; i++) {
      const v = rollFormula(rng, '1d4×10')
      expect(v).toBeGreaterThanOrEqual(10)
      expect(v).toBeLessThanOrEqual(40)
    }
  })

  it('"2d6+1×100" = (2d6+1)×100 em range [300, 1300]', () => {
    const rng = mulberry32(35)
    for (let i = 0; i < 50; i++) {
      const v = rollFormula(rng, '2d6+1×100')
      expect(v).toBeGreaterThanOrEqual(300) // (2+1)×100
      expect(v).toBeLessThanOrEqual(1300) // (12+1)×100
    }
  })

  it('"2d10×1.000" com vírgula-milhar em range [2000, 20000]', () => {
    const rng = mulberry32(36)
    for (let i = 0; i < 50; i++) {
      const v = rollFormula(rng, '2d10×1.000')
      expect(v).toBeGreaterThanOrEqual(2000)
      expect(v).toBeLessThanOrEqual(20000)
    }
  })

  it('"1d6" simples em [1, 6]', () => {
    const rng = mulberry32(37)
    for (let i = 0; i < 50; i++) {
      const v = rollFormula(rng, '1d6')
      expect(v).toBeGreaterThanOrEqual(1)
      expect(v).toBeLessThanOrEqual(6)
    }
  })

  it('throws para fórmula inválida', () => {
    expect(() => rollFormula(mulberry32(1), 'foo')).toThrow(/cannot parse/)
  })

  it('determinismo: mesma seed → mesmo resultado', () => {
    const a = mulberry32(777)
    const b = mulberry32(777)
    expect(rollFormula(a, '4d6×10')).toBe(rollFormula(b, '4d6×10'))
  })
})

describe('pickOne', () => {
  it('retorna elemento existente do array', () => {
    const arr = ['a', 'b', 'c', 'd']
    const rng = mulberry32(50)
    for (let i = 0; i < 100; i++) {
      expect(arr).toContain(pickOne(rng, arr))
    }
  })

  it('throws em array vazio', () => {
    expect(() => pickOne(mulberry32(1), [])).toThrow(/empty/)
  })
})
