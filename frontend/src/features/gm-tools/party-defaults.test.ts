import { describe, expect, it } from 'vitest'
import type { CampaignMember } from '@/shared/api/types'
import { partyFromMembers } from './party-defaults'

/**
 * O grupo derivado da campanha (ALE-209).
 *
 * É unitário e não integração porque carrega REGRA: quem conta como jogador, o
 * arredondamento da média, e o que fazer quando não há ninguém. As três decidem
 * o ND que a Tabela 7-1 (p282) devolve, e um erro aqui faz o mestre montar um
 * encontro que mata a mesa sem que nada pareça errado na tela.
 */

function membro(over: Partial<CampaignMember> & { level?: number }): CampaignMember {
  const { level, ...resto } = over
  return {
    id: 1,
    campaignId: 1,
    characterId: 1,
    role: 'player',
    addedAt: '2026-01-01T00:00:00Z',
    ...(level === undefined
      ? {}
      : {
          character: {
            id: 1,
            ownerId: 1,
            name: 'PC',
            level,
            hpCurrent: 1,
            hpMax: 1,
            mpCurrent: 0,
            mpMax: 0,
            classes: [],
          },
        }),
    ...resto,
  } as CampaignMember
}

describe('o grupo da campanha', () => {
  it('é a média dos níveis e quantos são', () => {
    const grupo = partyFromMembers([
      membro({ level: 5 }),
      membro({ level: 6 }),
      membro({ level: 7 }),
    ])

    expect(grupo).toEqual({ level: 6, size: 3 })
  })

  // A Tabela 7-1 (p282) só tem linha para nível inteiro, então a média TEM de
  // arredondar. Meio nível vai para cima.
  it('arredonda a média para o nível inteiro', () => {
    const grupo = partyFromMembers([
      membro({ level: 1 }),
      membro({ level: 1 }),
      membro({ level: 2 }),
      membro({ level: 2 }),
    ])

    expect(grupo?.level).toBe(2)
  })

  // O mestre é membro da campanha e NÃO é um dos jogadores para quem o encontro
  // é dimensionado — contá-lo inflaria o grupo e faria o encontro sair fácil.
  it('o mestre não entra na conta', () => {
    const grupo = partyFromMembers([
      membro({ level: 10, role: 'gm' }),
      membro({ level: 2 }),
      membro({ level: 2 }),
    ])

    expect(grupo).toEqual({ level: 2, size: 2 })
  })

  // Estado parcial de carregamento: chutar o nível de quem não veio mentiria no
  // ND, e contá-lo no tamanho mentiria na dificuldade.
  it('membro sem o personagem embutido não conta', () => {
    const grupo = partyFromMembers([membro({ level: 4 }), membro({})])

    expect(grupo).toEqual({ level: 4, size: 1 })
  })

  // `null` e não `{level: 1, size: 0}`: um grupo de tamanho zero não tem linha
  // na tabela, e quem chama tem o próprio padrão para usar.
  it('sem jogador nenhum, não inventa grupo', () => {
    expect(partyFromMembers([])).toBeNull()
    expect(partyFromMembers([membro({ level: 9, role: 'gm' })])).toBeNull()
  })
})
