import { describe, expect, it } from 'vitest'
import {
  ORIGEM_BENEFICIOS_PER_CHARACTER,
  ORIGEM_IDS,
  ORIGENS,
  origemById,
  validateBenefits,
} from '../origens'

/**
 * Origens — PDF book p85-95 (Tabela 1-19 p87).
 *
 * Pinned:
 *  - 35 origens total
 *  - Player picks 2 benefícios from combined perícias+poderes
 *  - Each origem has a `poderUnico` (last in poderes list)
 *  - Amnésico is GM-driven (no fixed lists; mandatory Lembranças Graduais)
 *  - Some origens add "+ um poder de combate/Tormenta a sua escolha"
 *  - Soldado / Fazendeiro / Membro de Guilda have 4 perícias fixas
 */

describe('ORIGENS — catalog shape', () => {
  it('has 35 origens', () => {
    expect(ORIGEM_IDS.length).toBe(35)
  })

  it('all ids unique', () => {
    expect(new Set(ORIGEM_IDS).size).toBe(ORIGEM_IDS.length)
  })

  it('every entry round-trips through origemById', () => {
    for (const id of ORIGEM_IDS) {
      expect(origemById(id).id).toBe(id)
    }
  })

  it('origemById throws on unknown id', () => {
    expect(() => origemById('not-real')).toThrow(/unknown origem id/)
  })

  it('every entry has positive bookPage', () => {
    for (const id of ORIGEM_IDS) {
      expect(origemById(id).bookPage).toBeGreaterThan(0)
    }
  })

  it('every non-GM origem has at least 1 perícia', () => {
    for (const id of ORIGEM_IDS) {
      const o = origemById(id)
      if (!o.gmDriven) {
        expect(o.pericias.length).toBeGreaterThan(0)
      }
    }
  })
})

describe('ORIGEM_BENEFICIOS_PER_CHARACTER', () => {
  it('player picks exactly 2 benefícios (book p85)', () => {
    expect(ORIGEM_BENEFICIOS_PER_CHARACTER).toBe(2)
  })
})

describe('Poder Único — present in the origem poderes list', () => {
  it('every origem has its poderUnico inside its poderes list', () => {
    for (const id of ORIGEM_IDS) {
      const o = origemById(id)
      expect(o.poderes).toContain(o.poderUnico)
    }
  })

  it('poderes únicos are themselves unique across origens (excluding Amnésico)', () => {
    const seen: string[] = []
    for (const id of ORIGEM_IDS) {
      const o = origemById(id)
      if (o.gmDriven) continue
      seen.push(o.poderUnico)
    }
    expect(new Set(seen).size).toBe(seen.length)
  })
})

describe('Pinned canonical origens — PDF integrity', () => {
  it('Acólito: 3 perícias + Membro da Igreja, p85', () => {
    const o = origemById('acolito')
    expect(o.pericias).toEqual(['Cura', 'Religião', 'Vontade'])
    expect(o.poderUnico).toBe('Membro da Igreja')
    expect(o.bookPage).toBe(85)
  })

  it('Amigo dos Animais: 2 perícias + Amigo Especial, p85', () => {
    const o = origemById('amigo-dos-animais')
    expect(o.pericias).toEqual(['Adestramento', 'Cavalgar'])
    expect(o.poderUnico).toBe('Amigo Especial')
  })

  it('Soldado: 4 perícias + Influência Militar + combate slot, p94', () => {
    const o = origemById('soldado')
    expect(o.pericias.length).toBe(4)
    expect(o.pericias).toEqual(['Fortitude', 'Guerra', 'Luta', 'Pontaria'])
    expect(o.poderUnico).toBe('Influência Militar')
    expect(o.poderChoiceCategory).toBe('combate')
  })

  it('Charlatão: Alpinista Social poder único, p88', () => {
    const o = origemById('charlatao')
    expect(o.poderUnico).toBe('Alpinista Social')
  })

  it('Batedor: À Prova de Tudo poder único, p88', () => {
    const o = origemById('batedor')
    expect(o.poderUnico).toBe('À Prova de Tudo')
  })
})

