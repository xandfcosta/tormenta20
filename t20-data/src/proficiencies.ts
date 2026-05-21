/**
 * T20 weapon/armor proficiencies per class.
 * Source: Livro de Regras Tormenta 20, p36–83 (one "Proficiências" line per class).
 *
 * Baseline applied at runtime:
 *  - All characters are proficient with armas simples (p142).
 *  - Proficiency with armaduras pesadas implies armaduras leves (T20 stacking convention).
 */

export const PROFICIENCY_CATEGORIES = [
  'armas-simples',
  'armas-marciais',
  'armas-exoticas',
  'armas-de-fogo',
  'armaduras-leves',
  'armaduras-pesadas',
  'escudos',
] as const
export type ProficiencyCategory = (typeof PROFICIENCY_CATEGORIES)[number]

export const PROFICIENCY_LABELS: Record<ProficiencyCategory, string> = {
  'armas-simples': 'Armas simples',
  'armas-marciais': 'Armas marciais',
  'armas-exoticas': 'Armas exóticas',
  'armas-de-fogo': 'Armas de fogo',
  'armaduras-leves': 'Armaduras leves',
  'armaduras-pesadas': 'Armaduras pesadas',
  escudos: 'Escudos',
}

export const CLASS_PROFICIENCIES: Record<string, readonly ProficiencyCategory[]> = {
  Arcanista: [],
  Bárbaro: ['armas-marciais', 'escudos'],
  Bardo: ['armas-marciais'],
  Bucaneiro: ['armas-marciais'],
  Caçador: ['armas-marciais', 'escudos'],
  Cavaleiro: ['armas-marciais', 'armaduras-pesadas', 'escudos'],
  Clérigo: ['armaduras-pesadas', 'escudos'],
  Druida: ['escudos'],
  Guerreiro: ['armas-marciais', 'armaduras-pesadas', 'escudos'],
  Inventor: [],
  Ladino: [],
  Lutador: [],
  Nobre: ['armas-marciais', 'armaduras-pesadas', 'escudos'],
  Paladino: ['armas-marciais', 'armaduras-pesadas', 'escudos'],
}

export type ProficiencyEntry = {
  category: ProficiencyCategory
  label: string
  granted: boolean
  sources: string[]
}

/**
 * Resolve proficiencies for a character based on their classes. Returns a
 * stable-ordered list covering every category, with `sources` listing which
 * classes grant each one.
 */
export function characterProficiencies(
  classNames: readonly string[],
): ProficiencyEntry[] {
  const sourcesByCategory = new Map<ProficiencyCategory, Set<string>>()
  const add = (cat: ProficiencyCategory, source: string) => {
    let set = sourcesByCategory.get(cat)
    if (!set) {
      set = new Set()
      sourcesByCategory.set(cat, set)
    }
    set.add(source)
  }

  add('armas-simples', 'Todas as classes')

  for (const cls of classNames) {
    const list = CLASS_PROFICIENCIES[cls]
    if (!list) continue
    for (const cat of list) add(cat, cls)
    if (list.includes('armaduras-pesadas')) {
      add('armaduras-leves', cls)
    }
  }

  return PROFICIENCY_CATEGORIES.map((category) => {
    const sources = sourcesByCategory.get(category)
    return {
      category,
      label: PROFICIENCY_LABELS[category],
      granted: !!sources,
      sources: sources ? [...sources] : [],
    }
  })
}
