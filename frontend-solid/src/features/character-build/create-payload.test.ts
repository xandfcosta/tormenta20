import { describe, expect, it } from 'vitest'
import { createCharacterPayload } from './create-payload'
import { wizardDefaults } from './wizard-steps'

const base = {
  ...wizardDefaults,
  name: 'Thal',
  races: ['Humano'],
  origin: 'Acólito',
  classes: [{ className: 'Guerreiro', level: 1 }],
  size: 'Médio',
}

describe('createCharacterPayload', () => {
  it('recomputes the pools instead of trusting the draft', () => {
    // The draft could carry a stale máximo (level changed on the last screen);
    // the server heals vitals anyway, but the payload must not disagree.
    const payload = createCharacterPayload({ ...base, hpMax: 1, hpCurrent: 1 }, {})

    expect(payload.hpMax).toBeGreaterThan(1)
    expect(payload.hpCurrent).toBeLessThanOrEqual(payload.hpMax)
  })

  it('keeps a deliberately wounded current below the maximum', () => {
    const payload = createCharacterPayload({ ...base, hpCurrent: 5 }, {})

    expect(payload.hpCurrent).toBe(5)
  })

  it('strips the wizard-only equipment fields from the body', () => {
    const payload = createCharacterPayload(
      { ...base, startingWeaponSimple: 'adaga', startingMoneyRolled: true },
      {},
    ) as Record<string, unknown>

    expect(payload.startingWeaponSimple).toBeUndefined()
    expect(payload.startingMoneyRolled).toBeUndefined()
    expect(payload.startingPurchases).toBeUndefined()
    expect(payload.originItemPicks).toBeUndefined()
  })

  it('turns the kit picks into inventory rows', () => {
    const payload = createCharacterPayload({ ...base, startingWeaponSimple: 'adaga' }, {})

    expect(payload.items.some((item) => item.catalogId === 'adaga')).toBe(true)
  })

  it('saves the money LEFT after the shop, not what was rolled', () => {
    const payload = createCharacterPayload(
      { ...base, tibar: 100, startingPurchases: { adaga: 1 } },
      {},
    )

    expect(payload.tibar).toBeLessThan(100)
    expect(payload.items.some((item) => item.catalogId === 'adaga')).toBe(true)
  })

  it('never saves a negative purse', () => {
    const payload = createCharacterPayload(
      { ...base, tibar: 0, startingPurchases: { adaga: 40 } },
      {},
    )

    expect(payload.tibar).toBe(0)
  })

  it('drops elective powers beyond the slots the levels earn', () => {
    // Lowering the level after picking powers must not save the excess.
    const payload = createCharacterPayload(
      { ...base, classPowers: ['ataque-poderoso', 'reflexos-rapidos'] },
      {},
    )

    // Nv 1 earns no elective slot at all.
    expect(payload.classPowers).toEqual([])
  })

  it('omits the god and its granted power when no god was chosen', () => {
    const payload = createCharacterPayload({ ...base, god: '', godPower: 'Espada Justiceira' }, {})

    expect(payload.god).toBeUndefined()
    expect(payload.godPower).toBeUndefined()
  })

  it('keeps the granted power once there IS a god', () => {
    const payload = createCharacterPayload(
      { ...base, god: 'Khalmyr', godPower: 'Espada Justiceira' },
      {},
    )

    expect(payload.god).toBe('Khalmyr')
    expect(payload.godPower).toBe('Espada Justiceira')
  })

  it('sends the primary race choices as raceAttributeChoices', () => {
    const payload = createCharacterPayload(base, {
      Humano: { floatingPicks: ['strength', 'dexterity'] },
    })

    expect(payload.raceAttributeChoices.floatingPicks).toEqual(['strength', 'dexterity'])
  })

  it('sends only the secondary races the player opted into', () => {
    const payload = createCharacterPayload(
      { ...base, races: ['Humano', 'Anão', 'Elfo'] },
      { 'Anão': { applied: true }, Elfo: { applied: false } },
    )

    expect(payload.secondaryRaceChoices.map((entry) => entry.race)).toEqual(['Anão'])
  })
})
