import type { SpecificItemTier } from './specific-magic-items'

/**
 * Itens Mágicos Maravilhosos / Acessórios — PDF Cap 6 (Tesouro):
 *
 *  - Tabela 8-13 (menores, p342) — T$ 3.000-9.000.
 *  - Tabela 8-14 (médios,  p343) — T$ 10.500-25.500.
 *  - Tabela 8-15 (maiores, p343) — T$ 30.000-150.000.
 *
 * Descriptions sprawl across p342-345.
 *
 * **T20 has NO rule-mechanical slot system.** Itens maravilhosos do
 * not compete for "head"/"neck"/"ring" slots like D&D 3.5. The `slot`
 * field below is a **semantic hint for UI / inventory grouping only** —
 * derived from item names (manto → corpo, anel → anel, etc.). It does
 * NOT enforce stacking rules; PDF p333 only says identical bonuses
 * from multiple items do not stack (use the highest).
 *
 * "Attunement-like" ramp-up ("somente após um dia de uso") lives in
 * the `effect` text — T20 has no boolean attunement flag, so every
 * entry is `requiresAttunement: false`.
 *
 * Subset: ~38 entries representing a cross-section across all 3
 * tabelas. Full catalogs in PDF have ~70 entries combined; later PRs
 * may extend this list.
 */
export type WondrousSlot =
  | 'anel'
  | 'cabeca'
  | 'pescoco'
  | 'corpo'
  | 'cintura'
  | 'maos'
  | 'pes'
  | 'outro'

export type WondrousItem = {
  id: string
  name: string
  slot: WondrousSlot
  tier: SpecificItemTier
  priceTibar: number
  effect: string
  requiresClass: string | null
  requiresAttunement: false
  bookPage: number
}

