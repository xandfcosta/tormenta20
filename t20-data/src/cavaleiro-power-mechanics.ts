/**
 * Cavaleiro elective power mechanics — action-economy + PM + uses
 * per poder eletivo de classe.
 *
 * Complementa `abilities/classes/cavaleiro.ts` (nome + descrição).
 * PDF Cap 1 Cavaleiro p52-55.
 *
 * Posturas de Combate (6) são flags `isPostura: true`. Ativar/trocar
 * postura é gratuito no início do turno; efeitos por-trigger detalhados
 * na descrição do poder. Modelo passivo por default; ativação real
 * fica com o motor de combate.
 */

export type CavaleiroPowerAction =
  | 'padrao'
  | 'movimento'
  | 'livre'
  | 'reacao'
  | 'gratuita'
  | 'completa'
  | 'passivo'
  | 'varia'

export type CavaleiroPowerUses = null | 'cena' | 'rodada' | 'dia' | number

export type CavaleiroPower = {
  id: string
  name: string
  action: CavaleiroPowerAction
  pmCost: number | 'variavel'
  uses: CavaleiroPowerUses
  /** True quando o poder é Postura de Combate (ativação stance). */
  isPostura?: boolean
  /** True se depende do caminho escolhido no 5º nível. */
  isCaminho?: boolean
  bookPage: number
}

const RAW: readonly CavaleiroPower[] = [
  { id: 'armadura-da-honra', name: 'Armadura da Honra', action: 'passivo', pmCost: 0, uses: 'cena', bookPage: 53 },
  { id: 'aumento-de-atributo', name: 'Aumento de Atributo', action: 'passivo', pmCost: 0, uses: null, bookPage: 53 },
  { id: 'autoridade-feudal', name: 'Autoridade Feudal', action: 'varia', pmCost: 2, uses: null, bookPage: 53 },
  { id: 'desprezar-os-covardes', name: 'Desprezar os Covardes', action: 'passivo', pmCost: 0, uses: null, bookPage: 53 },
  { id: 'escudeiro', name: 'Escudeiro', action: 'passivo', pmCost: 0, uses: null, bookPage: 53 },
  { id: 'especializacao-em-armadura', name: 'Especialização em Armadura', action: 'passivo', pmCost: 0, uses: null, bookPage: 53 },
  { id: 'estandarte', name: 'Estandarte', action: 'passivo', pmCost: 0, uses: 'cena', bookPage: 54 },
  { id: 'etiqueta', name: 'Etiqueta', action: 'passivo', pmCost: 0, uses: null, bookPage: 54 },
  { id: 'investida-destruidora', name: 'Investida Destruidora', action: 'livre', pmCost: 2, uses: null, bookPage: 54 },
  { id: 'montaria-corajosa', name: 'Montaria Corajosa', action: 'passivo', pmCost: 0, uses: null, bookPage: 54 },
  { id: 'pajem', name: 'Pajem', action: 'passivo', pmCost: 0, uses: null, bookPage: 54 },
  { id: 'postura-ariete-implacavel', name: 'Postura: Ariete Implacável', action: 'passivo', pmCost: 0, uses: null, isPostura: true, bookPage: 54 },
  { id: 'postura-castigo-de-ferro', name: 'Postura: Castigo de Ferro', action: 'passivo', pmCost: 0, uses: null, isPostura: true, bookPage: 54 },
  { id: 'postura-foco-de-batalha', name: 'Postura: Foco de Batalha', action: 'passivo', pmCost: 0, uses: null, isPostura: true, bookPage: 54 },
  { id: 'postura-muralha-intransponivel', name: 'Postura: Muralha Intransponível', action: 'passivo', pmCost: 0, uses: null, isPostura: true, bookPage: 54 },
  { id: 'postura-provocacao-petulante', name: 'Postura: Provocação Petulante', action: 'passivo', pmCost: 0, uses: null, isPostura: true, bookPage: 55 },
  { id: 'postura-torre-inabalavel', name: 'Postura: Torre Inabalável', action: 'passivo', pmCost: 0, uses: null, isPostura: true, bookPage: 55 },
  { id: 'solidez', name: 'Solidez', action: 'passivo', pmCost: 0, uses: null, bookPage: 55 },
  { id: 'titulo', name: 'Título', action: 'passivo', pmCost: 0, uses: null, bookPage: 55 },
  { id: 'torre-armada', name: 'Torre Armada', action: 'livre', pmCost: 1, uses: null, bookPage: 55 },
  { id: 'caminho-bastiao', name: 'Caminho: Bastião', action: 'passivo', pmCost: 0, uses: null, isCaminho: true, bookPage: 55 },
  { id: 'caminho-montaria', name: 'Caminho: Montaria', action: 'passivo', pmCost: 0, uses: null, isCaminho: true, bookPage: 55 },
]

export const CAVALEIRO_ELECTIVES: readonly CavaleiroPower[] = Object.freeze(RAW)

export function cavaleiroElectives(): readonly CavaleiroPower[] {
  return CAVALEIRO_ELECTIVES
}

export function cavaleiroPowerById(id: string): CavaleiroPower | undefined {
  return CAVALEIRO_ELECTIVES.find((p) => p.id === id)
}

/** Retorna as 6 Posturas de Combate. */
export function posturaPowers(): readonly CavaleiroPower[] {
  return CAVALEIRO_ELECTIVES.filter((p) => p.isPostura === true)
}

/** Retorna os poderes de Caminho (Bastião / Montaria). */
export function caminhoPowers(): readonly CavaleiroPower[] {
  return CAVALEIRO_ELECTIVES.filter((p) => p.isCaminho === true)
}

export function activeCavaleiroPowers(): readonly CavaleiroPower[] {
  return CAVALEIRO_ELECTIVES.filter((p) => p.action !== 'passivo')
}
