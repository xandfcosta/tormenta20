import { describe, expect, it } from 'vitest'
import {
  CLERIGO_ELECTIVES,
  activeClerigoPowers,
  clerigoElectives,
  clerigoPowerById,
  missaPowers,
} from '../clerigo-power-mechanics'

/**
 * PDF Cap 1 Clérigo p56-58. Pinned:
 *  - 18 eletivos
 *  - 5 Missas (isMissa=true, all varia + 1 PM)
 *  - Abençoar Arma: movimento 3 PM
 *  - Canalizar Energia: padrao pmCost variavel
 *  - Expulsar Mortos-Vivos: padrao 3 PM
 *  - Símbolo Sagrado Energizado: movimento 1 PM cena
 *  - Liturgia Mágica: movimento 0 PM
 */

describe('CLERIGO_ELECTIVES — shape', () => {
  it('18 eletivos total', () => {
    expect(CLERIGO_ELECTIVES.length).toBe(18)
  })

  it('frozen', () => {
    expect(Object.isFrozen(CLERIGO_ELECTIVES)).toBe(true)
  })

  it('bookPage entre 57 e 58', () => {
    for (const p of CLERIGO_ELECTIVES) {
      expect(p.bookPage).toBeGreaterThanOrEqual(57)
      expect(p.bookPage).toBeLessThanOrEqual(58)
    }
  })

  it('IDs únicos', () => {
    const ids = CLERIGO_ELECTIVES.map((p) => p.id)
    expect(new Set(ids).size).toBe(ids.length)
  })
})

describe('CLERIGO_ELECTIVES — pinned entries', () => {
  it('Abençoar Arma: movimento, 3 PM', () => {
    const p = clerigoPowerById('abencoar-arma')!
    expect(p.action).toBe('movimento')
    expect(p.pmCost).toBe(3)
  })

  it('Canalizar Energia: padrao, pmCost variavel', () => {
    const p = clerigoPowerById('canalizar-energia-positiva-negativa')!
    expect(p.action).toBe('padrao')
    expect(p.pmCost).toBe('variavel')
  })

  it('Expulsar/Comandar Mortos-Vivos: padrao, 3 PM', () => {
    const p = clerigoPowerById('expulsar-comandar-mortos-vivos')!
    expect(p.action).toBe('padrao')
    expect(p.pmCost).toBe(3)
  })

  it('Liturgia Mágica: movimento, 0 PM', () => {
    const p = clerigoPowerById('liturgia-magica')!
    expect(p.action).toBe('movimento')
    expect(p.pmCost).toBe(0)
  })

  it('Símbolo Sagrado Energizado: movimento, 1 PM, cena', () => {
    const p = clerigoPowerById('simbolo-sagrado-energizado')!
    expect(p.action).toBe('movimento')
    expect(p.pmCost).toBe(1)
    expect(p.uses).toBe('cena')
  })
})

describe('missaPowers', () => {
  it('exatamente 5 Missas', () => {
    expect(missaPowers().length).toBe(5)
  })

  it('todas começam com "Missa:" no nome', () => {
    for (const p of missaPowers()) {
      expect(p.name.startsWith('Missa:')).toBe(true)
    }
  })

  it('IDs corretos', () => {
    const ids = missaPowers().map((p) => p.id).sort()
    expect(ids).toEqual([
      'missa-bencao-da-vida',
      'missa-chamado-as-armas',
      'missa-elevacao-do-espirito',
      'missa-escudo-divino',
      'missa-superar-as-limitacoes',
    ])
  })

  it('todas varia + 1 PM', () => {
    for (const p of missaPowers()) {
      expect(p.action).toBe('varia')
      expect(p.pmCost).toBe(1)
    }
  })
})

describe('clerigoPowerById', () => {
  it('miss retorna undefined', () => {
    expect(clerigoPowerById('inexistente')).toBeUndefined()
  })
})

describe('clerigoElectives', () => {
  it('retorna todos', () => {
    expect(clerigoElectives()).toBe(CLERIGO_ELECTIVES)
  })
})

describe('activeClerigoPowers', () => {
  it('exclui passivos', () => {
    for (const p of activeClerigoPowers()) {
      expect(p.action).not.toBe('passivo')
    }
  })

  it('inclui Abençoar Arma, Canalizar Energia, Expulsar, Liturgia, Símbolo, 5 Missas', () => {
    const ids = activeClerigoPowers().map((p) => p.id)
    expect(ids).toContain('abencoar-arma')
    expect(ids).toContain('canalizar-energia-positiva-negativa')
    expect(ids).toContain('expulsar-comandar-mortos-vivos')
    expect(ids).toContain('liturgia-magica')
    expect(ids).toContain('simbolo-sagrado-energizado')
    for (const missa of missaPowers()) {
      expect(ids).toContain(missa.id)
    }
  })
})
