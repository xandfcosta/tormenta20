/**
 * Magic items seed catalog — PDF book Cap 8 — Recompensas
 * (book p329-349, PDF ~p335-355).
 *
 * Curated seed (13 entries), not the full Tesouros chapter. Coverage
 * goal: every item kind (potion / scroll / staff / weapon-enchant /
 * armor-enchant / ring / wondrous / artifact) appears at least once,
 * across the three Tormenta 20 tiers (menor / médio / maior) plus
 * artefato.
 *
 * T20-specific contracts pinned here:
 *  - **No sintonização/attunement.** Items work by being worn or wielded.
 *    Non-stacking rule comes from bonus-type collision (item / armor /
 *    enhancement), already handled by `BonusType` in items/types.ts.
 *  - **Tiers are menor / médio / maior + artefato** (book p329 Tabela
 *    8-1), NOT the D&D-style 5-tier ladder.
 *  - **Weapon/armor enchants are discrete named effects** (Flamejante,
 *    Formidável, Defensor, Magnífica). There is no generic "+1 weapon".
 *  - **Wands are NOT charged at 50 like D&D.** T20's `varinha arcana`
 *    in Cap 5 is a mundane esoteric focus. Charged spell items are
 *    pergaminhos (1 charge, consumed) and cajados (continuous, no
 *    charges — they discount PM cost instead).
 *  - **Activation by PM**, not by daily charges. Items that need PM to
 *    fire list `pmActivationCost`. `usesPerDay` is only set for the
 *    rare items the book caps at 1/day.
 */

export type MagicItemKind =
  | 'potion'
  | 'scroll'
  | 'staff'
  | 'weapon-enchant'
  | 'armor-enchant'
  | 'ring'
  | 'wondrous'
  | 'artifact'

/** PDF p329 Tabela 8-1. T20 native tier ladder. */
export type MagicItemTier = 'menor' | 'medio' | 'maior' | 'artefato'

/**
 * Slot hint for rendering. T20 doesn't formally tag accessory body
 * regions — these labels derive from item descriptions. `null` means
 * the item is a free-form modifier (encantamento applied to a base
 * arma/armadura) or an artefato whose slot is dictated by its base
 * form.
 */
export type MagicItemSlot =
  | 'mao'
  | 'arma'
  | 'armadura'
  | 'consumivel'
  | 'dedo'
  | 'ombros'
  | 'pe'
  | 'pulso'
  | 'pescoco'
  | 'cintura'

export type MagicItem = {
  id: string
  name: string
  kind: MagicItemKind
  tier: MagicItemTier
  slot: MagicItemSlot | null
  /**
   * Price in Tibares. `null` for artefatos (not for sale per book).
   * For weapon-enchant / armor-enchant entries, this is the encanto's
   * **additive** cost over the base superior item (book p334 Tab 8-7).
   */
  priceTibar: number | null
  /** Spell-item items: how many uses before the item is consumed. */
  charges: number | null
  /** Rare items the book caps at 1/day (e.g. Medalhão de Lena). */
  usesPerDay: number | null
  /** PM the wielder pays per activation (0 for passive items). */
  pmActivationCost: number
  /** Spell circle the item replicates, when relevant. */
  castingCircle: number | null
  /** Minimum character level required to use (artefatos). */
  requiresLevel: number | null
  /** Verbatim-close PT description of the effect. */
  effect: string
  bookPage: number
}

