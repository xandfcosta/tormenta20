import { describe, expect, it } from 'vitest'
import type { ClassChoices, ItemEffects } from '@tormenta20/t20-data'
import type { Character, CharacterItem } from './api'
import {
  armorPenaltyTotal,
  attributeContributions,
  attributeTotal,
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
  it('10 + 2 * |strength| with no items', () => {
    expect(inventorySlotsTotal(character({ strength: 3 }), emptyEffects())).toBe(16)
    expect(inventorySlotsTotal(character({ strength: 0 }), emptyEffects())).toBe(10)
  })

  it('uses |strength| even when negative', () => {
    expect(inventorySlotsTotal(character({ strength: -2 }), emptyEffects())).toBe(14)
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

describe('pmLimitTotal — PDF gating ½ level (min 1)', () => {
  it('L1 → base 1 (min)', () => {
    expect(pmLimitTotal(character({ level: 1 }), emptyEffects()).total).toBe(1)
  })

  it('L7 → base 3 (½ rounds down)', () => {
    expect(pmLimitTotal(character({ level: 7 }), emptyEffects()).total).toBe(3)
  })

  it('L20 → base 10', () => {
    expect(pmLimitTotal(character({ level: 20 }), emptyEffects()).total).toBe(10)
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
