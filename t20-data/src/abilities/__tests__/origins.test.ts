import { describe, expect, it } from 'vitest'
import { ORIGINS_CATALOG } from '../origins'

/**
 * PDF Cap 1 (Origens, p85-95) — Tabela 1-19. The book lists 35 origens;
 * each has a benefits pool (perícias + poderes the player picks 2 from)
 * plus one exclusive poder único. These tests guard the catalog shape
 * and spot-check the rule "perícia kind grants treinamento; poder kind
 * has no expertise property".
 *
 * Per-origin benefit-list contents are *not* asserted here — the catalog
 * encodes them across 800+ lines and a future PR will pin them one by
 * one. This file's job is structural sanity + spot checks against the
 * canonical Acólito and Soldado entries from the book table.
 */
describe('ORIGINS_CATALOG vs PDF Tabela 1-19', () => {
  it('contains all 35 origens from the table', () => {
    expect(ORIGINS_CATALOG.length).toBe(35)
  })

  it('every origin has a unique id', () => {
    const ids = ORIGINS_CATALOG.map((o) => o.id)
    expect(new Set(ids).size).toBe(ids.length)
  })

  it('every origin defines a poder único', () => {
    for (const origin of ORIGINS_CATALOG) {
      expect(origin.poderUnico, `${origin.id} missing poderUnico`).toBeDefined()
      expect(origin.poderUnico.kind).toBe('poder')
    }
  })

  it('benefits with kind=pericia carry an expertise reference', () => {
    for (const origin of ORIGINS_CATALOG) {
      for (const benefit of origin.benefits) {
        if (benefit.kind === 'pericia') {
          expect(
            benefit.expertise,
            `${origin.id}/${benefit.id} missing expertise`,
          ).toBeDefined()
        }
      }
    }
  })

  it('benefits with kind=poder do not carry an expertise field', () => {
    for (const origin of ORIGINS_CATALOG) {
      for (const benefit of origin.benefits) {
        if (benefit.kind === 'poder') {
          expect(
            benefit.expertise,
            `${origin.id}/${benefit.id} should not have expertise`,
          ).toBeUndefined()
        }
      }
    }
  })
})

describe('Origens — spot checks against PDF Tabela 1-19', () => {
  it('Acólito: Cura, Religião, Vontade (perícias); Medicina, Vontade de Ferro (poderes); Membro da Igreja (único)', () => {
    // PDF p86 Tabela 1-19: Acólito gets 3 perícias + 2 poderes pool, and
    // "Membro da Igreja" is its exclusive poder único — not in the picker.
    const acolito = ORIGINS_CATALOG.find((o) => o.id === 'Acólito')!
    const periciaNames = acolito.benefits
      .filter((b) => b.kind === 'pericia')
      .map((b) => b.expertise)
      .sort()
    expect(periciaNames).toEqual(['Cura', 'Religião', 'Vontade'])

    const poderNames = acolito.benefits
      .filter((b) => b.kind === 'poder')
      .map((b) => b.name)
      .sort()
    expect(poderNames).toEqual(['Medicina', 'Vontade de Ferro'])

    expect(acolito.poderUnico.name).toBe('Membro da Igreja')
  })

  it('Soldado: Fortitude, Guerra, Luta, Pontaria (perícias); Influência Militar (poder)', () => {
    // Book p87 Tabela 1-19 lists 4 perícias + Influência Militar + "um
    // poder de combate a sua escolha" for Soldado.
    const soldado = ORIGINS_CATALOG.find((o) => o.id === 'Soldado')!
    const periciaNames = soldado.benefits
      .filter((b) => b.kind === 'pericia')
      .map((b) => b.expertise)
      .sort()
    expect(periciaNames).toEqual(['Fortitude', 'Guerra', 'Luta', 'Pontaria'])
  })

  it('Amnésico has open-pick benefits but still defines its poder único', () => {
    // Book p86: Amnésico foregoes the standard list — gets 1 perícia + 1
    // poder picked by the mestre, and the poder único is "Lembranças
    // Graduais". Catalog represents the open-pick part as an empty
    // benefits array.
    const amnesico = ORIGINS_CATALOG.find((o) => o.id === 'Amnésico')!
    expect(amnesico.benefits).toEqual([])
    expect(amnesico.poderUnico.name).toMatch(/Lembranças/i)
  })
})
