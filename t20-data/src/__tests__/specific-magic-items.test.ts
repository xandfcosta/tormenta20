import { describe, expect, it } from 'vitest'
import {
  SPECIFIC_MAGIC_ITEMS,
  classRestrictedItems,
  specificItemById,
  specificItemsByCategory,
  specificItemsByTier,
} from '../specific-magic-items'

/**
 * PDF Cap 6 (Tesouro). Pinned:
 *  - Tabela 8-9 (armas, p337): 18 entries.
 *  - Tabela 8-11 (armaduras + escudos, p340): 13 entries (8 armaduras
 *    + 5 escudos).
 *  - Total 31 itens específicos.
 *  - ALL tier 'maior' per p334: "Todas as armas e armaduras
 *    específicas deste livro são itens maiores."
 *  - Prices range 30.000-200.000 T$.
 *  - No T20 attunement: every requiresAttunement === false.
 *  - Hard class restriction: Vingadora Sagrada (Paladino).
 *  - Conditional class scaling: Espada Baronial / Lâmina da Luz /
 *    Cajados da Destruição+Vida+Poder / Martelo de Doherimm / Lança
 *    Animalesca.
 *  - Tabela 8-11 entries: no hard locks; conditional bonuses
 *    (devoto / código de conduta / poder Comandar) descritas no effect.
 */

