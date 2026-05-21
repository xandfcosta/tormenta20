import type { CatalogItem } from '../types'

/**
 * Animais (PDF p157, p162). Cataloged so the player can buy/track partners
 * and mounts in the inventory list. No equip / modifier integration — the
 * parceiro system handles their combat stats separately. Slots = 0 because
 * an animal isn't carried like gear; alforje extends the mount's carrying
 * capacity but is itself a vested item, not modeled here.
 */
export const ANIMALS: CatalogItem[] = [
  {
    id: 'alforje',
    name: 'Alforje',
    category: 'animal',
    price: 30,
    slots: 0,
    equip: 'either',
    modifiers: [],
  },
  {
    id: 'cao-de-caca',
    name: 'Cão de caça',
    category: 'animal',
    price: 150,
    slots: 0,
    equip: 'either',
    modifiers: [],
  },
  {
    id: 'cavalo',
    name: 'Cavalo',
    category: 'animal',
    price: 75,
    slots: 0,
    equip: 'either',
    modifiers: [],
  },
  {
    id: 'cavalo-de-guerra',
    name: 'Cavalo de guerra',
    category: 'animal',
    price: 400,
    slots: 0,
    equip: 'either',
    modifiers: [],
  },
  {
    id: 'estabulo',
    name: 'Estábulo (por dia)',
    category: 'animal',
    price: 0.1,
    slots: 0,
    equip: 'either',
    modifiers: [],
  },
  {
    id: 'ponei',
    name: 'Pônei',
    category: 'animal',
    price: 5,
    slots: 0,
    equip: 'either',
    modifiers: [],
  },
  {
    id: 'ponei-de-guerra',
    name: 'Pônei de guerra',
    category: 'animal',
    price: 30,
    slots: 0,
    equip: 'either',
    modifiers: [],
  },
  {
    id: 'trobo',
    name: 'Trobo',
    category: 'animal',
    price: 60,
    slots: 0,
    equip: 'either',
    modifiers: [],
  },
]
