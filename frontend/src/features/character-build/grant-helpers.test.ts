import { describe, expect, it } from 'vitest'
import {
  anyRacePending,
  classGrant,
  classTiles,
  deformidadePayload,
  deformidadeSummary,
  originGrant,
  raceAttributeDeltas,
  raceChoiceMeta,
  raceGrant,
  racePending,
  racesByTier,
  raceSignature,
  resolveRaceDeltas,
  signed,
} from './grant-helpers'

describe('signed', () => {
  it('prefixes non-negative with +', () => {
    expect(signed(2)).toBe('+2')
    expect(signed(0)).toBe('+0')
    expect(signed(-1)).toBe('-1')
  })
})

describe('raceAttributeDeltas', () => {
  it('returns the fixed racial bonuses of a single race', () => {
    expect(raceAttributeDeltas(['Anão'])).toEqual({
      constitution: 2,
      wisdom: 1,
      dexterity: -1,
    })
  })

  it('sums bonuses across multiple races', () => {
    // Anão CON+2/SAB+1/DES-1 + Elfo INT+2/DES+1/CON-1
    expect(raceAttributeDeltas(['Anão', 'Elfo'])).toEqual({
      constitution: 1,
      wisdom: 1,
      dexterity: 0,
      intelligence: 2,
    })
  })

  it('excludes floating-choice races (Humano gets no fixed bonus)', () => {
    expect(raceAttributeDeltas(['Humano'])).toEqual({})
  })

  it('ignores unknown race ids', () => {
    expect(raceAttributeDeltas(['NotARace'])).toEqual({})
  })

  it('folds floating picks + guaranteed penalty via choices', () => {
    // Lefou: pick 3 (+1 each), guaranteed −1 Carisma
    expect(
      raceAttributeDeltas(['Lefou'], {
        Lefou: { floatingPicks: ['strength', 'dexterity', 'wisdom'] },
      }),
    ).toEqual({ strength: 1, dexterity: 1, wisdom: 1, charisma: -1 })
  })

  it('folds subrace (Suraggel ascendência) via choices', () => {
    expect(
      raceAttributeDeltas(['Suraggel'], { Suraggel: { ascendencia: 'aggelus' } }),
    ).toEqual({ wisdom: 2, charisma: 1 })
  })
})

describe('resolveRaceDeltas — live-safe partial resolution', () => {
  it('applies floating penalty even before picks are placed', () => {
    // Osteon: −1 Con guaranteed; no +1s yet
    expect(resolveRaceDeltas('Osteon', {})).toEqual({ constitution: -1 })
  })

  it('applies partial floating picks as they are made', () => {
    expect(resolveRaceDeltas('Humano', { floatingPicks: ['strength'] })).toEqual(
      { strength: 1 },
    )
  })

  it('ignores an excluded floating pick', () => {
    // Lefou cannot place +1 in Carisma
    expect(
      resolveRaceDeltas('Lefou', { floatingPicks: ['charisma', 'strength'] }),
    ).toEqual({ charisma: -1, strength: 1 })
  })

  it('returns empty for an unset subrace', () => {
    expect(resolveRaceDeltas('Suraggel', {})).toEqual({})
  })

  it('deformidade com poder trocado perde 1 CAR no preview (p136)', () => {
    expect(
      resolveRaceDeltas('Lefou', {
        deformidade: { pericias: ['Furtividade'], tormentaPower: 'dentes-afiados' },
      }),
    ).toEqual({ charisma: -2 }) // −1 penalidade Lefou − 1 poder
  })

  it('deformidade só com perícias não perde CAR (p23)', () => {
    expect(
      resolveRaceDeltas('Lefou', {
        deformidade: { pericias: ['Furtividade', 'Percepção'] },
      }),
    ).toEqual({ charisma: -1 }) // apenas a penalidade racial
  })

  it('deformidade em raça sem a habilidade não altera nada', () => {
    expect(
      resolveRaceDeltas('Humano', {
        deformidade: { pericias: [], tormentaPower: 'antenas' },
      }),
    ).toEqual({})
  })
})

describe('deformidadePayload — draft → payload de submit', () => {
  it('remove slots vazios e poder não escolhido', () => {
    expect(
      deformidadePayload('Lefou', {
        deformidade: { pericias: ['Furtividade', ''], tormentaPower: '' },
      }),
    ).toEqual({ pericias: ['Furtividade'], tormentaPower: undefined })
  })

  it('undefined quando nada foi escolhido', () => {
    expect(
      deformidadePayload('Lefou', { deformidade: { pericias: [] } }),
    ).toBeUndefined()
    expect(deformidadePayload('Lefou', {})).toBeUndefined()
  })

  it('undefined para raça sem a habilidade (draft stale)', () => {
    expect(
      deformidadePayload('Humano', {
        deformidade: { pericias: ['Furtividade'] },
      }),
    ).toBeUndefined()
  })
})

