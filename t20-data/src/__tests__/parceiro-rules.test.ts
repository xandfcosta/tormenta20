import { describe, expect, it } from 'vitest'
import {
  PARCEIRO_TIERS,
  PARCEIRO_TYPES,
  PARCEIRO_TYPE_LABELS,
  VULNERABLE_PARTNER_DIE,
  cavaleiroMontariaTier,
  companheiroAnimalTier,
  parceiroLimit,
  vulnerablePartnerDie,
} from '../parceiro-rules'

/**
 * PDF Cap 6 p260-262 "Regras de Parceiros". Pinned:
 *  - Limite: L1-4 → 1, L5-16 → 2, L17-20 → 3
 *  - Companheiro Animal (druida/caçador): iniciante <7, veterano 7-14, mestre 15+
 *  - Cavaleiro Montaria: veterano <11, mestre 11+
 *  - Variante Vulneráveis dado: d4/d6/d8 por patamar
 */

describe('PARCEIRO_TIERS + PARCEIRO_TYPES', () => {
  it('3 patamares em ordem canônica', () => {
    expect([...PARCEIRO_TIERS]).toEqual(['iniciante', 'veterano', 'mestre'])
  })

  it('PARCEIRO_TIERS frozen', () => {
    expect(Object.isFrozen(PARCEIRO_TIERS)).toBe(true)
  })

  it('12 tipos', () => {
    expect(PARCEIRO_TYPES.length).toBe(12)
  })

  it('PARCEIRO_TYPES frozen', () => {
    expect(Object.isFrozen(PARCEIRO_TYPES)).toBe(true)
  })

  it('tipos únicos + esperados', () => {
    expect(new Set(PARCEIRO_TYPES).size).toBe(12)
    expect(PARCEIRO_TYPES).toContain('adepto')
    expect(PARCEIRO_TYPES).toContain('magivocador')
    expect(PARCEIRO_TYPES).toContain('vigilante')
  })

  it('todo tipo tem label', () => {
    for (const t of PARCEIRO_TYPES) {
      expect(PARCEIRO_TYPE_LABELS[t]).toBeTruthy()
    }
  })

  it('label do Fortão tem cedilha (PT-BR)', () => {
    expect(PARCEIRO_TYPE_LABELS.fortao).toBe('Fortão')
  })
})

describe('parceiroLimit — p260', () => {
  it('L1 → 1', () => {
    expect(parceiroLimit(1)).toBe(1)
  })

  it('L4 → 1 (limite superior de iniciante)', () => {
    expect(parceiroLimit(4)).toBe(1)
  })

  it('L5 → 2 (primeiro patamar veterano)', () => {
    expect(parceiroLimit(5)).toBe(2)
  })

  it('L16 → 2 (limite superior de veterano/campeão)', () => {
    expect(parceiroLimit(16)).toBe(2)
  })

  it('L17 → 3 (lenda)', () => {
    expect(parceiroLimit(17)).toBe(3)
  })

  it('L20 → 3', () => {
    expect(parceiroLimit(20)).toBe(3)
  })

  it('throws se pcLevel < 1', () => {
    expect(() => parceiroLimit(0)).toThrow(/pcLevel/)
    expect(() => parceiroLimit(-3)).toThrow(/pcLevel/)
  })
})

describe('companheiroAnimalTier — druida/caçador p50 p61-62', () => {
  it('L1 → iniciante', () => {
    expect(companheiroAnimalTier(1)).toBe('iniciante')
  })

  it('L6 → iniciante (última classe antes de veterano)', () => {
    expect(companheiroAnimalTier(6)).toBe('iniciante')
  })

  it('L7 → veterano (upgrade explícito)', () => {
    expect(companheiroAnimalTier(7)).toBe('veterano')
  })

  it('L14 → veterano (última classe antes de mestre)', () => {
    expect(companheiroAnimalTier(14)).toBe('veterano')
  })

  it('L15 → mestre (upgrade explícito)', () => {
    expect(companheiroAnimalTier(15)).toBe('mestre')
  })

  it('L20 → mestre', () => {
    expect(companheiroAnimalTier(20)).toBe('mestre')
  })

  it('throws se classLevel < 1', () => {
    expect(() => companheiroAnimalTier(0)).toThrow(/classLevel/)
  })
})

describe('cavaleiroMontariaTier — Caminho p54-55', () => {
  it('L1 → veterano (baseline da habilidade)', () => {
    expect(cavaleiroMontariaTier(1)).toBe('veterano')
  })

  it('L10 → veterano', () => {
    expect(cavaleiroMontariaTier(10)).toBe('veterano')
  })

  it('L11 → mestre (upgrade explícito)', () => {
    expect(cavaleiroMontariaTier(11)).toBe('mestre')
  })

  it('L20 → mestre', () => {
    expect(cavaleiroMontariaTier(20)).toBe('mestre')
  })

  it('throws se cavaleiroLevel < 1', () => {
    expect(() => cavaleiroMontariaTier(0)).toThrow(/cavaleiroLevel/)
  })
})

describe('vulnerablePartnerDie — variante p262', () => {
  it('iniciante → d4', () => {
    expect(vulnerablePartnerDie('iniciante')).toBe('d4')
  })

  it('veterano → d6', () => {
    expect(vulnerablePartnerDie('veterano')).toBe('d6')
  })

  it('mestre → d8', () => {
    expect(vulnerablePartnerDie('mestre')).toBe('d8')
  })

  it('VULNERABLE_PARTNER_DIE frozen', () => {
    expect(Object.isFrozen(VULNERABLE_PARTNER_DIE)).toBe(true)
  })
})
