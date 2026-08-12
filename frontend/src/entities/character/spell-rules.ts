import {
  CLASS_SPELLCASTING_ATTRIBUTE,
  SPELLCASTER_CLASSES,
  type AttributeKey,
  type SpellCircle,
  type SpellcasterClass,
  highestCircleAtLevel,
} from '@tormenta20/t20-data'
import type { Character } from '@/shared/api/api'

/**
 * The character's spellcasting classes that appear on THIS spell's list — the
 * intersection that decides both its CD and whether the circle is reachable.
 *
 * @example castableClassesFor(character, spell.classes)
 */
export function castableClassesFor(
  character: Character,
  spellClasses: readonly string[],
): SpellcasterClass[] {
  return character.classes
    .map((entry) => entry.className)
    .filter((name): name is SpellcasterClass =>
      (SPELLCASTER_CLASSES as readonly string[]).includes(name),
    )
    .filter((name) => spellClasses.includes(name))
}

/**
 * Best save CD among the classes able to cast this spell. Each class casts with
 * its own key attribute (p171), so a multiclass caster gets the better of them;
 * the per-attribute map comes from the computed sheet, with race and item
 * bonuses already in.
 *
 * @example bestSpellCd(['Arcanista'], sheet.spellCdByAttribute) // 18
 */
export function bestSpellCd(
  applicableClasses: readonly SpellcasterClass[],
  spellCdByAttribute: Record<AttributeKey, number>,
): number | null {
  const cds = applicableClasses
    .map((className) => CLASS_SPELLCASTING_ATTRIBUTE[className])
    .filter((attribute): attribute is AttributeKey => Boolean(attribute))
    .map((attribute) => spellCdByAttribute[attribute])
  return cds.length > 0 ? Math.max(...cds) : null
}

/**
 * Highest spell circle the character can reach through these classes. Driven by
 * the level IN the casting class — a Guerreiro 9 / Arcanista 1 still casts as a
 * 1st-level Arcanista.
 *
 * @example highestCastableCircle(character, ['Arcanista']) // 3
 */
export function highestCastableCircle(
  character: Character,
  applicableClasses: readonly SpellcasterClass[],
): SpellCircle {
  let best: SpellCircle = 0
  for (const className of applicableClasses) {
    const entry = character.classes.find((c) => c.className === className)
    if (!entry) continue
    const circle = highestCircleAtLevel(className, entry.level)
    if (circle > best) best = circle
  }
  return best
}
