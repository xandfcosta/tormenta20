import { allCatalogItems } from '@/shared/lib/catalog-cache'

/** pt-BR labels for the English catalog category ids. */
const CATEGORY_LABEL: Record<string, string> = {
  animal: 'Animal',
  apparel: 'Vestuário',
  'armor-heavy': 'Armadura pesada',
  'armor-light': 'Armadura leve',
  catalyst: 'Catalisador',
  consumable: 'Consumível',
  improvement: 'Melhoria',
  material: 'Material',
  meal: 'Alimentação',
  shield: 'Escudo',
  vehicle: 'Veículo',
  'weapon-exotic': 'Arma exótica',
  'weapon-firearm': 'Arma de fogo',
  'weapon-martial': 'Arma marcial',
  'weapon-simple': 'Arma simples',
}

/**
 * Category id → the word the player reads. An unmapped id falls through as
 * itself rather than disappearing, so a new catalog category is visible (and
 * ugly) instead of silently blank.
 *
 * @example categoryLabel('weapon-martial') // "Arma marcial"
 */
export function categoryLabel(id: string): string {
  return CATEGORY_LABEL[id] ?? id
}

/**
 * Distinct catalog categories, sorted, for the add-dialog filter. A function
 * rather than a module const: the catalog is fetched and primed at runtime
 * (project_front_decouple_catalog), so a const would evaluate empty at import.
 *
 * @example catalogCategories().includes('shield') // true
 */
export function catalogCategories(): string[] {
  return [...new Set(allCatalogItems().map((c) => c.category))].sort()
}
