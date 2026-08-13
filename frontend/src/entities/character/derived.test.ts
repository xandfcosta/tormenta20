import { describe, expect, it } from 'vitest'
import { conditionalId, statFor } from '@/shared/rules/items-engine'
import type { ClassChoices } from '@/shared/api/catalog-types'
import type { ItemEffects } from '@/shared/api/item-types'
import type { Character, CharacterItem } from '@/shared/api/api'
import {
  armorPenaltyTotal,
  attributeContributions,
  attributeTotal,
  bestBaseSpellCd,
  characterDamageReduction,
  characterEffects,
  defenseTotal,
  displacementTotal,
  evaluatePrerequisite,
  expertiseTotalWithItems,
  inventorySlotsTotal,
  isItemProficient,
  parseClassChoices,
  parseImprovementIds,
  pmCostMod,
  pmLimitTotal,
  spellDCBonus,
  tempHpFromPowers,
} from './derived'

/**
 * derived.ts is the integration layer between t20-data engine and the
 * frontend character sheet. These specs pin both the *formulas* (PDF
 * rules) and the *behavior* (defensive parsing of JSON blobs, fallback
 * when catalog ids miss, prerequisite resolution against character
 * state).
 *
 * PDF refs:
 *  - Defense = 10 + Dex + armor + shield (p106)
 *  - Deslocamento padrão = 9m (p106)
 *  - Inventory slots = 10 + 2*|FOR| (p141)
 *  - ½ level + atributo + (treinamento) (p123)
 *  - Armadura aplica penalidade em Acrobacia, Furtividade, Ladinagem (p143)
 */

function character(over: Partial<Character> = {}): Character {
  return {
    id: 1,
    name: 'X',
    level: 1,
    hpMax: 12,
    hpCurrent: 12,
    mpMax: 4,
    mpCurrent: 4,
    strength: 0,
    dexterity: 0,
    constitution: 0,
    intelligence: 0,
    wisdom: 0,
    charisma: 0,
    size: 'M',
    displacement: 9,
    proficiencies: JSON.stringify([
      'armas-simples',
      'armas-marciais',
      'armaduras-leves',
      'armaduras-pesadas',
      'escudos',
    ]),
    raceAbilityChoices: '[]',
    raceAttributeChoices: '{}',
    secondaryRaceChoices: '[]',
    originChoices: '[]',
    classPowers: '[]',
    classChoices: '{}',
    origin: 'Soldado',
    god: null,
    expertises: [],
    races: [],
    classes: [],
    items: [],
    activeEffects: [],
    ...over,
  } as Character
}

function item(over: Partial<CharacterItem> = {}): CharacterItem {
  return {
    id: 1,
    catalogId: null,
    name: 'X',
    quantity: 1,
    slots: 1,
    equipped: null,
    improvements: '[]',
    material: null,
    ...over,
  } as CharacterItem
}

function emptyEffects(): ItemEffects {
  return { byTarget: {}, flags: new Set(), conditional: [] }
}

describe('active-effect attack modifiers (Phase 0 — buff engine)', () => {
  // Regression: a buff/conditional granting {k:'attack', scope:'all'} (e.g.
  // Fúria, Bênção) must flow from character.activeEffects through
  // characterEffects into statFor so the combat display can add it.
  it('folds an active {k:attack,scope:all} effect into statFor', () => {
    const c = character({
      activeEffects: [
        {
          id: 1,
          catalogId: 'buff',
          scope: 'scene',
          modifiers: JSON.stringify([
            { target: { k: 'attack', scope: 'all' }, amount: 2, bonusType: 'morale' },
          ]),
          createdAt: '',
        },
      ],
    })
    const eff = characterEffects(c)
    expect(statFor(eff, { k: 'attack', scope: 'all' }).total).toBe(2)
  })

  it('does not report a scope:all bonus when only scope:this is present', () => {
    const c = character({
      activeEffects: [
        {
          id: 1,
          catalogId: 'buff',
          scope: 'scene',
          modifiers: JSON.stringify([
            { target: { k: 'attack', scope: 'this' }, amount: -5, bonusType: 'untyped' },
          ]),
          createdAt: '',
        },
      ],
    })
    const eff = characterEffects(c)
    expect(statFor(eff, { k: 'attack', scope: 'all' }).total).toBe(0)
  })
})

describe('general-power modifiers', () => {
  // Regression: general powers are stored in classPowers by their bare id
  // (e.g. 'esquiva'). generalPowerActiveItem used to require a 'general.'
  // prefix the picker never writes, so their modifiers never applied.
  it('folds a chosen general power (esquiva) into defense + Reflexos', () => {
    const c = character({ classPowers: JSON.stringify(['esquiva']) })
    const eff = characterEffects(c)
    expect(statFor(eff, { k: 'defense' }).total).toBe(2)
    expect(statFor(eff, { k: 'expertise', name: 'Reflexos' }).total).toBe(2)
  })

  it('ignores a bare class-elective id that is not a general power', () => {
    const c = character({ classPowers: JSON.stringify(['not-a-general-power']) })
    const eff = characterEffects(c)
    expect(statFor(eff, { k: 'defense' }).total).toBe(0)
  })
})

describe('parseClassChoices', () => {
  it('returns empty object for malformed JSON', () => {
    expect(parseClassChoices('not json')).toEqual({})
  })

  it('returns empty object for non-object JSON (array)', () => {
    expect(parseClassChoices('[]')).toEqual({})
  })

  it('returns the parsed object when valid', () => {
    const blob: ClassChoices = { Clérigo: { devoto: 'khalmyr' } }
    expect(parseClassChoices(JSON.stringify(blob))).toEqual(blob)
  })
})

