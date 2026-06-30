import { describe, expect, it } from 'vitest'
import { DEUSES, type Deus } from '../abilities/deuses'

/**
 * Enrichment tests for the 20 deuses maiores — PDF book p96-105.
 *
 * Pinned for each deus maior:
 *  - portfolio (string)
 *  - energia ('positiva' | 'negativa' | 'qualquer')
 *  - simbolo (string)
 *  - armaPreferida (string OR null when explicitly proibida)
 *  - poderesConcedidos (4 named powers)
 *  - devotos (race+class whitelist)
 *  - bookPage
 *
 * Existing plumbing (major / paladinoEligible / druidaEligible) is
 * already covered by `abilities/__tests__/deuses.test.ts`.
 */

const MAIORES: Deus[] = DEUSES.filter((d) => d.major)

describe('DEUSES enrichment — every deus maior has full fields', () => {
  it('all 20 deuses maiores present', () => {
    expect(MAIORES.length).toBe(20)
  })

  it.each(MAIORES.map((d) => [d.name, d]))(
    '%s has portfolio + energia + simbolo + 4 poderes + devotos + bookPage',
    (_name, d) => {
      const deus = d as Deus
      expect(deus.portfolio).toBeTruthy()
      expect(deus.energia).toMatch(/^(positiva|negativa|qualquer)$/)
      expect(deus.simbolo).toBeTruthy()
      expect(deus.poderesConcedidos?.length).toBe(4)
      expect(deus.devotos?.length).toBeGreaterThan(0)
      expect(deus.bookPage).toBeGreaterThanOrEqual(96)
      expect(deus.bookPage).toBeLessThanOrEqual(105)
    },
  )
})

describe('DEUSES enrichment — energia distribution', () => {
  const counts = (e: 'positiva' | 'negativa' | 'qualquer') =>
    MAIORES.filter((d) => d.energia === e).length

  it('has at least one deus per energia category', () => {
    expect(counts('positiva')).toBeGreaterThan(0)
    expect(counts('negativa')).toBeGreaterThan(0)
    expect(counts('qualquer')).toBeGreaterThan(0)
  })

  it('counts sum to 20', () => {
    expect(counts('positiva') + counts('negativa') + counts('qualquer')).toBe(20)
  })
})

describe('DEUSES enrichment — armaPreferida null only for Lena/Marah', () => {
  it('Lena has armaPreferida === null (proíbe Arma Espiritual)', () => {
    expect(DEUSES.find((d) => d.id === 'lena')?.armaPreferida).toBeNull()
  })

  it('Marah has armaPreferida === null (proíbe Arma Espiritual)', () => {
    expect(DEUSES.find((d) => d.id === 'marah')?.armaPreferida).toBeNull()
  })

  it('Nimb uses sentinel "todas" (arma preferida caótica)', () => {
    expect(DEUSES.find((d) => d.id === 'nimb')?.armaPreferida).toBe('todas')
  })

  it('all other deuses maiores have a concrete arma string', () => {
    const others = MAIORES.filter(
      (d) => d.id !== 'lena' && d.id !== 'marah' && d.id !== 'nimb',
    )
    for (const d of others) {
      expect(d.armaPreferida).toBeTruthy()
      expect(typeof d.armaPreferida).toBe('string')
    }
  })
})

describe('DEUSES enrichment — pinned canonical entries (PDF integrity)', () => {
  it('Khalmyr: justiça, positiva, Espada longa, 4 poderes Justiceiros', () => {
    const k = DEUSES.find((d) => d.id === 'khalmyr')!
    expect(k.portfolio).toMatch(/Justiça/)
    expect(k.energia).toBe('positiva')
    expect(k.armaPreferida).toBe('Espada longa')
    expect(k.poderesConcedidos).toEqual([
      'Coragem Total',
      'Dom da Verdade',
      'Espada Justiceira',
      'Reparar Injustiça',
    ])
    expect(k.bookPage).toBe(99)
  })

  it('Valkaria: ambição/humanidade/liberdade, positiva, Mangual, p105', () => {
    const v = DEUSES.find((d) => d.id === 'valkaria')!
    expect(v.portfolio).toMatch(/Ambição/)
    expect(v.energia).toBe('positiva')
    expect(v.armaPreferida).toBe('Mangual')
    expect(v.poderesConcedidos).toContain('Liberdade Divina')
    expect(v.bookPage).toBe(105)
  })

  it('Tenebra: noite/escuridão/mortos-vivos, negativa, Adaga, p104', () => {
    const t = DEUSES.find((d) => d.id === 'tenebra')!
    expect(t.energia).toBe('negativa')
    expect(t.armaPreferida).toBe('Adaga')
    expect(t.poderesConcedidos).toContain('Zumbificar')
    expect(t.bookPage).toBe(104)
  })

  it('Allihanna: druidaEligible + natureza + positiva + Bordão', () => {
    const a = DEUSES.find((d) => d.id === 'allihanna')!
    expect(a.druidaEligible).toBe(true)
    expect(a.portfolio).toMatch(/Natureza/)
    expect(a.armaPreferida).toBe('Bordão')
  })

  it('Aharadak: Tormenta portfolio, negativa, p96', () => {
    const ah = DEUSES.find((d) => d.id === 'aharadak')!
    expect(ah.portfolio).toMatch(/Tormenta/)
    expect(ah.energia).toBe('negativa')
    expect(ah.bookPage).toBe(96)
  })
})

describe('DEUSES enrichment — devotos lists are non-empty', () => {
  it.each(MAIORES.map((d) => [d.name, d.id]))(
    '%s has at least one devoto entry',
    (_name, id) => {
      const deus = DEUSES.find((d) => d.id === id)!
      expect(deus.devotos?.length).toBeGreaterThan(0)
    },
  )
})

describe('DEUSES enrichment — sentinels (Panteão / Paladino do Bem) do NOT have deus fields', () => {
  it('Panteão / Paladino do Bem sentinels are produced separately (not in DEUSES)', () => {
    const ids = DEUSES.map((d) => d.id)
    expect(ids).not.toContain('panteao')
    expect(ids).not.toContain('paladino-do-bem')
  })
})
