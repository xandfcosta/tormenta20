/**
 * Perícia Ofício (INT, treinada) — regras estruturais complementares.
 *
 * PDF Cap 2 Perícias — Ofício (p121-122). Fabricar economics (CDs,
 * matéria-prima, tempos) já em `crafting-rules.ts` e verificado
 * verbatim contra o PDF.
 *
 * Este módulo cobre GAPS não encodados:
 *  - Instrumentos de ofício exigidos (-5 sem, análogo à maleta de Cura).
 *  - Uso "Sustento" (CD 15, 1 semana): renda por Ofício.
 *  - Uso "Identificar" (CD 20): identifica itens raros/exóticos ligados
 *    ao próprio Ofício.
 *  - Regra de retry em Consertar (p121: falha perde tempo+material,
 *    mas pode tentar novamente).
 *  - Categorias oficiais de Ofício (Armeiro, Artesão, Alquimista,
 *    Cozinheiro, Alfaiate) + suporte a categorias customizadas.
 *
 * Cross-ref:
 *  - `crafting-rules.ts:OFICIO_CD_SIMPLE|OFICIO_CD_COMPLEX|MATERIAL_COST_FRACTION|REPAIR_MATERIAL_FRACTION|CRAFT_TIME_DAYS|CONSUMABLE_DOUBLE_PENALTY|materialCost|repairCost`
 */

// ─── Instrumentos ────────────────────────────────────────────────────
/**
 * PDF p122 verbatim: "Cada Ofício exige instrumentos de ofício
 * específicos. Sem eles, você sofre –5 no teste."
 */
export const OFICIO_INSTRUMENT_PENALTY = -5

/** Penalidade agregada por falta de instrumentos. */
export function oficioInstrumentPenalty(hasInstruments: boolean): number {
  return hasInstruments ? 0 : OFICIO_INSTRUMENT_PENALTY
}

// ─── Sustento (p122) ─────────────────────────────────────────────────
/**
 * PDF p122 verbatim: "Com uma semana de trabalho e um teste de Ofício
 * (CD 15), você ganha T$ 1, mais T$ 1 por ponto que seu teste exceder
 * a CD."
 */
export const SUSTENTO_CD = 15

/** Duração do trabalho para render Sustento (verbatim "uma semana"). */
export const SUSTENTO_DURATION_WEEKS = 1

/** Renda base ao passar no teste (verbatim T$ 1). */
export const SUSTENTO_BASE_INCOME_TIBAR = 1

/** Renda por ponto excedente sobre a CD (verbatim T$ 1). */
export const SUSTENTO_INCOME_PER_MARGIN = 1

/**
 * Renda em Tibares por uma semana de Sustento.
 * `checkResult` = total do teste de Ofício.
 * Falha (`checkResult < SUSTENTO_CD`): 0.
 * Sucesso: base + margin × per-point.
 */
export function sustentoIncome(checkResult: number): number {
  if (checkResult < SUSTENTO_CD) return 0
  const margin = checkResult - SUSTENTO_CD
  return SUSTENTO_BASE_INCOME_TIBAR + margin * SUSTENTO_INCOME_PER_MARGIN
}

// ─── Identificar (p122) ──────────────────────────────────────────────
/**
 * PDF p122 verbatim: "Você pode identificar itens raros e exóticos
 * ligados ao seu Ofício." Ação padrão treinada, CD 20.
 */
export const IDENTIFICAR_CD = 20

// ─── Consertar retry (p121) ──────────────────────────────────────────
/**
 * PDF p121 verbatim: "Em caso de falha, o tempo e o dinheiro são
 * perdidos (mas você pode tentar novamente)."
 * Regra explícita apenas para Consertar; Fabricar é omisso na entrada.
 */
export const REPAIR_RETRY_ALLOWED = true
export const REPAIR_FAIL_LOSES_TIME_AND_MATERIAL = true

// ─── Categorias oficiais (p121) ──────────────────────────────────────
/** Categoria oficial listada explicitamente em p121. */
export type OficioCategory =
  | 'armeiro'
  | 'artesao'
  | 'alquimista'
  | 'cozinheiro'
  | 'alfaiate'

export type OficioCategoryInfo = {
  id: OficioCategory
  name: string
  /** Itens fabricáveis (verbatim curto). */
  produces: readonly string[]
}

export const OFICIO_CATEGORIES: readonly OficioCategoryInfo[] = Object.freeze([
  Object.freeze({
    id: 'armeiro',
    name: 'Armeiro',
    produces: Object.freeze(['armas', 'armaduras', 'escudos']),
  }),
  Object.freeze({
    id: 'artesao',
    name: 'Artesão',
    produces: Object.freeze([
      'equipamento de aventura',
      'ferramentas',
      'esotéricos',
      'veículos',
    ]),
  }),
  Object.freeze({
    id: 'alquimista',
    name: 'Alquimista',
    produces: Object.freeze(['itens alquímicos']),
  }),
  Object.freeze({
    id: 'cozinheiro',
    name: 'Cozinheiro',
    produces: Object.freeze(['alimentação']),
  }),
  Object.freeze({
    id: 'alfaiate',
    name: 'Alfaiate',
    produces: Object.freeze(['vestuário']),
  }),
])

const categoryById = new Map<OficioCategory, OficioCategoryInfo>(
  OFICIO_CATEGORIES.map((c) => [c.id, c]),
)

export function oficioCategoryById(
  id: OficioCategory,
): OficioCategoryInfo {
  const info = categoryById.get(id)
  if (!info) {
    throw new Error(`oficioCategoryById: unknown category ${id}`)
  }
  return info
}

/**
 * PDF p121: "Você pode inventar outros tipos de Ofício: carpinteiro,
 * pedreiro, ourives, fazendeiro, pescador..." Categorias custom são
 * permitidas fora da lista oficial.
 */
export const OFICIO_ALLOWS_CUSTOM_CATEGORIES = true
