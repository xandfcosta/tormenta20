import type { CatalogItem } from '../types'

export const SHIELDS: CatalogItem[] = [
  {
    id: 'escudo-leve',
    name: 'Escudo leve',
    category: 'shield',
    price: 5,
    slots: 1,
    equip: 'wielded',
    hands: 1,
    shield: { defense: 1, penalty: -1, heavy: false },
    modifiers: [
      { target: { k: 'defense' }, amount: 1, bonusType: 'armor', condition: { c: 'wielded' } },
      { target: { k: 'armorPenalty' }, amount: -1, bonusType: 'untyped', condition: { c: 'wielded' } },
    ],
  },
  {
    id: 'escudo-pesado',
    name: 'Escudo pesado',
    category: 'shield',
    price: 15,
    slots: 2,
    equip: 'wielded',
    hands: 1,
    shield: { defense: 2, penalty: -2, heavy: true },
    modifiers: [
      { target: { k: 'defense' }, amount: 2, bonusType: 'armor', condition: { c: 'wielded' } },
      { target: { k: 'armorPenalty' }, amount: -2, bonusType: 'untyped', condition: { c: 'wielded' } },
    ],
  },
]
