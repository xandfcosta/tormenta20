import type { LucideIcon } from 'lucide-react'
import { cn } from '@/shared/lib/utils'
import { GameMenuButton } from '@/shared/ui/game-menu-button'

/** One entry in the Hub's main menu — a label, an icon, and what to do when
 *  chosen. `hasNext` shows the ► "continue" chevron (e.g. resume session). */
type HubMenuItem = {
  label: string
  icon?: LucideIcon
  onSelect: () => void
  hasNext?: boolean
}

type HubMenuProps = {
  items: HubMenuItem[]
  className?: string
}

/** The Hub's vertical menu — the game's "main menu". Pure: the page wires each
 *  item's `onSelect` to navigation. */
function HubMenu({ items, className }: HubMenuProps) {
  return (
    <nav
      aria-label="Menu principal"
      className={cn('mx-auto flex w-full max-w-md flex-col gap-2.5', className)}
    >
      {items.map((item) => (
        <GameMenuButton
          key={item.label}
          icon={item.icon}
          hasNext={item.hasNext}
          onClick={item.onSelect}
        >
          {item.label}
        </GameMenuButton>
      ))}
    </nav>
  )
}

export { HubMenu }
export type { HubMenuItem }
