import { describe, expect, it } from 'vitest'
import {
  CULTO_PALADINO_DO_BEM,
  CULTO_PANTEAO,
  DEUSES,
  devotoOptionsFor,
} from '../deuses'

/**
 * Behavioral checks against the PDF rules — not implementation details.
 *
 * References:
 *  - Tabela 1-20 (p97): full list of 20 deuses maiores.
 *  - Clérigo Devoto Fiel (p57): any deus maior OR cultuar o Panteão.
 *  - Paladino Abençoado (p82): whitelist of 8 deuses OR "paladino do bem".
 *  - Druida Devoto Fiel (p61): Allihanna, Megalokk ou Oceano.
 */
describe('DEUSES catalog', () => {
  it('contains all 20 deuses maiores from Tabela 1-20', () => {
    const expected = [
      'aharadak',
      'allihanna',
      'arsenal',
      'azgher',
      'hyninn',
      'kallyadranoch',
      'khalmyr',
      'lena',
      'lin-wu',
      'marah',
      'megalokk',
      'nimb',
      'oceano',
      'sszzaas',
      'tanna-toh',
      'tenebra',
      'thwor',
      'thyatis',
      'valkaria',
      'wynna',
    ]
    const got = DEUSES.filter((d) => d.major)
      .map((d) => d.id)
      .sort()
    expect(got).toEqual(expected.sort())
  })

  it('includes Aharadak as deus maior (Deus da Tormenta, ascendido)', () => {
    const aharadak = DEUSES.find((d) => d.id === 'aharadak')
    expect(aharadak).toBeDefined()
    expect(aharadak?.major).toBe(true)
  })

  it('does NOT mark Aharadak as paladino-eligible', () => {
    // PDF p82 Paladino whitelist excludes Aharadak.
    const aharadak = DEUSES.find((d) => d.id === 'aharadak')
    expect(aharadak?.paladinoEligible).toBe(false)
  })

  it('does NOT mark Aharadak as druida-eligible', () => {
    // PDF p61 Druida list: Allihanna, Megalokk, Oceano only.
    const aharadak = DEUSES.find((d) => d.id === 'aharadak')
    expect(aharadak?.druidaEligible).toBe(false)
  })
})

describe('devotoOptionsFor (class-level picker)', () => {
  it("offers all 20 deuses maiores + 'Panteão' option for Clérigo", () => {
    const opts = devotoOptionsFor('Clérigo')
    expect(opts).not.toBeNull()
    const ids = opts!.map((o) => o.id)
    expect(ids).toContain('aharadak')
    expect(ids).toContain(CULTO_PANTEAO)
    // 20 deuses maiores + Panteão.
    expect(opts!.length).toBe(21)
  })

  it('offers the 8 paladino deuses + paladino-do-bem option for Paladino', () => {
    const opts = devotoOptionsFor('Paladino')
    expect(opts).not.toBeNull()
    const ids = opts!.map((o) => o.id).sort()
    expect(ids).toEqual(
      [
        'azgher',
        'khalmyr',
        'lena',
        'lin-wu',
        'marah',
        'tanna-toh',
        'thyatis',
        'valkaria',
        CULTO_PALADINO_DO_BEM,
      ].sort(),
    )
  })

  it('offers exactly Allihanna, Megalokk, Oceano for Druida', () => {
    const opts = devotoOptionsFor('Druida')
    expect(opts).not.toBeNull()
    const ids = opts!.map((o) => o.id).sort()
    expect(ids).toEqual(['allihanna', 'megalokk', 'oceano'])
  })

  it('returns null for classes without devoto slot', () => {
    expect(devotoOptionsFor('Bárbaro')).toBeNull()
    expect(devotoOptionsFor('Guerreiro')).toBeNull()
    expect(devotoOptionsFor('Arcanista')).toBeNull()
  })
})
