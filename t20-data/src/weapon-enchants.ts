/**
 * Encantos de Arma (PDF Cap 6 — Tesouro, p335-336, Tabela 8-7 + 8-8).
 *
 * T20 rejects D&D 3.5's generic +N enhancement progression. Instead,
 * every weapon enhancement is a **named encanto** with a discrete
 * effect. Pricing is uniform: 1 encanto = +T$ 18.000, 2 encantos =
 * +T$ 36.000, 3 encantos = +T$ 72.000 (Tabela 8-7, p334).
 *
 * Slot budget vs item tier:
 *  - Item mágico menor: 1 encanto
 *  - Item mágico médio:  2 encantos
 *  - Item mágico maior:  3 encantos
 *
 * Three encantos consume **two slots** of the budget (marked `*` in
 * Tabela 8-8) and have prerequisite encantos on the same weapon:
 *
 *  - Energética (req. Formidável) — counts as 2 slots
 *  - Lancinante (req. Dilacerante) — counts as 2 slots; replaces
 *    Dilacerante's effect
 *  - Magnífica  (req. Formidável) — counts as 2 slots
 *
 * There is **no attunement** in T20 — every encanto is `requiresAttunement: false`.
 */

export type EnchantTier = 'menor' | 'medio' | 'maior' | 'artefato'
export type EnchantApplicability = 'todas' | 'corpo a corpo' | 'a distancia'

export type WeaponEnchant = {
  id: string
  name: string
  tier: EnchantTier
  /** Cost added to the base weapon price, in tibares. */
  priceTibar: number
  /** How many slots this encanto consumes (1 default; 2 for *-marked). */
  slotCost: 1 | 2
  /** Other encantos that must already be on the weapon. */
  prerequisiteEnchants: readonly string[]
  /** Weapon-category restriction. T20 mostly uses 'todas'. */
  applicableTo: readonly EnchantApplicability[]
  effect: string
  requiresAttunement: false
  bookPage: number
}

