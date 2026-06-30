import { describe, expect, it } from 'vitest'
import {
  DISEASES,
  DISEASE_CURE_SAVES_REQUIRED,
  POISONS,
  POISON_BASE_CRAFT_CD,
  advanceDisease,
  diseaseById_,
  diseasesByVector,
  poisonById_,
  poisonResistCd,
  poisonsByVector,
} from '../diseases-poisons'

/**
 * PDF Cap 7 (doenças p318) + Cap 3 (venenos p161). Pinned:
 *  - 8 doenças core (p318) e 10 venenos alquímicos (p161).
 *  - Vectors restritos a contato | inalacao | ingestao (NO ferimento).
 *  - Doença: Fortitude p/ progredir; 2 sucessos consecutivos curam.
 *  - Veneno: Fortitude única; CD = base 20 + Int + modifier (Beladona
 *    e Pó de Lich +5).
 *  - Doenças com marcador `*` no efeito = perda permanente.
 */

describe('DISEASES — shape & invariants', () => {
  it('catálogo tem exatamente 8 doenças (p318)', () => {
    expect(DISEASES.length).toBe(8)
  })

  it('todos ids únicos', () => {
    const ids = DISEASES.map((d) => d.id)
    expect(new Set(ids).size).toBe(ids.length)
  })

  it('vector restrito a contato|inalacao|ingestao', () => {
    for (const d of DISEASES) {
      expect(['contato', 'inalacao', 'ingestao']).toContain(d.vector)
    }
  })

  it('toda doença tem progression array (mesmo se 1 entrada)', () => {
    for (const d of DISEASES) {
      expect(Array.isArray(d.progression)).toBe(true)
      expect(d.progression.length).toBeGreaterThan(0)
    }
  })

  it('toda doença bookPage === 318', () => {
    for (const d of DISEASES) expect(d.bookPage).toBe(318)
  })

  it('catálogo é frozen', () => {
    expect(Object.isFrozen(DISEASES)).toBe(true)
  })
})

describe('DISEASES — pinned canonical entries', () => {
  it('Calafrio Diabólico: CD 25, contato, ladder fraca→morre', () => {
    const d = diseaseById_('calafrio-diabolico')!
    expect(d.initialSaveCd).toBe(25)
    expect(d.cureCd).toBe(25)
    expect(d.vector).toBe('contato')
    expect(d.progression).toEqual([
      'fraca',
      'debilitada',
      'inconsciente',
      'morre',
    ])
  })

  it('Maldição Pegajosa: ladder 1d12/2d12/4d12 PV', () => {
    const d = diseaseById_('maldicao-pegajosa')!
    expect(d.progression).toEqual([
      'perde 1d12 PV',
      'perde 2d12 PV',
      'perde 4d12 PV',
    ])
  })

  it('Moléstia Demoníaca: hasPermanentEffect (1 Con*)', () => {
    const d = diseaseById_('molestia-demoniaca')!
    expect(d.hasPermanentEffect).toBe(true)
    expect(d.progression).toContain('perde 1 de Constituição*')
  })

  it('Varíola: hasPermanentEffect (1 Car*)', () => {
    const d = diseaseById_('variola')!
    expect(d.hasPermanentEffect).toBe(true)
    expect(d.progression).toContain('perde 1 de Carisma*')
  })

  it('apenas 2 doenças têm hasPermanentEffect (Moléstia + Varíola)', () => {
    const perma = DISEASES.filter((d) => d.hasPermanentEffect).map((d) => d.id)
    expect(perma.sort()).toEqual(['molestia-demoniaca', 'variola'])
  })
})

describe('diseasesByVector', () => {
  it('inalacao: Febre do Riso, Febre Mental, Varíola', () => {
    const ids = diseasesByVector('inalacao').map((d) => d.id).sort()
    expect(ids).toEqual(['febre-do-riso', 'febre-mental', 'variola'])
  })

  it('ingestao: vazio (core T20 não tem doença ingerida)', () => {
    expect(diseasesByVector('ingestao').length).toBe(0)
  })

  it('contato: 5 entradas', () => {
    expect(diseasesByVector('contato').length).toBe(5)
  })
})

describe('POISONS — shape & invariants', () => {
  it('catálogo tem exatamente 10 venenos (p161)', () => {
    expect(POISONS.length).toBe(10)
  })

  it('todos ids únicos', () => {
    const ids = POISONS.map((p) => p.id)
    expect(new Set(ids).size).toBe(ids.length)
  })

  it('vector restrito a contato|inalacao|ingestao', () => {
    for (const p of POISONS) {
      expect(['contato', 'inalacao', 'ingestao']).toContain(p.vector)
    }
  })

  it('toda priceTibar positiva', () => {
    for (const p of POISONS) expect(p.priceTibar).toBeGreaterThan(0)
  })

  it('toda bookPage === 161', () => {
    for (const p of POISONS) expect(p.bookPage).toBe(161)
  })

  it('catálogo é frozen', () => {
    expect(Object.isFrozen(POISONS)).toBe(true)
  })
})

