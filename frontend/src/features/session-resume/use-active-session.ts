import { useQueries, useQuery } from '@tanstack/react-query'
import { campaignsQueryOptions } from '@/entities/campaign/queries'
import { campaignSessionsQueryOptions } from '@/entities/session/queries'
import type { Session } from '@/shared/api/api'

/** Where a live session lives — enough to route into the match-mode screen. */
export type ActiveSessionRef = { campaignId: number; sessionId: number }

type CampaignSessions = { campaignId: number; sessions: Session[] | undefined }

/**
 * The first live (`status === 'active'`) session across the given campaigns'
 * session lists, in campaign order — or null. Pure, so the selection rule is
 * unit-tested without the network. Undefined lists (still loading) are skipped.
 */
export function firstActiveSession(
  lists: CampaignSessions[],
): ActiveSessionRef | null {
  for (const { campaignId, sessions } of lists) {
    const live = sessions?.find((s) => s.status === 'active')
    if (live) return { campaignId, sessionId: live.id }
  }
  return null
}

/**
 * Watches every campaign the player belongs to (GM or player) for a live
 * session and returns a ref to the first one, or null. Reactive via TanStack
 * Query — the value flips as the per-campaign session lists refetch (on mount,
 * window focus, or cache invalidation after a session starts/ends), so the Hub
 * shows/hides "Continuar sessão" without a manual reload. Fans out one sessions
 * query per campaign (no server-side "my active sessions" endpoint exists).
 */
export function useActiveSession(): ActiveSessionRef | null {
  const campaigns = useQuery(campaignsQueryOptions)
  const ids = campaigns.data?.map((c) => c.id) ?? []
  const results = useQueries({
    queries: ids.map((id) => campaignSessionsQueryOptions(id)),
  })
  const lists = ids.map((campaignId, i) => ({
    campaignId,
    sessions: results[i]?.data,
  }))
  return firstActiveSession(lists)
}
