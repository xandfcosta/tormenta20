import { pointBuyCost } from '@tormenta20/t20-data'
import { ATTRIBUTE_KEYS, type AttributeKey } from '@/shared/api/attribute-keys'
import type { AttributeMode } from '@/shared/stores/character-draft-store'
import {
  type RaceChoiceState,
  appliedRaceDeltas,
  draftTormentaCarismaExtra,
} from './grant-helpers'
import type { CharacterFormValues } from './wizard-steps'

export const ATTRIBUTE_LABEL: Record<AttributeKey, string> = {
  strength: 'Força',
  dexterity: 'Destreza',
  constitution: 'Constituição',
  intelligence: 'Inteligência',
  wisdom: 'Sabedoria',
  charisma: 'Carisma',
}

export type AttributeRow = {
  key: AttributeKey
  label: string
  /** What the player edits — the pre-race value the character is saved with. */
  base: number
  /** Everything derived on top: racial mods plus the Tormenta Carisma loss. */
  raceDelta: number
  total: number
  /** Point-buy cost of `base`, or null in free mode / outside the table. */
  cost: number | null
}

type AttributeValues = Pick<
  CharacterFormValues,
  AttributeKey | 'races' | 'classPowers' | 'powerChoices' | 'originChoices'
>

/**
 * The six attribute columns of the Forja: what the player typed, what the
 * character sheet will add on top, and what each point costs.
 *
 * Base and delta stay apart on purpose — the base is what gets saved, and the
 * racial mod is re-derived by the sheet. Baking the delta into the field would
 * double-count it the moment the character is opened.
 *
 * @example attributeRows(draft.values, draft.raceChoices, 'point-buy')
 */
export function attributeRows(
  values: AttributeValues,
  raceChoices: RaceChoiceState,
  mode: AttributeMode,
): AttributeRow[] {
  const deltas = appliedRaceDeltas(values.races, raceChoices)
  // Carisma lost to poderes da Tormenta (p136) is derived from the choices, not
  // typed, so it belongs with the racial delta rather than in the editable base.
  const carismaExtra = draftTormentaCarismaExtra(
    values.races,
    raceChoices,
    values.classPowers ?? [],
    values.powerChoices ?? {},
    values.originChoices ?? [],
  )

  return ATTRIBUTE_KEYS.map((key) => {
    const base = values[key] ?? 0
    const raceDelta = (deltas[key] ?? 0) + (key === 'charisma' ? carismaExtra : 0)
    return {
      key,
      label: ATTRIBUTE_LABEL[key],
      base,
      raceDelta,
      total: base + raceDelta,
      cost: mode === 'point-buy' ? costOf(base) : null,
    }
  })
}

/** `pointBuyCost` throws outside −1..4; free-mode values simply have no price. */
function costOf(base: number): number | null {
  try {
    return pointBuyCost(base)
  } catch {
    return null
  }
}
