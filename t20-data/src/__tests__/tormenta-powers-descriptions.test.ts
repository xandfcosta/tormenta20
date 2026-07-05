import { describe, expect, it } from 'vitest'
import {
  TORMENTA_POWERS,
  TORMENTA_POWER_IDS,
} from '../tormenta'

/**
 * Cap 2 residual audit 2026-07-05 — os 22 poderes da Tormenta já
 * existiam em `tormenta.ts` como skeleton (id/name/prereqs) mas sem
 * `description`. Este teste pin o novo campo em toda a lista.
 */

describe('TORMENTA_POWERS — descriptions (Cap 2 p135-137)', () => {
  it('todos os 22 poderes têm description não-vazio', () => {
    for (const id of TORMENTA_POWER_IDS) {
      const power = TORMENTA_POWERS[id]
      expect(
        power.description.length,
        `${id} sem descrição`,
      ).toBeGreaterThan(0)
    }
  })

  it('todos os 22 poderes têm bookPage entre 135 e 137', () => {
    for (const id of TORMENTA_POWER_IDS) {
      const page = TORMENTA_POWERS[id].bookPage
      expect(page, `${id} bookPage ${page}`).toBeGreaterThanOrEqual(135)
      expect(page).toBeLessThanOrEqual(137)
    }
  })

  it('Anatomia Insana pinning (25% chance ignorar dano crítico/furtivo)', () => {
    const p = TORMENTA_POWERS['anatomia-insana']
    expect(p.description).toMatch(/25%/)
    expect(p.description).toMatch(/crítico|ataque furtivo/i)
  })

  it('Dentes Afiados pinning (arma natural mordida 1d4 corte)', () => {
    const p = TORMENTA_POWERS['dentes-afiados']
    expect(p.description).toMatch(/mordida/)
    expect(p.description).toMatch(/1d4/)
  })

  it('Larva Explosiva depende de Dentes Afiados', () => {
    expect(TORMENTA_POWERS['larva-explosiva'].requiresPower).toBe(
      'dentes-afiados',
    )
  })

  it('Legião Aberrante depende de Anatomia Insana + 3 outros', () => {
    const p = TORMENTA_POWERS['legiao-aberrante']
    expect(p.requiresPower).toBe('anatomia-insana')
    expect(p.requiresOtherPowers).toBe(3)
  })

  it('Asas Insetoides + Desprezar a Realidade + Membros Extras exigem 4 outros', () => {
    for (const id of ['asas-insetoides', 'desprezar-a-realidade', 'membros-extras'] as const) {
      expect(
        TORMENTA_POWERS[id].requiresOtherPowers,
        id,
      ).toBe(4)
    }
  })
})
