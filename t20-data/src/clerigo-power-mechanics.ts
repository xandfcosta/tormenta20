/**
 * Clérigo elective power mechanics — action-economy + PM + uses.
 *
 * Complementa `abilities/classes/clerigo.ts` (nome + descrição).
 * PDF Cap 1 Clérigo p56-58.
 *
 * Missas (5) são cerimônias de 1 hora — modeladas como `varia` action
 * pmCost 1 (base). Modificadores de magia (Comunhão Vital, Prece de
 * Combate, Magia Sagrada/Profana, Canalizar Amplo) são passivos que
 * empilham custo extra na magia base.
 */

export type ClerigoPowerAction =
  | 'padrao'
  | 'movimento'
  | 'livre'
  | 'reacao'
  | 'gratuita'
  | 'completa'
  | 'passivo'
  | 'varia'

export type ClerigoPowerUses = null | 'cena' | 'rodada' | 'dia' | number

export type ClerigoPower = {
  id: string
  name: string
  action: ClerigoPowerAction
  pmCost: number | 'variavel'
  uses: ClerigoPowerUses
  /** True quando é uma Missa (cerimônia de 1h). */
  isMissa?: boolean
  bookPage: number
}

const RAW: readonly ClerigoPower[] = [
  { id: 'abencoar-arma', name: 'Abençoar Arma', action: 'movimento', pmCost: 3, uses: null, bookPage: 57 },
  { id: 'aumento-de-atributo', name: 'Aumento de Atributo', action: 'passivo', pmCost: 0, uses: null, bookPage: 57 },
  { id: 'autoridade-eclesiastica', name: 'Autoridade Eclesiástica', action: 'passivo', pmCost: 0, uses: null, bookPage: 57 },
  { id: 'canalizar-energia-positiva-negativa', name: 'Canalizar Energia Positiva/Negativa', action: 'padrao', pmCost: 'variavel', uses: null, bookPage: 57 },
  { id: 'canalizar-amplo', name: 'Canalizar Amplo', action: 'passivo', pmCost: 2, uses: null, bookPage: 57 },
  { id: 'comunhao-vital', name: 'Comunhão Vital', action: 'passivo', pmCost: 2, uses: null, bookPage: 57 },
  { id: 'conhecimento-magico', name: 'Conhecimento Mágico', action: 'passivo', pmCost: 0, uses: null, bookPage: 57 },
  { id: 'expulsar-comandar-mortos-vivos', name: 'Expulsar/Comandar Mortos-Vivos', action: 'padrao', pmCost: 3, uses: null, bookPage: 57 },
  { id: 'liturgia-magica', name: 'Liturgia Mágica', action: 'movimento', pmCost: 0, uses: null, bookPage: 57 },
  { id: 'magia-sagrada-profana', name: 'Magia Sagrada/Profana', action: 'passivo', pmCost: 1, uses: null, bookPage: 57 },
  { id: 'mestre-celebrante', name: 'Mestre Celebrante', action: 'passivo', pmCost: 0, uses: null, bookPage: 57 },
  { id: 'missa-bencao-da-vida', name: 'Missa: Bênção da Vida', action: 'varia', pmCost: 1, uses: null, isMissa: true, bookPage: 58 },
  { id: 'missa-chamado-as-armas', name: 'Missa: Chamado às Armas', action: 'varia', pmCost: 1, uses: null, isMissa: true, bookPage: 58 },
  { id: 'missa-elevacao-do-espirito', name: 'Missa: Elevação do Espírito', action: 'varia', pmCost: 1, uses: null, isMissa: true, bookPage: 58 },
  { id: 'missa-escudo-divino', name: 'Missa: Escudo Divino', action: 'varia', pmCost: 1, uses: null, isMissa: true, bookPage: 58 },
  { id: 'missa-superar-as-limitacoes', name: 'Missa: Superar as Limitações', action: 'varia', pmCost: 1, uses: null, isMissa: true, bookPage: 58 },
  { id: 'prece-de-combate', name: 'Prece de Combate', action: 'passivo', pmCost: 2, uses: null, bookPage: 58 },
  { id: 'simbolo-sagrado-energizado', name: 'Símbolo Sagrado Energizado', action: 'movimento', pmCost: 1, uses: 'cena', bookPage: 58 },
]

export const CLERIGO_ELECTIVES: readonly ClerigoPower[] = Object.freeze(RAW)

export function clerigoElectives(): readonly ClerigoPower[] {
  return CLERIGO_ELECTIVES
}

export function clerigoPowerById(id: string): ClerigoPower | undefined {
  return CLERIGO_ELECTIVES.find((p) => p.id === id)
}

/** Retorna as 5 Missas. */
export function missaPowers(): readonly ClerigoPower[] {
  return CLERIGO_ELECTIVES.filter((p) => p.isMissa === true)
}

export function activeClerigoPowers(): readonly ClerigoPower[] {
  return CLERIGO_ELECTIVES.filter((p) => p.action !== 'passivo')
}
