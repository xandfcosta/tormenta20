import { describe, expect, it } from 'vitest'
import {
  BUSCA_CHALLENGE_TABLE,
  BUSCA_OUTCOME_TABLE,
  CHASE_EVENT_TABLE,
  REWARD_CASTIGO_TABLE,
  RUINA_TABLE,
  buscaChallengeFromRoll,
  buscaOutcomeFromSuccesses,
  buscaTestCd,
  chaseEventFromRoll,
  rewardCastigoFromRoll,
  ruinaFromRoll,
} from '../gm-random-tables'

// ─── Ermos: Ruína d6 (p269) ─────────────────────────────────────────

describe('RUINA_TABLE (Cap 6 p269)', () => {
  it('covers d6 = 1..6 contiguously', () => {
    for (let r = 1; r <= 6; r++) {
      expect(ruinaFromRoll(r)).toBeDefined()
    }
  })

  it('1-2 → ameaça', () => {
    expect(ruinaFromRoll(1).outcome).toBe('ameaca')
    expect(ruinaFromRoll(2).outcome).toBe('ameaca')
  })

  it('3-4 → vazia', () => {
    expect(ruinaFromRoll(3).outcome).toBe('vazia')
    expect(ruinaFromRoll(4).outcome).toBe('vazia')
  })

  it('5 → ameaça-e-tesouro', () => {
    expect(ruinaFromRoll(5).outcome).toBe('ameaca-e-tesouro')
  })

  it('6 → tesouro', () => {
    expect(ruinaFromRoll(6).outcome).toBe('tesouro')
  })

  it('rejects out-of-range', () => {
    expect(() => ruinaFromRoll(0)).toThrow()
    expect(() => ruinaFromRoll(7)).toThrow()
  })

  it('ranges are contiguous', () => {
    for (let i = 1; i < RUINA_TABLE.length; i++) {
      expect(RUINA_TABLE[i]!.rollMin).toBe(RUINA_TABLE[i - 1]!.rollMax + 1)
    }
  })
})

// ─── Tabela 6-5: Eventos de Perseguições d20 (p274) ─────────────────

describe('Tabela 6-5: Eventos de Perseguições (p274)', () => {
  it('covers d20 = 1..20 contiguously', () => {
    for (let r = 1; r <= 20; r++) {
      expect(chaseEventFromRoll(r)).toBeDefined()
    }
  })

  it('1-6 → nenhum', () => {
    for (let r = 1; r <= 6; r++) {
      expect(chaseEventFromRoll(r).kind).toBe('nenhum')
    }
  })

  it('7-8 → obstáculo Força CD 15', () => {
    const row = chaseEventFromRoll(7)
    expect(row.kind).toBe('obstaculo')
    expect(row.test).toBe('Força')
    expect(row.cd).toBe(15)
  })

  it('11-12 → obstáculo Reflexos CD 20', () => {
    const row = chaseEventFromRoll(11)
    expect(row.kind).toBe('obstaculo')
    expect(row.test).toBe('Reflexos')
    expect(row.cd).toBe(20)
  })

  it('15-16 → atalho Adestramento CD 20', () => {
    const row = chaseEventFromRoll(15)
    expect(row.kind).toBe('atalho')
    expect(row.test).toBe('Adestramento')
  })

  it('19-20 → atalho Percepção CD 20', () => {
    const row = chaseEventFromRoll(20)
    expect(row.kind).toBe('atalho')
    expect(row.test).toBe('Percepção')
    expect(row.cd).toBe(20)
  })

  it('ranges are contiguous', () => {
    for (let i = 1; i < CHASE_EVENT_TABLE.length; i++) {
      expect(CHASE_EVENT_TABLE[i]!.rollMin).toBe(
        CHASE_EVENT_TABLE[i - 1]!.rollMax + 1,
      )
    }
  })

  it('rejects out-of-range', () => {
    expect(() => chaseEventFromRoll(0)).toThrow()
    expect(() => chaseEventFromRoll(21)).toThrow()
  })
})

// ─── Tabela 6-6: Desafios de Buscas 2d12 (p279) ─────────────────────

