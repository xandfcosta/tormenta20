/**
 * List operations behind the ability pickers. They live outside the components
 * because "which id replaces which" is a rule, not markup — and a rule that
 * silently drops a sibling is a bug you only see two screens later.
 */

/**
 * Pick one option out of a mutually exclusive group (a race ability's
 * variants): every sibling leaves the list, the picked one goes in.
 *
 * @example pickExclusive(['versatil-pericia'], new Set([...]), 'versatil-poder')
 */
export function pickExclusive(
  choices: readonly string[],
  siblingIds: ReadonlySet<string>,
  picked: string,
): string[] {
  return [...choices.filter((id) => !siblingIds.has(id)), picked]
}

/**
 * Toggle one id in a capped selection. Unchecking always works; checking past
 * the cap returns the list UNCHANGED, which is how the caller can tell nothing
 * happened without a second rule.
 *
 * @example toggleWithLimit(['a', 'b'], 'c', 2) // ['a', 'b'] — cheio
 */
export function toggleWithLimit(
  selected: readonly string[],
  id: string,
  limit: number,
): string[] {
  if (selected.includes(id)) return selected.filter((x) => x !== id)
  if (selected.length >= limit) return [...selected]
  return [...selected, id]
}
