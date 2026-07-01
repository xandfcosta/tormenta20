/**
 * Caçador elective power mechanics — action-economy + PM + uses
 * per poder eletivo de classe.
 *
 * Complementa `abilities/classes/cacador.ts` (nome + descrição).
 * PDF Cap 1 Caçador p48-51.
 *
 * Armadilhas (Arataca, Espinhos, Laço, Rede) modeladas como padrão
 * pmCost 0 — deploy físico (materiais + tempo separado do PM).
 */

export type CacadorPowerAction =
  | 'padrao'
  | 'movimento'
  | 'livre'
  | 'reacao'
  | 'gratuita'
  | 'completa'
  | 'passivo'
  | 'varia'

export type CacadorPowerUses = null | 'cena' | 'rodada' | 'dia' | number

export type CacadorPower = {
  id: string
  name: string
  action: CacadorPowerAction
  pmCost: number | 'variavel'
  uses: CacadorPowerUses
  /** True se o poder é da subcategoria "Armadilha:" (deploy físico). */
  isArmadilha?: boolean
  bookPage: number
}

const RAW: readonly CacadorPower[] = [
  { id: 'ambidestria', name: 'Ambidestria', action: 'passivo', pmCost: 0, uses: null, bookPage: 50 },
  { id: 'armadilha-arataca', name: 'Armadilha: Arataca', action: 'padrao', pmCost: 0, uses: null, isArmadilha: true, bookPage: 50 },
  { id: 'armadilha-espinhos', name: 'Armadilha: Espinhos', action: 'padrao', pmCost: 0, uses: null, isArmadilha: true, bookPage: 50 },
  { id: 'armadilha-laco', name: 'Armadilha: Laço', action: 'padrao', pmCost: 0, uses: null, isArmadilha: true, bookPage: 50 },
  { id: 'armadilha-rede', name: 'Armadilha: Rede', action: 'padrao', pmCost: 0, uses: null, isArmadilha: true, bookPage: 50 },
  { id: 'armadilheiro', name: 'Armadilheiro', action: 'passivo', pmCost: 0, uses: null, bookPage: 50 },
  { id: 'arqueiro', name: 'Arqueiro', action: 'passivo', pmCost: 0, uses: null, bookPage: 50 },
  { id: 'aumento-de-atributo', name: 'Aumento de Atributo', action: 'passivo', pmCost: 0, uses: null, bookPage: 50 },
  { id: 'bote', name: 'Bote', action: 'livre', pmCost: 1, uses: null, bookPage: 50 },
  { id: 'camuflagem', name: 'Camuflagem', action: 'livre', pmCost: 2, uses: null, bookPage: 50 },
  { id: 'chuva-de-laminas', name: 'Chuva de Lâminas', action: 'livre', pmCost: 2, uses: 'rodada', bookPage: 50 },
  { id: 'companheiro-animal', name: 'Companheiro Animal', action: 'passivo', pmCost: 0, uses: null, bookPage: 50 },
  { id: 'elo-com-a-natureza', name: 'Elo com a Natureza', action: 'varia', pmCost: 'variavel', uses: null, bookPage: 50 },
  { id: 'emboscar', name: 'Emboscar', action: 'livre', pmCost: 2, uses: 'rodada', bookPage: 50 },
  { id: 'empatia-selvagem', name: 'Empatia Selvagem', action: 'passivo', pmCost: 0, uses: null, bookPage: 50 },
  { id: 'escaramuca', name: 'Escaramuça', action: 'passivo', pmCost: 0, uses: null, bookPage: 51 },
  { id: 'escaramuca-superior', name: 'Escaramuça Superior', action: 'passivo', pmCost: 0, uses: null, bookPage: 51 },
  { id: 'espreitar', name: 'Espreitar', action: 'passivo', pmCost: 0, uses: null, bookPage: 51 },
  { id: 'ervas-curativas', name: 'Ervas Curativas', action: 'completa', pmCost: 'variavel', uses: null, bookPage: 51 },
  { id: 'impeto', name: 'Ímpeto', action: 'livre', pmCost: 1, uses: null, bookPage: 51 },
  { id: 'inimigo-criatura', name: 'Inimigo de (Criatura)', action: 'passivo', pmCost: 0, uses: null, bookPage: 51 },
  { id: 'olho-do-falcao', name: 'Olho do Falcão', action: 'passivo', pmCost: 0, uses: null, bookPage: 51 },
  { id: 'ponto-fraco', name: 'Ponto Fraco', action: 'passivo', pmCost: 0, uses: null, bookPage: 51 },
]

export const CACADOR_ELECTIVES: readonly CacadorPower[] = Object.freeze(RAW)

export function cacadorElectives(): readonly CacadorPower[] {
  return CACADOR_ELECTIVES
}

export function cacadorPowerById(id: string): CacadorPower | undefined {
  return CACADOR_ELECTIVES.find((p) => p.id === id)
}

export function armadilhaPowers(): readonly CacadorPower[] {
  return CACADOR_ELECTIVES.filter((p) => p.isArmadilha === true)
}

export function activeCacadorPowers(): readonly CacadorPower[] {
  return CACADOR_ELECTIVES.filter((p) => p.action !== 'passivo')
}
