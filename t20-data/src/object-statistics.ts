/**
 * Estatísticas de Objetos.
 *
 * PDF Cap 5 Jogando p239 (Tabela 5-4) + Cap 6 Aventuras p265 (Tabela 6-3
 * Portas + inline pilares/paredes) + p268 (inline árvores).
 *
 * Nota-rodapé p239: PV de itens comuns. Divida por 2 para reduzidos,
 * multiplique por 2 para aumentados e por 5 para gigantes.
 *
 * "Quebrando objetos" (p239):
 *  - Ataque contra Defesa do objeto (definida por tamanho).
 *  - Objeto em movimento: +5 na Defesa.
 *  - Objeto segurado por criatura: usar manobra Quebrar.
 *  - Objeto reduzido a 0 PV = destruído. RD reduz o dano normal.
 *  - Livro NÃO cita "dano de área × 2 vs objetos" nem "imunidade a
 *    ataque furtivo" nesta seção.
 */

import type { Size } from './size'

// ─── Types ────────────────────────────────────────────────────────────
export type ObjectCategory =
  | 'general'
  | 'weapon-armor'
  | 'door'
  | 'tree'
  | 'pillar'
  | 'wall'
  | 'tapestry'
  | 'altar'

/** Escala de tamanho para o multiplicador de PV (p239 nota-rodapé). */
export type ObjectSizeScale = 'reduzido' | 'comum' | 'aumentado' | 'gigante'

export type ObjectStat = {
  id: string
  name: string
  category: ObjectCategory
  size?: Size
  defense?: number
  rd: number
  pv: number
  /** CD de Força para arrombar (apenas portas + paredes). */
  cdForceBreak?: number
  /** CD de Atletismo para escalar/subir (paredes, tapeçaria, árvore). */
  cdAtletismo?: number
  bookPage: 239 | 265 | 268
}

// ─── Constantes ──────────────────────────────────────────────────────
/** Multiplicador de PV por escala de tamanho (p239 nota-rodapé). */
export const OBJECT_PV_SIZE_MULTIPLIER: Readonly<
  Record<ObjectSizeScale, number>
> = Object.freeze({
  reduzido: 0.5,
  comum: 1,
  aumentado: 2,
  gigante: 5,
})

/** Bônus de Defesa em objeto em movimento (p239). */
export const MOVING_OBJECT_DEFENSE_BONUS = 5

/** Falha por 5+ ao arrombar porta sofre 1d6 de impacto (p265). */
export const DOOR_FORCE_BREAK_CRITICAL_FAIL_DAMAGE = '1d6' as const

/** Falha por 5+ em arrombar tem margem 5 (p265). */
export const DOOR_FORCE_BREAK_CRITICAL_FAIL_MARGIN = 5

// ─── Catálogo — Tabela 5-4 objetos gerais (p239) ────────────────────
const GENERAL_OBJECTS: readonly ObjectStat[] = Object.freeze([
  { id: 'pergaminho', name: 'Pergaminho', category: 'general', size: 'Minúsculo', defense: 15, rd: 0, pv: 1, bookPage: 239 },
  { id: 'corda', name: 'Corda', category: 'general', size: 'Minúsculo', defense: 15, rd: 0, pv: 2, bookPage: 239 },
  { id: 'corrente', name: 'Corrente', category: 'general', size: 'Minúsculo', defense: 15, rd: 10, pv: 2, bookPage: 239 },
  { id: 'cadeira', name: 'Cadeira', category: 'general', size: 'Pequeno', defense: 12, rd: 5, pv: 5, bookPage: 239 },
  { id: 'barril', name: 'Barril', category: 'general', size: 'Médio', defense: 10, rd: 5, pv: 10, bookPage: 239 },
  { id: 'porta-madeira-p239', name: 'Porta de madeira', category: 'general', size: 'Grande', defense: 8, rd: 5, pv: 20, bookPage: 239 },
  { id: 'porta-pedra-p239', name: 'Porta de pedra', category: 'general', size: 'Grande', defense: 8, rd: 8, pv: 100, bookPage: 239 },
  { id: 'porta-ferro-p239', name: 'Porta de ferro', category: 'general', size: 'Grande', defense: 8, rd: 10, pv: 100, bookPage: 239 },
  { id: 'carroca', name: 'Carroça', category: 'general', size: 'Grande', defense: 8, rd: 5, pv: 50, bookPage: 239 },
  { id: 'casebre', name: 'Casebre', category: 'general', size: 'Enorme', defense: 5, rd: 5, pv: 100, bookPage: 239 },
  { id: 'celeiro', name: 'Celeiro', category: 'general', size: 'Colossal', defense: 0, rd: 5, pv: 200, bookPage: 239 },
])

