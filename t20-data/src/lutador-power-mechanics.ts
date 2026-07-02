/**
 * Lutador elective power mechanics — action-economy + PM + uses.
 *
 * Complementa `abilities/classes/lutador.ts` (nome + descrição).
 * PDF Cap 1 Lutador p76-78.
 */

export type LutadorPowerAction =
  | 'padrao'
  | 'movimento'
  | 'livre'
  | 'reacao'
  | 'gratuita'
  | 'completa'
  | 'passivo'
  | 'varia'

export type LutadorPowerUses = null | 'cena' | 'rodada' | 'dia' | number

export type LutadorPower = {
  id: string
  name: string
  action: LutadorPowerAction
  pmCost: number | 'variavel'
  uses: LutadorPowerUses
  bookPage: number
}

const RAW: readonly LutadorPower[] = [
  { id: 'arma-improvisada', name: 'Arma Improvisada', action: 'passivo', pmCost: 0, uses: null, bookPage: 76 },
  { id: 'ate-acertar', name: 'Até Acertar', action: 'passivo', pmCost: 0, uses: 'cena', bookPage: 76 },
  { id: 'aumento-de-atributo', name: 'Aumento de Atributo', action: 'passivo', pmCost: 0, uses: null, bookPage: 76 },
  { id: 'bracos-calejados', name: 'Braços Calejados', action: 'passivo', pmCost: 0, uses: null, bookPage: 76 },
  { id: 'cabecada', name: 'Cabeçada', action: 'livre', pmCost: 2, uses: 'cena', bookPage: 76 },
  { id: 'chave', name: 'Chave', action: 'passivo', pmCost: 0, uses: null, bookPage: 76 },
  { id: 'confianca-dos-ringues', name: 'Confiança dos Ringues', action: 'passivo', pmCost: 0, uses: 'cena', bookPage: 76 },
  { id: 'convencido', name: 'Convencido', action: 'passivo', pmCost: 0, uses: null, bookPage: 77 },
  { id: 'golpe-baixo', name: 'Golpe Baixo', action: 'livre', pmCost: 2, uses: 'cena', bookPage: 77 },
  { id: 'golpe-imprudente', name: 'Golpe Imprudente', action: 'passivo', pmCost: 0, uses: null, bookPage: 77 },
  { id: 'imobilizacao', name: 'Imobilização', action: 'completa', pmCost: 0, uses: null, bookPage: 77 },
  { id: 'lingua-dos-becos', name: 'Língua dos Becos', action: 'livre', pmCost: 1, uses: null, bookPage: 77 },
  { id: 'lutador-de-chao', name: 'Lutador de Chão', action: 'passivo', pmCost: 1, uses: null, bookPage: 77 },
  { id: 'nome-na-arena', name: 'Nome na Arena', action: 'completa', pmCost: 0, uses: 'cena', bookPage: 77 },
  { id: 'punhos-de-adamante', name: 'Punhos de Adamante', action: 'passivo', pmCost: 0, uses: null, bookPage: 77 },
  { id: 'rasteira', name: 'Rasteira', action: 'livre', pmCost: 2, uses: null, bookPage: 77 },
  { id: 'sarado', name: 'Sarado', action: 'passivo', pmCost: 0, uses: null, bookPage: 77 },
  { id: 'sequencia-destruidora', name: 'Sequência Destruidora', action: 'livre', pmCost: 2, uses: null, bookPage: 77 },
  { id: 'trincado', name: 'Trincado', action: 'passivo', pmCost: 0, uses: null, bookPage: 77 },
  { id: 'trocacao', name: 'Trocação', action: 'livre', pmCost: 'variavel', uses: null, bookPage: 77 },
  { id: 'trocacao-tumultuosa', name: 'Trocação Tumultuosa', action: 'livre', pmCost: 2, uses: null, bookPage: 77 },
  { id: 'valentao', name: 'Valentão', action: 'passivo', pmCost: 0, uses: null, bookPage: 78 },
  { id: 'voadora', name: 'Voadora', action: 'livre', pmCost: 2, uses: null, bookPage: 78 },
]

export const LUTADOR_ELECTIVES: readonly LutadorPower[] = Object.freeze(RAW)

export function lutadorElectives(): readonly LutadorPower[] {
  return LUTADOR_ELECTIVES
}

export function lutadorPowerById(id: string): LutadorPower | undefined {
  return LUTADOR_ELECTIVES.find((p) => p.id === id)
}

export function activeLutadorPowers(): readonly LutadorPower[] {
  return LUTADOR_ELECTIVES.filter((p) => p.action !== 'passivo')
}