describe('parseImprovementIds', () => {
  it('returns empty array for malformed JSON', () => {
    expect(parseImprovementIds('not json')).toEqual([])
  })

  it('returns empty array for non-array JSON', () => {
    expect(parseImprovementIds('{"x":1}')).toEqual([])
  })

  it('filters non-string entries', () => {
    expect(parseImprovementIds('["a", 1, null, "b"]')).toEqual(['a', 'b'])
  })
})

describe('evaluatePrerequisite', () => {
  const c = character({ strength: 2, dexterity: 1 })

  it('power: met when id is in chosen set', () => {
    const result = evaluatePrerequisite(
      { kind: 'power', id: 'class.guerreiro.ambidestria' },
      c,
      new Set(['class.guerreiro.ambidestria']),
      {},
    )
    expect(result.met).toBe(true)
  })

  it('power: not met when id is absent', () => {
    const result = evaluatePrerequisite(
      { kind: 'power', id: 'class.guerreiro.ambidestria' },
      c,
      new Set(),
      {},
    )
    expect(result.met).toBe(false)
  })

  it('anyPower: met when any id matches', () => {
    const result = evaluatePrerequisite(
      { kind: 'anyPower', ids: ['a', 'b'] },
      c,
      new Set(['b']),
      {},
    )
    expect(result.met).toBe(true)
  })

  it('trained: met when expertise.trained=true', () => {
    const c2 = character({
      expertises: [
        {
          name: 'Atletismo',
          attribute: 'strength',
          trained: true,
          custom: false,
        },
      ],
    })
    const result = evaluatePrerequisite(
      { kind: 'trained', expertise: 'Atletismo' },
      c2,
      new Set(),
      {},
    )
    expect(result.met).toBe(true)
  })

  it('trained: not met when expertise row missing', () => {
    const result = evaluatePrerequisite(
      { kind: 'trained', expertise: 'Atletismo' },
      c,
      new Set(),
      {},
    )
    expect(result.met).toBe(false)
  })

  it('attribute: met when raw attribute ≥ min', () => {
    const result = evaluatePrerequisite(
      { kind: 'attribute', attr: 'strength', min: 2 },
      c,
      new Set(),
      {},
    )
    expect(result.met).toBe(true)
  })

  it('attribute: not met when raw attribute < min', () => {
    const result = evaluatePrerequisite(
      { kind: 'attribute', attr: 'strength', min: 3 },
      c,
      new Set(),
      {},
    )
    expect(result.met).toBe(false)
  })

  it('classChoice: met when value is set and in allowed list', () => {
    const result = evaluatePrerequisite(
      {
        kind: 'classChoice',
        class: 'Clérigo',
        field: 'devoto',
        allowed: ['khalmyr', 'marah'],
        label: 'Devoto de Khalmyr ou Marah',
      },
      c,
      new Set(),
      { Clérigo: { devoto: 'khalmyr' } },
    )
    expect(result.met).toBe(true)
  })

  it('classChoice: not met when value is in forbidden list', () => {
    const result = evaluatePrerequisite(
      {
        kind: 'classChoice',
        class: 'Paladino',
        field: 'devoto',
        forbidden: ['lena', 'marah'],
        label: 'Devoto, exceto Lena/Marah',
      },
      c,
      new Set(),
      { Paladino: { devoto: 'lena' } },
    )
    expect(result.met).toBe(false)
  })

  it('classChoice: not met when value is empty', () => {
    const result = evaluatePrerequisite(
      {
        kind: 'classChoice',
        class: 'Clérigo',
        field: 'devoto',
        label: 'Devoto',
      },
      c,
      new Set(),
      {},
    )
    expect(result.met).toBe(false)
  })

  it('note: always returns met=true (UI-only hint)', () => {
    const result = evaluatePrerequisite(
      { kind: 'note', description: 'GM call' },
      c,
      new Set(),
      {},
    )
    expect(result.met).toBe(true)
    expect(result.reason).toBe('GM call')
  })
})

describe('isItemProficient', () => {
  it('custom items (no catalogId) are always proficient', () => {
    const c = character({ proficiencies: '[]' })
    expect(isItemProficient(c, item({ catalogId: null }))).toBe(true)
  })

  it('items with no proficiency requirement (apparel) are proficient', () => {
    const c = character({ proficiencies: '[]' })
    expect(isItemProficient(c, item({ catalogId: 'bandana' }))).toBe(true)
  })

  it('weapon: proficient when category is in the set', () => {
    const c = character({
      proficiencies: JSON.stringify(['armas-simples']),
    })
    expect(isItemProficient(c, item({ catalogId: 'adaga' }))).toBe(true)
  })

  it('weapon: not proficient when category is missing', () => {
    const c = character({ proficiencies: JSON.stringify(['armas-simples']) })
    expect(isItemProficient(c, item({ catalogId: 'espada-longa' }))).toBe(
      false,
    )
  })

  it('falls back to proficient when catalogId is unknown', () => {
    const c = character({ proficiencies: '[]' })
    expect(isItemProficient(c, item({ catalogId: 'no-such-id' }))).toBe(true)
  })
})

