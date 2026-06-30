import { describe, expect, it } from 'vitest'
import {
  WEAPON_ENCHANTS,
  WEAPON_TIER_SLOT_BUDGET,
  isValidEnchantLoadout,
  loadoutPriceTibar,
  weaponEnchantById,
  weaponEnchantsByTier,
  type EnchantTier,
} from '../weapon-enchants'

/**
 * PDF Cap 6 (Tesouro), Tabela 8-7 + 8-8, p334-336. Pinned:
 *
 *  - 28 named encantos (T20 has NO +1/+2/+3 enhancement progression).
 *  - Uniform pricing: 1 slot = T$ 18.000; 2 slots = T$ 36.000.
 *  - 3 encantos consume 2 slots AND have prerequisites:
 *      Energética (req Formidável), Lancinante (req Dilacerante),
 *      Magnífica (req Formidável).
 *  - No attunement: every entry `requiresAttunement: false`.
 *  - Slot budget: menor=1, medio=2, maior=3.
 */

const ALL_TIERS: readonly EnchantTier[] = ['menor', 'medio', 'maior', 'artefato']

describe('WEAPON_ENCHANTS — shape & invariants', () => {
  it('catalog has exactly 28 entries', () => {
    expect(WEAPON_ENCHANTS.length).toBe(28)
  })

  it('all ids unique', () => {
    const ids = WEAPON_ENCHANTS.map((e) => e.id)
    expect(new Set(ids).size).toBe(ids.length)
  })

  it('all names unique', () => {
    const names = WEAPON_ENCHANTS.map((e) => e.name)
    expect(new Set(names).size).toBe(names.length)
  })

  it('every entry has requiresAttunement === false (T20 has no attunement)', () => {
    for (const e of WEAPON_ENCHANTS) {
      expect(e.requiresAttunement).toBe(false)
    }
  })

  it('every bookPage falls in 335-336 (PDF tesouro range)', () => {
    for (const e of WEAPON_ENCHANTS) {
      expect(e.bookPage).toBeGreaterThanOrEqual(335)
      expect(e.bookPage).toBeLessThanOrEqual(336)
    }
  })

  it('catalog is frozen', () => {
    expect(Object.isFrozen(WEAPON_ENCHANTS)).toBe(true)
  })

  it('tier is one of the known EnchantTier values', () => {
    for (const e of WEAPON_ENCHANTS) {
      expect(ALL_TIERS).toContain(e.tier)
    }
  })

  it('slotCost is 1 or 2 only', () => {
    for (const e of WEAPON_ENCHANTS) {
      expect([1, 2]).toContain(e.slotCost)
    }
  })
})

describe('WEAPON_ENCHANTS — uniform pricing per Tabela 8-7', () => {
  it('slotCost 1 → priceTibar 18000', () => {
    for (const e of WEAPON_ENCHANTS) {
      if (e.slotCost === 1) expect(e.priceTibar).toBe(18000)
    }
  })

  it('slotCost 2 → priceTibar 36000', () => {
    for (const e of WEAPON_ENCHANTS) {
      if (e.slotCost === 2) expect(e.priceTibar).toBe(36000)
    }
  })
})

describe('WEAPON_ENCHANTS — 2-slot encantos and prerequisites', () => {
  it('exactly 3 encantos consume 2 slots', () => {
    const twoSlot = WEAPON_ENCHANTS.filter((e) => e.slotCost === 2).map(
      (e) => e.id,
    )
    expect(twoSlot.sort()).toEqual([
      'encanto-energetica',
      'encanto-lancinante',
      'encanto-magnifica',
    ])
  })

  it('Energética requires Formidável', () => {
    expect(weaponEnchantById('encanto-energetica')!.prerequisiteEnchants).toEqual([
      'encanto-formidavel',
    ])
  })

  it('Lancinante requires Dilacerante', () => {
    expect(
      weaponEnchantById('encanto-lancinante')!.prerequisiteEnchants,
    ).toEqual(['encanto-dilacerante'])
  })

  it('Magnífica requires Formidável', () => {
    expect(weaponEnchantById('encanto-magnifica')!.prerequisiteEnchants).toEqual([
      'encanto-formidavel',
    ])
  })

  it('all other encantos have no prerequisites', () => {
    for (const e of WEAPON_ENCHANTS) {
      if (
        e.id !== 'encanto-energetica' &&
        e.id !== 'encanto-lancinante' &&
        e.id !== 'encanto-magnifica'
      ) {
        expect(e.prerequisiteEnchants).toEqual([])
      }
    }
  })
})

