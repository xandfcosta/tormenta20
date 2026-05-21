export const SIZES = [
  'Minúsculo',
  'Pequeno',
  'Médio',
  'Grande',
  'Enorme',
  'Colossal',
] as const

export type Size = (typeof SIZES)[number]

export const DEFAULT_SIZE: Size = 'Médio'
export const DEFAULT_DISPLACEMENT = 9
