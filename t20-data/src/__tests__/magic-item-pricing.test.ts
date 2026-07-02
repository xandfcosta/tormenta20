import { describe, expect, it } from 'vitest'
import {
  MAX_ENCANTOS_PER_ITEM,
  MAX_MELHORIAS_PER_ITEM,
  OBRA_PRIMA_MELHORIAS,
  POCAO_PM_BY_CIRCULO,
  computeEncantadoCd,
  computeEncantadoPrice,
  pocaoCdByCirculo,
  pocaoPriceByCirculo,
} from '../magic-item-pricing'

/**
 * PDF Cap 8 p333-334. Worked examples verbatim:
 *  - Espada longa (T$ 15) + 1 encanto = T$ 18.015, CD 30
 *  - Espada longa + 4 melhorias (T$ 18.000 total) + 3 encantos = T$ 90.015
 */

describe('constants', () => {
  it('MAX_MELHORIAS_PER_ITEM = 4', () => {
    expect(MAX_MELHORIAS_PER_ITEM).toBe(4)
  })

  it('MAX_ENCANTOS_PER_ITEM = 3', () => {
    expect(MAX_ENCANTOS_PER_ITEM).toBe(3)
  })

  it('OBRA_PRIMA_MELHORIAS = 4', () => {
    expect(OBRA_PRIMA_MELHORIAS).toBe(4)
  })
})

describe('POCAO_PM_BY_CIRCULO', () => {
  it('1º círculo = 1 PM', () => {
    expect(POCAO_PM_BY_CIRCULO[1]).toBe(1)
  })

  it('2º círculo = 3 PM', () => {
    expect(POCAO_PM_BY_CIRCULO[2]).toBe(3)
  })

  it('3º círculo = 6 PM', () => {
    expect(POCAO_PM_BY_CIRCULO[3]).toBe(6)
  })

  it('4º círculo = 10 PM', () => {
    expect(POCAO_PM_BY_CIRCULO[4]).toBe(10)
  })

  it('5º círculo = 15 PM', () => {
    expect(POCAO_PM_BY_CIRCULO[5]).toBe(15)
  })

  it('frozen', () => {
    expect(Object.isFrozen(POCAO_PM_BY_CIRCULO)).toBe(true)
  })
})

describe('computeEncantadoPrice — worked examples p334', () => {
  it('espada longa T$ 15 + 1 encanto = T$ 18.015', () => {
    expect(
      computeEncantadoPrice({
        mundaneBasePrice: 15,
        melhoriaPrices: [],
        encantoCount: 1,
      }),
    ).toBe(18015)
  })

  it('espada longa + 4 melhorias (T$ 18.000) + 3 encantos = T$ 90.015', () => {
    expect(
      computeEncantadoPrice({
        mundaneBasePrice: 15,
        melhoriaPrices: [4500, 4500, 4500, 4500],
        encantoCount: 3,
      }),
    ).toBe(90015)
  })
})

describe('computeEncantadoPrice — outras combinações', () => {
  it('item mundano puro (0 melhorias, 0 encantos)', () => {
    expect(
      computeEncantadoPrice({
        mundaneBasePrice: 30,
        melhoriaPrices: [],
        encantoCount: 0,
      }),
    ).toBe(30)
  })

  it('item superior sem encantos: base + soma melhorias', () => {
    expect(
      computeEncantadoPrice({
        mundaneBasePrice: 100,
        melhoriaPrices: [300, 3000, 3000],
        encantoCount: 0,
      }),
    ).toBe(6400)
  })

  it('2 encantos sem melhoria: base + T$ 36.000', () => {
    expect(
      computeEncantadoPrice({
        mundaneBasePrice: 50,
        melhoriaPrices: [],
        encantoCount: 2,
      }),
    ).toBe(36050)
  })
})

describe('computeEncantadoPrice — validação', () => {
  it('throws se mundaneBasePrice negativo', () => {
    expect(() =>
      computeEncantadoPrice({
        mundaneBasePrice: -1,
        melhoriaPrices: [],
        encantoCount: 0,
      }),
    ).toThrow(/mundaneBasePrice/)
  })

  it('throws se melhoria price negativo', () => {
    expect(() =>
      computeEncantadoPrice({
        mundaneBasePrice: 15,
        melhoriaPrices: [300, -50],
        encantoCount: 0,
      }),
    ).toThrow(/melhoria price/)
  })

  it('throws se mais que MAX_MELHORIAS_PER_ITEM', () => {
    expect(() =>
      computeEncantadoPrice({
        mundaneBasePrice: 15,
        melhoriaPrices: [1, 2, 3, 4, 5],
        encantoCount: 0,
      }),
    ).toThrow(/max 4 melhorias/)
  })
})

describe('computeEncantadoCd', () => {
  it('0 encantos → CD 20 (superior puro)', () => {
    expect(computeEncantadoCd(0)).toBe(20)
  })

  it('1 encanto → CD 30 (worked example espada longa)', () => {
    expect(computeEncantadoCd(1)).toBe(30)
  })

  it('2 encantos → CD 35', () => {
    expect(computeEncantadoCd(2)).toBe(35)
  })

  it('3 encantos → CD 40', () => {
    expect(computeEncantadoCd(3)).toBe(40)
  })
})

describe('pocaoPriceByCirculo', () => {
  it('1º círculo → T$ 30', () => {
    expect(pocaoPriceByCirculo(1)).toBe(30)
  })

  it('2º círculo → T$ 270', () => {
    expect(pocaoPriceByCirculo(2)).toBe(270)
  })

  it('3º círculo → T$ 1.080', () => {
    expect(pocaoPriceByCirculo(3)).toBe(1080)
  })

  it('4º círculo → T$ 3.000', () => {
    expect(pocaoPriceByCirculo(4)).toBe(3000)
  })

  it('5º círculo → T$ 6.750', () => {
    expect(pocaoPriceByCirculo(5)).toBe(6750)
  })
})

describe('pocaoCdByCirculo', () => {
  it('1º círculo → CD 21', () => {
    expect(pocaoCdByCirculo(1)).toBe(21)
  })

  it('3º círculo → CD 26', () => {
    expect(pocaoCdByCirculo(3)).toBe(26)
  })

  it('5º círculo → CD 35', () => {
    expect(pocaoCdByCirculo(5)).toBe(35)
  })
})
