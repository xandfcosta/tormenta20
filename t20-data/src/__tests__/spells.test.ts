import { describe, expect, it } from 'vitest'
import {
  CLASS_SPELLCASTING_ATTRIBUTE,
  DEFAULT_SPELL_COMPONENTS,
  SPELL_BASE_PM_COST,
  SPELL_RANGE_METERS,
  SPELL_SCHOOLS,
  SUSTAIN_PM_PER_TURN,
  concentrationCd,
  spellPmCost,
  spellSaveDc,
  totalAugmentPm,
  type SpellAugment,
} from '../spells'
import { spellcastingAttributeFor } from '../class-spellcasting'

/**
 * PDF book Cap 4 — Magia (p168-211). Pinned mechanics:
 *  - Custo em PM por círculo (Tabela 4-1, p170): 1/3/6/10/15 PM (truque = 0)
 *  - CD = 10 + ½ nível do PERSONAGEM + atributo-chave (p173)
 *  - Componentes default: gestos + palavras
 *  - 8 escolas
 *  - Sustentar: 1 PM/turno
 *  - Truques não recebem aprimoramentos
 *  - 'muda' não acumula; 'aumenta' acumula
 *  - Concentração: CD 15 / 15+custo / 20+custo
 */

describe('SPELL_BASE_PM_COST — PDF p170', () => {
  it('matches the PM cost table exactly', () => {
    expect(SPELL_BASE_PM_COST[0]).toBe(0)
    expect(SPELL_BASE_PM_COST[1]).toBe(1)
    expect(SPELL_BASE_PM_COST[2]).toBe(3)
    expect(SPELL_BASE_PM_COST[3]).toBe(6)
    expect(SPELL_BASE_PM_COST[4]).toBe(10)
    expect(SPELL_BASE_PM_COST[5]).toBe(15)
  })
})

describe('SPELL_SCHOOLS — PDF p170', () => {
  it('lists exactly the 8 canonical escolas', () => {
    expect([...SPELL_SCHOOLS].sort()).toEqual(
      [
        'abjuracao',
        'adivinhacao',
        'convocacao',
        'encantamento',
        'evocacao',
        'ilusao',
        'necromancia',
        'transmutacao',
      ].sort(),
    )
  })
})

describe('SPELL_RANGE_METERS — PDF p172', () => {
  it('curto = 9m, medio = 30m, longo = 90m', () => {
    expect(SPELL_RANGE_METERS.curto).toBe(9)
    expect(SPELL_RANGE_METERS.medio).toBe(30)
    expect(SPELL_RANGE_METERS.longo).toBe(90)
  })

  it('pessoal and toque are 0 (no horizontal distance)', () => {
    expect(SPELL_RANGE_METERS.pessoal).toBe(0)
    expect(SPELL_RANGE_METERS.toque).toBe(0)
  })

  it('ilimitado is Infinity (same plane)', () => {
    expect(SPELL_RANGE_METERS.ilimitado).toBe(Infinity)
  })
})

describe('DEFAULT_SPELL_COMPONENTS — PDF p173', () => {
  it('every magia defaults to verbal + gestual', () => {
    expect([...DEFAULT_SPELL_COMPONENTS].sort()).toEqual(['gestual', 'verbal'])
  })
})

describe('SUSTAIN_PM_PER_TURN — PDF p173', () => {
  it('sustained magias cost 1 PM/turn upkeep', () => {
    expect(SUSTAIN_PM_PER_TURN).toBe(1)
  })
})

/**
 * O atributo-chave, ALE-113. O caso "Arcanista uses Inteligência" morava aqui,
 * VERDE, defendendo uma regra que o livro não tem: a p37 diz "Seu atributo-chave
 * para lançar magias é definido pelo seu Caminho", e o feiticeiro lança por
 * Carisma. O exemplo trabalhado da p173 é justamente uma feiticeira.
 *
 * O mapa agora é só o que a entrada de classe FIXA (e o predicado de "lança
 * magias"); quem calcula CD passa por `spellcastingAttributeFor`.
 */
