/**
 * Bárbaro elective power mechanics — action-economy + PM + uses
 * per poder eletivo de classe.
 *
 * Complementa `abilities/classes/barbaro.ts` (que só carrega nome +
 * descrição). PDF Cap 1 Bárbaro p41-42.
 *
 * Auto powers (Fúria +N, RD tiers, Instinto Selvagem) já resolvidos
 * via `damage-reduction.barbaroRdForLevel` + `furiaMods` (nos
 * modifiers da classe). Este módulo cobre ELETIVOS.
 */

export type BarbaroPowerAction =
  | 'padrao'
  | 'movimento'
  | 'livre'
  | 'reacao'
  | 'gratuita'
  | 'completa'
  | 'passivo'
  | 'varia'

export type BarbaroPowerUses = null | 'cena' | 'rodada' | 'dia' | number

export type BarbaroPower = {
  /** Slug do poder — bate com IDs em `barbaro.ts` (`class.barbaro.<slug>`). */
  id: string
  name: string
  action: BarbaroPowerAction
  /** PM fixo, ou 'variavel' quando dependente (Vigor Primal). */
  pmCost: number | 'variavel'
  uses: BarbaroPowerUses
  /** Requer estar em Fúria pra ativar (true por padrão só nos que exigem). */
  requiresFuria?: boolean
  bookPage: number
}

const RAW: readonly BarbaroPower[] = [
  { id: 'alma-de-bronze', name: 'Alma de Bronze', action: 'passivo', pmCost: 0, uses: null, bookPage: 41 },
  { id: 'aumento-de-atributo', name: 'Aumento de Atributo', action: 'passivo', pmCost: 0, uses: null, bookPage: 41 },
  { id: 'brado-assustador', name: 'Brado Assustador', action: 'movimento', pmCost: 1, uses: 'cena', bookPage: 41 },
  { id: 'critico-brutal', name: 'Crítico Brutal', action: 'passivo', pmCost: 0, uses: null, bookPage: 41 },
  { id: 'destruidor', name: 'Destruidor', action: 'passivo', pmCost: 0, uses: null, bookPage: 41 },
  { id: 'espirito-inquebravel', name: 'Espírito Inquebrável', action: 'passivo', pmCost: 0, uses: null, requiresFuria: true, bookPage: 41 },
  { id: 'esquiva-sobrenatural', name: 'Esquiva Sobrenatural', action: 'passivo', pmCost: 0, uses: null, bookPage: 41 },
  { id: 'forca-indomavel', name: 'Força Indomável', action: 'livre', pmCost: 1, uses: null, bookPage: 41 },
  { id: 'frenesi', name: 'Frenesi', action: 'livre', pmCost: 2, uses: 'rodada', requiresFuria: true, bookPage: 41 },
  { id: 'furia-da-savana', name: 'Fúria da Savana', action: 'passivo', pmCost: 0, uses: null, bookPage: 42 },
  { id: 'furia-raivosa', name: 'Fúria Raivosa', action: 'livre', pmCost: 1, uses: null, requiresFuria: true, bookPage: 42 },
  { id: 'golpe-poderoso', name: 'Golpe Poderoso', action: 'livre', pmCost: 1, uses: null, bookPage: 42 },
  { id: 'impeto', name: 'Ímpeto', action: 'livre', pmCost: 1, uses: null, bookPage: 42 },
  { id: 'investida-imprudente', name: 'Investida Imprudente', action: 'passivo', pmCost: 0, uses: null, bookPage: 42 },
  { id: 'pele-de-aco', name: 'Pele de Aço', action: 'passivo', pmCost: 0, uses: null, bookPage: 42 },
  { id: 'pele-de-ferro', name: 'Pele de Ferro', action: 'passivo', pmCost: 0, uses: null, bookPage: 42 },
  { id: 'sangue-dos-inimigos', name: 'Sangue dos Inimigos', action: 'passivo', pmCost: 0, uses: null, requiresFuria: true, bookPage: 42 },
  { id: 'supersticao', name: 'Superstição', action: 'passivo', pmCost: 0, uses: null, bookPage: 42 },
  { id: 'totem-espiritual', name: 'Totem Espiritual', action: 'varia', pmCost: 'variavel', uses: null, bookPage: 42 },
  { id: 'vigor-primal', name: 'Vigor Primal', action: 'movimento', pmCost: 'variavel', uses: null, bookPage: 42 },
]

export const BARBARO_ELECTIVES: readonly BarbaroPower[] = Object.freeze(RAW)

/** Retorna todos os eletivos de Bárbaro. */
export function barbaroElectives(): readonly BarbaroPower[] {
  return BARBARO_ELECTIVES
}

/** Retorna eletivo por slug id. */
export function barbaroPowerById(id: string): BarbaroPower | undefined {
  return BARBARO_ELECTIVES.find((p) => p.id === id)
}

/** Filtra poderes que exigem estar em Fúria pra ativar. */
export function furiaOnlyPowers(): readonly BarbaroPower[] {
  return BARBARO_ELECTIVES.filter((p) => p.requiresFuria === true)
}

/** Filtra só poderes ativos (não-passivos). */
export function activeBarbaroPowers(): readonly BarbaroPower[] {
  return BARBARO_ELECTIVES.filter((p) => p.action !== 'passivo')
}
