import { describe, expect, it } from 'vitest'
import { IMPROVEMENTS } from '../catalog/improvements'

/**
 * PDF Cap 3 — Melhorias (Tabela 3-8, p164-166). Catalog holds 28 melhorias
 * split across four `appliesTo` family groups:
 *   - weapon-only (10): Certeira/Pungente, Cruel/Atroz, Equilibrada,
 *     Harmonizada, Injeção alquímica, Maciça, Mira telescópica, Precisa
 *   - armor/shield shared (5): Ajustada/Sob medida, Polida, Reforçada (+
 *     one-side: Delicada armor-only, Espinhosa armor, Espinhoso shield,
 *     Selada armor)
 *   - apparel (6): Canalizador, Energético, Harmonizado, Poderoso,
 *     Vigilante, Aprimorado
 *   - all four (4): Banhado a ouro, Cravejado de gemas, Discreto, Macabro
 *
 * Numeric chains: Certeira (+1 atq, T$300) → Pungente (+2 atq, T$3000);
 * Cruel (+1 dano, T$300) → Atroz (+2 dano, T$3000). The catalog encodes
 * the *bonus*, not the prereq chain — UI/picker enforces ordering.
 */
describe('IMPROVEMENTS catalog scope', () => {
  it('catalog contains 28 melhorias', () => {
    expect(IMPROVEMENTS).toHaveLength(28)
  })

  it('every melhoria has category=improvement, 0 slots, equip=either', () => {
    for (const m of IMPROVEMENTS) {
      expect(m.category).toBe('improvement')
      expect(m.slots).toBe(0)
      expect(m.equip).toBe('either')
    }
  })

  it('every melhoria declares an appliesTo family list', () => {
    for (const m of IMPROVEMENTS) {
      expect(m.appliesTo, `${m.id} missing appliesTo`).toBeDefined()
      expect(m.appliesTo!.length).toBeGreaterThan(0)
    }
  })
})

describe('Melhoria chains — bonus + price progression', () => {
  it('Certeira: +1 atq enhancement at T$300', () => {
    const m = IMPROVEMENTS.find((x) => x.id === 'melhoria-certeira')!
    expect(m.price).toBe(300)
    const mod = m.modifiers[0]
    expect(mod.bonusType).toBe('enhancement')
    expect(mod.target).toEqual({ k: 'attack', scope: 'this' })
    expect(mod.amount).toBe(1)
  })

  it('Pungente: +2 atq enhancement at T$3000', () => {
    const m = IMPROVEMENTS.find((x) => x.id === 'melhoria-pungente')!
    expect(m.price).toBe(3000)
    expect(m.modifiers[0].amount).toBe(2)
  })

  it('Cruel: +1 dano enhancement at T$300', () => {
    const m = IMPROVEMENTS.find((x) => x.id === 'melhoria-cruel')!
    expect(m.price).toBe(300)
    expect(m.modifiers[0].target).toEqual({ k: 'damage', scope: 'this' })
    expect(m.modifiers[0].amount).toBe(1)
  })

  it('Atroz: +2 dano enhancement at T$3000', () => {
    const m = IMPROVEMENTS.find((x) => x.id === 'melhoria-atroz')!
    expect(m.price).toBe(3000)
    expect(m.modifiers[0].amount).toBe(2)
  })
})

describe('Melhoria family gating', () => {
  it('weapon-only melhorias don’t list armor/shield/apparel', () => {
    const weaponOnly = [
      'melhoria-certeira',
      'melhoria-pungente',
      'melhoria-cruel',
      'melhoria-atroz',
      'melhoria-precisa',
    ]
    for (const id of weaponOnly) {
      const m = IMPROVEMENTS.find((x) => x.id === id)!
      expect(m.appliesTo).toEqual(['weapon'])
    }
  })

  it('apparel-only melhorias don’t list weapon/armor/shield', () => {
    const apparelOnly = [
      'melhoria-canalizador',
      'melhoria-energetico',
      'melhoria-poderoso',
      'melhoria-vigilante',
      'melhoria-aprimorado',
    ]
    for (const id of apparelOnly) {
      const m = IMPROVEMENTS.find((x) => x.id === id)!
      expect(m.appliesTo).toEqual(['apparel'])
    }
  })

  it('"universal" melhorias attach to all 4 families', () => {
    const universals = [
      'melhoria-banhado-a-ouro',
      'melhoria-cravejado-de-gemas',
      'melhoria-discreto',
      'melhoria-macabro',
    ]
    for (const id of universals) {
      const m = IMPROVEMENTS.find((x) => x.id === id)!
      expect([...m.appliesTo!].sort()).toEqual(
        ['apparel', 'armor', 'shield', 'weapon'],
      )
    }
  })
})

describe('Melhoria modifier types', () => {
  it('combat-stat modifiers (attack/damage) are "enhancement"-typed', () => {
    for (const m of IMPROVEMENTS) {
      for (const mod of m.modifiers) {
        const isCombatStat =
          mod.target.k === 'attack' || mod.target.k === 'damage'
        if (isCombatStat) {
          expect(
            mod.bonusType,
            `${m.id}: attack/damage modifier should be enhancement`,
          ).toBe('enhancement')
        }
      }
    }
  })
})
