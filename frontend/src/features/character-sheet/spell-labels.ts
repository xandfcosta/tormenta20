import type { SpellCircle, SpellSchool } from '@tormenta20/t20-data'

export const SCHOOL_LABEL: Record<SpellSchool, string> = {
  abjuracao: 'Abjuração',
  adivinhacao: 'Adivinhação',
  convocacao: 'Convocação',
  encantamento: 'Encantamento',
  evocacao: 'Evocação',
  ilusao: 'Ilusão',
  necromancia: 'Necromancia',
  transmutacao: 'Transmutação',
}

export const CIRCLE_LABEL: Record<SpellCircle, string> = {
  0: 'Truque',
  1: '1º',
  2: '2º',
  3: '3º',
  4: '4º',
  5: '5º',
}
