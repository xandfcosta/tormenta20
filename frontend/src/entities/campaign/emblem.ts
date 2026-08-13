import type { CampaignMemberRole } from '@/shared/api/api'
import { hueGradient } from '@/shared/lib/hue-from-name'

// A chronicle's visual identity, derived from its name — shared by the roster
// book (campaign-select) and the detail tome (campaign-manage), so it lives in
// entities (both features point down to it).

/** Deterministic hue-tinted gradient for a campaign's emblem/cover, so each
 *  chronicle reads as its own sigil without any stored image. */
export function campaignEmblemGradient(name: string): string {
  return hueGradient(name, 0.5, 0.14)
}

/** The caller's stance in a chronicle, as a scene label. */
export function roleLabel(role: CampaignMemberRole | undefined): string {
  return role === 'gm' ? 'Mestrando' : 'Jogando'
}
