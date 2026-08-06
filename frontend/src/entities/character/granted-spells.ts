import type { AttributeKey, CatalogSpell } from '@tormenta20/t20-data'
import { getClassPower } from '@/shared/lib/abilities-cache'
import { spellByName } from '@/shared/lib/spell-cache'
import type { Character } from '@/shared/api/api'
import { parseChoiceSet } from './derived'

export type GrantedSpell = {
  spell: CatalogSpell
  /** Display name of the power that teaches the spell ("Totem Espiritual"). */
  sourcePower: string
  /** Key attribute the power casts with (Totem: Sab, PDF p42) — overrides the
   *  class spellcasting attribute in CD math. */
  keyAttribute: AttributeKey
}

/**
 * Spells taught by owned class powers whose sub-choice carries
 * `grantsSpellAttribute` (Bárbaro Totem Espiritual, PDF p42): the stored pick's
 * option `note` names the spell, resolved accent-insensitively against the
 * catalog. Unknown picks/names are skipped, never thrown.
 * Ex.: grantedSpells(barbaroComTotemLobo) →
 *   [{ spell: Concentração de Combate, sourcePower: 'Totem Espiritual',
 *      keyAttribute: 'wisdom' }]
 */
export function grantedSpells(character: Character): GrantedSpell[] {
  const picksByPower = parsePowerChoicePicks(character.powerChoices)
  const out: GrantedSpell[] = []
  for (const powerId of parseChoiceSet(character.classPowers)) {
    const power = getClassPower(powerId)
    const choice = power?.choice
    if (!power || !choice?.grantsSpellAttribute || !choice.options) continue
    for (const pickId of picksByPower[powerId] ?? []) {
      const note = choice.options.find((o) => o.id === pickId)?.note
      const spell = note ? spellByName(note) : null
      if (!spell) continue
      out.push({
        spell,
        sourcePower: power.name,
        keyAttribute: choice.grantsSpellAttribute,
      })
    }
  }
  return out
}

/** Parse the `powerChoices` JSON blob ({ [powerId]: pickedOptionIds[] }) the
 *  creation wizard writes; malformed blobs degrade to no picks. */
function parsePowerChoicePicks(raw: string): Record<string, string[]> {
  let blob: unknown
  try {
    blob = JSON.parse(raw)
  } catch {
    return {}
  }
  if (!blob || typeof blob !== 'object' || Array.isArray(blob)) return {}
  const out: Record<string, string[]> = {}
  for (const [powerId, picks] of Object.entries(blob)) {
    if (!Array.isArray(picks)) continue
    out[powerId] = picks.filter((x): x is string => typeof x === 'string')
  }
  return out
}
