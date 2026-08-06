import type { PowerKind } from '../general-powers'

/**
 * Class power-slot tables — pure level math, NO power data. Split out of
 * `./index` (which also holds the heavy CLASS_POWERS_CATALOG) so the frontend
 * can value-import `slotsForClassLevel`/`CLASS_POWER_SLOTS` without anchoring
 * the class-power catalog into its bundle (see project_front_decouple_catalog
 * B.3 — same item-classify split as `items/catalog/item-classify.ts`).
 */
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
