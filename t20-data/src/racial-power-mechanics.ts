/**
 * Racial power mechanics — supplementary catalog anotando ação,
 * custo em PM e limite de uso por poder racial ATIVO.
 *
 * Complementa `racas.ts` (que só carrega `name` + `summary`).
 * PDF Cap 1 (Construção de Personagem), Raças p19-31.
 *
 * Cobre apenas poderes com custo/ativação (skip stat mods puros e
 * poderes passivos).
 */

/** Ação necessária pra ativar. */
export type RacialPowerAction =
  | 'padrao'
  | 'movimento'
  | 'livre'
  | 'reacao'
  | 'gratuita'
  | 'completa'
  | 'varia'

/**
 * Limite de uso:
 *  - null = ilimitado (só limitado por PM)
 *  - 'cena' = 1 por cena
 *  - 'rodada' = 1 por rodada (não empilha em turno)
 *  - number = uses per dia
 */
export type RacialPowerUses = null | 'cena' | 'rodada' | number

export type RacialPower = {
  racaId: string
  name: string
  action: RacialPowerAction
  /** PM fixo, ou 'variavel' quando depende da magia lançada. */
  pmCost: number | 'variavel'
  uses: RacialPowerUses
  bookPage: number
}

const POWERS: readonly RacialPower[] = [
  // Dahllan (p21)
  {
    racaId: 'dahllan',
    name: 'Amiga das Plantas',
    action: 'padrao',
    pmCost: 3,
    uses: null,
    bookPage: 21,
  },
  {
    racaId: 'dahllan',
    name: 'Armadura de Allihanna',
    action: 'movimento',
    pmCost: 1,
    uses: 'cena',
    bookPage: 21,
  },

  // Minotauro (p25)
  {
    racaId: 'minotauro',
    name: 'Chifres',
    action: 'livre',
    pmCost: 1,
    uses: 'rodada',
    bookPage: 25,
  },

  // Qareen (p26)
  {
    racaId: 'qareen',
    name: 'Tatuagem Mística',
    action: 'padrao',
    pmCost: 1,
    uses: null,
    bookPage: 26,
  },

  // Hynne (p27-28)
  {
    racaId: 'hynne',
    name: 'Sorte Salvadora',
    action: 'reacao',
    pmCost: 1,
    uses: null,
    bookPage: 27,
  },

  // Kliren (p28)
  {
    racaId: 'kliren',
    name: 'Engenhosidade',
    action: 'livre',
    pmCost: 2,
    uses: null,
    bookPage: 28,
  },

  // Medusa (p28-29)
  {
    racaId: 'medusa',
    name: 'Natureza Venenosa',
    action: 'movimento',
    pmCost: 1,
    uses: null,
    bookPage: 28,
  },
  {
    racaId: 'medusa',
    name: 'Olhar Atordoante',
    action: 'movimento',
    pmCost: 1,
    uses: 'cena',
    bookPage: 28,
  },

  // Sereia/Tritão (p29-30)
  {
    racaId: 'sereia-tritao',
    name: 'Canção dos Mares',
    action: 'padrao',
    pmCost: 'variavel',
    uses: null,
    bookPage: 29,
  },

  // Sílfide (p30)
  {
    racaId: 'silfide',
    name: 'Asas de Borboleta',
    action: 'livre',
    pmCost: 1,
    uses: 'rodada',
    bookPage: 30,
  },
  {
    racaId: 'silfide',
    name: 'Magia das Fadas',
    action: 'padrao',
    pmCost: 'variavel',
    uses: null,
    bookPage: 30,
  },

  // Suraggel (p30-31)
  {
    racaId: 'suraggel',
    name: 'Luz Sagrada',
    action: 'padrao',
    pmCost: 1,
    uses: null,
    bookPage: 30,
  },
  {
    racaId: 'suraggel',
    name: 'Sombras Profanas',
    action: 'padrao',
    pmCost: 1,
    uses: null,
    bookPage: 30,
  },

  // Trog (p31)
  {
    racaId: 'trog',
    name: 'Mau Cheiro',
    action: 'padrao',
    pmCost: 2,
    uses: null,
    bookPage: 31,
  },
  {
    racaId: 'trog',
    name: 'Mordida',
    action: 'livre',
    pmCost: 1,
    uses: 'rodada',
    bookPage: 31,
  },
]

export const RACIAL_POWERS: readonly RacialPower[] = Object.freeze(POWERS)

/** Poderes ativos de uma raça específica. */
export function racialPowersOf(racaId: string): readonly RacialPower[] {
  return RACIAL_POWERS.filter((p) => p.racaId === racaId)
}

/** Poderes que consomem PM (fixo ou variável). */
export function pmConsumingPowers(): readonly RacialPower[] {
  return RACIAL_POWERS.filter((p) => p.pmCost !== 0)
}

/** Total de PM fixo pra combo completa (ignora 'variavel'). */
export function totalFixedPmCost(powers: readonly RacialPower[]): number {
  return powers.reduce((n, p) => (p.pmCost === 'variavel' ? n : n + p.pmCost), 0)
}
