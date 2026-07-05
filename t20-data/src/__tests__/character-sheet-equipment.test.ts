import { describe, expect, it } from 'vitest'
import { computeCharacterSheet } from '../character-sheet'

/**
 * v3 orchestrator — equipamento (armor, shield, mainHand, offHand):
 *   - Defesa completa (10 + DES + armor + shield)
 *   - Penalidade de armadura derivada de armor+shield e aplicada aos skills
 *   - Ataques por arma equipada (Luta/Pontaria + dano por atributo)
 */

const BASE_ATTR = {
  strength: 3,
  dexterity: 2,
  constitution: 2,
  intelligence: 0,
  wisdom: 1,
  charisma: 0,
} as const

describe('Defesa com armadura + escudo', () => {
  const sheet = computeCharacterSheet({
    level: 5,
    className: 'Guerreiro',
    baseAttributes: BASE_ATTR,
    equipment: {
      armor: { name: 'cota-de-malha', defense: 5, penalty: 5 },
      shield: { name: 'escudo-pesado', defense: 3, penalty: 2 },
    },
  })

  it('Defesa = 10 + DES 2 + armor 5 + shield 3 = 20', () => {
    expect(sheet.defense.total).toBe(20)
    expect(sheet.defense.armor).toBe(5)
    expect(sheet.defense.shield).toBe(3)
  })

  it('penalidade derivada = 5 + 2 = 7 aplicada em Furtividade', () => {
    // Furtividade DES 2 sem treino = 2, com pen 7 = -5
    expect(sheet.skills.furtividade.armorPenaltyApplied).toBe(7)
    expect(sheet.skills.furtividade.total).toBe(-5)
  })

  it('penalidade não afeta Iniciativa', () => {
    expect(sheet.skills.iniciativa.armorPenaltyApplied).toBe(0)
    expect(sheet.skills.iniciativa.total).toBe(2)
  })
})

describe('Defesa sem armor (nem shield)', () => {
  const sheet = computeCharacterSheet({
    level: 1,
    className: 'Arcanista',
    baseAttributes: BASE_ATTR,
  })

  it('armor=0, shield=0, total = 10 + DES 2 = 12', () => {
    expect(sheet.defense.armor).toBe(0)
    expect(sheet.defense.shield).toBe(0)
    expect(sheet.defense.total).toBe(12)
  })
})

describe('armorPenalty override (v2 API) ignorada se equipment presente', () => {
  const sheet = computeCharacterSheet({
    level: 5,
    className: 'Guerreiro',
    baseAttributes: BASE_ATTR,
    armorPenalty: 99, // ignorada
    equipment: {
      armor: { defense: 4, penalty: 3 },
    },
  })

  it('penalidade = 3 (não 99)', () => {
    expect(sheet.skills.furtividade.armorPenaltyApplied).toBe(3)
  })
})

describe('armorPenalty override (v2 API) usada se equipment ausente', () => {
  const sheet = computeCharacterSheet({
    level: 5,
    className: 'Guerreiro',
    baseAttributes: BASE_ATTR,
    armorPenalty: 4,
  })

  it('penalidade = 4', () => {
    expect(sheet.skills.furtividade.armorPenaltyApplied).toBe(4)
  })
})

describe('Ataque corpo a corpo — arma em mainHand', () => {
  const sheet = computeCharacterSheet({
    level: 5,
    className: 'Guerreiro',
    baseAttributes: BASE_ATTR,
    trainedSkills: ['luta'],
    equipment: {
      mainHand: {
        name: 'espada-longa',
        hand: 'one',
        purpose: 'melee',
        damage: '1d8',
        critRange: 19,
        critMult: 2,
        damageType: 'corte',
      },
    },
  })

  it('mainHand attackTotal = valor de Luta treinada = 2 (½5) + 3 (FOR) + 2 = 7', () => {
    expect(sheet.attacks.mainHand?.attackTotal).toBe(7)
    expect(sheet.attacks.mainHand?.skill).toBe('luta')
  })

  it('mainHand damageAttributeBonus = FOR 3', () => {
    expect(sheet.attacks.mainHand?.damageAttributeBonus).toBe(3)
  })

  it('offHand = null quando ausente', () => {
    expect(sheet.attacks.offHand).toBeNull()
  })

  it('preserva critRange/critMult e nome', () => {
    expect(sheet.attacks.mainHand?.critRange).toBe(19)
    expect(sheet.attacks.mainHand?.critMult).toBe(2)
    expect(sheet.attacks.mainHand?.weaponName).toBe('espada-longa')
  })
})

describe('Ataque à distância — arma ranged em mainHand', () => {
  const sheet = computeCharacterSheet({
    level: 5,
    className: 'Caçador',
    baseAttributes: BASE_ATTR,
    trainedSkills: ['pontaria'],
    equipment: {
      mainHand: {
        name: 'arco-longo',
        hand: 'two',
        purpose: 'ranged',
        damage: '1d8',
        critRange: 20,
        critMult: 3,
        damageType: 'perfuracao',
      },
    },
  })

  it('mainHand attackTotal usa Pontaria treinada L5 (½5=2 mín+1=2, +2 treino) + DES 2 = 6', () => {
    expect(sheet.attacks.mainHand?.attackTotal).toBe(6)
    expect(sheet.attacks.mainHand?.skill).toBe('pontaria')
  })

  it('ranged não soma FOR no dano (bônus = 0)', () => {
    expect(sheet.attacks.mainHand?.damageAttributeBonus).toBe(0)
  })
})

describe('Ataque arremesso (thrown) — soma FOR como corpo a corpo', () => {
  const sheet = computeCharacterSheet({
    level: 5,
    className: 'Bárbaro',
    baseAttributes: BASE_ATTR,
    trainedSkills: ['pontaria'],
    equipment: {
      mainHand: {
        name: 'lança-de-arremesso',
        hand: 'one',
        purpose: 'thrown',
        damage: '1d6',
        critRange: 20,
        critMult: 2,
        damageType: 'perfuracao',
      },
    },
  })

  it('thrown usa Luta ou Pontaria? T20 = Pontaria mas soma FOR no dano', () => {
    // Nossa implementação: purpose ranged → pontaria; thrown → luta.
    // Se book diverge, ajustar. Por ora: thrown = luta.
    expect(sheet.attacks.mainHand?.skill).toBe('luta')
    expect(sheet.attacks.mainHand?.damageAttributeBonus).toBe(3) // FOR
  })
})

describe('Ataque com duas armas (mainHand + offHand)', () => {
  const sheet = computeCharacterSheet({
    level: 3,
    className: 'Ladino',
    baseAttributes: BASE_ATTR,
    trainedSkills: ['luta'],
    equipment: {
      mainHand: {
        name: 'espada-curta',
        hand: 'light',
        purpose: 'melee',
        damage: '1d6',
        critRange: 19,
        critMult: 2,
        damageType: 'perfuracao',
      },
      offHand: {
        name: 'adaga',
        hand: 'light',
        purpose: 'melee',
        damage: '1d4',
        critRange: 19,
        critMult: 2,
        damageType: 'perfuracao',
      },
    },
  })

  it('ambas as mãos populam ComputedAttack', () => {
    expect(sheet.attacks.mainHand?.weaponName).toBe('espada-curta')
    expect(sheet.attacks.offHand?.weaponName).toBe('adaga')
  })
})
