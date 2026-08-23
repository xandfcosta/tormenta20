import type { CampaignMember } from '@/shared/api/types'

/** O grupo para o qual o encontro está sendo montado. */
export type PartyDefaults = { level: number; size: number }

/**
 * O grupo desta campanha, para o construtor de encontros já nascer preenchido
 * (ALE-209).
 *
 * O mestre digitava nível e quantidade a cada encontro, e o app já sabe os
 * dois. "Os jogadores" são os **PCs da campanha** e não quem está na iniciativa
 * agora — decisão do dono: montar encontro é coisa que se faz ANTES da briga, e
 * a leitura pela iniciativa daria zero justamente aí, o que é pior do que vir
 * vazio.
 *
 * Devolve `null` quando não há PC nenhum, e isso é deliberado: quem chama fica
 * com o próprio padrão em vez de receber um grupo de tamanho zero, que a Tabela
 * 7-1 (p282) não sabe pontuar. Membro sem o personagem embutido não conta —
 * é estado parcial de carregamento, e chutar o nível dele mentiria no ND.
 *
 * @example partyFromMembers(members) // { level: 6, size: 5 }
 */
export function partyFromMembers(members: readonly CampaignMember[]): PartyDefaults | null {
  const levels = members
    .filter((member) => member.role === 'player')
    .map((member) => member.character?.level)
    .filter((level): level is number => typeof level === 'number')

  if (levels.length === 0) return null
  return { level: averageLevel(levels), size: levels.length }
}

/**
 * A média ARREDONDADA, porque a Tabela 7-1 (p282) só tem linha para nível
 * inteiro. Meio nível para cima: um grupo 1/1/2/2 mira o 2, e é o mestre quem
 * corrige se quis o contrário — o campo continua editável.
 */
function averageLevel(levels: readonly number[]): number {
  const soma = levels.reduce((total, level) => total + level, 0)
  return Math.round(soma / levels.length)
}
