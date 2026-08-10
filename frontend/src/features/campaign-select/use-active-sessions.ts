import { useQueries } from '@tanstack/react-query'
import { campaignSessionsQueryOptions } from '@/entities/session/queries'
import { activeSessionByCampaign } from './campaign-select-helpers'

/**
 * campaignId → live session id, for the given campaigns. Reactive via TanStack
 * Query — the map flips as the per-campaign session lists refetch, so the rail
 * ember + stage "Continuar" CTA appear/disappear without a reload. Fans out one
 * sessions query per campaign (no server-side "my active sessions" endpoint);
 * fine for the handful of campaigns a user belongs to. Mirrors
 * `useActiveSession` (the Hub's single-session variant).
 */
export function useActiveSessionByCampaign(
  campaignIds: number[],
): Record<number, number> {
  const results = useQueries({
    queries: campaignIds.map((id) => campaignSessionsQueryOptions(id)),
  })
  const lists = campaignIds.map((campaignId, i) => ({
    campaignId,
    sessions: results[i]?.data,
  }))
  return activeSessionByCampaign(lists)
}
