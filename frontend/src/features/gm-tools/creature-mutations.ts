import type { QueryClient } from '@tanstack/solid-query'
import { api } from '@/shared/api/api'
import type { CampaignCreature, CreatureInput } from '@/shared/api/creature-types'

export type CreatureActions = {
  create: (input: CreatureInput) => Promise<CampaignCreature>
  update: (id: number, input: CreatureInput) => Promise<CampaignCreature>
  remove: (id: number) => Promise<void>
}

/**
 * Escrita dos blocos de criatura da campanha (ALE-137).
 *
 * Sem otimismo, ao contrário do resto das mutações da casa: aqui o servidor
 * VALIDA contra as listas do livro (tipo e tamanho fechados, PV mínimo) e pode
 * recusar. Pintar a criatura na tela antes da resposta mostraria um bloco que o
 * banco não aceitou — e o mestre está no meio de um combate, não conferindo
 * telas. O ganho do otimismo é imperceptível num diálogo que já é modal.
 *
 * Fora do componente e recebendo o `queryClient`, como as outras: assim a regra
 * é testável sem montar tela.
 *
 * @example const actions = creatureActions(queryClient, campaignId)
 */
export function creatureActions(
  queryClient: QueryClient,
  campaignId: number,
): CreatureActions {
  const queryKey = ['campaigns', campaignId, 'creatures'] as const
  const refresh = () => queryClient.invalidateQueries({ queryKey })

  return {
    create: async (input) => {
      const created = await api.creatures.create(campaignId, input)
      await refresh()
      return created
    },
    update: async (id, input) => {
      const updated = await api.creatures.update(campaignId, id, input)
      await refresh()
      return updated
    },
    remove: async (id) => {
      await api.creatures.delete(campaignId, id)
      await refresh()
    },
  }
}
