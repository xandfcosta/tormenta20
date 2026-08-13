import type { MonsterTipo } from '@tormenta20/t20-data'

/** Ordered MonsterTipo list — the order the filter chips appear in. */
export const MONSTER_TIPOS: readonly MonsterTipo[] = [
  'humanoide',
  'animal',
  'monstro',
  'morto-vivo',
  'construto',
  'espirito',
  'planar',
]

/** Display labels per MonsterTipo (bestiário + adicionar monstro na sessão). */
export const MONSTER_TIPO_LABEL: Record<MonsterTipo, string> = {
  humanoide: 'Humanoide',
  animal: 'Animal',
  monstro: 'Monstro',
  'morto-vivo': 'Morto-vivo',
  construto: 'Construto',
  espirito: 'Espírito',
  planar: 'Planar',
}

/**
 * Renders a fractional ND the way the book writes it (0.25 → "1/4"). A bare
 * "0.25" on a monster row reads like a rounding artifact instead of a rating.
 *
 * @example formatNd(0.5) // '1/2'
 */
export function formatNd(nd: number): string {
  if (nd < 1) {
    if (Math.abs(nd - 0.25) < 0.001) return '1/4'
    if (Math.abs(nd - 0.5) < 0.001) return '1/2'
  }
  return String(nd)
}