describe('WEAPON_ENCHANTS — applicability', () => {
  it('Arremesso restrito a corpo a corpo (que se torna arremessável)', () => {
    expect(weaponEnchantById('encanto-arremesso')!.applicableTo).toEqual([
      'corpo a corpo',
    ])
  })

  it('Dançarina restrito a corpo a corpo', () => {
    expect(weaponEnchantById('encanto-dancarina')!.applicableTo).toEqual([
      'corpo a corpo',
    ])
  })

  it('Flamejante / Congelante / Elétrica / Corrosiva: aplicáveis a todas', () => {
    for (const id of [
      'encanto-flamejante',
      'encanto-congelante',
      'encanto-eletrica',
      'encanto-corrosiva',
    ]) {
      expect(weaponEnchantById(id)!.applicableTo).toEqual(['todas'])
    }
  })
})

describe('WEAPON_ENCHANTS — pinned canonical entries', () => {
  it('Flamejante: T$ 18000, +1d6 fogo, bola de fogo 6d6, p335', () => {
    const e = weaponEnchantById('encanto-flamejante')!
    expect(e.priceTibar).toBe(18000)
    expect(e.slotCost).toBe(1)
    expect(e.tier).toBe('menor')
    expect(e.bookPage).toBe(335)
    expect(e.effect).toMatch(/\+1d6 de dano de fogo/)
    expect(e.effect).toMatch(/6d6/)
  })

  it('Formidável: +2 ataque e dano, p336', () => {
    const e = weaponEnchantById('encanto-formidavel')!
    expect(e.effect).toMatch(/\+2 em testes de ataque/)
    expect(e.bookPage).toBe(336)
  })

  it('Magnífica: +4 ataque/dano, 2 slots, p336', () => {
    const e = weaponEnchantById('encanto-magnifica')!
    expect(e.effect).toMatch(/\+4/)
    expect(e.slotCost).toBe(2)
    expect(e.priceTibar).toBe(36000)
  })

  it('Ameaçadora: duplica margem de ameaça', () => {
    expect(weaponEnchantById('encanto-ameacadora')!.effect).toMatch(
      /margem de ameaça/,
    )
  })

  it('Drenante: crit + PV temporários', () => {
    expect(weaponEnchantById('encanto-drenante')!.effect).toMatch(
      /pontos de vida temporários/,
    )
  })
})

describe('WEAPON_TIER_SLOT_BUDGET', () => {
  it('menor=1, medio=2, maior=3, artefato=3', () => {
    expect(WEAPON_TIER_SLOT_BUDGET.menor).toBe(1)
    expect(WEAPON_TIER_SLOT_BUDGET.medio).toBe(2)
    expect(WEAPON_TIER_SLOT_BUDGET.maior).toBe(3)
    expect(WEAPON_TIER_SLOT_BUDGET.artefato).toBe(3)
  })
})

describe('weaponEnchantsByTier', () => {
  it('returns only entries of that tier', () => {
    for (const tier of ALL_TIERS) {
      const slice = weaponEnchantsByTier(tier)
      for (const e of slice) {
        expect(e.tier).toBe(tier)
      }
    }
  })

  it('25 encantos tier menor, 3 tier medio, 0 maior/artefato', () => {
    expect(weaponEnchantsByTier('menor').length).toBe(25)
    expect(weaponEnchantsByTier('medio').length).toBe(3)
    expect(weaponEnchantsByTier('maior').length).toBe(0)
  })
})

