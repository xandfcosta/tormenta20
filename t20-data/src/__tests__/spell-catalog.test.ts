import { describe, expect, it } from 'vitest'
import { SPELL_SCHOOLS, type SpellSchool } from '../spells'
import {
  SPELL_CATALOG,
  SPELL_IDS,
  spellById,
  spellByName,
  spellEffectByName,
  spellsByCircle,
  spellsByClass,
  spellsBySchool,
} from '../spell-catalog'

/**
 * Spell seed catalog — PDF book Cap 4 (p174-211).
 *
 * Tests pin diversity coverage (escolas, resistências, save types,
 * execução, alcance, duração) so future trimming of the seed can't
 * silently regress to a less-representative set.
 */

describe('SPELL_CATALOG — buff spells', () => {
  it('models known self/ally buffs with well-formed modifiers', () => {
    const buffed = Object.values(SPELL_CATALOG).filter((s) => s.buff)
    expect(buffed.length).toBeGreaterThanOrEqual(10)
    for (const spell of buffed) {
      expect(['scene', 'day']).toContain(spell.buff!.defaultScope)
      // A buff must carry *something* — computed modifiers, display-only
      // facts, or both. Display-only buffs (RD, immunities, senses) ship with
      // an empty modifiers list and non-empty facts.
      const factCount = spell.buff!.facts?.length ?? 0
      expect(spell.buff!.modifiers.length + factCount).toBeGreaterThan(0)
      for (const m of spell.buff!.modifiers) {
        expect(typeof m.amount).toBe('number')
        expect(m.target).toHaveProperty('k')
        expect(typeof m.bonusType).toBe('string')
      }
      for (const f of spell.buff!.facts ?? []) {
        expect(typeof f.text).toBe('string')
        expect(f.text.length).toBeGreaterThan(0)
        expect(['dr', 'immunity', 'sense', 'movement', 'action', 'other']).toContain(
          f.category,
        )
      }
    }
  })

  it('spellEffectByName returns the base effect for a known name', () => {
    expect(spellEffectByName('Armadura Arcana')).toBe(
      spellById('armadura-arcana').baseEffect,
    )
  })

  it('spellEffectByName returns null for an unknown name', () => {
    expect(spellEffectByName('Não Existe Magia')).toBeNull()
  })

  it('spellByName resolves a display name to its catalog spell', () => {
    expect(spellByName('Armadura Arcana')?.id).toBe('armadura-arcana')
  })

  it('spellByName is accent/case-insensitive', () => {
    expect(spellByName('visao mistica')?.id).toBe('visao-mistica')
    expect(spellByName('VISÃO MÍSTICA')?.id).toBe('visao-mistica')
  })

  it('spellByName returns null for an unknown name', () => {
    expect(spellByName('Não Existe Magia')).toBeNull()
  })

  // spellByName's map would silently shadow on a dupe — pin uniqueness.
  it('normalized spell names are unique across the catalog', () => {
    const normalized = SPELL_IDS.map((id) =>
      spellById(id)
        .name.normalize('NFD')
        .replace(/[\u0300-\u036f]/g, '')
        .toLowerCase(),
    )
    expect(new Set(normalized).size).toBe(normalized.length)
  })

  it('Armadura Arcana grants +5 Defesa', () => {
    const mod = spellById('armadura-arcana').buff!.modifiers[0]!
    expect(mod.target).toEqual({ k: 'defense' })
    expect(mod.amount).toBe(5)
  })

  it('Bênção grants +1 attack and +1 damage (scope all)', () => {
    const mods = spellById('bencao').buff!.modifiers
    expect(mods).toContainEqual(
      expect.objectContaining({ target: { k: 'attack', scope: 'all' }, amount: 1 }),
    )
    expect(mods).toContainEqual(
      expect.objectContaining({ target: { k: 'damage', scope: 'all' }, amount: 1 }),
    )
  })

  // Regression: these self/ally buffs were book-described but unannotated, so
  // they never showed up in the "Aplicar efeito" dialog (buff-only list).
  it('Campo de Força grants 30 temporary HP', () => {
    const mods = spellById('campo-de-forca').buff!.modifiers
    expect(mods).toContainEqual(
      expect.objectContaining({ target: { k: 'tempHp' }, amount: 30 }),
    )
  })

  it('Proteção Divina grants +2 resistance', () => {
    const mods = spellById('protecao-divina').buff!.modifiers
    expect(mods).toContainEqual(
      expect.objectContaining({ target: { k: 'resistance' }, amount: 2 }),
    )
  })

  it('Arma Mágica grants +1 attack and +1 damage (enhancement)', () => {
    const mods = spellById('arma-magica').buff!.modifiers
    expect(mods).toContainEqual(
      expect.objectContaining({
        target: { k: 'attack', scope: 'all' },
        amount: 1,
        bonusType: 'enhancement',
      }),
    )
    expect(mods).toContainEqual(
      expect.objectContaining({ target: { k: 'damage', scope: 'all' }, amount: 1 }),
    )
  })

  it('Aura Divina grants +10 Defesa and +10 resistance to the caster', () => {
    const mods = spellById('aura-divina').buff!.modifiers
    expect(mods).toContainEqual(
      expect.objectContaining({ target: { k: 'defense' }, amount: 10 }),
    )
    expect(mods).toContainEqual(
      expect.objectContaining({ target: { k: 'resistance' }, amount: 10 }),
    )
  })

  it('Transformação de Guerra grants +6 Def/atk/dano and 30 temp HP', () => {
    const mods = spellById('transformacao-de-guerra').buff!.modifiers
    expect(mods).toContainEqual(
      expect.objectContaining({ target: { k: 'defense' }, amount: 6 }),
    )
    expect(mods).toContainEqual(
      expect.objectContaining({ target: { k: 'tempHp' }, amount: 30 }),
    )
  })

  // Display-only facts: spells with no computable modifier still appear in the
  // apply dialog and surface their mechanical text as reference chips.
  it('Pele de Pedra is a display-only DR buff (no modifiers, one dr fact)', () => {
    const buff = spellById('pele-de-pedra').buff!
    expect(buff.modifiers).toHaveLength(0)
    expect(buff.facts).toContainEqual({ category: 'dr', text: 'RD 5' })
  })

  it('Velocidade carries an action-economy fact', () => {
    const facts = spellById('velocidade').buff!.facts ?? []
    expect(facts.some((f) => f.category === 'action')).toBe(true)
  })

  it('Heroísmo keeps its numeric buff and adds an immunity fact', () => {
    const buff = spellById('heroismo').buff!
    expect(buff.modifiers.length).toBeGreaterThan(0)
    expect(buff.facts).toContainEqual({
      category: 'immunity',
      text: 'Imune a medo',
    })
  })
})