describe('attributeTotal + attributeContributions', () => {
  it('returns raw attribute when no item effects target it', () => {
    const c = character({ strength: 3 })
    expect(attributeTotal(c, 'strength', emptyEffects())).toBe(3)
  })

  it('contributions list is empty when no items', () => {
    expect(attributeContributions('strength', emptyEffects())).toEqual([])
  })
})

describe('attributeTotal — race applied once from BASE + choices', () => {
  it('fixed race (Anão) folds its mod onto the base attribute', () => {
    const c = character({ constitution: 2, races: [{ race: 'Anão' }] })
    // base 2 + Anão CON+2 = 4 (applied exactly once, no double).
    expect(attributeTotal(c, 'constitution', characterEffects(c))).toBe(4)
    expect(attributeTotal(c, 'dexterity', characterEffects(c))).toBe(-1)
  })

  it('floating race (Humano) applies the persisted floating picks', () => {
    const c = character({
      strength: 1,
      races: [{ race: 'Humano' }],
      raceAttributeChoices: JSON.stringify({
        floatingPicks: ['strength', 'constitution', 'wisdom'],
      }),
    })
    expect(attributeTotal(c, 'strength', characterEffects(c))).toBe(2) // 1 + 1
    expect(attributeTotal(c, 'wisdom', characterEffects(c))).toBe(1) // 0 + 1
    expect(attributeTotal(c, 'dexterity', characterEffects(c))).toBe(0)
  })

  it('floating race without picks applies nothing (graceful)', () => {
    const c = character({ strength: 1, races: [{ race: 'Humano' }] })
    expect(attributeTotal(c, 'strength', characterEffects(c))).toBe(1)
  })

  it('applies an opted-in secondary race on top of the primary', () => {
    // Primary Minotauro (CON+1), secondary Lefou applied (CAR-1).
    const base = character({
      constitution: 2,
      charisma: 2,
      races: [{ race: 'Minotauro' }, { race: 'Lefou' }],
    })
    // Not applied → only Minotauro: CON 3, CAR unchanged.
    expect(attributeTotal(base, 'constitution', characterEffects(base))).toBe(3)
    expect(attributeTotal(base, 'charisma', characterEffects(base))).toBe(2)
    // Applied with Lefou's 3 floating picks → +1 each + the CAR-1 penalty.
    const applied = character({
      ...base,
      secondaryRaceChoices: JSON.stringify([
        { race: 'Lefou', floatingPicks: ['strength', 'dexterity', 'constitution'] },
      ]),
    })
    // CON: Minotauro +1 + Lefou pick +1 = base 2 + 2 = 4.
    expect(attributeTotal(applied, 'constitution', characterEffects(applied))).toBe(4)
    // CAR: Lefou penalty -1 = base 2 - 1 = 1.
    expect(attributeTotal(applied, 'charisma', characterEffects(applied))).toBe(1)
  })
})

describe('Deformidade (Lefou p23) — +2 perícias e perda de CAR', () => {
  const furtividade = {
    name: 'Furtividade',
    attribute: 'dexterity',
    trained: false,
  } as Character['expertises'][number]

  it('primary Lefou: +2 na perícia escolhida, sem perda de CAR sem poder', () => {
    const c = character({
      charisma: 1,
      races: [{ race: 'Lefou' }],
      raceAttributeChoices: JSON.stringify({
        floatingPicks: ['strength', 'constitution', 'wisdom'],
        deformidade: { pericias: ['Furtividade', 'Percepção'] },
      }),
    })
    const effects = characterEffects(c)
    expect(expertiseTotalWithItems(c, furtividade, effects).itemBonus).toBe(2)
    // CAR: base 1 + Lefou -1 = 0; bônus de perícia NÃO perde Carisma (p23).
    expect(attributeTotal(c, 'charisma', effects)).toBe(0)
  })

  it('poder trocado perde 1 CAR (p136)', () => {
    const c = character({
      charisma: 1,
      races: [{ race: 'Lefou' }],
      raceAttributeChoices: JSON.stringify({
        floatingPicks: ['strength', 'constitution', 'wisdom'],
        deformidade: { pericias: ['Furtividade'], tormentaPower: 'dentes-afiados' },
      }),
    })
    const effects = characterEffects(c)
    // CAR: base 1 − 1 Lefou − 1 poder = −1.
    expect(attributeTotal(c, 'charisma', effects)).toBe(-1)
    expect(expertiseTotalWithItems(c, furtividade, effects).itemBonus).toBe(2)
  })

  it('secondary Lefou aplicado carrega a deformidade', () => {
    const c = character({
      races: [{ race: 'Minotauro' }, { race: 'Lefou' }],
      secondaryRaceChoices: JSON.stringify([
        {
          race: 'Lefou',
          floatingPicks: ['strength', 'dexterity', 'constitution'],
          deformidade: { pericias: ['Furtividade'] },
        },
      ]),
    })
    expect(
      expertiseTotalWithItems(c, furtividade, characterEffects(c)).itemBonus,
    ).toBe(2)
  })

  it('poderes da Tormenta escolhidos no pool também perdem CAR (escalada p136)', () => {
    const c = character({
      charisma: 1,
      races: [{ race: 'Lefou' }],
      raceAttributeChoices: JSON.stringify({
        floatingPicks: ['strength', 'constitution', 'wisdom'],
        deformidade: { pericias: [], tormentaPower: 'dentes-afiados' },
      }),
      classPowers: JSON.stringify(['antenas', 'carapaca']),
    })
    // 3 poderes reais: 1→1, 2→2, 3→4 ⇒ CAR 1 − 1 (Lefou) − 4 = −4
    expect(attributeTotal(c, 'charisma', characterEffects(c))).toBe(-4)
  })

  it('poder duplicado (Deformidade + pool) conta uma vez', () => {
    const c = character({
      charisma: 1,
      races: [{ race: 'Lefou' }],
      raceAttributeChoices: JSON.stringify({
        floatingPicks: ['strength', 'constitution', 'wisdom'],
        deformidade: { pericias: [], tormentaPower: 'dentes-afiados' },
      }),
      classPowers: JSON.stringify(['dentes-afiados']),
    })
    expect(attributeTotal(c, 'charisma', characterEffects(c))).toBe(-1)
  })

  it('deformidade em raça sem a habilidade é ignorada', () => {
    const c = character({
      races: [{ race: 'Humano' }],
      raceAttributeChoices: JSON.stringify({
        deformidade: { pericias: ['Furtividade'], tormentaPower: 'antenas' },
      }),
    })
    const effects = characterEffects(c)
    expect(expertiseTotalWithItems(c, furtividade, effects).itemBonus).toBe(0)
    expect(attributeTotal(c, 'charisma', effects)).toBe(0)
  })
})

