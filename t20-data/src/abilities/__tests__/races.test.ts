import { describe, expect, it } from 'vitest'
import type { AttributeKey } from '../../attributes'
import { RACES_CATALOG } from '../races'

/**
 * PDF Cap 1, Raças (p18-31). Tabela 1-2 (p18) lists the canonical attribute
 * modifiers per race. These tests pin the catalog to the table so any
 * future edit to attributeBonuses is caught immediately.
 *
 * Races where the player picks "+1 em três atributos diferentes" only encode
 * the *forced* modifiers (Lefou's Car-1, Osteon's Con-1) — the picker
 * portion is applied at character creation in the UI, not in the catalog.
 */
type AttrBonuses = Partial<Record<AttributeKey, number>>

const BOOK_TABELA_1_2: Record<string, AttrBonuses> = {
  Humano: {},
  'Anão': { constitution: 2, wisdom: 1, dexterity: -1 },
  Dahllan: { wisdom: 2, dexterity: 1, intelligence: -1 },
  Elfo: { intelligence: 2, dexterity: 1, constitution: -1 },
  Goblin: { dexterity: 2, intelligence: 1, charisma: -1 },
  Lefou: { charisma: -1 },
  Minotauro: { strength: 2, constitution: 1, wisdom: -1 },
  Qareen: { charisma: 2, intelligence: 1, wisdom: -1 },
  Golem: { strength: 2, constitution: 1, charisma: -1 },
  Hynne: { dexterity: 2, charisma: 1, strength: -1 },
  // Code preserves the legacy "Kiiren" spelling for backend compat; the
  // PDF spells it "Kliren" (p28). The test uses the catalog id.
  Kiiren: { intelligence: 2, charisma: 1, strength: -1 },
  Medusa: { dexterity: 2, charisma: 1 },
  Osteon: { constitution: -1 },
  'Sereia/Tritão': {},
  'Sílfide': { charisma: 2, dexterity: 1, strength: -2 },
  // Suraggel picks aggelus (Sab+2 Car+1) OR sulfure (Des+2 Int+1) at
  // creation; the *base* catalog can't fold either path in without the
  // variant pick, so the table is empty. Verified separately below.
  Suraggel: {},
  Trog: { constitution: 2, strength: 1, intelligence: -1 },
}

describe('RACES_CATALOG vs PDF Tabela 1-2', () => {
  it('contains all 17 races from Cap 1', () => {
    expect(RACES_CATALOG.length).toBe(17)
  })

  for (const [raceId, bonuses] of Object.entries(BOOK_TABELA_1_2)) {
    it(`${raceId}: attributeBonuses match the book`, () => {
      const race = RACES_CATALOG.find((r) => r.id === raceId)
      expect(race, `race ${raceId} missing from catalog`).toBeDefined()
      expect(race!.attributeBonuses).toEqual(bonuses)
    })
  }
})

describe('Suraggel — variant-driven attribute bonuses', () => {
  // PDF p30-31: aggelus = Sab+2 Car+1; sulfure = Des+2 Int+1. Both come
  // from picking the herança variant. The catalog must therefore expose
  // both variant ids so the UI picker can drive the bonuses.
  it('exposes both aggelus and sulfure variant ids on Herança Divina', () => {
    const suraggel = RACES_CATALOG.find((r) => r.id === 'Suraggel')
    expect(suraggel).toBeDefined()
    const heranca = suraggel!.abilities.find((a) =>
      a.id.startsWith('suraggel-heranca'),
    )
    expect(heranca?.variants?.map((v) => v.id).sort()).toEqual(
      ['suraggel-aggelus', 'suraggel-sulfure'].sort(),
    )
  })
})

describe('Open-choice races — "+1 em três atributos diferentes"', () => {
  // Humano, Lefou, Osteon, Sereia/Tritão all give "+1 em três atributos
  // diferentes" (with race-specific exceptions). The catalog can only
  // encode the *forced* modifiers; the picker lives at character creation
  // time, so attributeBonuses just shouldn't include any positive +2 entry.
  const openRaces = ['Humano', 'Lefou', 'Osteon', 'Sereia/Tritão']
  for (const raceId of openRaces) {
    it(`${raceId}: forced bonuses don't include any +2`, () => {
      const race = RACES_CATALOG.find((r) => r.id === raceId)
      const positives = Object.values(race!.attributeBonuses).filter(
        (v) => v >= 2,
      )
      expect(positives).toEqual([])
    })
  }
})
