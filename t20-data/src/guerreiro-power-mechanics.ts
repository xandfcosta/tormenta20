/**
 * Guerreiro elective power mechanics — action-economy + PM + uses.
 *
 * Complementa `abilities/classes/guerreiro.ts` (nome + descrição).
 * PDF Cap 1 Guerreiro p64-67.
 *
 * Auto powers (Ataque Extra, Redução de Dano por nível, Especialidade
 * em Arma auto) já cobertos em `damage-reduction.guerreiroRdForLevel`
 * + progressão de classe.
 */

export type GuerreiroPowerAction =
  | 'padrao'
  | 'movimento'
  | 'livre'
  | 'reacao'
  | 'gratuita'
  | 'completa'
  | 'passivo'
  | 'varia'

export type GuerreiroPowerUses = null | 'cena' | 'rodada' | 'dia' | number

export type GuerreiroPower = {
  id: string
  name: string
  action: GuerreiroPowerAction
  pmCost: number | 'variavel'
  uses: GuerreiroPowerUses
  bookPage: number
}

const RAW: readonly GuerreiroPower[] = [
  { id: 'ambidestria', name: 'Ambidestria', action: 'passivo', pmCost: 0, uses: null, bookPage: 65 },
  { id: 'arqueiro', name: 'Arqueiro', action: 'passivo', pmCost: 0, uses: null, bookPage: 65 },
  { id: 'ataque-reflexo', name: 'Ataque Reflexo', action: 'reacao', pmCost: 1, uses: 'rodada', bookPage: 65 },
  { id: 'aumento-de-atributo', name: 'Aumento de Atributo', action: 'passivo', pmCost: 0, uses: null, bookPage: 65 },
  { id: 'bater-e-correr', name: 'Bater e Correr', action: 'passivo', pmCost: 2, uses: null, bookPage: 65 },
  { id: 'destruidor', name: 'Destruidor', action: 'passivo', pmCost: 0, uses: null, bookPage: 65 },
  { id: 'esgrimista', name: 'Esgrimista', action: 'passivo', pmCost: 0, uses: null, bookPage: 65 },
  { id: 'especializacao-em-arma', name: 'Especialização em Arma', action: 'passivo', pmCost: 0, uses: null, bookPage: 65 },
  { id: 'especializacao-em-armadura', name: 'Especialização em Armadura', action: 'passivo', pmCost: 0, uses: null, bookPage: 65 },
  { id: 'golpe-de-raspao', name: 'Golpe de Raspão', action: 'reacao', pmCost: 2, uses: 'rodada', bookPage: 66 },
  { id: 'golpe-demolidor', name: 'Golpe Demolidor', action: 'livre', pmCost: 2, uses: null, bookPage: 66 },
  { id: 'golpe-pessoal', name: 'Golpe Pessoal', action: 'varia', pmCost: 'variavel', uses: null, bookPage: 66 },
  { id: 'impeto', name: 'Ímpeto', action: 'livre', pmCost: 1, uses: null, bookPage: 66 },
  { id: 'mestre-em-arma', name: 'Mestre em Arma', action: 'passivo', pmCost: 2, uses: null, bookPage: 66 },
  { id: 'planejamento-marcial', name: 'Planejamento Marcial', action: 'varia', pmCost: 3, uses: 'dia', bookPage: 67 },
  { id: 'romper-resistencias', name: 'Romper Resistências', action: 'passivo', pmCost: 1, uses: null, bookPage: 67 },
  { id: 'solidez', name: 'Solidez', action: 'passivo', pmCost: 0, uses: null, bookPage: 67 },
  { id: 'tornado-de-dor', name: 'Tornado de Dor', action: 'padrao', pmCost: 2, uses: null, bookPage: 67 },
  { id: 'valentao', name: 'Valentão', action: 'passivo', pmCost: 0, uses: null, bookPage: 67 },
]

export const GUERREIRO_ELECTIVES: readonly GuerreiroPower[] = Object.freeze(RAW)

export function guerreiroElectives(): readonly GuerreiroPower[] {
  return GUERREIRO_ELECTIVES
}

export function guerreiroPowerById(id: string): GuerreiroPower | undefined {
  return GUERREIRO_ELECTIVES.find((p) => p.id === id)
}

export function activeGuerreiroPowers(): readonly GuerreiroPower[] {
  return GUERREIRO_ELECTIVES.filter((p) => p.action !== 'passivo')
}
