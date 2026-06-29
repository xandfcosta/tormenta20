import { describe, expect, it } from 'vitest'
import {
  INVESTIDA_ATTACK_BONUS,
  INVESTIDA_DEFENSE_PENALTY,
  attackAttributeBonus,
  damageTotal,
  initiativeOrder,
  isAttackHit,
  isCriticalHit,
  isLegalTurn,
} from '../combat'

/**
 * PDF book p230-235 — Combate.
 *
 * Pinned:
 *  - Ataque: d20 + bônus ≥ Defesa → acerta
 *  - Crit: raw d20 ≥ weapon margem (default 20); no confirm roll
 *  - Crit dano: weapon dice * mult + attr (NOT mult) + extras (NOT mult)
 *  - Dano: melee/arremesso + Força; disparo sem Força
 *  - Iniciativa tie-break: highest modifier; equal modifiers → reroll
 *  - Action economy: 1 padrão + 1 movimento; pode trocar padrão por
 *    movimento; ação completa consome ambos
 *  - Investida: +2 atk, -2 Defesa
 */
describe('isAttackHit — PDF p230', () => {
  it('hit when d20 + bonus equals Defesa', () => {
    expect(isAttackHit(10, 5, 15)).toBe(true)
  })

  it('hit when d20 + bonus exceeds Defesa', () => {
    expect(isAttackHit(15, 5, 18)).toBe(true)
  })

  it('miss when d20 + bonus is below Defesa', () => {
    expect(isAttackHit(5, 3, 20)).toBe(false)
  })

  it('rejects invalid d20 values (defensive)', () => {
    expect(isAttackHit(0, 100, 1)).toBe(false)
    expect(isAttackHit(21, 100, 1)).toBe(false)
    expect(isAttackHit(-3, 100, 1)).toBe(false)
  })
})

describe('isCriticalHit — PDF p231 (no confirmation roll)', () => {
  it('natural 20 with weapon critRange 20 + hit → crit', () => {
    expect(isCriticalHit(20, 20, true)).toBe(true)
  })

  it('natural 18 with weapon critRange 18 + hit → crit (margem)', () => {
    expect(isCriticalHit(18, 18, true)).toBe(true)
  })

  it('natural 19 with weapon critRange 18 + hit → crit (within range)', () => {
    expect(isCriticalHit(19, 18, true)).toBe(true)
  })

  it('natural 17 with weapon critRange 18 → no crit (below range)', () => {
    expect(isCriticalHit(17, 18, true)).toBe(false)
  })

  it('crit-range hit but the attack missed → no crit', () => {
    // Even rolling a natural 20, if the total still doesn't beat Defesa
    // (impossible per PDF since 20 always hits — but defensive).
    expect(isCriticalHit(20, 20, false)).toBe(false)
  })
})

describe('damageTotal — PDF p231 crit math', () => {
  it('normal hit: weapon dice + attribute + extra', () => {
    // weapon 5 + STR 3 + sneak 2 = 10
    expect(
      damageTotal({
        weaponDice: 5,
        attributeBonus: 3,
        extraDice: 2,
        isCrit: false,
        critMultiplier: 2,
      }),
    ).toBe(10)
  })

  it('crit x2: doubles ONLY weapon dice', () => {
    // (weapon 5 × 2) + STR 3 + sneak 2 = 15
    expect(
      damageTotal({
        weaponDice: 5,
        attributeBonus: 3,
        extraDice: 2,
        isCrit: true,
        critMultiplier: 2,
      }),
    ).toBe(15)
  })

  it('crit x3: triples ONLY weapon dice, attribute and extras stay flat', () => {
    // (weapon 6 × 3) + STR 4 + extras 0 = 22
    expect(
      damageTotal({
        weaponDice: 6,
        attributeBonus: 4,
        extraDice: 0,
        isCrit: true,
        critMultiplier: 3,
      }),
    ).toBe(22)
  })

  it('clamps damage at 0 (negative bonuses don\'t over-mitigate)', () => {
    expect(
      damageTotal({
        weaponDice: 2,
        attributeBonus: -5,
        extraDice: 0,
        isCrit: false,
        critMultiplier: 2,
      }),
    ).toBe(0)
  })
})

