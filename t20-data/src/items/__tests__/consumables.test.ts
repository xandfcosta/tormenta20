import { describe, expect, it } from 'vitest'
import { CONSUMABLES } from '../catalog/consumables'

/**
 * PDF Cap 3 — Consumíveis (poções p159, alquímicos p160-161, pratos
 * especiais p168). Catalog mixes four families under the `consumable`
 * field:
 *
 *  - poções (Bálsamo, Essência de mana, Cosmético) — instant HP/MP or
 *    scene-scoped attribute boost
 *  - alquímicos preparados (Ácido, Bomba, Elixir do amor, Fogo
 *    alquímico, Pó do desaparecimento) — instant, GM resolves throw
 *  - catalisadores (12 entries, p160-161) — instant decrement,
 *    spell-engine integration deferred (see memory)
 *  - venenos (Beladona, Cicuta, Peçonha, …) — instant decrement
 *  - pratos especiais (Gorad, Macarrão, Batata, Prato, Sopa) — day
 *    scope with oncePerDay flag
 */

describe('CONSUMABLES — consumable spec invariants', () => {
  it('every entry defines a consumable spec', () => {
    for (const c of CONSUMABLES) {
      expect(c.consumable, `${c.id} missing consumable`).toBeDefined()
    }
  })

  it('scope is one of instant/scene/day', () => {
    for (const c of CONSUMABLES) {
      expect(['instant', 'scene', 'day']).toContain(c.consumable!.scope)
    }
  })

  it('only meal-scoped pratos use oncePerDay', () => {
    for (const c of CONSUMABLES) {
      if (c.consumable!.oncePerDay) {
        expect(c.consumable!.scope).toBe('day')
        expect(c.category).toBe('meal')
      }
    }
  })

  it('day-scoped entries are all meals with oncePerDay=true', () => {
    const dayEntries = CONSUMABLES.filter(
      (c) => c.consumable!.scope === 'day',
    )
    for (const c of dayEntries) {
      expect(c.consumable!.oncePerDay).toBe(true)
      expect(c.category).toBe('meal')
    }
  })
})

describe('CONSUMABLES — known canonical entries', () => {
  it('Bálsamo restaurador: 2d4 HP instant heal (p159)', () => {
    const b = CONSUMABLES.find((c) => c.id === 'balsamo-restaurador')!
    expect(b.price).toBe(25)
    expect(b.consumable!.scope).toBe('instant')
    expect(b.consumable!.instant?.hp?.dice).toBe('2d4')
  })

  it('Essência de mana: 1d4 PM instant restore (p159)', () => {
    const e = CONSUMABLES.find((c) => c.id === 'essencia-de-mana')!
    expect(e.price).toBe(50)
    expect(e.consumable!.instant?.mp?.dice).toBe('1d4')
  })

  it('Cosmético: scene-scope +2 untyped Carisma (p159)', () => {
    const c = CONSUMABLES.find((x) => x.id === 'cosmetico')!
    expect(c.consumable!.scope).toBe('scene')
    const mod = c.consumable!.modifiers![0]
    expect(mod.target).toEqual({ k: 'attribute', name: 'charisma' })
    expect(mod.amount).toBe(2)
    expect(mod.bonusType).toBe('untyped')
  })

  it('catalisadores ship without spell-engine wiring (modifiers + instant empty)', () => {
    // Per the spell-engine-deferred design: catalysts decrement stock on
    // use but don't auto-trigger any spell or stat change. They may carry
    // any scope label (e.g., baga-de-fogo is scene-scoped) — what matters
    // is that no modifiers or instant vitals patch is attached yet.
    const catalysts = CONSUMABLES.filter((c) => c.category === 'catalyst')
    expect(catalysts.length).toBeGreaterThan(0)
    for (const c of catalysts) {
      expect(c.consumable!.modifiers?.length ?? 0).toBe(0)
      expect(c.consumable!.instant).toBeUndefined()
    }
  })

  it('venenos catalog covers the 10 entries from p161', () => {
    // Beladona, Bruma sonolenta, Cicuta, Essência de sombra, Névoa
    // tóxica, Peçonha (comum/concentrada/potente), Pó de lich, Riso
    // de Nimb.
    const venenoIds = [
      'beladona',
      'bruma-sonolenta',
      'cicuta',
      'essencia-sombra',
      'nevoa-toxica',
      'peconha-comum',
      'peconha-concentrada',
      'peconha-potente',
      'po-lich',
      'riso-nimb',
    ]
    for (const id of venenoIds) {
      const v = CONSUMABLES.find((c) => c.id === id)
      expect(v, `${id} missing`).toBeDefined()
      expect(v!.category).toBe('consumable')
      expect(v!.consumable!.scope).toBe('instant')
    }
  })

  it('pratos especiais: 5 entries, all day-scope + oncePerDay (p168)', () => {
    const pratosIds = [
      'gorad-quente',
      'macarrao-de-yuvalin',
      'batata-valkariana',
      'prato-aventureiro',
      'sopa-peixe',
    ]
    for (const id of pratosIds) {
      const p = CONSUMABLES.find((c) => c.id === id)
      expect(p, `${id} missing`).toBeDefined()
      expect(p!.category).toBe('meal')
      expect(p!.consumable!.scope).toBe('day')
      expect(p!.consumable!.oncePerDay).toBe(true)
    }
  })
})

describe('CONSUMABLES — physical-carry invariants', () => {
  it('every consumable occupies ≤ 1 slot (small/light per PDF p141)', () => {
    for (const c of CONSUMABLES) {
      expect(c.slots).toBeLessThanOrEqual(1)
    }
  })

  it('every consumable can be carried either vested or wielded (equip=either)', () => {
    for (const c of CONSUMABLES) {
      expect(c.equip).toBe('either')
    }
  })
})
