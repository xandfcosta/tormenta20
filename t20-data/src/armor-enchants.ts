import type { EnchantTier } from './weapon-enchants'

/**
 * Encantos de Armadura/Escudo (PDF Cap 6 — Tesouro, p338-339, Tabela 8-10).
 *
 * Same pricing model as weapon encantos (Tabela 8-7, p334): 1 slot =
 * +T$ 18.000, 2 slots = +T$ 36.000, 3 slots = +T$ 72.000. T20 has no
 * generic +N armor enhancement bonus — every Defesa boost lives inside
 * a named encanto (`Defensor` = +2 Defesa; `Guardião` = +4 Defesa).
 *
 * Slot budget mirrors weapons:
 *  - menor = 1, médio = 2, maior = 3, artefato = 3
 *
 * Only one 2-slot armor encanto exists: `Guardião` (req `Defensor`).
 *
 * Shield-only encantos (footnote 1 in Tabela 8-10): `Animado`,
 * `Esmagador`. All others apply to armaduras leves, pesadas e escudos
 * (`'todas'`).
 *
 * No attunement: `requiresAttunement: false` for every entry.
 */

export type ArmorEnchantApplicability =
  | 'todas'
  | 'armaduras-leves'
  | 'armaduras-pesadas'
  | 'escudos'

export type ArmorEnchant = {
  id: string
  name: string
  tier: EnchantTier
  priceTibar: number
  slotCost: 1 | 2
  prerequisiteEnchants: readonly string[]
  applicableTo: readonly ArmorEnchantApplicability[]
  effect: string
  requiresAttunement: false
  bookPage: number
}

