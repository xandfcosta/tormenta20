import { describe, expect, it } from 'vitest'
import { getActivation } from '@tormenta20/t20-data'
import type { ActivationSpec } from '@tormenta20/t20-data'
import type { Character } from '@/shared/api/api'
import {
  enforcedScopeOf,
  grantPowersForFlag,
  powerUseDecision,
  resolveActivationSpec,
  stanceActivationDecision,
  stanceClassLevel,
  stanceFlagOf,
  stanceMaxSteps,
  stanceTotalPm,
  type PowerUseContext,
} from './use-power-action'

// Named fakes — hand-built specs/contexts so each case controls exactly the
// axis under test instead of depending on live registry contents.
function fakeSpec(overrides: Partial<ActivationSpec> = {}): ActivationSpec {
  return {
    id: 'class.teste.poder-teste',
    name: 'Poder Teste',
    kind: 'instant',
    action: 'padrao',
    pmCost: 2,
    uses: null,
    bookPage: 1,
    ...overrides,
  }
}

function fakeContext(overrides: Partial<PowerUseContext> = {}): PowerUseContext {
  return {
    mpCurrent: 10,
    usedScene: 0,
    usedDay: 0,
    activeFlags: new Set<string>(),
    ...overrides,
  }
}

describe('powerUseDecision', () => {
  it('ok path — enough PM, no limit, no flag required', () => {
    expect(powerUseDecision(fakeSpec(), fakeContext())).toEqual({ ok: true })
  })

  it('refuses when PM is insufficient', () => {
    expect(
      powerUseDecision(fakeSpec({ pmCost: 3 }), fakeContext({ mpCurrent: 2 })),
    ).toEqual({ ok: false, reason: 'PM insuficiente' })
  })

  it('allows spending the last PM (cost == current)', () => {
    expect(
      powerUseDecision(fakeSpec({ pmCost: 2 }), fakeContext({ mpCurrent: 2 })),
    ).toEqual({ ok: true })
  })

  it('refuses a scene-limited power already used this scene', () => {
    expect(
      powerUseDecision(fakeSpec({ uses: 'cena' }), fakeContext({ usedScene: 1 })),
    ).toEqual({ ok: false, reason: 'limite por cena atingido' })
  })

  it('refuses a day-limited power already used today', () => {
    expect(
      powerUseDecision(fakeSpec({ uses: 'dia' }), fakeContext({ usedDay: 1 })),
    ).toEqual({ ok: false, reason: 'limite por dia atingido' })
  })

  it('refuses when the required stance flag is inactive', () => {
    expect(
      powerUseDecision(fakeSpec({ requiresFlag: 'furia' }), fakeContext()),
    ).toEqual({ ok: false, reason: 'requer furia' })
  })

  it('passes when the required stance flag is active', () => {
    expect(
      powerUseDecision(
        fakeSpec({ requiresFlag: 'furia' }),
        fakeContext({ activeFlags: new Set(['furia']) }),
      ),
    ).toEqual({ ok: true })
  })

  it('treats variable-cost powers as unusable in one tap', () => {
    expect(powerUseDecision(fakeSpec({ pmCost: 'variavel' }), fakeContext())).toEqual(
      { ok: false, reason: 'custo variável' },
    )
  })

  it('leaves rodada/numeric limits unenforced (badge-only this phase)', () => {
    expect(
      powerUseDecision(fakeSpec({ uses: 'rodada' }), fakeContext({ usedScene: 5 })),
    ).toEqual({ ok: true })
    expect(
      powerUseDecision(fakeSpec({ uses: 3 }), fakeContext({ usedDay: 5 })),
    ).toEqual({ ok: true })
  })
})

describe('enforcedScopeOf', () => {
  it('maps cena→scene, dia→day, everything else→undefined', () => {
    expect(enforcedScopeOf(fakeSpec({ uses: 'cena' }))).toBe('scene')
    expect(enforcedScopeOf(fakeSpec({ uses: 'dia' }))).toBe('day')
    expect(enforcedScopeOf(fakeSpec({ uses: null }))).toBeUndefined()
    expect(enforcedScopeOf(fakeSpec({ uses: 'rodada' }))).toBeUndefined()
    expect(enforcedScopeOf(fakeSpec({ uses: 3 }))).toBeUndefined()
  })
})

// Live Fúria spec — the stepper math is authored against its scaling block
// (base 2 PM, +1 PM per step, one step every 5 Bárbaro levels, p40).
const FURIA = getActivation('class.barbaro.furia')!

describe('stanceTotalPm — stepper cost math', () => {
  it('base cost with zero steps', () => {
    expect(stanceTotalPm(FURIA, 0)).toBe(2)
  })

  it('adds stepPm per step (2 base + 3×1)', () => {
    expect(stanceTotalPm(FURIA, 3)).toBe(5)
  })

  it('ignores steps for fixed-cost stances without scaling', () => {
    expect(stanceTotalPm(fakeSpec({ kind: 'stance', pmCost: 1 }), 5)).toBe(1)
  })
})

