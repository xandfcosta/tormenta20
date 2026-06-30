import { describe, expect, it } from 'vitest'
import {
  ARMOR_ENCHANTS,
  ARMOR_TIER_SLOT_BUDGET,
  armorEnchantById,
  armorEnchantsByTier,
  armorLoadoutPriceTibar,
  isApplicableToBase,
  isValidArmorEnchantLoadout,
} from '../armor-enchants'
import type { EnchantTier } from '../weapon-enchants'

/**
 * PDF Cap 6 (Tesouro), Tabela 8-10, p338-339. Pinned:
 *  - 25 named encantos (T20 has no D&D 3.5 +N armor enhancement).
 *  - Uniform Tabela 8-7 pricing: 1 slot = T$ 18.000; 2 slots = T$ 36.000.
 *  - Only 1 two-slot encanto: Guardião (req Defensor).
 *  - Shield-only encantos: Animado, Esmagador.
 *  - All others applicableTo 'todas' (armaduras leves, pesadas, escudos).
 *  - No attunement (T20 ausente).
 */

const ALL_TIERS: readonly EnchantTier[] = [
  'menor',
  'medio',
  'maior',
  'artefato',
]

describe('ARMOR_ENCHANTS — shape & invariants', () => {
  it('catalog has exactly 25 entries', () => {
    expect(ARMOR_ENCHANTS.length).toBe(25)
  })

  it('all ids unique', () => {
    const ids = ARMOR_ENCHANTS.map((e) => e.id)
    expect(new Set(ids).size).toBe(ids.length)
  })

  it('all names unique', () => {
    const names = ARMOR_ENCHANTS.map((e) => e.name)
    expect(new Set(names).size).toBe(names.length)
  })

  it('every entry has requiresAttunement === false', () => {
    for (const e of ARMOR_ENCHANTS) {
      expect(e.requiresAttunement).toBe(false)
    }
  })

  it('every bookPage in 338-339 (Tabela 8-10 range)', () => {
    for (const e of ARMOR_ENCHANTS) {
      expect(e.bookPage).toBeGreaterThanOrEqual(338)
      expect(e.bookPage).toBeLessThanOrEqual(339)
    }
  })

  it('catalog is frozen', () => {
    expect(Object.isFrozen(ARMOR_ENCHANTS)).toBe(true)
  })

  it('every tier is a known EnchantTier value', () => {
    for (const e of ARMOR_ENCHANTS) {
      expect(ALL_TIERS).toContain(e.tier)
    }
  })

  it('slotCost is 1 or 2', () => {
    for (const e of ARMOR_ENCHANTS) {
      expect([1, 2]).toContain(e.slotCost)
    }
  })
})

describe('ARMOR_ENCHANTS — pricing per Tabela 8-7', () => {
  it('slotCost 1 → priceTibar 18000', () => {
    for (const e of ARMOR_ENCHANTS) {
      if (e.slotCost === 1) expect(e.priceTibar).toBe(18000)
    }
  })

  it('slotCost 2 → priceTibar 36000', () => {
    for (const e of ARMOR_ENCHANTS) {
      if (e.slotCost === 2) expect(e.priceTibar).toBe(36000)
    }
  })
})

describe('ARMOR_ENCHANTS — 2-slot + prerequisites', () => {
  it('Guardião é o único encanto de 2 slots', () => {
    const twoSlot = ARMOR_ENCHANTS.filter((e) => e.slotCost === 2).map(
      (e) => e.id,
    )
    expect(twoSlot).toEqual(['encanto-guardiao'])
  })

  it('Guardião requer Defensor', () => {
    expect(
      armorEnchantById('encanto-guardiao')!.prerequisiteEnchants,
    ).toEqual(['encanto-defensor'])
  })

  it('os demais encantos não têm pré-requisitos', () => {
    for (const e of ARMOR_ENCHANTS) {
      if (e.id !== 'encanto-guardiao') {
        expect(e.prerequisiteEnchants).toEqual([])
      }
    }
  })
})