export const WONDROUS_ITEMS: readonly WondrousItem[] = Object.freeze([
  // ─── Menores — Tabela 8-13, p342 ────────────────────────────────
  {
    id: 'anel-do-sustento',
    name: 'Anel do Sustento',
    slot: 'anel',
    tier: 'menor',
    priceTibar: 3000,
    effect:
      'Não precisa comer ou beber e precisa dormir apenas duas horas por noite para descansar. Os efeitos só se ativam após uma semana de uso.',
    requiresClass: null,
    requiresAttunement: false,
    bookPage: 342,
  },
  {
    id: 'bainha-magica',
    name: 'Bainha Mágica',
    slot: 'outro',
    tier: 'menor',
    priceTibar: 3000,
    effect:
      'Esta bainha de couro curtido e prata muda de tamanho para acomodar qualquer arma corpo a corpo. Pode lançar Arma Mágica em qualquer arma na bainha sem pagar seu custo em PM.',
    requiresClass: null,
    requiresAttunement: false,
    bookPage: 342,
  },
  {
    id: 'corda-da-escalada',
    name: 'Corda da Escalada',
    slot: 'outro',
    tier: 'menor',
    priceTibar: 3000,
    effect:
      'Esta corda de 15m é fina mas suporta até seis criaturas Médias (ou 120 espaços). Com um comando (ação de movimento), move-se em qualquer direção (incluindo para cima) a 3m por rodada, fixando-se firmemente onde quiser. Pode desamarrar e voltar do mesmo modo.',
    requiresClass: null,
    requiresAttunement: false,
    bookPage: 343,
  },
  {
    id: 'ferraduras-da-velocidade',
    name: 'Ferraduras da Velocidade',
    slot: 'outro',
    tier: 'menor',
    priceTibar: 3000,
    effect:
      'Conjunto de ferraduras fixadas em cascos de cavalo (ou outro parceiro montaria, a critério do mestre) para aumentar o deslocamento em +3m.',
    requiresClass: null,
    requiresAttunement: false,
    bookPage: 344,
  },
  {
    id: 'gema-da-luminosidade',
    name: 'Gema da Luminosidade',
    slot: 'outro',
    tier: 'menor',
    priceTibar: 3000,
    effect:
      'Cristal em prisma longo. Com um comando, emite luz equivalente a uma tocha ou um raio brilhante que deixa criatura em alcance curto cega por 1d4 rodadas (Fort CD Car evita).',
    requiresClass: null,
    requiresAttunement: false,
    bookPage: 344,
  },
  {
    id: 'manto-elfico',
    name: 'Manto Élfico',
    slot: 'corpo',
    tier: 'menor',
    priceTibar: 3000,
    effect:
      'Indistinguível de um manto cinza comum. Quando usado com o capuz cobrindo o rosto, fornece +5 em Furtividade.',
    requiresClass: null,
    requiresAttunement: false,
    bookPage: 345,
  },
  {
    id: 'brincos-da-sagacidade',
    name: 'Brincos da Sagacidade',
    slot: 'cabeca',
    tier: 'menor',
    priceTibar: 4500,
    effect:
      'Par de brincos de safira que aguça o raciocínio. +1 em Inteligência (somente após um dia de uso).',
    requiresClass: null,
    requiresAttunement: false,
    bookPage: 342,
  },
  {
    id: 'luvas-da-delicadeza',
    name: 'Luvas da Delicadeza',
    slot: 'maos',
    tier: 'menor',
    priceTibar: 4500,
    effect:
      'Luvas de tecido fino permitem manipulação delicada. +1 em Destreza (somente após um dia de uso).',
    requiresClass: null,
    requiresAttunement: false,
    bookPage: 344,
  },
  {
    id: 'manoplas-da-forca-do-ogro',
    name: 'Manoplas da Força do Ogro',
    slot: 'maos',
    tier: 'menor',
    priceTibar: 4500,
    effect:
      'Par de luvas de couro grosso com rebites de ferro. +1 em Força (somente após um dia de uso).',
    requiresClass: null,
    requiresAttunement: false,
    bookPage: 344,
  },
  {
    id: 'manto-da-resistencia',
    name: 'Manto da Resistência',
    slot: 'corpo',
    tier: 'menor',
    priceTibar: 4500,
    effect:
      'Manto de tecido grosso e pesado que protege o usuário. +2 em testes de resistência.',
    requiresClass: null,
    requiresAttunement: false,
    bookPage: 344,
  },
  {
    id: 'manto-do-fascinio',
    name: 'Manto do Fascínio',
    slot: 'corpo',
    tier: 'menor',
    priceTibar: 4500,
    effect:
      'Manto de veludo com bordados de ouro. +1 em Carisma (somente após um dia de uso).',
    requiresClass: null,
    requiresAttunement: false,
    bookPage: 344,
  },
  {
    id: 'pingente-da-sensatez',
    name: 'Pingente da Sensatez',
    slot: 'pescoco',
    tier: 'menor',
    priceTibar: 4500,
    effect:
      'Pequena pérola com corrente leve usada como colar. +1 em Sabedoria (somente após um dia de uso).',
    requiresClass: null,
    requiresAttunement: false,
    bookPage: 345,
  },
  {
    id: 'torque-do-vigor',
    name: 'Torque do Vigor',
    slot: 'pescoco',
    tier: 'menor',
    priceTibar: 4500,
    effect:
      'Colar ou bracelete com acabamento remetendo a animal poderoso (urso, lobo). +1 em Constituição (somente após um dia de uso).',
    requiresClass: null,
    requiresAttunement: false,
    bookPage: 345,
  },
  {
    id: 'chapeu-do-disfarce',
    name: 'Chapéu do Disfarce',
    slot: 'cabeca',
    tier: 'menor',
    priceTibar: 6000,
    effect:
      'Pode lançar Disfarce Ilusório (CD Car) com aprimoramento de odor/sensação e +20 em Enganação para disfarces, sem pagar PM. Não pode usar outros aprimoramentos. Como parte do disfarce, o chapéu pode mudar para elmo, faixa, tiara, gorro, touca, etc.',
    requiresClass: null,
    requiresAttunement: false,
    bookPage: 343,
  },
  {
    id: 'anel-da-protecao',
    name: 'Anel da Proteção',
    slot: 'anel',
    tier: 'menor',
    priceTibar: 9000,
    effect: 'Este anel desvia ataques contra o usuário. +2 em Defesa.',
    requiresClass: null,
    requiresAttunement: false,
    bookPage: 342,
  },
  {
    id: 'anel-do-escudo-mental',
    name: 'Anel do Escudo Mental',
    slot: 'anel',
    tier: 'menor',
    priceTibar: 9000,
    effect: 'Imunidade a magias de adivinhação.',
    requiresClass: null,
    requiresAttunement: false,
    bookPage: 342,
  },

  // ─── Médios — Tabela 8-14, p343 ─────────────────────────────────
  {
    id: 'anel-de-telecinesia',
    name: 'Anel de Telecinesia',
    slot: 'anel',
    tier: 'medio',
    priceTibar: 10500,
    effect:
      'Pode lançar Telecinesia (CD Int). Caso já conheça a magia, o custo para lançá-la diminui em −1 PM.',
    requiresClass: null,
    requiresAttunement: false,
    bookPage: 342,
  },
  {
    id: 'bola-de-cristal',
    name: 'Bola de Cristal',
    slot: 'outro',
    tier: 'medio',
    priceTibar: 10500,
    effect:
      'Pequena esfera que revela pessoas e lugares distantes. Olhar através dela é ação completa e gera a magia Vidência (CD Sab).',
    requiresClass: null,
    requiresAttunement: false,
    bookPage: 342,
  },
  {
    id: 'botas-aladas',
    name: 'Botas Aladas',
    slot: 'pes',
    tier: 'medio',
    priceTibar: 15000,
    effect:
      'Pode gastar 2 PM para fazer asas brotarem dos calcanhares e receber deslocamento de voo 12m por uma rodada. Pode gastar 1 PM no início de cada turno para manter o efeito.',
    requiresClass: null,
    requiresAttunement: false,
    bookPage: 342,
  },
  {
    id: 'bracadeiras-do-arqueiro',
    name: 'Braçadeiras do Arqueiro',
    slot: 'maos',
    tier: 'medio',
    priceTibar: 21000,
    effect:
      '+2 em rolagens de dano com armas de ataque à distância (cumulativo com outros itens).',
    requiresClass: null,
    requiresAttunement: false,
    bookPage: 342,
  },
  {
    id: 'anel-da-energia',
    name: 'Anel da Energia',
    slot: 'anel',
    tier: 'medio',
    priceTibar: 21000,
    effect: '+5 PM (somente após um dia de uso).',
    requiresClass: null,
    requiresAttunement: false,
    bookPage: 342,
  },
  {
    id: 'anel-da-vitalidade',
    name: 'Anel da Vitalidade',
    slot: 'anel',
    tier: 'medio',
    priceTibar: 21000,
    effect: '+10 PV (somente após um dia de uso).',
    requiresClass: null,
    requiresAttunement: false,
    bookPage: 342,
  },
  {
    id: 'anel-de-invisibilidade',
    name: 'Anel de Invisibilidade',
    slot: 'anel',
    tier: 'medio',
    priceTibar: 21000,
    effect:
      'Ao colocar este anel de prata, fica sob efeito de Invisibilidade. O efeito termina se fizer ataque ou lançar magia ofensiva, mas pode tirar e recolocar o anel (ação padrão) para reativar.',
    requiresClass: null,
    requiresAttunement: false,
    bookPage: 342,
  },
  {
    id: 'manto-da-aranha',
    name: 'Manto da Aranha',
    slot: 'corpo',
    tier: 'medio',
    priceTibar: 21000,
    effect:
      'Manto de seda negra com fios de prata bordados. Recebe deslocamento de escalada igual ao terrestre, +5 em testes de resistência contra venenos e imunidade a teias mundanas ou mágicas. Pode lançar Teia (CD Des). Caso já conheça a magia, custo diminui em −1 PM.',
    requiresClass: null,
    requiresAttunement: false,
    bookPage: 344,
  },
  {
    id: 'simbolo-abencoado',
    name: 'Símbolo Abençoado',
    slot: 'outro',
    tier: 'medio',
    priceTibar: 21000,
    effect:
      'Conta como símbolo sagrado. Se for devoto do deus, custo das magias divinas diminui em −1 PM (cumulativo com Símbolo Sagrado Energizado). Apenas devotos desse deus podem fabricar.',
    requiresClass: null,
    requiresAttunement: false,
    bookPage: 345,
  },
  {
    id: 'botas-velozes',
    name: 'Botas Velozes',
    slot: 'pes',
    tier: 'medio',
    priceTibar: 25500,
    effect:
      '+3m em deslocamento e pode lançar Velocidade (apenas sobre você mesmo).',
    requiresClass: null,
    requiresAttunement: false,
    bookPage: 342,
  },
  {
    id: 'cinto-da-forca-do-gigante',
    name: 'Cinto da Força do Gigante',
    slot: 'cintura',
    tier: 'medio',
    priceTibar: 25500,
    effect:
      'Cinto largo de couro com rebites de ferro. +2 em Força (somente após um dia de uso).',
    requiresClass: null,
    requiresAttunement: false,
    bookPage: 343,
  },
  {
    id: 'coroa-majestosa',
    name: 'Coroa Majestosa',
    slot: 'cabeca',
    tier: 'medio',
    priceTibar: 25500,
    effect:
      'Coroa de ouro com dezenas de pedras preciosas. +2 em Carisma (somente após um dia de uso).',
    requiresClass: null,
    requiresAttunement: false,
    bookPage: 344,
  },
  {
    id: 'manto-do-morcego',
    name: 'Manto do Morcego',
    slot: 'corpo',
    tier: 'medio',
    priceTibar: 25500,
    effect:
      'Manto marrom escuro ou negro: +5 em Furtividade e permite ficar pendurado de ponta-cabeça no teto. Pode gastar ação padrão para segurar as pontas e se transformar em morcego (Minúsculo, voo 12m, arma natural mordida 1d4 perfuração; funciona como Forma Selvagem do druida). Só transforma à noite ou em ambientes escuros.',
    requiresClass: null,
    requiresAttunement: false,
    bookPage: 344,
  },
  {
    id: 'tiara-da-sapiencia',
    name: 'Tiara da Sapiência',
    slot: 'cabeca',
    tier: 'medio',
    priceTibar: 25500,
    effect:
      'Tiara delicada com gema sobre a testa. +2 em Inteligência (somente após um dia de uso).',
    requiresClass: null,
    requiresAttunement: false,
    bookPage: 345,
  },

  // ─── Maiores — Tabela 8-15, p343 ────────────────────────────────
  {
    id: 'elmo-do-teletransporte',
    name: 'Elmo do Teletransporte',
    slot: 'cabeca',
    tier: 'maior',
    priceTibar: 30000,
    effect:
      'Pode lançar Salto Dimensional e Teletransporte, mas apenas em si mesmo. Caso já conheça as magias, custo diminui em −1 PM.',
    requiresClass: null,
    requiresAttunement: false,
    bookPage: 344,
  },
  {
    id: 'medalhao-de-lena',
    name: 'Medalhão de Lena',
    slot: 'pescoco',
    tier: 'maior',
    priceTibar: 30000,
    effect:
      'Quando reduzido a 0 ou menos PV, esta joia emite explosão de energia positiva que cura 100 PV (antes de cair). Ativa-se apenas uma vez por dia.',
    requiresClass: null,
    requiresAttunement: false,
    bookPage: 345,
  },
  {
    id: 'colar-guardiao',
    name: 'Colar Guardião',
    slot: 'pescoco',
    tier: 'maior',
    priceTibar: 51000,
    effect:
      'Diamante lapidado preso em corrente de platina deflete ataques. +5 na Defesa.',
    requiresClass: null,
    requiresAttunement: false,
    bookPage: 343,
  },
  {
    id: 'anel-da-liberdade',
    name: 'Anel da Liberdade',
    slot: 'anel',
    tier: 'maior',
    priceTibar: 60000,
    effect:
      'Forjado em ouro, relíquia da Igreja de Valkaria. Fica permanentemente sob efeito de Libertação.',
    requiresClass: null,
    requiresAttunement: false,
    bookPage: 342,
  },
  {
    id: 'tapete-voador',
    name: 'Tapete Voador',
    slot: 'outro',
    tier: 'maior',
    priceTibar: 60000,
    effect:
      'Com um comando, este tapete flutua e fornece deslocamento de voo 12m. Tem 3m x 3m e carrega seis criaturas Médias (ou 120 espaços). Se estiver em alcance curto ao longo do tapete, pode comandar o voo.',
    requiresClass: null,
    requiresAttunement: false,
    bookPage: 345,
  },
  {
    id: 'braceletes-de-ouro',
    name: 'Braceletes de Ouro',
    slot: 'maos',
    tier: 'maior',
    priceTibar: 64500,
    effect:
      '+8 na Defesa (campo de força invisível, tangível; cumulativo com itens mágicos, mas não com armaduras). Não cumulativo com Braceletes de Bronze.',
    requiresClass: null,
    requiresAttunement: false,
    bookPage: 342,
  },
  {
    id: 'robe-do-arquimago',
    name: 'Robe do Arquimago',
    slot: 'corpo',
    tier: 'maior',
    priceTibar: 90000,
    effect:
      'Traje pesado que alinha-se com energias arcanas emitidas pelo usuário. Se conjurador arcano, recebe bônus na Defesa = 5 + círculo mais alto que puder lançar; bônus em testes de resistência = metade do bônus na Defesa. Arcanista de 9º nível (3º círculo) → +8 Defesa e +4 resistência.',
    requiresClass: 'Conjurador arcano',
    requiresAttunement: false,
    bookPage: 345,
  },
  {
    id: 'anel-da-regeneracao',
    name: 'Anel da Regeneração',
    slot: 'anel',
    tier: 'maior',
    priceTibar: 150000,
    effect: 'Cura Acelerada 5 (somente após um dia de uso).',
    requiresClass: null,
    requiresAttunement: false,
    bookPage: 342,
  },
])

const byId = new Map(WONDROUS_ITEMS.map((i) => [i.id, i]))

export function wondrousItemById(id: string): WondrousItem | undefined {
  return byId.get(id)
}

export function wondrousItemsBySlot(
  slot: WondrousSlot,
): readonly WondrousItem[] {
  return WONDROUS_ITEMS.filter((i) => i.slot === slot)
}

export function wondrousItemsByTier(
  tier: SpecificItemTier,
): readonly WondrousItem[] {
  return WONDROUS_ITEMS.filter((i) => i.tier === tier)
}
