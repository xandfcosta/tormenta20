import { describe, expect, it } from 'vitest'
import {
  STARTING_KIT_BASE_ITEMS,
  STARTING_KIT_BOOK_PAGE,
  STARTING_TIBARES_DICE,
  arcanistaPathStartingItem,
  startingKitFor,
} from '../class-starting-kits'

/**
 * L1 starting equipment — PDF book p140.
 *
 * Pinned:
 *  - Unified kit, NOT per-class
 *  - Mochila + Saco de dormir + Traje de viajante (always)
 *  - T$ 4d6 for every class
 *  - Arcanista exception: no armor at L1
 *  - Martial weapon granted only if class has armas-marciais proficiency
 *  - Brunea replaces armadura leve only if armaduras-pesadas proficiency
 *  - Shield-leve granted only if escudos proficiency (and not Arcanista)
 *  - Inventor Protótipo (≤ T$500), Nobre Espólio (≤ T$2000) extras
 *  - Arcanista path implicit items: Bruxo foco / Mago grimório / Feiticeiro none
 */

describe('STARTING_KIT_BASE_ITEMS — fixed core', () => {
  it('always includes mochila, saco de dormir, traje de viajante', () => {
    expect([...STARTING_KIT_BASE_ITEMS].sort()).toEqual([
      'Mochila',
      'Saco de dormir',
      'Traje de viajante',
    ])
  })
})

describe('STARTING_TIBARES_DICE — uniform across classes', () => {
  it('every L1 character gets T$ 4d6', () => {
    expect(STARTING_TIBARES_DICE).toBe('4d6')
  })

  it('book page is 140 (Cap 3 Equipamento)', () => {
    expect(STARTING_KIT_BOOK_PAGE).toBe(140)
  })
})

describe('startingKitFor — Arcanista (exception class)', () => {
  it('Arcanista starts WITHOUT armor (exception explícita p140)', () => {
    const kit = startingKitFor('Arcanista')
    expect(kit.armor).toBe('nenhuma')
  })

  it('Arcanista has no shield even though normally tied to proficiency', () => {
    const kit = startingKitFor('Arcanista')
    expect(kit.shieldLeve).toBe(false)
  })

  it('Arcanista has no marciais → gets only simples weapon', () => {
    expect(startingKitFor('Arcanista').weapons).toBe('simples')
  })
})

describe('startingKitFor — full-martial classes', () => {
  it('Guerreiro: marciais + brunea + escudo leve', () => {
    const kit = startingKitFor('Guerreiro')
    expect(kit.weapons).toBe('simples+marcial')
    expect(kit.armor).toBe('brunea')
    expect(kit.shieldLeve).toBe(true)
  })

  it('Paladino: marciais + brunea + escudo leve', () => {
    const kit = startingKitFor('Paladino')
    expect(kit.weapons).toBe('simples+marcial')
    expect(kit.armor).toBe('brunea')
    expect(kit.shieldLeve).toBe(true)
  })

  it('Cavaleiro: marciais + brunea + escudo leve', () => {
    const kit = startingKitFor('Cavaleiro')
    expect(kit.armor).toBe('brunea')
    expect(kit.shieldLeve).toBe(true)
  })
})

describe('startingKitFor — light-armor martial classes', () => {
  it('Bárbaro: marciais + leve + escudo leve (no heavy armor)', () => {
    const kit = startingKitFor('Bárbaro')
    expect(kit.weapons).toBe('simples+marcial')
    expect(kit.armor).toBe('leve-a-escolha')
    expect(kit.shieldLeve).toBe(true)
  })

  it('Caçador: marciais + leve + escudo leve', () => {
    const kit = startingKitFor('Caçador')
    expect(kit.armor).toBe('leve-a-escolha')
    expect(kit.shieldLeve).toBe(true)
  })

  it('Bardo: marciais + leve + NO shield', () => {
    const kit = startingKitFor('Bardo')
    expect(kit.weapons).toBe('simples+marcial')
    expect(kit.armor).toBe('leve-a-escolha')
    expect(kit.shieldLeve).toBe(false)
  })

  it('Bucaneiro: marciais + leve + NO shield', () => {
    const kit = startingKitFor('Bucaneiro')
    expect(kit.shieldLeve).toBe(false)
  })
})

