/**
 * Display-only mechanical facts — surfaced to the player as reference text but
 * NOT folded into any computed number. This app is a quick reference for
 * tabletop play, so effects like damage reduction, immunities, senses, and
 * action economy are worth *showing* even when the engine can't compute them.
 *
 * Categorized so the UI can group them into chips (all DR together, all
 * immunities together, …) the way `conditions` group by tag. Kept separate
 * from `Modifier` (which stays strictly numeric) and shared across spells,
 * items, and abilities so all three stop inventing per-source note hacks
 * (e.g. the `amount: 0` fake-modifier in materials).
 */
export type FactCategory =
  | 'dr'
  | 'immunity'
  | 'sense'
  | 'movement'
  | 'action'
  | 'other'

export type DisplayFact = {
  category: FactCategory
  text: string
}

/** Human labels for grouping headers / chip tints. */
export const FACT_CATEGORY_LABEL: Record<FactCategory, string> = {
  dr: 'Redução de dano',
  immunity: 'Imunidade',
  sense: 'Sentido',
  movement: 'Movimento',
  action: 'Ação',
  other: 'Outro',
}
