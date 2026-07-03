import { describe, expect, it } from 'vitest'
import {
  PARCEIRO_BENEFITS,
  PARCEIRO_BENEFIT_DESCRIPTIONS,
  parceiroBenefit,
  parceiroBenefitDescription,
} from '../parceiro-benefits'
import { PARCEIRO_TIERS, PARCEIRO_TYPES } from '../parceiro-rules'

/**
 * PDF Cap 6 p260-261. Pinned mecanical + verbatim descriptions per
 * (tipo × patamar). 12 × 3 = 36 entries.
 */

describe('matrix cobertura completa', () => {
  it('todos 12 tipos × 3 patamares presentes', () => {
    for (const t of PARCEIRO_TYPES) {
      for (const p of PARCEIRO_TIERS) {
        expect(PARCEIRO_BENEFITS[t][p]).toBeDefined()
        expect(PARCEIRO_BENEFIT_DESCRIPTIONS[t][p]).toBeTruthy()
      }
    }
  })

  it('kind de cada benefit bate com tipo do índice', () => {
    for (const t of PARCEIRO_TYPES) {
      for (const p of PARCEIRO_TIERS) {
        expect(PARCEIRO_BENEFITS[t][p].kind).toBe(t)
      }
    }
  })

  it('matrix frozen', () => {
    expect(Object.isFrozen(PARCEIRO_BENEFITS)).toBe(true)
    expect(Object.isFrozen(PARCEIRO_BENEFITS.adepto)).toBe(true)
    expect(Object.isFrozen(PARCEIRO_BENEFIT_DESCRIPTIONS)).toBe(true)
  })
})

describe('adepto', () => {
  it('iniciante: só 1º círculo -1 PM, não cumulativo', () => {
    const b = parceiroBenefit('adepto', 'iniciante')
    expect(b.kind).toBe('adepto')
    if (b.kind === 'adepto') {
      expect(b.pmDiscountCirculos).toEqual([1])
      expect(b.cumulative).toBe(false)
    }
  })

  it('veterano: 1º e 2º círculo, não cumulativo', () => {
    const b = parceiroBenefit('adepto', 'veterano')
    if (b.kind === 'adepto') {
      expect(b.pmDiscountCirculos).toEqual([1, 2])
      expect(b.cumulative).toBe(false)
    }
  })

  it('mestre: 1º e 2º círculo, cumulativo com outras reduções', () => {
    const b = parceiroBenefit('adepto', 'mestre')
    if (b.kind === 'adepto') {
      expect(b.pmDiscountCirculos).toEqual([1, 2])
      expect(b.cumulative).toBe(true)
    }
  })
})

describe('ajudante', () => {
  it('iniciante: +2 em 2 perícias', () => {
    const b = parceiroBenefit('ajudante', 'iniciante')
    if (b.kind === 'ajudante') {
      expect(b.skillBonus).toBe(2)
      expect(b.skillCount).toBe(2)
    }
  })

  it('veterano: +2 em 3 perícias', () => {
    const b = parceiroBenefit('ajudante', 'veterano')
    if (b.kind === 'ajudante') {
      expect(b.skillBonus).toBe(2)
      expect(b.skillCount).toBe(3)
    }
  })

  it('mestre: +4 em 3 perícias', () => {
    const b = parceiroBenefit('ajudante', 'mestre')
    if (b.kind === 'ajudante') {
      expect(b.skillBonus).toBe(4)
      expect(b.skillCount).toBe(3)
    }
  })

  it('exclui Luta/Pontaria em todos os patamares', () => {
    for (const p of PARCEIRO_TIERS) {
      const b = parceiroBenefit('ajudante', p)
      if (b.kind === 'ajudante') {
        expect(b.excludedSkills).toEqual(['Luta', 'Pontaria'])
      }
    }
  })
})

describe('assassino', () => {
  it('iniciante: +1d6 furtivo sem flank', () => {
    const b = parceiroBenefit('assassino', 'iniciante')
    if (b.kind === 'assassino') {
      expect(b.furtivoBonusDice).toBe('+1d6')
      expect(b.hasFlankBonus).toBe(false)
    }
  })

  it('veterano: +1d6 furtivo + flank', () => {
    const b = parceiroBenefit('assassino', 'veterano')
    if (b.kind === 'assassino') {
      expect(b.furtivoBonusDice).toBe('+1d6')
      expect(b.hasFlankBonus).toBe(true)
      expect(b.flankAssistsFurtivo).toBe(false)
    }
  })

  it('mestre: +2d6 furtivo + flank que facilita furtivo', () => {
    const b = parceiroBenefit('assassino', 'mestre')
    if (b.kind === 'assassino') {
      expect(b.furtivoBonusDice).toBe('+2d6')
      expect(b.hasFlankBonus).toBe(true)
      expect(b.flankAssistsFurtivo).toBe(true)
    }
  })
})

