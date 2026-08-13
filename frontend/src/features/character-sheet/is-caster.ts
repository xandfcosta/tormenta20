import { SPELLCASTER_CLASSES } from '@tormenta20/t20-data'
import type { Character } from '@/shared/api/api'

/**
 * True when any of the character's classes casts by the book.
 *
 * A three-line predicate that lived inside the React `combat-magic-stats.tsx`
 * (575 lines of stat tiles), so the block registry — which only wants to know
 * whether to dim the Magias tab — had to import that whole module.
 *
 * Note this is about CLASSES only: a non-caster can still own a granted spell
 * (Bárbaro with Totem Espiritual), so callers deciding whether there is
 * anything castable check `grantedSpells` too.
 *
 * @example isCasterCharacter({ classes: [{ className: 'Arcanista', level: 3 }] }) // true
 */
export function isCasterCharacter(character: Character): boolean {
  return character.classes.some((c) =>
    (SPELLCASTER_CLASSES as readonly string[]).includes(c.className),
  )
}