export const ARMOR_ENCHANTS: readonly ArmorEnchant[] = Object.freeze([
  {
    id: 'encanto-abascanto',
    name: 'Abascanto',
    tier: 'menor',
    priceTibar: 18000,
    slotCost: 1,
    prerequisiteEnchants: [],
    applicableTo: ['todas'],
    effect: 'Você recebe resistência a magia +5.',
    requiresAttunement: false,
    bookPage: 338,
  },
  {
    id: 'encanto-abencoado',
    name: 'Abençoado',
    tier: 'menor',
    priceTibar: 18000,
    slotCost: 1,
    prerequisiteEnchants: [],
    applicableTo: ['todas'],
    effect:
      'Você recebe redução de trevas 10 e +5 em testes de resistência contra efeitos de necromancia. Um item abençoado é decorado com gravuras de símbolos sagrados de deuses do Bem.',
    requiresAttunement: false,
    bookPage: 338,
  },
  {
    id: 'encanto-acrobatico',
    name: 'Acrobático',
    tier: 'menor',
    priceTibar: 18000,
    slotCost: 1,
    prerequisiteEnchants: [],
    applicableTo: ['todas'],
    effect:
      'Você recebe +5 em Acrobacia e ignora a penalidade de armadura do item para testes dessa perícia.',
    requiresAttunement: false,
    bookPage: 338,
  },
  {
    id: 'encanto-alado',
    name: 'Alado',
    tier: 'menor',
    priceTibar: 18000,
    slotCost: 1,
    prerequisiteEnchants: [],
    applicableTo: ['todas'],
    effect:
      'Você pode gastar 2 PM para fazer asas emergirem de suas costas e receber deslocamento de voo 12m com duração sustentada.',
    requiresAttunement: false,
    bookPage: 338,
  },
  {
    id: 'encanto-animado',
    name: 'Animado',
    tier: 'menor',
    priceTibar: 18000,
    slotCost: 1,
    prerequisiteEnchants: [],
    applicableTo: ['escudos'],
    effect:
      'Pode gastar uma ação de movimento e 1 PM para fazer o escudo flutuar ao seu redor até o fim da cena. Você recebe o mesmo bônus na Defesa que receberia se estivesse empunhando o escudo, mas fica com as duas mãos livres. Só pode ser protegido por um escudo ao mesmo tempo.',
    requiresAttunement: false,
    bookPage: 338,
  },
  {
    id: 'encanto-assustador',
    name: 'Assustador',
    tier: 'menor',
    priceTibar: 18000,
    slotCost: 1,
    prerequisiteEnchants: [],
    applicableTo: ['todas'],
    effect:
      'Pode gastar uma ação de movimento e 2 PM para gerar uma onda de medo. Inimigos em alcance curto devem passar num teste de Vontade (CD Car) ou ficarão abalados até o fim da cena. Um item assustador possui manchas de sangue, ossos pendurados e outras decorações horripilantes.',
    requiresAttunement: false,
    bookPage: 338,
  },
  {
    id: 'encanto-caustica',
    name: 'Cáustica',
    tier: 'menor',
    priceTibar: 18000,
    slotCost: 1,
    prerequisiteEnchants: [],
    applicableTo: ['todas'],
    effect:
      'Você recebe redução de ácido 10 e pode gastar uma ação de movimento e 2 PM para fazer o item gotejar ácido. Se fizer isso, seus ataques causam +1d4 de dano de ácido até o fim da cena.',
    requiresAttunement: false,
    bookPage: 338,
  },
  {
    id: 'encanto-defensor',
    name: 'Defensor',
    tier: 'menor',
    priceTibar: 18000,
    slotCost: 1,
    prerequisiteEnchants: [],
    applicableTo: ['todas'],
    effect:
      'O item é encantado para desviar golpes. O bônus na Defesa do item aumenta em +2.',
    requiresAttunement: false,
    bookPage: 338,
  },
  {
    id: 'encanto-escorregadio',
    name: 'Escorregadio',
    tier: 'menor',
    priceTibar: 18000,
    slotCost: 1,
    prerequisiteEnchants: [],
    applicableTo: ['todas'],
    effect:
      'Você recebe +10 em testes de Acrobacia para escapar e em testes de manobra contra agarrar. Um item escorregadio parece estar sempre coberto de óleo levemente gorduroso.',
    requiresAttunement: false,
    bookPage: 338,
  },
  {
    id: 'encanto-esmagador',
    name: 'Esmagador',
    tier: 'menor',
    priceTibar: 18000,
    slotCost: 1,
    prerequisiteEnchants: [],
    applicableTo: ['escudos'],
    effect:
      'Este escudo fornece +2 em ataques e dano e tem seu dano aumentado em um passo.',
    requiresAttunement: false,
    bookPage: 339,
  },
  {
    id: 'encanto-fantasmagorico',
    name: 'Fantasmagórico',
    tier: 'menor',
    priceTibar: 18000,
    slotCost: 1,
    prerequisiteEnchants: [],
    applicableTo: ['todas'],
    effect:
      'Você pode lançar a magia Manto de Sombras. Um item fantasmagórico é cinzento e esfumaçado.',
    requiresAttunement: false,
    bookPage: 339,
  },
  {
    id: 'encanto-fortificado',
    name: 'Fortificado',
    tier: 'menor',
    priceTibar: 18000,
    slotCost: 1,
    prerequisiteEnchants: [],
    applicableTo: ['todas'],
    effect:
      'Você recebe 25% de chance (para escudos) e 50% de chance (para armaduras) de ignorar o dano extra de acertos críticos e ataques furtivos.',
    requiresAttunement: false,
    bookPage: 339,
  },
  {
    id: 'encanto-gelido',
    name: 'Gélido',
    tier: 'menor',
    priceTibar: 18000,
    slotCost: 1,
    prerequisiteEnchants: [],
    applicableTo: ['todas'],
    effect:
      'Você recebe redução de frio 10 e pode gastar uma ação de movimento e 2 PM para se cobrir de gelo até o fim da cena. Se fizer isso, recebe 10 PV temporários. Um item gélido é azulado e frio ao toque.',
    requiresAttunement: false,
    bookPage: 339,
  },
  {
    id: 'encanto-guardiao',
    name: 'Guardião',
    tier: 'medio',
    priceTibar: 36000,
    slotCost: 2,
    prerequisiteEnchants: ['encanto-defensor'],
    applicableTo: ['todas'],
    effect:
      'O item emite um campo de força que desvia ataques. O bônus na Defesa do item aumenta em +4. Conta como dois encantos. Pré-requisito: Defensor.',
    requiresAttunement: false,
    bookPage: 339,
  },
  {
    id: 'encanto-hipnotico',
    name: 'Hipnótico',
    tier: 'menor',
    priceTibar: 18000,
    slotCost: 1,
    prerequisiteEnchants: [],
    applicableTo: ['todas'],
    effect:
      'Pode gastar uma ação padrão e 3 PM para emitir luzes coloridas. Inimigos em alcance curto devem passar num teste de Vontade (CD Car) ou ficarão fascinados por 1d6 rodadas. O efeito termina se qualquer criatura afetada for atacada. Um item hipnótico é espalhafatoso e colorido.',
    requiresAttunement: false,
    bookPage: 339,
  },
  {
    id: 'encanto-ilusorio',
    name: 'Ilusório',
    tier: 'menor',
    priceTibar: 18000,
    slotCost: 1,
    prerequisiteEnchants: [],
    applicableTo: ['todas'],
    effect:
      'Pode gastar uma ação de movimento e 1 PM para fazer o item adquirir a aparência de uma roupa comum, mas mantendo suas propriedades (bônus na Defesa, penalidade de armadura...). A magia Visão da Verdade revela o item disfarçado.',
    requiresAttunement: false,
    bookPage: 339,
  },
  {
    id: 'encanto-incandescente',
    name: 'Incandescente',
    tier: 'menor',
    priceTibar: 18000,
    slotCost: 1,
    prerequisiteEnchants: [],
    applicableTo: ['todas'],
    effect:
      'Você recebe redução de fogo 10 e pode gastar uma ação de movimento e 2 PM para fazer o item emitir labaredas até o fim da cena. Se fizer isso, no início de cada um de seus turnos causa 1d6 pontos de dano de fogo em todas as criaturas adjacentes. Um item incandescente é avermelhado e quente ao toque.',
    requiresAttunement: false,
    bookPage: 339,
  },
  {
    id: 'encanto-invulneravel',
    name: 'Invulnerável',
    tier: 'menor',
    priceTibar: 18000,
    slotCost: 1,
    prerequisiteEnchants: [],
    applicableTo: ['todas'],
    effect: 'Você recebe redução de dano 2 (para escudos) ou 5 (para armaduras).',
    requiresAttunement: false,
    bookPage: 339,
  },
  {
    id: 'encanto-opaco',
    name: 'Opaco',
    tier: 'menor',
    priceTibar: 18000,
    slotCost: 1,
    prerequisiteEnchants: [],
    applicableTo: ['todas'],
    effect:
      'Você recebe redução de ácido, eletricidade, fogo e frio 10. Um item opaco parece sem cor, totalmente comum e desinteressante.',
    requiresAttunement: false,
    bookPage: 339,
  },
  {
    id: 'encanto-protetor',
    name: 'Protetor',
    tier: 'menor',
    priceTibar: 18000,
    slotCost: 1,
    prerequisiteEnchants: [],
    applicableTo: ['todas'],
    effect: 'Você recebe +2 em testes de resistência.',
    requiresAttunement: false,
    bookPage: 339,
  },
  {
    id: 'encanto-refletor',
    name: 'Refletor',
    tier: 'menor',
    priceTibar: 18000,
    slotCost: 1,
    prerequisiteEnchants: [],
    applicableTo: ['todas'],
    effect:
      'Uma vez por rodada, quando é alvo de uma magia, pode gastar PM igual ao custo dela para refleti-la de volta ao conjurador. As características da magia (efeitos, CD...) se mantêm, mas você toma qualquer decisão exigida por ela. Um item refletor parece espelhado.',
    requiresAttunement: false,
    bookPage: 339,
  },
  {
    id: 'encanto-relampejante',
    name: 'Relampejante',
    tier: 'menor',
    priceTibar: 18000,
    slotCost: 1,
    prerequisiteEnchants: [],
    applicableTo: ['todas'],
    effect:
      'Você recebe redução de eletricidade 10 e pode gastar uma ação de movimento e 2 PM para gerar arcos voltaicos até o fim da cena. Se fizer isso, qualquer criatura que o ataque em corpo a corpo sofre 2d6 pontos de dano de eletricidade. Um item relampejante é decorado com ouro, prata e cobre.',
    requiresAttunement: false,
    bookPage: 339,
  },
  {
    id: 'encanto-reluzente',
    name: 'Reluzente',
    tier: 'menor',
    priceTibar: 18000,
    slotCost: 1,
    prerequisiteEnchants: [],
    applicableTo: ['todas'],
    effect:
      'Pode gastar uma ação de movimento e 2 PM para emitir um clarão de luz. Todos os inimigos em alcance curto devem passar num teste de Reflexos (CD Car) ou ficarão cegos por uma rodada. Um item reluzente é polido e brilhante.',
    requiresAttunement: false,
    bookPage: 339,
  },
  {
    id: 'encanto-sombrio',
    name: 'Sombrio',
    tier: 'menor',
    priceTibar: 18000,
    slotCost: 1,
    prerequisiteEnchants: [],
    applicableTo: ['todas'],
    effect:
      'Você recebe +5 em Furtividade e ignora a penalidade de armadura do item para testes dessa perícia. Um item sombrio é escuro, fosco e bem lubrificado, para não fazer barulho.',
    requiresAttunement: false,
    bookPage: 339,
  },
  {
    id: 'encanto-zeloso',
    name: 'Zeloso',
    tier: 'menor',
    priceTibar: 18000,
    slotCost: 1,
    prerequisiteEnchants: [],
    applicableTo: ['todas'],
    effect:
      'Uma vez por rodada, se um aliado adjacente for alvo de um ataque, pode gastar 1 PM para se tornar o alvo do ataque, que então é resolvido normalmente.',
    requiresAttunement: false,
    bookPage: 339,
  },
])

