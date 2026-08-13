import { FLAG_ACTIVATIONS } from '@/shared/rules/flag-activations'
import type { ConditionalEntry } from '@/entities/character/derived'
import type { Character } from '@/shared/api/api'
import { parseActiveConditions } from './active-conditions'

/**
 * How many things the Efeitos block SHOWS — the number on the tab's pill.
 *
 * Counts units, not raw engine entries: active book conditions + running
 * scene/day effects + each active stance once (Fúria enters the engine as 8
 * tier modifiers, but it is one thing at the table) + each switched-on
 * situational group once. The React version counted raw entries and read "11"
 * for a lone Fúria plus two toggles.
 *
 * @example effectsShownCount(character, allConditionals(character, active))
 */
export function effectsShownCount(
  character: Character,
  entries: readonly ConditionalEntry[],
): number {
  const conditions = parseActiveConditions(character.activeConditions).length
  const running = (character.activeEffects ?? []).length

  const stanceFlags = new Set<string>()
  const situational = new Set<string>()
  for (const entry of entries) {
    if (!entry.active) continue
    const flag = entry.effect.flag
    if (flag && FLAG_ACTIVATIONS[flag]) stanceFlags.add(flag)
    else situational.add(flag ?? entry.id)
  }
  return conditions + running + stanceFlags.size + situational.size
}