describe('Tabela 6-6: Desafios de Buscas (p279)', () => {
  it('has entries for 2d12 = 2..24 (23 rows)', () => {
    expect(BUSCA_CHALLENGE_TABLE).toHaveLength(23)
    for (let r = 2; r <= 24; r++) {
      expect(buscaChallengeFromRoll(r as 2)).toBeDefined()
    }
  })

  it('2 = Misticismo, 24 = Guerra (boundaries)', () => {
    expect(buscaChallengeFromRoll(2).skill).toBe('Misticismo')
    expect(buscaChallengeFromRoll(24).skill).toBe('Guerra')
  })

  it('11 = Atletismo (2d12 mode) — Escalar um penhasco', () => {
    const row = buscaChallengeFromRoll(11)
    expect(row.skill).toBe('Atletismo')
    expect(row.example).toBe('Escalar um penhasco')
  })

  it('rolls are 2..24 unique + ordered', () => {
    expect(BUSCA_CHALLENGE_TABLE.map((r) => r.roll)).toEqual(
      Array.from({ length: 23 }, (_, i) => i + 2),
    )
  })

  it('rejects out-of-range', () => {
    expect(() => buscaChallengeFromRoll(1 as 2)).toThrow()
    expect(() => buscaChallengeFromRoll(25 as 2)).toThrow()
  })
})

describe('buscaTestCd (p278 — CD 20 + metade do nível)', () => {
  it('L1 → 20 (½=0)', () => {
    expect(buscaTestCd(1)).toBe(20)
  })

  it('L2 → 21', () => {
    expect(buscaTestCd(2)).toBe(21)
  })

  it('L10 → 25', () => {
    expect(buscaTestCd(10)).toBe(25)
  })

  it('L20 → 30 (capstone)', () => {
    expect(buscaTestCd(20)).toBe(30)
  })

  it('rejects L < 1', () => {
    expect(() => buscaTestCd(0)).toThrow()
    expect(() => buscaTestCd(-1)).toThrow()
  })
})

// ─── Tabela 6-7: Consequências de Buscas (p279) ─────────────────────

describe('Tabela 6-7: Consequências de Buscas (p279)', () => {
  it('0 sucessos → 1 castigo', () => {
    expect(buscaOutcomeFromSuccesses(0).result).toBe('1-castigo')
  })

  it('1 sucesso → nenhuma', () => {
    expect(buscaOutcomeFromSuccesses(1).result).toBe('nenhuma')
  })

  it('2 sucessos → 1 recompensa', () => {
    expect(buscaOutcomeFromSuccesses(2).result).toBe('1-recompensa')
  })

  it('3 sucessos → 2 recompensas', () => {
    expect(buscaOutcomeFromSuccesses(3).result).toBe('2-recompensas')
  })

  it('table covers all 4 rows', () => {
    expect(BUSCA_OUTCOME_TABLE).toHaveLength(4)
  })
})

describe('Recompensas & Castigos d6 (p279)', () => {
  it('has 6 rows aligned by d6', () => {
    expect(REWARD_CASTIGO_TABLE.map((r) => r.roll)).toEqual([1, 2, 3, 4, 5, 6])
  })

  it('1 = tesouro-riqueza / ruína-menor', () => {
    const r = rewardCastigoFromRoll(1)
    expect(r.reward).toBe('tesouro-riqueza')
    expect(r.castigo).toBe('ruina-menor')
  })

  it('5 = tesouro-ambos / maldição', () => {
    const r = rewardCastigoFromRoll(5)
    expect(r.reward).toBe('tesouro-ambos')
    expect(r.castigo).toBe('maldicao')
  })

  it('6 = poder / ruína-maior', () => {
    const r = rewardCastigoFromRoll(6)
    expect(r.reward).toBe('poder')
    expect(r.castigo).toBe('ruina-maior')
  })

  it('rejects out-of-range', () => {
    expect(() => rewardCastigoFromRoll(0)).toThrow()
    expect(() => rewardCastigoFromRoll(7)).toThrow()
  })
})
