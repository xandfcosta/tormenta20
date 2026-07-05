import { describe, expect, it } from 'vitest'
import {
  computeCharacterSheet,
  effectTargetKey,
  stackModifiersForTarget,
} from '../character-sheet'

/**
 * v4 orchestrator — buffs de magia + condições + modifier stacking:
 *   - Diferentes fontes acumulam
 *   - Habilidade/perícia: acumula entre effectIds diferentes
 *   - Item/magia/parceiro/ambiente: só o maior por source
 *   - effectId duplicado dentro do mesmo source: só o maior
 */

const BASE_ATTR = {
  strength: 3,
  dexterity: 2,
  constitution: 2,
  intelligence: 0,
  wisdom: 1,
  charisma: 0,
} as const

describe('effectTargetKey — chaves canônicas', () => {
  it('attribute', () => {
    expect(effectTargetKey({ k: 'attribute', attribute: 'strength' })).toBe(
      'attribute:strength',
    )
  })

  it('defense', () => {
    expect(effectTargetKey({ k: 'defense' })).toBe('defense')
  })

  it('save', () => {
    expect(effectTargetKey({ k: 'save', save: 'fortitude' })).toBe(
      'save:fortitude',
    )
  })

  it('skill', () => {
    expect(effectTargetKey({ k: 'skill', skill: 'ladinagem' })).toBe(
      'skill:ladinagem',
    )
  })
})

describe('stackModifiersForTarget — regras de acumulação', () => {
  it('duas magias diferentes: só a melhor aplica (magia é non-self-stacking)', () => {
    const res = stackModifiersForTarget([
      { effectId: 'bencao', effectName: 'Bênção', source: 'magia', amount: 1 },
      {
        effectId: 'protecao-divina',
        effectName: 'Proteção Divina',
        source: 'magia',
        amount: 2,
      },
    ])
    expect(res.total).toBe(2)
    // Bênção não aplicada (menor)
    const bencao = res.contributions.find((c) => c.effectId === 'bencao')
    expect(bencao?.applied).toBe(false)
  })

  it('duas habilidades diferentes: ambas somam (habilidade é self-stacking)', () => {
    const res = stackModifiersForTarget([
      {
        effectId: 'foco-em-arma',
        effectName: 'Foco em Arma',
        source: 'habilidade',
        amount: 1,
      },
      {
        effectId: 'inspiracao',
        effectName: 'Inspiração',
        source: 'habilidade',
        amount: 2,
      },
    ])
    expect(res.total).toBe(3)
  })

  it('mesmo effectId aplicado 2x: só a melhor', () => {
    const res = stackModifiersForTarget([
      { effectId: 'bencao', effectName: 'Bênção', source: 'magia', amount: 1 },
      { effectId: 'bencao', effectName: 'Bênção', source: 'magia', amount: 3 },
    ])
    expect(res.total).toBe(3)
  })

  it('sources diferentes empilham: item +1, magia +2, habilidade +1 = 4', () => {
    const res = stackModifiersForTarget([
      { effectId: 'espada-mag', effectName: 'Espada Mágica', source: 'item', amount: 1 },
      { effectId: 'bencao', effectName: 'Bênção', source: 'magia', amount: 2 },
      {
        effectId: 'foco-arma',
        effectName: 'Foco em Arma',
        source: 'habilidade',
        amount: 1,
      },
    ])
    expect(res.total).toBe(4)
  })
})

describe('computeCharacterSheet — buffs em atributo', () => {
  const sheet = computeCharacterSheet({
    level: 5,
    className: 'Guerreiro',
    baseAttributes: BASE_ATTR,
    activeEffects: [
      {
        id: 'fisico-divino',
        name: 'Físico Divino',
        source: 'magia',
        modifiers: [
          { target: { k: 'attribute', attribute: 'strength' }, amount: 2 },
        ],
      },
    ],
  })

  it('FOR total = base 3 + buff 2 = 5', () => {
    expect(sheet.attributes.strength.total).toBe(5)
  })

  it('Luta sem treino agora usa FOR 5', () => {
    expect(sheet.skills.luta.total).toBe(5)
  })
})

describe('computeCharacterSheet — buffs em Defesa', () => {
  const sheet = computeCharacterSheet({
    level: 5,
    className: 'Guerreiro',
    baseAttributes: BASE_ATTR,
    activeEffects: [
      {
        id: 'protecao-divina',
        name: 'Proteção Divina',
        source: 'magia',
        modifiers: [{ target: { k: 'defense' }, amount: 2 }],
      },
    ],
  })

  it('Defesa = 10 + DES 2 + magia +2 = 14', () => {
    expect(sheet.defense.total).toBe(14)
  })
})

