/**
 * Os seis atributos, escritos À MÃO de propósito.
 *
 * O gerador de tipos da fronteira (`engine-types.ts`) importa `AttributeKey`
 * daqui em vez de emiti-lo: no Go a chave desses mapas é `string`, e uma struct
 * não sabe expressar união de literais. Deixar o Go mandar aqui trocaria um erro
 * de compilação por um `undefined` em runtime — `sheet.attributes.forca` passaria
 * batido (ALE-108).
 *
 * É a única coisa da fronteira que o TS declara sozinho, e é por isso.
 */
export const ATTRIBUTE_KEYS = [
  'strength',
  'dexterity',
  'constitution',
  'intelligence',
  'wisdom',
  'charisma',
] as const

export type AttributeKey = (typeof ATTRIBUTE_KEYS)[number]

/** Abreviação de ficha, em pt-BR. */
export const ATTRIBUTE_ABBR: Record<AttributeKey, string> = {
  strength: 'FOR',
  dexterity: 'DES',
  constitution: 'CON',
  intelligence: 'INT',
  wisdom: 'SAB',
  charisma: 'CAR',
}
