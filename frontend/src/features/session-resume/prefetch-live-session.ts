import { useQueryClient } from '@tanstack/solid-query'
import { useRouter } from '@tanstack/solid-router'
import { createEffect } from 'solid-js'
import { campaignMembersQueryOptions, campaignQueryOptions } from '@/entities/campaign/queries'
import { campaignSessionQueryOptions } from '@/entities/session/queries'
import type { ActiveSessionRef } from './active-session'

/**
 * Esquenta a cena da sessão ao vivo enquanto o jogador ainda olha o Hub.
 *
 * "Continuar sessão" é a ação primária da mesa, e o trabalho que ela exige é
 * conhecido antes do clique: o CHUNK da rota (a cena é code-split) e três
 * leituras (sessão, campanha, membros). Fazer isso depois do clique é o que se
 * sente como demora; fazer antes torna a troca imediata.
 *
 * Só dispara quando existe sessão viva — ou seja, quando o botão existe —, e o
 * custo é o de uma navegação que quase certamente vai acontecer: quem está no
 * Hub com uma partida em andamento entra nela.
 *
 * @example createLiveSessionPrefetch(activeSession)
 */
export function createLiveSessionPrefetch(activeSession: () => ActiveSessionRef | null): void {
  const router = useRouter()
  const queryClient = useQueryClient()

  createEffect(() => {
    const live = activeSession()
    if (!live) return
    void router.preloadRoute({
      to: '/campaigns/$id/sessions/$sid',
      params: { id: String(live.campaignId), sid: String(live.sessionId) },
    })
    void queryClient.prefetchQuery(campaignSessionQueryOptions(live.campaignId, live.sessionId))
    void queryClient.prefetchQuery(campaignQueryOptions(live.campaignId))
    void queryClient.prefetchQuery(campaignMembersQueryOptions(live.campaignId))
  })
}
