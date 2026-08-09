import { LogOut, type LucideIcon, Moon, Settings, Sun } from 'lucide-react'
import type * as React from 'react'
import { cn } from '@/shared/lib/utils'
import type { Theme } from '@/shared/stores/ui-store'
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from '@/shared/ui/popover'

/**
 * HubFooter — the player's identity strip at the foot of the Hub. The
 * initialed portrait + name is the trigger for a quick menu (popover): toggle
 * theme, Configurações (placeholder until a settings screen exists), and Sair.
 * Pure — the page injects `me`-derived name + the callbacks. "Continuar sessão"
 * (ALE-40) is a separate menu entry, not part of this footer.
 */
type HubFooterProps = {
  name: string
  theme: Theme
  onToggleTheme: () => void
  onLogout: () => void
  logoutPending?: boolean
  className?: string
}

function HubFooter({
  name,
  theme,
  onToggleTheme,
  onLogout,
  logoutPending,
  className,
}: HubFooterProps) {
  const initial = name.trim().charAt(0).toUpperCase() || '?'
  return (
    <footer
      className={cn(
        'mx-auto flex w-full max-w-md border-t border-grimorio-iron pt-4',
        className,
      )}
    >
      <Popover>
        <PopoverTrigger asChild>
          <button
            type="button"
            aria-label={`Menu de ${name}`}
            className="group flex flex-1 items-center gap-3 rounded-sm p-1 text-left focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-grimorio-gold"
          >
            <span
              aria-hidden
              className="grid size-10 place-items-center rounded-full border-2 border-grimorio-iron-light font-heading font-bold text-grimorio-gold transition-colors group-hover:border-grimorio-gold"
            >
              {initial}
            </span>
            <span className="min-w-0 flex-1 truncate font-heading tracking-wide text-foreground">
              {name}
            </span>
            <Settings
              aria-hidden
              className="size-4 text-muted-foreground transition-colors group-hover:text-grimorio-gold"
            />
          </button>
        </PopoverTrigger>
        {/* The portal escapes the .scene-grimorio subtree, so re-apply the
            scope here or the menu falls back to the light shadcn tokens. */}
        <PopoverContent align="start" side="top" className="scene-grimorio w-56 p-1.5">
          <QuickMenuItem
            icon={theme === 'light' ? Moon : Sun}
            onClick={onToggleTheme}
          >
            {theme === 'light' ? 'Modo escuro' : 'Modo claro'}
          </QuickMenuItem>
          <QuickMenuItem icon={Settings} disabled>
            Configurações
          </QuickMenuItem>
          <QuickMenuItem
            icon={LogOut}
            onClick={onLogout}
            disabled={logoutPending}
            danger
          >
            Sair
          </QuickMenuItem>
        </PopoverContent>
      </Popover>
    </footer>
  )
}

type QuickMenuItemProps = React.ComponentProps<'button'> & {
  icon: LucideIcon
  danger?: boolean
}

function QuickMenuItem({
  icon: Icon,
  children,
  danger,
  className,
  ...props
}: QuickMenuItemProps) {
  return (
    <button
      type="button"
      className={cn(
        'flex w-full items-center gap-2.5 rounded-sm px-2.5 py-2 text-left font-heading text-sm tracking-wide transition-colors',
        'hover:bg-accent focus-visible:bg-accent focus-visible:outline-none',
        'disabled:pointer-events-none disabled:opacity-50',
        danger ? 'text-grimorio-crimson-bright' : 'text-foreground',
        className,
      )}
      {...props}
    >
      <Icon aria-hidden className="size-4" />
      {children}
    </button>
  )
}

export { HubFooter }
