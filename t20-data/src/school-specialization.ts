/**
 * School specialization resolution — aplica efeitos numéricos dos
 * poderes Especialista/Mestre em Escola (Arcanista, Cap 2 p38).
 *
 * Poderes já catalogados em `abilities/classes/arcanista.ts` (só
 * nome + descrição). Aqui: mechanical helpers pra CD + PM discount.
 *
 * Regras:
 *  - **Especialista em Escola**: escolha 1 escola. CD para resistir
 *    às suas magias dessa escola +2. Prereq: Bruxo ou Mago.
 *    Pode ser tomado múltiplas vezes p/ escolas diferentes.
 *  - **Mestre em Escola**: escolha 1 escola. Custo em PM das magias
 *    dessa escola -1. Prereq: Especialista mesma escola + nível 8.
 *    Empilha com outras reduções (Foco em Magia etc).
 */
import type { SpellSchool } from './spells'

export type SchoolSpecializationKind = 'especialista' | 'mestre'

export type SchoolSpecialization = {
  school: SpellSchool
  kind: SchoolSpecializationKind
}

/** Bonus +2 CD por Especialista da escola casando com magia. */
export const ESPECIALISTA_CD_BONUS = 2

/** Redução -1 PM por Mestre da escola casando com magia. */
export const MESTRE_PM_DISCOUNT = 1

/**
 * Ajusta CD do teste de resistência aplicando Especialista em Escola.
 * Cada Especialista da escola da magia adiciona +2 CD.
 *
 * NOTA: PDF não diz explicitamente que múltiplos Especialistas na
 * mesma escola empilham (dado que cada pick é uma escola diferente
 * já é vetado pelo prereq "escola escolhida"). Duplicates são
 * ignorados via Set.
 */
export function spellSaveCdWithSpecialization(
  baseCd: number,
  spellSchool: SpellSchool,
  specializations: readonly SchoolSpecialization[],
): number {
  const uniqueEspecialistas = new Set(
    specializations
      .filter(
        (s) => s.kind === 'especialista' && s.school === spellSchool,
      )
      .map((s) => s.school),
  )
  return baseCd + uniqueEspecialistas.size * ESPECIALISTA_CD_BONUS
}

/**
 * Ajusta PM total de uma magia aplicando Mestre em Escola.
 * Cada Mestre da escola aplica -1 PM (mínimo 0).
 *
 * Empilha com outras reduções (Foco em Magia) — caller passa PM já
 * decrementado ou compõe antes.
 */
export function spellPmCostWithSpecialization(
  basePm: number,
  spellSchool: SpellSchool,
  specializations: readonly SchoolSpecialization[],
): number {
  if (basePm < 0) {
    throw new Error(
      `spellPmCostWithSpecialization: basePm must be ≥ 0, got ${basePm}`,
    )
  }
  const uniqueMestres = new Set(
    specializations
      .filter((s) => s.kind === 'mestre' && s.school === spellSchool)
      .map((s) => s.school),
  )
  const reduced = basePm - uniqueMestres.size * MESTRE_PM_DISCOUNT
  return reduced > 0 ? reduced : 0
}

/**
 * Sanity: valida prereq (Mestre requer Especialista da mesma escola).
 * Retorna array de escolas onde há Mestre sem Especialista.
 */
export function invalidMestreWithoutEspecialista(
  specializations: readonly SchoolSpecialization[],
): readonly SpellSchool[] {
  const especialistas = new Set(
    specializations
      .filter((s) => s.kind === 'especialista')
      .map((s) => s.school),
  )
  const mestres = specializations
    .filter((s) => s.kind === 'mestre')
    .map((s) => s.school)
  return mestres.filter((s) => !especialistas.has(s))
}
