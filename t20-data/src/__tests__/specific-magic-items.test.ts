import { describe, expect, it } from 'vitest'
import {
  SPECIFIC_MAGIC_ITEMS,
  classRestrictedItems,
  specificItemById,
  specificItemsByCategory,
  specificItemsByTier,
} from '../specific-magic-items'

/**
 * PDF Cap 6 (Tesouro), Tabela 8-9, p337. Pinned:
 *  - 18 named weapons (Tabela 8-9 has exactly 18 entries).
 *  - ALL tier 'maior' per p334: "Todas as armas e armaduras
 *    específicas deste livro são itens maiores."
 *  - Prices range 30.000-200.000 T$.
 *  - No T20 attunement: every requiresAttunement === false.
 *  - Hard class restriction: Vingadora Sagrada (Paladino).
 *  - Conditional class scaling: Espada Baronial / Lâmina da Luz /
 *    Cajados da Destruição+Vida+Poder / Martelo de Doherimm / Lança
 *    Animalesca.
 */

describe('SPECIFIC_MAGIC_ITEMS — shape & invariants', () => {
  it('catalog has exactly 18 entries', () => {
    expect(SPECIFIC_MAGIC_ITEMS.length).toBe(18)
  })

  it('all ids unique', () => {
    const ids = SPECIFIC_MAGIC_ITEMS.map((i) => i.id)
    expect(new Set(ids).size).toBe(ids.length)
  })

  it('all names unique', () => {
    const names = SPECIFIC_MAGIC_ITEMS.map((i) => i.name)
    expect(new Set(names).size).toBe(names.length)
  })

  it('every entry has requiresAttunement === false (T20 sem attunement)', () => {
    for (const i of SPECIFIC_MAGIC_ITEMS) {
      expect(i.requiresAttunement).toBe(false)
    }
  })

  it('every entry is tier "maior" (p334)', () => {
    for (const i of SPECIFIC_MAGIC_ITEMS) {
      expect(i.tier).toBe('maior')
    }
  })

  it('every entry has category "arma" (Tabela 8-9 = armas)', () => {
    for (const i of SPECIFIC_MAGIC_ITEMS) {
      expect(i.category).toBe('arma')
    }
  })

  it('every entry has priceTibar in [30000, 200000]', () => {
    for (const i of SPECIFIC_MAGIC_ITEMS) {
      expect(i.priceTibar).toBeGreaterThanOrEqual(30000)
      expect(i.priceTibar).toBeLessThanOrEqual(200000)
    }
  })

  it('every entry has bookPage in 336-338', () => {
    for (const i of SPECIFIC_MAGIC_ITEMS) {
      expect(i.bookPage).toBeGreaterThanOrEqual(336)
      expect(i.bookPage).toBeLessThanOrEqual(338)
    }
  })

  it('every entry has a baseItem string', () => {
    for (const i of SPECIFIC_MAGIC_ITEMS) {
      expect(typeof i.baseItem).toBe('string')
      expect(i.baseItem!.length).toBeGreaterThan(0)
    }
  })

  it('catalog is frozen', () => {
    expect(Object.isFrozen(SPECIFIC_MAGIC_ITEMS)).toBe(true)
  })
})

describe('SPECIFIC_MAGIC_ITEMS — pinned canonical entries', () => {
  it('Vingadora Sagrada: T$ 200000, Paladino, Espada Longa, p338', () => {
    const i = specificItemById('vingadora-sagrada')!
    expect(i.priceTibar).toBe(200000)
    expect(i.requiresClass).toBe('Paladino')
    expect(i.baseItem).toBe('Espada Longa')
    expect(i.bookPage).toBe(338)
    expect(i.effect).toMatch(/paladino/)
  })

  it('Avalanche: T$ 140000, Martelo de Guerra, tempestade de gelo', () => {
    const i = specificItemById('avalanche')!
    expect(i.priceTibar).toBe(140000)
    expect(i.baseItem).toBe('Martelo de Guerra')
    expect(i.effect).toMatch(/tempestade de gelo/)
  })

  it('Espada Baronial: T$ 30000, bônus escala c/ Khalmyr/Nobreza/conduta', () => {
    const i = specificItemById('espada-baronial')!
    expect(i.priceTibar).toBe(30000)
    expect(i.effect).toMatch(/Khalmyr/)
    expect(i.effect).toMatch(/Nobreza/)
  })

  it('Lâmina da Luz: T$ 45000, concedida a Cavaleiros da Luz', () => {
    const i = specificItemById('lamina-da-luz')!
    expect(i.priceTibar).toBe(45000)
    expect(i.requiresClass).toMatch(/Cavaleiro da Luz/)
    expect(i.baseItem).toBe('Espada Bastarda')
  })

  it('Martelo de Doherimm: bônus extra para Anão', () => {
    const i = specificItemById('martelo-de-doherimm')!
    expect(i.requiresClass).toMatch(/Anão/)
    expect(i.effect).toMatch(/anão/)
  })

  it('Arco do Poder: T$ 90000, 4 tipos de flecha energética', () => {
    const i = specificItemById('arco-do-poder')!
    expect(i.priceTibar).toBe(90000)
    expect(i.effect).toMatch(/Flecha Normal/)
    expect(i.effect).toMatch(/Flecha Piedosa/)
    expect(i.effect).toMatch(/Flecha Explosiva/)
    expect(i.effect).toMatch(/Flecha-Rede/)
  })

  it('Cajado do Poder: T$ 180000, -1 PM em magias arcanas', () => {
    const i = specificItemById('cajado-do-poder')!
    expect(i.priceTibar).toBe(180000)
    expect(i.requiresClass).toMatch(/arcano/)
    expect(i.effect).toMatch(/-1 PM/)
  })

  it('Azagaia dos Relâmpagos: T$ 30000 (item mais barato), 8d6 eletric.', () => {
    const i = specificItemById('azagaia-dos-relampagos')!
    expect(i.priceTibar).toBe(30000)
    expect(i.effect).toMatch(/8d6/)
  })

  it('Punhal Sszzaazita: T$ 100000, transformação em objeto inofensivo', () => {
    const i = specificItemById('punhal-sszzaazita')!
    expect(i.effect).toMatch(/transformar/)
    expect(i.effect).toMatch(/colher|pena/)
  })
})

