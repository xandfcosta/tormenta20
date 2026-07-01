import { describe, expect, it } from 'vitest'
import { RACAS } from '../racas'
import {
  RACIAL_POWERS,
  pmConsumingPowers,
  racialPowersOf,
  totalFixedPmCost,
  type RacialPower,
} from '../racial-power-mechanics'

/**
 * PDF Cap 1 (Construção de Personagem) — Raças p19-31.
 * Pinned:
 *  - Cada RacialPower referencia racaId existente em RACAS.
 *  - Poderes de conjuração usam pmCost 'variavel' (Canção dos Mares,
 *    Magia das Fadas).
 *  - Poderes 1/rodada (Chifres, Asas de Borboleta, Mordida) usam
 *    uses: 'rodada'.
 *  - Anão/Elfo/Goblin/Lefou/Golem/Osteon/Humano: nenhum poder ativo
 *    (só passivos).
 */

describe('RACIAL_POWERS — shape', () => {
  it('não vazio', () => {
    expect(RACIAL_POWERS.length).toBeGreaterThan(0)
  })

  it('frozen', () => {
    expect(Object.isFrozen(RACIAL_POWERS)).toBe(true)
  })

  it('cada racaId existe em RACAS', () => {
    for (const p of RACIAL_POWERS) {
      expect(RACAS[p.racaId]).toBeDefined()
    }
  })

  it('bookPage entre 19 e 31 (Cap 1 raças)', () => {
    for (const p of RACIAL_POWERS) {
      expect(p.bookPage).toBeGreaterThanOrEqual(19)
      expect(p.bookPage).toBeLessThanOrEqual(31)
    }
  })

  it('actions são values válidos', () => {
    const valid = new Set([
      'padrao',
      'movimento',
      'livre',
      'reacao',
      'gratuita',
      'completa',
      'varia',
    ])
    for (const p of RACIAL_POWERS) {
      expect(valid.has(p.action)).toBe(true)
    }
  })
})

describe('RACIAL_POWERS — pinned entries', () => {
  it('Dahllan tem 2 poderes ativos', () => {
    const powers = racialPowersOf('dahllan')
    expect(powers.length).toBe(2)
    expect(powers.map((p) => p.name)).toEqual([
      'Amiga das Plantas',
      'Armadura de Allihanna',
    ])
  })

  it('Amiga das Plantas: padrao, 3 PM, ilimitado', () => {
    const p = RACIAL_POWERS.find((p) => p.name === 'Amiga das Plantas')!
    expect(p.action).toBe('padrao')
    expect(p.pmCost).toBe(3)
    expect(p.uses).toBeNull()
  })

  it('Armadura de Allihanna: movimento, 1 PM, 1/cena', () => {
    const p = RACIAL_POWERS.find((p) => p.name === 'Armadura de Allihanna')!
    expect(p.action).toBe('movimento')
    expect(p.pmCost).toBe(1)
    expect(p.uses).toBe('cena')
  })

  it('Chifres: livre, 1 PM, 1/rodada', () => {
    const p = RACIAL_POWERS.find((p) => p.name === 'Chifres')!
    expect(p.action).toBe('livre')
    expect(p.pmCost).toBe(1)
    expect(p.uses).toBe('rodada')
  })

  it('Sorte Salvadora (Hynne): reacao, 1 PM', () => {
    const p = RACIAL_POWERS.find((p) => p.name === 'Sorte Salvadora')!
    expect(p.action).toBe('reacao')
    expect(p.pmCost).toBe(1)
    expect(p.racaId).toBe('hynne')
  })

  it('Engenhosidade (Kliren): livre, 2 PM', () => {
    const p = RACIAL_POWERS.find((p) => p.name === 'Engenhosidade')!
    expect(p.action).toBe('livre')
    expect(p.pmCost).toBe(2)
  })

  it('Canção dos Mares (Sereia/Tritão): pmCost variavel', () => {
    const p = RACIAL_POWERS.find((p) => p.name === 'Canção dos Mares')!
    expect(p.pmCost).toBe('variavel')
    expect(p.racaId).toBe('sereia-tritao')
  })

  it('Magia das Fadas (Sílfide): pmCost variavel', () => {
    const p = RACIAL_POWERS.find((p) => p.name === 'Magia das Fadas')!
    expect(p.pmCost).toBe('variavel')
    expect(p.racaId).toBe('silfide')
  })

  it('Suraggel tem Luz Sagrada + Sombras Profanas', () => {
    const powers = racialPowersOf('suraggel').map((p) => p.name)
    expect(powers).toEqual(['Luz Sagrada', 'Sombras Profanas'])
  })

  it('Trog tem Mau Cheiro (padrao) + Mordida (livre/rodada)', () => {
    const powers = racialPowersOf('trog')
    expect(powers.length).toBe(2)
    const cheiro = powers.find((p) => p.name === 'Mau Cheiro')!
    expect(cheiro.action).toBe('padrao')
    expect(cheiro.pmCost).toBe(2)
    const mordida = powers.find((p) => p.name === 'Mordida')!
    expect(mordida.uses).toBe('rodada')
  })

  it('Medusa tem 2 poderes: Natureza Venenosa + Olhar Atordoante', () => {
    const powers = racialPowersOf('medusa').map((p) => p.name)
    expect(powers).toEqual(['Natureza Venenosa', 'Olhar Atordoante'])
  })
})

describe('racialPowersOf', () => {
  it('Anão sem poderes ativos (todos passivos)', () => {
    expect(racialPowersOf('anao')).toEqual([])
  })

  it('Elfo sem poderes ativos', () => {
    expect(racialPowersOf('elfo')).toEqual([])
  })

  it('Goblin sem poderes ativos', () => {
    expect(racialPowersOf('goblin')).toEqual([])
  })

  it('Humano sem poderes ativos (Versátil = passivo)', () => {
    expect(racialPowersOf('humano')).toEqual([])
  })

  it('Golem sem poderes ativos', () => {
    expect(racialPowersOf('golem')).toEqual([])
  })

  it('Osteon sem poderes ativos', () => {
    expect(racialPowersOf('osteon')).toEqual([])
  })

  it('miss retorna array vazio', () => {
    expect(racialPowersOf('inexistente')).toEqual([])
  })
})

describe('pmConsumingPowers', () => {
  it('inclui todos os RACIAL_POWERS (nenhum tem pmCost 0)', () => {
    expect(pmConsumingPowers().length).toBe(RACIAL_POWERS.length)
  })
})

describe('totalFixedPmCost', () => {
  it('soma custos fixos ignorando "variavel"', () => {
    const powers: RacialPower[] = [
      { racaId: 'x', name: 'a', action: 'padrao', pmCost: 2, uses: null, bookPage: 20 },
      { racaId: 'x', name: 'b', action: 'padrao', pmCost: 'variavel', uses: null, bookPage: 20 },
      { racaId: 'x', name: 'c', action: 'livre', pmCost: 1, uses: null, bookPage: 20 },
    ]
    expect(totalFixedPmCost(powers)).toBe(3)
  })

  it('array vazio = 0', () => {
    expect(totalFixedPmCost([])).toBe(0)
  })

  it('só variaveis = 0', () => {
    const powers = racialPowersOf('silfide').filter((p) => p.pmCost === 'variavel')
    expect(totalFixedPmCost(powers)).toBe(0)
  })

  it('Dahllan (Amiga 3 + Allihanna 1) = 4', () => {
    expect(totalFixedPmCost(racialPowersOf('dahllan'))).toBe(4)
  })
})