describe('SPELL_CATALOG — shape & invariants', () => {
  it('has at least 14 entries', () => {
    expect(SPELL_IDS.length).toBeGreaterThanOrEqual(14)
  })

  it('all ids are unique', () => {
    expect(new Set(SPELL_IDS).size).toBe(SPELL_IDS.length)
  })

  it('every entry round-trips through spellById', () => {
    for (const id of SPELL_IDS) {
      const spell = spellById(id)
      expect(spell.id).toBe(id)
    }
  })

  it('spellById throws on unknown id', () => {
    expect(() => spellById('not-a-real-spell')).toThrow(/unknown spell id/)
  })

  it('every entry has non-empty name + baseEffect', () => {
    for (const id of SPELL_IDS) {
      const spell = spellById(id)
      expect(spell.name).toBeTruthy()
      expect(spell.baseEffect).toBeTruthy()
    }
  })

  it('every entry has a positive book page', () => {
    for (const id of SPELL_IDS) {
      expect(spellById(id).bookPage).toBeGreaterThan(0)
    }
  })

  it('every entry has at least one class', () => {
    for (const id of SPELL_IDS) {
      expect(spellById(id).classes.length).toBeGreaterThan(0)
    }
  })

  it('definida duration entries carry a durationNote', () => {
    for (const id of SPELL_IDS) {
      const spell = spellById(id)
      if (spell.duration === 'definida') {
        expect(spell.durationNote).toBeTruthy()
      }
    }
  })

  it("saveType === 'none' implies resistance is null", () => {
    for (const id of SPELL_IDS) {
      const spell = spellById(id)
      if (spell.saveType === 'none') {
        expect(spell.resistance).toBeNull()
      }
    }
  })

  it('augment pmCost is non-negative', () => {
    for (const id of SPELL_IDS) {
      for (const aug of spellById(id).augments) {
        expect(aug.pmCost).toBeGreaterThanOrEqual(0)
      }
    }
  })
})

