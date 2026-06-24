import { describe, expect, it } from 'vitest'
import {
  CAMINHOS,
  CULTO_PALADINO_DO_BEM,
  CULTO_PANTEAO,
  DEUSES,
  DEUS_BY_ID,
  caminhoSlotFor,
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

  it('returns null for unknown class names (defensive default)', () => {
    expect(devotoOptionsFor('Hexer')).toBeNull()
    expect(devotoOptionsFor('')).toBeNull()
  })

  it('Clérigo Panteão option carries the non-divindade sentinel id', () => {
    // CULTO_PANTEAO is the sentinel persisted in ClassChoiceBlob.devoto when
    // the player picks "cultuar o Panteão" (p57) — Arma Sagrada and other
    // prereqs that require a real divindade must reject this id.
    const panteao = devotoOptionsFor('Clérigo')!.find(
      (d) => d.id === CULTO_PANTEAO,
    )
    expect(panteao).toBeDefined()
    expect(panteao!.major).toBe(false)
    expect(panteao!.paladinoEligible).toBe(false)
    expect(panteao!.druidaEligible).toBe(false)
  })

  it('Paladino paladino-do-bem option carries the non-divindade sentinel id', () => {
    const pdb = devotoOptionsFor('Paladino')!.find(
      (d) => d.id === CULTO_PALADINO_DO_BEM,
    )
    expect(pdb).toBeDefined()
    expect(pdb!.major).toBe(false)
    expect(pdb!.paladinoEligible).toBe(false)
  })
})

describe('DEUS_BY_ID', () => {
  it('indexes every deus in the catalog by id', () => {
    for (const d of DEUSES) {
      expect(DEUS_BY_ID[d.id]).toBe(d)
    }
  })

  it('does not include the non-divindade sentinels', () => {
    // CULTO_PANTEAO / CULTO_PALADINO_DO_BEM are surfaced only by
    // devotoOptionsFor — they must not be treated as real divindades by
    // prereq checks that hit DEUS_BY_ID.
    expect(DEUS_BY_ID[CULTO_PANTEAO]).toBeUndefined()
    expect(DEUS_BY_ID[CULTO_PALADINO_DO_BEM]).toBeUndefined()
  })

  it('returns undefined for unknown ids (lookup is safe to use directly)', () => {
    expect(DEUS_BY_ID['not-a-god']).toBeUndefined()
  })
})

describe('caminhoSlotFor', () => {
  it('Arcanista unlocks at L1 with 3 caminhos (bruxo, feiticeiro, mago)', () => {
    // Arcanista picks caminho at character creation — minLevel must be 1
    // so the picker shows the slot immediately. PDF p36 (Caminho do
    // Arcanista) — chosen at L1.
    const slot = caminhoSlotFor('Arcanista')
    expect(slot).not.toBeNull()
    expect(slot!.minLevel).toBe(1)
    expect(slot!.options.map((o) => o.id).sort()).toEqual([
      'bruxo',
      'feiticeiro',
      'mago',
    ])
  })

  it('Paladino unlocks at L5 with 2 caminhos (Égide / Montaria Sagrada)', () => {
    const slot = caminhoSlotFor('Paladino')
    expect(slot).not.toBeNull()
    expect(slot!.minLevel).toBe(5)
    expect(slot!.options.map((o) => o.id).sort()).toEqual([
      'egide-sagrada',
      'montaria-sagrada',
    ])
  })

  it('Cavaleiro unlocks at L5 with 2 caminhos (Bastião / Montaria)', () => {
    const slot = caminhoSlotFor('Cavaleiro')
    expect(slot).not.toBeNull()
    expect(slot!.minLevel).toBe(5)
    expect(slot!.options.map((o) => o.id).sort()).toEqual([
      'bastiao',
      'montaria',
    ])
  })

  it('returns null for classes without a caminho slot', () => {
    expect(caminhoSlotFor('Guerreiro')).toBeNull()
    expect(caminhoSlotFor('Bárbaro')).toBeNull()
    expect(caminhoSlotFor('Clérigo')).toBeNull()
    expect(caminhoSlotFor('Druida')).toBeNull()
  })

  it('returns null for unknown class names', () => {
    expect(caminhoSlotFor('Hexer')).toBeNull()
    expect(caminhoSlotFor('')).toBeNull()
  })

  it('every CAMINHOS entry has non-empty id and name', () => {
    // Picker uses id as the React key and name as the visible label —
    // catches an empty-string slip on future catalog edits.
    for (const [className, options] of Object.entries(CAMINHOS)) {
      for (const opt of options) {
        expect(opt.id, `${className}: empty id`).toBeTruthy()
        expect(opt.name, `${className}: empty name`).toBeTruthy()
      }
    }
  })

  it('every caminho id is unique within its class (picker key invariant)', () => {
    for (const [className, options] of Object.entries(CAMINHOS)) {
      const ids = options.map((o) => o.id)
      expect(new Set(ids).size, `${className} has duplicate caminho id`).toBe(
        ids.length,
      )
    }
  })
})
