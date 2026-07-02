/**
 * Inventor elective power mechanics — action-economy + PM + uses.
 *
 * Complementa `abilities/classes/inventor.ts` (nome + descrição).
 * PDF Cap 1 Inventor p68-71.
 *
 * Muitos poderes são passives de crafting (Armeiro, Couraceiro,
 * Engenhoqueiro, etc). Modificadores tipo "gasta +N PM ao usar
 * item" são passives com pmCost > 0 (aplicam quando ação disparadora
 * ocorre — combat engine resolve).
 */

export type InventorPowerAction =
  | 'padrao'
  | 'movimento'
  | 'livre'
  | 'reacao'
  | 'gratuita'
  | 'completa'
  | 'passivo'
  | 'varia'

export type InventorPowerUses = null | 'cena' | 'rodada' | 'dia' | number

export type InventorPower = {
  id: string
  name: string
  action: InventorPowerAction
  pmCost: number | 'variavel'
  uses: InventorPowerUses
  bookPage: number
}

const RAW: readonly InventorPower[] = [
  { id: 'agite-antes-de-usar', name: 'Agite Antes de Usar', action: 'passivo', pmCost: 'variavel', uses: null, bookPage: 68 },
  { id: 'ajuste-de-mira', name: 'Ajuste de Mira', action: 'padrao', pmCost: 'variavel', uses: 'cena', bookPage: 68 },
  { id: 'alquimista-de-batalha', name: 'Alquimista de Batalha', action: 'passivo', pmCost: 0, uses: null, bookPage: 68 },
  { id: 'alquimista-iniciado', name: 'Alquimista Iniciado', action: 'passivo', pmCost: 0, uses: null, bookPage: 69 },
  { id: 'armeiro', name: 'Armeiro', action: 'passivo', pmCost: 0, uses: null, bookPage: 69 },
  { id: 'ativacao-rapida', name: 'Ativação Rápida', action: 'passivo', pmCost: 2, uses: 'rodada', bookPage: 69 },
  { id: 'aumento-de-atributo', name: 'Aumento de Atributo', action: 'passivo', pmCost: 0, uses: null, bookPage: 69 },
  { id: 'automato', name: 'Autômato', action: 'passivo', pmCost: 0, uses: null, bookPage: 69 },
  { id: 'automato-prototipado', name: 'Autômato Prototipado', action: 'padrao', pmCost: 2, uses: null, bookPage: 69 },
  { id: 'balistica', name: 'Balística', action: 'passivo', pmCost: 0, uses: null, bookPage: 69 },
  { id: 'blindagem', name: 'Blindagem', action: 'passivo', pmCost: 0, uses: null, bookPage: 69 },
  { id: 'cano-raiado', name: 'Cano Raiado', action: 'passivo', pmCost: 0, uses: null, bookPage: 69 },
  { id: 'catalisador-instavel', name: 'Catalisador Instável', action: 'completa', pmCost: 3, uses: null, bookPage: 69 },
  { id: 'chutes-e-palavroes', name: 'Chutes e Palavrões', action: 'livre', pmCost: 1, uses: 'rodada', bookPage: 70 },
  { id: 'conhecimento-de-formulas', name: 'Conhecimento de Fórmulas', action: 'passivo', pmCost: 0, uses: null, bookPage: 70 },
  { id: 'couraceiro', name: 'Couraceiro', action: 'passivo', pmCost: 0, uses: null, bookPage: 70 },
  { id: 'engenhoqueiro', name: 'Engenhoqueiro', action: 'passivo', pmCost: 0, uses: null, bookPage: 70 },
  { id: 'farmaceutico', name: 'Farmacêutico', action: 'passivo', pmCost: 'variavel', uses: null, bookPage: 70 },
  { id: 'ferreiro', name: 'Ferreiro', action: 'passivo', pmCost: 0, uses: null, bookPage: 70 },
  { id: 'granadeiro', name: 'Granadeiro', action: 'passivo', pmCost: 0, uses: null, bookPage: 70 },
  { id: 'homunculo', name: 'Homúnculo', action: 'passivo', pmCost: 0, uses: 'cena', bookPage: 70 },
  { id: 'invencao-potente', name: 'Invenção Potente', action: 'passivo', pmCost: 1, uses: null, bookPage: 70 },
  { id: 'maestria-em-pericia', name: 'Maestria em Perícia', action: 'livre', pmCost: 1, uses: null, bookPage: 70 },
  { id: 'manutencao-eficiente', name: 'Manutenção Eficiente', action: 'passivo', pmCost: 0, uses: null, bookPage: 70 },
  { id: 'mestre-alquimista', name: 'Mestre Alquimista', action: 'passivo', pmCost: 0, uses: null, bookPage: 71 },
  { id: 'mestre-cuca', name: 'Mestre Cuca', action: 'passivo', pmCost: 0, uses: null, bookPage: 71 },
  { id: 'mistura-fervilhante', name: 'Mistura Fervilhante', action: 'passivo', pmCost: 2, uses: null, bookPage: 71 },
  { id: 'oficina-de-campo', name: 'Oficina de Campo', action: 'varia', pmCost: 2, uses: 'dia', bookPage: 71 },
  { id: 'pedra-de-amolar', name: 'Pedra de Amolar', action: 'movimento', pmCost: 'variavel', uses: 'cena', bookPage: 71 },
  { id: 'sintese-rapida', name: 'Síntese Rápida', action: 'passivo', pmCost: 0, uses: null, bookPage: 71 },
]

export const INVENTOR_ELECTIVES: readonly InventorPower[] = Object.freeze(RAW)

export function inventorElectives(): readonly InventorPower[] {
  return INVENTOR_ELECTIVES
}

export function inventorPowerById(id: string): InventorPower | undefined {
  return INVENTOR_ELECTIVES.find((p) => p.id === id)
}

export function activeInventorPowers(): readonly InventorPower[] {
  return INVENTOR_ELECTIVES.filter((p) => p.action !== 'passivo')
}
