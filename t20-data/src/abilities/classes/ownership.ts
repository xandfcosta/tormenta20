import type { Modifier } from '../../items/types'
import type { ClassPower } from '../types'

/**
 * Class-power ownership rules — pure logic over a catalog PASSED IN, NO power
 * data. Split out of `./index` so the frontend can run the exact same rules
 * against its fetched-and-cached catalog (via `@/shared/lib/abilities-cache`)
 * without build-time importing CLASS_POWERS_CATALOG (see
 * project_front_decouple_catalog B.3). `./index` re-wraps these with the real
 * CLASS_POWERS_CATALOG for the backend/engine.
 */

/** One class's picks in Character.classChoices (per-class JSON blob). */
export type ClassChoiceSelections = { devoto?: string; caminho?: string }

/**
 * Ownership rule shared by modifiers + UI lists: auto powers by class level,
 * electives by picked id, and `grantedByChoice` rows (Caminho do Arcanista)
 * by the matching classChoices value — those cost no slot (p36).
 */
function ownsClassPower(
  power: ClassPower,
  classLevel: number,
  chosenIds: ReadonlySet<string>,
  choices?: ClassChoiceSelections,
): boolean {
  if (power.grantedAtLevel !== undefined && power.grantedAtLevel <= classLevel)
    return true
  if (chosenIds.has(power.id)) return true
  return (
    power.grantedByChoice !== undefined &&
    choices?.[power.grantedByChoice.field] === power.grantedByChoice.value
  )
}

/**
 * Union of modifiers from class powers the character owns, drawn from the
 * given `catalog`. Auto-granted powers (`grantedAtLevel <= classLevel`) are
 * always folded in; elective powers must be present in `chosenIds`; `choices`
 * resolves grantedByChoice rows (Caminho → +atributo-chave no PM).
 */
export function classPowerModifiersIn(
  catalog: readonly ClassPower[],
  className: string,
  classLevel: number,
  chosenIds: ReadonlySet<string>,
  choices?: ClassChoiceSelections,
): Modifier[] {
  const out: Modifier[] = []
  for (const power of catalog) {
    if (power.className !== className) continue
    if (!power.modifiers) continue
    if (!ownsClassPower(power, classLevel, chosenIds, choices)) continue
    out.push(...power.modifiers)
  }
  return out
}

/**
 * List of class powers the character "owns" for a given class + level +
 * chosen ids (+ classChoices picks), drawn from the given `catalog`. Used by
 * the UI to render the auto-granted + elective lists.
 */
export function ownedClassPowersIn(
  catalog: readonly ClassPower[],
  className: string,
  classLevel: number,
  chosenIds: ReadonlySet<string>,
  choices?: ClassChoiceSelections,
): ClassPower[] {
  return catalog.filter(
    (power) =>
      power.className === className &&
      ownsClassPower(power, classLevel, chosenIds, choices),
  )
}
