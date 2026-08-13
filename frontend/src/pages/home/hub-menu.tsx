import type { LucideIcon } from 'lucide-solid'
import { For } from 'solid-js'
import { cn } from '@/shared/lib/utils'
import { GameMenuButton } from '@/shared/ui/game-menu-button'

/** One entry in the Hub's main menu — a label, an icon, and what to do when
 *  chosen. `hasNext` shows the ► "continue" chevron (e.g. resume session). */
export type HubMenuItem = {
  label: string
  icon?: LucideIcon
  onSelect: () => void
  hasNext?: boolean
}

export type HubMenuProps = {
  items: HubMenuItem[]
  class?: string
  /** Fired when the pointer enters any item — e.g. a subtle hover cue. */
  onItemHover?: () => void
}

/**
 * The Hub's vertical menu — the game's "main menu". Pure: the page wires each
 * item's `onSelect` to navigation and the optional hover cue.
 *
 * `data-nav-region`/`data-nav-layout` are what make the arrows work: the scene
 * nav driver reads them off the DOM, so the menu declares its shape and never
 * imports the driver.
 */
export function HubMenu(props: HubMenuProps) {
  return (
    <nav
      aria-label="Menu principal"
      data-nav-region="menu"
      data-nav-layout="column"
      class={cn('mx-auto flex w-full max-w-md flex-col gap-2.5', props.class)}
    >
      <For each={props.items}>
        {(item) => (
          <GameMenuButton
            icon={item.icon}
            hasNext={item.hasNext}
            onClick={() => item.onSelect()}
            onMouseEnter={() => props.onItemHover?.()}
          >
            {item.label}
          </GameMenuButton>
        )}
      </For>
    </nav>
  )
}