describe('defenseTotal — PDF p106 formula', () => {
  it('base = 10 + Dex when no flags + no items', () => {
    const c = character({ dexterity: 3 })
    const result = defenseTotal(c, emptyEffects())
    expect(result.base).toBe(13)
    expect(result.total).toBe(13)
    expect(result.dexApplied).toBe(true)
  })

  it('drops Dex when cannot-apply-dex-to-defense flag is on', () => {
    const c = character({ dexterity: 3 })
    const effects: ItemEffects = {
      byTarget: {},
      flags: new Set(['cannot-apply-dex-to-defense']),
      conditional: [],
    }
    const result = defenseTotal(c, effects)
    expect(result.base).toBe(10)
    expect(result.dexApplied).toBe(false)
  })

  it('adds defense modifiers (armor/shield) to base', () => {
    const c = character({ dexterity: 2 })
    const effects: ItemEffects = {
      byTarget: {
        defense: {
          total: 6,
          contributions: [
            { source: 'cota-malha', amount: 6, bonusType: 'armor' },
          ],
        },
      },
      flags: new Set(),
      conditional: [],
    }
    const result = defenseTotal(c, effects)
    expect(result.total).toBe(10 + 2 + 6)
    expect(result.contributions).toEqual([
      { source: 'cota-malha', amount: 6 },
    ])
  })
})

describe('displacementTotal — PDF p106 default 9m', () => {
  it('returns the character displacement when no items', () => {
    const c = character({ displacement: 9 })
    expect(displacementTotal(c, emptyEffects()).total).toBe(9)
  })

  it('clamps to 0 when item penalty exceeds base', () => {
    const c = character({ displacement: 3 })
    const effects: ItemEffects = {
      byTarget: {
        displacement: {
          total: -10,
          contributions: [{ source: 'cota-malha', amount: -10, bonusType: 'untyped' }],
        },
      },
      flags: new Set(),
      conditional: [],
    }
    expect(displacementTotal(c, effects).total).toBe(0)
  })

  it('adds positive bonuses (Botas reforçadas +1.5)', () => {
    const c = character({ displacement: 9 })
    const effects: ItemEffects = {
      byTarget: {
        displacement: {
          total: 1.5,
          contributions: [{ source: 'botas', amount: 1.5, bonusType: 'item' }],
        },
      },
      flags: new Set(),
      conditional: [],
    }
    expect(displacementTotal(c, effects).total).toBe(10.5)
  })
})

describe('inventorySlotsTotal — PDF p141 formula', () => {
  it('10 + 2 per positive Força point', () => {
    expect(inventorySlotsTotal(character({ strength: 3 }), emptyEffects())).toBe(16)
    expect(inventorySlotsTotal(character({ strength: 4 }), emptyEffects())).toBe(18)
    expect(inventorySlotsTotal(character({ strength: 0 }), emptyEffects())).toBe(10)
  })

  it('−1 per negative Força point (regression: |FOR| gave 14 instead of 8)', () => {
    expect(inventorySlotsTotal(character({ strength: -2 }), emptyEffects())).toBe(8)
  })

  it('folds in inventorySlots stat modifier', () => {
    const effects: ItemEffects = {
      byTarget: {
        inventorySlots: {
          total: 4,
          contributions: [{ source: 'mochila', amount: 4, bonusType: 'item' }],
        },
      },
      flags: new Set(),
      conditional: [],
    }
    expect(inventorySlotsTotal(character({ strength: 2 }), effects)).toBe(18)
  })
})

