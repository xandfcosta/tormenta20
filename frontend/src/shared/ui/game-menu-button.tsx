import type * as React from 'react'
import type { LucideIcon } from 'lucide-react'
import { cn } from '@/shared/lib/utils'

/**
 * GameMenuButton — a menu entry for the Hub / scene menus. Cinzel label, a
 * crimson tick, an optional leading icon, an optional trailing "next" chevron,
 * and the iron→gold hover treatment (glow + slide). At least 44px tall for a
 * comfortable touch target. Meant to be used inside a `.scene-grimorio` scope;
 * navigate from its `onClick`.
 *
 * @example
 * <GameMenuButton icon={Users2} onClick={() => navigate({ to: '/characters' })}>
 *   Meus Heróis
 * </GameMenuButton>
 */
type GameMenuButtonProps = React.ComponentProps<'button'> & {
  icon?: LucideIcon
  /** Shows the ► "continue" chevron on the right (e.g. resume session). */
  hasNext?: boolean
  /** Marks the current destination — sets aria-current + the gold edge. */
  active?: boolean
}

function GameMenuButton({
  className,
  children,
  icon: Icon,
  hasNext = false,
  active = false,
  type = 'button',
  ...props
}: GameMenuButtonProps) {
  return (
    <button
      type={type}
      data-slot="game-menu-button"
      data-active={active || undefined}
      aria-current={active ? 'page' : undefined}
      className={cn(
        'grimorio-menu-item group flex min-h-11 w-full items-center gap-3 rounded-sm px-4 py-3 text-left',
        'font-heading text-lg tracking-wide text-foreground',
        'focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-grimorio-gold',
        className,
      )}
      {...props}
    >
      <span aria-hidden className="text-sm text-grimorio-crimson-bright">
        ▸
      </span>
      {Icon ? <Icon aria-hidden className="size-5 text-grimorio-gold" /> : null}
      <span className="flex-1">{children}</span>
      {hasNext ? (
        <span aria-hidden className="text-sm text-grimorio-gold">
          ►
        </span>
      ) : null}
    </button>
  )
}

export { GameMenuButton }
