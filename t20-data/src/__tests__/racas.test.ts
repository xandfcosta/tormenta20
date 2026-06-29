import { describe, expect, it } from 'vitest'
import {
  RACAS,
  RACA_IDS,
  racaById,
  racasByTier,
  resolveAtributoMod,
} from '../racas'

/**
 * Raças — PDF book p18-31 (Tabela 1-2 master list p18).
 *
 * Pinned:
 *  - 17 raças total (8 comuns + 9 extras)
 *  - Atributo mods: fixed / floating / subraca-gated
 *  - Tamanho/deslocamento defaults Médio/9m; overrides for Anão (6m),
 *    Goblin (Pequeno), Hynne (Pequeno/6m), Sílfide (Minúsculo), Golem (6m),
 *    Elfo (12m via Graça de Glórienn)
 *  - Visão no escuro: Anão, Goblin, Golem, Medusa, Osteon, Suraggel, Trog
 *  - Visão na penumbra: Elfo, Sílfide
 *  - Humano/Lefou/Osteon/Sereia use floating +1 trio; Lefou + Osteon exclude one + penalty
 *  - Suraggel uses subraca-gated mods (Aggelus vs Sulfure)
 */

describe('RACAS — catalog shape', () => {
  it('has 17 raças total (8 comuns + 9 extras)', () => {
    expect(RACA_IDS.length).toBe(17)
  })

  it('all ids unique', () => {
    expect(new Set(RACA_IDS).size).toBe(RACA_IDS.length)
  })

  it('every entry round-trips through racaById', () => {
    for (const id of RACA_IDS) {
      expect(racaById(id).id).toBe(id)
    }
  })

  it('racaById throws on unknown id', () => {
    expect(() => racaById('not-real')).toThrow(/unknown raça id/)
  })

  it('every entry has positive bookPage', () => {
    for (const id of RACA_IDS) {
      expect(racaById(id).bookPage).toBeGreaterThan(0)
    }
  })

  it('every raça has at least one habilidade racial', () => {
    for (const id of RACA_IDS) {
      expect(racaById(id).abilities.length).toBeGreaterThan(0)
    }
  })
})

describe('racasByTier — common vs extra', () => {
  it('8 comuns (Humano, Anão, Dahllan, Elfo, Goblin, Lefou, Minotauro, Qareen)', () => {
    expect(racasByTier('comum').length).toBe(8)
  })

  it('9 extras (Golem, Hynne, Kliren, Medusa, Osteon, Sereia/Tritão, Sílfide, Suraggel, Trog)', () => {
    expect(racasByTier('extra').length).toBe(9)
  })
})

describe('Tamanho / deslocamento — pinned cases', () => {
  it('Anão: Médio + 6m', () => {
    const a = racaById('anao')
    expect(a.tamanho).toBe('Médio')
    expect(a.deslocamento).toBe(6)
  })

  it('Goblin: Pequeno + 9m (Peste Esguia)', () => {
    const g = racaById('goblin')
    expect(g.tamanho).toBe('Pequeno')
    expect(g.deslocamento).toBe(9)
  })

  it('Hynne: Pequeno + 6m', () => {
    const h = racaById('hynne')
    expect(h.tamanho).toBe('Pequeno')
    expect(h.deslocamento).toBe(6)
  })

  it('Sílfide: Minúsculo + 9m base', () => {
    const s = racaById('silfide')
    expect(s.tamanho).toBe('Minúsculo')
    expect(s.deslocamento).toBe(9)
  })

  it('Golem: Médio + 6m', () => {
    expect(racaById('golem').deslocamento).toBe(6)
  })

  it('Elfo: 12m (Graça de Glórienn)', () => {
    expect(racaById('elfo').deslocamento).toBe(12)
  })

  it('Humano: Médio + 9m baseline', () => {
    const h = racaById('humano')
    expect(h.tamanho).toBe('Médio')
    expect(h.deslocamento).toBe(9)
  })
})

describe('Visão — categorized', () => {
  it.each([
    ['anao'],
    ['goblin'],
    ['golem'],
    ['medusa'],
    ['osteon'],
    ['suraggel'],
    ['trog'],
  ])('%s has visão no escuro', (id) => {
    expect(racaById(id).visaoNoEscuro).toBe(true)
  })

  it.each([['elfo'], ['silfide']])('%s has visão na penumbra', (id) => {
    expect(racaById(id).visaoNaPenumbra).toBe(true)
  })

  it('Humano has neither (normal vision)', () => {
    const h = racaById('humano')
    expect(h.visaoNoEscuro).toBe(false)
    expect(h.visaoNaPenumbra).toBe(false)
  })
})

