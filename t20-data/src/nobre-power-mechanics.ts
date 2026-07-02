/**
 * Nobre elective power mechanics — action-economy + PM + uses.
 *
 * Complementa `abilities/classes/nobre.ts` (nome + descrição).
 * PDF Cap 1 Nobre p79-82.
 */

export type NobrePowerAction =
  | 'padrao'
  | 'movimento'
  | 'livre'
  | 'reacao'
  | 'gratuita'
  | 'completa'
  | 'passivo'
  | 'varia'

export type NobrePowerUses = null | 'cena' | 'rodada' | 'dia' | number

export type NobrePower = {
  id: string
  name: string
  action: NobrePowerAction
  pmCost: number | 'variavel'
  uses: NobrePowerUses
  bookPage: number
}

const RAW: readonly NobrePower[] = [
  { id: 'armadura-brilhante', name: 'Armadura Brilhante', action: 'passivo', pmCost: 0, uses: null, bookPage: 79 },
  { id: 'aumento-de-atributo', name: 'Aumento de Atributo', action: 'passivo', pmCost: 0, uses: null, bookPage: 79 },
  { id: 'autoridade-feudal', name: 'Autoridade Feudal', action: 'varia', pmCost: 2, uses: null, bookPage: 79 },
  { id: 'educacao-privilegiada', name: 'Educação Privilegiada', action: 'passivo', pmCost: 0, uses: null, bookPage: 79 },
  { id: 'estrategista', name: 'Estrategista', action: 'padrao', pmCost: 'variavel', uses: null, bookPage: 79 },
  { id: 'favor', name: 'Favor', action: 'varia', pmCost: 5, uses: null, bookPage: 79 },
  { id: 'general', name: 'General', action: 'passivo', pmCost: 0, uses: null, bookPage: 80 },
  { id: 'grito-tiranico', name: 'Grito Tirânico', action: 'completa', pmCost: 0, uses: null, bookPage: 80 },
  { id: 'inspirar-confianca', name: 'Inspirar Confiança', action: 'reacao', pmCost: 2, uses: null, bookPage: 80 },
  { id: 'inspirar-gloria', name: 'Inspirar Glória', action: 'livre', pmCost: 5, uses: 'cena', bookPage: 80 },
  { id: 'jogo-da-corte', name: 'Jogo da Corte', action: 'livre', pmCost: 1, uses: null, bookPage: 80 },
  { id: 'liderar-pelo-exemplo', name: 'Liderar pelo Exemplo', action: 'livre', pmCost: 2, uses: null, bookPage: 80 },
  { id: 'lingua-de-ouro', name: 'Língua de Ouro', action: 'padrao', pmCost: 4, uses: null, bookPage: 80 },
  { id: 'lingua-de-prata', name: 'Língua de Prata', action: 'livre', pmCost: 2, uses: null, bookPage: 80 },
  { id: 'lingua-rapida', name: 'Língua Rápida', action: 'passivo', pmCost: 0, uses: null, bookPage: 80 },
  { id: 'presenca-majestosa', name: 'Presença Majestosa', action: 'passivo', pmCost: 0, uses: null, bookPage: 80 },
  { id: 'titulo', name: 'Título', action: 'passivo', pmCost: 0, uses: null, bookPage: 80 },
  { id: 'voz-poderosa', name: 'Voz Poderosa', action: 'passivo', pmCost: 0, uses: null, bookPage: 80 },
]

export const NOBRE_ELECTIVES: readonly NobrePower[] = Object.freeze(RAW)

export function nobreElectives(): readonly NobrePower[] {
  return NOBRE_ELECTIVES
}

export function nobrePowerById(id: string): NobrePower | undefined {
  return NOBRE_ELECTIVES.find((p) => p.id === id)
}

export function activeNobrePowers(): readonly NobrePower[] {
  return NOBRE_ELECTIVES.filter((p) => p.action !== 'passivo')
}
