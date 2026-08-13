import { useQueryClient, useQuery } from '@tanstack/solid-query'
import { type Accessor, createMemo } from 'solid-js'
import { campaignsQueryOptions } from '@/entities/campaign/queries'
import { campaignSessionsQueryOptions } from '@/entities/session/queries'
import type { Session } from '@/shared/api/api'
import { settledQuery } from '@/shared/lib/settled-query'

/** Where a live session lives — enough to route into the match-mode screen. */
export type ActiveSessionRef = { campaignId: number; sessionId: number }

export type CampaignSessions = { campaignId: number; sessions: Session[] | undefined }

/**
 * The first live (`status === 'active'`) session across the given campaigns'
 * session lists, in campaign order — or null. Pure, so the selection rule is
 * unit-tested without the network. Undefined lists (still loading) are skipped.
 *
 * @example firstActiveSession([{ campaignId: 1, sessions }]) // { campaignId: 1, sessionId: 4 }
 */
export function firstActiveSession(lists: readonly CampaignSessions[]): ActiveSessionRef | null {
  for (const { campaignId, sessions } of lists) {
    const live = sessions?.find((s) => s.status === 'active')
    if (live) return { campaignId, sessionId: live.id }
  }
  return null
}

/**
 * Watches every campaign the player belongs to for a live session and returns
 * a ref to the first one, or null — so the Hub can offer "Continuar sessão".
 * Fans out one sessions query per campaign (there's no server-side "my active
 * sessions" endpoint).
 *
 * Solid note: the React version used `useQueries` for the fan-out. Solid Query
 * has no stable equivalent, so this fetches each campaign's sessions through
 * the query client and derives from the cache — which is what `useQueries` was
 * doing underneath anyway.
 */
export function createActiveSession(): Accessor<ActiveSessionRef | null> {
  const queryClient = useQueryClient()
  const campaigns = useQuery(() => campaignsQueryOptions)

  // `settledQuery`, não `.data`: ler pendente suspende o route match inteiro e
  // a cena do Hub reanima (ALE-95/ALE-96). Vale para as QUATRO leituras deste
  // hook — a da chave e a do `enabled` suspendem tanto quanto a do resultado,
  // porque as opções são avaliadas num escopo reativo como qualquer outra.
  const settledCampaigns = () => settledQuery(campaigns)

  const sessionLists = useQuery(() => ({
    queryKey: ['session-resume', (settledCampaigns() ?? []).map((c) => c.id)] as const,
    enabled: settledCampaigns() !== null,
    queryFn: async (): Promise<CampaignSessions[]> =>
      Promise.all(
        (settledCampaigns() ?? []).map(async (campaign) => ({
          campaignId: campaign.id,
          sessions: await queryClient.ensureQueryData(campaignSessionsQueryOptions(campaign.id)),
        })),
      ),
  }))

  return createMemo(() => firstActiveSession(settledQuery(sessionLists) ?? []))
}
