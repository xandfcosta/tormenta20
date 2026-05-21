export const ATTRIBUTE_KEYS = [
  'strength',
  'dexterity',
  'constitution',
  'intelligence',
  'wisdom',
  'charisma',
] as const

export type AttributeKey = (typeof ATTRIBUTE_KEYS)[number]

export const ATTRIBUTE_ABBR: Record<AttributeKey, string> = {
  strength: 'FOR',
  dexterity: 'DES',
  constitution: 'CON',
  intelligence: 'INT',
  wisdom: 'SAB',
  charisma: 'CAR',
}
