import { describe, expect, it } from 'vitest'
import {
  COMPANHEIRO_ANIMAL_APRIMORADO_LEVEL,
  COMPANHEIRO_ANIMAL_BASE_TIPOS,
  COMPANHEIRO_ANIMAL_LENDARIO_LEVEL,
  COMPANHEIRO_ANIMAL_MAGICO_LEVEL,
  COMPANHEIRO_ANIMAL_MAGICO_TIPOS,
  COMPANHEIRO_ANIMAL_PREREQ_CARISMA,
  COMPANHEIRO_ANIMAL_PREREQ_SKILL,
  COMPANHEIRO_ANIMAL_REINVOKE_DAYS,
  COMPANHEIRO_ANIMAL_SAMPLES,
  COMPANHEIRO_ANIMAL_SAMPLE_DESCRIPTIONS,
  COMPANHEIRO_ANIMAL_STUN_ROUNDS_ON_DEATH,
  COMPANHEIRO_ANIMAL_VIGILANTE_EXCLUDED,
  baseTipoToParceiroType,
  isBaseTipoAllowed,
  requiresMagicoAugment,
} from '../companheiro-animal-catalog'

/**
 * PDF p61-62 (Druida — Companheiro Animal sidebar + augments).
 */

describe('base tipos — sidebar p62', () => {
  it('8 tipos base', () => {
    expect(COMPANHEIRO_ANIMAL_BASE_TIPOS.length).toBe(8)
  })

  it('inclui montaria como tipo base', () => {
    expect(COMPANHEIRO_ANIMAL_BASE_TIPOS).toContain('montaria')
  })

  it('exclui vigilante', () => {
    expect(COMPANHEIRO_ANIMAL_BASE_TIPOS).not.toContain('vigilante' as never)
    expect(COMPANHEIRO_ANIMAL_VIGILANTE_EXCLUDED).toBe(true)
  })

  it('exclui tipos arcanos do base', () => {
    expect(COMPANHEIRO_ANIMAL_BASE_TIPOS).not.toContain('adepto' as never)
    expect(COMPANHEIRO_ANIMAL_BASE_TIPOS).not.toContain('destruidor' as never)
    expect(COMPANHEIRO_ANIMAL_BASE_TIPOS).not.toContain('magivocador' as never)
    expect(COMPANHEIRO_ANIMAL_BASE_TIPOS).not.toContain('medico' as never)
  })

  it('frozen', () => {
    expect(Object.isFrozen(COMPANHEIRO_ANIMAL_BASE_TIPOS)).toBe(true)
  })
})

describe('mágico tipos — augment p62', () => {
  it('4 tipos arcanos', () => {
    expect(COMPANHEIRO_ANIMAL_MAGICO_TIPOS.length).toBe(4)
  })

  it('inclui adepto, destruidor, magivocador, medico', () => {
    expect([...COMPANHEIRO_ANIMAL_MAGICO_TIPOS].sort()).toEqual([
      'adepto',
      'destruidor',
      'magivocador',
      'medico',
    ])
  })
})

