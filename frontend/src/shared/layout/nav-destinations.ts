import type { LucideIcon } from 'lucide-react'
import { Home, Scroll, Swords, Users2, Wand2 } from 'lucide-react'

/**
 * Shared destination list — top nav, bottom nav, and mobile drawer
 * all consume the same source so a route rename doesn't drift across
 * three files.
 */
export type NavDestination = {
  to: string
  label: string
  Icon: LucideIcon
  /** primary = fits into the mobile bottom bar; secondary lives in
   * the hamburger drawer only. Total primaries capped at 4 to avoid
   * fighting for thumb reach + iOS home indicator. */
  tier: 'primary' | 'secondary'
}

/**
 * `authed` list — visible only once `me` resolves. `public` list
 * covers landing / login / register scenarios.
 */
export const AUTHED_DESTINATIONS: readonly NavDestination[] = [
  { to: '/', label: 'Início', Icon: Home, tier: 'primary' },
  { to: '/characters', label: 'Personagens', Icon: Users2, tier: 'primary' },
  { to: '/campaigns', label: 'Campanhas', Icon: Scroll, tier: 'primary' },
  { to: '/gm', label: 'GM', Icon: Wand2, tier: 'primary' },
  { to: '/users', label: 'Usuários', Icon: Swords, tier: 'secondary' },
]

export const PUBLIC_DESTINATIONS: readonly NavDestination[] = []
