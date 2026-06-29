import type { CatalogItem } from '../types'

/**
 * Mundane adventuring gear + ferramentas + rações novas — PDF book
 * Cap 3 (Tabela 3-6, p155-157; alimentação p162). Complements (does
 * NOT duplicate) the existing apparel/consumables/meal catalogs:
 *
 *  - Existing `APPAREL` already holds focos arcanos, trajes,
 *    mochila-aventureiro, luneta, símbolo sagrado, maleta-medicamentos,
 *    coleccao-de-livros, bandoleira-pocoes.
 *  - Existing `CONSUMABLES` already holds os alquímicos preparados
 *    (ácido, bálsamo restaurador, fogo alquímico, etc.) e os pratos
 *    especiais (prato-aventureiro, batata-valkariana, etc.).
 *
 * This file fills the **gaps** the book lists in Cap 3 that were
 * missing: corda, tocha, lampião, barraca, saco de dormir, gazua,
 * sela, instrumento musical, equipamento de viagem, etc., plus água
 * benta, óleo, ração de viagem, refeição comum.
 *
 * Category mapping:
 *  - Adventuring gear / ferramentas → `apparel`
 *  - Água benta + óleo → `consumable`
 *  - Rações / refeição comum → `meal`
 */
export const GEAR: CatalogItem[] = [
  // ─── Equipamento de aventura ────────────────────────────────────
  {
    id: 'algemas',
    name: 'Algemas',
    category: 'apparel',
    price: 15,
    slots: 1,
    equip: 'either',
    modifiers: [],
  },
  {
    id: 'arpeu',
    name: 'Arpéu',
    category: 'apparel',
    price: 5,
    slots: 1,
    equip: 'either',
    modifiers: [],
  },
  {
    id: 'barraca',
    name: 'Barraca',
    category: 'apparel',
    price: 10,
    slots: 1,
    equip: 'either',
    modifiers: [],
  },
  {
    id: 'corda',
    name: 'Corda (10 m)',
    category: 'apparel',
    price: 1,
    slots: 1,
    equip: 'either',
    modifiers: [],
  },
  {
    id: 'espelho',
    name: 'Espelho',
    category: 'apparel',
    price: 10,
    slots: 1,
    equip: 'either',
    modifiers: [],
  },
  {
    id: 'lampiao',
    name: 'Lampião',
    category: 'apparel',
    price: 7,
    slots: 1,
    equip: 'either',
    modifiers: [],
  },
  {
    id: 'mochila',
    name: 'Mochila',
    category: 'apparel',
    price: 2,
    slots: 0,
    equip: 'vested',
    modifiers: [],
  },
  {
    id: 'organizador-de-pergaminhos',
    name: 'Organizador de Pergaminhos',
    category: 'apparel',
    price: 25,
    slots: 1,
    equip: 'vested',
    modifiers: [],
  },
  {
    id: 'pe-de-cabra',
    name: 'Pé de Cabra',
    category: 'apparel',
    price: 2,
    slots: 1,
    equip: 'either',
    modifiers: [],
  },
  {
    id: 'saco-de-dormir',
    name: 'Saco de Dormir',
    category: 'apparel',
    price: 1,
    slots: 1,
    equip: 'either',
    modifiers: [],
  },
  {
    id: 'tocha',
    name: 'Tocha',
    category: 'apparel',
    price: 1,
    slots: 1,
    equip: 'either',
    modifiers: [],
  },
  {
    id: 'vara-de-madeira',
    name: 'Vara de Madeira (3 m)',
    category: 'apparel',
    price: 1,
    slots: 1,
    equip: 'either',
    modifiers: [],
  },

  // ─── Ferramentas ────────────────────────────────────────────────
  {
    id: 'equipamento-de-viagem',
    name: 'Equipamento de Viagem',
    category: 'apparel',
    price: 10,
    slots: 1,
    equip: 'either',
    modifiers: [],
  },
  {
    id: 'estojo-de-disfarces',
    name: 'Estojo de Disfarces',
    category: 'apparel',
    price: 50,
    slots: 1,
    equip: 'either',
    modifiers: [],
  },
  {
    id: 'gazua',
    name: 'Gazua',
    category: 'apparel',
    price: 5,
    slots: 1,
    equip: 'either',
    modifiers: [],
  },
  {
    id: 'instrumentos-de-oficio',
    name: 'Instrumentos de Ofício',
    category: 'apparel',
    price: 30,
    slots: 1,
    equip: 'either',
    modifiers: [],
  },
  {
    id: 'instrumento-musical',
    name: 'Instrumento Musical',
    category: 'apparel',
    price: 35,
    slots: 1,
    equip: 'either',
    modifiers: [],
  },
  {
    id: 'sela',
    name: 'Sela',
    category: 'apparel',
    price: 20,
    slots: 1,
    equip: 'either',
    modifiers: [],
  },

  // ─── Consumíveis novos ──────────────────────────────────────────
  {
    id: 'agua-benta',
    name: 'Água benta',
    category: 'consumable',
    price: 10,
    slots: 0.5,
    equip: 'either',
    modifiers: [],
    consumable: { scope: 'instant' },
  },
  {
    id: 'oleo',
    name: 'Óleo',
    category: 'consumable',
    price: 1,
    slots: 0.5,
    equip: 'either',
    modifiers: [],
    consumable: { scope: 'instant' },
  },

  // ─── Rações ─────────────────────────────────────────────────────
  {
    id: 'racao-de-viagem',
    name: 'Ração de Viagem',
    category: 'meal',
    price: 1,
    slots: 0.5,
    equip: 'either',
    modifiers: [],
    consumable: { scope: 'instant' },
  },
  {
    id: 'refeicao-comum',
    name: 'Refeição Comum',
    category: 'meal',
    price: 1,
    slots: 0.5,
    equip: 'either',
    modifiers: [],
    consumable: { scope: 'instant' },
  },
]
