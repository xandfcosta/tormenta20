import { type PericiaPlan, periciaBudget } from './pericia-helpers'

/**
 * Fold the class's auto-trained perícias into the trained set. Returns the SAME
 * array when there is nothing to add, so a caller can use identity to decide
 * whether to write — seeding on every render would otherwise loop.
 *
 * @example draft.setValue('trainedExpertises', seedFixedExpertises(trained, plan))
 */
export function seedFixedExpertises(trained: string[], plan: PericiaPlan): string[] {
  const missing = plan.fixed.filter((name) => !trained.includes(name))
  return missing.length === 0 ? trained : [...trained, ...missing]
}

/**
 * The line that explains the overflow rule at the moment it bites: picks beyond
 * the class cap are paid out of the FREE budget. Silent in every other case —
 * a player inside their quota has nothing to learn here, and one who has no
 * free budget at all simply hits a wall instead.
 *
 * @example overflowNotice(plan, trained) // '1 escolha da classe está usando…'
 */
export function overflowNotice(plan: PericiaPlan, trained: string[]): string | null {
  if (plan.freeCount === 0) return null
  const budget = periciaBudget(plan, trained)
  if (!budget.classOverflow) return null
  const spilled = plan.classPool.filter((name) => trained.includes(name)).length -
    plan.classCount
  if (spilled <= 0) return null
  return spilled === 1
    ? '1 escolha da classe está usando uma perícia livre.'
    : `${spilled} escolhas da classe estão usando perícias livres.`
}

/** "Falta 1 perícia" / "Faltam 3 perícias" — a contagem some quando zera. */
export function missingNotice(missing: number): string | null {
  if (missing <= 0) return null
  const noun = missing === 1 ? 'Falta 1 perícia' : `Faltam ${missing} perícias`
  return `${noun} — dá para terminar depois na ficha.`
}
