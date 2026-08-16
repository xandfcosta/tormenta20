import { computedSheetFor } from './computed-sheet'
import { describe, expect, it } from 'vitest'
import { conditionalId, statFor } from '@/shared/rules/items-engine'
import type { ClassChoices } from '@/shared/api/catalog-types'
import type { Character, CharacterItem } from '@/shared/api/api'
import {
  characterEffects,
  evaluatePrerequisite,
  isItemProficient,
  parseClassChoices,
  parseImprovementIds,
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

describe('characterEffects — empty character integration', () => {
  it('returns empty ItemEffects shape for a character with no items / races / classes', () => {
    const c = character()
    const effects = characterEffects(c)
    expect(effects.byTarget).toEqual({})
    expect(effects.flags.size).toBe(0)
    expect(effects.conditional).toEqual([])
  })
})

describe('a linha da arma chega em Luta pelo MOTOR', () => {
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

  // Lê pelo MESMO caminho da tela (o motor), não pela cópia TS que a produção
  // não executa mais: o bônus da arma só vale se chegar na ficha computada.
  // A ficha computada só devolve breakdown das perícias que a FICHA lista, e o
  // molde local nasce sem nenhuma — sem esta linha o motor não tem Luta para
  // espelhar o bônus da arma, e o teste falharia por fixture, não por regra.
  const comLuta = (over: Partial<Character> = {}) =>
    character({
      expertises: [
        { name: 'Luta', attribute: 'strength', trained: false, custom: false },
      ] as Character['expertises'],
      ...over,
    })

  const luta = (c: Character, conditionals?: ReadonlySet<string>) => {
    const found = computedSheetFor(c, conditionals).expertises.find((e) => e.name === 'Luta')
    if (!found) throw new Error('a ficha computada não trouxe a perícia Luta')
    return found
  }

  it('melhoria Certeira soma +1 na Luta via overlay', () => {
    const c = comLuta({
      items: [
        weaponItem('espada-curta', {
          equipped: 'wielded',
          improvements: JSON.stringify(['melhoria-certeira']),
        }),
      ],
    })
    expect(luta(c).itemBonus).toBe(1)
  })

  it('arma guardada (não equipada) não espelha nada', () => {
    const c = comLuta({
      proficiencies: JSON.stringify(['armas-exoticas']),
      items: [weaponItem('machado-taurico', { equipped: null })],
    })
    expect(luta(c).itemBonus).toBe(0)
  })

  describe('homebrew: Equilibrada anula desbalanceada (toggle no Efeitos)', () => {
    const tauricoEquilibrado = () =>
      comLuta({
        proficiencies: JSON.stringify(['armas-exoticas']),
        items: [
          weaponItem('machado-taurico', {
            improvements: JSON.stringify(['melhoria-equilibrada']),
          }),
        ],
      })

    it('RAW por padrão: sem o toggle, o -2 permanece', () => {
      expect(luta(tauricoEquilibrado()).itemBonus).toBe(-2)
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
      expect(luta(c, new Set([conditionalId(cond)])).itemBonus).toBe(0)
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
    expect(computedSheetFor(medalhao('vested')).pmLimit.contributions).toEqual([])
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
    expect(computedSheetFor(c, new Set([conditionalId(cond)])).pmLimit.total).toBe(
      computedSheetFor(c).pmLimit.total + 1,
    )
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
    expect(computedSheetFor(c).defense.contributions).toContainEqual(
      expect.objectContaining({
        source: 'Couraça',
        amount: 1,
        note: 'Reforçada: +1 Defesa',
      }),
    )
  })
})
