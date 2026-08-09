import { isCasterCharacter } from './combat-magic-stats'
import type { ReactNode } from 'react'
import { Backpack, BadgeCheck, BookOpen, ScrollText, ToggleRight, Zap } from 'lucide-react'
import type { LucideIcon } from 'lucide-react'
import type { Character } from '@/shared/api/api'
import { AbilitiesPanel } from './abilities-panel'
import { AbilitiesPendingBadge } from './abilities-pending-badge'
import { BagPanel } from './bag-panel'
import { EffectsCountBadge } from './effects-count-badge'
import { EffectsPanel } from './effects-panel'
import { ExpertisesPanel } from './expertises-panel'
import { grantedSpells } from '@/entities/character/granted-spells'
import { ProficienciesTab } from './proficiencies-panel'
import { SpellbookPanel } from './spellbook-panel'

/**
 * One switchable block of the character sheet: its tab value, short label,
 * icon, optional live badge, and how it renders for a character. Shared by the
 * desktop right-panel tabs and the mobile bottom-bar tabs so the two layouts
 * stay in lockstep — add a block here and both pick it up.
 */
export type SheetSection = {
  value: string
  label: string
  icon: LucideIcon
  badge?: (character: Character) => ReactNode
  render: (character: Character) => ReactNode
  /** Dim (but keep) the tab when irrelevant for this character. */
  dim?: (character: Character) => boolean
}

/**
 * Old deep-link tab values → their current home. Equipado + Inventário were
 * merged into the Mochila bag; ?tab=inventory bookmarks must keep landing on
 * items, not fall back to the first tab.
 */
export function resolveSheetTab(tab: string): string {
  if (tab === 'inventory' || tab === 'equipment') return 'bag'
  return tab
}

// The non-vitals blocks. The mobile layout prepends a "Vitais" section
// (header + vitals aside); the desktop layout renders vitals persistently and
// keeps these as the right-panel tabs. Campaign membership is managed on the
// campaign screens, not here (ALE-32), so there is no "Campanhas" tab.
export const SHEET_PANELS: SheetSection[] = [
  {
    value: 'expertises',
    label: 'Perícias',
    icon: ScrollText,
    render: (c) => <ExpertisesPanel character={c} />,
  },
  {
    value: 'bag',
    label: 'Mochila',
    icon: Backpack,
    render: (c) => <BagPanel character={c} />,
  },
  {
    value: 'proficiencies',
    label: 'Proficiências',
    icon: BadgeCheck,
    render: (c) => <ProficienciesTab character={c} />,
  },
  {
    value: 'conditionals',
    label: 'Efeitos',
    icon: ToggleRight,
    badge: (c) => <EffectsCountBadge character={c} />,
    render: (c) => <EffectsPanel character={c} />,
  },
  // Value stays 'abilities' — ?tab=abilities deep links must survive the
  // Habilidades→Poderes rename.
  {
    value: 'abilities',
    label: 'Poderes',
    icon: Zap,
    badge: (c) => <AbilitiesPendingBadge character={c} />,
    render: (c) => <AbilitiesPanel character={c} />,
  },
  {
    value: 'spells',
    // Dim only when there is truly nothing castable — a Bárbaro with Totem
    // Espiritual owns a granted spell and must see the tab lit.
    dim: (c) => !isCasterCharacter(c) && grantedSpells(c).length === 0,
    label: 'Magias',
    icon: BookOpen,
    render: (c) => <SpellbookPanel character={c} />,
  },
]
