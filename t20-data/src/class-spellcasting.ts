import type { SpellCircle } from './spells'

/**
 * Spell progression per spellcaster class — PDF book Cap 2 class
 * tables (p37, p44, p57, p61, p81).
 *
 * What's pinned here:
 *  - **Which classes cast** at all (5 total).
 *  - **Level at which each círculo unlocks** (per Tabelas 1-5/7/11/12/18).
 *  - **Max círculo achievable** by the base class (Bardo/Druida cap at 4°;
 *    Paladino caps at 1° via the optional `Orar` poder de classe).
 *  - **Magias conhecidas** progression: starting count + learn cadence.
 *  - **Atributo-chave** for CD save.
 *  - **School restriction** (Bardo/Druida pick 3 escolas at L1, locked).
 *  - **Spell list** (arcana vs divina).
 *
 * What's intentionally NOT here:
 *  - Caminho do Arcanista variants (Bruxo / Feiticeiro / Mago) — they
 *    change the *atributo-chave* and *learn cadence* but share the
 *    círculo-unlock table. Encoded as a separate `ArcanistaPath` enum
 *    that overrides the base entry where it differs.
 *  - Paladino's `Orar` poder details — paladin's progression is a
 *    function of how many times Orar is picked, not class level.
 */

import type { AttributeKey } from './attributes'

export type SpellcasterClass =
  | 'Arcanista'
  | 'Bardo'
  | 'Clérigo'
  | 'Druida'
  | 'Paladino'

export const SPELLCASTER_CLASSES: readonly SpellcasterClass[] = [
  'Arcanista',
  'Bardo',
  'Clérigo',
  'Druida',
  'Paladino',
]

export type SpellList = 'arcana' | 'divina'

/** Cadence at which a class learns new magias once a círculo is unlocked. */
export type LearnCadence =
  | 'every-level'
  | 'every-even-level'
  | 'every-odd-level'
  | 'via-power'

/**
 * Caminho do Arcanista (book p37). All paths share the círculo unlock
 * table; they differ in atributo-chave and learn cadence:
 *  - bruxo     → Int, +1 magia / nível (foco arcano, Misticismo on fail)
 *  - feiticeiro → Car, +1 magia / nível ÍMPAR (linhagem sobrenatural)
 *  - mago      → Int, +1 magia / nível (grimório, memorização diária)
 */
export type ArcanistaPath = 'bruxo' | 'feiticeiro' | 'mago'

export type SpellProgression = {
  className: SpellcasterClass
  list: SpellList
  attribute: AttributeKey | null
  /** Number of escolas locked at L1 (Bardo/Druida pick 3 of 8). */
  schoolRestriction: number | null
  /**
   * Level (1-20) at which each círculo unlocks. `null` means the class
   * never accesses that círculo.
   */
  unlockLevel: Readonly<Record<SpellCircle, number | null>>
  maxCircle: SpellCircle
  /** Magias conhecidas at L1. */
  startingSpellsKnown: number
  learnCadence: LearnCadence
}

export const SPELL_PROGRESSION: Readonly<
  Record<SpellcasterClass, SpellProgression>
> = {
  Arcanista: {
    className: 'Arcanista',
    list: 'arcana',
    attribute: 'intelligence',
    schoolRestriction: null,
    unlockLevel: { 0: 1, 1: 1, 2: 5, 3: 9, 4: 13, 5: 17 },
    maxCircle: 5,
    startingSpellsKnown: 3,
    learnCadence: 'every-level',
  },
  Bardo: {
    className: 'Bardo',
    list: 'arcana',
    attribute: 'charisma',
    schoolRestriction: 3,
    unlockLevel: { 0: 1, 1: 1, 2: 6, 3: 10, 4: 14, 5: null },
    maxCircle: 4,
    startingSpellsKnown: 2,
    learnCadence: 'every-even-level',
  },
  'Clérigo': {
    className: 'Clérigo',
    list: 'divina',
    attribute: 'wisdom',
    schoolRestriction: null,
    unlockLevel: { 0: 1, 1: 1, 2: 5, 3: 9, 4: 13, 5: 17 },
    maxCircle: 5,
    startingSpellsKnown: 3,
    learnCadence: 'every-level',
  },
  Druida: {
    className: 'Druida',
    list: 'divina',
    attribute: 'wisdom',
    schoolRestriction: 3,
    unlockLevel: { 0: 1, 1: 1, 2: 6, 3: 10, 4: 14, 5: null },
    maxCircle: 4,
    startingSpellsKnown: 2,
    learnCadence: 'every-even-level',
  },
  Paladino: {
    className: 'Paladino',
    list: 'divina',
    attribute: 'wisdom',
    schoolRestriction: null,
    unlockLevel: { 0: null, 1: 2, 2: null, 3: null, 4: null, 5: null },
    maxCircle: 1,
    startingSpellsKnown: 0,
    learnCadence: 'via-power',
  },
}