// ─── Catálogo — Tabela 5-4 armas/armaduras/escudos (p239) ───────────
const WEAPON_ARMOR_OBJECTS: readonly ObjectStat[] = Object.freeze([
  { id: 'arma-leve-madeira', name: 'Arma leve de madeira (machadinha)', category: 'weapon-armor', rd: 5, pv: 2, bookPage: 239 },
  { id: 'arma-uma-mao-madeira', name: 'Arma de uma mão de madeira (clava)', category: 'weapon-armor', rd: 5, pv: 5, bookPage: 239 },
  { id: 'arma-duas-maos-madeira', name: 'Arma de duas mãos de madeira (bordão)', category: 'weapon-armor', rd: 5, pv: 10, bookPage: 239 },
  { id: 'arma-leve-metal', name: 'Arma leve de metal (adaga)', category: 'weapon-armor', rd: 10, pv: 2, bookPage: 239 },
  { id: 'arma-uma-mao-metal', name: 'Arma de uma mão de metal (espada longa)', category: 'weapon-armor', rd: 10, pv: 5, bookPage: 239 },
  { id: 'arma-duas-maos-metal', name: 'Arma de duas mãos de metal (montante)', category: 'weapon-armor', rd: 10, pv: 10, bookPage: 239 },
  { id: 'escudo-leve', name: 'Escudo leve', category: 'weapon-armor', rd: 5, pv: 10, bookPage: 239 },
  { id: 'escudo-pesado', name: 'Escudo pesado', category: 'weapon-armor', rd: 10, pv: 20, bookPage: 239 },
  { id: 'armadura-leve', name: 'Armadura leve', category: 'weapon-armor', rd: 5, pv: 20, bookPage: 239 },
  { id: 'armadura-pesada', name: 'Armadura pesada', category: 'weapon-armor', rd: 10, pv: 40, bookPage: 239 },
])

// ─── Catálogo — Tabela 6-3 Portas (p265) ────────────────────────────
const DOORS: readonly ObjectStat[] = Object.freeze([
  { id: 'porta-madeira', name: 'Porta de madeira', category: 'door', rd: 5, pv: 20, cdForceBreak: 15, bookPage: 265 },
  { id: 'porta-madeira-reforcada', name: 'Porta de madeira reforçada', category: 'door', rd: 5, pv: 30, cdForceBreak: 20, bookPage: 265 },
  { id: 'porta-pedra', name: 'Porta de pedra', category: 'door', rd: 8, pv: 100, cdForceBreak: 25, bookPage: 265 },
  { id: 'porta-ferro', name: 'Porta de ferro', category: 'door', rd: 10, pv: 100, cdForceBreak: 25, bookPage: 265 },
  { id: 'porta-grade', name: 'Grade', category: 'door', rd: 10, pv: 60, cdForceBreak: 20, bookPage: 265 },
])