describe('ARMOR_ENCHANTS — shield-only entries', () => {
  it('Animado é apenas escudo', () => {
    expect(armorEnchantById('encanto-animado')!.applicableTo).toEqual([
      'escudos',
    ])
  })

  it('Esmagador é apenas escudo', () => {
    expect(armorEnchantById('encanto-esmagador')!.applicableTo).toEqual([
      'escudos',
    ])
  })

  it('exatamente 2 encantos shield-only (Animado + Esmagador)', () => {
    const shieldOnly = ARMOR_ENCHANTS.filter(
      (e) => e.applicableTo.length === 1 && e.applicableTo[0] === 'escudos',
    ).map((e) => e.id)
    expect(shieldOnly.sort()).toEqual(['encanto-animado', 'encanto-esmagador'])
  })

  it('todos os outros são applicableTo "todas"', () => {
    const others = ARMOR_ENCHANTS.filter(
      (e) => e.id !== 'encanto-animado' && e.id !== 'encanto-esmagador',
    )
    for (const e of others) {
      expect(e.applicableTo).toEqual(['todas'])
    }
  })
})

describe('ARMOR_ENCHANTS — pinned canonical entries', () => {
  it('Defensor: +2 Defesa, T$ 18000, p338', () => {
    const e = armorEnchantById('encanto-defensor')!
    expect(e.effect).toMatch(/\+2/)
    expect(e.effect).toMatch(/Defesa/)
    expect(e.priceTibar).toBe(18000)
    expect(e.bookPage).toBe(338)
  })

  it('Guardião: +4 Defesa, 2 slots, p339', () => {
    const e = armorEnchantById('encanto-guardiao')!
    expect(e.effect).toMatch(/\+4/)
    expect(e.slotCost).toBe(2)
    expect(e.bookPage).toBe(339)
  })

  it('Abascanto: resistência a magia +5', () => {
    expect(armorEnchantById('encanto-abascanto')!.effect).toMatch(
      /resistência a magia/,
    )
  })

  it('Opaco: redução de 10 dos 4 tipos de energia', () => {
    expect(armorEnchantById('encanto-opaco')!.effect).toMatch(
      /ácido, eletricidade, fogo e frio 10/,
    )
  })

  it('Fortificado: 25% escudo / 50% armadura ignore crit', () => {
    expect(armorEnchantById('encanto-fortificado')!.effect).toMatch(/25%/)
    expect(armorEnchantById('encanto-fortificado')!.effect).toMatch(/50%/)
  })

  it('Zeloso: pega ataque por aliado adjacente', () => {
    expect(armorEnchantById('encanto-zeloso')!.effect).toMatch(/aliado/)
  })
})

describe('ARMOR_TIER_SLOT_BUDGET — mirror weapons', () => {
  it('menor=1, medio=2, maior=3, artefato=3', () => {
    expect(ARMOR_TIER_SLOT_BUDGET.menor).toBe(1)
    expect(ARMOR_TIER_SLOT_BUDGET.medio).toBe(2)
    expect(ARMOR_TIER_SLOT_BUDGET.maior).toBe(3)
    expect(ARMOR_TIER_SLOT_BUDGET.artefato).toBe(3)
  })
})

describe('armorEnchantsByTier', () => {
  it('returns only entries of that tier', () => {
    for (const tier of ALL_TIERS) {
      for (const e of armorEnchantsByTier(tier)) {
        expect(e.tier).toBe(tier)
      }
    }
  })

  it('24 tier menor + 1 tier medio (Guardião)', () => {
    expect(armorEnchantsByTier('menor').length).toBe(24)
    expect(armorEnchantsByTier('medio').length).toBe(1)
    expect(armorEnchantsByTier('medio')[0]!.id).toBe('encanto-guardiao')
  })
})