const ARCANISTA_PATH_OVERRIDES: Readonly<
  Record<
    ArcanistaPath,
    Partial<Pick<SpellProgression, 'attribute' | 'learnCadence' | 'startingSpellsKnown'>>
  >
> = {
  bruxo: { attribute: 'intelligence', learnCadence: 'every-level' },
  feiticeiro: { attribute: 'charisma', learnCadence: 'every-odd-level' },
  mago: { attribute: 'intelligence', learnCadence: 'every-level', startingSpellsKnown: 4 },
}

/** Resolve the Arcanista progression for a specific Caminho. */
export function arcanistaProgression(path: ArcanistaPath): SpellProgression {
  const base = SPELL_PROGRESSION.Arcanista
  const overrides = ARCANISTA_PATH_OVERRIDES[path]
  return { ...base, ...overrides }
}

/**
 * Highest círculo the class can currently cast at the given character
 * level. Returns `0` if the class doesn't yet cast (Paladino at L1).
 */
export function highestCircleAtLevel(
  className: SpellcasterClass,
  level: number,
): SpellCircle {
  if (level < 1 || level > 20) {
    throw new Error(`highestCircleAtLevel: level must be 1..20, got ${level}`)
  }
  const prog = SPELL_PROGRESSION[className]
  let best: SpellCircle = 0
  for (const c of [1, 2, 3, 4, 5] as const) {
    const unlock = prog.unlockLevel[c]
    if (unlock !== null && level >= unlock) best = c
  }
  return best
}

export function canCastCircle(
  className: SpellcasterClass,
  level: number,
  circle: SpellCircle,
): boolean {
  return circle <= highestCircleAtLevel(className, level)
}

/**
 * Magias conhecidas at a given level for one of the spontaneous casters
 * (Arcanista bruxo/mago, Bardo, Clérigo, Druida). Does NOT cover
 * Arcanista feiticeiro (learns on odd levels) or Paladino (via Orar
 * poder count) — use the specialized helpers below.
 */
export function totalSpellsKnown(
  className: Exclude<SpellcasterClass, 'Paladino'>,
  level: number,
): number {
  if (level < 1 || level > 20) {
    throw new Error(`totalSpellsKnown: level must be 1..20, got ${level}`)
  }
  const prog = SPELL_PROGRESSION[className]
  const base = prog.startingSpellsKnown
  switch (prog.learnCadence) {
    case 'every-level':
      return base + (level - 1)
    case 'every-even-level':
      return base + Math.floor(level / 2)
    case 'every-odd-level':
      return base + Math.floor((level + 1) / 2) - 1
    case 'via-power':
      return base
  }
}

/** Magias conhecidas for an Arcanista on the feiticeiro path (odd-level cadence). */
export function feiticeiroSpellsKnown(level: number): number {
  if (level < 1 || level > 20) {
    throw new Error(`feiticeiroSpellsKnown: level must be 1..20, got ${level}`)
  }
  const base = SPELL_PROGRESSION.Arcanista.startingSpellsKnown
  return base + Math.floor((level + 1) / 2) - 1
}

/** Paladino's spell count is determined entirely by Orar pick count. */
export function paladinSpellsViaOrar(orarPicks: number): number {
  if (orarPicks < 0) {
    throw new Error(`paladinSpellsViaOrar: orarPicks must be ≥ 0, got ${orarPicks}`)
  }
  return orarPicks
}