describe('POISONS — pinned canonical entries', () => {
  it('Beladona: ingestão, +5 CD modifier, T$ 1500', () => {
    const p = poisonById_('beladona')!
    expect(p.vector).toBe('ingestao')
    expect(p.saveCdModifier).toBe(5)
    expect(p.priceTibar).toBe(1500)
  })

  it('Pó de Lich: ingestão, +5 CD modifier, 4d12 PV/rodada, T$ 3000', () => {
    const p = poisonById_('po-de-lich')!
    expect(p.vector).toBe('ingestao')
    expect(p.saveCdModifier).toBe(5)
    expect(p.priceTibar).toBe(3000)
    expect(p.initialEffect).toMatch(/4d12 PV/)
  })

  it('Peçonha Comum: contato, T$ 15 (mais barato)', () => {
    const p = poisonById_('peconha-comum')!
    expect(p.vector).toBe('contato')
    expect(p.priceTibar).toBe(15)
  })

  it('Riso de Nimb: inalação, fica confusa', () => {
    const p = poisonById_('riso-de-nimb')!
    expect(p.vector).toBe('inalacao')
    expect(p.initialEffect).toMatch(/confusa/)
  })

  it('apenas Beladona e Pó de Lich têm saveCdModifier !== 0', () => {
    const modded = POISONS.filter((p) => p.saveCdModifier !== 0).map((p) => p.id)
    expect(modded.sort()).toEqual(['beladona', 'po-de-lich'])
  })
})

describe('poisonsByVector', () => {
  it('contato: 4 entradas (Essência Sombra + 3 peçonhas)', () => {
    expect(poisonsByVector('contato').length).toBe(4)
  })

  it('inalacao: 3 entradas (Bruma + Névoa + Riso de Nimb)', () => {
    expect(poisonsByVector('inalacao').length).toBe(3)
  })

  it('ingestao: 3 entradas (Beladona + Cicuta + Pó de Lich)', () => {
    expect(poisonsByVector('ingestao').length).toBe(3)
  })
})

describe('poisonResistCd', () => {
  it('Cicuta sem Int do aplicador: 20 (base)', () => {
    const p = poisonById_('cicuta')!
    expect(poisonResistCd(p, 0)).toBe(POISON_BASE_CRAFT_CD)
  })

  it('Beladona sem Int: 25 (base 20 + modifier 5)', () => {
    expect(poisonResistCd(poisonById_('beladona')!, 0)).toBe(25)
  })

  it('Pó de Lich com Int 4: 29 (20 + 4 + 5)', () => {
    expect(poisonResistCd(poisonById_('po-de-lich')!, 4)).toBe(29)
  })
})

describe('DISEASE_CURE_SAVES_REQUIRED', () => {
  it('exige 2 sucessos consecutivos (p318)', () => {
    expect(DISEASE_CURE_SAVES_REQUIRED).toBe(2)
  })
})

describe('advanceDisease — modelo de progressão', () => {
  const disease = diseaseById_('calafrio-diabolico')!

  it('falha avança step (-1 → 0)', () => {
    const out = advanceDisease(
      disease,
      { step: -1, consecutiveSuccesses: 0 },
      false,
    )
    expect(out.step).toBe(0) // 'fraca'
    expect(out.consecutiveSuccesses).toBe(0)
    expect(out.cured).toBe(false)
  })

  it('falha em step máx mantém step (clamp)', () => {
    const out = advanceDisease(
      disease,
      { step: 3, consecutiveSuccesses: 0 },
      false,
    )
    expect(out.step).toBe(3) // 'morre' continua no topo
  })

  it('1 sucesso não cura (precisa de 2 consecutivos)', () => {
    const out = advanceDisease(
      disease,
      { step: 0, consecutiveSuccesses: 0 },
      true,
    )
    expect(out.consecutiveSuccesses).toBe(1)
    expect(out.cured).toBe(false)
  })

  it('2 sucessos consecutivos curam', () => {
    const step1 = advanceDisease(
      disease,
      { step: 1, consecutiveSuccesses: 0 },
      true,
    )
    const step2 = advanceDisease(disease, step1, true)
    expect(step2.cured).toBe(true)
  })

  it('falha após 1 sucesso reseta o contador (precisa 2 NOVOS)', () => {
    const s1 = advanceDisease(
      disease,
      { step: 0, consecutiveSuccesses: 0 },
      true,
    )
    expect(s1.consecutiveSuccesses).toBe(1)
    const f1 = advanceDisease(disease, s1, false)
    expect(f1.consecutiveSuccesses).toBe(0)
  })
})

describe('vector taxonomy — NO ferimento (T20 specific)', () => {
  it('union restrita a 3 valores apenas', () => {
    // Compila? — TS apenas garante union. Aqui validamos catálogos.
    const seen = new Set<string>()
    for (const d of DISEASES) seen.add(d.vector)
    for (const p of POISONS) seen.add(p.vector)
    expect([...seen].sort()).toEqual(['contato', 'inalacao', 'ingestao'])
  })
})