describe('computeCharacterSheet — buffs em save (não acumula com base)', () => {
  const sheet = computeCharacterSheet({
    level: 10,
    className: 'Guerreiro',
    baseAttributes: BASE_ATTR,
    activeEffects: [
      {
        id: 'protecao-magia',
        name: 'Proteção Contra Magia',
        source: 'magia',
        modifiers: [
          { target: { k: 'save', save: 'fortitude' }, amount: 5 },
          { target: { k: 'save', save: 'reflexos' }, amount: 5 },
          { target: { k: 'save', save: 'vontade' }, amount: 5 },
        ],
      },
    ],
  })

  it('Fortitude = ½ 10 + CON 2 + magia 5 = 12', () => {
    expect(sheet.saves.fortitude).toBe(12)
  })

  it('Reflexos = ½ 10 + DES 2 + magia 5 = 12', () => {
    expect(sheet.saves.reflexos).toBe(12)
  })
})

describe('computeCharacterSheet — buffs em skill específica', () => {
  const sheet = computeCharacterSheet({
    level: 10,
    className: 'Ladino',
    baseAttributes: BASE_ATTR,
    trainedSkills: ['ladinagem'],
    activeEffects: [
      {
        id: 'guiar',
        name: 'Guiar',
        source: 'magia',
        modifiers: [{ target: { k: 'skill', skill: 'ladinagem' }, amount: 2 }],
      },
    ],
  })

  it('Ladinagem treinada L10 DES 2 = 5 + 2 + 2 = 9, + buff 2 = 11', () => {
    expect(sheet.skills.ladinagem.total).toBe(11)
  })
})

describe('computeCharacterSheet — buffs em ataque + dano', () => {
  const sheet = computeCharacterSheet({
    level: 5,
    className: 'Guerreiro',
    baseAttributes: BASE_ATTR,
    trainedSkills: ['luta'],
    equipment: {
      mainHand: {
        name: 'espada-mágica',
        hand: 'one',
        purpose: 'melee',
        damage: '1d8',
        critRange: 19,
        critMult: 2,
        damageType: 'corte',
      },
    },
    activeEffects: [
      {
        id: 'arma-magica',
        name: 'Arma Mágica',
        source: 'magia',
        modifiers: [
          { target: { k: 'attack' }, amount: 1 },
          { target: { k: 'damage' }, amount: 1 },
        ],
      },
    ],
  })

  it('mainHand attackTotal += 1 buff', () => {
    // Luta L5 treinada + FOR 3 = 4 + 3 + 2 = 9? Actually 2 (½5) + 3 (FOR) + 2 (treino) = 7. +1 buff = 8.
    expect(sheet.attacks.mainHand?.attackTotal).toBe(8)
  })

  it('damageAttributeBonus = FOR 3 + damage buff 1 = 4', () => {
    expect(sheet.attacks.mainHand?.damageAttributeBonus).toBe(4)
  })
})

describe('computeCharacterSheet — condições ativas expostas', () => {
  const sheet = computeCharacterSheet({
    level: 1,
    className: 'Guerreiro',
    baseAttributes: BASE_ATTR,
    activeConditions: ['abalado', 'fatigado'],
  })

  it('conditions inclui abalado + fatigado com descrição', () => {
    expect(sheet.conditions.length).toBe(2)
    expect(sheet.conditions.map((c) => c.id).sort()).toEqual([
      'abalado',
      'fatigado',
    ])
    for (const c of sheet.conditions) {
      expect(c.description).toBeTruthy()
      expect(c.name).toBeTruthy()
    }
  })

  it('condição desconhecida gera warning', () => {
    const s = computeCharacterSheet({
      level: 1,
      className: 'Guerreiro',
      baseAttributes: BASE_ATTR,
      // @ts-expect-error — inválido de propósito
      activeConditions: ['inexistente'],
    })
    expect(s.warnings.some((w) => w.match(/condição desconhecida/))).toBe(true)
    expect(s.conditions.length).toBe(0)
  })
})

describe('computeCharacterSheet — buffs.contributions expõe debug', () => {
  const sheet = computeCharacterSheet({
    level: 5,
    className: 'Guerreiro',
    baseAttributes: BASE_ATTR,
    activeEffects: [
      {
        id: 'bencao',
        name: 'Bênção',
        source: 'magia',
        modifiers: [{ target: { k: 'attack' }, amount: 1 }],
      },
      {
        id: 'inspiracao-bardo',
        name: 'Inspiração do Bardo',
        source: 'habilidade',
        modifiers: [{ target: { k: 'attack' }, amount: 2 }],
      },
    ],
  })

  it('contributions incluem ambos os buffs', () => {
    const attackContribs = sheet.buffs.contributions.filter(
      (c) => c.targetKey === 'attack',
    )
    expect(attackContribs.length).toBe(2)
    for (const c of attackContribs) {
      expect(c.applied).toBe(true)
    }
  })

  it('totals.attack = magia 1 + habilidade 2 = 3', () => {
    expect(sheet.buffs.totals.attack).toBe(3)
  })
})