export const WEAPON_ENCHANTS: readonly WeaponEnchant[] = Object.freeze([
  {
    id: 'encanto-ameacadora',
    name: 'Ameaçadora',
    tier: 'menor',
    priceTibar: 18000,
    slotCost: 1,
    prerequisiteEnchants: [],
    applicableTo: ['todas'],
    effect:
      'A margem de ameaça da arma duplica. Por exemplo, uma espada longa ameaçadora tem margem de ameaça 17. Efeitos que duplicam a margem são aplicados antes de quaisquer efeitos que a aumentem.',
    requiresAttunement: false,
    bookPage: 335,
  },
  {
    id: 'encanto-anticriatura',
    name: 'Anticriatura',
    tier: 'menor',
    priceTibar: 18000,
    slotCost: 1,
    prerequisiteEnchants: [],
    applicableTo: ['todas'],
    effect:
      'A arma é letal contra um tipo de criatura (ou uma raça de humanoides). Uma vez por rodada, ao atacar uma criatura desse tipo, pode gastar 2 PM; se acertar, causa +4d8 de dano. Tipo determinado aleatoriamente (1d6): animal, construto, espírito, monstro, morto-vivo ou uma raça de humanoides.',
    requiresAttunement: false,
    bookPage: 335,
  },
  {
    id: 'encanto-arremesso',
    name: 'Arremesso',
    tier: 'menor',
    priceTibar: 18000,
    slotCost: 1,
    prerequisiteEnchants: [],
    applicableTo: ['corpo a corpo'],
    effect:
      'A arma pode ser arremessada em alcance curto. Caso já pudesse ser arremessada, seu alcance aumenta em uma categoria. Após o ataque, se estiver livre, a arma volta voando para você. Pegá-la é uma reação.',
    requiresAttunement: false,
    bookPage: 335,
  },
  {
    id: 'encanto-assassina',
    name: 'Assassina',
    tier: 'menor',
    priceTibar: 18000,
    slotCost: 1,
    prerequisiteEnchants: [],
    applicableTo: ['todas'],
    effect:
      'A arma aumenta os dados de dano extra de um ataque furtivo para d8. Além disso, ao fazer um Ataque Furtivo, pode gastar 2 PM para rolar novamente quaisquer resultados 1 nesses dados de dano extra.',
    requiresAttunement: false,
    bookPage: 335,
  },
  {
    id: 'encanto-cacadora',
    name: 'Caçadora',
    tier: 'menor',
    priceTibar: 18000,
    slotCost: 1,
    prerequisiteEnchants: [],
    applicableTo: ['todas'],
    effect:
      'A arma persegue o alvo, anulando penalidades por camuflagem leve e total e por cobertura leve. Caso a arma seja de ataque à distância, seu alcance também aumenta em uma categoria.',
    requiresAttunement: false,
    bookPage: 335,
  },
  {
    id: 'encanto-congelante',
    name: 'Congelante',
    tier: 'menor',
    priceTibar: 18000,
    slotCost: 1,
    prerequisiteEnchants: [],
    applicableTo: ['todas'],
    effect:
      'A arma causa +1d6 de dano de frio. Uma vez por rodada, ao atacar, pode gastar 2 PM; se acertar, a vítima fica enredada por uma rodada. Uma arma congelante é coberta por uma camada de gelo e névoa.',
    requiresAttunement: false,
    bookPage: 335,
  },
  {
    id: 'encanto-conjuradora',
    name: 'Conjuradora',
    tier: 'menor',
    priceTibar: 18000,
    slotCost: 1,
    prerequisiteEnchants: [],
    applicableTo: ['todas'],
    effect:
      'Um conjurador pode lançar na arma uma magia que tenha como alvo uma criatura ou que afete uma área. A magia fica guardada no item. Ao acertar um ataque com a arma, pode descarregar a magia guardada como ação livre e sem pagar seu custo, tendo como alvo (ou ponto de origem da área) a criatura ou ponto atingido. Uma vez descarregada, outra magia pode ser armazenada.',
    requiresAttunement: false,
    bookPage: 335,
  },
  {
    id: 'encanto-corrosiva',
    name: 'Corrosiva',
    tier: 'menor',
    priceTibar: 18000,
    slotCost: 1,
    prerequisiteEnchants: [],
    applicableTo: ['todas'],
    effect:
      'A arma causa +1d6 de dano de ácido. Uma vez por rodada, ao atacar, pode gastar 2 PM; se acertar, a vítima sofre 4d4 pontos de dano de ácido na próxima rodada. Uma arma corrosiva exala vapores e goteja líquido tóxico.',
    requiresAttunement: false,
    bookPage: 335,
  },
  {
    id: 'encanto-dancarina',
    name: 'Dançarina',
    tier: 'menor',
    priceTibar: 18000,
    slotCost: 1,
    prerequisiteEnchants: [],
    applicableTo: ['corpo a corpo'],
    effect:
      'Pode gastar uma ação de movimento e 1 PM para fazer a arma flutuar e atacar uma criatura em alcance curto a sua escolha, com as mesmas estatísticas que teria se você a estivesse empunhando. Efeito sustentado; se parar de sustentá-lo, a arma cai no chão.',
    requiresAttunement: false,
    bookPage: 335,
  },
  {
    id: 'encanto-defensora',
    name: 'Defensora',
    tier: 'menor',
    priceTibar: 18000,
    slotCost: 1,
    prerequisiteEnchants: [],
    applicableTo: ['todas'],
    effect:
      'A arma se movimenta para aparar ataques contra você. Você recebe +2 na Defesa.',
    requiresAttunement: false,
    bookPage: 335,
  },
  {
    id: 'encanto-destruidora',
    name: 'Destruidora',
    tier: 'menor',
    priceTibar: 18000,
    slotCost: 1,
    prerequisiteEnchants: [],
    applicableTo: ['todas'],
    effect:
      'Se usada contra construtos e objetos (com a manobra quebrar), a arma fornece +2 no teste de ataque e causa +2d8 de dano.',
    requiresAttunement: false,
    bookPage: 335,
  },
  {
    id: 'encanto-dilacerante',
    name: 'Dilacerante',
    tier: 'menor',
    priceTibar: 18000,
    slotCost: 1,
    prerequisiteEnchants: [],
    applicableTo: ['todas'],
    effect:
      'A arma inflige ferimentos profundos. Em um acerto crítico, causa +10 pontos de dano.',
    requiresAttunement: false,
    bookPage: 335,
  },
  {
    id: 'encanto-drenante',
    name: 'Drenante',
    tier: 'menor',
    priceTibar: 18000,
    slotCost: 1,
    prerequisiteEnchants: [],
    applicableTo: ['todas'],
    effect:
      'Em um acerto crítico contra criatura viva, a criatura fica fraca e você ganha 2d10 pontos de vida temporários. Uma arma drenante emite um brilho púrpura.',
    requiresAttunement: false,
    bookPage: 335,
  },
  {
    id: 'encanto-eletrica',
    name: 'Elétrica',
    tier: 'menor',
    priceTibar: 18000,
    slotCost: 1,
    prerequisiteEnchants: [],
    applicableTo: ['todas'],
    effect:
      'A arma causa +1d6 de dano de eletricidade. Uma vez por rodada, ao atacar, pode gastar 2 PM; se acertar, um raio atinge outra criatura em alcance curto, causando 3d8 pontos de dano de eletricidade. Uma arma elétrica emite faíscas e arcos voltaicos.',
    requiresAttunement: false,
    bookPage: 335,
  },
  {
    id: 'encanto-energetica',
    name: 'Energética',
    tier: 'medio',
    priceTibar: 36000,
    slotCost: 2,
    prerequisiteEnchants: ['encanto-formidavel'],
    applicableTo: ['todas'],
    effect:
      'A parte perigosa da arma (lâmina, ponta...) vira magia pura. +4 em testes de ataque, ignora 20 pontos de redução de dano, converte todo o dano para essência e emana luz como uma tocha. Conta como dois encantos. Pré-requisito: Formidável.',
    requiresAttunement: false,
    bookPage: 335,
  },
  {
    id: 'encanto-excruciante',
    name: 'Excruciante',
    tier: 'menor',
    priceTibar: 18000,
    slotCost: 1,
    prerequisiteEnchants: [],
    applicableTo: ['todas'],
    effect:
      'A arma inflige dor terrível. Uma criatura viva atingida fica fraca; se já estiver fraca (mesmo por este efeito), fica debilitada (condição máxima causada por esta arma).',
    requiresAttunement: false,
    bookPage: 335,
  },
  {
    id: 'encanto-flamejante',
    name: 'Flamejante',
    tier: 'menor',
    priceTibar: 18000,
    slotCost: 1,
    prerequisiteEnchants: [],
    applicableTo: ['todas'],
    effect:
      'A arma causa +1d6 de dano de fogo. Uma vez por rodada, ao atacar, pode gastar 2 PM; em vez do ataque normal, dispara uma bola de fogo em alcance médio (6d6 dano de fogo; Reflexos CD For ou Des, à sua escolha, reduz à metade). Uma arma flamejante emana chamas como uma tocha.',
    requiresAttunement: false,
    bookPage: 335,
  },
  {
    id: 'encanto-formidavel',
    name: 'Formidável',
    tier: 'menor',
    priceTibar: 18000,
    slotCost: 1,
    prerequisiteEnchants: [],
    applicableTo: ['todas'],
    effect:
      'A arma é encantada para desferir golpes precisos. Fornece +2 em testes de ataque e rolagens de dano.',
    requiresAttunement: false,
    bookPage: 336,
  },
  {
    id: 'encanto-lancinante',
    name: 'Lancinante',
    tier: 'medio',
    priceTibar: 36000,
    slotCost: 2,
    prerequisiteEnchants: ['encanto-dilacerante'],
    applicableTo: ['todas'],
    effect:
      'A arma inflige ferimentos mortais. Em acerto crítico, causa +10 pontos de dano ou, além de multiplicar os dados de dano, multiplica também quaisquer bônus numéricos, à sua escolha. Substitui o efeito de Dilacerante. Conta como dois encantos. Pré-requisito: Dilacerante.',
    requiresAttunement: false,
    bookPage: 336,
  },
  {
    id: 'encanto-magnifica',
    name: 'Magnífica',
    tier: 'medio',
    priceTibar: 36000,
    slotCost: 2,
    prerequisiteEnchants: ['encanto-formidavel'],
    applicableTo: ['todas'],
    effect:
      'A arma é encantada para desferir golpes perfeitos. +4 em testes de ataque e rolagens de dano. Conta como dois encantos. Pré-requisito: Formidável.',
    requiresAttunement: false,
    bookPage: 336,
  },
  {
    id: 'encanto-piedosa',
    name: 'Piedosa',
    tier: 'menor',
    priceTibar: 18000,
    slotCost: 1,
    prerequisiteEnchants: [],
    applicableTo: ['todas'],
    effect:
      'A arma causa +1d8 de dano, mas todo o dano causado é não letal. Pode gastar 1 PM para desativar/ativar este encanto.',
    requiresAttunement: false,
    bookPage: 336,
  },
  {
    id: 'encanto-profana',
    name: 'Profana',
    tier: 'menor',
    priceTibar: 18000,
    slotCost: 1,
    prerequisiteEnchants: [],
    applicableTo: ['todas'],
    effect:
      'A arma causa +2d8 de dano contra devotos de deuses que canalizam apenas energia positiva e criaturas bondosas (a critério do mestre). Uma arma profana emite luz rubra pulsante.',
    requiresAttunement: false,
    bookPage: 336,
  },
  {
    id: 'encanto-sagrada',
    name: 'Sagrada',
    tier: 'menor',
    priceTibar: 18000,
    slotCost: 1,
    prerequisiteEnchants: [],
    applicableTo: ['todas'],
    effect:
      'A arma causa +2d8 de dano contra devotos de deuses que canalizam apenas energia negativa e criaturas malignas (a critério do mestre). Uma arma sagrada emite uma sutil luz pura.',
    requiresAttunement: false,
    bookPage: 336,
  },
  {
    id: 'encanto-sanguinaria',
    name: 'Sanguinária',
    tier: 'menor',
    priceTibar: 18000,
    slotCost: 1,
    prerequisiteEnchants: [],
    applicableTo: ['todas'],
    effect:
      'Uma criatura viva atingida fica sangrando. A perda de PV por sangramento causada pela arma é cumulativa — uma criatura atingida duas vezes perde 2d6 PV por sangramento por rodada.',
    requiresAttunement: false,
    bookPage: 336,
  },
  {
    id: 'encanto-trovejante',
    name: 'Trovejante',
    tier: 'menor',
    priceTibar: 18000,
    slotCost: 1,
    prerequisiteEnchants: [],
    applicableTo: ['todas'],
    effect:
      'A arma emite um trovão ribombante a cada golpe. Em um acerto crítico, a vítima fica atordoada por uma rodada (apenas uma vez por cena; Fortitude CD For ou Des, à sua escolha, evita).',
    requiresAttunement: false,
    bookPage: 336,
  },
  {
    id: 'encanto-tumular',
    name: 'Tumular',
    tier: 'menor',
    priceTibar: 18000,
    slotCost: 1,
    prerequisiteEnchants: [],
    applicableTo: ['todas'],
    effect:
      'A arma causa +1d8 de dano de trevas. Uma vez por rodada, ao atacar, pode gastar 2 PM; o bônus de dano aumenta para +2d8, mas você perde 1d8 pontos de vida. Uma arma tumular drena o calor ao redor.',
    requiresAttunement: false,
    bookPage: 336,
  },
  {
    id: 'encanto-veloz',
    name: 'Veloz',
    tier: 'menor',
    priceTibar: 18000,
    slotCost: 1,
    prerequisiteEnchants: [],
    applicableTo: ['todas'],
    effect:
      'Você recebe a habilidade Ataque Extra (do guerreiro), mas só pode usá-la com esta arma. Se já a possui, em vez disso o custo para usá-la com esta arma diminui em −1 PM.',
    requiresAttunement: false,
    bookPage: 336,
  },
  {
    id: 'encanto-venenosa',
    name: 'Venenosa',
    tier: 'menor',
    priceTibar: 18000,
    slotCost: 1,
    prerequisiteEnchants: [],
    applicableTo: ['todas'],
    effect:
      'Uma vez por rodada, ao atacar, pode gastar 2 PM; se acertar, a vítima fica envenenada, perdendo 1d12 pontos de vida por rodada durante 3 rodadas. Uma arma venenosa verte um líquido verde e viscoso.',
    requiresAttunement: false,
    bookPage: 336,
  },
])

