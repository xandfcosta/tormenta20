import type { AttributeKey } from './attributes'

/**
 * T20 doesn't dictate exact attribute spreads per class, but each class has
 * obvious primary/secondary leanings derived from its class skills, attacks,
 * and magic key attribute (PDF Capítulo 1, class entries). This table is a
 * convenience for character creation auto-fill — players are still free to
 * edit any value afterwards.
 *
 * The priority list is ordered from most-needed to least-needed for the class
 * baseline. Combined with `computeClassAttributePreset` it yields a 10-point
 * point-buy spread that follows the PDF cost table (p17):
 *   primary +3  (4 pts), 2 secondaries +2 each (4 pts), 2 tertiaries +1 each
 *   (2 pts), and the lowest-priority attribute left at 0 — total 10 pts.
 */
export const CLASS_ATTRIBUTE_PRIORITY: Record<string, AttributeKey[]> = {
  Arcanista: ['intelligence', 'constitution', 'dexterity', 'charisma', 'wisdom', 'strength'],
  Bárbaro: ['strength', 'constitution', 'dexterity', 'wisdom', 'charisma', 'intelligence'],
  Bardo: ['charisma', 'dexterity', 'constitution', 'intelligence', 'wisdom', 'strength'],
  Bucaneiro: ['charisma', 'dexterity', 'constitution', 'intelligence', 'strength', 'wisdom'],
  Caçador: ['dexterity', 'wisdom', 'constitution', 'strength', 'intelligence', 'charisma'],
  Cavaleiro: ['strength', 'constitution', 'charisma', 'dexterity', 'wisdom', 'intelligence'],
  Clérigo: ['wisdom', 'constitution', 'charisma', 'strength', 'intelligence', 'dexterity'],
  Druida: ['wisdom', 'constitution', 'dexterity', 'strength', 'intelligence', 'charisma'],
  Guerreiro: ['strength', 'constitution', 'dexterity', 'wisdom', 'charisma', 'intelligence'],
  Inventor: ['intelligence', 'dexterity', 'constitution', 'wisdom', 'charisma', 'strength'],
  Ladino: ['dexterity', 'charisma', 'intelligence', 'constitution', 'wisdom', 'strength'],
  Lutador: ['strength', 'constitution', 'dexterity', 'wisdom', 'charisma', 'intelligence'],
  Nobre: ['charisma', 'intelligence', 'dexterity', 'wisdom', 'constitution', 'strength'],
  Paladino: ['strength', 'charisma', 'constitution', 'wisdom', 'intelligence', 'dexterity'],
}

/**
 * Applies the 3/2/2/1/1/0 spread to a priority list. Total cost on the T20
 * point-buy table = 4 + 2 + 2 + 1 + 1 = 10, matching the standard budget.
 * Returns 0 for any attribute not present in the priority list (shouldn't
 * happen for a well-formed entry).
 */
export const CLASS_PRESET_VALUES: readonly number[] = [3, 2, 2, 1, 1, 0]

export function computeClassAttributePreset(
  priority: readonly AttributeKey[],
): Record<AttributeKey, number> {
  const out: Record<AttributeKey, number> = {
    strength: 0,
    dexterity: 0,
    constitution: 0,
    intelligence: 0,
    wisdom: 0,
    charisma: 0,
  }
  priority.forEach((attr, i) => {
    out[attr] = CLASS_PRESET_VALUES[i] ?? 0
  })
  return out
}

/**
 * Auto-fill helper for character creation. T20 multi-classing isn't legal at
 * level 1 (it's gained via the Poder de Multiclasse at later levels), so the
 * first class in the array drives the preset. Returns `null` for unknown
 * class names so the caller can decide whether to leave attributes untouched.
 */
export function attributePresetForClass(
  className: string,
): Record<AttributeKey, number> | null {
  const priority = CLASS_ATTRIBUTE_PRIORITY[className]
  if (!priority) return null
  return computeClassAttributePreset(priority)
}
