/**
 * Spell PM cost composition — junta base cost (círculo + aprimoramentos)
 * com todas as reduções cumulativas: Foco em Magia + Mestre em Escola.
 *
 * PDF refs:
 *  - Base cost (p171): `spellPmCost(circle, augment)` — já em `spells.ts`
 *  - Foco em Magia (p131 Cap 2): -1 PM por magia escolhida; cumulativo
 *    com outras reduções; pode ser tomado várias vezes p/ magias
 *    diferentes.
 *  - Mestre em Escola (p38 Cap 2): -1 PM em magias da escola escolhida;
 *    empilha com Foco.
 *
 * Convenções:
 *  - Reduções podem levar PM total até 0 (magia "gratuita" — comum p/
 *    truques com múltiplos focos, ou magia de 1º com Foco + Mestre).
 *  - PM total nunca negativo.
 */
import { spellPmCost, type SpellCircle, type SpellSchool } from './spells'
import {
  spellPmCostWithSpecialization,
  type SchoolSpecialization,
} from './school-specialization'

/**
 * Registro de Foco em Magia. `picks` = quantas vezes o poder foi tomado
 * para essa magia específica (cumulativo -1 PM cada).
 */
export type FocoEmMagia = {
  spellId: string
  picks: number
}

/**
 * Composição total do custo em PM de uma magia. Todos os campos além
 * de `circle` são opcionais — caso ausentes, retorna base cost puro.
 */
export type SpellPmComposition = {
  circle: SpellCircle
  augmentPm?: number
  spellId: string
  spellSchool: SpellSchool
  focos?: readonly FocoEmMagia[]
  specializations?: readonly SchoolSpecialization[]
}

/** Soma total de picks de Foco em Magia para uma magia específica. */
export function focoTotalPicks(
  spellId: string,
  focos: readonly FocoEmMagia[],
): number {
  return focos
    .filter((f) => f.spellId === spellId)
    .reduce((n, f) => n + f.picks, 0)
}

/**
 * Custo total após aplicar base + aprimoramentos - Foco - Mestre.
 * Floor em 0.
 *
 * Ordem: base → Foco → Mestre. Ordem não importa numericamente pois
 * ambos são subtrações lineares floored at 0, mas Foco primeiro faz
 * intent claro no debug.
 */
export function totalSpellPmCost(c: SpellPmComposition): number {
  if (c.spellId.length === 0) {
    throw new Error('totalSpellPmCost: spellId cannot be empty')
  }
  const base = spellPmCost(c.circle, c.augmentPm ?? 0)
  const focoReduction = focoTotalPicks(c.spellId, c.focos ?? [])
  const afterFoco = Math.max(0, base - focoReduction)
  return spellPmCostWithSpecialization(
    afterFoco,
    c.spellSchool,
    c.specializations ?? [],
  )
}
