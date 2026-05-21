import { ARCANISTA_POWERS } from './arcanista'
import { BARBARO_POWERS } from './barbaro'
import { BARDO_POWERS } from './bardo'
import { BUCANEIRO_POWERS } from './bucaneiro'
import { CACADOR_POWERS } from './cacador'
import { CAVALEIRO_POWERS } from './cavaleiro'
import { CLERIGO_POWERS } from './clerigo'
import { DRUIDA_POWERS } from './druida'
import { GUERREIRO_POWERS } from './guerreiro'
import { INVENTOR_POWERS } from './inventor'
import { LADINO_POWERS } from './ladino'
import { LUTADOR_POWERS } from './lutador'
import { NOBRE_POWERS } from './nobre'
import { PALADINO_POWERS } from './paladino'
import type { Modifier } from '../../items/types'
import type { PowerKind } from '../general-powers'
import type { ClassPower } from '../types'

export type ClassPowerSlot = {
  /** Class level at which this slot opens. */
  level: number
  /** Pools the player may draw from at this slot. */
  kinds: PowerKind[]
}

/**
 * Slot tracks per class (PDF Cap 1 — Tabela 1-5 a 1-18). Per PDF p33:
 * "Todas as classes possuem uma habilidade 'Poder' (Poder de Arcanista,
 * Poder de Bárbaro...) que permite escolher um poder de uma lista... Você
 * sempre pode substituir um poder de classe por um poder geral (veja no
 * Capítulo 2)."
 *
 * For all 14 classes the slot opens every level from L2 to L20 (19 slots
 * total). Bardo also has a L2 auto power "Eclético" that doesn't consume a
 * slot. `kinds` is the className-slug — general-power pools live separately
 * and aren't filtered here (the catalog is empty until Cap 2 review).
 */
function levelsForAllClasses(): number[] {
  return [2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20]
}

function classSlots(kind: PowerKind): ClassPowerSlot[] {
  return levelsForAllClasses().map((level) => ({ level, kinds: [kind] }))
}

export const CLASS_POWER_SLOTS: Record<string, ClassPowerSlot[]> = {
  Arcanista: classSlots('arcanista'),
  'Bárbaro': classSlots('barbaro'),
  Bardo: classSlots('bardo'),
  Bucaneiro: classSlots('bucaneiro'),
  'Caçador': classSlots('cacador'),
  Cavaleiro: classSlots('cavaleiro'),
  'Clérigo': classSlots('clerigo'),
  Druida: classSlots('druida'),
  Guerreiro: classSlots('guerreiro'),
  Inventor: classSlots('inventor'),
  Ladino: classSlots('ladino'),
  Lutador: classSlots('lutador'),
  Nobre: classSlots('nobre'),
  Paladino: classSlots('paladino'),
}

/**
 * How many power slots the player has *earned* (level ≤ classLevel). Used by
 * the UI to enforce "Poderes: X de N".
 */
export function slotsForClassLevel(
  className: string,
  classLevel: number,
): ClassPowerSlot[] {
  const all = CLASS_POWER_SLOTS[className]
  if (!all) return []
  return all.filter((s) => s.level <= classLevel)
}

/**
 * Union of kinds across all unlocked slots — used to filter the general
 * power list shown in the picker.
 */
export function unlockedKinds(
  className: string,
  classLevel: number,
): PowerKind[] {
  const slots = slotsForClassLevel(className, classLevel)
  const set = new Set<PowerKind>()
  for (const slot of slots) for (const k of slot.kinds) set.add(k)
  return [...set]
}

/**
 * Catalog merge — one array used by the rest of the engine. Per-class files
 * stay focused so each one fits comfortably under the project's 500-line cap.
 */
export const CLASS_POWERS_CATALOG: ClassPower[] = [
  ...ARCANISTA_POWERS,
  ...BARBARO_POWERS,
  ...BARDO_POWERS,
  ...BUCANEIRO_POWERS,
  ...CACADOR_POWERS,
  ...CAVALEIRO_POWERS,
  ...CLERIGO_POWERS,
  ...DRUIDA_POWERS,
  ...GUERREIRO_POWERS,
  ...INVENTOR_POWERS,
  ...LADINO_POWERS,
  ...LUTADOR_POWERS,
  ...NOBRE_POWERS,
  ...PALADINO_POWERS,
]

/**
 * Returns the union of modifiers from class powers the character actually
 * owns. Auto-granted powers (`grantedAtLevel <= classLevel`) are always
 * folded in; elective powers must be present in `chosenIds` to count.
 */
export function classPowerModifiers(
  className: string,
  classLevel: number,
  chosenIds: ReadonlySet<string>,
): Modifier[] {
  const out: Modifier[] = []
  for (const power of CLASS_POWERS_CATALOG) {
    if (power.className !== className) continue
    if (!power.modifiers) continue
    const isAuto =
      power.grantedAtLevel !== undefined &&
      power.grantedAtLevel <= classLevel
    const isChosen = chosenIds.has(power.id)
    if (!isAuto && !isChosen) continue
    out.push(...power.modifiers)
  }
  return out
}

/**
 * Returns the list of class powers the character "owns" for a given class +
 * level + chosen ids. Used by the UI to render the auto-granted + elective
 * lists.
 */
export function ownedClassPowers(
  className: string,
  classLevel: number,
  chosenIds: ReadonlySet<string>,
): ClassPower[] {
  return CLASS_POWERS_CATALOG.filter((power) => {
    if (power.className !== className) return false
    const isAuto =
      power.grantedAtLevel !== undefined &&
      power.grantedAtLevel <= classLevel
    return isAuto || chosenIds.has(power.id)
  })
}