describe('pmLimitTotal — PDF p224: limite = nível na classe conjuradora', () => {
  // Regression: base was ½ nível do personagem; the book keys the variable-PM
  // cap to "seu nível na classe que fornece a habilidade" (p224).
  it('Arcanista 12 → 12 (não ½ nível)', () => {
    const c = character({
      level: 12,
      classes: [{ className: 'Arcanista', level: 12 }],
    })
    expect(pmLimitTotal(c, emptyEffects()).total).toBe(12)
  })

  it('Guerreiro 4 / Arcanista 4 → 4 (nível na classe conjuradora)', () => {
    const c = character({
      level: 8,
      classes: [
        { className: 'Guerreiro', level: 4 },
        { className: 'Arcanista', level: 4 },
      ],
    })
    expect(pmLimitTotal(c, emptyEffects()).total).toBe(4)
  })

  it('duas classes conjuradoras → maior nível entre elas', () => {
    const c = character({
      level: 8,
      classes: [
        { className: 'Clérigo', level: 3 },
        { className: 'Arcanista', level: 5 },
      ],
    })
    expect(pmLimitTotal(c, emptyEffects()).total).toBe(5)
  })

  it('sem classe conjuradora → nível do personagem (caixa oculta na ficha)', () => {
    const c = character({
      level: 7,
      classes: [{ className: 'Guerreiro', level: 7 }],
    })
    expect(pmLimitTotal(c, emptyEffects()).total).toBe(7)
  })

  it('L1 → base 1 (min)', () => {
    expect(pmLimitTotal(character({ level: 1 }), emptyEffects()).total).toBe(1)
  })
})

describe('bestBaseSpellCd — CD = 10 + ½ nível + atributo-chave FINAL (PDF p173)', () => {
  // Regression: the CD used the RAW stored attribute, dropping racial/item
  // attribute bonuses (Necromante Osteon: CD 21 shown, 22 correct).
  it('folds attribute modifiers from effects into the key attribute', () => {
    const c = character({
      level: 12,
      intelligence: 5,
      classes: [{ className: 'Arcanista', level: 12 }],
    })
    const effects: ItemEffects = {
      byTarget: {
        'attribute:intelligence': {
          total: 1,
          contributions: [{ source: 'Osteon', amount: 1, bonusType: 'untyped' }],
        },
      },
      flags: new Set(),
      conditional: [],
    }
    // 10 + 6 (½ de 12) + 6 (5 base + 1 racial) = 22
    expect(bestBaseSpellCd(c, effects)).toBe(22)
    expect(bestBaseSpellCd(c, emptyEffects())).toBe(21)
  })

  it('picks the best CD across caster classes', () => {
    const c = character({
      level: 8,
      intelligence: 4,
      wisdom: 1,
      classes: [
        { className: 'Arcanista', level: 4 },
        { className: 'Clérigo', level: 4 },
      ],
    })
    // Arcanista (INT 4): 10 + 4 + 4 = 18; Clérigo (SAB 1): 15 → best 18
    expect(bestBaseSpellCd(c, emptyEffects())).toBe(18)
  })

  it('returns null when no class casts spells', () => {
    const c = character({
      level: 5,
      classes: [{ className: 'Guerreiro', level: 5 }],
    })
    expect(bestBaseSpellCd(c, emptyEffects())).toBeNull()
  })
})

describe('armorPenaltyTotal', () => {
  it('returns 0 when no armorPenalty stat', () => {
    expect(armorPenaltyTotal(emptyEffects())).toBe(0)
  })

  it('returns the stat total', () => {
    const effects: ItemEffects = {
      byTarget: {
        armorPenalty: {
          total: -3,
          contributions: [{ source: 'a', amount: -3, bonusType: 'untyped' }],
        },
      },
      flags: new Set(),
      conditional: [],
    }
    expect(armorPenaltyTotal(effects)).toBe(-3)
  })
})

describe('expertiseTotalWithItems', () => {
  it('matches the base formula when no items', () => {
    const c = character({ level: 7, strength: 3 })
    const result = expertiseTotalWithItems(
      c,
      { name: 'Atletismo', attribute: 'strength', trained: true, custom: false },
      emptyEffects(),
    )
    // ½ level (3) + STR (3) + training L7 (+4) = 10
    expect(result.halfLevel).toBe(3)
    expect(result.attrValue).toBe(3)
    expect(result.training).toBe(4)
    expect(result.base).toBe(10)
    expect(result.total).toBe(10)
  })

  it('applies armor penalty only to Acrobacia/Furtividade/Ladinagem', () => {
    const effects: ItemEffects = {
      byTarget: {
        armorPenalty: {
          total: -3,
          contributions: [{ source: 'cota', amount: -3, bonusType: 'untyped' }],
        },
      },
      flags: new Set(),
      conditional: [],
    }
    const c = character({ level: 1, dexterity: 2 })

    const acro = expertiseTotalWithItems(
      c,
      { name: 'Acrobacia', attribute: 'dexterity', trained: false, custom: false },
      effects,
    )
    expect(acro.armorPenaltyApplied).toBe(-3)

    const luta = expertiseTotalWithItems(
      c,
      { name: 'Luta', attribute: 'strength', trained: false, custom: false },
      effects,
    )
    expect(luta.armorPenaltyApplied).toBe(0)
  })

  it('includes itemContributions for typed expertise bonuses', () => {
    const effects: ItemEffects = {
      byTarget: {
        'expertise:Intimidação': {
          total: 1,
          contributions: [{ source: 'bandana', amount: 1, bonusType: 'item' }],
        },
      },
      flags: new Set(),
      conditional: [],
    }
    const c = character({ level: 1, charisma: 0 })
    const result = expertiseTotalWithItems(
      c,
      {
        name: 'Intimidação',
        attribute: 'charisma',
        trained: false,
        custom: false,
      },
      effects,
    )
    expect(result.itemBonus).toBe(1)
    expect(result.itemContributions).toEqual([
      { source: 'bandana', amount: 1 },
    ])
  })
})

