import type { Session } from '@/shared/api/api'

// The chronicle's visual identity lives in entities/campaign, and consumers
// import it from there directly — this file used to re-export it, which hid the
// origin without buying anything (campaign-tome already went straight to the
// source).

export type CampaignSessions = { campaignId: number; sessions: Session[] | undefined }

/**
 * campaignId → id of its live (`status === 'active'`) session, for every
 * campaign that has one. Pure, so the "which chronicles are live" rule is
 * unit-tested without the network. Undefined lists (still loading) are skipped.
 *
 * @example activeSessionByCampaign([{ campaignId: 5, sessions: [live] }]) // { 5: 12 }
 */
export function activeSessionByCampaign(
  lists: readonly CampaignSessions[],
): Record<number, number> {
  const map: Record<number, number> = {}
  for (const { campaignId, sessions } of lists) {
    const live = sessions?.find((s) => s.status === 'active')
    if (live) map[campaignId] = live.id
  }
  return map
}
