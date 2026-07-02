/**
 * Ladino elective power mechanics — action-economy + PM + uses.
 *
 * Complementa `abilities/classes/ladino.ts` (nome + descrição).
 * PDF Cap 1 Ladino p72-75.
 */

export type LadinoPowerAction =
  | 'padrao'
  | 'movimento'
  | 'livre'
  | 'reacao'
  | 'gratuita'
  | 'completa'
  | 'passivo'
  | 'varia'

export type LadinoPowerUses = null | 'cena' | 'rodada' | 'dia' | number

export type LadinoPower = {
  id: string
  name: string
  action: LadinoPowerAction
  pmCost: number | 'variavel'
  uses: LadinoPowerUses
  bookPage: number
}

const RAW: readonly LadinoPower[] = [
  { id: 'assassinar', name: 'Assassinar', action: 'movimento', pmCost: 3, uses: null, bookPage: 73 },
  { id: 'aumento-de-atributo', name: 'Aumento de Atributo', action: 'passivo', pmCost: 0, uses: null, bookPage: 73 },
  { id: 'contatos-no-submundo', name: 'Contatos no Submundo', action: 'passivo', pmCost: 2, uses: null, bookPage: 73 },
  { id: 'emboscar', name: 'Emboscar', action: 'livre', pmCost: 2, uses: 'rodada', bookPage: 73 },
  { id: 'escapista', name: 'Escapista', action: 'passivo', pmCost: 0, uses: null, bookPage: 73 },
  { id: 'fuga-formidavel', name: 'Fuga Formidável', action: 'completa', pmCost: 1, uses: 'cena', bookPage: 73 },
  { id: 'gatuno', name: 'Gatuno', action: 'passivo', pmCost: 0, uses: null, bookPage: 73 },
  { id: 'ladrao-arcano', name: 'Ladrão Arcano', action: 'passivo', pmCost: 'variavel', uses: 'cena', bookPage: 74 },
  { id: 'mao-na-boca', name: 'Mão na Boca', action: 'passivo', pmCost: 0, uses: null, bookPage: 74 },
  { id: 'maos-rapidas', name: 'Mãos Rápidas', action: 'livre', pmCost: 1, uses: 'rodada', bookPage: 74 },
  { id: 'mente-criminosa', name: 'Mente Criminosa', action: 'passivo', pmCost: 0, uses: null, bookPage: 74 },
  { id: 'oportunismo', name: 'Oportunismo', action: 'reacao', pmCost: 2, uses: 'rodada', bookPage: 74 },
  { id: 'rolamento-defensivo', name: 'Rolamento Defensivo', action: 'reacao', pmCost: 2, uses: null, bookPage: 74 },
  { id: 'roubo-de-mana', name: 'Roubo de Mana', action: 'passivo', pmCost: 0, uses: 'cena', bookPage: 74 },
  { id: 'saqueador-de-tumbas', name: 'Saqueador de Tumbas', action: 'passivo', pmCost: 0, uses: null, bookPage: 74 },
  { id: 'sombra', name: 'Sombra', action: 'passivo', pmCost: 0, uses: null, bookPage: 74 },
  { id: 'truque-magico', name: 'Truque Mágico', action: 'varia', pmCost: 'variavel', uses: null, bookPage: 74 },
  { id: 'velocidade-ladina', name: 'Velocidade Ladina', action: 'livre', pmCost: 2, uses: 'rodada', bookPage: 74 },
  { id: 'veneno-persistente', name: 'Veneno Persistente', action: 'passivo', pmCost: 0, uses: null, bookPage: 74 },
  { id: 'veneno-potente', name: 'Veneno Potente', action: 'passivo', pmCost: 0, uses: null, bookPage: 74 },
]

export const LADINO_ELECTIVES: readonly LadinoPower[] = Object.freeze(RAW)

export function ladinoElectives(): readonly LadinoPower[] {
  return LADINO_ELECTIVES
}

export function ladinoPowerById(id: string): LadinoPower | undefined {
  return LADINO_ELECTIVES.find((p) => p.id === id)
}

export function activeLadinoPowers(): readonly LadinoPower[] {
  return LADINO_ELECTIVES.filter((p) => p.action !== 'passivo')
}
