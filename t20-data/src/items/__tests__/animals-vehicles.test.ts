import { describe, expect, it } from 'vitest'
import { ANIMALS } from '../catalog/animals'
import { VEHICLES } from '../catalog/vehicles'

/**
 * PDF Cap 3 — Animais (p157, p162) and Veículos (p163). Both are tracked
 * purely as inventory: zero slots, no modifiers, no proficiency gating.
 * Animal/vehicle combat stats are handled by the parceiro system or GM
 * narrative respectively, not by the items engine.
 */
describe('ANIMALS vs PDF p157 + p162', () => {
  it('catalog contains 8 entries', () => {
    expect(ANIMALS).toHaveLength(8)
  })

  const BOOK_ANIMAL_PRICES: Record<string, number> = {
    alforje: 30,
    'cao-de-caca': 150,
    cavalo: 75,
    'cavalo-de-guerra': 400,
    estabulo: 0.1,
    ponei: 5,
    'ponei-de-guerra': 30,
    trobo: 60,
  }

  for (const [id, price] of Object.entries(BOOK_ANIMAL_PRICES)) {
    it(`${id}: T$${price}`, () => {
      const animal = ANIMALS.find((a) => a.id === id)
      expect(animal, `${id} missing`).toBeDefined()
      expect(animal!.price).toBe(price)
    })
  }
})

describe('VEHICLES vs PDF p163', () => {
  it('catalog contains 5 entries', () => {
    expect(VEHICLES).toHaveLength(5)
  })

  const BOOK_VEHICLE_PRICES: Record<string, number> = {
    'balao-goblin': 200,
    carroca: 150,
    carruagem: 500,
    canoa: 70,
    veleiro: 10000,
  }

  for (const [id, price] of Object.entries(BOOK_VEHICLE_PRICES)) {
    it(`${id}: T$${price}`, () => {
      const vehicle = VEHICLES.find((v) => v.id === id)
      expect(vehicle, `${id} missing`).toBeDefined()
      expect(vehicle!.price).toBe(price)
    })
  }
})

describe('ANIMALS + VEHICLES — inventory-only invariants', () => {
  it('animals occupy zero inventory slots (mount-borne)', () => {
    for (const a of ANIMALS) expect(a.slots).toBe(0)
  })

  it('vehicles occupy zero inventory slots (GM-narrative)', () => {
    for (const v of VEHICLES) expect(v.slots).toBe(0)
  })

  it('neither animals nor vehicles carry modifiers', () => {
    for (const a of ANIMALS) expect(a.modifiers).toEqual([])
    for (const v of VEHICLES) expect(v.modifiers).toEqual([])
  })

  it('every entry uses equip=either (free placement)', () => {
    for (const a of ANIMALS) expect(a.equip).toBe('either')
    for (const v of VEHICLES) expect(v.equip).toBe('either')
  })
})