describe('SPELL_CATALOG — diversity coverage', () => {
  it('every escola is represented at least once', () => {
    for (const school of SPELL_SCHOOLS) {
      const found = spellsBySchool(school as SpellSchool)
      expect(found.length, `escola ${school} missing from seed`).toBeGreaterThan(0)
    }
  })

  it('círculos 1, 2 are both represented', () => {
    expect(spellsByCircle(1).length).toBeGreaterThan(0)
    expect(spellsByCircle(2).length).toBeGreaterThan(0)
  })

  it.each([['fortitude'], ['reflexos'], ['vontade'], ['none']] as const)(
    'save type %s appears at least once',
    (save) => {
      const found = SPELL_IDS.filter((id) => spellById(id).saveType === save)
      expect(found.length).toBeGreaterThan(0)
    },
  )

  it.each([['anula'], ['parcial'], ['metade'], ['desacredita']] as const)(
    'resistance type %s appears at least once',
    (res) => {
      const found = SPELL_IDS.filter((id) => spellById(id).resistance === res)
      expect(found.length).toBeGreaterThan(0)
    },
  )

  it('execução variants livre and reação both appear', () => {
    const livre = SPELL_IDS.filter((id) => spellById(id).execution === 'livre')
    const reacao = SPELL_IDS.filter((id) => spellById(id).execution === 'reacao')
    expect(livre.length).toBeGreaterThan(0)
    expect(reacao.length).toBeGreaterThan(0)
  })

  it.each([['pessoal'], ['toque'], ['curto'], ['medio']] as const)(
    'alcance %s appears at least once',
    (range) => {
      const found = SPELL_IDS.filter((id) => spellById(id).range === range)
      expect(found.length).toBeGreaterThan(0)
    },
  )

  it.each([
    ['instantanea'],
    ['cena'],
    ['sustentada'],
    ['definida'],
  ] as const)('duração %s appears at least once', (dur) => {
    const found = SPELL_IDS.filter((id) => spellById(id).duration === dur)
    expect(found.length).toBeGreaterThan(0)
  })
})

describe('spellsByClass — class-list filters', () => {
  it('Arcanista list is non-empty', () => {
    expect(spellsByClass('Arcanista').length).toBeGreaterThan(0)
  })

  it('Clérigo list includes Curar Ferimentos', () => {
    const ids = spellsByClass('Clérigo').map((s) => s.id)
    expect(ids).toContain('curar-ferimentos')
  })

  it('Paladino list is non-empty (shares divine spells)', () => {
    expect(spellsByClass('Paladino').length).toBeGreaterThan(0)
  })
})

describe('SPELL_CATALOG — pinned key entries (PDF integrity)', () => {
  it('Bola de Fogo: círculo 2, evocação, médio, Reflexos metade, p182', () => {
    const spell = spellById('bola-de-fogo')
    expect(spell.circle).toBe(2)
    expect(spell.school).toBe('evocacao')
    expect(spell.range).toBe('medio')
    expect(spell.saveType).toBe('reflexos')
    expect(spell.resistance).toBe('metade')
    expect(spell.bookPage).toBe(182)
  })

  it('Invisibilidade: livre, duração definida 1 rodada, p195', () => {
    const spell = spellById('invisibilidade')
    expect(spell.execution).toBe('livre')
    expect(spell.duration).toBe('definida')
    expect(spell.durationNote).toMatch(/1 rodada/)
    expect(spell.bookPage).toBe(195)
  })

  it('Queda Suave: reação, círculo 1, transmutação, p202', () => {
    const spell = spellById('queda-suave')
    expect(spell.execution).toBe('reacao')
    expect(spell.circle).toBe(1)
    expect(spell.school).toBe('transmutacao')
    expect(spell.bookPage).toBe(202)
  })

  it('Conjurar Monstro: sustentada, convocação, sem teste de resistência', () => {
    const spell = spellById('conjurar-monstro')
    expect(spell.duration).toBe('sustentada')
    expect(spell.school).toBe('convocacao')
    expect(spell.saveType).toBe('none')
    expect(spell.resistance).toBeNull()
  })

  it('Criar Ilusão: desacredita resistência (Vontade), p189', () => {
    const spell = spellById('criar-ilusao')
    expect(spell.resistance).toBe('desacredita')
    expect(spell.saveType).toBe('vontade')
    expect(spell.school).toBe('ilusao')
    expect(spell.bookPage).toBe(189)
  })

  it('Curar Ferimentos: toque, instantânea, classes divinas', () => {
    const spell = spellById('curar-ferimentos')
    expect(spell.range).toBe('toque')
    expect(spell.duration).toBe('instantanea')
    expect(spell.classes).toEqual(
      expect.arrayContaining(['Clérigo', 'Druida', 'Paladino']),
    )
    expect(spell.classes).not.toContain('Arcanista')
  })
})

describe('SPELL_CATALOG — augment metadata', () => {
  it('Enfeitiçar carries a círculo-3 gated augment', () => {
    const enfeit = spellById('enfeiticar')
    expect(enfeit.augments.some((a) => a.requiresCircle === 3)).toBe(true)
  })

  it('Luz carries an arcanos-only augment', () => {
    const luz = spellById('luz')
    expect(luz.augments.some((a) => a.classOnly === 'arcanos')).toBe(true)
  })

  it("augments use 'aumenta' or 'muda' kinds only", () => {
    for (const id of SPELL_IDS) {
      for (const aug of spellById(id).augments) {
        expect(['aumenta', 'muda']).toContain(aug.kind)
      }
    }
  })
})

describe('SPELL_CATALOG immutability', () => {
  it('cannot mutate the SPELL_CATALOG record at runtime', () => {
    expect(() => {
      // @ts-expect-error — guarded by Object.freeze, runtime should throw
      SPELL_CATALOG['ghost'] = {} as never
    }).toThrow()
  })
})
