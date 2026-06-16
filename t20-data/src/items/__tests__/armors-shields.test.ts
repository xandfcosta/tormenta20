import { describe, expect, it } from 'vitest'
import { ARMORS } from '../catalog/armors'
import { SHIELDS } from '../catalog/shields'

/**
 * PDF Cap 3 (Equipamento, Tabela 3-4: Armaduras e Escudos). Values pinned:
 *
 * Armaduras Leves
 *   Acolchoada     T$5    Def +1  Pen  0
 *   De couro       T$20   Def +2  Pen  0
 *   Couro batido   T$35   Def +3  Pen -1
 *   Gibão de peles T$25   Def +4  Pen -3
 *   Couraça        T$500  Def +5  Pen -4
 *
 * Armaduras Pesadas
 *   Brunea               T$50    Def +5  Pen -2
 *   Cota de malha        T$150   Def +6  Pen -2
 *   Loriga segmentada    T$250   Def +7  Pen -3
 *   Meia armadura        T$600   Def +8  Pen -4
 *   Armadura completa    T$3000  Def +10 Pen -5
 *
 * Escudos
 *   Escudo leve    T$5   Def +1  Pen -1
 *   Escudo pesado  T$15  Def +2  Pen -2
 */

type ArmorRow = {
  price: number
  defense: number
  penalty: number
  heavy: boolean
}

const BOOK_ARMORS: Record<string, ArmorRow> = {
  'armadura-acolchoada': { price: 5, defense: 1, penalty: 0, heavy: false },
  'armadura-couro': { price: 20, defense: 2, penalty: 0, heavy: false },
  'couro-batido': { price: 35, defense: 3, penalty: -1, heavy: false },
  'gibao-peles': { price: 25, defense: 4, penalty: -3, heavy: false },
  couraca: { price: 500, defense: 5, penalty: -4, heavy: false },
  brunea: { price: 50, defense: 5, penalty: -2, heavy: true },
  'cota-malha': { price: 150, defense: 6, penalty: -2, heavy: true },
  'loriga-segmentada': { price: 250, defense: 7, penalty: -3, heavy: true },
  'meia-armadura': { price: 600, defense: 8, penalty: -4, heavy: true },
  'armadura-completa': { price: 3000, defense: 10, penalty: -5, heavy: true },
}

const BOOK_SHIELDS: Record<string, ArmorRow> = {
  'escudo-leve': { price: 5, defense: 1, penalty: -1, heavy: false },
  'escudo-pesado': { price: 15, defense: 2, penalty: -2, heavy: true },
}

describe('ARMORS vs PDF Tabela 3-4', () => {
  it('catalog contains 5 light + 5 heavy armors', () => {
    const light = ARMORS.filter((a) => a.category === 'armor-light')
    const heavy = ARMORS.filter((a) => a.category === 'armor-heavy')
    expect(light).toHaveLength(5)
    expect(heavy).toHaveLength(5)
  })

  for (const [id, row] of Object.entries(BOOK_ARMORS)) {
    it(`${id}: price/defense/penalty/heavy match the book`, () => {
      const armor = ARMORS.find((a) => a.id === id)
      expect(armor, `${id} missing from catalog`).toBeDefined()
      expect(armor!.price).toBe(row.price)
      expect(armor!.armor!.defense).toBe(row.defense)
      expect(armor!.armor!.penalty).toBe(row.penalty)
      expect(armor!.armor!.heavy).toBe(row.heavy)
    })
  }
})

describe('SHIELDS vs PDF Tabela 3-4', () => {
  it('catalog contains exactly 2 shields', () => {
    expect(SHIELDS).toHaveLength(2)
  })

  for (const [id, row] of Object.entries(BOOK_SHIELDS)) {
    it(`${id}: price/defense/penalty/heavy match the book`, () => {
      const shield = SHIELDS.find((s) => s.id === id)
      expect(shield, `${id} missing from catalog`).toBeDefined()
      expect(shield!.price).toBe(row.price)
      expect(shield!.shield!.defense).toBe(row.defense)
      expect(shield!.shield!.penalty).toBe(row.penalty)
      expect(shield!.shield!.heavy).toBe(row.heavy)
    })
  }
})

describe('Armor catalog invariants', () => {
  it('every armor is vested', () => {
    for (const a of ARMORS) expect(a.equip).toBe('vested')
  })

  it('every armor carries an armor-defense modifier matching armor.defense', () => {
    for (const a of ARMORS) {
      const defMod = a.modifiers.find(
        (m) =>
          m.target.k === 'defense' &&
          m.bonusType === 'armor' &&
          m.condition?.c === 'vested',
      )
      expect(defMod, `${a.id} missing armor defense modifier`).toBeDefined()
      expect(defMod!.amount).toBe(a.armor!.defense)
    }
  })

  it('every heavy armor carries the cannot-apply-dex-to-defense flag', () => {
    const heavy = ARMORS.filter((a) => a.armor?.heavy)
    for (const a of heavy) {
      const flag = a.modifiers.find(
        (m) =>
          m.target.k === 'flag' &&
          m.target.name === 'cannot-apply-dex-to-defense',
      )
      expect(flag, `${a.id} heavy armor missing dex-cap flag`).toBeDefined()
    }
  })

  it('every heavy armor carries the fatigue-on-sleep flag', () => {
    const heavy = ARMORS.filter((a) => a.armor?.heavy)
    for (const a of heavy) {
      const flag = a.modifiers.find(
        (m) =>
          m.target.k === 'flag' && m.target.name === 'fatigue-on-sleep',
      )
      expect(flag, `${a.id} heavy armor missing sleep-fatigue flag`).toBeDefined()
    }
  })

  it('no light armor carries heavy-only flags', () => {
    const light = ARMORS.filter((a) => !a.armor?.heavy)
    for (const a of light) {
      const dexFlag = a.modifiers.find(
        (m) =>
          m.target.k === 'flag' &&
          m.target.name === 'cannot-apply-dex-to-defense',
      )
      expect(dexFlag).toBeUndefined()
    }
  })
})

describe('Shield catalog invariants', () => {
  it('every shield is wielded with one hand', () => {
    for (const s of SHIELDS) {
      expect(s.equip).toBe('wielded')
      expect(s.hands).toBe(1)
    }
  })

  it('shield Defense bonus is "armor"-typed (won’t stack with armor)', () => {
    for (const s of SHIELDS) {
      const defMod = s.modifiers.find((m) => m.target.k === 'defense')
      expect(defMod, `${s.id} missing defense modifier`).toBeDefined()
      expect(defMod!.bonusType).toBe('armor')
    }
  })
})