describe('Amnésico — GM-driven outlier', () => {
  it('marked gmDriven=true', () => {
    expect(origemById('amnesico').gmDriven).toBe(true)
  })

  it('has Lembranças Graduais as poderObrigatorio', () => {
    expect(origemById('amnesico').poderObrigatorio).toBe('Lembranças Graduais')
  })

  it('has empty fixed perícia list', () => {
    expect(origemById('amnesico').pericias).toEqual([])
  })
})

describe('poderChoiceCategory — origens with +1 power choice', () => {
  it('Capanga / Gladiador / Guarda / Soldado have combate slot', () => {
    for (const id of ['capanga', 'gladiador', 'guarda', 'soldado']) {
      expect(origemById(id).poderChoiceCategory).toBe('combate')
    }
  })

  it('Assistente de Laboratório has Tormenta slot', () => {
    expect(origemById('assistente-de-laboratorio').poderChoiceCategory).toBe(
      'tormenta',
    )
  })

  it('most origens have no extra-choice slot (undefined)', () => {
    expect(origemById('acolito').poderChoiceCategory).toBeUndefined()
    expect(origemById('artesao').poderChoiceCategory).toBeUndefined()
  })
})

describe('validateBenefits — pick rules', () => {
  it('accepts 1 perícia + 1 poder from the origem pool', () => {
    expect(() =>
      validateBenefits('acolito', [
        { kind: 'pericia', name: 'Cura' },
        { kind: 'poder', name: 'Membro da Igreja' },
      ]),
    ).not.toThrow()
  })

  it('accepts 2 perícias from the same origem', () => {
    expect(() =>
      validateBenefits('acolito', [
        { kind: 'pericia', name: 'Cura' },
        { kind: 'pericia', name: 'Vontade' },
      ]),
    ).not.toThrow()
  })

  it('accepts 2 poderes from the same origem', () => {
    expect(() =>
      validateBenefits('artista', [
        { kind: 'poder', name: 'Atraente' },
        { kind: 'poder', name: 'Dom Artístico' },
      ]),
    ).not.toThrow()
  })

  it('rejects fewer than 2 picks', () => {
    expect(() =>
      validateBenefits('acolito', [{ kind: 'pericia', name: 'Cura' }]),
    ).toThrow(/must pick exactly 2/)
  })

  it('rejects more than 2 picks', () => {
    expect(() =>
      validateBenefits('acolito', [
        { kind: 'pericia', name: 'Cura' },
        { kind: 'pericia', name: 'Religião' },
        { kind: 'pericia', name: 'Vontade' },
      ]),
    ).toThrow(/must pick exactly 2/)
  })

  it('rejects perícia not in origem pool', () => {
    expect(() =>
      validateBenefits('acolito', [
        { kind: 'pericia', name: 'Furtividade' },
        { kind: 'pericia', name: 'Cura' },
      ]),
    ).toThrow(/not in origem/)
  })

  it('rejects poder not in origem pool', () => {
    expect(() =>
      validateBenefits('acolito', [
        { kind: 'poder', name: 'Não Existe' },
        { kind: 'pericia', name: 'Cura' },
      ]),
    ).toThrow(/not in origem/)
  })

  it('rejects duplicate identical picks', () => {
    expect(() =>
      validateBenefits('acolito', [
        { kind: 'pericia', name: 'Cura' },
        { kind: 'pericia', name: 'Cura' },
      ]),
    ).toThrow(/duplicate pick/)
  })

  it('refuses to validate Amnésico (GM-driven)', () => {
    expect(() =>
      validateBenefits('amnesico', [
        { kind: 'pericia', name: 'Cura' },
        { kind: 'poder', name: 'Lembranças Graduais' },
      ]),
    ).toThrow(/GM-driven/)
  })
})

describe('Itens iniciais — always granted regardless of benefit picks', () => {
  it('every origem has at least 1 starting item', () => {
    for (const id of ORIGEM_IDS) {
      expect(origemById(id).itensIniciais.length).toBeGreaterThan(0)
    }
  })

  it('Aristocrata has joia de família + traje da corte', () => {
    expect(origemById('aristocrata').itensIniciais).toEqual([
      'Joia de família (T$ 300)',
      'Traje da corte',
    ])
  })
})

describe('ORIGENS immutability', () => {
  it('cannot mutate the ORIGENS record at runtime', () => {
    expect(() => {
      // @ts-expect-error — guarded by Object.freeze
      ORIGENS['ghost'] = {} as never
    }).toThrow()
  })
})
