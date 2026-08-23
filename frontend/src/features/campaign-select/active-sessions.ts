import { useQuery, useQueryClient } from '@tanstack/solid-query'
import { type Accessor, createMemo } from 'solid-js'
import { campaignSessionsQueryOptions } from '@/entities/session/queries'
import { settledQuery } from '@/shared/lib/settled-query'
import { activeSessionByCampaign } from './campaign-select-helpers'

/**
 * campaignId → live session id, for the given campaigns — so the rail ember and
 * the "Continuar a sessão" CTA appear without a reload. Fans out one sessions
 * query per campaign (there's no server-side "my active sessions" endpoint);
 * fine for the handful a user belongs to.
 *
 * Mirrors `createActiveSession` (the Hub's single-session variant), including
 * its note: Solid Query has no stable `useQueries`, so the fan-out goes through
 * the query client and derives from the cache.
 *
 * @example const live = createActiveSessionByCampaign(() => ids()); live()[5]
 */
export function createActiveSessionByCampaign(
  campaignIds: Accessor<number[]>,
): Accessor<Record<number, number>> {
  const queryClient = useQueryClient()

  const lists = useQuery(() => ({
    queryKey: ['campaign-select', 'live', campaignIds()] as const,
    enabled: campaignIds().length > 0,
    queryFn: async () =>
      Promise.all(
        campaignIds().map(async (campaignId) => ({
          campaignId,
          sessions: await queryClient.ensureQueryData(campaignSessionsQueryOptions(campaignId)),
        })),
      ),
  }))

  // `settledQuery`, não `.data`: ler pendente suspende o route match inteiro e
  // a cena das Campanhas reanima (ALE-95/ALE-96).
  return createMemo(() => activeSessionByCampaign(settledQuery(lists) ?? []))
}