// ─── Catálogo — árvores/pilares/paredes/tapeçaria/altar ─────────────
const TERRAIN_OBJECTS: readonly ObjectStat[] = Object.freeze([
  // Árvores (p268)
  { id: 'arvore-estreita', name: 'Árvore estreita (< 1,5m)', category: 'tree', rd: 5, pv: 100, cdAtletismo: 15, bookPage: 268 },
  { id: 'arvore-larga', name: 'Árvore larga (≥ 1,5m)', category: 'tree', rd: 5, pv: 500, cdAtletismo: 15, bookPage: 268 },
  // Pilares (p265)
  { id: 'pilar-estreito', name: 'Pilar estreito (< 1,5m)', category: 'pillar', rd: 8, pv: 100, bookPage: 265 },
  { id: 'pilar-largo', name: 'Pilar largo (≥ 1,5m)', category: 'pillar', rd: 8, pv: 500, bookPage: 265 },
  // Paredes (p265)
  { id: 'parede-alvenaria', name: 'Parede de alvenaria (1,5m)', category: 'wall', rd: 8, pv: 200, cdAtletismo: 20, bookPage: 265 },
  { id: 'parede-pedra-bruta', name: 'Parede de pedra bruta (1,5m)', category: 'wall', rd: 8, pv: 500, cdAtletismo: 15, bookPage: 265 },
  { id: 'parede-madeira', name: 'Parede de madeira (1,5m)', category: 'wall', rd: 5, pv: 100, cdAtletismo: 20, bookPage: 265 },
  // Tapeçaria (p265)
  { id: 'tapecaria', name: 'Tapeçaria (1,5m largura)', category: 'tapestry', rd: 0, pv: 10, cdAtletismo: 15, bookPage: 265 },
  // Altar (p265)
  { id: 'altar-tipico', name: 'Altar típico (1,5m × 3m)', category: 'altar', rd: 8, pv: 200, bookPage: 265 },
])

/** Catálogo completo de objetos (frozen). */
export const OBJECT_STATS: readonly ObjectStat[] = Object.freeze([
  ...GENERAL_OBJECTS,
  ...WEAPON_ARMOR_OBJECTS,
  ...DOORS,
  ...TERRAIN_OBJECTS,
])

const statsById = new Map<string, ObjectStat>(
  OBJECT_STATS.map((o) => [o.id, o]),
)

// ─── Resolvers ──────────────────────────────────────────────────────
/** Busca por ID. Lança se não encontrar. */
export function objectStatById(id: string): ObjectStat {
  const stat = statsById.get(id)
  if (!stat) {
    throw new Error(`objectStatById: unknown id '${id}'`)
  }
  return stat
}

/** Filtra por categoria. */
export function objectsByCategory(
  category: ObjectCategory,
): readonly ObjectStat[] {
  return OBJECT_STATS.filter((o) => o.category === category)
}

// ─── Helpers — escala de PV por tamanho ────────────────────────────
/**
 * Ajusta PV do objeto pela escala (p239 nota-rodapé):
 * reduzido 0,5× / comum 1× / aumentado 2× / gigante 5×.
 */
export function scaledObjectPv(
  basePv: number,
  scale: ObjectSizeScale,
): number {
  if (basePv < 0) {
    throw new Error(`scaledObjectPv: basePv must be ≥ 0, got ${basePv}`)
  }
  return Math.floor(basePv * OBJECT_PV_SIZE_MULTIPLIER[scale])
}

// ─── Helpers — Defesa em movimento (p239) ───────────────────────────
/** Bônus +5 na Defesa se o objeto está em movimento (p239). */
export function movingObjectDefenseBonus(inMotion: boolean): number {
  return inMotion ? MOVING_OBJECT_DEFENSE_BONUS : 0
}

// ─── Helpers — arrombar portas (p265) ───────────────────────────────
export type DoorBreakOutcome = 'broken' | 'failed' | 'failed-and-hurt'

/**
 * Resolve tentativa de arrombar porta (p265):
 *  - roll ≥ CD → broken.
 *  - falha por < 5 → failed.
 *  - falha por ≥ 5 → failed-and-hurt (sofre 1d6 de impacto).
 */
export function doorForceBreakOutcome(
  forceRoll: number,
  cd: number,
): DoorBreakOutcome {
  const delta = forceRoll - cd
  if (delta >= 0) return 'broken'
  if (Math.abs(delta) >= DOOR_FORCE_BREAK_CRITICAL_FAIL_MARGIN) {
    return 'failed-and-hurt'
  }
  return 'failed'
}
