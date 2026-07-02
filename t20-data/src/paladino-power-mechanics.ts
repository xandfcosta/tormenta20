/**
 * Paladino elective power mechanics — action-economy + PM + uses.
 *
 * Complementa `abilities/classes/paladino.ts` (nome + descrição).
 * PDF Cap 1 Paladino p82-85.
 *
 * Julgamentos Divinos (9) ativação livre + PM variado. Auras (5)
 * são passives-while-active — dependem do poder auto "Aura" da classe.
 * Caminhos (2) gated pelo escolha do 5º nível.
 */

export type PaladinoPowerAction =
  | 'padrao'
  | 'movimento'
  | 'livre'
  | 'reacao'
  | 'gratuita'
  | 'completa'
  | 'passivo'
  | 'varia'

export type PaladinoPowerUses = null | 'cena' | 'rodada' | 'dia' | number

export type PaladinoPower = {
  id: string
  name: string
  action: PaladinoPowerAction
  pmCost: number | 'variavel'
  uses: PaladinoPowerUses
  isJulgamento?: boolean
  isVirtude?: boolean
  isAura?: boolean
  isCaminho?: boolean
  bookPage: number
}

const RAW: readonly PaladinoPower[] = [
  { id: 'arma-sagrada', name: 'Arma Sagrada', action: 'passivo', pmCost: 0, uses: null, bookPage: 82 },
  { id: 'aumento-de-atributo', name: 'Aumento de Atributo', action: 'passivo', pmCost: 0, uses: null, bookPage: 82 },
  { id: 'aura-antimagia', name: 'Aura Antimagia', action: 'passivo', pmCost: 0, uses: null, isAura: true, bookPage: 82 },
  { id: 'aura-ardente', name: 'Aura Ardente', action: 'passivo', pmCost: 0, uses: null, isAura: true, bookPage: 82 },
  { id: 'aura-de-cura', name: 'Aura de Cura', action: 'passivo', pmCost: 0, uses: null, isAura: true, bookPage: 83 },
  { id: 'aura-de-invencibilidade', name: 'Aura de Invencibilidade', action: 'passivo', pmCost: 0, uses: 'cena', isAura: true, bookPage: 83 },
  { id: 'aura-poderosa', name: 'Aura Poderosa', action: 'passivo', pmCost: 0, uses: null, isAura: true, bookPage: 83 },
  { id: 'fulgor-divino', name: 'Fulgor Divino', action: 'passivo', pmCost: 0, uses: null, bookPage: 83 },
  { id: 'julgamento-arrependimento', name: 'Julgamento Divino: Arrependimento', action: 'livre', pmCost: 2, uses: null, isJulgamento: true, bookPage: 83 },
  { id: 'julgamento-autoridade', name: 'Julgamento Divino: Autoridade', action: 'livre', pmCost: 1, uses: 'cena', isJulgamento: true, bookPage: 83 },
  { id: 'julgamento-coragem', name: 'Julgamento Divino: Coragem', action: 'livre', pmCost: 2, uses: null, isJulgamento: true, bookPage: 83 },
  { id: 'julgamento-iluminacao', name: 'Julgamento Divino: Iluminação', action: 'livre', pmCost: 2, uses: 'cena', isJulgamento: true, bookPage: 83 },
  { id: 'julgamento-justica', name: 'Julgamento Divino: Justiça', action: 'livre', pmCost: 2, uses: null, isJulgamento: true, bookPage: 83 },
  { id: 'julgamento-libertacao', name: 'Julgamento Divino: Libertação', action: 'livre', pmCost: 5, uses: null, isJulgamento: true, bookPage: 84 },
  { id: 'julgamento-salvacao', name: 'Julgamento Divino: Salvação', action: 'livre', pmCost: 2, uses: null, isJulgamento: true, bookPage: 84 },
  { id: 'julgamento-vindicacao', name: 'Julgamento Divino: Vindicação', action: 'livre', pmCost: 2, uses: null, isJulgamento: true, bookPage: 84 },
  { id: 'julgamento-zelo', name: 'Julgamento Divino: Zelo', action: 'livre', pmCost: 1, uses: null, isJulgamento: true, bookPage: 84 },
  { id: 'orar', name: 'Orar', action: 'varia', pmCost: 'variavel', uses: null, bookPage: 84 },
  { id: 'virtude-caridade', name: 'Virtude Paladinesca: Caridade', action: 'passivo', pmCost: 0, uses: null, isVirtude: true, bookPage: 84 },
  { id: 'virtude-castidade', name: 'Virtude Paladinesca: Castidade', action: 'passivo', pmCost: 0, uses: null, isVirtude: true, bookPage: 84 },
  { id: 'virtude-compaixao', name: 'Virtude Paladinesca: Compaixão', action: 'passivo', pmCost: 0, uses: null, isVirtude: true, bookPage: 84 },
  { id: 'virtude-humildade', name: 'Virtude Paladinesca: Humildade', action: 'completa', pmCost: 0, uses: 'cena', isVirtude: true, bookPage: 84 },
  { id: 'virtude-temperanca', name: 'Virtude Paladinesca: Temperança', action: 'passivo', pmCost: 0, uses: null, isVirtude: true, bookPage: 84 },
  { id: 'caminho-egide-sagrada', name: 'Caminho: Égide Sagrada', action: 'movimento', pmCost: 2, uses: 'cena', isCaminho: true, bookPage: 85 },
  { id: 'caminho-montaria-sagrada', name: 'Caminho: Montaria Sagrada', action: 'movimento', pmCost: 2, uses: null, isCaminho: true, bookPage: 85 },
]

export const PALADINO_ELECTIVES: readonly PaladinoPower[] = Object.freeze(RAW)

export function paladinoElectives(): readonly PaladinoPower[] {
  return PALADINO_ELECTIVES
}

export function paladinoPowerById(id: string): PaladinoPower | undefined {
  return PALADINO_ELECTIVES.find((p) => p.id === id)
}

export function julgamentoPowers(): readonly PaladinoPower[] {
  return PALADINO_ELECTIVES.filter((p) => p.isJulgamento === true)
}

export function virtudePowers(): readonly PaladinoPower[] {
  return PALADINO_ELECTIVES.filter((p) => p.isVirtude === true)
}

export function auraPowers(): readonly PaladinoPower[] {
  return PALADINO_ELECTIVES.filter((p) => p.isAura === true)
}

export function paladinoCaminhoPowers(): readonly PaladinoPower[] {
  return PALADINO_ELECTIVES.filter((p) => p.isCaminho === true)
}

export function activePaladinoPowers(): readonly PaladinoPower[] {
  return PALADINO_ELECTIVES.filter((p) => p.action !== 'passivo')
}