/** Item-tier → slot budget (Tabela 8-7, p334). Mirrors weapons. */
export const ARMOR_TIER_SLOT_BUDGET: Record<EnchantTier, number> = Object.freeze(
  {
    menor: 1,
    medio: 2,
    maior: 3,
    artefato: 3,
  },
)

export type ArmorBaseKind = 'armadura-leve' | 'armadura-pesada' | 'escudo'

const byId = new Map(ARMOR_ENCHANTS.map((e) => [e.id, e]))

export function armorEnchantById(id: string): ArmorEnchant | undefined {
  return byId.get(id)
}

export function armorEnchantsByTier(
  tier: EnchantTier,
): readonly ArmorEnchant[] {
  return ARMOR_ENCHANTS.filter((e) => e.tier === tier)
}

/** True if the encanto is fittable on the given base item kind. */
export function isApplicableToBase(
  enchant: ArmorEnchant,
  base: ArmorBaseKind,
): boolean {
  for (const allow of enchant.applicableTo) {
    if (allow === 'todas') return true
    if (allow === 'escudos' && base === 'escudo') return true
    if (allow === 'armaduras-leves' && base === 'armadura-leve') return true
    if (allow === 'armaduras-pesadas' && base === 'armadura-pesada') return true
  }
  return false
}

/**
 * Slot-budget + prerequisite + base-compatibility validator. Returns
 * true if the loadout fits the base item's tier budget, every encanto
 * applies to the base kind, every prerequisite is satisfied within the
 * same loadout, and no duplicates appear.
 */
export function isValidArmorEnchantLoadout(
  base: ArmorBaseKind,
  baseTier: EnchantTier,
  enchantIds: readonly string[],
): boolean {
  const budget = ARMOR_TIER_SLOT_BUDGET[baseTier]
  const present = new Set<string>()
  let used = 0
  for (const id of enchantIds) {
    const e = byId.get(id)
    if (!e) return false
    if (present.has(id)) return false
    if (!isApplicableToBase(e, base)) return false
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

export function armorLoadoutPriceTibar(
  enchantIds: readonly string[],
): number {
  let total = 0
  for (const id of enchantIds) {
    const e = byId.get(id)
    if (!e) return -1
    total += e.priceTibar
  }
  return total
}
