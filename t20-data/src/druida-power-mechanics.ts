/**
 * Druida elective power mechanics — action-economy + PM + uses.
 *
 * Complementa `abilities/classes/druida.ts` (nome + descrição).
 * PDF Cap 1 Druida p60-63.
 *
 * Forma Selvagem (3 tiers) escala PM (3/6/10) mantendo mesma ação
 * completa. Aspectos (4) mixados: Primavera/Inverno passivos +
 * Verão/Outono ativos (livre 1 PM).
 */

export type DruidaPowerAction =
  | 'padrao'
  | 'movimento'
  | 'livre'
  | 'reacao'
  | 'gratuita'
  | 'completa'
  | 'passivo'
  | 'varia'

export type DruidaPowerUses = null | 'cena' | 'rodada' | 'dia' | number

export type DruidaPower = {
  id: string
  name: string
  action: DruidaPowerAction
  pmCost: number | 'variavel'
  uses: DruidaPowerUses
  /** True se é um Aspecto (Primavera/Verão/Outono/Inverno). */
  isAspecto?: boolean
  /** True se é uma tier de Forma Selvagem. */
  isFormaSelvagem?: boolean
  bookPage: number
}

const RAW: readonly DruidaPower[] = [
  { id: 'aspecto-da-primavera', name: 'Aspecto da Primavera', action: 'passivo', pmCost: 0, uses: null, isAspecto: true, bookPage: 61 },
  { id: 'aspecto-do-verao', name: 'Aspecto do Verão', action: 'livre', pmCost: 1, uses: 'cena', isAspecto: true, bookPage: 61 },
  { id: 'aspecto-do-outono', name: 'Aspecto do Outono', action: 'livre', pmCost: 1, uses: null, isAspecto: true, bookPage: 61 },
  { id: 'aspecto-do-inverno', name: 'Aspecto do Inverno', action: 'passivo', pmCost: 0, uses: null, isAspecto: true, bookPage: 61 },
  { id: 'aumento-de-atributo', name: 'Aumento de Atributo', action: 'passivo', pmCost: 0, uses: null, bookPage: 61 },
  { id: 'companheiro-animal', name: 'Companheiro Animal', action: 'passivo', pmCost: 0, uses: null, bookPage: 61 },
  { id: 'companheiro-animal-aprimorado', name: 'Companheiro Animal Aprimorado', action: 'passivo', pmCost: 0, uses: null, bookPage: 61 },
  { id: 'companheiro-animal-lendario', name: 'Companheiro Animal Lendário', action: 'passivo', pmCost: 0, uses: null, bookPage: 62 },
  { id: 'companheiro-animal-magico', name: 'Companheiro Animal Mágico', action: 'passivo', pmCost: 0, uses: null, bookPage: 62 },
  { id: 'coracao-da-selva', name: 'Coração da Selva', action: 'passivo', pmCost: 0, uses: null, bookPage: 62 },
  { id: 'espirito-dos-equinocios', name: 'Espírito dos Equinócios', action: 'livre', pmCost: 4, uses: 'cena', bookPage: 62 },
  { id: 'espirito-dos-solsticios', name: 'Espírito dos Solstícios', action: 'passivo', pmCost: 4, uses: null, bookPage: 62 },
  { id: 'forca-dos-penhascos', name: 'Força dos Penhascos', action: 'reacao', pmCost: 'variavel', uses: null, bookPage: 62 },
  { id: 'forma-primal', name: 'Forma Primal', action: 'passivo', pmCost: 0, uses: null, bookPage: 62 },
  { id: 'forma-selvagem', name: 'Forma Selvagem', action: 'completa', pmCost: 3, uses: null, isFormaSelvagem: true, bookPage: 62 },
  { id: 'forma-selvagem-aprimorada', name: 'Forma Selvagem Aprimorada', action: 'completa', pmCost: 6, uses: null, isFormaSelvagem: true, bookPage: 62 },
  { id: 'forma-selvagem-superior', name: 'Forma Selvagem Superior', action: 'completa', pmCost: 10, uses: null, isFormaSelvagem: true, bookPage: 62 },
  { id: 'liberdade-da-pradaria', name: 'Liberdade da Pradaria', action: 'passivo', pmCost: 1, uses: null, bookPage: 63 },
  { id: 'magia-natural', name: 'Magia Natural', action: 'passivo', pmCost: 0, uses: null, bookPage: 63 },
  { id: 'presas-afiadas', name: 'Presas Afiadas', action: 'passivo', pmCost: 0, uses: null, bookPage: 63 },
  { id: 'segredos-da-natureza', name: 'Segredos da Natureza', action: 'passivo', pmCost: 0, uses: null, bookPage: 63 },
  { id: 'tranquilidade-dos-lagos', name: 'Tranquilidade dos Lagos', action: 'reacao', pmCost: 1, uses: 'rodada', bookPage: 63 },
]

export const DRUIDA_ELECTIVES: readonly DruidaPower[] = Object.freeze(RAW)

export function druidaElectives(): readonly DruidaPower[] {
  return DRUIDA_ELECTIVES
}

export function druidaPowerById(id: string): DruidaPower | undefined {
  return DRUIDA_ELECTIVES.find((p) => p.id === id)
}

export function aspectoPowers(): readonly DruidaPower[] {
  return DRUIDA_ELECTIVES.filter((p) => p.isAspecto === true)
}

export function formaSelvagemPowers(): readonly DruidaPower[] {
  return DRUIDA_ELECTIVES.filter((p) => p.isFormaSelvagem === true)
}

export function activeDruidaPowers(): readonly DruidaPower[] {
  return DRUIDA_ELECTIVES.filter((p) => p.action !== 'passivo')
}
