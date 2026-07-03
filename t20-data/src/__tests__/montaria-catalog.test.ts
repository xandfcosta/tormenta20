import { describe, expect, it } from 'vitest'
import {
  CAVALGAR_CD_GUIAR,
  CAVALGAR_DANO_QUEDA,
  CAVALGAR_MARGEM_QUEDA,
  COMPRA_PATAMAR_PADRAO,
  DESMONTAR_ACAO,
  MONTAR_ACAO,
  MONTARIA_CATALOG,
  PENALIDADE_ATAQUE_DISTANCIA_MONTADO,
  cavalgarCdAposDano,
  montariaById,
  montariaTier,
} from '../montaria-catalog'
import { PARCEIRO_TIERS } from '../parceiro-rules'

/**
 * PDF Cap 6 p261-262. Pinned:
 *  - 6 montarias × 3 patamares
 *  - Cavalgar CD 10 guiar, falha por 5+ → cai 1d6, CD dano = dano sofrido
 *  - Penalidade -2 em ataques à distância + condição ruim conjurar
 *  - Compra fora de habilidade sempre iniciante (p262)
 */

describe('combate montado — constantes p261-262', () => {
  it('Cavalgar CD guiar = 10', () => {
    expect(CAVALGAR_CD_GUIAR).toBe(10)
  })

  it('margem de queda = 5', () => {
    expect(CAVALGAR_MARGEM_QUEDA).toBe(5)
  })

  it('dano de queda = 1d6', () => {
    expect(CAVALGAR_DANO_QUEDA).toBe('1d6')
  })

  it('penalidade ataque à distância montado = -2', () => {
    expect(PENALIDADE_ATAQUE_DISTANCIA_MONTADO).toBe(-2)
  })

  it('montar/desmontar são ações de movimento', () => {
    expect(MONTAR_ACAO).toBe('movimento')
    expect(DESMONTAR_ACAO).toBe('movimento')
  })

  it('compra fora de habilidade → sempre iniciante', () => {
    expect(COMPRA_PATAMAR_PADRAO).toBe('iniciante')
  })
})

describe('cavalgarCdAposDano — CD = dano sofrido', () => {
  it('dano 12 → CD 12', () => {
    expect(cavalgarCdAposDano(12)).toBe(12)
  })

  it('dano 0 → CD 0', () => {
    expect(cavalgarCdAposDano(0)).toBe(0)
  })

  it('throws se dano negativo', () => {
    expect(() => cavalgarCdAposDano(-1)).toThrow(/dano/)
  })
})

describe('MONTARIA_CATALOG — cobertura', () => {
  it('6 montarias', () => {
    expect(MONTARIA_CATALOG.length).toBe(6)
  })

  it('frozen', () => {
    expect(Object.isFrozen(MONTARIA_CATALOG)).toBe(true)
  })

  it('IDs únicos', () => {
    const ids = MONTARIA_CATALOG.map((m) => m.id)
    expect(new Set(ids).size).toBe(6)
  })

  it('todas com bookPage 262', () => {
    for (const m of MONTARIA_CATALOG) {
      expect(m.bookPage).toBe(262)
    }
  })

  it('todo tier presente para toda montaria', () => {
    for (const m of MONTARIA_CATALOG) {
      for (const p of PARCEIRO_TIERS) {
        expect(m.tiers[p]).toBeDefined()
      }
    }
  })
})

describe('cavalo', () => {
  it('deslocamento 12/15/15', () => {
    const m = montariaById('cavalo')!
    expect(m.tiers.iniciante.deslocamentoM).toBe(12)
    expect(m.tiers.veterano.deslocamentoM).toBe(15)
    expect(m.tiers.mestre.deslocamentoM).toBe(15)
  })

  it('mestre tem 2 ações de movimento extras', () => {
    expect(montariaTier('cavalo', 'mestre')!.acoesMovimentoExtras).toBe(2)
  })

  it('veterano/mestre: +2 em ataques CaC', () => {
    expect(montariaTier('cavalo', 'veterano')!.bonusAtaqueCaC).toBe(2)
    expect(montariaTier('cavalo', 'mestre')!.bonusAtaqueCaC).toBe(2)
  })

  it('alias pônei = médio', () => {
    const m = montariaById('cavalo')!
    expect(m.aliasWithSize).toEqual({ name: 'Pônei', size: 'medio' })
  })

  it('preço T$ 75 (tabela Cap 4)', () => {
    expect(montariaById('cavalo')!.precoTS).toBe(75)
  })

  it('todos os tiers são ridáveis', () => {
    for (const p of PARCEIRO_TIERS) {
      expect(montariaTier('cavalo', p)!.ridavel).toBe(true)
    }
  })
})

