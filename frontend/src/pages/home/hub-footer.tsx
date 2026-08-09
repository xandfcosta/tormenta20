import { LogOut, Moon, Sun } from 'lucide-react'
import { cn } from '@/shared/lib/utils'
import type { Theme } from '@/shared/stores/ui-store'
import { Button } from '@/shared/ui/button'

/**
 * HubFooter — the player's identity strip at the foot of the Hub: an initialed
 * portrait, the name, and quick actions (theme toggle + logout). Pure — the
 * page injects the callbacks. The richer portrait quick-menu (config) lands in
 * ALE-39; this is the functional baseline so nothing is stranded once the Hub
 * drops the app nav.
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
        'mx-auto flex w-full max-w-md items-center gap-3 border-t border-grimorio-iron pt-4',
        className,
      )}
    >
      <span
        aria-hidden
        className="grid size-10 place-items-center rounded-full border-2 border-grimorio-gold font-heading font-bold text-grimorio-gold"
      >
        {initial}
      </span>
      <span className="flex-1 truncate font-heading tracking-wide text-foreground">
        {name}
      </span>
      <Button
        variant="ghost"
        size="icon"
        onClick={onToggleTheme}
        aria-label={theme === 'light' ? 'Modo escuro' : 'Modo claro'}
      >
        {theme === 'light' ? (
          <Moon className="size-4" />
        ) : (
          <Sun className="size-4" />
        )}
      </Button>
      <Button
        variant="ghost"
        size="icon"
        onClick={onLogout}
        disabled={logoutPending}
        aria-label="Sair"
      >
        <LogOut className="size-4" />
      </Button>
    </footer>
  )
}

export { HubFooter }
