import type { MonsterTipo } from '@tormenta20/t20-data'

/** Ordered MonsterTipo list for filter chips. */
export const MONSTER_TIPOS: readonly MonsterTipo[] = [
  'humanoide',
  'animal',
  'monstro',
  'morto-vivo',
  'construto',
  'espirito',
  'planar',
]

/** Display labels for each MonsterTipo (bestiary browser + in-session add). */
export const MONSTER_TIPO_LABEL: Record<MonsterTipo, string> = {
  humanoide: 'Humanoide',
  animal: 'Animal',
  monstro: 'Monstro',
  'morto-vivo': 'Morto-vivo',
  construto: 'Construto',
  espirito: 'Espírito',
  planar: 'Planar',
}

/** Renders fractional NDs as book fractions (0.25 → "1/4"), else the number. */
export function formatNd(nd: number): string {
  if (nd < 1) {
    if (Math.abs(nd - 0.25) < 0.001) return '1/4'
    if (Math.abs(nd - 0.5) < 0.001) return '1/2'
  }
  return String(nd)
}

/** Accent- and case-insensitive normalize for monster name search. */
export function normalizeMonsterName(s: string): string {
  return s
    .normalize('NFD')
    .replace(/[̀-ͯ]/g, '')
    .toLowerCase()
    .trim()
}
