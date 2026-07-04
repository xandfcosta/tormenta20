import { describe, expect, it } from 'vitest'
import {
  INTERROGAR_CD_CONFIDENCIAL,
  INTERROGAR_CD_RESTRITA,
  INTERROGAR_COST_DICE,
  INTERROGAR_DURATION_HOURS,
  INVESTIGACAO_ARMOR_PENALTY,
  INVESTIGACAO_TRAINED_ONLY,
  INVESTIGACAO_USAGES,
  PROCURAR_CD_BAGUNCA,
  PROCURAR_CD_ESCONDIDO,
  PROCURAR_CD_MUITO_ESCONDIDO,
  followingTracksSkill,
  interrogarCd,
  interrogarRequiresRoll,
  investigacaoUsageByKind,
  procurarCd,
} from '../investigacao-skill-usages'

/**
 * PDF livro p120 — Perícia Investigação (INT, aberta). 2 usos:
 *  1. Interrogar — 1 hora + T$ 3d6; geral sem teste, restrita CD 20, confidencial CD 30
 *  2. Procurar — ação completa até 1 dia; CDs 15/20/30 por ocultação;
 *     armadilhas CD própria; rastros → seguir requer Sobrevivência
 */

describe('INVESTIGACAO_USAGES — shape', () => {
  it('exatamente 2 usos', () => {
    expect(INVESTIGACAO_USAGES.length).toBe(2)
  })

  it('frozen', () => {
    expect(Object.isFrozen(INVESTIGACAO_USAGES)).toBe(true)
  })

  it('ids canônicos', () => {
    expect(INVESTIGACAO_USAGES.map((u) => u.id).sort()).toEqual([
      'interrogar',
      'procurar',
    ])
  })

  it('todos bookPage 120', () => {
    for (const u of INVESTIGACAO_USAGES) expect(u.bookPage).toBe(120)
  })
})

describe('Perícia flags — Tabela 2-1', () => {
  it('perícia aberta', () => {
    expect(INVESTIGACAO_TRAINED_ONLY).toBe(false)
  })

  it('sem penalidade de armadura', () => {
    expect(INVESTIGACAO_ARMOR_PENALTY).toBe(false)
  })
})

describe('Interrogar — p120', () => {
  const usage = investigacaoUsageByKind('interrogar')

  it('ação uma hora', () => {
    if (usage.kind !== 'interrogar') throw new Error('narrow failed')
    expect(usage.action).toBe('uma-hora')
    expect(INTERROGAR_DURATION_HOURS).toBe(1)
  })

  it('custo T$ 3d6', () => {
    if (usage.kind !== 'interrogar') throw new Error('narrow failed')
    expect(usage.costDice).toBe('3d6')
    expect(usage.costCurrency).toBe('tibar')
    expect(INTERROGAR_COST_DICE).toBe('3d6')
  })

  it('CDs verbatim', () => {
    expect(INTERROGAR_CD_RESTRITA).toBe(20)
    expect(INTERROGAR_CD_CONFIDENCIAL).toBe(30)
  })
})

describe('interrogarCd', () => {
  it('geral → 0 (sem teste)', () => {
    expect(interrogarCd('geral')).toBe(0)
  })

  it('restrita → 20', () => {
    expect(interrogarCd('restrita')).toBe(20)
  })

  it('confidencial → 30', () => {
    expect(interrogarCd('confidencial')).toBe(30)
  })
})

describe('interrogarRequiresRoll', () => {
  it('geral não exige roll', () => {
    expect(interrogarRequiresRoll('geral')).toBe(false)
  })

  it('restrita exige roll', () => {
    expect(interrogarRequiresRoll('restrita')).toBe(true)
  })

  it('confidencial exige roll', () => {
    expect(interrogarRequiresRoll('confidencial')).toBe(true)
  })
})

describe('Procurar — p120', () => {
  const usage = investigacaoUsageByKind('procurar')

  it('ação completa até 1 dia', () => {
    if (usage.kind !== 'procurar') throw new Error('narrow failed')
    expect(usage.actionMin).toBe('completa')
    expect(usage.actionMax).toBe('um-dia')
  })

  it('CDs por tier verbatim', () => {
    expect(PROCURAR_CD_BAGUNCA).toBe(15)
    expect(PROCURAR_CD_ESCONDIDO).toBe(20)
    expect(PROCURAR_CD_MUITO_ESCONDIDO).toBe(30)
  })

  it('armadilhas usam CD própria', () => {
    if (usage.kind !== 'procurar') throw new Error('narrow failed')
    expect(usage.trapsUseTheirOwnCd).toBe(true)
  })

  it('seguir rastros requer Sobrevivência', () => {
    if (usage.kind !== 'procurar') throw new Error('narrow failed')
    expect(usage.followingTracksRequiresSobrevivencia).toBe(true)
  })
})

describe('procurarCd', () => {
  it('bagunça → 15', () => {
    expect(procurarCd('bagunca')).toBe(15)
  })

  it('escondido → 20', () => {
    expect(procurarCd('escondido')).toBe(20)
  })

  it('muito-escondido → 30', () => {
    expect(procurarCd('muito-escondido')).toBe(30)
  })
})

describe('followingTracksSkill', () => {
  it('retorna sobrevivencia (verbatim redir)', () => {
    expect(followingTracksSkill()).toBe('sobrevivencia')
  })
})

describe('investigacaoUsageByKind', () => {
  it('throws se kind inválido', () => {
    expect(() =>
      // @ts-expect-error — invalid kind on purpose
      investigacaoUsageByKind('decifrar'),
    ).toThrow(/unknown kind/)
  })

  it('resolve interrogar', () => {
    expect(investigacaoUsageByKind('interrogar').name).toBe('Interrogar')
  })

  it('resolve procurar', () => {
    expect(investigacaoUsageByKind('procurar').name).toBe('Procurar')
  })
})
