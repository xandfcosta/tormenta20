import { describe, expect, it } from 'vitest'
import {
  AREA_ORIGIN_BY_SHAPE,
  CONE_DEFAULT_LENGTHS_M,
  CUBO_DEFAULT_EDGES_M,
  ESFERA_DEFAULT_RADII_M,
  HIDDEN_AREA_PERCEPTION_CD,
  LINHA_DEFAULT_LENGTH_M,
  LINHA_DEFAULT_WIDTH_M,
  REDIRECT_AREA_ACTION,
  SQUARE_METERS,
  areaOrigin,
  coneSquareCount,
  cuboSquareCount,
  esferaSquareCount,
  isDefaultConeLength,
  isDefaultCuboEdge,
  isDefaultEsferaRadius,
  linhaSquareCount,
  metersToSquares,
  squaresToMeters,
} from '../spell-areas'

/**
 * PDF livro p225 — Áreas de Efeitos (Cap 4 Magia).
 */

describe('Constantes verbatim', () => {
  it('escala 1 quadrado = 1,5m', () => {
    expect(SQUARE_METERS).toBe(1.5)
  })

  it('cone default lengths 4,5m / 6m / 9m', () => {
    expect(CONE_DEFAULT_LENGTHS_M).toEqual([4.5, 6, 9])
  })

  it('cubo default edges 1,5m / 3m', () => {
    expect(CUBO_DEFAULT_EDGES_M).toEqual([1.5, 3])
  })

  it('esfera default radii 1,5m / 3m / 6m', () => {
    expect(ESFERA_DEFAULT_RADII_M).toEqual([1.5, 3, 6])
  })

  it('linha default 15m × 1,5m', () => {
    expect(LINHA_DEFAULT_LENGTH_M).toBe(15)
    expect(LINHA_DEFAULT_WIDTH_M).toBe(1.5)
  })

  it('templates frozen', () => {
    expect(Object.isFrozen(CONE_DEFAULT_LENGTHS_M)).toBe(true)
    expect(Object.isFrozen(CUBO_DEFAULT_EDGES_M)).toBe(true)
    expect(Object.isFrozen(ESFERA_DEFAULT_RADII_M)).toBe(true)
  })

  it('hidden area Percepção CD 20', () => {
    expect(HIDDEN_AREA_PERCEPTION_CD).toBe(20)
  })

  it('redirecionar = ação padrão', () => {
    expect(REDIRECT_AREA_ACTION).toBe('padrao')
  })
})

describe('AREA_ORIGIN_BY_SHAPE / areaOrigin', () => {
  it('frozen', () => {
    expect(Object.isFrozen(AREA_ORIGIN_BY_SHAPE)).toBe(true)
  })

  it('Cone / Linha surgem adjacentes ao conjurador', () => {
    expect(areaOrigin('cone')).toBe('adjacent-to-caster')
    expect(areaOrigin('linha')).toBe('adjacent-to-caster')
  })

  it('Cilindro / Esfera / Cubo / Quadrado surgem em intersecção', () => {
    expect(areaOrigin('cilindro')).toBe('four-square-intersection')
    expect(areaOrigin('esfera')).toBe('four-square-intersection')
    expect(areaOrigin('cubo')).toBe('four-square-intersection')
    expect(areaOrigin('quadrado')).toBe('four-square-intersection')
  })
})

describe('metersToSquares / squaresToMeters', () => {
  it('1,5m → 1 quadrado', () => {
    expect(metersToSquares(1.5)).toBe(1)
  })

  it('9m → 6 quadrados', () => {
    expect(metersToSquares(9)).toBe(6)
  })

  it('4,5m → 3 quadrados', () => {
    expect(metersToSquares(4.5)).toBe(3)
  })

  it('0 → 0', () => {
    expect(metersToSquares(0)).toBe(0)
  })

  it('metros negativos lançam', () => {
    expect(() => metersToSquares(-1.5)).toThrow(/meters must be ≥ 0/)
  })

  it('3 quadrados → 4,5m', () => {
    expect(squaresToMeters(3)).toBe(4.5)
  })

  it('squares negativo lança', () => {
    expect(() => squaresToMeters(-1)).toThrow(/squares must be ≥ 0/)
  })
})

describe('coneSquareCount — cobertura triangular', () => {
  it('4,5m (3 quadrados) → 6 (1+2+3)', () => {
    expect(coneSquareCount(4.5)).toBe(6)
  })

  it('6m (4 quadrados) → 10 (1+2+3+4)', () => {
    expect(coneSquareCount(6)).toBe(10)
  })

  it('9m (6 quadrados) → 21 (1+2+3+4+5+6)', () => {
    expect(coneSquareCount(9)).toBe(21)
  })
})

describe('cuboSquareCount — N²', () => {
  it('1,5m (1) → 1', () => {
    expect(cuboSquareCount(1.5)).toBe(1)
  })

  it('3m (2) → 4', () => {
    expect(cuboSquareCount(3)).toBe(4)
  })

  it('4,5m (3) → 9', () => {
    expect(cuboSquareCount(4.5)).toBe(9)
  })
})

describe('esferaSquareCount — π × N²', () => {
  it('raio 1,5m (1) → floor(π) = 3', () => {
    expect(esferaSquareCount(1.5)).toBe(3)
  })

  it('raio 3m (2) → floor(4π) = 12', () => {
    expect(esferaSquareCount(3)).toBe(12)
  })

  it('raio 6m (4) → floor(16π) = 50', () => {
    expect(esferaSquareCount(6)).toBe(50)
  })
})

describe('linhaSquareCount', () => {
  it('padrão 15m × 1,5m → 10 quadrados', () => {
    expect(linhaSquareCount()).toBe(10)
  })

  it('15m × 3m → 20 quadrados', () => {
    expect(linhaSquareCount(15, 3)).toBe(20)
  })

  it('largura < 1 quadrado ainda vale 1 (mínimo)', () => {
    expect(linhaSquareCount(15, 0)).toBe(10)
  })
})

describe('isDefaultConeLength / isDefaultCuboEdge / isDefaultEsferaRadius', () => {
  it('cone: 4,5m / 6m / 9m padrão', () => {
    expect(isDefaultConeLength(4.5)).toBe(true)
    expect(isDefaultConeLength(6)).toBe(true)
    expect(isDefaultConeLength(9)).toBe(true)
    expect(isDefaultConeLength(3)).toBe(false)
  })

  it('cubo: 1,5m / 3m padrão', () => {
    expect(isDefaultCuboEdge(1.5)).toBe(true)
    expect(isDefaultCuboEdge(3)).toBe(true)
    expect(isDefaultCuboEdge(4.5)).toBe(false)
  })

  it('esfera: 1,5m / 3m / 6m padrão', () => {
    expect(isDefaultEsferaRadius(1.5)).toBe(true)
    expect(isDefaultEsferaRadius(3)).toBe(true)
    expect(isDefaultEsferaRadius(6)).toBe(true)
    expect(isDefaultEsferaRadius(9)).toBe(false)
  })
})
