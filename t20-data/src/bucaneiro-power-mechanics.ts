/**
 * Bucaneiro elective power mechanics — action-economy + PM + uses
 * per poder eletivo de classe.
 *
 * Complementa `abilities/classes/bucaneiro.ts` (nome + descrição).
 * PDF Cap 1 Bucaneiro p45-48.
 */

export type BucaneiroPowerAction =
  | 'padrao'
  | 'movimento'
  | 'livre'
  | 'reacao'
  | 'gratuita'
  | 'completa'
  | 'passivo'
  | 'varia'

export type BucaneiroPowerUses = null | 'cena' | 'rodada' | 'dia' | number

export type BucaneiroPower = {
  id: string
  name: string
  action: BucaneiroPowerAction
  pmCost: number | 'variavel'
  uses: BucaneiroPowerUses
  bookPage: number
}

const RAW: readonly BucaneiroPower[] = [
  { id: 'abusar-dos-fracos', name: 'Abusar dos Fracos', action: 'passivo', pmCost: 0, uses: null, bookPage: 47 },
  { id: 'amigos-no-porto', name: 'Amigos no Porto', action: 'passivo', pmCost: 0, uses: 'dia', bookPage: 47 },
  { id: 'aparar', name: 'Aparar', action: 'reacao', pmCost: 1, uses: null, bookPage: 47 },
  { id: 'apostador', name: 'Apostador', action: 'passivo', pmCost: 0, uses: 'dia', bookPage: 47 },
  { id: 'ataque-acrobatico', name: 'Ataque Acrobático', action: 'passivo', pmCost: 0, uses: null, bookPage: 47 },
  { id: 'aumento-de-atributo', name: 'Aumento de Atributo', action: 'passivo', pmCost: 0, uses: null, bookPage: 47 },
  { id: 'aventureiro-avido', name: 'Aventureiro Ávido', action: 'livre', pmCost: 5, uses: 'rodada', bookPage: 47 },
  { id: 'bravata-audaz', name: 'Bravata Audaz', action: 'passivo', pmCost: 0, uses: null, bookPage: 47 },
  { id: 'bravata-imprudente', name: 'Bravata Imprudente', action: 'passivo', pmCost: 0, uses: null, bookPage: 47 },
  { id: 'en-garde', name: 'En Garde', action: 'movimento', pmCost: 1, uses: 'cena', bookPage: 47 },
  { id: 'esgrimista', name: 'Esgrimista', action: 'passivo', pmCost: 0, uses: null, bookPage: 48 },
  { id: 'flagelo-dos-mares', name: 'Flagelo dos Mares', action: 'varia', pmCost: 'variavel', uses: null, bookPage: 48 },
  { id: 'foliao', name: 'Folião', action: 'passivo', pmCost: 0, uses: null, bookPage: 48 },
  { id: 'grudar-o-cano', name: 'Grudar o Cano', action: 'passivo', pmCost: 0, uses: null, bookPage: 48 },
  { id: 'pernas-do-mar', name: 'Pernas do Mar', action: 'passivo', pmCost: 0, uses: null, bookPage: 48 },
  { id: 'pistoleiro', name: 'Pistoleiro', action: 'passivo', pmCost: 0, uses: null, bookPage: 48 },
  { id: 'presenca-paralisante', name: 'Presença Paralisante', action: 'passivo', pmCost: 0, uses: null, bookPage: 48 },
  { id: 'ripostar', name: 'Ripostar', action: 'reacao', pmCost: 1, uses: null, bookPage: 48 },
  { id: 'touche', name: 'Touché', action: 'livre', pmCost: 2, uses: null, bookPage: 48 },
]

export const BUCANEIRO_ELECTIVES: readonly BucaneiroPower[] = Object.freeze(RAW)

export function bucaneiroElectives(): readonly BucaneiroPower[] {
  return BUCANEIRO_ELECTIVES
}

export function bucaneiroPowerById(id: string): BucaneiroPower | undefined {
  return BUCANEIRO_ELECTIVES.find((p) => p.id === id)
}

export function activeBucaneiroPowers(): readonly BucaneiroPower[] {
  return BUCANEIRO_ELECTIVES.filter((p) => p.action !== 'passivo')
}