describe('spellDCBonus + pmCostMod', () => {
  it('returns zeroed defaults for empty effects', () => {
    expect(spellDCBonus(emptyEffects())).toEqual({
      total: 0,
      contributions: [],
    })
    expect(pmCostMod(emptyEffects())).toEqual({
      total: 0,
      contributions: [],
    })
  })
})

describe('characterEffects — empty character integration', () => {
  it('returns empty ItemEffects shape for a character with no items / races / classes', () => {
    const c = character()
    const effects = characterEffects(c)
    expect(effects.byTarget).toEqual({})
    expect(effects.flags.size).toBe(0)
    expect(effects.conditional).toEqual([])
  })
})

/**
 * Edge cases: previously-untested branches of derived.ts that fold a
 * stat into the raw value. Each spec pins a specific byTarget key so a
 * future engine refactor that changes the key shape (e.g. `attribute:str`
 * vs `attribute:strength`) breaks the spec immediately rather than
 * silently dropping the bonus in the sheet.
 */
describe('attributeTotal + attributeContributions — folded modifiers', () => {
  it('adds attribute stat total to the raw value (race bonus +2)', () => {
    const c = character({ strength: 1 })
    const effects: ItemEffects = {
      byTarget: {
        'attribute:strength': {
          total: 2,
          contributions: [
            { source: 'Raça: Minotauro', amount: 2, bonusType: 'untyped' },
          ],
        },
      },
      flags: new Set(),
      conditional: [],
    }
    expect(attributeTotal(c, 'strength', effects)).toBe(3)
  })

  it('applies negative attribute modifiers (Elfo: Con-1)', () => {
    const c = character({ constitution: 0 })
    const effects: ItemEffects = {
      byTarget: {
        'attribute:constitution': {
          total: -1,
          contributions: [
            { source: 'Raça: Elfo', amount: -1, bonusType: 'untyped' },
          ],
        },
      },
      flags: new Set(),
      conditional: [],
    }
    expect(attributeTotal(c, 'constitution', effects)).toBe(-1)
  })

  it('exposes contributions for the targeted attribute only', () => {
    const effects: ItemEffects = {
      byTarget: {
        'attribute:strength': {
          total: 2,
          contributions: [
            { source: 'Raça: Minotauro', amount: 2, bonusType: 'untyped' },
          ],
        },
        'attribute:dexterity': {
          total: 1,
          contributions: [
            { source: 'Raça: Elfo', amount: 1, bonusType: 'untyped' },
          ],
        },
      },
      flags: new Set(),
      conditional: [],
    }
    expect(attributeContributions('strength', effects)).toEqual([
      { source: 'Raça: Minotauro', amount: 2 },
    ])
    expect(attributeContributions('dexterity', effects)).toEqual([
      { source: 'Raça: Elfo', amount: 1 },
    ])
  })
})

describe('pmLimitTotal — folded pmLimit stat', () => {
  it('adds pmLimit stat to the caster-level base', () => {
    const c = character({
      level: 7,
      classes: [{ className: 'Druida', level: 7 }],
    })
    const effects: ItemEffects = {
      byTarget: {
        pmLimit: {
          total: 2,
          contributions: [
            { source: 'Foco em Magia', amount: 2, bonusType: 'untyped' },
          ],
        },
      },
      flags: new Set(),
      conditional: [],
    }
    const result = pmLimitTotal(c, effects)
    expect(result.base).toBe(7)
    expect(result.itemBonus).toBe(2)
    expect(result.total).toBe(9)
    expect(result.contributions).toEqual([
      { source: 'Foco em Magia', amount: 2 },
    ])
  })

  it('clamps base to min 1 at L1 even when pmLimit stat is 0', () => {
    expect(pmLimitTotal(character({ level: 1 }), emptyEffects()).base).toBe(1)
  })
})

describe('spellDCBonus — stat present', () => {
  it('returns the spellDC stat total + contributions', () => {
    const effects: ItemEffects = {
      byTarget: {
        spellDC: {
          total: 1,
          contributions: [
            { source: 'Foco em Encantamento', amount: 1, bonusType: 'untyped' },
          ],
        },
      },
      flags: new Set(),
      conditional: [],
    }
    expect(spellDCBonus(effects)).toEqual({
      total: 1,
      contributions: [{ source: 'Foco em Encantamento', amount: 1 }],
    })
  })
})

describe('pmCostMod — stat present', () => {
  it('returns the pmCost stat total + contributions (negative = cheaper)', () => {
    const effects: ItemEffects = {
      byTarget: {
        pmCost: {
          total: -1,
          contributions: [
            { source: 'Magia Eficiente', amount: -1, bonusType: 'untyped' },
          ],
        },
      },
      flags: new Set(),
      conditional: [],
    }
    expect(pmCostMod(effects)).toEqual({
      total: -1,
      contributions: [{ source: 'Magia Eficiente', amount: -1 }],
    })
  })
})

