/**
 * Origem poder-único mechanics — action-economy + PM metadata pra
 * cada poder exclusivo de origem.
 *
 * Complementa `origens.ts` (que só carrega `poderUnico: string`).
 * PDF Cap 1 Origens p85-95. Páginas alinhadas com `origens.ts`.
 *
 * 35 origens × 1 poder único = 35 entries. Maioria passiva.
 */

export type OrigemPowerAction =
  | 'padrao'
  | 'movimento'
  | 'livre'
  | 'reacao'
  | 'gratuita'
  | 'completa'
  | 'passivo'
  | 'varia'

export type OrigemPowerUses = null | 'cena' | 'rodada' | 'dia' | number

export type OrigemPower = {
  origemId: string
  name: string
  action: OrigemPowerAction
  pmCost: number | 'variavel'
  uses: OrigemPowerUses
  bookPage: number
}

const RAW: readonly OrigemPower[] = [
  { origemId: 'acolito', name: 'Membro da Igreja', action: 'passivo', pmCost: 0, uses: null, bookPage: 85 },
  { origemId: 'amigo-dos-animais', name: 'Amigo Especial', action: 'passivo', pmCost: 0, uses: null, bookPage: 85 },
  { origemId: 'amnesico', name: 'Lembranças Graduais', action: 'passivo', pmCost: 0, uses: null, bookPage: 86 },
  { origemId: 'aristocrata', name: 'Sangue Azul', action: 'passivo', pmCost: 0, uses: null, bookPage: 86 },
  { origemId: 'artesao', name: 'Frutos do Trabalho', action: 'passivo', pmCost: 0, uses: null, bookPage: 86 },
  { origemId: 'artista', name: 'Dom Artístico', action: 'passivo', pmCost: 0, uses: null, bookPage: 87 },
  { origemId: 'assistente-de-laboratorio', name: 'Esse Cheiro...', action: 'passivo', pmCost: 0, uses: null, bookPage: 87 },
  { origemId: 'batedor', name: 'À Prova de Tudo', action: 'passivo', pmCost: 0, uses: null, bookPage: 88 },
  { origemId: 'capanga', name: 'Confissão', action: 'passivo', pmCost: 0, uses: null, bookPage: 88 },
  { origemId: 'charlatao', name: 'Alpinista Social', action: 'passivo', pmCost: 0, uses: null, bookPage: 88 },
  { origemId: 'circense', name: 'Truque de Mágica', action: 'varia', pmCost: 'variavel', uses: null, bookPage: 89 },
  { origemId: 'criminoso', name: 'Punguista', action: 'passivo', pmCost: 0, uses: 'dia', bookPage: 89 },
  { origemId: 'curandeiro', name: 'Médico de Campo', action: 'passivo', pmCost: 0, uses: null, bookPage: 89 },
  { origemId: 'eremita', name: 'Busca Interior', action: 'completa', pmCost: 1, uses: null, bookPage: 89 },
  { origemId: 'escravo', name: 'Desejo de Liberdade', action: 'passivo', pmCost: 0, uses: null, bookPage: 89 },
  { origemId: 'estudioso', name: 'Palpite Fundamentado', action: 'livre', pmCost: 2, uses: null, bookPage: 90 },
  { origemId: 'fazendeiro', name: 'Água no Feijão', action: 'passivo', pmCost: 0, uses: null, bookPage: 90 },
  { origemId: 'forasteiro', name: 'Cultura Exótica', action: 'livre', pmCost: 1, uses: null, bookPage: 90 },
  { origemId: 'gladiador', name: 'Pão e Circo', action: 'passivo', pmCost: 0, uses: null, bookPage: 90 },
  { origemId: 'guarda', name: 'Detetive', action: 'livre', pmCost: 1, uses: 'cena', bookPage: 91 },
  { origemId: 'herdeiro', name: 'Herança', action: 'passivo', pmCost: 0, uses: null, bookPage: 91 },
  { origemId: 'heroi-camponês', name: 'Coração Heroico', action: 'passivo', pmCost: 0, uses: null, bookPage: 91 },
  { origemId: 'marujo', name: 'Passagem de Navio', action: 'passivo', pmCost: 0, uses: null, bookPage: 92 },
  { origemId: 'mateiro', name: 'Vendedor de Carcaças', action: 'passivo', pmCost: 0, uses: null, bookPage: 92 },
  { origemId: 'membro-de-guilda', name: 'Rede de Contatos', action: 'passivo', pmCost: 0, uses: null, bookPage: 92 },
  { origemId: 'mercador', name: 'Negociação', action: 'passivo', pmCost: 0, uses: null, bookPage: 93 },
  { origemId: 'minerador', name: 'Escavador', action: 'passivo', pmCost: 0, uses: null, bookPage: 93 },
  { origemId: 'nomade', name: 'Mochileiro', action: 'passivo', pmCost: 0, uses: null, bookPage: 93 },
  { origemId: 'pivete', name: 'Quebra-Galho', action: 'passivo', pmCost: 0, uses: null, bookPage: 93 },
  { origemId: 'refugiado', name: 'Estoico', action: 'passivo', pmCost: 0, uses: null, bookPage: 93 },
  { origemId: 'seguidor', name: 'Antigo Mestre', action: 'passivo', pmCost: 0, uses: 'cena', bookPage: 94 },
  { origemId: 'selvagem', name: 'Vida Rústica', action: 'passivo', pmCost: 0, uses: null, bookPage: 94 },
  { origemId: 'soldado', name: 'Influência Militar', action: 'passivo', pmCost: 0, uses: null, bookPage: 94 },
  { origemId: 'taverneiro', name: 'Gororoba', action: 'passivo', pmCost: 0, uses: null, bookPage: 94 },
  { origemId: 'trabalhador', name: 'Atlético', action: 'passivo', pmCost: 0, uses: null, bookPage: 95 },
]

export const ORIGEM_POWERS: readonly OrigemPower[] = Object.freeze(RAW)

/** Poder único de uma origem específica. */
export function origemPowerOf(origemId: string): OrigemPower | undefined {
  return ORIGEM_POWERS.find((p) => p.origemId === origemId)
}

/** Só os poderes ativos (excluindo passivos). */
export function activeOrigemPowers(): readonly OrigemPower[] {
  return ORIGEM_POWERS.filter((p) => p.action !== 'passivo')
}

/** Só poderes que consomem PM. */
export function pmConsumingOrigemPowers(): readonly OrigemPower[] {
  return ORIGEM_POWERS.filter((p) => p.pmCost !== 0)
}
