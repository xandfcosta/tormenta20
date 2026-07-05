/**
 * Tipos de Comunidades — PDF Cap 6 p271-273. Quatro patamares narrativos
 * de assentamento (aldeia → vila → cidade → metrópole). Encoda o que é
 * mecânico: população, guarda em dados, teto de preço de item disponível
 * ("Economia"), dinheiro em cofre da comunidade (dado × T$) e grau de
 * formalidade de governo/lei/justiça.
 *
 * Prosa restante (advertências narrativas, exemplos de Valkaria/Tiberus,
 * "Lei & Ordem" p273) fica no livro — não é regra encodable.
 *
 * População: livro apresenta como "até N habitantes" (aldeia/vila/cidade)
 * e "por volta de 100k, maiores metrópoles > 1M" (metrópole). Guarda-se
 * `maxPopulation` como número; metrópole é `null` (sem teto formal).
 */

export const COMMUNITY_TIERS = [
  'aldeia',
  'vila',
  'cidade',
  'metropole',
] as const

export type CommunityTier = (typeof COMMUNITY_TIERS)[number]

export type LawFormality = 'nenhuma' | 'simples' | 'complexa'

export type CommunityRow = {
  tier: CommunityTier
  label: string
  /** Teto de habitantes por PDF. `null` = sem teto (metrópole). */
  maxPopulation: number | null
  /**
   * Guarda formal. Aldeia: `null` (sem guarda formal — camponeses
   * improvisam; magistrado, se houver, tem 1d4+1 guardas — codificado
   * em `magistrateGuards`). Metrópole: `'exercito'` (livro descreve
   * como "exército" sem dado; sem teto encodable).
   */
  guardForce: {
    kind: 'nenhuma' | 'milicia' | 'formal' | 'exercito'
    dice: string | null
    leaderLabel: string | null
  }
  /**
   * Guardas do magistrado local, apenas para aldeia (livro p271: "1d4+1
   * guardas" caso a aldeia tenha um magistrado apontado por nobre).
   */
  magistrateGuards: string | null
  /** Formalidade de governo (bookkeeping narrativo). */
  government: string
  /** Formalidade de lei/justiça (nenhuma/simples/complexa). */
  lawFormality: LawFormality
  /**
   * Teto de preço em T$ para itens disponíveis no mercado local.
   * `null` = qualquer item mundano (metrópole). Itens raros / superiores
   * podem existir em quantidade limitada mesmo abaixo do teto — decisão
   * de GM.
   */
  itemPriceCapTS: number | null
  /**
   * Dinheiro disponível em cofre da comunidade, em dados × T$. `null`
   * para aldeia (sem cofre formal) e metrópole (virtualmente ilimitado).
   */
  cashOnHand: { dice: string; multiplierTS: number } | null
  bookPage: number
}

export const COMMUNITY_TABLE: readonly CommunityRow[] = Object.freeze([
  {
    tier: 'aldeia',
    label: 'Aldeia',
    maxPopulation: 1000,
    guardForce: {
      kind: 'nenhuma',
      dice: '2d10',
      leaderLabel: 'camponeses armados com ferramentas',
    },
    magistrateGuards: '1d4+1',
    government:
      'nenhum formal; sábio ancião ou magistrado apontado por nobre local',
    lawFormality: 'nenhuma',
    itemPriceCapTS: 0,
    cashOnHand: null,
    bookPage: 271,
  },
  {
    tier: 'vila',
    label: 'Vila',
    maxPopulation: 5000,
    guardForce: {
      kind: 'milicia',
      dice: '10d10',
      leaderLabel: 'sargento',
    },
    magistrateGuards: null,
    government: 'burgomestre eleito ou apontado por nobre local',
    lawFormality: 'simples',
    itemPriceCapTS: 1000,
    cashOnHand: { dice: '1d6', multiplierTS: 1000 },
    bookPage: 271,
  },
  {
    tier: 'cidade',
    label: 'Cidade',
    maxPopulation: 25000,
    guardForce: {
      kind: 'formal',
      dice: null,
      leaderLabel: 'capitão (cavaleiro ou guerreiro de pelo menos 8º nível)',
    },
    magistrateGuards: null,
    government:
      'lorde prefeito apontado pelo regente do reino, com conselho eleito',
    lawFormality: 'complexa',
    itemPriceCapTS: 10000,
    cashOnHand: { dice: '2d4', multiplierTS: 10000 },
    bookPage: 272,
  },
  {
    tier: 'metropole',
    label: 'Metrópole',
    maxPopulation: null,
    guardForce: {
      kind: 'exercito',
      dice: null,
      leaderLabel: 'oficiais + clérigos + arcanistas + construtos',
    },
    magistrateGuards: null,
    government: 'o próprio regente do reino, delegando a oficiais e conselheiros',
    lawFormality: 'complexa',
    itemPriceCapTS: null,
    cashOnHand: null,
    bookPage: 272,
  },
])

/**
 * Retorna a linha da tabela para o dado patamar. Lança se `tier` não
 * pertencer ao enum (chamador deve ter tipado corretamente).
 */
export function communityRow(tier: CommunityTier): CommunityRow {
  const row = COMMUNITY_TABLE.find((r) => r.tier === tier)
  if (!row) throw new Error(`Unknown CommunityTier: ${tier}`)
  return row
}

/**
 * Menor patamar cuja economia comporta um item de preço `priceTS`.
 * Retorna `null` se o item for tão caro que só a metrópole (sem teto)
 * consegue — o chamador decide se rejeita ou faz venda.
 *
 * Aldeia (teto 0) nunca serve nada; vila cobre até T$ 1.000; cidade
 * até T$ 10.000; metrópole cobre qualquer item mundano.
 */
export function minCommunityForItemPrice(
  priceTS: number,
): CommunityTier | null {
  if (priceTS < 0) throw new Error(`priceTS must be >= 0, got ${priceTS}`)
  for (const row of COMMUNITY_TABLE) {
    if (row.itemPriceCapTS === null) return row.tier
    if (priceTS <= row.itemPriceCapTS && row.itemPriceCapTS > 0) return row.tier
  }
  return null
}