describe('racePending / anyRacePending', () => {
  it('a fixed race is never pending', () => {
    expect(racePending('Anão')).toBe(false)
  })

  it('a floating race is pending until all picks are placed', () => {
    expect(racePending('Humano', { floatingPicks: ['strength'] })).toBe(true)
    expect(
      racePending('Humano', {
        floatingPicks: ['strength', 'wisdom', 'charisma'],
      }),
    ).toBe(false)
  })

  it('a subrace race is pending until an ascendência is chosen', () => {
    expect(racePending('Suraggel')).toBe(true)
    expect(racePending('Suraggel', { ascendencia: 'sulfure' })).toBe(false)
  })

  it('only the primary race can be pending (secondary races are GM-negotiated)', () => {
    // Primary Suraggel needs an ascendência → pending.
    expect(anyRacePending(['Suraggel', 'Anão'], {})).toBe(true)
    // Primary Anão is fixed; a pending SECONDARY (Suraggel) is ignored.
    expect(anyRacePending(['Anão', 'Suraggel'], {})).toBe(false)
    expect(anyRacePending(['Anão'], {})).toBe(false)
  })
})

describe('raceSignature', () => {
  it('shows the top positive fixed delta', () => {
    expect(raceSignature('Anão')).toBe('+2 CON')
  })
  it('shows +1×3 for a floating race', () => {
    expect(raceSignature('Humano')).toBe('+1×3')
  })
  it('flags a subrace race', () => {
    expect(raceSignature('Suraggel')).toBe('2 ascend.')
  })
})

describe('raceChoiceMeta', () => {
  it('reports floating params incl. exclude + penalty', () => {
    expect(raceChoiceMeta('Lefou')).toEqual({
      kind: 'floating',
      count: 3,
      value: 1,
      exclude: 'charisma',
      penalty: { attribute: 'charisma', value: -1 },
    })
  })
  it('reports subrace options', () => {
    const meta = raceChoiceMeta('Suraggel')
    expect(meta.kind).toBe('subrace')
    if (meta.kind === 'subrace') {
      expect(meta.options.sort()).toEqual(['aggelus', 'sulfure'])
    }
  })
  it('reports none for a fixed race', () => {
    expect(raceChoiceMeta('Anão')).toEqual({ kind: 'none' })
  })
})

describe('racesByTier / classTiles', () => {
  it('splits available race names into comuns + extras', () => {
    const { comuns, extras } = racesByTier(['Anão', 'Suraggel', 'Elfo'])
    expect(comuns).toEqual(['Anão', 'Elfo'])
    expect(extras).toEqual(['Suraggel'])
  })

  it('classTiles attaches level-1 vitals', () => {
    const [g] = classTiles(['Guerreiro'])
    expect(g.className).toBe('Guerreiro')
    expect(g.pvInicial).toBeGreaterThan(0)
  })
})

describe('raceGrant', () => {
  it('lists attribute deltas and abilities for a known race', () => {
    const grant = raceGrant('Anão')
    expect(grant?.name).toBe('Anão')
    expect(grant?.deltas).toContainEqual(['constitution', 2])
    expect(grant?.abilities.length).toBeGreaterThan(0)
  })

  it('returns null for an unknown race', () => {
    expect(raceGrant('NotARace')).toBeNull()
  })
})

describe('classGrant', () => {
  it('exposes class vitals for a known class', () => {
    const grant = classGrant('Guerreiro')
    expect(grant.vitals?.pvInicial).toBeGreaterThan(0)
  })

  it('returns null vitals for an unknown class', () => {
    expect(classGrant('NotAClass').vitals).toBeNull()
  })

  it('reveals more (or equal) auto powers as the level rises', () => {
    const l1 = classGrant('Guerreiro', 1).powers.length
    const l20 = classGrant('Guerreiro', 20).powers.length
    expect(l20).toBeGreaterThanOrEqual(l1)
  })

  it('labels each power with the level it is gained', () => {
    for (const line of classGrant('Guerreiro', 20).powers) {
      expect(line.name).toMatch(/^Nv \d+ · /)
    }
  })
})

describe('originGrant', () => {
  it('lists benefits and the unique power for a known origin', () => {
    const grant = originGrant('Acólito')
    expect(grant?.name).toBe('Acólito')
    expect(grant?.benefits.length).toBeGreaterThan(0)
    expect(grant?.poderUnico?.name).toBeTruthy()
  })

  it('returns null for an unknown origin', () => {
    expect(originGrant('NotAnOrigin')).toBeNull()
  })
})

describe('deformidadeSummary — linha de escolhas do Resumo', () => {
  it('perícias + poder trocado', () => {
    expect(
      deformidadeSummary('Lefou', {
        deformidade: { pericias: ['Furtividade'], tormentaPower: 'dentes-afiados' },
      }),
    ).toBe('Deformidade: +2 Furtividade · poder da Tormenta: Dentes Afiados (−1 CAR)')
  })

  it('só perícias', () => {
    expect(
      deformidadeSummary('Lefou', {
        deformidade: { pericias: ['Furtividade', 'Percepção'] },
      }),
    ).toBe('Deformidade: +2 Furtividade · +2 Percepção')
  })

  it('null sem escolha ou raça sem a habilidade', () => {
    expect(deformidadeSummary('Lefou', {})).toBeNull()
    expect(
      deformidadeSummary('Humano', { deformidade: { pericias: ['Furtividade'] } }),
    ).toBeNull()
  })
})