describe('isValidEnchantLoadout — slot budget + prereqs', () => {
  it('1 menor encanto fits a menor weapon (1 slot)', () => {
    expect(isValidEnchantLoadout('menor', ['encanto-flamejante'])).toBe(true)
  })

  it('2 menor encantos não cabem em arma menor (excede budget 1)', () => {
    expect(
      isValidEnchantLoadout('menor', [
        'encanto-flamejante',
        'encanto-congelante',
      ]),
    ).toBe(false)
  })

  it('2 menor encantos cabem em arma médio (budget 2)', () => {
    expect(
      isValidEnchantLoadout('medio', [
        'encanto-flamejante',
        'encanto-congelante',
      ]),
    ).toBe(true)
  })

  it('Magnífica em arma médio: ocupa todo o budget E exige Formidável', () => {
    // Magnífica sozinha = 2 slots, falta o prereq Formidável.
    expect(isValidEnchantLoadout('medio', ['encanto-magnifica'])).toBe(false)
    // Magnífica + Formidável = 3 slots > médio budget 2.
    expect(
      isValidEnchantLoadout('medio', [
        'encanto-formidavel',
        'encanto-magnifica',
      ]),
    ).toBe(false)
    // Magnífica + Formidável = 3 slots cabem em arma maior (budget 3).
    expect(
      isValidEnchantLoadout('maior', [
        'encanto-formidavel',
        'encanto-magnifica',
      ]),
    ).toBe(true)
  })

  it('Lancinante exige Dilacerante na mesma loadout', () => {
    expect(isValidEnchantLoadout('maior', ['encanto-lancinante'])).toBe(false)
    expect(
      isValidEnchantLoadout('maior', [
        'encanto-dilacerante',
        'encanto-lancinante',
      ]),
    ).toBe(true)
  })

  it('Energética exige Formidável', () => {
    expect(isValidEnchantLoadout('maior', ['encanto-energetica'])).toBe(false)
    expect(
      isValidEnchantLoadout('maior', [
        'encanto-formidavel',
        'encanto-energetica',
      ]),
    ).toBe(true)
  })

  it('duplicatas inválidas', () => {
    expect(
      isValidEnchantLoadout('maior', [
        'encanto-flamejante',
        'encanto-flamejante',
      ]),
    ).toBe(false)
  })

  it('id desconhecido falha', () => {
    expect(isValidEnchantLoadout('menor', ['encanto-vorpal'])).toBe(false)
  })

  it('loadout vazio é válido', () => {
    expect(isValidEnchantLoadout('menor', [])).toBe(true)
    expect(isValidEnchantLoadout('maior', [])).toBe(true)
  })
})

describe('loadoutPriceTibar', () => {
  it('Flamejante = 18000', () => {
    expect(loadoutPriceTibar(['encanto-flamejante'])).toBe(18000)
  })

  it('Flamejante + Congelante = 36000 (2 × 18000)', () => {
    expect(
      loadoutPriceTibar(['encanto-flamejante', 'encanto-congelante']),
    ).toBe(36000)
  })

  it('Formidável + Magnífica = 18000 + 36000 = 54000', () => {
    expect(
      loadoutPriceTibar(['encanto-formidavel', 'encanto-magnifica']),
    ).toBe(54000)
  })

  it('id desconhecido retorna -1', () => {
    expect(loadoutPriceTibar(['encanto-flamejante', 'foo'])).toBe(-1)
  })

  it('loadout vazio retorna 0', () => {
    expect(loadoutPriceTibar([])).toBe(0)
  })
})

describe('weaponEnchantById', () => {
  it('lookup hit', () => {
    expect(weaponEnchantById('encanto-flamejante')?.name).toBe('Flamejante')
  })

  it('lookup miss', () => {
    expect(weaponEnchantById('encanto-vorpal')).toBeUndefined()
  })
})
