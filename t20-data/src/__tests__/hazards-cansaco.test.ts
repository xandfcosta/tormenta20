import { describe, expect, it } from 'vitest'
import {
  CANSACO_CD_PROGRESSAO,
  CANSACO_FORT_CD_BASE,
  CANSACO_IMUNE_TIPOS,
  FOME_SEDE_RECOVERY_METHOD,
  FOME_SEDE_TOLERANCIA_DIAS,
  SONO_RECOVERY_METHOD,
  fomeSedeFortCd,
  fomeSedeStage,
  isImuneCansaco,
  sonoStage,
} from '../hazards-cansaco'

/**
 * PDF Cap 7 p318 (Fome/Sede + Sono) + p221 (Cansaço effect type).
 */

describe('constantes', () => {
  it('tolerância 1 dia', () => {
    expect(FOME_SEDE_TOLERANCIA_DIAS).toBe(1)
  })

  it('CD base 15', () => {
    expect(CANSACO_FORT_CD_BASE).toBe(15)
  })

  it('progressão +1 por teste', () => {
    expect(CANSACO_CD_PROGRESSAO).toBe(1)
  })

  it('recovery fome/sede = comida-bebida (Metabolismo)', () => {
    expect(FOME_SEDE_RECOVERY_METHOD).toBe('comida-bebida')
  })

  it('recovery sono = dormir-8h', () => {
    expect(SONO_RECOVERY_METHOD).toBe('dormir-8h')
  })

  it('imunes: construto + morto-vivo', () => {
    expect([...CANSACO_IMUNE_TIPOS]).toEqual(['construto', 'morto-vivo'])
  })

  it('CANSACO_IMUNE_TIPOS frozen', () => {
    expect(Object.isFrozen(CANSACO_IMUNE_TIPOS)).toBe(true)
  })
})

describe('fomeSedeFortCd — CD 15+n', () => {
  it('primeiro teste → 15', () => {
    expect(fomeSedeFortCd(0)).toBe(15)
  })

  it('segundo → 16', () => {
    expect(fomeSedeFortCd(1)).toBe(16)
  })

  it('quinto → 19', () => {
    expect(fomeSedeFortCd(4)).toBe(19)
  })

  it('throws se previousChecks negativo', () => {
    expect(() => fomeSedeFortCd(-1)).toThrow(/previousChecks/)
  })
})

describe('fomeSedeStage — ladder p318 (4 falhas → letal)', () => {
  it('0 falhas → ok', () => {
    expect(fomeSedeStage(0)).toBe('ok')
  })

  it('1 falha → fatigado', () => {
    expect(fomeSedeStage(1)).toBe('fatigado')
  })

  it('2 falhas → exausto', () => {
    expect(fomeSedeStage(2)).toBe('exausto')
  })

  it('3 falhas → inconsciente', () => {
    expect(fomeSedeStage(3)).toBe('inconsciente')
  })

  it('4 falhas → letal', () => {
    expect(fomeSedeStage(4)).toBe('letal')
  })

  it('5+ falhas continua letal', () => {
    expect(fomeSedeStage(5)).toBe('letal')
    expect(fomeSedeStage(99)).toBe('letal')
  })

  it('throws se consecutiveFailures negativo', () => {
    expect(() => fomeSedeStage(-1)).toThrow(/consecutiveFailures/)
  })
})

describe('sonoStage — ladder p318 (sem letal)', () => {
  it('0 → ok', () => {
    expect(sonoStage(0)).toBe('ok')
  })

  it('1 → fatigado', () => {
    expect(sonoStage(1)).toBe('fatigado')
  })

  it('2 → exausto', () => {
    expect(sonoStage(2)).toBe('exausto')
  })

  it('3+ → inconsciente (nunca letal)', () => {
    expect(sonoStage(3)).toBe('inconsciente')
    expect(sonoStage(10)).toBe('inconsciente')
  })

  it('throws se consecutiveFailures negativo', () => {
    expect(() => sonoStage(-1)).toThrow(/consecutiveFailures/)
  })
})

describe('isImuneCansaco', () => {
  it('construto → true', () => {
    expect(isImuneCansaco('construto')).toBe(true)
  })

  it('morto-vivo → true', () => {
    expect(isImuneCansaco('morto-vivo')).toBe(true)
  })

  it('humanoide → false', () => {
    expect(isImuneCansaco('humanoide')).toBe(false)
  })

  it('animal → false', () => {
    expect(isImuneCansaco('animal')).toBe(false)
  })
})