/** Item-tier → slot budget (Tabela 8-7, p334). */
export const WEAPON_TIER_SLOT_BUDGET: Record<EnchantTier, number> = Object.freeze(
  {
    menor: 1,
    medio: 2,
    maior: 3,
    artefato: 3,
  },
)

const byId = new Map(WEAPON_ENCHANTS.map((e) => [e.id, e]))

export function weaponEnchantById(id: string): WeaponEnchant | undefined {
  return byId.get(id)
}

export function weaponEnchantsByTier(
  tier: EnchantTier,
): readonly WeaponEnchant[] {
  return WEAPON_ENCHANTS.filter((e) => e.tier === tier)
}

/**
 * Slot-budget check for a proposed encanto loadout. Returns true if
 * the loadout's total slotCost fits within the weapon's tier budget
 * AND every prerequisite is satisfied within the same loadout.
 */
export function isValidEnchantLoadout(
  weaponTier: EnchantTier,
  enchantIds: readonly string[],
): boolean {
  const budget = WEAPON_TIER_SLOT_BUDGET[weaponTier]
  const present = new Set<string>()
  let used = 0
  for (const id of enchantIds) {
    const e = byId.get(id)
    if (!e) return false
    if (present.has(id)) return false // no duplicates
    present.add(id)
    used += e.slotCost
  }
  if (used > budget) return false
  for (const id of enchantIds) {
    const e = byId.get(id)!
    for (const req of e.prerequisiteEnchants) {
      if (!present.has(req)) return false
    }
  }
  return true
}

/**
 * Total tibar surcharge of a loadout (sum of priceTibar values).
 * Returns -1 if any id is unknown.
 */
export function loadoutPriceTibar(enchantIds: readonly string[]): number {
  let total = 0
  for (const id of enchantIds) {
    const e = byId.get(id)
    if (!e) return -1
    total += e.priceTibar
  }
  return total
}
