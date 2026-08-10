import { isCasterCharacter } from './combat-magic-stats'
import { memo, type ReactNode } from 'react'
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

// The layouts keep every opened block mounted (see useVisitedTabs), so a tab
// switch re-renders the parent and would otherwise re-run EVERY mounted panel —
// the heaviest, Perícias, is ~850 DOM nodes. Memoizing each panel on its only
// prop (`character`, a stable query object between switches) lets the untouched
// panels bail out of that re-render, so a switch becomes a Radix visibility
// toggle instead of a ~250–540ms reconcile. Store-driven updates (conditionals,
// edits that mint a new character) still re-render — memo only skips the
// parent's tab-switch churn. Measured: the tab-switch jank fix.
const Expertises = memo(ExpertisesPanel)
const Bag = memo(BagPanel)
const Proficiencies = memo(ProficienciesTab)
const Effects = memo(EffectsPanel)
const Abilities = memo(AbilitiesPanel)
const Spellbook = memo(SpellbookPanel)

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
    render: (c) => <Expertises character={c} />,
  },
  {
    value: 'bag',
    label: 'Mochila',
    icon: Backpack,
    render: (c) => <Bag character={c} />,
  },
  {
    value: 'proficiencies',
    label: 'Proficiências',
    icon: BadgeCheck,
    render: (c) => <Proficiencies character={c} />,
  },
  {
    value: 'conditionals',
    label: 'Efeitos',
    icon: ToggleRight,
    badge: (c) => <EffectsCountBadge character={c} />,
    render: (c) => <Effects character={c} />,
  },
  // Value stays 'abilities' — ?tab=abilities deep links must survive the
  // Habilidades→Poderes rename.
  {
    value: 'abilities',
    label: 'Poderes',
    icon: Zap,
    badge: (c) => <AbilitiesPendingBadge character={c} />,
    render: (c) => <Abilities character={c} />,
  },
  {
    value: 'spells',
    // Dim only when there is truly nothing castable — a Bárbaro with Totem
    // Espiritual owns a granted spell and must see the tab lit.
    dim: (c) => !isCasterCharacter(c) && grantedSpells(c).length === 0,
    label: 'Magias',
    icon: BookOpen,
    render: (c) => <Spellbook character={c} />,
  },
]
