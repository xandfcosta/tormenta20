import { CLASS_PROFICIENCIES } from './proficiencies'
import type { ArcanistaPath } from './class-spellcasting'

/**
 * L1 starting equipment — PDF book p140 ("Equipamento Inicial").
 *
 * Major correction vs the original ask: Tormenta 20 uses a SINGLE unified
 * L1 kit (book p140), NOT a per-class equipment list. Class entries in
 * Cap 2 (p36-83) do not contain an "Equipamento Inicial" section.
 *
 * The unified kit is parameterised by the character's proficiências
 * (which come from the class) plus a small set of class-specific extras:
 *  - Inventor `Protótipo` (p68): 1 item superior OR 10 alquímicos ≤ T$ 500
 *  - Nobre `Espólio` (p79): 1 item à escolha ≤ T$ 2.000
 *  - Arcanista Bruxo: foco arcano (implícito por Caminho, p37)
 *  - Arcanista Mago: grimório (implícito por Caminho, p37)
 *
 * Tibares iniciais: T$ 4d6 para TODAS as classes (Tabela 3-1, p140).
 */

export const STARTING_KIT_BASE_ITEMS: readonly string[] = [
  'Mochila',
  'Saco de dormir',
  'Traje de viajante',
]

export const STARTING_TIBARES_DICE = '4d6' as const

/** Página canônica de onde o kit unificado vem (book p140). */
export const STARTING_KIT_BOOK_PAGE = 140

export type StartingArmorChoice =
  /** L1 Arcanistas começam sem armadura (exceção explícita p140). */
  | 'nenhuma'
  /** Player picks among the three light options. */
  | 'leve-a-escolha'
  /** Proficiência em pesadas substitui leve por brunea. */
  | 'brunea'

export type StartingWeaponGrant = 'simples' | 'simples+marcial'

export type StartingExtra = {
  /** Classe que concede o item extra. */
  source: string
  description: string
  /** Teto de preço do item, em Tibares. `null` = item sem teto explícito. */
  maxValueTibar: number | null
}

export type StartingKit = {
  baseItems: readonly string[]
  weapons: StartingWeaponGrant
  armor: StartingArmorChoice
  /** Granted only when the class is proficient in escudos. */
  shieldLeve: boolean
  /** Always `'4d6'` per Tabela 3-1. */
  tibarDice: string
  extras: readonly StartingExtra[]
}

const CLASS_EXTRAS: Readonly<Record<string, readonly StartingExtra[]>> = {
  Inventor: [
    {
      source: 'Inventor — Protótipo (p68)',
      description: '1 item superior OU 10 itens alquímicos, preço total ≤ T$ 500.',
      maxValueTibar: 500,
    },
  ],
  Nobre: [
    {
      source: 'Nobre — Espólio (p79)',
      description: '1 item à escolha, preço ≤ T$ 2.000.',
      maxValueTibar: 2000,
    },
  ],
}

/**
 * Compute the L1 starting kit for a class. The result reflects the
 * unified p140 kit narrowed by the class's proficiências.
 */
export function startingKitFor(className: string): StartingKit {
  const profs = CLASS_PROFICIENCIES[className] ?? []
  const isArcanista = className === 'Arcanista'
  const hasHeavyArmor = profs.includes('armaduras-pesadas')
  const hasMartial = profs.includes('armas-marciais')
  const hasShield = profs.includes('escudos')

  let armor: StartingArmorChoice
  if (isArcanista) {
    armor = 'nenhuma'
  } else if (hasHeavyArmor) {
    armor = 'brunea'
  } else {
    armor = 'leve-a-escolha'
  }

  return {
    baseItems: STARTING_KIT_BASE_ITEMS,
    weapons: hasMartial ? 'simples+marcial' : 'simples',
    armor,
    shieldLeve: hasShield && !isArcanista,
    tibarDice: STARTING_TIBARES_DICE,
    extras: CLASS_EXTRAS[className] ?? [],
  }
}

/**
 * Caminho do Arcanista implicit starting focus (book p37).
 * Feiticeiro doesn't need a foco — returns null.
 */
export function arcanistaPathStartingItem(path: ArcanistaPath): string | null {
  switch (path) {
    case 'bruxo':
      return 'Foco arcano'
    case 'mago':
      return 'Grimório'
    case 'feiticeiro':
      return null
  }
}