describe('cão de caça', () => {
  it('faro em todos os patamares', () => {
    for (const p of PARCEIRO_TIERS) {
      expect(montariaTier('cao-de-caca', p)!.sentidosGanhos).toBe('faro')
    }
  })

  it('veterano +2 defesa', () => {
    expect(montariaTier('cao-de-caca', 'veterano')!.bonusDefesa).toBe(2)
  })

  it('mestre: derrubar como manobra livre', () => {
    expect(montariaTier('cao-de-caca', 'mestre')!.manobrasLivres).toBe('derrubar')
  })

  it('tamanho médio + alias pequeno', () => {
    const m = montariaById('cao-de-caca')!
    expect(m.tamanho).toBe('medio')
    expect(m.aliasWithSize?.size).toBe('pequeno')
  })
})

describe('lobo-das-cavernas', () => {
  it('veterano/mestre +1d8 dano CaC 1×/rodada', () => {
    expect(montariaTier('lobo-das-cavernas', 'veterano')!.bonusDanoCaC).toEqual({
      die: '+1d8',
      timesPerRound: 1,
    })
    expect(montariaTier('lobo-das-cavernas', 'mestre')!.bonusDanoCaC).toEqual({
      die: '+1d8',
      timesPerRound: 1,
    })
  })

  it('iniciante SEM bônus de dano', () => {
    expect(montariaTier('lobo-das-cavernas', 'iniciante')!.bonusDanoCaC).toBeUndefined()
  })

  it('mestre: derrubar como manobra livre', () => {
    expect(montariaTier('lobo-das-cavernas', 'mestre')!.manobrasLivres).toBe(
      'derrubar',
    )
  })

  it('alias lobo comum = médio', () => {
    expect(montariaById('lobo-das-cavernas')!.aliasWithSize?.size).toBe('medio')
  })

  it('sem preço na tabela', () => {
    expect(montariaById('lobo-das-cavernas')!.precoTS).toBeNull()
  })
})

describe('grifo — caso especial iniciante', () => {
  it('iniciante NÃO é ridável (filhote)', () => {
    expect(montariaTier('grifo', 'iniciante')!.ridavel).toBe(false)
  })

  it('veterano/mestre são ridáveis com voo 18m', () => {
    const v = montariaTier('grifo', 'veterano')!
    const m = montariaTier('grifo', 'mestre')!
    expect(v.ridavel).toBe(true)
    expect(m.ridavel).toBe(true)
    expect(v.movementType).toBe('voo')
    expect(m.movementType).toBe('voo')
    expect(v.deslocamentoM).toBe(18)
    expect(m.deslocamentoM).toBe(18)
  })

  it('+1d8 dano CaC em todos os patamares', () => {
    for (const p of PARCEIRO_TIERS) {
      expect(montariaTier('grifo', p)!.bonusDanoCaC).toEqual({
        die: '+1d8',
        timesPerRound: 1,
      })
    }
  })

  it('nota de preço: "quase impossível de encontrar"', () => {
    expect(montariaById('grifo')!.precoTS).toBeNull()
    expect(montariaById('grifo')!.notaPreco).toMatch(/quase impossível/)
  })

  it('mestre: ação de movimento extra', () => {
    expect(montariaTier('grifo', 'mestre')!.acoesMovimentoExtras).toBe(1)
  })
})

describe('gorlogg — ladder de dano CaC', () => {
  it('iniciante +1d6, veterano +1d10, mestre +2d8', () => {
    expect(montariaTier('gorlogg', 'iniciante')!.bonusDanoCaC?.die).toBe('+1d6')
    expect(montariaTier('gorlogg', 'veterano')!.bonusDanoCaC?.die).toBe('+1d10')
    expect(montariaTier('gorlogg', 'mestre')!.bonusDanoCaC?.die).toBe('+2d8')
  })

  it('deslocamento 12/12/15', () => {
    expect(montariaTier('gorlogg', 'iniciante')!.deslocamentoM).toBe(12)
    expect(montariaTier('gorlogg', 'veterano')!.deslocamentoM).toBe(12)
    expect(montariaTier('gorlogg', 'mestre')!.deslocamentoM).toBe(15)
  })

  it('sem preço na tabela', () => {
    expect(montariaById('gorlogg')!.precoTS).toBeNull()
  })
})

describe('trobo — ladder de resistências', () => {
  it('+1/+2/+5 em testes de resistência', () => {
    expect(montariaTier('trobo', 'iniciante')!.bonusResistencias).toBe(1)
    expect(montariaTier('trobo', 'veterano')!.bonusResistencias).toBe(2)
    expect(montariaTier('trobo', 'mestre')!.bonusResistencias).toBe(5)
  })

  it('preço T$ 60', () => {
    expect(montariaById('trobo')!.precoTS).toBe(60)
  })

  it('deslocamento 9/12/12', () => {
    expect(montariaTier('trobo', 'iniciante')!.deslocamentoM).toBe(9)
    expect(montariaTier('trobo', 'veterano')!.deslocamentoM).toBe(12)
    expect(montariaTier('trobo', 'mestre')!.deslocamentoM).toBe(12)
  })
})

describe('lookup helpers', () => {
  it('montariaById miss → undefined', () => {
    // @ts-expect-error id inválido
    expect(montariaById('unicornio')).toBeUndefined()
  })

  it('montariaTier retorna o tier certo', () => {
    expect(montariaTier('cavalo', 'iniciante')!.deslocamentoM).toBe(12)
  })
})
