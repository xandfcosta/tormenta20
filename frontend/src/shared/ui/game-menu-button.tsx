import type { LucideIcon } from 'lucide-solid'
import { Dynamic } from 'solid-js/web'
import { type ComponentProps, Show, splitProps } from 'solid-js'
import { cn } from '@/shared/lib/utils'

export type GameMenuButtonProps = ComponentProps<'button'> & {
  icon?: LucideIcon
  /** Shows the ► "continue" chevron on the right (e.g. resume session). */
  hasNext?: boolean
  /** Marks the current destination — sets aria-current + the gold edge. */
  active?: boolean
}

/**
 * A menu entry for the Hub / scene menus: Cinzel label, a crimson tick, an
 * optional leading icon, an optional trailing "next" chevron, and the
 * iron→gold hover treatment. At least 44px tall for a comfortable touch
 * target. Meant to live inside a `.scene-grimorio` scope; navigate from
 * `onClick`.
 *
 * @example
 * <GameMenuButton icon={Users2} onClick={() => navigate({ to: '/characters' })}>
 *   Meus Heróis
 * </GameMenuButton>
 */
export function GameMenuButton(props: GameMenuButtonProps) {
  const [local, rest] = splitProps(props, ['class', 'children', 'icon', 'hasNext', 'active', 'type'])
  return (
    <button
      type={local.type ?? 'button'}
      data-slot="game-menu-button"
      data-active={local.active || undefined}
      aria-current={local.active ? 'page' : undefined}
      class={cn(
        'grimorio-menu-item group flex min-h-11 w-full items-center gap-3 rounded-none px-4 py-3 text-left',
        'font-heading text-lg tracking-wide text-foreground',
        'focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-grimorio-gold',
        local.class,
      )}
      {...rest}
    >
      <span aria-hidden="true" class="text-sm text-grimorio-crimson-bright">
        ▸
      </span>
      <Show when={local.icon}>
        {(icon) => <Dynamic component={icon()} aria-hidden="true" class="size-5 text-grimorio-gold" />}
      </Show>
      <span class="flex-1">{local.children}</span>
      <Show when={local.hasNext}>
        <span aria-hidden="true" class="text-sm text-grimorio-gold">
          ►
        </span>
      </Show>
    </button>
  )
}
