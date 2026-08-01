import { describe, expect, it } from 'vitest'
import type { AttributeKey } from './attributes'
import { computeCharacterSheet } from './character-sheet'
import { collectVitalGrants, evalVitalScale } from './vital-grants'

const attrs = (
  o: Partial<Record<AttributeKey, number>> = {},
): Record<AttributeKey, number> => ({
  strength: 0,
  dexterity: 0,
  constitution: 0,
  intelligence: 0,
  wisdom: 0,
  charisma: 0,
  ...o,
})

describe('evalVitalScale', () => {
  it('flat (or omitted) returns the amount unchanged', () => {
    expect(evalVitalScale(3, undefined, 5, attrs())).toBe(3)
    expect(evalVitalScale(3, { per: 'flat' }, 5, attrs())).toBe(3)
  })

  it('level multiplies by character level', () => {
    expect(evalVitalScale(1, { per: 'level' }, 8, attrs())).toBe(8)
  })

  it('levelStep floors for "a cada dois níveis"', () => {
    expect(evalVitalScale(1, { per: 'levelStep', step: 2, round: 'down' }, 10, attrs())).toBe(5)
    expect(evalVitalScale(1, { per: 'levelStep', step: 2, round: 'down' }, 9, attrs())).toBe(4)
  })

  it('levelStep ceils for "a cada nível ímpar"', () => {
    expect(evalVitalScale(1, { per: 'levelStep', step: 2, round: 'up' }, 7, attrs())).toBe(4)
  })

  it('attribute multiplies by the attribute total', () => {
    expect(evalVitalScale(1, { per: 'attribute', attribute: 'wisdom' }, 3, attrs({ wisdom: 4 }))).toBe(4)
  })
})

describe('collectVitalGrants — permanent max-pool grants', () => {
  it('Anão Duro como Pedra: +3 no 1º nível +1/nível = nível + 2 PV', () => {
    expect(collectVitalGrants({ level: 1, className: 'Guerreiro', raceId: 'anao', attrTotals: attrs() }).pv).toBe(3)
    expect(collectVitalGrants({ level: 5, className: 'Guerreiro', raceId: 'anao', attrTotals: attrs() }).pv).toBe(7)
  })

  it('Elfo Sangue Mágico: +1 PM por nível', () => {
    expect(collectVitalGrants({ level: 8, className: 'Guerreiro', raceId: 'elfo', attrTotals: attrs() }).pm).toBe(8)
  })

  it('Clérigo Magia Divina (auto): +Sabedoria no PM — sem powerIds', () => {
    const g = collectVitalGrants({ level: 3, className: 'Clérigo', attrTotals: attrs({ wisdom: 4 }) })
    expect(g.pm).toBe(4)
  })

  it('Paladino Abençoado (auto): +Carisma no PM', () => {
    expect(collectVitalGrants({ level: 1, className: 'Paladino', attrTotals: attrs({ charisma: 3 }) }).pm).toBe(3)
  })

  it('Bárbaro Totem Espiritual (elective): +Sabedoria no PM só quando escolhido', () => {
    const base = { level: 6, className: 'Bárbaro', attrTotals: attrs({ wisdom: 2 }) }
    expect(collectVitalGrants(base).pm).toBe(0)
    expect(collectVitalGrants({ ...base, powerIds: ['class.barbaro.totem-espiritual'] }).pm).toBe(2)
  })

  it('Arcanista Poder Mágico (elective): +1 PM por nível quando escolhido', () => {
    expect(
      collectVitalGrants({ level: 7, className: 'Arcanista', powerIds: ['class.arcanista.poder-magico'], attrTotals: attrs() }).pm,
    ).toBe(7)
  })

  it('Vitalidade (poder geral): +1 PV por nível quando escolhido', () => {
    expect(
      collectVitalGrants({ level: 6, className: 'Guerreiro', powerIds: ['vitalidade'], attrTotals: attrs() }).pv,
    ).toBe(6)
  })

  it('Vontade de Ferro (origem): +1 PM a cada 2 níveis quando o benefício é escolhido', () => {
    const base = { level: 10, className: 'Guerreiro', origin: 'Acólito', attrTotals: attrs() }
    expect(collectVitalGrants(base).pm).toBe(0)
    expect(collectVitalGrants({ ...base, originChoices: ['poder-vontade-de-ferro'] }).pm).toBe(5)
  })

  it('no race/powers/origin ⇒ zero grant', () => {
    expect(collectVitalGrants({ level: 5, className: 'Guerreiro', attrTotals: attrs() })).toEqual({ pv: 0, pm: 0 })
  })
})

describe('computeCharacterSheet — vitals fold in the grants', () => {
  const base = {
    baseAttributes: { strength: 0, dexterity: 0, constitution: 0, intelligence: 0, wisdom: 4, charisma: 0 },
  }

  it('Clérigo PM includes +Sabedoria (Magia Divina auto)', () => {
    // pmBase = mpPerLevel(5) * 3 = 15; +Sabedoria 4 = 19.
    const sheet = computeCharacterSheet({ level: 3, className: 'Clérigo', ...base })
    expect(sheet.vitals.pmMax).toBe(19)
  })

  it('Paladino PM unchanged after migrating the special-case to Abençoado', () => {
    // pmBase = 3 * 4 = 12; +Carisma 3 = 15.
    const sheet = computeCharacterSheet({
      level: 4,
      className: 'Paladino',
      baseAttributes: { strength: 0, dexterity: 0, constitution: 0, intelligence: 0, wisdom: 0, charisma: 3 },
    })
    expect(sheet.vitals.pmMax).toBe(15)
  })

  it('Anão Duro como Pedra folds in via the racas slug id ("anao")', () => {
    // Anão Guerreiro L10, Con base 4 → total 6 (raça +2). pvBase = 20 + 9*5 +
    // 6*10 = 125; Duro como Pedra = nível + 2 = 12 → 137. Guards the
    // racas-slug → abilities-name bridge (a name id would drop the grant).
    const sheet = computeCharacterSheet({
      level: 10,
      className: 'Guerreiro',
      raceId: 'anao',
      baseAttributes: { strength: 4, dexterity: 0, constitution: 4, intelligence: 0, wisdom: 0, charisma: 0 },
    })
    expect(sheet.attributes.constitution.total).toBe(6)
    expect(sheet.vitals.pvMax).toBe(137)
  })
})
