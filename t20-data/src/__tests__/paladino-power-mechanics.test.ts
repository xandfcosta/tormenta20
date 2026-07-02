import { describe, expect, it } from 'vitest'
import {
  PALADINO_ELECTIVES,
  activePaladinoPowers,
  auraPowers,
  julgamentoPowers,
  paladinoCaminhoPowers,
  paladinoElectives,
  paladinoPowerById,
  virtudePowers,
} from '../paladino-power-mechanics'

/**
 * PDF Cap 1 Paladino p82-85. Pinned:
 *  - 25 eletivos
 *  - 9 Julgamentos Divinos (todos livre + PM 1-5)
 *  - 5 Virtudes Paladinescas (4 passivas + Humildade completa)
 *  - 5 Auras (isAura=true)
 *  - 2 Caminhos: Égide Sagrada + Montaria Sagrada
 */

describe('PALADINO_ELECTIVES — shape', () => {
  it('25 eletivos total', () => {
    expect(PALADINO_ELECTIVES.length).toBe(25)
  })

  it('frozen', () => {
    expect(Object.isFrozen(PALADINO_ELECTIVES)).toBe(true)
  })

  it('bookPage entre 82 e 85', () => {
    for (const p of PALADINO_ELECTIVES) {
      expect(p.bookPage).toBeGreaterThanOrEqual(82)
      expect(p.bookPage).toBeLessThanOrEqual(85)
    }
  })

  it('IDs únicos', () => {
    const ids = PALADINO_ELECTIVES.map((p) => p.id)
    expect(new Set(ids).size).toBe(ids.length)
  })
})

describe('julgamentoPowers', () => {
  it('exatamente 9 Julgamentos', () => {
    expect(julgamentoPowers().length).toBe(9)
  })

  it('todos action livre', () => {
    for (const p of julgamentoPowers()) {
      expect(p.action).toBe('livre')
    }
  })

  it('nomes começam com "Julgamento Divino:"', () => {
    for (const p of julgamentoPowers()) {
      expect(p.name.startsWith('Julgamento Divino:')).toBe(true)
    }
  })

  it('Libertação tem custo maior (5 PM)', () => {
    const p = paladinoPowerById('julgamento-libertacao')!
    expect(p.pmCost).toBe(5)
  })

  it('Autoridade + Zelo custam 1 PM', () => {
    expect(paladinoPowerById('julgamento-autoridade')!.pmCost).toBe(1)
    expect(paladinoPowerById('julgamento-zelo')!.pmCost).toBe(1)
  })
})

describe('virtudePowers', () => {
  it('exatamente 5 Virtudes', () => {
    expect(virtudePowers().length).toBe(5)
  })

  it('nomes começam com "Virtude Paladinesca:"', () => {
    for (const p of virtudePowers()) {
      expect(p.name.startsWith('Virtude Paladinesca:')).toBe(true)
    }
  })

  it('Humildade: completa cena', () => {
    const p = paladinoPowerById('virtude-humildade')!
    expect(p.action).toBe('completa')
    expect(p.uses).toBe('cena')
  })

  it('outras 4 virtudes são passivas', () => {
    const nonHumildade = virtudePowers().filter(
      (p) => p.id !== 'virtude-humildade',
    )
    expect(nonHumildade.length).toBe(4)
    for (const p of nonHumildade) {
      expect(p.action).toBe('passivo')
    }
  })
})

describe('auraPowers', () => {
  it('exatamente 5 Auras', () => {
    expect(auraPowers().length).toBe(5)
  })

  it('IDs corretos', () => {
    const ids = auraPowers().map((p) => p.id).sort()
    expect(ids).toEqual([
      'aura-antimagia',
      'aura-ardente',
      'aura-de-cura',
      'aura-de-invencibilidade',
      'aura-poderosa',
    ])
  })

  it('todas passivas (dependem do poder auto Aura)', () => {
    for (const p of auraPowers()) {
      expect(p.action).toBe('passivo')
    }
  })
})

describe('paladinoCaminhoPowers', () => {
  it('exatamente 2 Caminhos', () => {
    expect(paladinoCaminhoPowers().length).toBe(2)
  })

  it('Égide Sagrada: movimento 2 PM cena', () => {
    const p = paladinoPowerById('caminho-egide-sagrada')!
    expect(p.action).toBe('movimento')
    expect(p.pmCost).toBe(2)
    expect(p.uses).toBe('cena')
  })

  it('Montaria Sagrada: movimento 2 PM', () => {
    const p = paladinoPowerById('caminho-montaria-sagrada')!
    expect(p.action).toBe('movimento')
    expect(p.pmCost).toBe(2)
  })
})

describe('PALADINO_ELECTIVES — Orar', () => {
  it('Orar: varia pmCost variavel', () => {
    const p = paladinoPowerById('orar')!
    expect(p.action).toBe('varia')
    expect(p.pmCost).toBe('variavel')
  })
})

describe('paladinoPowerById', () => {
  it('miss retorna undefined', () => {
    expect(paladinoPowerById('inexistente')).toBeUndefined()
  })
})

describe('paladinoElectives', () => {
  it('retorna todos', () => {
    expect(paladinoElectives()).toBe(PALADINO_ELECTIVES)
  })
})

describe('activePaladinoPowers', () => {
  it('exclui passivos', () => {
    for (const p of activePaladinoPowers()) {
      expect(p.action).not.toBe('passivo')
    }
  })

  it('inclui 9 Julgamentos + Orar + Humildade + 2 Caminhos = 13', () => {
    expect(activePaladinoPowers().length).toBe(13)
  })
})