describe('isApplicableToBase', () => {
  it('encanto "todas" cabe em qualquer base', () => {
    const e = armorEnchantById('encanto-defensor')!
    expect(isApplicableToBase(e, 'armadura-leve')).toBe(true)
    expect(isApplicableToBase(e, 'armadura-pesada')).toBe(true)
    expect(isApplicableToBase(e, 'escudo')).toBe(true)
  })

  it('Animado só cabe em escudo', () => {
    const e = armorEnchantById('encanto-animado')!
    expect(isApplicableToBase(e, 'escudo')).toBe(true)
    expect(isApplicableToBase(e, 'armadura-leve')).toBe(false)
    expect(isApplicableToBase(e, 'armadura-pesada')).toBe(false)
  })

  it('Esmagador só cabe em escudo', () => {
    const e = armorEnchantById('encanto-esmagador')!
    expect(isApplicableToBase(e, 'escudo')).toBe(true)
    expect(isApplicableToBase(e, 'armadura-leve')).toBe(false)
  })
})

describe('isValidArmorEnchantLoadout', () => {
  it('1 encanto menor em armadura-leve menor é válido', () => {
    expect(
      isValidArmorEnchantLoadout('armadura-leve', 'menor', [
        'encanto-defensor',
      ]),
    ).toBe(true)
  })

  it('2 menor encantos não cabem em armadura menor (budget 1)', () => {
    expect(
      isValidArmorEnchantLoadout('armadura-leve', 'menor', [
        'encanto-defensor',
        'encanto-protetor',
      ]),
    ).toBe(false)
  })

  it('Guardião sozinho falha (sem pré-requisito Defensor)', () => {
    expect(
      isValidArmorEnchantLoadout('armadura-leve', 'maior', [
        'encanto-guardiao',
      ]),
    ).toBe(false)
  })

  it('Defensor + Guardião cabe em armadura maior (budget 3, usa 3 slots)', () => {
    expect(
      isValidArmorEnchantLoadout('armadura-leve', 'maior', [
        'encanto-defensor',
        'encanto-guardiao',
      ]),
    ).toBe(true)
  })

  it('Defensor + Guardião excede budget de armadura médio (2)', () => {
    expect(
      isValidArmorEnchantLoadout('armadura-leve', 'medio', [
        'encanto-defensor',
        'encanto-guardiao',
      ]),
    ).toBe(false)
  })

  it('Animado em armadura falha (shield-only)', () => {
    expect(
      isValidArmorEnchantLoadout('armadura-leve', 'menor', [
        'encanto-animado',
      ]),
    ).toBe(false)
  })

  it('Animado em escudo é válido', () => {
    expect(
      isValidArmorEnchantLoadout('escudo', 'menor', ['encanto-animado']),
    ).toBe(true)
  })

  it('Esmagador + Defensor em escudo médio cabe e ambos aplicam', () => {
    expect(
      isValidArmorEnchantLoadout('escudo', 'medio', [
        'encanto-esmagador',
        'encanto-defensor',
      ]),
    ).toBe(true)
  })

  it('duplicatas falham', () => {
    expect(
      isValidArmorEnchantLoadout('armadura-leve', 'maior', [
        'encanto-defensor',
        'encanto-defensor',
      ]),
    ).toBe(false)
  })

  it('id desconhecido falha', () => {
    expect(
      isValidArmorEnchantLoadout('escudo', 'menor', ['encanto-vorpal']),
    ).toBe(false)
  })

  it('loadout vazio é válido', () => {
    expect(isValidArmorEnchantLoadout('armadura-leve', 'menor', [])).toBe(true)
  })
})

describe('armorLoadoutPriceTibar', () => {
  it('Defensor sozinho = 18000', () => {
    expect(armorLoadoutPriceTibar(['encanto-defensor'])).toBe(18000)
  })

  it('Defensor + Guardião = 18000 + 36000 = 54000', () => {
    expect(
      armorLoadoutPriceTibar(['encanto-defensor', 'encanto-guardiao']),
    ).toBe(54000)
  })

  it('id desconhecido retorna -1', () => {
    expect(armorLoadoutPriceTibar(['foo'])).toBe(-1)
  })

  it('loadout vazio retorna 0', () => {
    expect(armorLoadoutPriceTibar([])).toBe(0)
  })
})

describe('armorEnchantById', () => {
  it('lookup hit', () => {
    expect(armorEnchantById('encanto-defensor')?.name).toBe('Defensor')
  })

  it('lookup miss', () => {
    expect(armorEnchantById('encanto-vorpal')).toBeUndefined()
  })
})
