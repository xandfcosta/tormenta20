import type { CatalogItem } from '../types'

/**
 * Veículos (PDF p163). Tracked in catalog for purchase/ownership; no equip
 * or modifier integration. Vehicle combat stats (deslocamento, PV, capacidade)
 * are handled by GM narrative, not by the character engine.
 */
export const VEHICLES: CatalogItem[] = [
  {
    id: 'balao-goblin',
    name: 'Balão goblin',
    category: 'vehicle',
    price: 200,
    slots: 0,
    equip: 'either',
    modifiers: [],
  },
  {
    id: 'carroca',
    name: 'Carroça',
    category: 'vehicle',
    price: 150,
    slots: 0,
    equip: 'either',
    modifiers: [],
  },
  {
    id: 'carruagem',
    name: 'Carruagem',
    category: 'vehicle',
    price: 500,
    slots: 0,
    equip: 'either',
    modifiers: [],
  },
  {
    id: 'canoa',
    name: 'Canoa',
    category: 'vehicle',
    price: 70,
    slots: 0,
    equip: 'either',
    modifiers: [],
  },
  {
    id: 'veleiro',
    name: 'Veleiro',
    category: 'vehicle',
    price: 10000,
    slots: 0,
    equip: 'either',
    modifiers: [],
  },
]