describe('startingKitFor — divine spellcasters', () => {
  it('Clérigo: SIMPLES only (no marciais) + brunea + escudo leve', () => {
    const kit = startingKitFor('Clérigo')
    expect(kit.weapons).toBe('simples')
    expect(kit.armor).toBe('brunea')
    expect(kit.shieldLeve).toBe(true)
  })

  it('Druida: simples + leve + escudo leve (no heavy)', () => {
    const kit = startingKitFor('Druida')
    expect(kit.weapons).toBe('simples')
    expect(kit.armor).toBe('leve-a-escolha')
    expect(kit.shieldLeve).toBe(true)
  })
})

describe('startingKitFor — light classes without proficiencies', () => {
  it('Ladino: simples + leve + no shield', () => {
    const kit = startingKitFor('Ladino')
    expect(kit.weapons).toBe('simples')
    expect(kit.armor).toBe('leve-a-escolha')
    expect(kit.shieldLeve).toBe(false)
  })

  it('Lutador: simples + leve + no shield', () => {
    const kit = startingKitFor('Lutador')
    expect(kit.weapons).toBe('simples')
    expect(kit.shieldLeve).toBe(false)
  })
})

describe('startingKitFor — Inventor extras (Protótipo, p68)', () => {
  it('Inventor receives Protótipo extra ≤ T$ 500', () => {
    const kit = startingKitFor('Inventor')
    expect(kit.extras.length).toBe(1)
    expect(kit.extras[0].source).toMatch(/Protótipo/)
    expect(kit.extras[0].maxValueTibar).toBe(500)
  })
})

describe('startingKitFor — Nobre extras (Espólio, p79)', () => {
  it('Nobre receives Espólio extra ≤ T$ 2.000', () => {
    const kit = startingKitFor('Nobre')
    expect(kit.extras.length).toBe(1)
    expect(kit.extras[0].source).toMatch(/Espólio/)
    expect(kit.extras[0].maxValueTibar).toBe(2000)
  })

  it('Nobre also gets marciais + brunea + escudo (full-martial profile)', () => {
    const kit = startingKitFor('Nobre')
    expect(kit.weapons).toBe('simples+marcial')
    expect(kit.armor).toBe('brunea')
    expect(kit.shieldLeve).toBe(true)
  })
})

describe('startingKitFor — non-extra classes have empty extras', () => {
  it('Guerreiro has no class-specific extra', () => {
    expect(startingKitFor('Guerreiro').extras).toEqual([])
  })

  it('Arcanista (base) has no automatic extras', () => {
    expect(startingKitFor('Arcanista').extras).toEqual([])
  })
})

describe('startingKitFor — uniform fields across all classes', () => {
  const allClasses = [
    'Arcanista',
    'Bárbaro',
    'Bardo',
    'Bucaneiro',
    'Caçador',
    'Cavaleiro',
    'Clérigo',
    'Druida',
    'Guerreiro',
    'Inventor',
    'Ladino',
    'Lutador',
    'Nobre',
    'Paladino',
  ]

  it.each(allClasses.map((c) => [c]))(
    '%s starts with 4d6 Tibares and the base 3 items',
    (className) => {
      const kit = startingKitFor(className)
      expect(kit.tibarDice).toBe('4d6')
      expect(kit.baseItems).toEqual(STARTING_KIT_BASE_ITEMS)
    },
  )
})

describe('arcanistaPathStartingItem — implicit items per Caminho (p37)', () => {
  it('bruxo → Foco arcano', () => {
    expect(arcanistaPathStartingItem('bruxo')).toBe('Foco arcano')
  })

  it('mago → Grimório', () => {
    expect(arcanistaPathStartingItem('mago')).toBe('Grimório')
  })

  it('feiticeiro → null (linhagem sobrenatural, no focus needed)', () => {
    expect(arcanistaPathStartingItem('feiticeiro')).toBeNull()
  })
})