describe('SPECIFIC_MAGIC_ITEMS — pricing distribution', () => {
  it('items at T$ 30000 are the cheapest (entry floor per p334)', () => {
    const cheapest = SPECIFIC_MAGIC_ITEMS.reduce(
      (m, i) => Math.min(m, i.priceTibar),
      Infinity,
    )
    expect(cheapest).toBe(30000)
  })

  it('Vingadora Sagrada is the most expensive at T$ 200000', () => {
    const max = SPECIFIC_MAGIC_ITEMS.reduce(
      (m, i) => Math.max(m, i.priceTibar),
      0,
    )
    expect(max).toBe(200000)
    const top = SPECIFIC_MAGIC_ITEMS.filter((i) => i.priceTibar === max)
    expect(top.length).toBe(1)
    expect(top[0]!.id).toBe('vingadora-sagrada')
  })

  it('prices are tier-maior valid (>= 30000)', () => {
    for (const i of SPECIFIC_MAGIC_ITEMS) {
      expect(i.priceTibar).toBeGreaterThanOrEqual(30000)
    }
  })
})

describe('SPECIFIC_MAGIC_ITEMS — class restrictions', () => {
  it('Vingadora Sagrada is the only hard-restricted item ("Paladino" exact)', () => {
    const paladinOnly = SPECIFIC_MAGIC_ITEMS.filter(
      (i) => i.requiresClass === 'Paladino',
    )
    expect(paladinOnly.length).toBe(1)
    expect(paladinOnly[0]!.id).toBe('vingadora-sagrada')
  })

  it('classRestrictedItems retorna 7 itens com requiresClass não-null', () => {
    const restricted = classRestrictedItems()
    expect(restricted.length).toBe(7)
  })

  it('itens sem restrição (11 entradas) têm requiresClass null', () => {
    const unrestricted = SPECIFIC_MAGIC_ITEMS.filter(
      (i) => i.requiresClass === null,
    )
    expect(unrestricted.length).toBe(11)
  })

  it('restritos + livres = 18 total', () => {
    expect(classRestrictedItems().length).toBe(7)
    expect(
      SPECIFIC_MAGIC_ITEMS.filter((i) => i.requiresClass === null).length,
    ).toBe(11)
    expect(SPECIFIC_MAGIC_ITEMS.length).toBe(18)
  })
})

describe('SPECIFIC_MAGIC_ITEMS — lookup helpers', () => {
  it('specificItemById hit', () => {
    expect(specificItemById('avalanche')?.name).toBe('Avalanche')
  })

  it('specificItemById miss', () => {
    expect(specificItemById('martelo-de-thor')).toBeUndefined()
  })

  it('specificItemsByCategory("arma") returns all 18', () => {
    expect(specificItemsByCategory('arma').length).toBe(18)
  })

  it('specificItemsByCategory("armadura") returns 0 (Tabela 8-11 nao incluida)', () => {
    expect(specificItemsByCategory('armadura').length).toBe(0)
  })

  it('specificItemsByTier("maior") returns all 18', () => {
    expect(specificItemsByTier('maior').length).toBe(18)
  })

  it('specificItemsByTier("menor") returns 0', () => {
    expect(specificItemsByTier('menor').length).toBe(0)
  })
})

describe('SPECIFIC_MAGIC_ITEMS — baseItem references', () => {
  it('cada baseItem é a base mundana correspondente', () => {
    // Pinned spot-checks por verbatim correspondence c/ Tabela 8-9.
    const expectations: Record<string, string> = {
      'azagaia-dos-relampagos': 'Azagaia',
      'espada-baronial': 'Espada Longa',
      'lamina-da-luz': 'Espada Bastarda',
      'florete-fugaz': 'Florete',
      'cajado-da-destruicao': 'Bordão',
      'cajado-da-vida': 'Bordão',
      'cajado-do-poder': 'Bordão',
      'machado-silvestre': 'Machado de Batalha',
      'martelo-de-doherimm': 'Martelo de Guerra',
      'arco-do-poder': 'Arco Longo',
      'lingua-do-deserto': 'Cimitarra',
      'besta-explosiva': 'Besta Pesada',
      'punhal-sszzaazita': 'Adaga',
      'espada-sortuda': 'Espada Curta',
      avalanche: 'Martelo de Guerra',
      'vingadora-sagrada': 'Espada Longa',
    }
    for (const [id, base] of Object.entries(expectations)) {
      expect(specificItemById(id)?.baseItem).toBe(base)
    }
  })
})