describe('stanceMaxSteps — clamp by OWNING class level', () => {
  it('Bárbaro 6 buys one step, Bárbaro 4 none', () => {
    expect(stanceMaxSteps(FURIA, [{ className: 'Bárbaro', level: 6 }])).toBe(1)
    expect(stanceMaxSteps(FURIA, [{ className: 'Bárbaro', level: 4 }])).toBe(0)
  })

  it('multiclass counts only the owning class (p40)', () => {
    expect(
      stanceMaxSteps(FURIA, [
        { className: 'Guerreiro', level: 10 },
        { className: 'Bárbaro', level: 5 },
      ]),
    ).toBe(1)
  })

  it('no Bárbaro entry → level 0 → zero steps', () => {
    expect(stanceMaxSteps(FURIA, [{ className: 'Guerreiro', level: 20 }])).toBe(0)
  })

  it('non-scaling specs never step', () => {
    expect(
      stanceMaxSteps(fakeSpec({ kind: 'stance' }), [
        { className: 'Bárbaro', level: 20 },
      ]),
    ).toBe(0)
  })
})

describe('stanceClassLevel', () => {
  it('matches the accented class name to the id segment', () => {
    expect(stanceClassLevel(FURIA, [{ className: 'Bárbaro', level: 7 }])).toBe(7)
  })

  it('is 0 for non-class spec ids', () => {
    expect(
      stanceClassLevel(fakeSpec({ id: 'race.humano.algo' }), [
        { className: 'Bárbaro', level: 7 },
      ]),
    ).toBe(0)
  })
})

describe('stanceActivationDecision', () => {
  it('refuses when the pool cannot pay base + steps', () => {
    expect(stanceActivationDecision(FURIA, 1, 2)).toEqual({
      ok: false,
      reason: 'PM insuficiente',
    })
  })

  it('allows spending the exact pool', () => {
    expect(stanceActivationDecision(FURIA, 1, 3)).toEqual({ ok: true })
    expect(stanceActivationDecision(FURIA, 0, 2)).toEqual({ ok: true })
  })

  it('refuses variable-cost stances without scaling', () => {
    expect(
      stanceActivationDecision(
        fakeSpec({ kind: 'stance', pmCost: 'variavel' }),
        0,
        99,
      ),
    ).toEqual({ ok: false, reason: 'custo variável' })
  })
})

describe('stanceFlagOf', () => {
  it('maps the Fúria spec back to its FLAG_ACTIVATIONS flag', () => {
    expect(stanceFlagOf(FURIA)).toBe('furia')
  })

  it('is undefined for specs outside FLAG_ACTIVATIONS', () => {
    expect(stanceFlagOf(fakeSpec())).toBeUndefined()
  })
})

// Fase 4: stance grants — only OWNED powers with a grant on the given flag.
function fakeOwner(classPowers: string[]): Character {
  return { classPowers: JSON.stringify(classPowers) } as unknown as Character
}

describe('grantPowersForFlag', () => {
  it('finds Alma de Bronze for furia by bare slug ownership', () => {
    const specs = grantPowersForFlag(fakeOwner(['alma-de-bronze']), 'furia')
    expect(specs.map((s) => s.id)).toEqual(['class.barbaro.alma-de-bronze'])
  })

  it('also matches full convention ids stored in classPowers', () => {
    const specs = grantPowersForFlag(
      fakeOwner(['class.barbaro.alma-de-bronze']),
      'furia',
    )
    expect(specs.map((s) => s.id)).toEqual(['class.barbaro.alma-de-bronze'])
  })

  it('is empty without ownership or for flags without grant powers', () => {
    expect(grantPowersForFlag(fakeOwner(['golpe-poderoso']), 'furia')).toEqual([])
    expect(grantPowersForFlag(fakeOwner(['alma-de-bronze']), 'inspiracao')).toEqual(
      [],
    )
  })
})

describe('resolveActivationSpec', () => {
  it('resolves a class power by its full convention id', () => {
    const spec = resolveActivationSpec('Golpe Poderoso', 'class.barbaro.golpe-poderoso')
    expect(spec?.id).toBe('class.barbaro.golpe-poderoso')
  })

  it('falls back to name lookup when the local id is off-convention', () => {
    const spec = resolveActivationSpec('Golpe Poderoso', 'golpe-poderoso-local-id')
    expect(spec?.name).toBe('Golpe Poderoso')
  })

  it('returns undefined for abilities without an activation entry', () => {
    expect(resolveActivationSpec('Não Existe No Registro')).toBeUndefined()
  })
})
