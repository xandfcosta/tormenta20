/**
 * Spell Áreas de Efeito — geometria padrão e helpers.
 *
 * PDF Cap 4 Magia p225 — quadro/ilustração "ÁREAS DE EFEITOS".
 * Reutiliza `AreaShape` de [[spells]] (bare union sem defaults).
 *
 * Escala: **1 quadrado = 1,5m**.
 *
 * Templates padrão (p225):
 *  - Cone: 4,5m / 6m / 9m (comprimento)
 *  - Cubo: 1,5m / 3m (aresta)
 *  - Esfera: raio 1,5m / 3m / 6m
 *  - Linha: 15m comprimento, 1,5m largura
 *  - Cilindro: definido caso a caso (raio + altura)
 *  - Quadrado: alias 2D para efeitos que ignoram altura
 *
 * Origem (p225):
 *  - Cone/Linha: surge adjacente ao conjurador.
 *  - Cilindro/Esfera/Cubo/Quadrado: surge na intersecção de 4 quadrados.
 *
 * Mirar em área invisível (p224):
 *  - Percepção CD 20.
 *  - Custo adicional de PM conforme a descrição do efeito (variável).
 *
 * Redirecionar área (p225): ação padrão do conjurador (sem custo de PM).
 */

import type { AreaShape } from './spells'

// ─── Types ────────────────────────────────────────────────────────────
/** Origem da área conforme forma (p225). */
export type AreaOrigin =
  | 'adjacent-to-caster'
  | 'four-square-intersection'

// ─── Constantes ──────────────────────────────────────────────────────
/** Escala de conversão: 1 quadrado = 1,5m. */
export const SQUARE_METERS = 1.5

/** Templates padrão de Cone (p225 verbatim). */
export const CONE_DEFAULT_LENGTHS_M: readonly number[] = Object.freeze([
  4.5, 6, 9,
])

/** Templates padrão de Cubo (p225 verbatim). */
export const CUBO_DEFAULT_EDGES_M: readonly number[] = Object.freeze([1.5, 3])

/** Templates padrão de Esfera (raio; p225 verbatim). */
export const ESFERA_DEFAULT_RADII_M: readonly number[] = Object.freeze([
  1.5, 3, 6,
])

/** Comprimento default de Linha (p225). */
export const LINHA_DEFAULT_LENGTH_M = 15

/** Largura default de Linha (p225). */
export const LINHA_DEFAULT_WIDTH_M = 1.5

/** Origem por forma (p225). Frozen. */
export const AREA_ORIGIN_BY_SHAPE: Readonly<Record<AreaShape, AreaOrigin>> =
  Object.freeze({
    cone: 'adjacent-to-caster',
    linha: 'adjacent-to-caster',
    cilindro: 'four-square-intersection',
    esfera: 'four-square-intersection',
    cubo: 'four-square-intersection',
    quadrado: 'four-square-intersection',
  })

/** CD de Percepção para localizar/mirar área invisível (p224). */
export const HIDDEN_AREA_PERCEPTION_CD = 20

/** Redirecionar área é ação padrão do conjurador (p225). */
export const REDIRECT_AREA_ACTION = 'padrao' as const

// ─── Helpers — conversão quadrado ↔ metros ──────────────────────────
/** Converte metros para quadrados 1,5m (arredonda para baixo). */
export function metersToSquares(meters: number): number {
  if (meters < 0) {
    throw new Error(`metersToSquares: meters must be ≥ 0, got ${meters}`)
  }
  return Math.floor(meters / SQUARE_METERS)
}

/** Converte quadrados 1,5m para metros. */
export function squaresToMeters(squares: number): number {
  if (squares < 0) {
    throw new Error(`squaresToMeters: squares must be ≥ 0, got ${squares}`)
  }
  return squares * SQUARE_METERS
}

// ─── Helpers — origem ───────────────────────────────────────────────
/** Origem da área para uma dada forma. */
export function areaOrigin(shape: AreaShape): AreaOrigin {
  return AREA_ORIGIN_BY_SHAPE[shape]
}

// ─── Helpers — quadrados afetados ───────────────────────────────────
/**
 * Quadrados afetados por Cone.
 * Aproximação triangular T20 (p225): cone de comprimento N quadrados
 * cobre `1 + 2 + ... + N` quadrados = N * (N + 1) / 2.
 * Ex.: 3 quadrados (4,5m) = 6 quadrados; 4 (6m) = 10; 6 (9m) = 21.
 */
export function coneSquareCount(coneLengthMeters: number): number {
  const squares = metersToSquares(coneLengthMeters)
  return (squares * (squares + 1)) / 2
}

/**
 * Quadrados afetados por Cubo/Quadrado.
 * Aresta em quadrados = N; total = N²; ex.: 1,5m = 1; 3m = 4; 4,5m = 9.
 */
export function cuboSquareCount(edgeMeters: number): number {
  const squares = metersToSquares(edgeMeters)
  return squares * squares
}

/**
 * Quadrados afetados por Esfera projetada em 2D (aproximação círculo
 * inscrito em N² quadrados). Ex.: raio 1,5m (1 quadrado) = 5 quadrados
 * cruz; raio 3m (2) = ~13; raio 6m (4) = ~49.
 *
 * Fórmula usada: `⌊π × N²⌋` com N = raio em quadrados.
 */
export function esferaSquareCount(radiusMeters: number): number {
  const squares = metersToSquares(radiusMeters)
  return Math.floor(Math.PI * squares * squares)
}

/**
 * Quadrados afetados por Linha padrão (15m × 1,5m largura).
 * Comprimento em quadrados × largura em quadrados.
 */
export function linhaSquareCount(
  lengthMeters: number = LINHA_DEFAULT_LENGTH_M,
  widthMeters: number = LINHA_DEFAULT_WIDTH_M,
): number {
  const lengthSquares = metersToSquares(lengthMeters)
  const widthSquares = Math.max(1, metersToSquares(widthMeters))
  return lengthSquares * widthSquares
}

// ─── Helpers — templates padrão ─────────────────────────────────────
/** True se `meters` é um dos comprimentos padrão de Cone (p225). */
export function isDefaultConeLength(meters: number): boolean {
  return CONE_DEFAULT_LENGTHS_M.includes(meters)
}

/** True se `meters` é uma das arestas padrão de Cubo (p225). */
export function isDefaultCuboEdge(meters: number): boolean {
  return CUBO_DEFAULT_EDGES_M.includes(meters)
}

/** True se `meters` é um dos raios padrão de Esfera (p225). */
export function isDefaultEsferaRadius(meters: number): boolean {
  return ESFERA_DEFAULT_RADII_M.includes(meters)
}
