import { describe, expect, it } from 'vitest'
import {
  AURA_BY_ITEM_MAGICO_CATEGORY,
  AURA_BY_SPELL_CIRCLE,
  DETECTAR_MAGIA_BARREIRA_FERRO_CHUMBO_PENALTY,
  DETECTAR_MAGIA_BARREIRA_MADEIRA_PEDRA_PENALTY,
  DETECTAR_MAGIA_CD,
  IDENTIFICAR_CRIATURA_CD_BASE,
  IDENTIFICAR_CRIATURA_EXTRA_TRAIT_MARGIN,
  IDENTIFICAR_CRIATURA_WRONG_INFO_MARGIN,
  IDENTIFICAR_ITEM_MAGICO_CD_MAIOR,
  IDENTIFICAR_ITEM_MAGICO_CD_MEDIO,
  IDENTIFICAR_ITEM_MAGICO_CD_MENOR,
  IDENTIFICAR_ITEM_MAGICO_RUSHED_PENALTY,
  IDENTIFICAR_MAGIA_CD_BASE,
  INFORMACAO_CD_COMPLEXA,
  INFORMACAO_CD_MISTERIO,
  LANCAR_MAGIA_ARMADURA_CD_BASE,
  MISTICISMO_ARMOR_PENALTY,
  MISTICISMO_TRAINED_ONLY,
  MISTICISMO_USAGES,
  auraForItemMagicoCategory,
  auraForSpecialSource,
  auraForSpellCircle,
  detectarMagiaBarrierPenalty,
  misticismoIdentificarCriaturaCd,
  identificarCriaturaOutcome,
  identificarCriaturaTraitsRecalled,
  identificarItemMagicoCd,
  identificarItemMagicoRushedPenalty,
  identificarMagiaCd,
  informacaoCd,
  lancarMagiaArmaduraCd,
  lancarMagiaArmaduraOutcome,
  misticismoUsageByKind,
} from '../misticismo-skill-usages'

/**
 * PDF livro p121 — Perícia Misticismo (INT, treinada).
 *  1. Detectar Magia — CD 15, ação completa, barreira -5/-10
 *  2. Identificar Criatura — CD 15+ND, +1 trait por 5 acima, falha 5+ = errada
 *  3. Identificar Item Mágico — CD 20/25/30, 1h ou completa @ -10
 *  4. Identificar Magia — CD 15 + custo PM, reação
 *  5. Informação — sem teste / CD 20 / CD 30
 *  6. Lançar Magia de Armadura — CD 20 + custo PM, armor penalty, falha consome PM
 */

describe('MISTICISMO_USAGES — shape', () => {
  it('exatamente 6 usos', () => {
    expect(MISTICISMO_USAGES.length).toBe(6)
  })

  it('frozen', () => {
    expect(Object.isFrozen(MISTICISMO_USAGES)).toBe(true)
  })

  it('ids canônicos', () => {
    expect(MISTICISMO_USAGES.map((u) => u.id).sort()).toEqual([
      'detectar-magia',
      'identificar-criatura',
      'identificar-item-magico',
      'identificar-magia',
      'informacao',
      'lancar-magia-de-armadura',
    ])
  })

  it('todos em p121', () => {
    for (const u of MISTICISMO_USAGES) {
      expect(u.bookPage).toBe(121)
    }
  })
})

describe('Perícia flags — Tabela 2-1', () => {
  it('apenas treinada', () => {
    expect(MISTICISMO_TRAINED_ONLY).toBe(true)
  })

  it('sem penalidade de armadura', () => {
    expect(MISTICISMO_ARMOR_PENALTY).toBe(false)
  })
})

describe('Detectar Magia — p121', () => {
  it('CD 15 e penalidades verbatim', () => {
    expect(DETECTAR_MAGIA_CD).toBe(15)
    expect(DETECTAR_MAGIA_BARREIRA_MADEIRA_PEDRA_PENALTY).toBe(-5)
    expect(DETECTAR_MAGIA_BARREIRA_FERRO_CHUMBO_PENALTY).toBe(-10)
  })

  it('alcance curto + ação completa', () => {
    const usage = misticismoUsageByKind('detectar-magia')
    if (usage.kind !== 'detectar-magia') throw new Error('narrow failed')
    expect(usage.range).toBe('curto')
    expect(usage.action).toBe('completa')
  })
})