describe('Atributo modifiers — fixed shape', () => {
  it('Anão: Con +2 / Sab +1 / Des −1', () => {
    expect(resolveAtributoMod(racaById('anao'))).toEqual({
      constitution: 2,
      wisdom: 1,
      dexterity: -1,
    })
  })

  it('Elfo: Int +2 / Des +1 / Con −1', () => {
    expect(resolveAtributoMod(racaById('elfo'))).toEqual({
      intelligence: 2,
      dexterity: 1,
      constitution: -1,
    })
  })

  it('Medusa: Des +2 / Car +1 (no penalty)', () => {
    expect(resolveAtributoMod(racaById('medusa'))).toEqual({
      dexterity: 2,
      charisma: 1,
    })
  })

  it('Sílfide: Car +2 / Des +1 / For −2 (double-penalty For)', () => {
    expect(resolveAtributoMod(racaById('silfide'))).toEqual({
      charisma: 2,
      dexterity: 1,
      strength: -2,
    })
  })
})

describe('Atributo modifiers — floating shape', () => {
  it('Humano: picks 3 distinct → each +1', () => {
    const mods = resolveAtributoMod(racaById('humano'), {
      floatingPicks: ['strength', 'wisdom', 'charisma'],
    })
    expect(mods).toEqual({ strength: 1, wisdom: 1, charisma: 1 })
  })

  it('Humano rejects fewer than 3 picks', () => {
    expect(() =>
      resolveAtributoMod(racaById('humano'), {
        floatingPicks: ['strength', 'wisdom'],
      }),
    ).toThrow(/requires exactly 3 floating picks/)
  })

  it('Humano rejects duplicate picks', () => {
    expect(() =>
      resolveAtributoMod(racaById('humano'), {
        floatingPicks: ['strength', 'strength', 'wisdom'],
      }),
    ).toThrow(/picks must be distinct/)
  })

  it('Lefou: 3 picks exclude Carisma; carries Car −1', () => {
    const mods = resolveAtributoMod(racaById('lefou'), {
      floatingPicks: ['strength', 'dexterity', 'wisdom'],
    })
    expect(mods).toEqual({
      strength: 1,
      dexterity: 1,
      wisdom: 1,
      charisma: -1,
    })
  })

  it('Lefou rejects floating pick on Carisma (exclude)', () => {
    expect(() =>
      resolveAtributoMod(racaById('lefou'), {
        floatingPicks: ['strength', 'dexterity', 'charisma'],
      }),
    ).toThrow(/cannot place \+1 in charisma/)
  })

  it('Osteon: 3 picks exclude Con; carries Con −1', () => {
    const mods = resolveAtributoMod(racaById('osteon'), {
      floatingPicks: ['intelligence', 'wisdom', 'charisma'],
    })
    expect(mods).toEqual({
      intelligence: 1,
      wisdom: 1,
      charisma: 1,
      constitution: -1,
    })
  })

  it('Sereia/Tritão: 3 floating picks, no penalty', () => {
    const mods = resolveAtributoMod(racaById('sereia-tritao'), {
      floatingPicks: ['constitution', 'wisdom', 'charisma'],
    })
    expect(mods).toEqual({ constitution: 1, wisdom: 1, charisma: 1 })
  })
})

describe('Atributo modifiers — subraca-gated (Suraggel)', () => {
  it('Aggelus: Sab +2 / Car +1', () => {
    const mods = resolveAtributoMod(racaById('suraggel'), {
      ascendencia: 'aggelus',
    })
    expect(mods).toEqual({ wisdom: 2, charisma: 1 })
  })

  it('Sulfure: Des +2 / Int +1', () => {
    const mods = resolveAtributoMod(racaById('suraggel'), {
      ascendencia: 'sulfure',
    })
    expect(mods).toEqual({ dexterity: 2, intelligence: 1 })
  })

  it('rejects missing ascendência', () => {
    expect(() => resolveAtributoMod(racaById('suraggel'))).toThrow(
      /requires ascendência in \[aggelus, sulfure\]/,
    )
  })

  it('rejects invalid ascendência', () => {
    expect(() =>
      resolveAtributoMod(racaById('suraggel'), { ascendencia: 'ghost' }),
    ).toThrow(/requires ascendência in/)
  })
})

describe('Ascendências — listed where relevant', () => {
  it('Qareen has 6 elemental ascendências', () => {
    expect(racaById('qareen').ascendencias?.length).toBe(6)
  })

  it('Golem has 4 fontes elementais', () => {
    expect(racaById('golem').ascendencias?.length).toBe(4)
  })

  it('Suraggel has Aggelus and Sulfure', () => {
    expect([...(racaById('suraggel').ascendencias ?? [])].sort()).toEqual([
      'aggelus',
      'sulfure',
    ])
  })

  it('Anão has no ascendências (single profile)', () => {
    expect(racaById('anao').ascendencias).toBeUndefined()
  })
})

describe('RACAS immutability', () => {
  it('cannot mutate the RACAS record at runtime', () => {
    expect(() => {
      // @ts-expect-error — guarded by Object.freeze
      RACAS['ghost'] = {} as never
    }).toThrow()
  })
})
