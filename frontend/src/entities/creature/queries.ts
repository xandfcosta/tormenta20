import { queryOptions } from '@tanstack/solid-query'
import { api } from '@/shared/api/api'

/**
 * Os blocos de criatura da campanha (ALE-137).
 *
 * Só o mestre consegue ler: o servidor responde 403 ao jogador até no GET,
 * porque o bloco é informação do mestre. Quem monta esta query numa tela de
 * jogador está pedindo um erro — a tela do jogador mostra a criatura pela
 * iniciativa, não por aqui.
 */
export const campaignCreaturesQueryOptions = (campaignId: number) =>
  queryOptions({
    queryKey: ['campaigns', campaignId, 'creatures'] as const,
    queryFn: () => api.creatures.list(campaignId),
  })
