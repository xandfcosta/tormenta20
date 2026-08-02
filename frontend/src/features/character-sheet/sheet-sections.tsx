import { isCasterCharacter } from './combat-magic-stats'
import type { ReactNode } from 'react'
import { BookOpen, Package, ScrollText, Shirt, Star, Tent, ToggleRight } from 'lucide-react'
import type { LucideIcon } from 'lucide-react'
import type { Character } from '@/shared/api/api'
import { AbilitiesPanel } from './abilities-panel'
import { AbilitiesPendingBadge } from './abilities-pending-badge'
import { CampaignsPanel } from './campaigns-panel'
import { EffectsCountBadge } from './effects-count-badge'
import { EffectsPanel } from './effects-panel'
import { EquipmentPanel } from './equipment-panel'
import { ExpertisesPanel } from './expertises-panel'
import { InventoryPanel } from './inventory-panel'
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

// The eight non-vitals blocks. The mobile layout prepends a "Vitais" section
// (header + vitals aside); the desktop layout renders vitals persistently and
// keeps these as the right-panel tabs.
export const SHEET_PANELS: SheetSection[] = [
  {
    value: 'expertises',
    label: 'Perícias',
    icon: ScrollText,
    render: (c) => <ExpertisesPanel character={c} />,
  },
  {
    value: 'equipment',
    label: 'Equipado',
    icon: Shirt,
    render: (c) => <EquipmentPanel character={c} />,
  },
  {
    value: 'inventory',
    label: 'Inventário',
    icon: Package,
    render: (c) => <InventoryPanel character={c} />,
  },
  {
    value: 'conditionals',
    label: 'Efeitos',
    icon: ToggleRight,
    badge: (c) => <EffectsCountBadge character={c} />,
    render: (c) => <EffectsPanel character={c} />,
  },
  {
    value: 'abilities',
    label: 'Habilidades',
    icon: Star,
    badge: (c) => <AbilitiesPendingBadge character={c} />,
    render: (c) => <AbilitiesPanel character={c} />,
  },
  {
    value: 'spells',
    dim: (c) => !isCasterCharacter(c),
    label: 'Magias',
    icon: BookOpen,
    render: (c) => <SpellbookPanel character={c} />,
  },
  {
    value: 'campaigns',
    label: 'Campanhas',
    icon: Tent,
    render: (c) => <CampaignsPanel characterId={c.id} />,
  },
]