describe('mirrorWeaponAttackMods — linha da arma chega em Luta/Pontaria', () => {
  const LUTA = {
    name: 'Luta',
    attribute: 'strength',
    trained: false,
    custom: false,
  } as const

  const weaponItem = (
    catalogId: string,
    over: Partial<Character['items'][number]> = {},
  ) =>
    ({
      id: 1,
      catalogId,
      name: catalogId,
      quantity: 1,
      slots: 2,
      equipped: 'wielded2',
      improvements: '[]',
      material: null,
      ...over,
    }) as Character['items'][number]

  it('desbalanceada do Machado táurico soma -2 na Luta (p149)', () => {
    const c = character({
      proficiencies: JSON.stringify(['armas-exoticas']),
      items: [weaponItem('machado-taurico')],
    })
    const luta = expertiseTotalWithItems(c, LUTA, characterEffects(c))
    expect(luta.itemBonus).toBe(-2)
    expect(luta.itemContributions).toContainEqual({
      source: 'machado-taurico',
      amount: -2,
      // The note is what makes the breakdown row self-explanatory.
      note: 'desbalanceada: -2 em ataque',
    })
  })

  it('melhoria Certeira soma +1 na Luta via overlay', () => {
    const c = character({
      items: [
        weaponItem('espada-curta', {
          equipped: 'wielded',
          improvements: JSON.stringify(['melhoria-certeira']),
        }),
      ],
    })
    expect(
      expertiseTotalWithItems(c, LUTA, characterEffects(c)).itemBonus,
    ).toBe(1)
  })

  it('arma guardada (não equipada) não espelha nada', () => {
    const c = character({
      proficiencies: JSON.stringify(['armas-exoticas']),
      items: [weaponItem('machado-taurico', { equipped: null })],
    })
    expect(
      expertiseTotalWithItems(c, LUTA, characterEffects(c)).itemBonus,
    ).toBe(0)
  })

  describe('homebrew: Equilibrada anula desbalanceada (toggle no Efeitos)', () => {
    const tauricoEquilibrado = () =>
      character({
        proficiencies: JSON.stringify(['armas-exoticas']),
        items: [
          weaponItem('machado-taurico', {
            improvements: JSON.stringify(['melhoria-equilibrada']),
          }),
        ],
      })

    it('RAW por padrão: sem o toggle, o -2 permanece', () => {
      const c = tauricoEquilibrado()
      expect(
        expertiseTotalWithItems(c, LUTA, characterEffects(c)).itemBonus,
      ).toBe(-2)
    })

    it('expõe o contraponto como conditional opt-in', () => {
      const cond = characterEffects(tauricoEquilibrado()).conditional.find(
        (x) => /Homebrew: Equilibrada/.test(x.note),
      )
      expect(cond).toBeDefined()
      expect(cond!.amount).toBe(2)
      expect(cond!.target).toEqual({ k: 'expertise', name: 'Luta' })
    })

    it('com o toggle ativo, -2 e +2 se anulam (net 0)', () => {
      const c = tauricoEquilibrado()
      const cond = characterEffects(c).conditional.find((x) =>
        /Homebrew: Equilibrada/.test(x.note),
      )!
      const effects = characterEffects(c, new Set([conditionalId(cond)]))
      expect(expertiseTotalWithItems(c, LUTA, effects).itemBonus).toBe(0)
    })

    it('sem a melhoria não há toggle homebrew', () => {
      const c = character({
        proficiencies: JSON.stringify(['armas-exoticas']),
        items: [weaponItem('machado-taurico')],
      })
      expect(
        characterEffects(c).conditional.some((x) =>
          /Homebrew: Equilibrada/.test(x.note),
        ),
      ).toBe(false)
    })
  })
})

describe('homebrew: Medalhão de prata vestido (registry HOMEBREW_VESTED_OK)', () => {
  const medalhao = (equipped: 'vested' | 'wielded') =>
    character({
      items: [
        item({ catalogId: 'medalhao-de-prata', name: 'Medalhão de prata', equipped }),
      ],
    })

  it('vestido sem toggle: RAW, nenhum bônus de limite de PM', () => {
    const c = medalhao('vested')
    expect(pmLimitTotal(c, characterEffects(c)).contributions).toEqual([])
  })

  it('vestido expõe o conditional homebrew no Efeitos', () => {
    const cond = characterEffects(medalhao('vested')).conditional.find((x) =>
      /Homebrew: esotérico vestido/.test(x.note),
    )
    expect(cond).toBeDefined()
    expect(cond!.target).toEqual({ k: 'pmLimit' })
    expect(cond!.amount).toBe(1)
  })

  it('vestido com o toggle ativo soma o +1 de limite de PM', () => {
    const c = medalhao('vested')
    const cond = characterEffects(c).conditional.find((x) =>
      /Homebrew: esotérico vestido/.test(x.note),
    )!
    const effects = characterEffects(c, new Set([conditionalId(cond)]))
    expect(statFor(effects, { k: 'pmLimit' }).total).toBe(1)
  })

  it('vestido com Vigilante: o toggle também reativa o overlay (+2 Defesa)', () => {
    // Regressão: a versão inicial só reativava os modifiers do CATÁLOGO —
    // a melhoria Vigilante (overlay) ficava morta com o medalhão vestido.
    const c = character({
      items: [
        item({
          catalogId: 'medalhao-de-prata',
          name: 'Medalhão de prata',
          equipped: 'vested',
          improvements: JSON.stringify(['melhoria-vigilante']),
        }),
      ],
    })
    const conds = characterEffects(c).conditional.filter((x) =>
      /Homebrew: esotérico vestido/.test(x.note),
    )
    // One flag-grouped toggle covering BOTH bonuses.
    expect(conds).toHaveLength(2)
    expect(new Set(conds.map((x) => x.flag))).toEqual(
      new Set(['homebrew-vestido-medalhao-de-prata']),
    )
    const effects = characterEffects(
      c,
      new Set(conds.map((x) => conditionalId(x))),
    )
    expect(statFor(effects, { k: 'defense' }).total).toBe(2)
    expect(statFor(effects, { k: 'pmLimit' }).total).toBe(1)
  })

  it('empunhado segue RAW direto: +1 sem conditional homebrew', () => {
    const c = medalhao('wielded')
    const effects = characterEffects(c)
    expect(statFor(effects, { k: 'pmLimit' }).total).toBe(1)
    expect(
      effects.conditional.some((x) => /Homebrew: esotérico/.test(x.note)),
    ).toBe(false)
  })
})