const ITEMS: readonly MagicItem[] = [
  {
    id: 'pocao-de-curar-ferimentos',
    name: 'Poção de Curar Ferimentos',
    kind: 'potion',
    tier: 'menor',
    slot: 'consumivel',
    priceTibar: 30,
    charges: null,
    usesPerDay: null,
    pmActivationCost: 0,
    castingCircle: 1,
    requiresLevel: null,
    effect: 'Ao beber, recupera 2d8+2 PV (replica Curar Ferimentos de 1º círculo).',
    bookPage: 341,
  },
  {
    id: 'pocao-de-velocidade',
    name: 'Poção de Velocidade',
    kind: 'potion',
    tier: 'medio',
    slot: 'consumivel',
    priceTibar: 270,
    charges: null,
    usesPerDay: null,
    pmActivationCost: 0,
    castingCircle: 2,
    requiresLevel: null,
    effect:
      'Replica Velocidade por uma cena: +1 ação de movimento extra e +3m de deslocamento.',
    bookPage: 341,
  },
  {
    id: 'pergaminho-de-curar-ferimentos',
    name: 'Pergaminho de Curar Ferimentos',
    kind: 'scroll',
    tier: 'menor',
    slot: 'consumivel',
    priceTibar: 30,
    charges: 1,
    usesPerDay: null,
    pmActivationCost: 0,
    castingCircle: 1,
    requiresLevel: null,
    effect:
      'Ao ler, lança Curar Ferimentos (2d8+2 PV); o pergaminho se desfaz em cinzas. Exige conhecer a magia ou Misticismo CD 20 + custo em PM.',
    bookPage: 341,
  },
  {
    id: 'pergaminho-de-bola-de-fogo',
    name: 'Pergaminho de Bola de Fogo',
    kind: 'scroll',
    tier: 'medio',
    slot: 'consumivel',
    priceTibar: 270,
    charges: 1,
    usesPerDay: null,
    pmActivationCost: 0,
    castingCircle: 3,
    requiresLevel: null,
    effect:
      'Lê em voz alta para lançar Bola de Fogo (6d6 fogo, esfera 6m, Reflexos metade); o pergaminho se desfaz.',
    bookPage: 341,
  },
  {
    id: 'cajado-do-poder',
    name: 'Cajado do Poder',
    kind: 'staff',
    tier: 'maior',
    slot: 'mao',
    priceTibar: 180000,
    charges: null,
    usesPerDay: null,
    pmActivationCost: 0,
    castingCircle: null,
    requiresLevel: null,
    effect:
      'Bordão defensor magnífico (+4 ataque/dano). Reduz em −1 o custo em PM das magias arcanas e aumenta em +2 a CD para resistir a elas.',
    bookPage: 337,
  },
  {
    id: 'cajado-da-vida',
    name: 'Cajado da Vida',
    kind: 'staff',
    tier: 'maior',
    slot: 'mao',
    priceTibar: 60000,
    charges: null,
    usesPerDay: null,
    pmActivationCost: 0,
    castingCircle: null,
    requiresLevel: null,
    effect:
      'Bordão formidável (+2 ataque/dano). Quando o portador lança uma magia de cura, ela cura +2 PV por dado.',
    bookPage: 337,
  },
  {
    id: 'encanto-flamejante',
    name: 'Encanto Flamejante (arma)',
    kind: 'weapon-enchant',
    tier: 'menor',
    slot: 'arma',
    priceTibar: 18000,
    charges: null,
    usesPerDay: null,
    pmActivationCost: 0,
    castingCircle: null,
    requiresLevel: null,
    effect:
      'A arma causa +1d6 de dano de fogo. Uma vez por rodada, pode gastar 2 PM para, no lugar do ataque normal, lançar Bola de Fogo (6d6 fogo, alcance médio).',
    bookPage: 335,
  },
  {
    id: 'encanto-defensor',
    name: 'Encanto Defensor (armadura/escudo)',
    kind: 'armor-enchant',
    tier: 'menor',
    slot: 'armadura',
    priceTibar: 18000,
    charges: null,
    usesPerDay: null,
    pmActivationCost: 0,
    castingCircle: null,
    requiresLevel: null,
    effect:
      'O bônus na Defesa do item aumenta em +2. Aplicável a armaduras e escudos.',
    bookPage: 338,
  },
  {
    id: 'cota-elfica',
    name: 'Cota Élfica',
    kind: 'armor-enchant',
    tier: 'maior',
    slot: 'armadura',
    priceTibar: 30000,
    charges: null,
    usesPerDay: null,
    pmActivationCost: 0,
    castingCircle: null,
    requiresLevel: null,
    effect:
      'Cota de malha de mitral, leve como seda. Aplica Destreza na Defesa como se fosse uma armadura leve.',
    bookPage: 340,
  },
  {
    id: 'anel-da-protecao',
    name: 'Anel da Proteção',
    kind: 'ring',
    tier: 'menor',
    slot: 'dedo',
    priceTibar: 9000,
    charges: null,
    usesPerDay: null,
    pmActivationCost: 0,
    castingCircle: null,
    requiresLevel: null,
    effect: 'Desvia ataques: +2 na Defesa do usuário (bônus de item).',
    bookPage: 342,
  },
  {
    id: 'manto-elfico',
    name: 'Manto Élfico',
    kind: 'wondrous',
    tier: 'menor',
    slot: 'ombros',
    priceTibar: 3000,
    charges: null,
    usesPerDay: null,
    pmActivationCost: 0,
    castingCircle: null,
    requiresLevel: null,
    effect:
      'Quando usado com o capuz cobrindo o rosto, fornece +5 em Furtividade.',
    bookPage: 344,
  },
  {
    id: 'botas-aladas',
    name: 'Botas Aladas',
    kind: 'wondrous',
    tier: 'medio',
    slot: 'pe',
    priceTibar: 15000,
    charges: null,
    usesPerDay: null,
    pmActivationCost: 2,
    castingCircle: null,
    requiresLevel: null,
    effect:
      'Gasta 2 PM para receber deslocamento de voo 12m por uma rodada; pode gastar 1 PM/turno para manter.',
    bookPage: 342,
  },
  {
    id: 'espada-deus',
    name: 'A Espada-Deus',
    kind: 'artifact',
    tier: 'artefato',
    slot: 'mao',
    priceTibar: null,
    charges: null,
    usesPerDay: null,
    pmActivationCost: 0,
    castingCircle: null,
    requiresLevel: 15,
    effect:
      'Espada longa atroz precisa pungente ameaçadora magnífica veloz (2d12 dano base). Ignora redução de dano e seus críticos afetam até criaturas imunes a críticos. Indestrutível. Exige 15 níveis em classe com Luta inicial.',
    bookPage: 346,
  },
]

export const MAGIC_ITEMS: Readonly<Record<string, MagicItem>> = Object.freeze(
  ITEMS.reduce<Record<string, MagicItem>>((acc, item) => {
    acc[item.id] = item
    return acc
  }, {}),
)

export const MAGIC_ITEM_IDS: readonly string[] = ITEMS.map((i) => i.id)

export function magicItemById(id: string): MagicItem {
  const item = MAGIC_ITEMS[id]
  if (!item) {
    throw new Error(`magicItemById: unknown magic item id "${id}"`)
  }
  return item
}

export function magicItemsByKind(kind: MagicItemKind): readonly MagicItem[] {
  return ITEMS.filter((i) => i.kind === kind)
}

export function magicItemsByTier(tier: MagicItemTier): readonly MagicItem[] {
  return ITEMS.filter((i) => i.tier === tier)
}