describe('attackAttributeBonus — PDF p230', () => {
  it('melee adds Força', () => {
    expect(attackAttributeBonus('melee', 4)).toBe(4)
  })

  it('thrown adds Força (same as melee)', () => {
    expect(attackAttributeBonus('thrown', 4)).toBe(4)
  })

  it('ranged does NOT add Força (disparo: dano da arma apenas)', () => {
    expect(attackAttributeBonus('ranged', 4)).toBe(0)
  })

  it('passes through negative Força (penalty applies to melee/thrown)', () => {
    expect(attackAttributeBonus('melee', -2)).toBe(-2)
    expect(attackAttributeBonus('ranged', -2)).toBe(0)
  })
})

describe('initiativeOrder — PDF p231 tie-break', () => {
  it('sorts higher total first', () => {
    const order = initiativeOrder([
      { name: 'a', total: 12, modifier: 3 },
      { name: 'b', total: 18, modifier: 1 },
      { name: 'c', total: 9, modifier: 5 },
    ])
    expect(order.map((e) => e.name)).toEqual(['b', 'a', 'c'])
  })

  it('on equal totals, higher modifier acts first', () => {
    const order = initiativeOrder([
      { name: 'a', total: 15, modifier: 2 },
      { name: 'b', total: 15, modifier: 5 },
    ])
    expect(order.map((e) => e.name)).toEqual(['b', 'a'])
  })

  it('on equal totals AND equal modifiers, preserves input order (caller rerolls)', () => {
    // PDF: "Se o empate persistir, eles fazem um novo teste de Iniciativa
    // entre si." The function is pure — caller is responsible for the reroll.
    const order = initiativeOrder([
      { name: 'a', total: 15, modifier: 2 },
      { name: 'b', total: 15, modifier: 2 },
    ])
    expect(order.map((e) => e.name)).toEqual(['a', 'b'])
  })
})

describe('Investida constants — PDF p235', () => {
  it('attack bonus is +2', () => {
    expect(INVESTIDA_ATTACK_BONUS).toBe(2)
  })

  it('defense penalty is -2 (guarda aberta)', () => {
    expect(INVESTIDA_DEFENSE_PENALTY).toBe(-2)
  })
})

describe('isLegalTurn — PDF p233 action economy', () => {
  it('1 padrão + 1 movimento is legal (default)', () => {
    expect(isLegalTurn({ padrao: 1, movimento: 1, completa: 0 })).toBe(true)
  })

  it('0 padrão + 2 movimento is legal (trade-down)', () => {
    expect(isLegalTurn({ padrao: 0, movimento: 2, completa: 0 })).toBe(true)
  })

  it('1 padrão + 2 movimento is ILLEGAL (no inverse trade)', () => {
    expect(isLegalTurn({ padrao: 1, movimento: 2, completa: 0 })).toBe(false)
  })

  it('2 padrão + 0 movimento is ILLEGAL', () => {
    expect(isLegalTurn({ padrao: 2, movimento: 0, completa: 0 })).toBe(false)
  })

  it('ação completa consumes the entire padrão + movimento budget', () => {
    expect(isLegalTurn({ padrao: 0, movimento: 0, completa: 1 })).toBe(true)
    expect(isLegalTurn({ padrao: 1, movimento: 0, completa: 1 })).toBe(false)
    expect(isLegalTurn({ padrao: 0, movimento: 1, completa: 1 })).toBe(false)
  })

  it('two ações completas is illegal', () => {
    expect(isLegalTurn({ padrao: 0, movimento: 0, completa: 2 })).toBe(false)
  })

  it('rejects negative counts (defensive)', () => {
    expect(isLegalTurn({ padrao: -1, movimento: 0, completa: 0 })).toBe(false)
  })

  it('an empty turn (no actions used) is legal — character delays / waits', () => {
    expect(isLegalTurn({ padrao: 0, movimento: 0, completa: 0 })).toBe(true)
  })
})