describe('atributo-chave de conjuração', () => {
  it('as classes de atributo fixo, direto do mapa', () => {
    expect(CLASS_SPELLCASTING_ATTRIBUTE.Bardo).toBe('charisma')
    expect(CLASS_SPELLCASTING_ATTRIBUTE.Clérigo).toBe('wisdom')
    expect(CLASS_SPELLCASTING_ATTRIBUTE.Druida).toBe('wisdom')
    expect(CLASS_SPELLCASTING_ATTRIBUTE.Paladino).toBe('wisdom')
  })

  // p37: Bruxo e Mago → Inteligência, Feiticeiro → Carisma. Trocar o caminho e
  // ver o mesmo atributo é exatamente o bug que chegou à tela.
  it('o Arcanista segue o Caminho', () => {
    expect(spellcastingAttributeFor('Arcanista', 'bruxo')).toBe('intelligence')
    expect(spellcastingAttributeFor('Arcanista', 'mago')).toBe('intelligence')
    expect(spellcastingAttributeFor('Arcanista', 'feiticeiro')).toBe('charisma')
  })

  it('sem caminho escolhido, o Arcanista cai em Inteligência', () => {
    expect(spellcastingAttributeFor('Arcanista', undefined)).toBe('intelligence')
    expect(spellcastingAttributeFor('Arcanista', 'lixo')).toBe('intelligence')
  })

  // O caminho só manda no Arcanista — passá-lo para outra classe não a move.
  it('o caminho não contamina as outras classes', () => {
    expect(spellcastingAttributeFor('Clérigo', 'feiticeiro')).toBe('wisdom')
    expect(spellcastingAttributeFor('Guerreiro', 'mago')).toBeUndefined()
  })
})

describe('spellSaveDc — PDF p173', () => {
  it('CD = 10 + ½ nível + atributo at level 1, mod +3 → 13', () => {
    expect(spellSaveDc(1, 3)).toBe(13)
  })

  it('½ nível rounds down (level 5 + mod 0 → 10 + 2 = 12)', () => {
    expect(spellSaveDc(5, 0)).toBe(12)
  })

  it('level 10 + mod 4 → 10 + 5 + 4 = 19', () => {
    expect(spellSaveDc(10, 4)).toBe(19)
  })

  it('level 20 + mod 5 → 10 + 10 + 5 = 25', () => {
    expect(spellSaveDc(20, 5)).toBe(25)
  })

  it('rejects level 0', () => {
    expect(() => spellSaveDc(0, 3)).toThrow(/casterLevel must be ≥ 1/)
  })

  it('handles negative attribute mod (penalty)', () => {
    expect(spellSaveDc(1, -1)).toBe(9)
  })
})

describe('spellPmCost — base + augments', () => {
  it('truque always costs 0', () => {
    expect(spellPmCost(0)).toBe(0)
  })

  it('círculo 1 base + 2 PM augment = 3', () => {
    expect(spellPmCost(1, 2)).toBe(3)
  })

  it('círculo 5 base + 5 PM augment = 20', () => {
    expect(spellPmCost(5, 5)).toBe(20)
  })

  it('rejects augment on truque (PDF p171)', () => {
    expect(() => spellPmCost(0, 1)).toThrow(/truques cannot receive augments/)
  })

  it('rejects negative augment PM', () => {
    expect(() => spellPmCost(1, -1)).toThrow(/augmentPm must be ≥ 0/)
  })
})

describe('concentrationCd — PDF p173', () => {
  it('normal distraction → CD 15 (cost-independent)', () => {
    expect(concentrationCd('normal', 6)).toBe(15)
  })

  it('ruim → 15 + spell total PM', () => {
    expect(concentrationCd('ruim', 6)).toBe(21)
  })

  it('terrível → 20 + spell total PM', () => {
    expect(concentrationCd('terrivel', 10)).toBe(30)
  })
})

describe('totalAugmentPm — stack rules', () => {
  const aumenta: SpellAugment = {
    id: 'plus-die',
    kind: 'aumenta',
    pmCost: 1,
    description: '+1 dado de dano',
  }
  const muda: SpellAugment = {
    id: 'change-target',
    kind: 'muda',
    pmCost: 2,
    description: 'muda alvo único para múltiplo',
  }

  it('single aumenta stack = pmCost', () => {
    expect(totalAugmentPm(2, [{ augment: aumenta, stacks: 1 }])).toBe(1)
  })

  it('aumenta accumulates (3 stacks × 1 PM = 3)', () => {
    expect(totalAugmentPm(2, [{ augment: aumenta, stacks: 3 }])).toBe(3)
  })

  it('mixed: 2 aumenta + 1 muda = 2 + 2 = 4', () => {
    expect(
      totalAugmentPm(2, [
        { augment: aumenta, stacks: 2 },
        { augment: muda, stacks: 1 },
      ]),
    ).toBe(4)
  })

  it("rejects 'muda' taken twice", () => {
    expect(() =>
      totalAugmentPm(2, [{ augment: muda, stacks: 2 }]),
    ).toThrow(/'muda' augment .* cannot stack/)
  })

  it('rejects 0 or negative stacks', () => {
    expect(() =>
      totalAugmentPm(2, [{ augment: aumenta, stacks: 0 }]),
    ).toThrow(/stacks must be ≥ 1/)
  })

  it('rejects any augments on truque', () => {
    expect(() =>
      totalAugmentPm(0, [{ augment: aumenta, stacks: 1 }]),
    ).toThrow(/truques cannot receive aprimoramentos/)
  })

  it('empty augment list on truque is allowed (returns 0)', () => {
    expect(totalAugmentPm(0, [])).toBe(0)
  })
})
