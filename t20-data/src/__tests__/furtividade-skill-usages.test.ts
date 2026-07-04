import { describe, expect, it } from 'vitest'
import {
  ESCONDER_SE_ATTACK_OR_FLASHY_PENALTY,
  ESCONDER_SE_MOVED_PENALTY,
  FURTIVIDADE_ARMOR_PENALTY,
  FURTIVIDADE_TRAINED_ONLY,
  FURTIVIDADE_USAGES,
  SEGUIR_FAILURE_NOTICE_FRACTION,
  SEGUIR_NO_COVER_PENALTY,
  SEGUIR_TARGET_PRECAUTION_PERCEPCAO_BONUS,
  esconderSeActionPenalty,
  esconderSeMovementPenalty,
  esconderSeOutcome,
  esconderSeTotalModifier,
  furtividadeUsageByKind,
  seguirOutcome,
  seguirTargetPercepcaoBonus,
  seguirTerrainPenalty,
} from '../furtividade-skill-usages'

/**
 * PDF livro p119 — Perícia Furtividade (DES, penalidade de armadura).
 *  1. Esconder-se — ação livre fim de turno, oposto Percepção; -5 movido; -20 chamativa
 *  2. Seguir — oposto Percepção; -5 sem cobertura; alvo precavido +5; falha → percebe na metade
 */

describe('FURTIVIDADE_USAGES — shape', () => {
  it('exatamente 2 usos', () => {
    expect(FURTIVIDADE_USAGES.length).toBe(2)
  })

  it('frozen', () => {
    expect(Object.isFrozen(FURTIVIDADE_USAGES)).toBe(true)
  })

  it('ids canônicos', () => {
    expect(FURTIVIDADE_USAGES.map((u) => u.id).sort()).toEqual([
      'esconder-se',
      'seguir',
    ])
  })

  it('todos em p119', () => {
    for (const u of FURTIVIDADE_USAGES) {
      expect(u.bookPage).toBe(119)
    }
  })

  it('todos opostos por Percepção', () => {
    for (const u of FURTIVIDADE_USAGES) {
      expect(u.opposedBy).toBe('percepcao')
    }
  })
})

describe('Perícia flags — Tabela 2-1', () => {
  it('NÃO é somente treinada', () => {
    expect(FURTIVIDADE_TRAINED_ONLY).toBe(false)
  })

  it('com penalidade de armadura', () => {
    expect(FURTIVIDADE_ARMOR_PENALTY).toBe(true)
  })
})

describe('Esconder-se — p119', () => {
  const usage = furtividadeUsageByKind('esconder-se')

  it('ação livre no fim do turno', () => {
    if (usage.kind !== 'esconder-se') throw new Error('narrow failed')
    expect(usage.action).toBe('livre-fim-de-turno')
  })

  it('constantes verbatim', () => {
    expect(ESCONDER_SE_MOVED_PENALTY).toBe(-5)
    expect(ESCONDER_SE_ATTACK_OR_FLASHY_PENALTY).toBe(-20)
  })

  it('metade do deslocamento evita penalidade de movimento', () => {
    if (usage.kind !== 'esconder-se') throw new Error('narrow failed')
    expect(usage.halfSpeedAvoidsMovementPenalty).toBe(true)
  })

  it('sucesso concede camuflagem total', () => {
    if (usage.kind !== 'esconder-se') throw new Error('narrow failed')
    expect(usage.successGrantsCamuflagemTotal).toBe(true)
  })
})

describe('esconderSeMovementPenalty', () => {
  it('parado → 0', () => {
    expect(esconderSeMovementPenalty('parado')).toBe(0)
  })

  it('metade-deslocamento → 0 (evita penalidade)', () => {
    expect(esconderSeMovementPenalty('metade-deslocamento')).toBe(0)
  })

  it('deslocamento-completo → -5', () => {
    expect(esconderSeMovementPenalty('deslocamento-completo')).toBe(-5)
  })
})

describe('esconderSeActionPenalty', () => {
  it('nenhuma → 0', () => {
    expect(esconderSeActionPenalty('nenhuma')).toBe(0)
  })

  it('atacou-ou-chamativa → -20', () => {
    expect(esconderSeActionPenalty('atacou-ou-chamativa')).toBe(-20)
  })
})

describe('esconderSeTotalModifier', () => {
  it('parado + nada → 0', () => {
    expect(esconderSeTotalModifier('parado', 'nenhuma')).toBe(0)
  })

  it('deslocamento-completo + ataque → -25', () => {
    expect(esconderSeTotalModifier('deslocamento-completo', 'atacou-ou-chamativa')).toBe(-25)
  })

  it('metade + nada → 0 (contorna movimento)', () => {
    expect(esconderSeTotalModifier('metade-deslocamento', 'nenhuma')).toBe(0)
  })
})

describe('esconderSeOutcome', () => {
  it('furt > percep observador → camuflagem-total', () => {
    expect(esconderSeOutcome(20, 15)).toBe('camuflagem-total')
  })

  it('empate → percebido (observador não falhou)', () => {
    expect(esconderSeOutcome(20, 20)).toBe('percebido')
  })

  it('furt < percep observador → percebido', () => {
    expect(esconderSeOutcome(15, 20)).toBe('percebido')
  })
})

describe('Seguir — p119', () => {
  const usage = furtividadeUsageByKind('seguir')

  it('ação estendida', () => {
    if (usage.kind !== 'seguir') throw new Error('narrow failed')
    expect(usage.action).toBe('estendida')
  })

  it('constantes verbatim', () => {
    expect(SEGUIR_NO_COVER_PENALTY).toBe(-5)
    expect(SEGUIR_TARGET_PRECAUTION_PERCEPCAO_BONUS).toBe(5)
    expect(SEGUIR_FAILURE_NOTICE_FRACTION).toBe(0.5)
  })

  it('falha faz vítima perceber na metade do caminho', () => {
    if (usage.kind !== 'seguir') throw new Error('narrow failed')
    expect(usage.failureNoticeFraction).toBe(0.5)
  })
})

describe('seguirTerrainPenalty', () => {
  it('com esconderijos → 0', () => {
    expect(seguirTerrainPenalty('com-esconderijos')).toBe(0)
  })

  it('sem esconderijos → -5', () => {
    expect(seguirTerrainPenalty('sem-esconderijos-ou-movimento')).toBe(-5)
  })
})

describe('seguirTargetPercepcaoBonus', () => {
  it('precavida → +5', () => {
    expect(seguirTargetPercepcaoBonus(true)).toBe(5)
  })

  it('não precavida → 0', () => {
    expect(seguirTargetPercepcaoBonus(false)).toBe(0)
  })
})

describe('seguirOutcome', () => {
  it('furt > percep → segue-ate-destino', () => {
    expect(seguirOutcome(20, 15)).toBe('segue-ate-destino')
  })

  it('empate → segue-ate-destino (perseguidor vence empate)', () => {
    expect(seguirOutcome(20, 20)).toBe('segue-ate-destino')
  })

  it('furt < percep → percebido-na-metade', () => {
    expect(seguirOutcome(15, 20)).toBe('percebido-na-metade')
  })
})

describe('furtividadeUsageByKind', () => {
  it('throws se kind inválido', () => {
    expect(() =>
      // @ts-expect-error — invalid kind on purpose
      furtividadeUsageByKind('espreitar'),
    ).toThrow(/unknown kind/)
  })

  it('resolve todos', () => {
    for (const k of ['esconder-se', 'seguir'] as const) {
      expect(furtividadeUsageByKind(k).kind).toBe(k)
    }
  })
})