describe('SPECIFIC_MAGIC_ITEMS — shape & invariants', () => {
  it('catalog has exactly 31 entries (18 armas + 13 armaduras/escudos)', () => {
    expect(SPECIFIC_MAGIC_ITEMS.length).toBe(31)
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

  it('every entry has category arma|armadura|escudo', () => {
    for (const i of SPECIFIC_MAGIC_ITEMS) {
      expect(['arma', 'armadura', 'escudo']).toContain(i.category)
    }
  })

  it('every entry has priceTibar in [30000, 200000]', () => {
    for (const i of SPECIFIC_MAGIC_ITEMS) {
      expect(i.priceTibar).toBeGreaterThanOrEqual(30000)
      expect(i.priceTibar).toBeLessThanOrEqual(200000)
    }
  })

  it('every entry has bookPage in 336-340 (Tabelas 8-9 + 8-11)', () => {
    for (const i of SPECIFIC_MAGIC_ITEMS) {
      expect(i.bookPage).toBeGreaterThanOrEqual(336)
      expect(i.bookPage).toBeLessThanOrEqual(340)
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

describe('SPECIFIC_MAGIC_ITEMS — Tabela 8-11 pinned (armaduras + escudos)', () => {
  it('Cota Élfica: T$ 30000, base Cota de Malha, aplica Des como leve', () => {
    const i = specificItemById('cota-elfica')!
    expect(i.priceTibar).toBe(30000)
    expect(i.category).toBe('armadura')
    expect(i.baseItem).toBe('Cota de Malha')
    expect(i.bookPage).toBe(340)
    expect(i.effect).toMatch(/Destreza/)
  })

  it('Armadura da Luz: T$ 150000, RD = Carisma (devoto/conduta)', () => {
    const i = specificItemById('armadura-da-luz')!
    expect(i.priceTibar).toBe(150000)
    expect(i.category).toBe('armadura')
    expect(i.effect).toMatch(/Khalmyr/)
    expect(i.effect).toMatch(/redução de dano igual ao seu Carisma/)
  })

  it('Escudo de Azgher: T$ 140000, cone 6d6 fogo (mortos-vivos 6d8)', () => {
    const i = specificItemById('escudo-de-azgher')!
    expect(i.priceTibar).toBe(140000)
    expect(i.category).toBe('escudo')
    expect(i.effect).toMatch(/6d6/)
    expect(i.effect).toMatch(/6d8/)
  })

  it('Baluarte Anão: armadura completa, RD 10 ao não se deslocar', () => {
    const i = specificItemById('baluarte-anao')!
    expect(i.baseItem).toBe('Armadura Completa')
    expect(i.effect).toMatch(/RD que ela fornece aumenta para 10/)
  })

  it('Manto da Noite: Couro Batido, sem penalidade Furtividade ao mover', () => {
    const i = specificItemById('manto-da-noite')!
    expect(i.baseItem).toBe('Couro Batido')
    expect(i.effect).toMatch(/Furtividade/)
  })

  it('Escudo do Eclipse: redução trevas 10 + lança Escuridão', () => {
    const i = specificItemById('escudo-do-eclipse')!
    expect(i.category).toBe('escudo')
    expect(i.effect).toMatch(/redução de trevas 10/)
    expect(i.effect).toMatch(/Escuridão/)
  })

  it('Carapaça Demoníaca: base Armadura Completa, +1d8 trevas (devoto negativo)', () => {
    const i = specificItemById('carapaca-demoniaca')!
    expect(i.baseItem).toBe('Armadura Completa')
    expect(i.effect).toMatch(/\+1d8 de dano de trevas/)
  })

  it('Couraça do Comando: +1 Car (+2 se Comandar)', () => {
    const i = specificItemById('couraca-do-comando')!
    expect(i.effect).toMatch(/\+1 em Carisma/)
    expect(i.effect).toMatch(/Comandar/)
  })

  it('todas as armaduras Tabela 8-11 são bookPage 340', () => {
    const tabela8_11 = SPECIFIC_MAGIC_ITEMS.filter(
      (i) => i.category === 'armadura' || i.category === 'escudo',
    )
    expect(tabela8_11.length).toBe(13)
    for (const i of tabela8_11) {
      expect(i.bookPage).toBe(340)
    }
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

  it('classRestrictedItems retorna 7 (todas armas; armaduras sem hard lock)', () => {
    const restricted = classRestrictedItems()
    expect(restricted.length).toBe(7)
    for (const i of restricted) {
      expect(i.category).toBe('arma')
    }
  })

  it('itens sem restrição (24 entradas) têm requiresClass null', () => {
    const unrestricted = SPECIFIC_MAGIC_ITEMS.filter(
      (i) => i.requiresClass === null,
    )
    expect(unrestricted.length).toBe(24)
  })

  it('restritos + livres = 31 total', () => {
    expect(classRestrictedItems().length).toBe(7)
    expect(
      SPECIFIC_MAGIC_ITEMS.filter((i) => i.requiresClass === null).length,
    ).toBe(24)
    expect(SPECIFIC_MAGIC_ITEMS.length).toBe(31)
  })
})

describe('SPECIFIC_MAGIC_ITEMS — lookup helpers', () => {
  it('specificItemById hit', () => {
    expect(specificItemById('avalanche')?.name).toBe('Avalanche')
  })

  it('specificItemById miss', () => {
    expect(specificItemById('martelo-de-thor')).toBeUndefined()
  })

  it('specificItemsByCategory("arma") returns 18 (Tabela 8-9)', () => {
    expect(specificItemsByCategory('arma').length).toBe(18)
  })

  it('specificItemsByCategory("armadura") returns 8 (Tabela 8-11)', () => {
    expect(specificItemsByCategory('armadura').length).toBe(8)
  })

  it('specificItemsByCategory("escudo") returns 5 (Tabela 8-11)', () => {
    expect(specificItemsByCategory('escudo').length).toBe(5)
  })

  it('specificItemsByTier("maior") returns all 31', () => {
    expect(specificItemsByTier('maior').length).toBe(31)
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
      // Tabela 8-11
      'cota-elfica': 'Cota de Malha',
      'couro-de-monstro': 'Gibão de Peles',
      'escudo-do-conjurador': 'Escudo Leve',
      'loriga-do-centuriao': 'Loriga Segmentada',
      'manto-da-noite': 'Couro Batido',
      'couraca-do-comando': 'Couraça',
      'baluarte-anao': 'Armadura Completa',
      'escudo-espinhoso': 'Escudo Pesado',
      'escudo-do-leao': 'Escudo Pesado',
      'carapaca-demoniaca': 'Armadura Completa',
      'escudo-do-eclipse': 'Escudo Pesado',
      'escudo-de-azgher': 'Escudo Pesado',
      'armadura-da-luz': 'Armadura Completa',
    }
    for (const [id, base] of Object.entries(expectations)) {
      expect(specificItemById(id)?.baseItem).toBe(base)
    }
  })
})