describe('atirador — damage dice ladder', () => {
  const expected: Record<string, string> = {
    iniciante: '+1d6',
    veterano: '+1d10',
    mestre: '+2d8',
  }
  for (const p of PARCEIRO_TIERS) {
    it(`${p} → ${expected[p]}`, () => {
      const b = parceiroBenefit('atirador', p)
      if (b.kind === 'atirador') expect(b.damageDie).toBe(expected[p])
    })
  }
})

describe('combatente', () => {
  it('iniciante +2 atk, sem ataque extra', () => {
    const b = parceiroBenefit('combatente', 'iniciante')
    if (b.kind === 'combatente') {
      expect(b.attackBonus).toBe(2)
      expect(b.extraAttackPm).toBeUndefined()
    }
  })

  it('veterano +3 atk', () => {
    const b = parceiroBenefit('combatente', 'veterano')
    if (b.kind === 'combatente') expect(b.attackBonus).toBe(3)
  })

  it('mestre +4 atk + 5 PM ataque extra', () => {
    const b = parceiroBenefit('combatente', 'mestre')
    if (b.kind === 'combatente') {
      expect(b.attackBonus).toBe(4)
      expect(b.extraAttackPm).toBe(5)
    }
  })
})

describe('destruidor — opções cumulativas', () => {
  it('iniciante: 1 opção (1 PM / 2d6)', () => {
    const b = parceiroBenefit('destruidor', 'iniciante')
    if (b.kind === 'destruidor') {
      expect(b.options.length).toBe(1)
      expect(b.options[0]).toEqual({ pmCost: 1, damageDice: '2d6' })
    }
  })

  it('veterano: 2 opções (1 PM / 2 PM)', () => {
    const b = parceiroBenefit('destruidor', 'veterano')
    if (b.kind === 'destruidor') {
      expect(b.options.length).toBe(2)
      expect(b.options[1]).toEqual({ pmCost: 2, damageDice: '4d6' })
    }
  })

  it('mestre: 3 opções, terceira em área', () => {
    const b = parceiroBenefit('destruidor', 'mestre')
    if (b.kind === 'destruidor') {
      expect(b.options.length).toBe(3)
      expect(b.options[2]).toEqual({
        pmCost: 4,
        damageDice: '6d6',
        area: 'raio 6m alcance médio',
      })
    }
  })

  it('menu de dano fixo em 4 elementos', () => {
    const b = parceiroBenefit('destruidor', 'iniciante')
    if (b.kind === 'destruidor') {
      expect(b.damageTypeMenu).toEqual(['acido', 'eletricidade', 'fogo', 'frio'])
    }
  })
})

describe('fortao — melee damage ladder', () => {
  const expected: Record<string, string> = {
    iniciante: '+1d8',
    veterano: '+1d12',
    mestre: '+3d6',
  }
  for (const p of PARCEIRO_TIERS) {
    it(`${p} → ${expected[p]}`, () => {
      const b = parceiroBenefit('fortao', p)
      if (b.kind === 'fortao') expect(b.damageDie).toBe(expected[p])
    })
  }
})

describe('guardiao', () => {
  it('iniciante +2 defesa', () => {
    const b = parceiroBenefit('guardiao', 'iniciante')
    if (b.kind === 'guardiao') expect(b.defesaBonus).toBe(2)
  })

  it('veterano +3 defesa', () => {
    const b = parceiroBenefit('guardiao', 'veterano')
    if (b.kind === 'guardiao') expect(b.defesaBonus).toBe(3)
  })

  it('mestre +4 defesa + 2 em resistência', () => {
    const b = parceiroBenefit('guardiao', 'mestre')
    if (b.kind === 'guardiao') {
      expect(b.defesaBonus).toBe(4)
      expect(b.resistBonus).toBe(2)
    }
  })
})

