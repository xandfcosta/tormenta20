import { describe, expect, it } from 'vitest'
import { getClassPower } from '../catalog'
import { CULTO_PALADINO_DO_BEM } from '../deuses'

/**
 * Arma Sagrada (PDF p82) — pré-requisito: "devoto de uma divindade (exceto
 * Lena e Marah)". Paladino do Bem isn't devoto of a divindade, so must NOT
 * qualify either.
 *
 * Tests check the prerequisite shape, since the actual evaluator lives in
 * frontend/derived.ts. We assert the wire-level forbidden list so any
 * future evaluator inherits the rule.
 */
describe('Paladino — Arma Sagrada prerequisite', () => {
  const power = getClassPower('class.paladino.arma-sagrada')

  it('exists in the catalog', () => {
    expect(power).toBeDefined()
  })

  it('has a classChoice prereq on Paladino devoto', () => {
    const prereq = power!.prerequisites?.find(
      (p) => p.kind === 'classChoice' && p.field === 'devoto',
    )
    expect(prereq).toBeDefined()
    if (prereq?.kind !== 'classChoice') throw new Error('expected classChoice')
    expect(prereq.class).toBe('Paladino')
  })

  it('forbids Lena and Marah explicitly (book exception)', () => {
    const prereq = power!.prerequisites?.find(
      (p) => p.kind === 'classChoice' && p.field === 'devoto',
    )
    if (prereq?.kind !== 'classChoice') throw new Error('expected classChoice')
    expect(prereq.forbidden).toContain('lena')
    expect(prereq.forbidden).toContain('marah')
  })

  it('forbids Paladino do Bem (must be devoto of a real divindade)', () => {
    const prereq = power!.prerequisites?.find(
      (p) => p.kind === 'classChoice' && p.field === 'devoto',
    )
    if (prereq?.kind !== 'classChoice') throw new Error('expected classChoice')
    expect(prereq.forbidden).toContain(CULTO_PALADINO_DO_BEM)
  })
})