describe('amostras — sidebar p62', () => {
  it('ajudante: corvo/macaco/raposa/serpente', () => {
    expect(COMPANHEIRO_ANIMAL_SAMPLES.ajudante).toEqual([
      'corvo',
      'macaco',
      'raposa',
      'serpente',
    ])
  })

  it('assassino: lince/onça', () => {
    expect(COMPANHEIRO_ANIMAL_SAMPLES.assassino).toEqual(['lince', 'onça'])
  })

  it('atirador: águia/falcão', () => {
    expect(COMPANHEIRO_ANIMAL_SAMPLES.atirador).toEqual(['águia', 'falcão'])
  })

  it('fortao: crocodilo/javali/leão/lobo', () => {
    expect(COMPANHEIRO_ANIMAL_SAMPLES.fortao).toEqual([
      'crocodilo',
      'javali',
      'leão',
      'lobo',
    ])
  })

  it('guardiao: alce/cão/coruja/tartaruga/urso', () => {
    expect(COMPANHEIRO_ANIMAL_SAMPLES.guardiao).toEqual([
      'alce',
      'cão',
      'coruja',
      'tartaruga',
      'urso',
    ])
  })

  it('perseguidor: gambá/sabujo', () => {
    expect(COMPANHEIRO_ANIMAL_SAMPLES.perseguidor).toEqual(['gambá', 'sabujo'])
  })

  it('combatente + montaria não têm amostras no p62', () => {
    expect(COMPANHEIRO_ANIMAL_SAMPLES.combatente).toBeUndefined()
    expect(COMPANHEIRO_ANIMAL_SAMPLES.montaria).toBeUndefined()
  })

  it('descrições verbatim citam animais + qualificador', () => {
    expect(COMPANHEIRO_ANIMAL_SAMPLE_DESCRIPTIONS.perseguidor).toMatch(
      /farejador/,
    )
    expect(COMPANHEIRO_ANIMAL_SAMPLE_DESCRIPTIONS.guardiao).toMatch(
      /pesado ou atento/,
    )
  })
})

describe('regras de morte — p62', () => {
  it('atordoado 1 rodada', () => {
    expect(COMPANHEIRO_ANIMAL_STUN_ROUNDS_ON_DEATH).toBe(1)
  })

  it('reinvocação em 1 dia', () => {
    expect(COMPANHEIRO_ANIMAL_REINVOKE_DAYS).toBe(1)
  })
})

describe('augments — pré-requisitos de nível p62', () => {
  it('Aprimorado: 6º nível', () => {
    expect(COMPANHEIRO_ANIMAL_APRIMORADO_LEVEL).toBe(6)
  })

  it('Mágico: 8º nível', () => {
    expect(COMPANHEIRO_ANIMAL_MAGICO_LEVEL).toBe(8)
  })

  it('Lendário: 18º nível', () => {
    expect(COMPANHEIRO_ANIMAL_LENDARIO_LEVEL).toBe(18)
  })
})

describe('pré-requisitos do poder base p61-62', () => {
  it('Carisma 1', () => {
    expect(COMPANHEIRO_ANIMAL_PREREQ_CARISMA).toBe(1)
  })

  it('treinado em Adestramento', () => {
    expect(COMPANHEIRO_ANIMAL_PREREQ_SKILL).toBe('Adestramento')
  })
})

describe('type helpers', () => {
  it('baseTipoToParceiroType: montaria → montaria', () => {
    expect(baseTipoToParceiroType('montaria')).toBe('montaria')
  })

  it('baseTipoToParceiroType: ajudante → ajudante', () => {
    expect(baseTipoToParceiroType('ajudante')).toBe('ajudante')
  })

  it('baseTipoToParceiroType: combatente → combatente', () => {
    expect(baseTipoToParceiroType('combatente')).toBe('combatente')
  })

  it('isBaseTipoAllowed: aceita fortao', () => {
    expect(isBaseTipoAllowed('fortao')).toBe(true)
  })

  it('isBaseTipoAllowed: rejeita adepto (arcano)', () => {
    expect(isBaseTipoAllowed('adepto')).toBe(false)
  })

  it('isBaseTipoAllowed: rejeita vigilante', () => {
    expect(isBaseTipoAllowed('vigilante')).toBe(false)
  })

  it('requiresMagicoAugment: aceita adepto/destruidor/magivocador/medico', () => {
    expect(requiresMagicoAugment('adepto')).toBe(true)
    expect(requiresMagicoAugment('destruidor')).toBe(true)
    expect(requiresMagicoAugment('magivocador')).toBe(true)
    expect(requiresMagicoAugment('medico')).toBe(true)
  })

  it('requiresMagicoAugment: rejeita tipos base', () => {
    expect(requiresMagicoAugment('ajudante')).toBe(false)
    expect(requiresMagicoAugment('combatente')).toBe(false)
    expect(requiresMagicoAugment('montaria')).toBe(false)
  })

  it('requiresMagicoAugment: rejeita vigilante', () => {
    expect(requiresMagicoAugment('vigilante')).toBe(false)
  })
})
