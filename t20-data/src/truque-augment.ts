/**
 * Truque aprimoramento — versão 0 PM de uma magia base.
 *
 * PDF Cap 4 p171 (verbatim): "Truque. Este aprimoramento transforma
 * a magia em uma versão mais simples e reduz seu custo em PM para
 * zero. Truques não podem ser usados em conjunto com outros
 * aprimoramentos."
 *
 * NOTA: T20 não tem catálogo separado de truques nem contagem por
 * classe. Truque é um aprimoramento (kind 'muda') que qualquer magia
 * pode receber; a magia base deve declarar a entrada "Truque:" pra
 * ser elegível.
 *
 * Este módulo complementa `spells.ts` (que trata `circle: 0` como
 * abstração pré-existente da codebase — mantida por compat).
 */
import type { CatalogAugment, CatalogSpell } from './spell-catalog'
import type { SpellAugment, SpellCircle } from './spells'

/** Estrutura mínima de qualquer aprimoramento (Catalog ou plain). */
type AugmentLike = Pick<SpellAugment, 'kind' | 'pmCost' | 'description'>

/** Um truque sempre resulta em 0 PM total, ignorando círculo base. */
export const TRUQUE_TOTAL_PM = 0

/** Prefixo canônico da descrição da entrada Truque no catálogo. */
export const TRUQUE_DESCRIPTION_PREFIX = 'Truque:'

/**
 * Identifica se um aprimoramento é a variante "Truque" — kind 'muda',
 * pmCost 0, descrição começa com "Truque:".
 */
export function isTruqueAugment(augment: AugmentLike): boolean {
  return (
    augment.kind === 'muda' &&
    augment.pmCost === 0 &&
    augment.description.startsWith(TRUQUE_DESCRIPTION_PREFIX)
  )
}

/** Retorna a entrada Truque de uma magia, ou undefined se não elegível. */
export function truqueAugmentOf(
  spell: CatalogSpell,
): CatalogAugment | undefined {
  return spell.augments.find(isTruqueAugment)
}

/** True se a magia tem aprimoramento Truque disponível. */
export function hasTruqueAugment(spell: CatalogSpell): boolean {
  return truqueAugmentOf(spell) !== undefined
}

/** Filtra o catálogo por magias com Truque disponível. */
export function spellsWithTruqueAugment(
  catalog: readonly CatalogSpell[],
): readonly CatalogSpell[] {
  return catalog.filter(hasTruqueAugment)
}

/**
 * Custo total em PM ao lançar como Truque — sempre 0, ignora círculo
 * base e aprimoramentos.
 */
export function castAsTruque(_circle: SpellCircle): number {
  return TRUQUE_TOTAL_PM
}

/**
 * Valida seleção de aprimoramentos por regra p171: Truque não pode
 * combinar com outros. Throws se combinação inválida.
 */
export function assertTruqueRestriction(
  augments: readonly AugmentLike[],
): void {
  const truques = augments.filter(isTruqueAugment)
  if (truques.length === 0) return
  if (augments.length > truques.length) {
    throw new Error(
      `assertTruqueRestriction: Truque não pode combinar com outros aprimoramentos (${augments.length - truques.length} outros presentes)`,
    )
  }
  if (truques.length > 1) {
    throw new Error(
      `assertTruqueRestriction: apenas 1 Truque por lançamento, got ${truques.length}`,
    )
  }
}
