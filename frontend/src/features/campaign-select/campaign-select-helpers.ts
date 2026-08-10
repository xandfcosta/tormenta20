import type { CampaignMemberRole, Session } from '@/shared/api/api'
import { hueFromName } from '@/shared/lib/hue-from-name'

/** Two-letter monogram from a campaign name — the emblem until real cover art
 *  lands (same slot). Mirrors the character-select initials. */
export function campaignInitials(name: string): string {
  const parts = name.trim().split(/\s+/).filter(Boolean)
  if (parts.length === 0) return '?'
  return parts
    .slice(0, 2)
    .map((w) => w[0])
    .join('')
    .toUpperCase()
}

/** Deterministic hue-tinted gradient for a campaign's emblem/cover, so each
 *  chronicle reads as its own sigil without any stored image. */
export function campaignEmblemGradient(name: string): string {
  const hue = hueFromName(name)
  return `linear-gradient(155deg, oklch(0.5 0.14 ${hue}) 0%, oklch(0.3 0.09 ${hue}) 70%, oklch(0.22 0.06 ${hue}) 100%)`
}

/** The caller's stance in a chronicle, as a scene label. */
export function roleLabel(role: CampaignMemberRole | undefined): string {
  return role === 'gm' ? 'Mestrando' : 'Jogando'
}

type CampaignSessions = { campaignId: number; sessions: Session[] | undefined }

/**
 * campaignId → id of its live (`status === 'active'`) session, for every
 * campaign that has one. Pure, so the "which chronicles are live" rule is
 * unit-tested without the network. Undefined lists (still loading) are skipped.
 *
 * @example activeSessionByCampaign([{campaignId: 5, sessions: [live]}]) // => {5: 12}
 */
export function activeSessionByCampaign(
  lists: CampaignSessions[],
): Record<number, number> {
  const map: Record<number, number> = {}
  for (const { campaignId, sessions } of lists) {
    const live = sessions?.find((s) => s.status === 'active')
    if (live) map[campaignId] = live.id
  }
  return map
}
