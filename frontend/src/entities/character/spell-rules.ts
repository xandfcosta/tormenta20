import {
  SPELLCASTER_CLASSES,
  type AttributeKey,
  type SpellCircle,
  type SpellcasterClass,
  highestCircleAtLevel,
  spellcastingAttributeFor,
} from '@tormenta20/t20-data'
import type { Character } from '@/shared/api/api'
import { spellPmLimit } from '@/shared/lib/engine-wasm'
import { arcanistaCaminhoOf } from './derived'

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
 * its own key attribute (p173), so a multiclass caster gets the better of them;
 * the per-attribute map comes from the computed sheet, with race and item
 * bonuses already in.
 *
 * Takes the character because the Arcanista's atributo-chave is not fixed by the
 * class — it is defined by the Caminho (p37), and a Feiticeiro casts with
 * Carisma (ALE-113).
 *
 * @example bestSpellCd(samira, ['Arcanista'], sheet.spellCdByAttribute) // 19
 */
export function bestSpellCd(
  character: Character,
  applicableClasses: readonly SpellcasterClass[],
  spellCdByAttribute: Record<AttributeKey, number>,
): number | null {
  const caminho = arcanistaCaminhoOf(character)
  const cds = applicableClasses
    .map((className) => spellcastingAttributeFor(className, caminho))
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

/**
 * The p224 PM ceiling for ONE spell: the character's level in the CLASS that
 * grants it, or the character level when the source is not a class (a race, an
 * origin, a general power). Item `pmLimit` bonuses add on top, resolved.
 *
 * NOT the same number as the HUD's "Limite PM" tile, which summarises the
 * character with "best caster level". Gating the cast dialog on the tile offered
 * a Bardo 7 / Arcanista 1 seven PM on an Arcanista spell and the server refused
 * anything over 1 (ALE-92).
 *
 * Choke point: production runs the Go engine — the SAME function the cast
 * handler runs, which is the whole point. The TS branch below exists only so
 * components can render in jsdom (no WASM there) and is dropped from the bundle
 * by `import.meta.env.MODE`; it dies with the other five when the engine becomes
 * the single authority (ALE-104).
 *
 * @example spellPmLimitFor(bardo7Arcanista1, ['Arcanista']) // 1
 */
export function spellPmLimitFor(
  character: Character,
  spellClasses: readonly string[],
): number {
  if (import.meta.env.MODE === 'test') {
    const grants = new Set(spellClasses)
    const best = character.classes
      .filter((c) => grants.has(c.className))
      .reduce((acc, c) => Math.max(acc, c.level), 0)
    return Math.max(1, best === 0 ? character.level : best)
  }
  return spellPmLimit(character, spellClasses)
}
