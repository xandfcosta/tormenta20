/**
 * Bardo elective power mechanics — action-economy + PM + uses
 * per poder eletivo de classe.
 *
 * Complementa `abilities/classes/bardo.ts` (nome + descrição only).
 * PDF Cap 1 Bardo p42-45.
 *
 * Auto powers (Inspiração +N, Magias por círculo, Eclético) já
 * cobertos em `bardo.ts` como progressão automática.
 */

export type BardoPowerAction =
  | 'padrao'
  | 'movimento'
  | 'livre'
  | 'reacao'
  | 'gratuita'
  | 'completa'
  | 'passivo'
  | 'varia'

export type BardoPowerUses = null | 'cena' | 'rodada' | 'dia' | number

export type BardoPower = {
  /** Slug — bate com ID em `bardo.ts` (`class.bardo.<slug>`). */
  id: string
  name: string
  action: BardoPowerAction
  /** PM fixo, ou 'variavel' (Prestidigitação usa custo da magia; Paródia = 1 + custo). */
  pmCost: number | 'variavel'
  uses: BardoPowerUses
  /** Só ativável sob Inspiração (poder auto do Bardo). */
  requiresInspiracao?: boolean
  /** Faz parte da subcategoria "Música" (exige instrumento + Atuação). */
  isMusic?: boolean
  bookPage: number
}

const RAW: readonly BardoPower[] = [
  { id: 'arte-magica', name: 'Arte Mágica', action: 'passivo', pmCost: 0, uses: null, requiresInspiracao: true, bookPage: 44 },
  { id: 'aumentar-repertorio', name: 'Aumentar Repertório', action: 'passivo', pmCost: 0, uses: null, bookPage: 44 },
  { id: 'aumento-de-atributo', name: 'Aumento de Atributo', action: 'passivo', pmCost: 0, uses: null, bookPage: 44 },
  { id: 'danca-das-laminas', name: 'Dança das Lâminas', action: 'livre', pmCost: 1, uses: null, bookPage: 44 },
  { id: 'esgrima-magica', name: 'Esgrima Mágica', action: 'passivo', pmCost: 0, uses: null, requiresInspiracao: true, bookPage: 44 },
  { id: 'estrelato', name: 'Estrelato', action: 'passivo', pmCost: 0, uses: null, bookPage: 44 },
  { id: 'fascinar-em-massa', name: 'Fascinar em Massa', action: 'passivo', pmCost: 2, uses: null, bookPage: 44 },
  { id: 'golpe-elemental', name: 'Golpe Elemental', action: 'livre', pmCost: 1, uses: null, requiresInspiracao: true, bookPage: 44 },
  { id: 'golpe-magico', name: 'Golpe Mágico', action: 'passivo', pmCost: 0, uses: null, requiresInspiracao: true, bookPage: 44 },
  { id: 'inspiracao-marcial', name: 'Inspiração Marcial', action: 'passivo', pmCost: 0, uses: null, bookPage: 44 },
  { id: 'lendas-e-historias', name: 'Lendas e Histórias', action: 'livre', pmCost: 1, uses: null, bookPage: 44 },
  { id: 'manipular', name: 'Manipular', action: 'padrao', pmCost: 1, uses: null, bookPage: 45 },
  { id: 'manipular-em-massa', name: 'Manipular em Massa', action: 'passivo', pmCost: 2, uses: null, bookPage: 45 },
  { id: 'mestre-dos-sussurros', name: 'Mestre dos Sussurros', action: 'passivo', pmCost: 0, uses: null, bookPage: 45 },
  { id: 'musica-balada-fascinante', name: 'Música: Balada Fascinante', action: 'padrao', pmCost: 1, uses: null, isMusic: true, bookPage: 45 },
  { id: 'musica-cancao-assustadora', name: 'Música: Canção Assustadora', action: 'padrao', pmCost: 1, uses: null, isMusic: true, bookPage: 45 },
  { id: 'musica-melodia-curativa', name: 'Música: Melodia Curativa', action: 'padrao', pmCost: 1, uses: null, isMusic: true, bookPage: 45 },
  { id: 'musica-melodia-restauradora', name: 'Música: Melodia Restauradora', action: 'passivo', pmCost: 2, uses: null, isMusic: true, bookPage: 45 },
  { id: 'parodia', name: 'Paródia', action: 'reacao', pmCost: 'variavel', uses: 'rodada', bookPage: 45 },
  { id: 'prestidigitacao', name: 'Prestidigitação', action: 'padrao', pmCost: 'variavel', uses: null, bookPage: 45 },
]

export const BARDO_ELECTIVES: readonly BardoPower[] = Object.freeze(RAW)

/** Retorna todos os eletivos de Bardo. */
export function bardoElectives(): readonly BardoPower[] {
  return BARDO_ELECTIVES
}

/** Retorna eletivo por slug id. */
export function bardoPowerById(id: string): BardoPower | undefined {
  return BARDO_ELECTIVES.find((p) => p.id === id)
}

/** Filtra poderes que exigem Inspiração ativa. */
export function inspiracaoOnlyBardoPowers(): readonly BardoPower[] {
  return BARDO_ELECTIVES.filter((p) => p.requiresInspiracao === true)
}

/** Filtra poderes de Música (exigem instrumento + Atuação). */
export function musicBardoPowers(): readonly BardoPower[] {
  return BARDO_ELECTIVES.filter((p) => p.isMusic === true)
}

/** Filtra só poderes ativos (não-passivos). */
export function activeBardoPowers(): readonly BardoPower[] {
  return BARDO_ELECTIVES.filter((p) => p.action !== 'passivo')
}