describe('detectarMagiaBarrierPenalty', () => {
  it('nenhuma → 0', () => {
    expect(detectarMagiaBarrierPenalty('nenhuma')).toBe(0)
  })

  it('madeira ou pedra → -5', () => {
    expect(detectarMagiaBarrierPenalty('madeira-ou-pedra')).toBe(-5)
  })

  it('ferro ou chumbo → -10', () => {
    expect(detectarMagiaBarrierPenalty('ferro-ou-chumbo')).toBe(-10)
  })
})

describe('AURA_BY_SPELL_CIRCLE / auraForSpellCircle', () => {
  it('frozen', () => {
    expect(Object.isFrozen(AURA_BY_SPELL_CIRCLE)).toBe(true)
  })

  it('1º-2º círculo → tênue', () => {
    expect(auraForSpellCircle(1)).toBe('tenue')
    expect(auraForSpellCircle(2)).toBe('tenue')
  })

  it('3º-4º círculo → moderada', () => {
    expect(auraForSpellCircle(3)).toBe('moderada')
    expect(auraForSpellCircle(4)).toBe('moderada')
  })

  it('5º círculo → poderosa', () => {
    expect(auraForSpellCircle(5)).toBe('poderosa')
  })
})

describe('AURA_BY_ITEM_MAGICO_CATEGORY / auraForItemMagicoCategory', () => {
  it('frozen', () => {
    expect(Object.isFrozen(AURA_BY_ITEM_MAGICO_CATEGORY)).toBe(true)
  })

  it('menor → tênue', () => {
    expect(auraForItemMagicoCategory('menor')).toBe('tenue')
  })

  it('médio → moderada', () => {
    expect(auraForItemMagicoCategory('medio')).toBe('moderada')
  })

  it('maior → poderosa', () => {
    expect(auraForItemMagicoCategory('maior')).toBe('poderosa')
  })
})

describe('auraForSpecialSource', () => {
  it('deus menor → avassaladora', () => {
    expect(auraForSpecialSource('deus-menor')).toBe('avassaladora')
  })

  it('artefato → avassaladora', () => {
    expect(auraForSpecialSource('artefato')).toBe('avassaladora')
  })
})

describe('Identificar Criatura — p121', () => {
  it('constantes verbatim', () => {
    expect(IDENTIFICAR_CRIATURA_CD_BASE).toBe(15)
    expect(IDENTIFICAR_CRIATURA_EXTRA_TRAIT_MARGIN).toBe(5)
    expect(IDENTIFICAR_CRIATURA_WRONG_INFO_MARGIN).toBe(5)
  })

  it('precisa ver criatura', () => {
    const usage = misticismoUsageByKind('identificar-criatura')
    if (usage.kind !== 'identificar-criatura') throw new Error('narrow failed')
    expect(usage.requiresSeeingCreature).toBe(true)
  })
})

describe('misticismoIdentificarCriaturaCd', () => {
  it('ND 0 → 15', () => {
    expect(misticismoIdentificarCriaturaCd(0)).toBe(15)
  })

  it('ND 8 → 23', () => {
    expect(misticismoIdentificarCriaturaCd(8)).toBe(23)
  })
})

describe('identificarCriaturaTraitsRecalled', () => {
  it('falha → 0', () => {
    expect(identificarCriaturaTraitsRecalled(14, 15)).toBe(0)
  })

  it('sucesso justo → 1', () => {
    expect(identificarCriaturaTraitsRecalled(15, 15)).toBe(1)
  })

  it('sucesso +4 (nada extra) → 1', () => {
    expect(identificarCriaturaTraitsRecalled(19, 15)).toBe(1)
  })

  it('sucesso +5 → 2', () => {
    expect(identificarCriaturaTraitsRecalled(20, 15)).toBe(2)
  })

  it('sucesso +15 → 4', () => {
    expect(identificarCriaturaTraitsRecalled(30, 15)).toBe(4)
  })
})

describe('identificarCriaturaOutcome', () => {
  it('sucesso → traits-recalled', () => {
    expect(identificarCriaturaOutcome(20, 15)).toBe('traits-recalled')
  })

  it('falha < 5 → failed', () => {
    expect(identificarCriaturaOutcome(11, 15)).toBe('failed')
  })

  it('falha ≥ 5 → wrong-info', () => {
    expect(identificarCriaturaOutcome(10, 15)).toBe('wrong-info')
  })
})

describe('Identificar Item Mágico — p121', () => {
  it('CDs verbatim', () => {
    expect(IDENTIFICAR_ITEM_MAGICO_CD_MENOR).toBe(20)
    expect(IDENTIFICAR_ITEM_MAGICO_CD_MEDIO).toBe(25)
    expect(IDENTIFICAR_ITEM_MAGICO_CD_MAIOR).toBe(30)
    expect(IDENTIFICAR_ITEM_MAGICO_RUSHED_PENALTY).toBe(-10)
  })
})

