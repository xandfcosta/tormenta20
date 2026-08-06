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
import type { ClassPower } from '../types'
import {
  type ClassChoiceSelections,
  classPowerModifiersIn,
  ownedClassPowersIn,
} from './ownership'

// Slot tables + ownership rules are pure logic with no power data — they live
// in ./slots and ./ownership so the frontend can import them without anchoring
// CLASS_POWERS_CATALOG (project_front_decouple_catalog B.3). Re-exported here to
// keep the classes barrel's public surface unchanged for backend/engine.
export {
  CLASS_POWER_SLOTS,
  slotsForClassLevel,
  unlockedKinds,
} from './slots'
export type { ClassPowerSlot } from './slots'
// The catalog-parametrized pure rules are exported too: the frontend calls
// `ownedClassPowersIn` with its fetched-and-cached catalog (B.3).
export { classPowerModifiersIn, ownedClassPowersIn } from './ownership'
export type { ClassChoiceSelections } from './ownership'

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
 * Union of modifiers from the class powers the character owns. Thin wrapper
 * binding the engine's real CLASS_POWERS_CATALOG to the pure rule in
 * `./ownership` (which the frontend calls with its cached catalog instead).
 */
export function classPowerModifiers(
  className: string,
  classLevel: number,
  chosenIds: ReadonlySet<string>,
  choices?: ClassChoiceSelections,
): Modifier[] {
  return classPowerModifiersIn(
    CLASS_POWERS_CATALOG,
    className,
    classLevel,
    chosenIds,
    choices,
  )
}

/**
 * List of class powers the character "owns" for a given class + level + chosen
 * ids (+ classChoices picks). Thin wrapper over the pure rule in `./ownership`.
 */
export function ownedClassPowers(
  className: string,
  classLevel: number,
  chosenIds: ReadonlySet<string>,
  choices?: ClassChoiceSelections,
): ClassPower[] {
  return ownedClassPowersIn(
    CLASS_POWERS_CATALOG,
    className,
    classLevel,
    chosenIds,
    choices,
  )
}