describe('overlay provenance nas notes (melhoria/material nomeados)', () => {
  it('Couraça + Reforçada: contribution de Defesa diz qual melhoria', () => {
    const c = character({
      items: [
        item({
          catalogId: 'couraca',
          name: 'Couraça',
          equipped: 'vested',
          improvements: JSON.stringify(['melhoria-reforcada']),
        }),
      ],
    })
    const def = defenseTotal(c, characterEffects(c))
    expect(def.contributions).toContainEqual(
      expect.objectContaining({
        source: 'Couraça',
        amount: 1,
        note: 'Reforçada: +1 Defesa',
      }),
    )
  })
})

describe('characterDamageReduction — RD junto da Defesa', () => {
  const heavyGear = () =>
    item({ id: 9, catalogId: 'armadura-completa', name: 'Armadura completa', equipped: 'vested' })

  it('Bárbaro segue a tabela p47 (nível 8 → RD 4), sem exigir armadura', () => {
    const c = character({ classes: [{ className: 'Bárbaro', level: 8 }] })
    const rd = characterDamageReduction(c, characterEffects(c))
    expect(rd.total).toBe(4)
    expect(rd.sources[0].source).toContain('Bárbaro')
  })

  // O Guerreiro NÃO tem RD passiva. O livro p65 dá a ele "Especialização em
  // Armadura": poder ESCOLHIDO, pré-requisito de 12º nível, RD 5 fixa com
  // armadura pesada. Antes o motor lhe dava a progressão do Bárbaro desde o 5º
  // nível, e este teste travava exatamente essa invenção (ALE-111).
  it('Guerreiro sem o poder escolhido não tem RD, nem de armadura pesada', () => {
    const guerreiro = character({
      classes: [{ className: 'Guerreiro', level: 11 }],
      items: [heavyGear()],
    })

    expect(characterDamageReduction(guerreiro, characterEffects(guerreiro)).total).toBe(0)
  })

  it('Guerreiro com Especialização em Armadura: RD 5 a partir do 12º, fixa', () => {
    const comPoder = (level: number) =>
      character({
        classes: [{ className: 'Guerreiro', level }],
        items: [heavyGear()],
        classPowers: JSON.stringify(['class.guerreiro.especializacao-em-armadura']),
      })

    // Pré-requisito de nível: o poder escolhido cedo demais ainda não vale.
    expect(characterDamageReduction(comPoder(11), characterEffects(comPoder(11))).total).toBe(0)
    expect(characterDamageReduction(comPoder(12), characterEffects(comPoder(12))).total).toBe(5)
    // Fixa: não escala como escalava a progressão inventada.
    expect(characterDamageReduction(comPoder(20), characterEffects(comPoder(20))).total).toBe(5)
  })

  it('RD geral não acumula entre classes — vale a maior (p290)', () => {
    const c = character({
      classes: [
        { className: 'Bárbaro', level: 17 },
        { className: 'Guerreiro', level: 5 },
      ],
      items: [heavyGear()],
    })
    expect(characterDamageReduction(c, characterEffects(c)).total).toBe(10)
  })

  it('Cavaleiro Bastião: RD 5 em armadura pesada quando o caminho foi escolhido', () => {
    const c = character({
      classes: [{ className: 'Cavaleiro', level: 5 }],
      classPowers: JSON.stringify(['caminho-bastiao']),
      items: [heavyGear()],
    })
    expect(characterDamageReduction(c, characterEffects(c)).total).toBe(5)
  })

  it('Especialização em Armadura acumula com Bastião (texto explícito, p54)', () => {
    const c = character({
      classes: [{ className: 'Cavaleiro', level: 12 }],
      classPowers: JSON.stringify([
        'class.cavaleiro.caminho-bastiao',
        'class.cavaleiro.especializacao-em-armadura',
      ]),
      items: [heavyGear()],
    })
    expect(characterDamageReduction(c, characterEffects(c)).total).toBe(10)
  })
})

describe('tempHpFromPowers — Alma de Bronze na Fúria (p41)', () => {
  const barbaro = (powers: string[]) =>
    character({
      level: 6,
      strength: 4,
      classes: [{ className: 'Bárbaro', level: 6 }],
      classPowers: JSON.stringify(powers),
    })

  it('fúria ativa + poder escolhido → PV temp = nível + Força', () => {
    const c = barbaro(['class.barbaro.alma-de-bronze'])
    const out = tempHpFromPowers(c, characterEffects(c), true)
    expect(out.total).toBe(10)
    expect(out.sources[0].source).toContain('Alma de Bronze')
  })

  it('sem fúria ativa não concede nada', () => {
    const c = barbaro(['class.barbaro.alma-de-bronze'])
    expect(tempHpFromPowers(c, characterEffects(c), false).total).toBe(0)
  })

  it('sem o poder escolhido não concede nada', () => {
    const c = barbaro([])
    expect(tempHpFromPowers(c, characterEffects(c), true).total).toBe(0)
  })
})