describe('magivocador', () => {
  it('iniciante: +1 dado, +0 CD', () => {
    const b = parceiroBenefit('magivocador', 'iniciante')
    if (b.kind === 'magivocador') {
      expect(b.damageDicePlus).toBe(1)
      expect(b.cdPlus).toBe(0)
    }
  })

  it('veterano: +1 dado, +1 CD', () => {
    const b = parceiroBenefit('magivocador', 'veterano')
    if (b.kind === 'magivocador') {
      expect(b.damageDicePlus).toBe(1)
      expect(b.cdPlus).toBe(1)
    }
  })

  it('mestre: +2 dado, +2 CD (dobra)', () => {
    const b = parceiroBenefit('magivocador', 'mestre')
    if (b.kind === 'magivocador') {
      expect(b.damageDicePlus).toBe(2)
      expect(b.cdPlus).toBe(2)
    }
  })
})

describe('medico — heal ladder', () => {
  it('iniciante: 1 opção (1 PM / 1d8+1)', () => {
    const b = parceiroBenefit('medico', 'iniciante')
    if (b.kind === 'medico') {
      expect(b.options.length).toBe(1)
      expect(b.options[0]).toEqual({ pmCost: 1, healDice: '1d8+1' })
    }
  })

  it('veterano: 2 opções, 3 PM remove condição', () => {
    const b = parceiroBenefit('medico', 'veterano')
    if (b.kind === 'medico') {
      expect(b.options.length).toBe(2)
      expect(b.options[1]).toEqual({
        pmCost: 3,
        healDice: '3d8+3',
        removesCondition: true,
      })
    }
  })

  it('mestre: 3 opções, cura 6d8+6 (5 PM)', () => {
    const b = parceiroBenefit('medico', 'mestre')
    if (b.kind === 'medico') {
      expect(b.options.length).toBe(3)
      expect(b.options[2]).toEqual({ pmCost: 5, healDice: '6d8+6' })
    }
  })
})

describe('perseguidor', () => {
  it('iniciante: +2 Percepção e Sobrevivência', () => {
    const b = parceiroBenefit('perseguidor', 'iniciante')
    if (b.kind === 'perseguidor') {
      expect(b.skillBonus).toEqual({
        skills: ['Percepção', 'Sobrevivência'],
        amount: 2,
      })
      expect(b.grantedPower).toBeUndefined()
    }
  })

  it('veterano: Sentidos Aguçados', () => {
    const b = parceiroBenefit('perseguidor', 'veterano')
    if (b.kind === 'perseguidor') {
      expect(b.grantedPower).toBe('Sentidos Aguçados')
    }
  })

  it('mestre: Percepção às Cegas', () => {
    const b = parceiroBenefit('perseguidor', 'mestre')
    if (b.kind === 'perseguidor') {
      expect(b.grantedPower).toBe('Percepção às Cegas')
    }
  })
})

describe('vigilante', () => {
  it('iniciante: +2 Percepção e Iniciativa', () => {
    const b = parceiroBenefit('vigilante', 'iniciante')
    if (b.kind === 'vigilante') {
      expect(b.skillBonus).toEqual({
        skills: ['Percepção', 'Iniciativa'],
        amount: 2,
      })
    }
  })

  it('veterano: Esquiva Sobrenatural', () => {
    const b = parceiroBenefit('vigilante', 'veterano')
    if (b.kind === 'vigilante') {
      expect(b.grantedPower).toBe('Esquiva Sobrenatural')
    }
  })

  it('mestre: Olhos nas Costas', () => {
    const b = parceiroBenefit('vigilante', 'mestre')
    if (b.kind === 'vigilante') {
      expect(b.grantedPower).toBe('Olhos nas Costas')
    }
  })
})

describe('descriptions verbatim (checkpoints)', () => {
  it('ajudante mestre menciona Luta/Pontaria', () => {
    const d = parceiroBenefitDescription('ajudante', 'mestre')
    expect(d).toMatch(/Luta ou Pontaria/)
  })

  it('assassino iniciante menciona cumulativo', () => {
    const d = parceiroBenefitDescription('assassino', 'iniciante')
    expect(d).toMatch(/cumulativo/)
  })

  it('destruidor mestre menciona 6m de raio + alcance médio', () => {
    const d = parceiroBenefitDescription('destruidor', 'mestre')
    expect(d).toMatch(/6m de raio.*alcance médio/)
  })

  it('médico veterano cita abalado/fatigado', () => {
    const d = parceiroBenefitDescription('medico', 'veterano')
    expect(d).toMatch(/abalado.*fatigado/)
  })
})
