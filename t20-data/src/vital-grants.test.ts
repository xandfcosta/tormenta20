import { describe, expect, it } from 'vitest'
import { defaultVitalResolver } from './abilities/vital-resolver'
import type { AttributeKey } from './attributes'
import { computeCharacterSheet } from './character-sheet'
import {
  collectVitalGrants,
  evalVitalScale,
  type VitalGrantContext,
} from './vital-grants'

/** Engine-side wrapper: bind the data-backed resolver (what the backend uses)
 *  so these rule tests read as before the DI refactor (B.3). */
const collect = (ctx: VitalGrantContext) =>
  collectVitalGrants(ctx, defaultVitalResolver)

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
    expect(collect({ level: 1, className: 'Guerreiro', raceId: 'anao', attrTotals: attrs() }).pv).toBe(3)
    expect(collect({ level: 5, className: 'Guerreiro', raceId: 'anao', attrTotals: attrs() }).pv).toBe(7)
  })

  it('Elfo Sangue Mágico: +1 PM por nível', () => {
    expect(collect({ level: 8, className: 'Guerreiro', raceId: 'elfo', attrTotals: attrs() }).pm).toBe(8)
  })

  it('Clérigo Magia Divina (auto): +Sabedoria no PM — sem powerIds', () => {
    const g = collect({ level: 3, className: 'Clérigo', attrTotals: attrs({ wisdom: 4 }) })
    expect(g.pm).toBe(4)
  })

  it('Paladino Abençoado (auto): +Carisma no PM', () => {
    expect(collect({ level: 1, className: 'Paladino', attrTotals: attrs({ charisma: 3 }) }).pm).toBe(3)
  })

  it('Bárbaro Totem Espiritual (elective): +Sabedoria no PM só quando escolhido', () => {
    const base = { level: 6, className: 'Bárbaro', attrTotals: attrs({ wisdom: 2 }) }
    expect(collect(base).pm).toBe(0)
    expect(collect({ ...base, powerIds: ['class.barbaro.totem-espiritual'] }).pm).toBe(2)
  })

  it('Arcanista Poder Mágico (elective): +1 PM por nível quando escolhido', () => {
    expect(
      collect({ level: 7, className: 'Arcanista', powerIds: ['class.arcanista.poder-magico'], attrTotals: attrs() }).pm,
    ).toBe(7)
  })

  it('Vitalidade (poder geral): +1 PV por nível quando escolhido', () => {
    expect(
      collect({ level: 6, className: 'Guerreiro', powerIds: ['vitalidade'], attrTotals: attrs() }).pv,
    ).toBe(6)
  })

  it('Vontade de Ferro (origem): +1 PM a cada 2 níveis quando o benefício é escolhido', () => {
    const base = { level: 10, className: 'Guerreiro', origin: 'Acólito', attrTotals: attrs() }
    expect(collect(base).pm).toBe(0)
    expect(collect({ ...base, originChoices: ['poder-vontade-de-ferro'] }).pm).toBe(5)
  })

  it('no race/powers/origin ⇒ zero grant', () => {
    expect(collect({ level: 5, className: 'Guerreiro', attrTotals: attrs() })).toEqual({ pv: 0, pm: 0 })
  })

  // "soma seu Carisma no seu total de PM" (Bardo p43) / "soma sua Sabedoria
  // no seu total de PM" (Druida p60) — same auto rule the Clérigo already had.
  it('Bardo Magias (auto): +Carisma no PM', () => {
    expect(collect({ level: 2, className: 'Bardo', attrTotals: attrs({ charisma: 3 }) }).pm).toBe(3)
  })

  it('Druida Magias (auto): +Sabedoria no PM', () => {
    expect(collect({ level: 2, className: 'Druida', attrTotals: attrs({ wisdom: 4 }) }).pm).toBe(4)
  })

  // Arcanista p36-37: "Seu atributo-chave para lançar magias é definido pelo
  // seu Caminho... e você soma seu atributo-chave no seu total de PM."
  // Bruxo = Int, Mago = Int, Feiticeiro = Car. Ownership comes from
  // classChoices.caminho, not from a power slot.
  it('Arcanista caminho Mago: +Inteligência no PM via classChoices', () => {
    const base = { level: 1, className: 'Arcanista', attrTotals: attrs({ intelligence: 4, charisma: 2 }) }
    expect(collect(base).pm).toBe(0)
    expect(
      collect({ ...base, classChoices: { Arcanista: { caminho: 'mago' } } }).pm,
    ).toBe(4)
  })

  it('Arcanista caminho Bruxo: +Inteligência no PM (p37 — não Carisma)', () => {
    expect(
      collect({
        level: 1,
        className: 'Arcanista',
        classChoices: { Arcanista: { caminho: 'bruxo' } },
        attrTotals: attrs({ intelligence: 3, charisma: 5 }),
      }).pm,
    ).toBe(3)
  })

  it('Arcanista caminho Feiticeiro: +Carisma no PM', () => {
    expect(
      collect({
        level: 1,
        className: 'Arcanista',
        classChoices: { Arcanista: { caminho: 'feiticeiro' } },
        attrTotals: attrs({ intelligence: 3, charisma: 5 }),
      }).pm,
    ).toBe(5)
  })

  it('multiclasse: grants de TODAS as classes entram (Guerreiro+Arcanista mago)', () => {
    const g = collect({
      level: 8,
      className: 'Guerreiro',
      classes: [
        { className: 'Guerreiro', level: 4 },
        { className: 'Arcanista', level: 4 },
      ],
      classChoices: { Arcanista: { caminho: 'mago' } },
      attrTotals: attrs({ intelligence: 3 }),
    })
    expect(g.pm).toBe(3)
  })

  it('p225: Clérigo/Druida NÃO soma Sabedoria duas vezes no PM', () => {
    const g = collect({
      level: 2,
      className: 'Clérigo',
      classes: [
        { className: 'Clérigo', level: 1 },
        { className: 'Druida', level: 1 },
      ],
      attrTotals: attrs({ wisdom: 4 }),
    })
    expect(g.pm).toBe(4)
  })

  it('atributos DIFERENTES somam (Paladino Car + Clérigo Sab)', () => {
    const g = collect({
      level: 2,
      className: 'Paladino',
      classes: [
        { className: 'Paladino', level: 1 },
        { className: 'Clérigo', level: 1 },
      ],
      attrTotals: attrs({ wisdom: 4, charisma: 3 }),
    })
    expect(g.pm).toBe(7)
  })

  it('dedupe por atributo não engole grants com scale diferente (Poder Mágico)', () => {
    // Caminho mago (+Int) + Poder Mágico (+1/nível): scales distintos, ambos valem.
    const g = collect({
      level: 5,
      className: 'Arcanista',
      classChoices: { Arcanista: { caminho: 'mago' } },
      powerIds: ['class.arcanista.poder-magico'],
      attrTotals: attrs({ intelligence: 4 }),
    })
    expect(g.pm).toBe(9)
  })

  it('Bênção do Mana (Wynna, p132): +1 PM a cada nível ímpar via godPower', () => {
    const base = { level: 7, className: 'Arcanista', attrTotals: attrs() }
    expect(collect(base).pm).toBe(0)
    // Níveis ímpares até 7: 1, 3, 5, 7 → +4.
    expect(
      collect({ ...base, godPower: 'Bênção do Mana' }).pm,
    ).toBe(4)
  })

  it('godPower sem modifier mecânico (Coragem Total) não altera PV/PM', () => {
    expect(
      collect({
        level: 5,
        className: 'Guerreiro',
        godPower: 'Coragem Total',
        attrTotals: attrs(),
      }),
    ).toEqual({ pv: 0, pm: 0 })
  })

  it('classChoice como PRÉ-REQUISITO de elective não concede o poder sozinho', () => {
    // Clérigo devoto escolhido ⇒ Autoridade Eclesiástica ainda exige o slot;
    // nenhum modifier de elective pode vazar só pela escolha de devoto.
    const g = collect({
      level: 5,
      className: 'Clérigo',
      classChoices: { 'Clérigo': { devoto: 'khalmyr' } },
      attrTotals: attrs(),
    })
    expect(g.pv).toBe(0)
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

  it('Arcanista Mago Int 4 L1 tem 10 PM (exemplo do livro, p37)', () => {
    // pmBase = 6; +atributo-chave (Int 4) = 10.
    const sheet = computeCharacterSheet({
      level: 1,
      className: 'Arcanista',
      classChoices: { Arcanista: { caminho: 'mago' } },
      baseAttributes: { strength: 0, dexterity: 0, constitution: 0, intelligence: 4, wisdom: 0, charisma: 0 },
    })
    expect(sheet.vitals.pmMax).toBe(10)
  })

  it('multiclasse Guerreiro 4 / Arcanista 4 (feiticeiro): PV seed + PM somado (p34-35)', () => {
    // PV: 20+2 (Guerreiro L1) + 3×(5+2) + 4×(2+2) = 59.
    // PM: 3×4 + 6×4 = 36; caminho feiticeiro soma Car 1 → 37.
    const sheet = computeCharacterSheet({
      level: 8,
      className: 'Guerreiro',
      classes: [
        { className: 'Guerreiro', level: 4 },
        { className: 'Arcanista', level: 4 },
      ],
      classChoices: { Arcanista: { caminho: 'feiticeiro' } },
      baseAttributes: { strength: 3, dexterity: 0, constitution: 2, intelligence: 0, wisdom: 0, charisma: 1 },
    })
    expect(sheet.vitals.pvMax).toBe(59)
    expect(sheet.vitals.pmMax).toBe(37)
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
