import type { Character, CharacterExpertise } from '@/shared/api/api'
import {
  ATTRIBUTE_ABBR,
  ATTRIBUTE_KEYS,
  EXPERTISES as RAW_EXPERTISES,
  trainingBonusForLevel,
  type AttributeKey,
  type ExpertiseName,
} from '@tormenta20/t20-data'

export { ATTRIBUTE_ABBR, ATTRIBUTE_KEYS, trainingBonusForLevel }
export type { AttributeKey, ExpertiseName }

export type ExpertiseDef = {
  name: string
  attribute: AttributeKey
  abbr: string
  /** Cannot be used at all unless the character is trained in it. */
  trainedOnly?: boolean
}

const TRAINED_ONLY = new Set<string>([
  'Adestramento',
  'Conhecimento',
  'Guerra',
  'Jogatina',
  'Ladinagem',
  'Misticismo',
  'Nobreza',
  'Ofício',
  'Pilotagem',
  'Religião',
])

export const EXPERTISES: ExpertiseDef[] = RAW_EXPERTISES.map((e) => ({
  name: e.name,
  attribute: e.attribute,
  abbr: ATTRIBUTE_ABBR[e.attribute],
  trainedOnly: TRAINED_ONLY.has(e.name),
}))

export function expertiseTotal(
  character: Character,
  state: CharacterExpertise,
): number {
  const halfLevel = Math.floor(character.level / 2)
  const attrValue = character[state.attribute]
  const training = state.trained ? trainingBonusForLevel(character.level) : 0
  return halfLevel + attrValue + training
}

export function expertiseStateFor(
  character: Character,
  def: ExpertiseDef,
): CharacterExpertise {
  return (
    character.expertises.find((e) => e.name === def.name) ?? {
      name: def.name,
      attribute: def.attribute,
      trained: false,
      custom: false,
    }
  )
}