describe('identificarItemMagicoCd', () => {
  it.each([
    ['menor', 20],
    ['medio', 25],
    ['maior', 30],
  ] as const)('%s → %s', (c, cd) => {
    expect(identificarItemMagicoCd(c)).toBe(cd)
  })
})

describe('identificarItemMagicoRushedPenalty', () => {
  it('rushed → -10', () => {
    expect(identificarItemMagicoRushedPenalty(true)).toBe(-10)
  })

  it('normal (1 hora) → 0', () => {
    expect(identificarItemMagicoRushedPenalty(false)).toBe(0)
  })
})

describe('Identificar Magia — p121', () => {
  it('CD base 15', () => {
    expect(IDENTIFICAR_MAGIA_CD_BASE).toBe(15)
  })

  it('reação', () => {
    const usage = misticismoUsageByKind('identificar-magia')
    if (usage.kind !== 'identificar-magia') throw new Error('narrow failed')
    expect(usage.action).toBe('reacao')
  })
})

describe('identificarMagiaCd', () => {
  it('custo PM 0 → 15', () => {
    expect(identificarMagiaCd(0)).toBe(15)
  })

  it('custo PM 5 → 20', () => {
    expect(identificarMagiaCd(5)).toBe(20)
  })
})

describe('Informação — p121', () => {
  it('CDs verbatim', () => {
    expect(INFORMACAO_CD_COMPLEXA).toBe(20)
    expect(INFORMACAO_CD_MISTERIO).toBe(30)
  })

  it('simples não exige teste', () => {
    const usage = misticismoUsageByKind('informacao')
    if (usage.kind !== 'informacao') throw new Error('narrow failed')
    expect(usage.simplesRequiresNoTest).toBe(true)
  })
})

describe('informacaoCd', () => {
  it('simples → null', () => {
    expect(informacaoCd('simples')).toBeNull()
  })

  it('complexa → 20', () => {
    expect(informacaoCd('complexa')).toBe(20)
  })

  it('mistério/enigma → 30', () => {
    expect(informacaoCd('misterio-ou-enigma')).toBe(30)
  })
})

describe('Lançar Magia de Armadura — p121', () => {
  const usage = misticismoUsageByKind('lancar-magia-de-armadura')

  it('CD base 20', () => {
    expect(LANCAR_MAGIA_ARMADURA_CD_BASE).toBe(20)
  })

  it('sofre armor penalty apesar da perícia não ter flag global', () => {
    if (usage.kind !== 'lancar-magia-de-armadura') throw new Error('narrow failed')
    expect(usage.armorPenaltyApplies).toBe(true)
    expect(MISTICISMO_ARMOR_PENALTY).toBe(false)
  })

  it('falha ainda consome PM', () => {
    if (usage.kind !== 'lancar-magia-de-armadura') throw new Error('narrow failed')
    expect(usage.failureStillConsumesPM).toBe(true)
  })

  it('só arcana', () => {
    if (usage.kind !== 'lancar-magia-de-armadura') throw new Error('narrow failed')
    expect(usage.arcaneOnly).toBe(true)
  })
})

describe('lancarMagiaArmaduraCd', () => {
  it('custo PM 3 → 23', () => {
    expect(lancarMagiaArmaduraCd(3)).toBe(23)
  })
})

describe('lancarMagiaArmaduraOutcome', () => {
  it('sucesso → success', () => {
    expect(lancarMagiaArmaduraOutcome(25, 23)).toBe('success')
  })

  it('falha → failed-still-consumes-pm', () => {
    expect(lancarMagiaArmaduraOutcome(20, 23)).toBe(
      'failed-still-consumes-pm',
    )
  })
})

describe('misticismoUsageByKind', () => {
  it('throws se kind inválido', () => {
    expect(() =>
      // @ts-expect-error — invalid kind on purpose
      misticismoUsageByKind('canalizar-tormenta'),
    ).toThrow(/unknown kind/)
  })

  it('resolve todos', () => {
    for (const k of [
      'detectar-magia',
      'identificar-criatura',
      'identificar-item-magico',
      'identificar-magia',
      'informacao',
      'lancar-magia-de-armadura',
    ] as const) {
      expect(misticismoUsageByKind(k).kind).toBe(k)
    }
  })
})
