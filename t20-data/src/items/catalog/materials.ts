import type { CatalogItem } from '../types'

/**
 * Materiais especiais from T20 p162. Each is an overlay attached to a base
 * item via CharacterItem.material. Mechanical effects only — narrative
 * material descriptions live in the rules text.
 */
export const MATERIALS: CatalogItem[] = [
  {
    id: 'material-aco-rubi',
    name: 'Aço-rubi',
    category: 'material',
    price: 1000,
    slots: 0,
    equip: 'either',
    appliesTo: ['weapon'],
    modifiers: [
      {
        target: { k: 'damage', scope: 'this' },
        amount: 2,
        bonusType: 'enhancement',
        condition: { c: 'wielded' },
        note: '+2 dano vs criaturas vivas',
      },
    ],
  },
  {
    id: 'material-adamante',
    name: 'Adamante',
    category: 'material',
    price: 5000,
    slots: 0,
    equip: 'either',
    appliesTo: ['weapon', 'armor', 'shield'],
    modifiers: [
      {
        target: { k: 'attack', scope: 'this' },
        amount: 0,
        bonusType: 'untyped',
        note: 'ignora redução de dano não-mágica',
      },
    ],
  },
  {
    id: 'material-mitral',
    name: 'Mitral',
    category: 'material',
    price: 1000,
    slots: 0,
    equip: 'either',
    appliesTo: ['armor', 'shield'],
    modifiers: [
      {
        target: { k: 'armorPenalty' },
        amount: 2,
        bonusType: 'untyped',
        condition: { c: 'vested' },
        note: 'reduz penalidade em 2',
      },
    ],
  },
  {
    id: 'material-gelo-eterno',
    name: 'Gelo eterno',
    category: 'material',
    price: 800,
    slots: 0,
    equip: 'either',
    appliesTo: ['weapon'],
    modifiers: [
      {
        target: { k: 'damage', scope: 'this' },
        amount: 1,
        bonusType: 'enhancement',
        condition: { c: 'wielded' },
        note: '+1d6 frio',
      },
    ],
  },
  {
    id: 'material-madeira-tollon',
    name: 'Madeira tollon',
    category: 'material',
    price: 600,
    slots: 0,
    equip: 'either',
    appliesTo: ['weapon', 'shield'],
    modifiers: [],
  },
  {
    id: 'material-materia-vermelha',
    name: 'Matéria vermelha',
    category: 'material',
    price: 1500,
    slots: 0,
    equip: 'either',
    appliesTo: ['weapon', 'armor', 'shield'],
    modifiers: [],
  },
]
