import type { AttributeKey } from './attributes'

export type ExpertiseDef = {
  name: string
  attribute: AttributeKey
}

export const EXPERTISES = [
  { name: 'Acrobacia', attribute: 'dexterity' },
  { name: 'Adestramento', attribute: 'charisma' },
  { name: 'Atletismo', attribute: 'strength' },
  { name: 'Atuação', attribute: 'charisma' },
  { name: 'Cavalgar', attribute: 'dexterity' },
  { name: 'Conhecimento', attribute: 'intelligence' },
  { name: 'Cura', attribute: 'wisdom' },
  { name: 'Diplomacia', attribute: 'charisma' },
  { name: 'Enganação', attribute: 'charisma' },
  { name: 'Fortitude', attribute: 'constitution' },
  { name: 'Furtividade', attribute: 'dexterity' },
  { name: 'Guerra', attribute: 'intelligence' },
  { name: 'Iniciativa', attribute: 'dexterity' },
  { name: 'Intimidação', attribute: 'charisma' },
  { name: 'Intuição', attribute: 'wisdom' },
  { name: 'Investigação', attribute: 'intelligence' },
  { name: 'Jogatina', attribute: 'charisma' },
  { name: 'Ladinagem', attribute: 'dexterity' },
  { name: 'Luta', attribute: 'strength' },
  { name: 'Misticismo', attribute: 'intelligence' },
  { name: 'Nobreza', attribute: 'intelligence' },
  { name: 'Ofício', attribute: 'intelligence' },
  { name: 'Percepção', attribute: 'wisdom' },
  { name: 'Pilotagem', attribute: 'dexterity' },
  { name: 'Pontaria', attribute: 'dexterity' },
  { name: 'Reflexos', attribute: 'dexterity' },
  { name: 'Religião', attribute: 'wisdom' },
  { name: 'Sobrevivência', attribute: 'wisdom' },
  { name: 'Vontade', attribute: 'wisdom' },
] as const satisfies ReadonlyArray<ExpertiseDef>

export type ExpertiseName = (typeof EXPERTISES)[number]['name']

export const EXPERTISE_NAMES: ReadonlyArray<ExpertiseName> = EXPERTISES.map(
  (e) => e.name,
)

export function trainingBonusForLevel(level: number): number {
  if (level >= 15) return 6
  if (level >= 7) return 4
  return 2
}
